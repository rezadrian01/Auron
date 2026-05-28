package route

import (
	"auron/inventory-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterInventoryRoutes(router *gin.Engine, inventoryHandler *handler.InventoryHandler) {
	api := router.Group("/")
	api.GET("/inventory/:product_id", inventoryHandler.GetInventory)
	api.PUT("/inventory/:product_id", inventoryHandler.SetInventory)
}
