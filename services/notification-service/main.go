package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/auron/notification-service/config"
	"github.com/auron/notification-service/handlers"
	"github.com/auron/notification-service/service"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	notificationSvc := service.NewNotificationService(cfg)
	handler := handlers.NewHandler(notificationSvc)

	router := setupRouter(handler)

	// Start Kafka consumers in background
	go notificationSvc.StartConsumers()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")
		notificationSvc.Stop()
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%s", cfg.Port)
	fmt.Printf("Starting Notification Service on %s\n", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter(handler *handlers.Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "notification-service",
		})
	})

	return router
}
