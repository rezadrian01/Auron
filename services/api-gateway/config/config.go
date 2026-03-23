package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ServiceUser         = "user"
	ServiceProduct      = "product"
	ServiceOrder        = "order"
	ServicePayment      = "payment"
	ServiceInventory    = "inventory"
	ServiceNotification = "notification"
)

// Config holds API Gateway configuration
type Config struct {
	Port              string
	ServiceURLs       map[string]string
	RedisURL          string
	JWTPublicKeyPath  string
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	serviceURLs := map[string]string{
		ServiceUser:      getEnv("USER_SERVICE_URL", "http://localhost:8081"),
		ServiceProduct:   getEnv("PRODUCT_SERVICE_URL", "http://localhost:8082"),
		ServiceOrder:     getEnv("ORDER_SERVICE_URL", "http://localhost:8083"),
		ServicePayment:   getEnv("PAYMENT_SERVICE_URL", "http://localhost:8084"),
		ServiceInventory: getEnv("INVENTORY_SERVICE_URL", "http://localhost:8085"),
	}

	for key, value := range parseServiceURLs(getEnv("SERVICE_URLS", "")) {
		serviceURLs[key] = value
	}

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(key, "SERVICE_URL_") || value == "" {
			continue
		}

		serviceName := strings.ToLower(strings.TrimPrefix(key, "SERVICE_URL_"))
		serviceName = strings.ReplaceAll(serviceName, "_", "-")
		if serviceName == "" {
			continue
		}

		serviceURLs[serviceName] = value
	}

	return &Config{
		Port:              getEnv("PORT", "8080"),
		ServiceURLs:       serviceURLs,
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTPublicKeyPath:  getEnv("JWT_PUBLIC_KEY", "/run/secrets/jwt-public-key"),
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
	}
}

func parseServiceURLs(raw string) map[string]string {
	serviceURLs := make(map[string]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return serviceURLs
	}

	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.ToLower(strings.TrimSpace(parts[0]))
		url := strings.TrimSpace(parts[1])
		if name == "" || url == "" {
			continue
		}

		serviceURLs[name] = url
	}

	return serviceURLs
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
