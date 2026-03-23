package service

import (
	"auron/user-service/internal/domain"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repository domain.UserRepository
	cache      domain.UserCache
	publisher  domain.EventPublisher
}

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

func NewUserService(repository domain.UserRepository, cache domain.UserCache, publisher domain.EventPublisher) domain.UserService {
	return &UserService{
		repository: repository,
		cache:      cache,
		publisher:  publisher,
	}
}

func (s *UserService) RegisterUser(req *domain.CreateUserRequest) (*domain.User, error) {
	if existingUser, _ := s.repository.GetUserByEmail(req.Email); existingUser != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	if req.Password != req.ConfirmPassword {
		return nil, domain.ErrPasswordMismatch
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Password:  hashedPassword,
		Name:      req.Name,
		Role:      domain.RoleCustomer,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdUser, err := s.repository.CreateUser(user)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetUser(context.Background(), createdUser); err != nil {
		slog.Error("failed to cache user", "error", err)
	}

	if err := s.publisher.Publish(context.Background(), "user.created", createdUser); err != nil {
		slog.Error("failed to publish user.created event", "error", err)
	}

	return createdUser, nil
}

func (s *UserService) Login(req *domain.LoginRequest) (*domain.User, string, error) {
	authResponse, err := s.LoginWithTokens(req)
	if err != nil {
		return nil, "", err
	}

	return authResponse.User, authResponse.AccessToken, nil
}

func (s *UserService) LoginWithTokens(req *domain.LoginRequest) (*domain.AuthResponse, error) {
	user, err := s.repository.GetUserByEmail(req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	accessToken, err := generateJWT(user.ID.String(), user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	if err := s.storeRefreshToken(user.ID, refreshToken); err != nil {
		return nil, err
	}

	return buildAuthResponse(user, accessToken, refreshToken), nil
}

func (s *UserService) RefreshToken(req *domain.RefreshTokenRequest) (string, error) {
	authResponse, err := s.RefreshTokenWithTokens(req)
	if err != nil {
		return "", err
	}

	return authResponse.AccessToken, nil
}

func (s *UserService) RefreshTokenWithTokens(req *domain.RefreshTokenRequest) (*domain.AuthResponse, error) {
	claims, err := validateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, err
	}

	hash := hashToken(req.RefreshToken)
	storedToken, err := s.repository.GetRefreshToken(hash)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}

	if storedToken.RevokedAt != nil {
		return nil, domain.ErrInvalidToken
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return nil, domain.ErrExpiredToken
	}

	userIDRaw, ok := claims["sub"].(string)
	if !ok || userIDRaw == "" {
		return nil, domain.ErrInvalidToken
	}

	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	user, err := s.repository.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}

	if err := s.repository.RevokeRefreshToken(hash); err != nil {
		return nil, err
	}

	accessToken, err := generateJWT(user.ID.String(), user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := generateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	if err := s.storeRefreshToken(user.ID, newRefreshToken); err != nil {
		return nil, err
	}

	return buildAuthResponse(user, accessToken, newRefreshToken), nil
}

func (s *UserService) storeRefreshToken(userID uuid.UUID, refreshToken string) error {
	refreshTokenEntity := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: time.Now().Add(refreshTokenTTL),
		CreatedAt: time.Now(),
	}

	if _, err := s.repository.AddRefreshToken(refreshTokenEntity); err != nil {
		return err
	}

	return nil
}

func buildAuthResponse(user *domain.User, accessToken, refreshToken string) *domain.AuthResponse {
	secureCookie := shouldUseSecureCookies()

	return &domain.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessTokenCookie: domain.CookieConfig{
			Name:     "access_token",
			Value:    accessToken,
			Path:     "/",
			MaxAge:   int(accessTokenTTL.Seconds()),
			HttpOnly: true,
			Secure:   secureCookie,
			SameSite: "Strict",
		},
		RefreshTokenCookie: domain.CookieConfig{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/",
			MaxAge:   int(refreshTokenTTL.Seconds()),
			HttpOnly: true,
			Secure:   secureCookie,
			SameSite: "Strict",
		},
	}
}

