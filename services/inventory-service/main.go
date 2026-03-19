package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/auron/inventory-service/config"
	"github.com/auron/inventory-service/handlers"
	"github.com/auron/inventory-service/repository"
	"github.com/auron/inventory-service/service"
	"github.com/gin-gonic/gin"
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

	inventoryRepo := repository.NewInventoryRepository(db)
	inventorySvc := service.NewInventoryService(inventoryRepo, cfg)
	handler := handlers.NewHandler(inventorySvc)

	router := setupRouter(handler, inventorySvc)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")
		db, _ := db.DB()
		db.Close()
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%s", cfg.Port)
	fmt.Printf("Starting Inventory Service on %s\n", addr)
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

func setupRouter(handler *handlers.Handler, svc *service.InventoryService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "inventory-service",
			"timestamp": time.Now().UTC(),
		})
	})

	api := router.Group("/")

	// Start Kafka consumer in background
	go svc.StartConsumer()

	{
		api.GET("/inventory/:product_id", handler.GetInventory)
		api.PUT("/inventory/:product_id", handler.UpdateInventory)
	}

	return router
}
