package handler

import (
	"errors"
	"net/http"
	"strconv"

	"auron/order-service/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CartHandler struct {
	service domain.CartService
}

func NewCartHandler(service domain.CartService) *CartHandler {
	return &CartHandler{service: service}
}

func (h *CartHandler) GetCart(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": domain.ErrUnauthorized.Error()})
		return
	}

	cart, err := h.service.GetCart(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": cart.ToResponse()})
}

func (h *CartHandler) AddItem(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": domain.ErrUnauthorized.Error()})
		return
	}

	var req domain.AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	cart, err := h.service.AddItem(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": cart.ToResponse()})
}

func (h *CartHandler) UpdateItem(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": domain.ErrUnauthorized.Error()})
		return
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid item id"})
		return
	}

	var body struct {
		Quantity int `json:"quantity" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	cart, err := h.service.UpdateItem(c.Request.Context(), userID, itemID, body.Quantity)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": cart.ToResponse()})
}

func (h *CartHandler) RemoveItem(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": domain.ErrUnauthorized.Error()})
		return
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid item id"})
		return
	}

	if err := h.service.RemoveItem(c.Request.Context(), userID, itemID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "item removed from cart"})
}

func (h *CartHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrCartNotFound), errors.Is(err, domain.ErrCartItemNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrProductNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrProductInactive):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrInvalidQuantity):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
	}
}

// getUserID reads the user UUID from the X-User-ID header set by the gateway.
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

// parseIntQuery parses a query param as int, returning defaultVal on empty/error.
func parseIntQuery(raw string, defaultVal int) int {
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}
