package config

import (
	"os"
)

type Config struct {
	Port            string
	SendGridAPIKey  string
	FromEmail       string
	TwilioAccountSID string
	TwilioAuthToken string
	TwilioFromNumber string
	KafkaBrokers    []string
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8086"),
		SendGridAPIKey:   getEnv("SENDGRID_API_KEY", ""),
		FromEmail:        getEnv("FROM_EMAIL", "noreply@auron.shop"),
		TwilioAccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioFromNumber: getEnv("TWILIO_FROM_NUMBER", ""),
		KafkaBrokers:     getEnvSlice("KAFKA_BROKERS", "localhost:9092"),
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
