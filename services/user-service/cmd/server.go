package cmd

import (
	"auron/user-service/internal/handler"
	"auron/user-service/internal/route"
	"time"

	"github.com/gin-gonic/gin"
)

func setupRouter(userHandler *handler.UserHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		c.Next()
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "user-service",
			"timestamp": time.Now().UTC(),
		})
	})

	route.RegisterUserRoutes(router, userHandler)

	return router
}
