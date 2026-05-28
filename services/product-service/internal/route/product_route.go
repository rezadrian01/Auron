package route

import (
	"auron/product-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterProductRoutes(router *gin.Engine, h *handler.ProductHandler) {
	api := router.Group("/")

	api.GET("/products", h.GetProducts)
	api.GET("/products/:id", h.GetProductByID)
	api.POST("/products", h.CreateProduct)
	api.PUT("/products/:id", h.UpdateProduct)
	api.DELETE("/products/:id", h.DeleteProduct)

	api.GET("/categories", h.GetCategories)
	api.POST("/categories", h.CreateCategory)
}
