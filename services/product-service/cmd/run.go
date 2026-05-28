package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"auron/product-service/internal/cache"
	"auron/product-service/internal/domain"
	"auron/product-service/internal/handler"
	"auron/product-service/internal/repository"
	"auron/product-service/internal/service"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func Run() {
	cfg := loadConfig()

	db, err := setupDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("database migrations completed")

	redisClient, err := setupRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	publisher := setupKafkaPublisher(cfg.KafkaBrokers)

	repo := repository.NewProductRepository(db)
	productCache := cache.NewProductCache(redisClient)
	svc := service.NewProductService(repo, productCache, publisher)
	h := handler.NewProductHandler(svc)

	router := setupRouter(h)
	registerGracefulShutdown(db, redisClient, publisher)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("starting product-service on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func registerGracefulShutdown(db *gorm.DB, redisClient *redis.Client, publisher domain.EventPublisher) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nshutting down product-service...")
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = redisClient.Close()
		closeKafkaPublisher(publisher)
		os.Exit(0)
	}()
}
