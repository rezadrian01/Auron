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

	// Image management (admin only — enforced by gateway)
	api.POST("/products/:id/images", h.UploadProductImage)
	api.DELETE("/products/:id/images/:image_id", h.DeleteProductImage)
	api.PUT("/products/:id/images/reorder", h.ReorderProductImages)

	api.GET("/categories", h.GetCategories)
	api.POST("/categories", h.CreateCategory)
}
