package cmd

import (
	"time"

	"auron/payment-service/internal/handler"
	"auron/payment-service/internal/route"

	"github.com/gin-gonic/gin"
)

func setupRouter(paymentHandler *handler.PaymentHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "payment-service",
			"timestamp": time.Now().UTC(),
		})
	})

	router.GET("/metrics", func(c *gin.Context) {
		c.String(200, "# Prometheus metrics endpoint\n")
	})

	route.RegisterPaymentRoutes(router, paymentHandler)

	return router
}
