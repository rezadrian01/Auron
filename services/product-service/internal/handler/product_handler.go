package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"auron/product-service/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductHandler struct {
	service domain.ProductService
}

func NewProductHandler(service domain.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

// ── Product handlers ──────────────────────────────────────────────────────────

func (h *ProductHandler) GetProducts(c *gin.Context) {
	filter, err := parseFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	result, err := h.service.GetProducts(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Products,
		"meta": gin.H{
			"page":  result.Page,
			"limit": result.Limit,
			"total": result.Total,
		},
	})
}

func (h *ProductHandler) GetProductByID(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid product id"})
		return
	}

	product, err := h.service.GetProductByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": product.ToResponse()})
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var body productBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	req, err := body.toDomain()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	product, err := h.service.CreateProduct(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": product.ToResponse()})
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid product id"})
		return
	}

	var body productBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	req, err := body.toDomain()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	product, err := h.service.UpdateProduct(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": product.ToResponse()})
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid product id"})
		return
	}

	if err := h.service.DeleteProduct(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "product deleted"})
}

// ── Category handlers ─────────────────────────────────────────────────────────

func (h *ProductHandler) GetCategories(c *gin.Context) {
	categories, err := h.service.GetCategories(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": categories})
}

func (h *ProductHandler) CreateCategory(c *gin.Context) {
	var req domain.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	category, err := h.service.CreateCategory(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": category.ToResponse()})
}

// ── Input DTO ─────────────────────────────────────────────────────────────────

// productBody is the HTTP request body for product create/update.
type productBody struct {
	CategoryID  string  `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required,max=500"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	ImageURL    string  `json:"image_url" binding:"omitempty,url"`
	IsActive    *bool   `json:"is_active"`
}

func (b *productBody) toDomain() (domain.ProductRequest, error) {
	categoryID, err := uuid.Parse(b.CategoryID)
	if err != nil {
		return domain.ProductRequest{}, fmt.Errorf("invalid category_id: %w", err)
	}

	return domain.ProductRequest{
		CategoryID:  categoryID,
		Name:        b.Name,
		Description: b.Description,
		Price:       b.Price,
		ImageURL:    b.ImageURL,
		IsActive:    b.IsActive,
	}, nil
}

// ── Error mapping ─────────────────────────────────────────────────────────────

func (h *ProductHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrProductNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrCategoryNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrCategorySlugExists), errors.Is(err, domain.ErrProductAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrInvalidSortParam),
		errors.Is(err, domain.ErrInvalidPageParam),
		errors.Is(err, domain.ErrInvalidLimitParam),
		errors.Is(err, domain.ErrPriceMustBePositive):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseUUID(raw string) (uuid.UUID, error) {
	return uuid.Parse(raw)
}

func parseFilter(c *gin.Context) (domain.ProductFilter, error) {
	filter := domain.ProductFilter{
		Q:     c.Query("q"),
		Sort:  c.Query("sort"),
		Page:  parseIntQuery(c.Query("page"), 1),
		Limit: parseIntQuery(c.Query("limit"), 20),
	}

	if raw := c.Query("category_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return filter, fmt.Errorf("invalid category_id: must be a valid UUID")
		}
		filter.CategoryID = &id
	}

	if raw := c.Query("min_price"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return filter, fmt.Errorf("invalid min_price: must be a number")
		}
		filter.MinPrice = &v
	}

	if raw := c.Query("max_price"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return filter, fmt.Errorf("invalid max_price: must be a number")
		}
		filter.MaxPrice = &v
	}

	return filter, nil
}

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
