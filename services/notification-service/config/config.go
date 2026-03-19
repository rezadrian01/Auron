package config

import (
	"os"
)

type Config struct {
	Port             string
	SMTPHost         string
	SMTPPort         string
	SMTPFrom         string
	SMTPSecure       bool
	SMTPUser         string
	SMTPPass         string
	KafkaBrokers     []string
}

func Load() *Config {
	return &Config{
		Port:       getEnv("PORT", "8086"),
		SMTPHost:   getEnv("SMTP_HOST", "smtp-host"),
		SMTPPort:   getEnv("SMTP_PORT", "587"),
		SMTPFrom:   getEnv("SMTP_FROM", "noreply@auron.shop"),
		SMTPSecure: getEnv("SMTP_SECURE", "false") == "true",
		SMTPUser:   getEnv("SMTP_USER", ""),
		SMTPPass:   getEnv("SMTP_PASS", ""),
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
