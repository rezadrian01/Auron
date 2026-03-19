package handlers

import (
	"net/http"

	"github.com/auron/notification-service/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.NotificationService }

func NewHandler(svc *service.NotificationService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetNotifications(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false})
}
