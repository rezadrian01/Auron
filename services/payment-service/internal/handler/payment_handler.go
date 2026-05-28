package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"auron/payment-service/internal/domain"
	"auron/payment-service/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	service domain.PaymentService
}

func NewPaymentHandler(service domain.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) GetPaymentByID(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": domain.ErrUnauthorized.Error()})
		return
	}

	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid payment id"})
		return
	}

	payment, err := h.service.GetPaymentByID(c.Request.Context(), userID, paymentID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": payment})
}

func (h *PaymentHandler) GetPaymentByOrderID(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": domain.ErrUnauthorized.Error()})
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid order id"})
		return
	}

	payment, err := h.service.GetPaymentByOrderID(c.Request.Context(), userID, orderID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": payment})
}

func (h *PaymentHandler) HandleStripeWebhook(c *gin.Context) {
	rawBody, exists := c.Get(middleware.RawBodyKey)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "missing request body"})
		return
	}

	payload, ok := rawBody.([]byte)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
		return
	}

	signature := c.GetHeader("Stripe-Signature")

	if err := h.service.HandleStripeWebhook(c.Request.Context(), payload, signature); err != nil {
		slog.Error("stripe webhook processing failed", "error", err)
	}

	// Always return 200 — Stripe retries on any non-2xx response.
	c.JSON(http.StatusOK, gin.H{"received": true})
}

func getUserID(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetHeader("X-User-ID")
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func (h *PaymentHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrPaymentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrPaymentAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrInvalidWebhookSignature):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
	}
}
