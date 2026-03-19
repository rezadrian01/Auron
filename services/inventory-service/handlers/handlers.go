package handlers

import (
	"net/http"

	"github.com/auron/inventory-service/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.InventoryService }

func NewHandler(svc *service.InventoryService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetInventory(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false})
}

func (h *Handler) UpdateInventory(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false})
}
