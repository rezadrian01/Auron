package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"auron/product-service/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	allowedMIME   = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}
	maxImageBytes = int64(5 << 20) // 5 MB
)

type ProductHandler struct {
	service domain.ProductService
	storage domain.StorageService
}

func NewProductHandler(service domain.ProductService, storage domain.StorageService) *ProductHandler {
	return &ProductHandler{service: service, storage: storage}
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

	responses := make([]*domain.ProductResponse, len(result.Products))
	for i := range result.Products {
		responses[i] = result.Products[i].ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responses,
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

// ── Image handlers ────────────────────────────────────────────────────────────

func (h *ProductHandler) UploadProductImage(c *gin.Context) {
	productID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid product id"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxImageBytes)
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "image field is required"})
		return
	}
	defer file.Close()

	if header.Size > maxImageBytes {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "image must be 5 MB or smaller"})
		return
	}

	// Detect MIME from first 512 bytes, then seek back.
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	ext, ok := allowedMIME[contentType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "only JPEG, PNG, and WebP images are accepted"})
		return
	}
	file.Seek(0, io.SeekStart) // multipart.File implements io.Seeker

	objectName := fmt.Sprintf("products/%s%s", uuid.New().String(), ext)
	publicURL, err := h.storage.UploadImage(c.Request.Context(), objectName, file, contentType)
	if err != nil {
		if errors.Is(err, domain.ErrStorageNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "image storage is not configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to upload image"})
		return
	}

	image, err := h.service.AddProductImage(c.Request.Context(), productID, publicURL)
	if err != nil {
		// GCS upload succeeded but DB insert failed — clean up the orphan object.
		_ = h.storage.DeleteImage(c.Request.Context(), objectName)
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": image})
}

func (h *ProductHandler) DeleteProductImage(c *gin.Context) {
	productID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid product id"})
		return
	}
	imageID, err := parseUUID(c.Param("image_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid image id"})
		return
	}

	deleted, err := h.service.DeleteProductImage(c.Request.Context(), productID, imageID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Best-effort GCS cleanup — don't fail the request if storage is slow.
	if objName, ok := h.storage.ObjectNameFromURL(deleted.URL); ok {
		if err := h.storage.DeleteImage(c.Request.Context(), objName); err != nil {
			slog.Warn("failed to delete GCS object", "url", deleted.URL, "error", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "image deleted"})
}

func (h *ProductHandler) ReorderProductImages(c *gin.Context) {
	productID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid product id"})
		return
	}

	var req domain.ReorderImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	images, err := h.service.ReorderProductImages(c.Request.Context(), productID, req.ImageIDs)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": images})
}

// ── Input DTO ─────────────────────────────────────────────────────────────────

type productBody struct {
	CategoryID  string  `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required,max=500"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required,gt=0"`
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
	case errors.Is(err, domain.ErrImageNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrCategorySlugExists), errors.Is(err, domain.ErrProductAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrInvalidSortParam),
		errors.Is(err, domain.ErrInvalidPageParam),
		errors.Is(err, domain.ErrInvalidLimitParam),
		errors.Is(err, domain.ErrPriceMustBePositive),
		errors.Is(err, domain.ErrInvalidImageOrder):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, domain.ErrStorageNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": err.Error()})
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
