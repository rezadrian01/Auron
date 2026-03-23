package cmd

import (
	"auron/user-service/internal/cache"
	"auron/user-service/internal/handler"
	"auron/user-service/internal/repository"
	"auron/user-service/internal/service"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"auron/user-service/internal/domain"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func Run() {
	config := loadConfig()

	db, err := setupDatabase(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed")

	redisClient, err := setupRedis(config.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	userRepository := repository.NewUserRepository(db)
	userCache := cache.NewUserCache(redisClient)
	publisher := setupKafkaPublisher(config.KafkaBrokers)

	userService := service.NewUserService(userRepository, userCache, publisher)
	userHandler := handler.NewUserHandler(userService)

	router := setupRouter(userHandler)
	registerGracefulShutdown(db, redisClient, publisher)

	addr := fmt.Sprintf(":%s", config.Port)
	fmt.Printf("Starting User Service on %s\n", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func registerGracefulShutdown(db *gorm.DB, redisClient *redis.Client, publisher domain.EventPublisher) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		closeKafkaPublisher(publisher)
		_ = redisClient.Close()
		os.Exit(0)
	}()
}
