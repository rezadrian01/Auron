package config

import (
	"os"
)

type Config struct {
	Port             string
	DatabaseURL      string
	StripeSecretKey  string
	StripeWebhookSecret string
	KafkaBrokers     []string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8084"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://auron:auron_pass@localhost:5435/payments_db?sslmode=disable"),
		StripeSecretKey:    getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		KafkaBrokers:       getEnvSlice("KAFKA_BROKERS", "localhost:9092"),
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
