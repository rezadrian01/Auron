package middleware

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const RawBodyKey = "rawBody"

// CaptureRawBody reads and stores the raw request bytes before any binding.
// Required because Stripe signature verification needs the exact original bytes,
// and c.Request.Body is consumed after the first read.
func CaptureRawBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "failed to read request body"})
			c.Abort()
			return
		}
		c.Set(RawBodyKey, body)
		c.Next()
	}
}
