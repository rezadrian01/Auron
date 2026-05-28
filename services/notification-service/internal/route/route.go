package route

import (
	"auron/notification-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, healthHandler *handler.HealthHandler) {
	router.GET("/health", healthHandler.GetHealth)
}
