package handler

import (
	"auron/user-service/internal/domain"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserHandler struct {
	service domain.UserService
}

func NewUserHandler(service domain.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
		return
	}

	user, err := h.service.RegisterUser(&req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, domain.UserEnvelopeResponse{User: toUserResponse(user)})
}

func (h *UserHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	issuer, ok := h.service.(interface {
		LoginWithTokens(req *domain.LoginRequest) (*domain.AuthResponse, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "service does not support token response"})
		return
	}

	authResponse, err := issuer.LoginWithTokens(&req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.applyCookie(c, authResponse.AccessTokenCookie)
	h.applyCookie(c, authResponse.RefreshTokenCookie)

	c.JSON(http.StatusOK, domain.LoginResponse{
		AccessToken:  authResponse.AccessToken,
		RefreshToken: authResponse.RefreshToken,
	})
}

func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req domain.RefreshTokenRequest
	_ = c.ShouldBindJSON(&req)

	if req.RefreshToken == "" {
		refreshToken, err := c.Cookie("refresh_token")
		if err == nil {
			req.RefreshToken = refreshToken
		}
	}

	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "refresh token is required"})
		return
	}

	issuer, ok := h.service.(interface {
		RefreshTokenWithTokens(req *domain.RefreshTokenRequest) (*domain.AuthResponse, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "service does not support token response"})
		return
	}

	authResponse, err := issuer.RefreshTokenWithTokens(&req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.applyCookie(c, authResponse.AccessTokenCookie)
	h.applyCookie(c, authResponse.RefreshTokenCookie)

	c.JSON(http.StatusOK, domain.RefreshTokenResponse{
		AccessToken:  authResponse.AccessToken,
		RefreshToken: authResponse.RefreshToken,
	})
}

func (h *UserHandler) Logout(c *gin.Context) {
	var req domain.RevokeTokenRequest
	_ = c.ShouldBindJSON(&req)

	if req.RefreshToken == "" {
		refreshToken, err := c.Cookie("refresh_token")
		if err == nil {
			req.RefreshToken = refreshToken
		}
	}

	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "refresh token is required"})
		return
	}

	if err := h.service.RevokeToken(&req); err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.clearCookie(c, "access_token")
	h.clearCookie(c, "refresh_token")

	c.JSON(http.StatusOK, domain.MessageResponse{Message: "logged out"})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	user, err := h.service.GetUserProfile(userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, domain.UserEnvelopeResponse{User: toUserResponse(user)})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
		return
	}

	user, err := h.service.UpdateUserProfile(userID, &req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, domain.UserEnvelopeResponse{User: toUserResponse(user)})
}

func (h *UserHandler) AddAddress(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	var req domain.CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
		return
	}

	address := &domain.Address{
		Label:      req.Label,
		Street:     req.Street,
		City:       req.City,
		State:      req.State,
		Country:    req.Country,
		PostalCode: req.PostalCode,
		IsDefault:  req.IsDefault,
	}

	createdAddress, err := h.service.AddAddress(userID, address)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, domain.AddressEnvelopeResponse{Address: toAddressResponse(createdAddress)})
}

func (h *UserHandler) GetAddresses(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	addresses, err := h.service.GetAddresses(userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	response := make([]domain.AddressResponse, 0, len(addresses))
	for index := range addresses {
		addr := addresses[index]
		response = append(response, toAddressResponse(&addr))
	}

	c.JSON(http.StatusOK, domain.AddressesEnvelopeResponse{Addresses: response})
}

func (h *UserHandler) UpdateAddress(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	addressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "invalid address id"})
		return
	}

	var req domain.UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
		return
	}

	address := &domain.Address{
		ID:         addressID,
		UserID:     userID,
		Label:      req.Label,
		State:      req.State,
		PostalCode: req.PostalCode,
	}
	if req.Street != nil {
		address.Street = *req.Street
	}
	if req.City != nil {
		address.City = *req.City
	}
	if req.Country != nil {
		address.Country = *req.Country
	}
	if req.IsDefault != nil {
		address.IsDefault = *req.IsDefault
	}

	updatedAddress, updateErr := h.service.UpdateAddress(address)
	if updateErr != nil {
		h.handleServiceError(c, updateErr)
		return
	}

	c.JSON(http.StatusOK, domain.AddressEnvelopeResponse{Address: toAddressResponse(updatedAddress)})
}

func (h *UserHandler) DeleteAddress(c *gin.Context) {
	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "address id is required"})
		return
	}

	if err := h.service.DeleteAddress(addressID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, domain.DeleteAddressResponse{Message: "address deleted"})
}

func (h *UserHandler) AuthMiddleware(c *gin.Context) {
	tokenString := extractBearerToken(c.GetHeader("Authorization"))
	if tokenString == "" {
		cookieToken, err := c.Cookie("access_token")
		if err == nil {
			tokenString = cookieToken
		}
	}

	if tokenString == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "JWT_SECRET is not set"})
		return
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, domain.ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	typeClaim, _ := claims["type"].(string)
	if typeClaim != "access" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, domain.ErrorResponse{Error: domain.ErrUnauthorized.Error()})
		return
	}

	c.Set("user_id", userID.String())
	c.Next()
}

func (h *UserHandler) applyCookie(c *gin.Context, cfg domain.CookieConfig) {
	c.SetSameSite(toSameSiteMode(cfg.SameSite))
	c.SetCookie(cfg.Name, cfg.Value, cfg.MaxAge, cfg.Path, "", cfg.Secure, cfg.HttpOnly)
}

func (h *UserHandler) clearCookie(c *gin.Context, name string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(name, "", -1, "/", "", true, true)
}

func (h *UserHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrExpiredToken):
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, domain.ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrUserNotFound):
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, domain.ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrPasswordMismatch):
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: err.Error()})
	}
}

func toUserResponse(user *domain.User) domain.UserResponse {
	if user == nil {
		return domain.UserResponse{}
	}

	return domain.UserResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}
}

func toAddressResponse(address *domain.Address) domain.AddressResponse {
	if address == nil {
		return domain.AddressResponse{}
	}

	return domain.AddressResponse{
		ID:         address.ID,
		Label:      address.Label,
		Street:     address.Street,
		City:       address.City,
		State:      address.State,
		Country:    address.Country,
		PostalCode: address.PostalCode,
		IsDefault:  address.IsDefault,
	}
}

func toSameSiteMode(value string) http.SameSite {
	switch strings.ToLower(value) {
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func userIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	value, ok := c.Get("user_id")
	if !ok {
		return uuid.Nil, false
	}

	userID, ok := value.(string)
	if !ok || userID == "" {
		return uuid.Nil, false
	}

	parsed, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, false
	}

	return parsed, true
}
