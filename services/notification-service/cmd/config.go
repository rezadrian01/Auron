package cmd

import (
	"os"
	"strconv"
)

type appConfig struct {
	Port         string
	SMTPHost     string
	SMTPPort     int
	SMTPFrom     string
	SMTPUser     string
	SMTPPass     string
	SMTPSecure   bool
	KafkaBrokers string
}

func loadConfig() appConfig {
	loadDotEnvFile(".env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "localhost"
	}

	smtpPort := 1025
	if v := os.Getenv("SMTP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			smtpPort = p
		}
	}

	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = "noreply@auron.shop"
	}

	smtpSecure := os.Getenv("SMTP_SECURE") == "true"

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	return appConfig{
		Port:         port,
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPFrom:     smtpFrom,
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPass:     os.Getenv("SMTP_PASS"),
		SMTPSecure:   smtpSecure,
		KafkaBrokers: kafkaBrokers,
	}
}
