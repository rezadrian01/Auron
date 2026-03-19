package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(key string) (bool, error)
}

// RedisRateLimiter implements rate limiting using Redis
type RedisRateLimiter struct {
	client      *redis.Client
	maxRequest  int
	window      time.Duration
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(redisURL string, maxRequests int, window time.Duration) (*RedisRateLimiter, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	return &RedisRateLimiter{
		client:     client,
		maxRequest: maxRequests,
		window:     window,
	}, nil
}

// RateLimit returns a rate limiting middleware
func RateLimiter(limiter RateLimiter) gin.HandlerFunc {
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

// Allow checks if the request is allowed
func (r *RedisRateLimiter) Allow(key string) (bool, error) {
	// Increment counter
	count, err := r.client.Incr(key).Result()
	if err != nil {
		return false, err
	}

	// Set expiration on first request
	if count == 1 {
		r.client.Expire(key, r.window)
	}

	return count <= int64(r.maxRequest), nil
}

// NewRateLimiter creates a simple in-memory rate limiter (for testing)
func NewRateLimiter(maxRequests int, window time.Duration) *SimpleRateLimiter {
	return &SimpleRateLimiter{
		maxRequests: maxRequests,
		window:      window,
		requests:    make(map[string][]time.Time),
	}
}

// SimpleRateLimiter is an in-memory rate limiter
type SimpleRateLimiter struct {
	maxRequests int
	window      time.Duration
	requests    map[string][]time.Time
}

// Allow checks if the request is allowed
func (s *SimpleRateLimiter) Allow(key string) (bool, error) {
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
