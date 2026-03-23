package cmd

import "os"

type appConfig struct {
	DatabaseURL string
	RedisURL    string
	Port        string
}

func loadConfig() appConfig {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/users_db?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	return appConfig{
		DatabaseURL: databaseURL,
		RedisURL:    redisURL,
		Port:        port,
	}
}
