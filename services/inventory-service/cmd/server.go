package cmd

import (
	"time"

	"auron/inventory-service/internal/handler"
	"auron/inventory-service/internal/route"

	"github.com/gin-gonic/gin"
)

func setupRouter(inventoryHandler *handler.InventoryHandler) *gin.Engine {
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

	router.GET("/metrics", func(c *gin.Context) {
		c.String(200, "# Prometheus metrics endpoint\n")
	})

	route.RegisterInventoryRoutes(router, inventoryHandler)

	return router
}
