package cmd

import (
	"time"

	"auron/product-service/internal/handler"
	"auron/product-service/internal/route"

	"github.com/gin-gonic/gin"
)

func setupRouter(h *handler.ProductHandler) *gin.Engine {
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

	router.GET("/metrics", func(c *gin.Context) {
		c.String(200, "# Prometheus metrics endpoint\n")
	})

	route.RegisterProductRoutes(router, h)

	return router
}
