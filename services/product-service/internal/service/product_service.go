package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"auron/product-service/internal/domain"

	"github.com/google/uuid"
)

type ProductService struct {
	repository domain.ProductRepository
	cache      domain.ProductCache
	publisher  domain.EventPublisher
	storage    domain.StorageService
}

func NewProductService(
	repo domain.ProductRepository,
	cache domain.ProductCache,
	publisher domain.EventPublisher,
	storage domain.StorageService,
) domain.ProductService {
	return &ProductService{
		repository: repo,
		cache:      cache,
		publisher:  publisher,
		storage:    storage,
	}
}

// ── Read methods ──────────────────────────────────────────────────────────────

func (s *ProductService) GetProducts(ctx context.Context, filter domain.ProductFilter) (*domain.ProductListResponse, error) {
	if err := normalizeFilter(&filter); err != nil {
		return nil, err
	}

	cacheKey := buildListCacheKey(filter)
	if cached, err := s.cache.GetProductList(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	result, err := s.repository.GetProducts(filter)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetProductList(ctx, cacheKey, result); err != nil {
		slog.Warn("failed to cache product list", "error", err)
	}

	return result, nil
}

func (s *ProductService) GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	if cached, err := s.cache.GetProduct(ctx, id.String()); err == nil && cached != nil {
		return cached, nil
	}

	product, err := s.repository.GetProductByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetProduct(ctx, product); err != nil {
		slog.Warn("failed to cache product", "product_id", id, "error", err)
	}

	return product, nil
}

func (s *ProductService) GetCategories(ctx context.Context) ([]domain.Category, error) {
	return s.repository.GetCategories()
}

func (s *ProductService) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	return s.repository.GetCategoryByID(id)
}

func (s *ProductService) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	return s.repository.GetCategoryBySlug(slug)
}

// ── Write methods ─────────────────────────────────────────────────────────────

