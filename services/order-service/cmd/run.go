package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"auron/order-service/internal/cache"
	"auron/order-service/internal/client"
	"auron/order-service/internal/domain"
	"auron/order-service/internal/handler"
	"auron/order-service/internal/repository"
	"auron/order-service/internal/service"

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

	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	cartCache := cache.NewCartCache(redisClient)
	orderCache := cache.NewOrderCache(redisClient)
	productClient := client.NewProductClient(cfg.ProductServiceURL)

	cartSvc := service.NewCartService(cartRepo, cartCache, productClient)
	orderSvc := service.NewOrderService(orderRepo, cartRepo, orderCache, cartCache, publisher)

	cartHandler := handler.NewCartHandler(cartSvc)
	orderHandler := handler.NewOrderHandler(orderSvc)

	router := setupRouter(cartHandler, orderHandler)
	registerGracefulShutdown(db, redisClient, publisher)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("starting order-service on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func registerGracefulShutdown(db *gorm.DB, redisClient *redis.Client, publisher domain.EventPublisher) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nshutting down order-service...")
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = redisClient.Close()
		closeKafkaPublisher(publisher)
		os.Exit(0)
	}()
}
