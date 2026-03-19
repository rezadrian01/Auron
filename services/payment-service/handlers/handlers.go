package handlers

import (
	"net/http"

	"github.com/auron/payment-service/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.PaymentService }

func NewHandler(svc *service.PaymentService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetPayment(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false})
}

func (h *Handler) HandleStripeWebhook(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}
