package cmd

import (
	"time"

	"auron/order-service/internal/handler"
	"auron/order-service/internal/route"

	"github.com/gin-gonic/gin"
)

func setupRouter(cartHandler *handler.CartHandler, orderHandler *handler.OrderHandler) *gin.Engine {
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

	router.GET("/metrics", func(c *gin.Context) {
		c.String(200, "# Prometheus metrics endpoint\n")
	})

	route.RegisterOrderRoutes(router, cartHandler, orderHandler)

	return router
}
