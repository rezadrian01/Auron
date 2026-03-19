package handlers

import (
	"net/http"
	"strings"

	"github.com/auron/user-service/models"
	"github.com/auron/user-service/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Request/Response types
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UpdateProfileRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type AddAddressRequest struct {
	Label      string `json:"label"`
	Street     string `json:"street" binding:"required"`
	City       string `json:"city" binding:"required"`
	State      string `json:"state"`
	Country    string `json:"country" binding:"required"`
	PostalCode string `json:"postal_code"`
	IsDefault  bool   `json:"is_default"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// Handler handles HTTP requests
type Handler struct {
	service *service.UserService
}

// NewHandler creates a new handler
func NewHandler(svc *service.UserService) *Handler {
	return &Handler{service: svc}
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	user, err := h.service.Register(req.Email, req.Password, req.Name)
	if err != nil {
		if err == service.ErrEmailExists {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error:   "Email already registered",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to register user",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
	})
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	tokens, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		if err == service.ErrUserNotFound || err == service.ErrInvalidPassword {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Success: false,
				Error:   "Invalid email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to login",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
			"expires_in":   tokens.ExpiresIn,
		},
	})
}

// RefreshToken handles token refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	tokens, err := h.service.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Invalid or expired refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
			"expires_in":   tokens.ExpiresIn,
		},
	})
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	// Get refresh token from header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Authorization header required",
		})
		return
	}

	// Extract refresh token (we'll use a simple approach - in real app, send via body or header)
	refreshToken := strings.TrimPrefix(authHeader, "Bearer ")
	if refreshToken == authHeader {
		// Try to get from body
		var req RefreshRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken != "" && refreshToken != authHeader {
		h.service.Logout(refreshToken)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

// GetProfile handles getting user profile
func (h *Handler) GetProfile(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	uid, _ := uuid.Parse(userID.(string))
	user, err := h.service.GetProfile(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
	})
}

// UpdateProfile handles updating user profile
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	uid, _ := uuid.Parse(userID.(string))
	user, err := h.service.UpdateProfile(uid, req.Name, req.Email)
	if err != nil {
		if err == service.ErrEmailExists {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error:   "Email already in use",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to update profile",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
	})
}

// GetAddresses handles getting user addresses
func (h *Handler) GetAddresses(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	uid, _ := uuid.Parse(userID.(string))
	addresses, err := h.service.GetAddresses(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to get addresses",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"addresses": addresses,
	})
}

// AddAddress handles adding user address
func (h *Handler) AddAddress(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	var req AddAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	uid, _ := uuid.Parse(userID.(string))
	address := &models.Address{
		Label:      req.Label,
		Street:     req.Street,
		City:       req.City,
		State:      req.State,
		Country:    req.Country,
		PostalCode: req.PostalCode,
		IsDefault:  req.IsDefault,
	}

	err := h.service.AddAddress(uid, address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to add address",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"address": address,
	})
}

// DeleteAddress handles deleting user address
func (h *Handler) DeleteAddress(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	addressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid address ID",
		})
		return
	}

	uid, _ := uuid.Parse(userID.(string))
	err = h.service.DeleteAddress(addressID, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to delete address",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message": "Address deleted successfully",
	})
}

// AuthMiddleware is authentication middleware
func (h *Handler) AuthMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Authorization header required",
		})
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Invalid authorization header format",
		})
		return
	}

	claims, err := h.service.ValidateToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Invalid or expired token",
		})
		return
	}

	// Set user info in context
	c.Set("user_id", claims.UserID)
	c.Set("user_email", claims.Email)
	c.Set("user_role", claims.Role)

	c.Next()
}
