package handlers

import (
	"net/http"

	"github.com/auron/product-service/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.ProductService
}

func NewHandler(svc *service.ProductService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) ListProducts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false, "error": gin.H{"code": "NOT_IMPLEMENTED"}})
}

func (h *Handler) GetProduct(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false, "error": gin.H{"code": "NOT_IMPLEMENTED"}})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false, "error": gin.H{"code": "NOT_IMPLEMENTED"}})
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false, "error": gin.H{"code": "NOT_IMPLEMENTED"}})
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false, "error": gin.H{"code": "NOT_IMPLEMENTED"}})
}

func (h *Handler) ListCategories(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false, "error": gin.H{"code": "NOT_IMPLEMENTED"}})
}

func (h *Handler) CreateCategory(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false, "error": gin.H{"code": "NOT_IMPLEMENTED"}})
}

func (h *Handler) AuthMiddleware(c *gin.Context) {
	c.Next()
}
