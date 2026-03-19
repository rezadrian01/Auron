package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/auron/user-service/config"
	"github.com/auron/user-service/models"
	"github.com/auron/user-service/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrEmailExists     = errors.New("email already exists")
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")

	RoleCustomer = "customer"
	RoleAdmin    = "admin"
)

// TokenResponse represents the login response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Claims represents JWT claims
type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// UserService handles user business logic
type UserService struct {
	repo        *repository.UserRepository
	addressRepo *repository.AddressRepository
	redis       *redis.Client
	cfg         *config.Config
	privateKey  *rsa.PrivateKey
}

// NewUserService creates a new user service
func NewUserService(repo *repository.UserRepository, addressRepo *repository.AddressRepository, redisClient *redis.Client, cfg *config.Config) *UserService {
	// Load or generate RSA key
	privateKey := loadOrGeneratePrivateKey(cfg)

	return &UserService{
		repo:        repo,
		addressRepo: addressRepo,
		redis:       redisClient,
		cfg:         cfg,
		privateKey:  privateKey,
	}
}

// Register creates a new user
func (s *UserService) Register(email, password, name string) (*models.User, error) {
	// Check if email already exists
	existing, _ := s.repo.FindByEmail(email)
	if existing != nil {
		return nil, ErrEmailExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:    email,
		Password: string(hashedPassword),
		Name:     name,
		Role:     RoleCustomer,
	}

	err = s.repo.Create(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *UserService) Login(email, password string) (*TokenResponse, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, ErrInvalidPassword
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken := uuid.New().String()

	// Store refresh token in Redis
	err = s.redis.Set(context.Background(), "refresh:"+refreshToken, user.ID.String(), s.cfg.JWTRefreshTTL).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.JWTAccessTTL.Seconds()),
	}, nil
}

// RefreshToken refreshes access token using refresh token
func (s *UserService) RefreshToken(refreshToken string) (*TokenResponse, error) {
	// Get user ID from Redis
	userID, err := s.redis.Get(context.Background(), "refresh:"+refreshToken).Result()
	if err == redis.Nil {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Get user
	uid, _ := uuid.Parse(userID)
	user, err := s.repo.FindByID(uid)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.JWTAccessTTL.Seconds()),
	}, nil
}

// Logout invalidates refresh token
func (s *UserService) Logout(refreshToken string) error {
	err := s.redis.Del(context.Background(), "refresh:"+refreshToken).Err()
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

// GetProfile gets user profile
func (s *UserService) GetProfile(userID uuid.UUID) (*models.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// UpdateProfile updates user profile
func (s *UserService) UpdateProfile(userID uuid.UUID, name, email string) (*models.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Check if new email is already taken by another user
	if email != user.Email {
		existing, _ := s.repo.FindByEmail(email)
		if existing != nil {
			return nil, ErrEmailExists
		}
		user.Email = email
	}

	user.Name = name
	err = s.repo.Update(user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// GetAddresses gets user addresses
func (s *UserService) GetAddresses(userID uuid.UUID) ([]models.Address, error) {
	return s.addressRepo.FindByUserID(userID)
}

// AddAddress adds a new address
func (s *UserService) AddAddress(userID uuid.UUID, address *models.Address) error {
	address.UserID = userID
	return s.addressRepo.Create(address)
}

// DeleteAddress deletes an address
func (s *UserService) DeleteAddress(addressID, userID uuid.UUID) error {
	// Verify address belongs to user
	addresses, err := s.addressRepo.FindByUserID(userID)
	if err != nil {
		return err
	}

	for _, addr := range addresses {
		if addr.ID == addressID {
			return s.addressRepo.Delete(addressID)
		}
	}

	return errors.New("address not found")
}

// ValidateToken validates JWT token and returns claims
func (s *UserService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &s.privateKey.PublicKey, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// generateAccessToken generates a new JWT access token
func (s *UserService) generateAccessToken(user *models.User) (string, error) {
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTAccessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   user.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// loadOrGeneratePrivateKey loads or generates RSA private key
func loadOrGeneratePrivateKey(cfg *config.Config) *rsa.PrivateKey {
	// Try to load existing key
	if cfg.JWTPrivateKey != "" {
		data, err := os.ReadFile(cfg.JWTPrivateKey)
		if err == nil {
			key, err := jwt.ParseRSAPrivateKeyFromPEM(data)
			if err == nil {
				return key
			}
		}
	}

	// Generate new key
	fmt.Println("Generating new RSA key pair...")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("failed to generate RSA key: %v", err))
	}

	return privateKey
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPasswordHash compares password with hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomToken generates a random token
func GenerateRandomToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

// HashToken creates SHA256 hash of token (for storage)
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(hash[:])
}
