package cmd

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"auron/payment-service/internal/cache"
	"auron/payment-service/internal/client"
	"auron/payment-service/internal/domain"
	"auron/payment-service/internal/events"
	"auron/payment-service/internal/handler"
	"auron/payment-service/internal/repository"
	"auron/payment-service/internal/service"

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

	paymentRepo := repository.NewPaymentRepository(db)
	paymentCache := cache.NewPaymentCache(redisClient)
	stripeClient := client.NewStripeClient(cfg.StripeSecretKey)

	paymentSvc := service.NewPaymentService(paymentRepo, paymentCache, stripeClient, publisher, cfg.StripeWebhookSecret)

	ctx, cancel := context.WithCancel(context.Background())
	consumer := setupKafkaConsumer(cfg.KafkaBrokers, paymentSvc)
	startKafkaConsumer(ctx, consumer)

	paymentHandler := handler.NewPaymentHandler(paymentSvc)
	router := setupRouter(paymentHandler)

	registerGracefulShutdown(db, redisClient, publisher, consumer, cancel)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("starting payment-service on %s", addr)
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
		fmt.Println("\nshutting down payment-service...")
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