func (s *ProductService) CreateProduct(ctx context.Context, req domain.ProductRequest) (*domain.Product, error) {
	if _, err := s.repository.GetCategoryByID(req.CategoryID); err != nil {
		return nil, domain.ErrCategoryNotFound
	}

	now := time.Now()
	product := &domain.Product{
		ID:          uuid.New(),
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	created, err := s.repository.CreateProduct(product)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetProduct(ctx, created); err != nil {
		slog.Warn("failed to cache new product", "product_id", created.ID, "error", err)
	}
	if err := s.cache.InvalidateProductList(ctx); err != nil {
		slog.Warn("failed to invalidate product list cache", "error", err)
	}

	go func() {
		if err := s.publisher.Publish(context.Background(), domain.TopicProductCreated, created); err != nil {
			slog.Warn("failed to publish product.created", "product_id", created.ID, "error", err)
		}
	}()

	return created, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, id uuid.UUID, req domain.ProductRequest) (*domain.Product, error) {
	existing, err := s.repository.GetProductByID(id)
	if err != nil {
		return nil, err
	}

	if req.CategoryID != existing.CategoryID {
		if _, err := s.repository.GetCategoryByID(req.CategoryID); err != nil {
			return nil, domain.ErrCategoryNotFound
		}
	}

	existing.CategoryID = req.CategoryID
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Price = req.Price
	existing.UpdatedAt = time.Now()
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	updated, err := s.repository.UpdateProduct(existing)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetProduct(ctx, updated); err != nil {
		slog.Warn("failed to update cached product", "product_id", id, "error", err)
	}
	if err := s.cache.InvalidateProductList(ctx); err != nil {
		slog.Warn("failed to invalidate product list cache", "error", err)
	}

	go func() {
		if err := s.publisher.Publish(context.Background(), domain.TopicProductUpdated, updated); err != nil {
			slog.Warn("failed to publish product.updated", "product_id", id, "error", err)
		}
	}()

	return updated, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repository.GetProductByID(id); err != nil {
		return err
	}

	// Fetch images before DB delete for GCS cleanup.
	images, _ := s.repository.GetProductImages(id) // best-effort; errors are non-fatal

	if err := s.repository.DeleteProduct(id); err != nil {
		return err
	}

	// Delete GCS objects for all product images — best-effort after DB row is gone.
	for _, img := range images {
		if objName, ok := s.storage.ObjectNameFromURL(img.URL); ok {
			if err := s.storage.DeleteImage(ctx, objName); err != nil {
				slog.Warn("failed to delete GCS object on product delete", "url", img.URL, "error", err)
			}
		}
	}

	if err := s.cache.DeleteProduct(ctx, id.String()); err != nil {
		slog.Warn("failed to evict product from cache", "product_id", id, "error", err)
	}
	if err := s.cache.InvalidateProductList(ctx); err != nil {
		slog.Warn("failed to invalidate product list cache", "error", err)
	}

	go func() {
		payload := map[string]string{"product_id": id.String()}
		if err := s.publisher.Publish(context.Background(), domain.TopicProductDeleted, payload); err != nil {
			slog.Warn("failed to publish product.deleted", "product_id", id, "error", err)
		}
	}()

	return nil
}

func (s *ProductService) CreateCategory(ctx context.Context, req domain.CategoryRequest) (*domain.Category, error) {
	if existing, _ := s.repository.GetCategoryBySlug(req.Slug); existing != nil {
		return nil, domain.ErrCategorySlugExists
	}

	category := &domain.Category{
		ID:        uuid.New(),
		Name:      req.Name,
		Slug:      req.Slug,
		ParentID:  req.ParentID,
		CreatedAt: time.Now(),
	}

	return s.repository.CreateCategory(category)
}

// ── Image methods ─────────────────────────────────────────────────────────────

func (s *ProductService) AddProductImage(ctx context.Context, productID uuid.UUID, url string) (*domain.ProductImage, error) {
	if _, err := s.repository.GetProductByID(productID); err != nil {
		return nil, err
	}

	existing, err := s.repository.GetProductImages(productID)
	if err != nil {
		return nil, err
	}

	image := &domain.ProductImage{
		ID:        uuid.New(),
		ProductID: productID,
		URL:       url,
		Position:  len(existing), // append after current last
		CreatedAt: time.Now(),
	}

	created, err := s.repository.AddProductImage(image)
	if err != nil {
		return nil, err
	}

	_ = s.cache.DeleteProduct(ctx, productID.String())
	_ = s.cache.InvalidateProductList(ctx)

	return created, nil
}

func (s *ProductService) DeleteProductImage(ctx context.Context, productID, imageID uuid.UUID) (*domain.ProductImage, error) {
	image, err := s.repository.GetProductImage(productID, imageID)
	if err != nil {
		return nil, err
	}

	if err := s.repository.DeleteProductImage(productID, imageID); err != nil {
		return nil, err
	}

	_ = s.cache.DeleteProduct(ctx, productID.String())
	_ = s.cache.InvalidateProductList(ctx)

	return image, nil
}

func (s *ProductService) ReorderProductImages(ctx context.Context, productID uuid.UUID, imageIDs []uuid.UUID) ([]domain.ProductImage, error) {
	existing, err := s.repository.GetProductImages(productID)
	if err != nil {
		return nil, err
	}

	if len(imageIDs) != len(existing) {
		return nil, domain.ErrInvalidImageOrder
	}

	imageMap := make(map[uuid.UUID]*domain.ProductImage, len(existing))
	for i := range existing {
		imageMap[existing[i].ID] = &existing[i]
	}

	reordered := make([]domain.ProductImage, 0, len(imageIDs))
	for pos, id := range imageIDs {
		img, ok := imageMap[id]
		if !ok {
			return nil, domain.ErrImageNotFound
		}
		img.Position = pos
		reordered = append(reordered, *img)
	}

	if err := s.repository.UpdateProductImagePositions(reordered); err != nil {
		return nil, err
	}

	_ = s.cache.DeleteProduct(ctx, productID.String())
	_ = s.cache.InvalidateProductList(ctx)

	return reordered, nil
}

func (s *ProductService) GetProductImages(ctx context.Context, productID uuid.UUID) ([]domain.ProductImage, error) {
	return s.repository.GetProductImages(productID)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func normalizeFilter(f *domain.ProductFilter) error {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Sort == "" {
		f.Sort = domain.SortNewest
	}
	if !domain.ValidSorts[f.Sort] {
		return domain.ErrInvalidSortParam
	}
	return nil
}

func buildListCacheKey(f domain.ProductFilter) string {
	categoryID := ""
	if f.CategoryID != nil {
		categoryID = f.CategoryID.String()
	}
	minPrice := "0.00"
	if f.MinPrice != nil {
		minPrice = fmt.Sprintf("%.2f", *f.MinPrice)
	}
	maxPrice := "0.00"
	if f.MaxPrice != nil {
		maxPrice = fmt.Sprintf("%.2f", *f.MaxPrice)
	}
	return fmt.Sprintf("product:list:%s_%s_%s_%s_%d_%d",
		f.Q, categoryID, minPrice, maxPrice, f.Page, f.Limit,
	)
}
