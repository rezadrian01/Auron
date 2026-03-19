package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	DatabaseURL     string
	RedisURL        string
	JWTPrivateKey   string
	JWTPublicKey    string
	JWTAccessTTL    time.Duration
	JWTRefreshTTL   time.Duration
	KafkaBrokers    []string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8081"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://auron:auron_pass@localhost:5432/users_db?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTPrivateKey:  getEnv("JWT_PRIVATE_KEY", ""),
		JWTPublicKey:   getEnv("JWT_PUBLIC_KEY", ""),
		JWTAccessTTL:   getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:  getEnvDuration("JWT_REFRESH_TTL", 168*time.Hour), // 7 days
		KafkaBrokers:   getEnvSlice("KAFKA_BROKERS", "localhost:9092"),
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

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
