package config

import (
	"os"
	"time"
)

type Config struct {
	Port       string
	DatabaseURL string
	RedisURL   string
	CacheTTL   time.Duration
	KafkaBrokers []string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8082"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://auron:auron_pass@localhost:5433/products_db?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		CacheTTL:    getEnvDuration("CACHE_TTL", 5*time.Minute),
		KafkaBrokers: getEnvSlice("KAFKA_BROKERS", "localhost:9092"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvSlice(key, defaultValue string) []string {
	if value := os.Getenv(key); value != "" {
		return []string{value}
	}
	return []string{defaultValue}
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
