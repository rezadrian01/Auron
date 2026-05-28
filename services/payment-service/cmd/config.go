package cmd

import "os"

type appConfig struct {
	Port                string
	DatabaseURL         string
	RedisURL            string
	KafkaBrokers        string
	StripeSecretKey     string
	StripeWebhookSecret string
}

func loadConfig() appConfig {
	loadDotEnvFile(".env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://auron:auron_pass@localhost:5435/payments_db?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	return appConfig{
		Port:                port,
		DatabaseURL:         databaseURL,
		RedisURL:            redisURL,
		KafkaBrokers:        kafkaBrokers,
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}
}
