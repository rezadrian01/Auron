package route

import (
	"auron/order-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterOrderRoutes(router *gin.Engine, cartHandler *handler.CartHandler, orderHandler *handler.OrderHandler) {
	api := router.Group("/")

	api.GET("/cart", cartHandler.GetCart)
	api.POST("/cart/items", cartHandler.AddItem)
	api.PUT("/cart/items/:id", cartHandler.UpdateItem)
	api.DELETE("/cart/items/:id", cartHandler.RemoveItem)

	api.GET("/orders", orderHandler.GetOrders)
	api.POST("/orders", orderHandler.CreateOrder)
	api.GET("/orders/:id", orderHandler.GetOrderByID)
	api.PUT("/orders/:id/cancel", orderHandler.CancelOrder)
}
