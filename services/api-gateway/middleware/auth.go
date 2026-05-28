package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	AuthorizationHeader = "Authorization"
	UserIDKey           = "user_id"
	UserEmailKey        = "user_email"
	UserRoleKey         = "user_role"
)

// Claims represents JWT claims produced by user-service (HS256).
// The user UUID lives in the standard "sub" field (RegisteredClaims.Subject).
type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"`
}

// JWTMiddleware validates HS256 tokens signed with a shared secret.
type JWTMiddleware struct {
	secret []byte
}

// NewJWTMiddleware creates a JWT middleware from the shared HMAC secret.
func NewJWTMiddleware(secret string) (*JWTMiddleware, error) {
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET must not be empty")
	}
	return &JWTMiddleware{secret: []byte(secret)}, nil
}

// parseToken validates the token string and returns its claims.
func (j *JWTMiddleware) parseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid or expired token")
	}
	return claims, nil
}

// Auth returns a middleware that validates the JWT and populates context keys.
// Proceeds even if validation fails — use RequireAuth to block unauthenticated requests.
func (j *JWTMiddleware) Auth() gin.HandlerFunc {
	return j.RequireAuth()
}

// RequireAuth returns a middleware that rejects requests without a valid JWT.
func (j *JWTMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractBearerToken(c.GetHeader(AuthorizationHeader))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header is required",
				},
			})
			return
		}

		claims, err := j.parseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
			})
			return
		}

		// Subject holds the user UUID ("sub" claim set by user-service)
		c.Set(UserIDKey, claims.Subject)
		c.Set(UserEmailKey, claims.Email)
		c.Set(UserRoleKey, claims.Role)

		c.Next()
	}
}

// RequireRole returns a middleware that requires a specific role
func (j *JWTMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(UserRoleKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization required",
				},
			})
			return
		}

		userRole := role.(string)
		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Insufficient permissions",
			},
		})
	}
}

func extractBearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// GetUserID returns the user ID from the context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get(UserIDKey); exists {
		return userID.(string)
	}
	return ""
}

// GetUserEmail returns the user email from the context
func GetUserEmail(c *gin.Context) string {
	if email, exists := c.Get(UserEmailKey); exists {
		return email.(string)
	}
	return ""
}

// GetUserRole returns the user role from the context
func GetUserRole(c *gin.Context) string {
	if role, exists := c.Get(UserRoleKey); exists {
		return role.(string)
	}
	return ""
}
