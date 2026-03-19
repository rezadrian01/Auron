package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(key string) (bool, error)
}

// RateLimit returns a rate limiting middleware
func RateLimit(limiter RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := c.ClientIP()

		// Create rate limit key
		key := fmt.Sprintf("ratelimit:%s:%d", clientIP, time.Now().Unix()/60)

		// Check if allowed
		allowed, err := limiter.Allow(key)
		if err != nil {
			// On error, allow the request but log
			c.Next()
			return
		}

		if !allowed {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				},
			})
			return
		}

		c.Next()
	}
}

// NewInMemoryRateLimiter creates a simple in-memory rate limiter
func NewInMemoryRateLimiter(maxRequests int, window time.Duration) *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		maxRequests: maxRequests,
		window:     window,
		mu:         sync.Mutex{},
		requests:   make(map[string][]time.Time),
	}
}

// InMemoryRateLimiter is an in-memory rate limiter
type InMemoryRateLimiter struct {
	maxRequests int
	window     time.Duration
	mu         sync.Mutex
	requests   map[string][]time.Time
}

// Allow checks if the request is allowed
func (s *InMemoryRateLimiter) Allow(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	oldestAllowed := now.Add(-s.window)

	// Clean old requests
	var validRequests []time.Time
	for _, t := range s.requests[key] {
		if t.After(oldestAllowed) {
			validRequests = append(validRequests, t)
		}
	}

	s.requests[key] = validRequests

	if len(validRequests) >= s.maxRequests {
		return false, nil
	}

	s.requests[key] = append(s.requests[key], now)
	return true, nil
}
