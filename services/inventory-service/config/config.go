package config

import (
	"os"
)

type Config struct {
	Port       string
	DatabaseURL string
	KafkaBrokers []string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8085"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://auron:auron_pass@localhost:5433/products_db?sslmode=disable"),
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
