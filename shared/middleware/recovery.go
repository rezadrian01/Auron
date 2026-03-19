// Package middleware provides shared middleware for all Auron services.
package middleware

import (
	"net/http"
	"os"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery returns a middleware that recovers from any panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := debug.Stack()

				// Log the error
				gin.DefaultWriter.Write([]byte("[PANIC RECOVERED]\n"))
				gin.DefaultWriter.Write([]byte("Error: "))
				gin.DefaultWriter.Write([]byte(err.(error).Error()))
				gin.DefaultWriter.Write([]byte("\n\nStack:\n"))
				gin.DefaultWriter.Write(stack)

				// Get service name from environment or default
				serviceName := os.Getenv("SERVICE_NAME")
				if serviceName == "" {
					serviceName = "auron-service"
				}

				// Abort with 500 Internal Server Error
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INTERNAL_SERVER_ERROR",
						"message": "An unexpected error occurred",
						"service": serviceName,
					},
				})
			}
		}()
		c.Next()
	}
}
