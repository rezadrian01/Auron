package cmd

import "os"

type appConfig struct {
	Port              string
	DatabaseURL       string
	RedisURL          string
	KafkaBrokers      string
	ProductServiceURL string
}

func loadConfig() appConfig {
	loadDotEnvFile(".env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://auron:auron_pass@localhost:5434/orders_db?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productServiceURL == "" {
		productServiceURL = "http://localhost:8082"
	}

	return appConfig{
		Port:              port,
		DatabaseURL:       databaseURL,
		RedisURL:          redisURL,
		KafkaBrokers:      kafkaBrokers,
		ProductServiceURL: productServiceURL,
	}
}
