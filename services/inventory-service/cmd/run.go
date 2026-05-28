package cmd

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"auron/inventory-service/internal/cache"
	"auron/inventory-service/internal/domain"
	"auron/inventory-service/internal/events"
	"auron/inventory-service/internal/handler"
	"auron/inventory-service/internal/repository"
	"auron/inventory-service/internal/service"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func Run() {
	cfg := loadConfig()

	db, err := setupDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	log.Println("database connected")

	redisClient, err := setupRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	publisher := setupKafkaPublisher(cfg.KafkaBrokers)

	inventoryRepo := repository.NewInventoryRepository(db)
	inventoryCache := cache.NewInventoryCache(redisClient)
	inventorySvc := service.NewInventoryService(inventoryRepo, inventoryCache, publisher)

	ctx, cancel := context.WithCancel(context.Background())
	consumer := setupKafkaConsumer(cfg.KafkaBrokers, inventorySvc)
	startKafkaConsumer(ctx, consumer)

	inventoryHandler := handler.NewInventoryHandler(inventorySvc)
	router := setupRouter(inventoryHandler)

	registerGracefulShutdown(db, redisClient, publisher, consumer, cancel)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("starting inventory-service on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func registerGracefulShutdown(
	db *gorm.DB,
	redisClient *redis.Client,
	publisher domain.EventPublisher,
	consumer *events.KafkaConsumer,
	cancel context.CancelFunc,
) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nshutting down inventory-service...")
		cancel()
		if err := consumer.Close(); err != nil {
			slog.Warn("error closing kafka consumer", "error", err)
		}
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = redisClient.Close()
		closeKafkaPublisher(publisher)
		os.Exit(0)
	}()
}
