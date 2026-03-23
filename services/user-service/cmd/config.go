package cmd

import "os"

type appConfig struct {
	DatabaseURL  string
	RedisURL     string
	Port         string
	KafkaBrokers string
}

func loadConfig() appConfig {
	loadDotEnvFile(".env")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://auron:auron_pass@localhost:5432/users_db?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	return appConfig{
		DatabaseURL:  databaseURL,
		RedisURL:     redisURL,
		Port:         port,
		KafkaBrokers: kafkaBrokers,
	}
}
