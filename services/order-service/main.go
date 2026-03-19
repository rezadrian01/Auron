package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/auron/order-service/config"
	"github.com/auron/order-service/handlers"
	"github.com/auron/order-service/repository"
	"github.com/auron/order-service/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.Load()

	db, err := setupDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	redisClient, err := setupRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	orderRepo := repository.NewOrderRepository(db)
	cartRepo := repository.NewCartRepository(redisClient)

	orderSvc := service.NewOrderService(orderRepo, cartRepo, cfg)
	handler := handlers.NewHandler(orderSvc)

	router := setupRouter(handler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")
		db, _ := db.DB()
		db.Close()
		redisClient.Close()
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%s", cfg.Port)
	fmt.Printf("Starting Order Service on %s\n", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupDatabase(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func setupRedis(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

func setupRouter(handler *handlers.Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "order-service",
			"timestamp": time.Now().UTC(),
		})
	})

	api := router.Group("/")
	protected := api.Group("")
	protected.Use(handler.AuthMiddleware)
	{
		protected.GET("/cart", handler.GetCart)
		protected.POST("/cart/items", handler.AddToCart)
		protected.PUT("/cart/items/:id", handler.UpdateCartItem)
		protected.DELETE("/cart/items/:id", handler.RemoveFromCart)
		protected.POST("/orders", handler.CreateOrder)
		protected.GET("/orders", handler.ListOrders)
		protected.GET("/orders/:id", handler.GetOrder)
		protected.PUT("/orders/:id/cancel", handler.CancelOrder)
	}

	return router
}
