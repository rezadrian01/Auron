package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/auron/user-service/config"
	"github.com/auron/user-service/handlers"
	"github.com/auron/user-service/models"
	"github.com/auron/user-service/repository"
	"github.com/auron/user-service/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup database
	db, err := setupDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate database tables
	if err := db.AutoMigrate(&models.User{}, &models.Address{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed")

	// Setup Redis
	redisClient, err := setupRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Setup repositories
	userRepo := repository.NewUserRepository(db)
	addressRepo := repository.NewAddressRepository(db)

	// Setup services
	userSvc := service.NewUserService(userRepo, addressRepo, redisClient, cfg)

	// Setup handlers
	handler := handlers.NewHandler(userSvc)

	// Setup router
	router := setupRouter(handler)

	// Graceful shutdown
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

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	fmt.Printf("Starting User Service on %s\n", addr)
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

	// Configure connection pool
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

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Request ID middleware
	router.Use(func(c *gin.Context) {
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "user-service",
			"timestamp": time.Now().UTC(),
		})
	})

	// API routes
	api := router.Group("/")
	{
		// Auth routes (no auth required)
		api.POST("/register", handler.Register)
		api.POST("/login", handler.Login)
		api.POST("/refresh", handler.RefreshToken)

		// Protected routes
		protected := api.Group("")
		protected.Use(handler.AuthMiddleware)
		{
			protected.POST("/logout", handler.Logout)
			protected.GET("/me", handler.GetProfile)
			protected.PUT("/me", handler.UpdateProfile)
			protected.POST("/me/addresses", handler.AddAddress)
			protected.GET("/me/addresses", handler.GetAddresses)
			protected.DELETE("/me/addresses/:id", handler.DeleteAddress)
		}
	}

	return router
}
