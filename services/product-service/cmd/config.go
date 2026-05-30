package cmd

import "os"

type appConfig struct {
	Port           string
	DatabaseURL    string
	RedisURL       string
	KafkaBrokers   string
	GCSBucketName  string
	GCSCredentials string
}

func loadConfig() appConfig {
	loadDotEnvFile(".env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://auron:auron_pass@localhost:5433/products_db?sslmode=disable"
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
		Port:           port,
		DatabaseURL:    databaseURL,
		RedisURL:       redisURL,
		KafkaBrokers:   kafkaBrokers,
		GCSBucketName:  os.Getenv("GCS_BUCKET_NAME"),
		GCSCredentials: os.Getenv("GCS_CREDENTIALS_JSON"),
	}
}
