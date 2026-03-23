package middleware

import (
	"auron/user-service/internal/domain"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const userIDContextKey = "user_id"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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

		c.Set(userIDContextKey, userID.String())
		c.Next()
	}
}

func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	value, ok := c.Get(userIDContextKey)
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
