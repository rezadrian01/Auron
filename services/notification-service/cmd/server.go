package cmd

import (
	"auron/notification-service/internal/handler"
	"auron/notification-service/internal/route"

	"github.com/gin-gonic/gin"
)

func setupRouter(healthHandler *handler.HealthHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	route.RegisterRoutes(router, healthHandler)

	return router
}
