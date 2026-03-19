package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/auron/api-gateway/config"
	"github.com/auron/api-gateway/middleware"
	"github.com/auron/api-gateway/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create Gin engine
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())

	// Health check endpoint (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "api-gateway",
			"timestamp": time.Now().UTC(),
		})
	})

	// Prometheus metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		c.String(200, "# Prometheus metrics endpoint\n")
	})

	// Setup routes
	routes.Setup(router, cfg)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")
		os.Exit(0)
	}()

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	fmt.Printf("Starting API Gateway on %s\n", addr)
	if err := router.Run(addr); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
