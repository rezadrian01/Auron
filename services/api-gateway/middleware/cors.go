package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a CORS middleware with configurable allowed origins
func CORS() gin.HandlerFunc {
	allowedOrigins := getAllowedOrigins()

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		// Allow credentials
		c.Header("Access-Control-Allow-Credentials", "true")

		// Allow methods
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")

		// Allow headers
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")

		// Expose headers
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")

		// Handle preflight
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func getAllowedOrigins() []string {
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	if originsEnv == "" {
		// Default: allow all in development
		return []string{"http://localhost:3000", "http://localhost:3001"}
	}
	return strings.Split(originsEnv, ",")
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		// Allow requests without origin (e.g., curl)
		return true
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		// Support wildcard subdomains
		if strings.HasPrefix(allowed, "*.") {
			suffix := allowed[1:]
			if strings.HasSuffix(origin, suffix) {
				return true
			}
		}
		if allowed == origin {
			return true
		}
	}
	return false
}
