package handlers

import (
	"net/http"

	"github.com/auron/order-service/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.OrderService }

func NewHandler(svc *service.OrderService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetCart(c *gin.Context)    { c.JSON(http.StatusNotImplemented, gin.H{"success": false}) }
func (h *Handler) AddToCart(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"success": false}) }
func (h *Handler) UpdateCartItem(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"success": false}) }
func (h *Handler) RemoveFromCart(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"success": false}) }
func (h *Handler) CreateOrder(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"success": false}) }
func (h *Handler) ListOrders(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"success": false}) }
func (h *Handler) GetOrder(c *gin.Context)    { c.JSON(http.StatusNotImplemented, gin.H{"success": false}) }
func (h *Handler) CancelOrder(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"success": false}) }
func (h *Handler) AuthMiddleware(c *gin.Context) { c.Next() }
