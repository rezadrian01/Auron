package config

import (
	"os"
)

type Config struct {
	Port       string
	DatabaseURL string
	RedisURL   string
	KafkaBrokers []string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8083"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://auron:auron_pass@localhost:5434/orders_db?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
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
