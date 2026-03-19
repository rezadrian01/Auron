package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds API Gateway configuration
type Config struct {
	Port               string
	UserServiceURL     string
	ProductServiceURL  string
	OrderServiceURL    string
	PaymentServiceURL  string
	InventoryServiceURL string
	RedisURL           string
	JWTPublicKeyPath   string
	RateLimitRequests  int
	RateLimitWindow    time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		UserServiceURL:      getEnv("USER_SERVICE_URL", "http://localhost:8081"),
		ProductServiceURL:   getEnv("PRODUCT_SERVICE_URL", "http://localhost:8082"),
		OrderServiceURL:    getEnv("ORDER_SERVICE_URL", "http://localhost:8083"),
		PaymentServiceURL:  getEnv("PAYMENT_SERVICE_URL", "http://localhost:8084"),
		InventoryServiceURL: getEnv("INVENTORY_SERVICE_URL", "http://localhost:8085"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTPublicKeyPath:    getEnv("JWT_PUBLIC_KEY", "/run/secrets/jwt-public-key"),
		RateLimitRequests:  getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:    getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