func shouldUseSecureCookies() bool {
	if value := strings.TrimSpace(os.Getenv("COOKIE_SECURE")); value != "" {
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}

	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	return env == "prod" || env == "production"
}

func (s *UserService) RevokeToken(req *domain.RevokeTokenRequest) error {
	hash := hashToken(req.RefreshToken)
	if err := s.repository.RevokeRefreshToken(hash); err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			return domain.ErrInvalidToken
		}
		return err
	}
	return nil
}

func (s *UserService) GetUserProfile(userID uuid.UUID) (*domain.User, error) {
	user, err := s.cache.GetUserByID(context.Background(), userID.String())
	if err == nil {
		return user, nil
	}

	user, err = s.repository.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetUser(context.Background(), user); err != nil {
		slog.Error("failed to cache user", "error", err)
	}

	return user, nil
}

func (s *UserService) UpdateUserProfile(userID uuid.UUID, req *domain.UpdateUserRequest) (*domain.User, error) {
	user, err := s.repository.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if req.Email != nil && *req.Email != "" && *req.Email != user.Email {
		existingUser, checkErr := s.repository.GetUserByEmail(*req.Email)
		if checkErr == nil && existingUser != nil && existingUser.ID != userID {
			return nil, domain.ErrEmailAlreadyExists
		}
		if checkErr != nil && !errors.Is(checkErr, domain.ErrUserNotFound) {
			return nil, checkErr
		}
		user.Email = *req.Email
	}

	if req.Name != nil {
		user.Name = *req.Name
	}

	if req.Password != nil {
		hashedPassword, hashErr := hashPassword(*req.Password)
		if hashErr != nil {
			return nil, hashErr
		}
		user.Password = hashedPassword
	}

	user.UpdatedAt = time.Now()
	updatedUser, err := s.repository.UpdateUser(user)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetUser(context.Background(), updatedUser); err != nil {
		slog.Error("failed to update cached user", "error", err)
	}

	return updatedUser, nil
}

func (s *UserService) AddAddress(userID uuid.UUID, address *domain.Address) (*domain.Address, error) {
	if _, err := s.repository.GetUserByID(userID); err != nil {
		return nil, err
	}

	if address.ID == uuid.Nil {
		address.ID = uuid.New()
	}
	address.UserID = userID
	address.CreatedAt = time.Now()

	return s.repository.AddAddress(address)
}

func (s *UserService) GetAddresses(userID uuid.UUID) ([]domain.Address, error) {
	if _, err := s.repository.GetUserByID(userID); err != nil {
		return nil, err
	}
	return s.repository.GetAddressesByUserID(userID)
}

func (s *UserService) UpdateAddress(userID, addressID uuid.UUID, req *domain.UpdateAddressRequest) (*domain.Address, error) {
	if addressID == uuid.Nil || userID == uuid.Nil {
		return nil, domain.ErrInvalidToken
	}

	return s.repository.UpdateAddress(userID, addressID, req)
}

func (s *UserService) DeleteAddress(userID, addressID uuid.UUID) error {
	if userID == uuid.Nil || addressID == uuid.Nil {
		return domain.ErrInvalidToken
	}

	return s.repository.DeleteAddress(userID, addressID)
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func generateJWT(userID, email, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
		"type":  "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET is not set")
	}

	return token.SignedString([]byte(secret))
}

func generateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type": "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secret := os.Getenv("JWT_REFRESH_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		return "", errors.New("JWT_REFRESH_SECRET/JWT_SECRET is not set")
	}

	return token.SignedString([]byte(secret))
}

func validateRefreshToken(refreshToken string) (jwt.MapClaims, error) {
	secret := os.Getenv("JWT_REFRESH_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		return nil, errors.New("JWT_REFRESH_SECRET/JWT_SECRET is not set")
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, domain.ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrExpiredToken
		}
		return nil, domain.ErrInvalidToken
	}

	if !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	typeClaim, _ := claims["type"].(string)
	if typeClaim != "refresh" {
		return nil, domain.ErrInvalidToken
	}

	return claims, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
