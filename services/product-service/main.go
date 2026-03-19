package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/auron/product-service/config"
	"github.com/auron/product-service/handlers"
	"github.com/auron/product-service/repository"
	"github.com/auron/product-service/service"
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

	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)

	productSvc := service.NewProductService(productRepo, categoryRepo, redisClient, cfg)
	handler := handlers.NewHandler(productSvc)

	router := setupRouter(handler, cfg)

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
	fmt.Printf("Starting Product Service on %s\n", addr)
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

func setupRouter(handler *handlers.Handler, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "product-service",
			"timestamp": time.Now().UTC(),
		})
	})

	api := router.Group("/")
	{
		api.GET("/products", handler.ListProducts)
		api.GET("/products/:id", handler.GetProduct)

		// Protected admin routes
		protected := api.Group("")
		protected.Use(handler.AuthMiddleware)
		{
			protected.POST("/products", handler.CreateProduct)
			protected.PUT("/products/:id", handler.UpdateProduct)
			protected.DELETE("/products/:id", handler.DeleteProduct)
			protected.POST("/categories", handler.CreateCategory)
		}

		api.GET("/categories", handler.ListCategories)
	}

	return router
}
