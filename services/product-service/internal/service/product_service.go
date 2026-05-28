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
}

func NewProductService(repo domain.ProductRepository, cache domain.ProductCache, publisher domain.EventPublisher) domain.ProductService {
	return &ProductService{
		repository: repo,
		cache:      cache,
		publisher:  publisher,
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
		ImageURL:    req.ImageURL,
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
	existing.ImageURL = req.ImageURL
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

	if err := s.repository.DeleteProduct(id); err != nil {
		return err
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

// buildListCacheKey produces a deterministic cache key for a product list query.
// All pointer fields are nil-safe.
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