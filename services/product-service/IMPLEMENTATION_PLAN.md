# Product Service Implementation Plan

> **Service:** Product Catalog Management  
> **Port:** `8082`  
> **Database:** `products_db` (PostgreSQL :5433)  
> **Stack:** Go 1.21 · Gin · GORM · Redis · Kafka  
> **Architecture Pattern:** Layered (Inner → Outer) — Domain → Repository → Cache → Service → Handler → Route → Bootstrap

---

## Table of Contents

1. [Overview & Scope](#1-overview--scope)
2. [Existing State Analysis](#2-existing-state-analysis)
3. [Architecture Flow](#3-architecture-flow)
4. [Layer 1: Domain (Core)](#4-layer-1-domain-core)
5. [Layer 2: Repository](#5-layer-2-repository)
6. [Layer 3: Cache](#6-layer-3-cache)
7. [Layer 4: Service](#7-layer-4-service)
8. [Layer 5: Handler](#8-layer-5-handler)
9. [Layer 6: Route](#9-layer-6-route)
10. [Layer 7: Bootstrap (Outer)](#10-layer-7-bootstrap-outer)
11. [Database & Migrations](#11-database--migrations)
12. [Kafka Integration](#12-kafka-integration)
13. [Configuration & Environment](#13-configuration--environment)
14. [Implementation Checklist](#14-implementation-checklist)
15. [File Structure](#15-file-structure)

---

## 1. Overview & Scope

### Endpoints (per Technical Plan §4.3)

| Method | Path | Description | Auth |
|---|---|---|---|
| `GET` | `/products` | List products (paginated, filtered, searched) | No |
| `GET` | `/products/:id` | Get product detail | No |
| `POST` | `/products` | Create product | Admin |
| `PUT` | `/products/:id` | Update product | Admin |
| `DELETE` | `/products/:id` | Delete product | Admin |
| `GET` | `/categories` | List categories | No |
| `POST` | `/categories` | Create category | Admin |

### Required Features

- **Full-text search** via PostgreSQL `tsvector` + `plainto_tsquery`
- **Filtering**: `category_id`, `min_price`, `max_price`
- **Sorting**: `price_asc`, `price_desc`, `newest`, `name_asc`, `name_desc`
- **Pagination**: `page`, `limit` (defaults: page=1, limit=20, max=100)
- **Redis caching**: product detail + list with 5-min TTL
- **Cache invalidation**: on any product mutation (create/update/delete)
- **Kafka events**: publish product lifecycle events (future: inventory sync)

### Key Constraints (from Technical Plan)

- Products table has `search_vector` tsvector column with GIN index
- Categories support hierarchical structure (`parent_id`)
- Inventory table shares the same `products_db` (separate service reads/writes it)
- All write operations require admin role
- Cache keys: `product:{id}` for detail, `products:list:{hash}` for listings

---

## 2. Existing State Analysis

### ✅ Already Implemented

| File | Status | Notes |
|---|---|---|
| `internal/domain/product.go` | ✅ Complete | Product, Category, Inventory models + all DTOs (request/response) |
| `internal/domain/errors.go` | ⚠️ Partial | Only `ErrProductNotFound`, `ErrInvalidProductID` — needs expansion |
| `internal/cache/` | ✅ Dir exists | Empty — implementation needed |
| `internal/domain/` | ✅ Dir exists | Has models + DTOs |
| `internal/repository/` | ✅ Dir exists | Empty — implementation needed |
| `internal/service/` | ✅ Dir exists | Empty — implementation needed |
| `internal/handler/` | ✅ Dir exists | Empty — implementation needed |
| `internal/middleware/` | ✅ Dir exists | Empty — may need admin auth middleware |
| `internal/events/` | ✅ Dir exists | Empty — for Kafka event publishing |
| `go.mod` / `go.sum` | ✅ Exists | Module defined, dependencies ready |

### ❌ Missing (to be created)

- Domain interfaces: `repository.go`, `service.go`, `cache.go`
- Repository implementation: `product_repository.go`
- Cache implementation: `product_cache.go`
- Service implementation: `product_service.go`
- Handler implementation: `product_handler.go`
- Route registration: `product_route.go`
- Bootstrap: `main.go`, `cmd/config.go`, `cmd/dotenv.go`, `cmd/infrastructure.go`, `cmd/server.go`, `cmd/run.go`
- Dockerfile
- `.env` / `.env.example`
- Database migrations directory

---

## 3. Architecture Flow

```
┌─────────────────────────────────────────────────────────┐
│                    BOOTSTRAP (Outer)                     │
│  main.go → cmd/run.go → cmd/infrastructure.go           │
│  ↓ Config loading, DB setup, Redis setup, DI wiring      │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                     ROUTE LAYER                          │
│  internal/route/product_route.go                        │
│  ↓ Route registration, middleware application            │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                    HANDLER LAYER                         │
│  internal/handler/product_handler.go                    │
│  ↓ HTTP binding, validation, error→status mapping        │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                    SERVICE LAYER                         │
│  internal/service/product_service.go                    │
│  ↓ Business logic, cache orchestration, validation       │
└──────────┬──────────────────────┬───────────────────────┘
           │                      │
┌──────────▼──────┐    ┌──────────▼──────────┐
│  REPOSITORY     │    │      CACHE          │
│  (PostgreSQL)   │    │    (Redis)          │
│  product_repo.go│    │  product_cache.go   │
└─────────────────┘    └─────────────────────┘
           │                      │
┌──────────▼──────────────────────▼───────────────────────┐
│                    DOMAIN (Core)                         │
│  internal/domain/{product,errors,repository,             │
│                   service,cache}.go                      │
│  Entities, interfaces, error types, DTOs                 │
└──────────────────────────────────────────────────────────┘
```

**Dependency Direction (Inner → Outer):**
```
Domain ← Repository
Domain ← Cache
Domain + Repository + Cache ← Service
Domain + Service ← Handler
Service + Handler ← Route
All layers ← Bootstrap
```

---

## 4. Layer 1: Domain (Core)

**Location:** `internal/domain/`  
**Purpose:** Define the business entities, interfaces, and error types. No implementation logic — only contracts.

### 4.1 Update `errors.go`

Expand existing errors to cover all product service scenarios:

```go
package domain

import "errors"

var (
    // Product errors
    ErrProductNotFound     = errors.New("product not found")
    ErrInvalidProductID    = errors.New("invalid product ID")
    ErrProductAlreadyExists = errors.New("product with this name already exists")

    // Category errors
    ErrCategoryNotFound   = errors.New("category not found")
    ErrCategorySlugExists = errors.New("category with this slug already exists")
    ErrInvalidCategoryID  = errors.New("invalid category ID")

    // Validation errors
    ErrInvalidSortParam    = errors.New("invalid sort parameter. allowed: price_asc, price_desc, newest, name_asc, name_desc")
    ErrInvalidPageParam    = errors.New("page must be >= 1")
    ErrInvalidLimitParam   = errors.New("limit must be between 1 and 100")
    ErrPriceMustBePositive = errors.New("price must be positive")

    // Generic
    ErrUnauthorized = errors.New("unauthorized")
    ErrForbidden    = errors.New("forbidden")
)
```

### 4.2 Create `repository.go`

Define the repository interface that the service layer will depend on:

```go
package domain

import "github.com/google/uuid"

// ProductFilter holds query parameters for listing products
type ProductFilter struct {
    Q          string     // Full-text search query
    CategoryID *uuid.UUID // Filter by category
    MinPrice   *float64   // Minimum price filter
    MaxPrice   *float64   // Maximum price filter
    Sort       string     // Sort order: price_asc, price_desc, newest, name_asc, name_desc
    Page       int        // Page number (1-based)
    Limit      int        // Items per page
}

// ProductListResponse holds paginated product results
type ProductListResponse struct {
    Products []Product
    Total    int64
    Page     int
    Limit    int
}

// ProductRepository defines the data access contract for products and categories
type ProductRepository interface {
    // Product CRUD
    CreateProduct(product *Product) (*Product, error)
    GetProductByID(id uuid.UUID) (*Product, error)
    ListProducts(filter ProductFilter) (*ProductListResponse, error)
    UpdateProduct(product *Product) (*Product, error)
    DeleteProduct(id uuid.UUID) error

    // Category operations
    CreateCategory(category *Category) (*Category, error)
    ListCategories() ([]Category, error)
    GetCategoryByID(id uuid.UUID) (*Category, error)
    GetCategoryBySlug(slug string) (*Category, error)
}
```

**Key Design Decisions:**
- `ProductFilter` uses pointers for optional filters to distinguish "not provided" from "zero value"
- `ListProducts` returns a struct (not slice + count) for cleaner API
- Default pagination: `Page=1`, `Limit=20`, `Sort="newest"` (handled by service layer)
- Categories are simple — no pagination needed (expected < 1000 categories)

### 4.3 Create `cache.go`

Define the cache interface:

```go
package domain

import "context"

// ProductCache defines the caching contract for product data
type ProductCache interface {
    // Product detail cache
    GetProduct(ctx context.Context, id string) (*Product, error)
    SetProduct(ctx context.Context, product *Product) error
    DeleteProduct(ctx context.Context, id string) error

    // Product list cache (paginated results)
    GetProductList(ctx context.Context, cacheKey string) (*ProductListResponse, error)
    SetProductList(ctx context.Context, cacheKey string, response *ProductListResponse) error
    InvalidateProductList(ctx context.Context) error

    // Utility
    ClearAll(ctx context.Context) error
}
```

**Key Design Decisions:**
- Separate methods for product detail vs list caching (different TTLs and invalidation patterns)
- `InvalidateProductList` deletes all `products:list:*` keys (wildcard invalidation)
- Context-aware for timeout/cancellation support

### 4.4 Update `product.go` (if needed)

Current models are complete. Verify alignment with Technical Plan §5 database schema:

| Technical Plan Schema | Current Model | Alignment |
|---|---|---|
| `products.id UUID` | `Product.ID uuid.UUID` | ✅ |
| `products.category_id UUID` | `Product.CategoryID uuid.UUID` | ✅ |
| `products.name VARCHAR(500)` | `Product.Name string` | ⚠️ Update to `varchar(500)` |
| `products.description TEXT` | `Product.Description string` | ✅ |
| `products.price DECIMAL(12,2)` | `Product.Price float64` | ⚠️ Consider `decimal.Decimal` for precision |
| `products.image_url TEXT` | `Product.ImageURL string` | ✅ |
| `products.search_vector TSVECTOR` | `Product.SearchVector string` | ⚠️ GORM tsvector support needs verification |
| `products.is_active BOOLEAN` | `Product.IsActive bool` | ✅ |
| `categories.parent_id UUID` | `Category.ParentID` | ❌ Missing — add to Category model |

**Required update to `Category` model:**

```go
type Category struct {
    ID        uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    Name      string     `json:"name" gorm:"type:varchar(255);not null"`
    Slug      string     `json:"slug" gorm:"type:varchar(255);not null;uniqueIndex"`
    ParentID  *uuid.UUID `json:"parent_id,omitempty" gorm:"type:uuid"`  // ADD THIS
    CreatedAt time.Time  `json:"created_at" gorm:"not null;default:now()"`
}
```

### 4.5 Create `events.go` (Optional — for Kafka publishing)

Define event types the service will publish:

```go
package domain

// EventPublisher defines the contract for publishing domain events
type EventPublisher interface {
    Publish(ctx context.Context, topic string, payload any) error
}

// Product event topic constants
const (
    TopicProductCreated = "product.created"
    TopicProductUpdated = "product.updated"
    TopicProductDeleted = "product.deleted"
)
```

---

## 5. Layer 2: Repository

**Location:** `internal/repository/product_repository.go`  
**Purpose:** Implement `domain.ProductRepository` using GORM. All database logic lives here.

### 5.1 Repository Structure

```go
package repository

import (
    "auron/product-service/internal/domain"
    "gorm.io/gorm"
)

type ProductRepository struct {
    db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
    return &ProductRepository{db: db}
}
```

### 5.2 Implementation Tasks

| Method | Implementation Details |
|---|---|
| `CreateProduct` | `db.Create(product)`, preload category after create |
| `GetProductByID` | `db.Where("id = ?", id).Preload("Category").First(&product)`, return `ErrProductNotFound` on `gorm.ErrRecordNotFound` |
| `ListProducts` | See detailed implementation below |
| `UpdateProduct` | `db.Save(product)`, update `updated_at` via GORM |
| `DeleteProduct` | Soft delete or hard delete (`db.Delete(&Product{}, id)`), also delete inventory row |
| `CreateCategory` | `db.Create(category)`, check slug uniqueness |
| `ListCategories` | `db.Order("name ASC").Find(&categories)` |
| `GetCategoryByID` | `db.First(&category, id)` |
| `GetCategoryBySlug` | `db.Where("slug = ?", slug).First(&category)` |

### 5.3 `ListProducts` Detailed Implementation

```go
func (r *ProductRepository) ListProducts(filter domain.ProductFilter) (*domain.ProductListResponse, error) {
    query := r.db.Model(&domain.Product{}).Where("is_active = ?", true)

    // Filter by category
    if filter.CategoryID != nil {
        query = query.Where("category_id = ?", *filter.CategoryID)
    }

    // Price range filters
    if filter.MinPrice != nil {
        query = query.Where("price >= ?", *filter.MinPrice)
    }
    if filter.MaxPrice != nil {
        query = query.Where("price <= ?", *filter.MaxPrice)
    }

    // Full-text search
    if filter.Q != "" {
        query = query.Where("search_vector @@ plainto_tsquery('english', ?)", filter.Q)
    }

    // Count total (before pagination)
    var total int64
    if err := query.Count(&total).Error; err != nil {
        return nil, err
    }

    // Apply sorting
    query = r.applySort(query, filter.Sort)

    // Apply pagination
    offset := (filter.Page - 1) * filter.Limit
    query = query.Offset(offset).Limit(filter.Limit)

    // Execute query with category preload
    var products []domain.Product
    if err := query.Preload("Category").Find(&products).Error; err != nil {
        return nil, err
    }

    return &domain.ProductListResponse{
        Products: products,
        Total:    total,
        Page:     filter.Page,
        Limit:    filter.Limit,
    }, nil
}

func (r *ProductRepository) applySort(query *gorm.DB, sort string) *gorm.DB {
    switch sort {
    case "price_asc":
        return query.Order("price ASC")
    case "price_desc":
        return query.Order("price DESC")
    case "newest":
        return query.Order("created_at DESC")
    case "name_asc":
        return query.Order("name ASC")
    case "name_desc":
        return query.Order("name DESC")
    default:
        return query.Order("created_at DESC") // Default: newest first
    }
}
```

### 5.4 Key Considerations

- **tsvector queries**: Use raw SQL via `db.Where()` since GORM doesn't natively support full-text search
- **Category preload**: Always preload `Category` relation to avoid N+1 queries
- **Pagination safety**: Validate `offset` doesn't go negative (service layer handles this)
- **Transaction support**: `DeleteProduct` may need to delete related inventory row — use `db.Transaction()`

---

## 6. Layer 3: Cache

**Location:** `internal/cache/product_cache.go`  
**Purpose:** Implement `domain.ProductCache` using Redis. Follow the caching strategy from Technical Plan §7.

### 6.1 Cache Structure

```go
package cache

import (
    "auron/product-service/internal/domain"
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

const (
    productDetailPrefix = "product:"
    productListPrefix   = "products:list:"
    cacheTTL            = 5 * time.Minute
)

type ProductCache struct {
    redis *redis.Client
}

func NewProductCache(redisClient *redis.Client) domain.ProductCache {
    return &ProductCache{redis: redisClient}
}
```

### 6.2 Implementation Tasks

| Method | Key Pattern | TTL | Notes |
|---|---|---|---|
| `GetProduct` | `product:{id}` | 5 min | JSON serialize/deserialize |
| `SetProduct` | `product:{id}` | 5 min | Marshal product to JSON |
| `DeleteProduct` | `product:{id}` | — | `DEL` command |
| `GetProductList` | `products:list:{hash}` | 5 min | Hash of filter params |
| `SetProductList` | `products:list:{hash}` | 5 min | Marshal response to JSON |
| `InvalidateProductList` | `products:list:*` | — | SCAN + DEL (pattern match) |

### 6.3 Cache Key Generation

```go
// GenerateCacheKey creates a deterministic cache key from filter params
func GenerateCacheKey(filter domain.ProductFilter) string {
    // Create a hash from filter params for list caching
    hash := fmt.Sprintf("%s_%s_%s_%s_%d_%d",
        filter.Q,
        filter.CategoryID.String(),
        fmt.Sprintf("%.2f", filter.MinPrice),
        fmt.Sprintf("%.2f", filter.MaxPrice),
        filter.Page,
        filter.Limit,
    )
    return productListPrefix + hash
}
```

### 6.4 Pattern Invalidation (SCAN + DEL)

```go
func (c *ProductCache) InvalidateProductList(ctx context.Context) error {
    var cursor uint64
    pattern := productListPrefix + "*"

    for {
        keys, nextCursor, err := c.redis.Scan(ctx, cursor, pattern, 100).Result()
        if err != nil {
            return err
        }

        if len(keys) > 0 {
            if err := c.redis.Del(ctx, keys...).Err(); err != nil {
                return err
            }
        }

        cursor = nextCursor
        if cursor == 0 {
            break
        }
    }

    return nil
}
```

### 6.5 Key Considerations

- Use `SCAN` instead of `KEYS` for production safety (non-blocking)
- JSON serialization for complex structs (`ProductListResponse`)
- TTL is consistent: 5 minutes for all product cache entries
- Cache misses return `redis.Nil` — service layer translates to `domain.ErrProductNotFound`

---

## 7. Layer 4: Service

**Location:** `internal/service/product_service.go`  
**Purpose:** Business logic layer. Orchestrates repository + cache + event publishing.

### 7.1 Service Structure

```go
package service

import (
    "auron/product-service/internal/domain"
    "context"
)

type ProductService struct {
    repo      domain.ProductRepository
    cache     domain.ProductCache
    publisher domain.EventPublisher
}

func NewProductService(
    repo domain.ProductRepository,
    cache domain.ProductCache,
    publisher domain.EventPublisher,
) domain.ProductService {
    return &ProductService{
        repo:      repo,
        cache:     cache,
        publisher: publisher,
    }
}
```

### 7.2 Service Interface (`domain/service.go`)

```go
package domain

import "github.com/google/uuid"

type ProductService interface {
    // Product operations
    CreateProduct(req *ProductRequest) (*Product, error)
    GetProduct(id uuid.UUID) (*Product, error)
    ListProducts(filter ProductFilter) (*ProductListResponse, error)
    UpdateProduct(id uuid.UUID, req *ProductRequest) (*Product, error)
    DeleteProduct(id uuid.UUID) error

    // Category operations
    CreateCategory(req *CategoryRequest) (*Category, error)
    ListCategories() ([]Category, error)
}
```

### 7.3 Implementation Tasks

| Method | Business Logic |
|---|---|
| `CreateProduct` | Validate request → check category exists → create product → create inventory row (qty=0) → cache product → invalidate list cache → publish `product.created` event |
| `GetProduct` | **Cache-first**: check cache → if miss, query repo → cache result → return |
| `ListProducts` | **Cache-first**: generate cache key → check cache → if miss, query repo → cache result → return |
| `UpdateProduct` | Validate request → check product exists → check category exists → update → delete product cache → invalidate list cache → publish `product.updated` |
| `DeleteProduct` | Check product exists → delete from repo → delete cache → invalidate list cache → publish `product.deleted` |
| `CreateCategory` | Validate request → check slug uniqueness → create category |
| `ListCategories` | Direct repo call (no caching needed for small dataset) |

### 7.4 Default Pagination & Validation

```go
func normalizeFilter(filter *domain.ProductFilter) error {
    // Default page
    if filter.Page < 1 {
        filter.Page = 1
    }

    // Default limit
    if filter.Limit < 1 {
        filter.Limit = 20
    }
    if filter.Limit > 100 {
        filter.Limit = 100
    }

    // Default sort
    if filter.Sort == "" {
        filter.Sort = "newest"
    }

    // Validate sort
    validSorts := map[string]bool{
        "price_asc": true, "price_desc": true,
        "newest": true, "name_asc": true, "name_desc": true,
    }
    if !validSorts[filter.Sort] {
        return domain.ErrInvalidSortParam
    }

    return nil
}
```

### 7.5 Cache-First Read Pattern

```go
func (s *ProductService) GetProduct(id uuid.UUID) (*domain.Product, error) {
    ctx := context.Background()

    // Try cache first
    if product, err := s.cache.GetProduct(ctx, id.String()); err == nil {
        return product, nil
    }

    // Cache miss → query repository
    product, err := s.repo.GetProductByID(id)
    if err != nil {
        return nil, err
    }

    // Populate cache (non-blocking — log errors, don't fail the request)
    if err := s.cache.SetProduct(ctx, product); err != nil {
        slog.Warn("failed to cache product", "product_id", id, "error", err)
    }

    return product, nil
}
```

### 7.6 Write Path with Cache Invalidation

```go
func (s *ProductService) UpdateProduct(id uuid.UUID, req *domain.ProductRequest) (*domain.Product, error) {
    ctx := context.Background()

    // Verify product exists
    existing, err := s.repo.GetProductByID(id)
    if err != nil {
        return nil, err
    }

    // Verify category exists (if changed)
    if req.CategoryID != existing.CategoryID {
        if _, err := s.repo.GetCategoryByID(req.CategoryID); err != nil {
            return nil, domain.ErrCategoryNotFound
        }
    }

    // Update fields
    existing.Name = req.Name
    existing.Description = req.Description
    existing.Price = req.Price
    existing.ImageURL = req.ImageURL
    existing.CategoryID = req.CategoryID
    if req.IsActive != nil {
        existing.IsActive = *req.IsActive
    }

    updated, err := s.repo.UpdateProduct(existing)
    if err != nil {
        return nil, err
    }

    // Invalidate caches
    _ = s.cache.DeleteProduct(ctx, id.String())
    _ = s.cache.InvalidateProductList(ctx)

    // Publish event (non-blocking)
    go func() {
        _ = s.publisher.Publish(context.Background(), domain.TopicProductUpdated, updated)
    }()

    return updated, nil
}
```

### 7.7 Key Considerations

- **Cache failures are non-fatal**: If cache set/delete fails, log warning but continue
- **Event publishing is async**: Use goroutine to avoid blocking HTTP response
- **Transaction safety**: Product creation needs product + inventory rows in same transaction (repo handles this)
- **Category validation**: Always verify category exists before product create/update

---

## 8. Layer 5: Handler

**Location:** `internal/handler/product_handler.go`  
**Purpose:** HTTP request/response handling. Thin layer — delegates to service, maps errors to HTTP status codes.

### 8.1 Handler Structure

```go
package handler

import (
    "auron/product-service/internal/domain"
    "net/http"

    "github.com/gin-gonic/gin"
)

type ProductHandler struct {
    service domain.ProductService
}

func NewProductHandler(service domain.ProductService) *ProductHandler {
    return &ProductHandler{service: service}
}
```

### 8.2 Handler Methods

| HTTP Handler | Service Method | Success Status | Notes |
|---|---|---|---|
| `ListProducts` | `service.ListProducts(filter)` | 200 | Parse query params → filter |
| `GetProduct` | `service.GetProduct(id)` | 200 | Parse UUID from path param |
| `CreateProduct` | `service.CreateProduct(req)` | 201 | Bind JSON body → validate |
| `UpdateProduct` | `service.UpdateProduct(id, req)` | 200 | Bind JSON body → validate |
| `DeleteProduct` | `service.DeleteProduct(id)` | 200 | Return success message |
| `ListCategories` | `service.ListCategories()` | 200 | No params needed |
| `CreateCategory` | `service.CreateCategory(req)` | 201 | Bind JSON body → validate |

### 8.3 `ListProducts` Handler Implementation

```go
func (h *ProductHandler) ListProducts(c *gin.Context) {
    filter, err := h.parseFilter(c)
    if err != nil {
        c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
        return
    }

    result, err := h.service.ListProducts(filter)
    if err != nil {
        h.handleServiceError(c, err)
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

func (h *ProductHandler) parseFilter(c *gin.Context) (domain.ProductFilter, error) {
    filter := domain.ProductFilter{
        Q:     c.Query("q"),
        Sort:  c.Query("sort"),
        Page:  parseIntOrDefault(c.Query("page"), 1),
        Limit: parseIntOrDefault(c.Query("limit"), 20),
    }

    // Parse optional UUID
    if categoryID := c.Query("category_id"); categoryID != "" {
        id, err := uuid.Parse(categoryID)
        if err != nil {
            return filter, fmt.Errorf("invalid category_id: %w", err)
        }
        filter.CategoryID = &id
    }

    // Parse optional floats
    if minPrice := c.Query("min_price"); minPrice != "" {
        price, err := strconv.ParseFloat(minPrice, 64)
        if err != nil {
            return filter, fmt.Errorf("invalid min_price: %w", err)
        }
        filter.MinPrice = &price
    }

    if maxPrice := c.Query("max_price"); maxPrice != "" {
        price, err := strconv.ParseFloat(maxPrice, 64)
        if err != nil {
            return filter, fmt.Errorf("invalid max_price: %w", err)
        }
        filter.MaxPrice = &price
    }

    return filter, nil
}
```

### 8.4 Error Mapping

```go
func (h *ProductHandler) handleServiceError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, domain.ErrProductNotFound):
        c.JSON(http.StatusNotFound, domain.ErrorResponse{
            "success": false,
            "error":   err.Error(),
        })
    case errors.Is(err, domain.ErrCategoryNotFound):
        c.JSON(http.StatusBadRequest, domain.ErrorResponse{
            "success": false,
            "error":   err.Error(),
        })
    case errors.Is(err, domain.ErrCategorySlugExists):
        c.JSON(http.StatusConflict, domain.ErrorResponse{
            "success": false,
            "error":   err.Error(),
        })
    case errors.Is(err, domain.ErrInvalidSortParam),
         errors.Is(err, domain.ErrInvalidPageParam),
         errors.Is(err, domain.ErrInvalidLimitParam):
        c.JSON(http.StatusBadRequest, domain.ErrorResponse{
            "success": false,
            "error":   err.Error(),
        })
    default:
        c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
            "success": false,
            "error":   "internal server error",
        })
    }
}
```

### 8.5 Response Format (per Technical Plan §10)

All responses follow the standard envelope:

```json
// Success
{
  "success": true,
  "data": { ... },
  "meta": { "page": 1, "limit": 20, "total": 100 }
}

// Error
{
  "success": false,
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "product not found"
  }
}
```

### 8.6 Key Considerations

- **Keep handlers thin**: No business logic — only HTTP binding and error mapping
- **Validate at handler level**: Use Gin's `binding` tags for required fields
- **Parse UUIDs safely**: Return 400 for invalid UUIDs (don't let it reach service layer)
- **Consistent error format**: Match the API Gateway error response contract

---

## 9. Layer 6: Route

**Location:** `internal/route/product_route.go`  
**Purpose:** Register routes with Gin engine. Apply middleware for auth/role checks.

### 9.1 Route Registration

```go
package route

import (
    "auron/product-service/internal/handler"

    "github.com/gin-gonic/gin"
)

func RegisterProductRoutes(router *gin.Engine, h *handler.ProductHandler) {
    api := router.Group("/")

    // ── Public routes (no auth) ──
    api.GET("/products", h.ListProducts)
    api.GET("/products/:id", h.GetProduct)
    api.GET("/categories", h.ListCategories)

    // ── Admin routes (auth + role check) ──
    // Note: Admin middleware is applied by API Gateway.
    // The service receives X-User-Role header from gateway.
    admin := api.Group("/")
    // admin.Use(middleware.RequireAdmin())  // Applied at gateway level
    {
        admin.POST("/products", h.CreateProduct)
        admin.PUT("/products/:id", h.UpdateProduct)
        admin.DELETE("/products/:id", h.DeleteProduct)
        admin.POST("/categories", h.CreateCategory)
    }
}
```

### 9.2 Route Mapping to API Gateway

Per Technical Plan §4.1 routing table, the API Gateway proxies:

| Gateway Route | Downstream Route | Auth |
|---|---|---|
| `GET /api/products` | `GET /products` | No |
| `GET /api/products/:id` | `GET /products/:id` | No |
| `POST /api/products` | `POST /products` | Yes (admin) |
| `PUT /api/products/:id` | `PUT /products/:id` | Yes (admin) |
| `DELETE /api/products/:id` | `DELETE /products/:id` | Yes (admin) |
| `GET /api/categories` | `GET /categories` | No |
| `POST /api/categories` | `POST /categories` | Yes (admin) |

**Important:** The API Gateway handles JWT validation and admin role checking. The product service can trust the `X-User-Role` header forwarded by the gateway.

### 9.3 Middleware Needs

| Middleware | Applied By | Purpose |
|---|---|---|
| JWT validation | API Gateway | Verify access token |
| Admin role check | API Gateway | Check `role=admin` in JWT claims |
| Request ID | API Gateway | Inject `X-Request-ID` header |
| CORS | API Gateway | Handle cross-origin requests |
| Rate limiting | API Gateway | 100 req/min per IP |

**Product service does NOT need its own auth middleware** — it relies on the API Gateway for all cross-cutting concerns.

---

## 10. Layer 7: Bootstrap (Outer)

**Location:** `main.go` + `cmd/` directory  
**Purpose:** Wire all layers together. Load config, setup infrastructure, start server.

### 10.1 File Structure

```
cmd/
├── config.go          # Configuration struct + loading
├── dotenv.go          # .env file loading
├── infrastructure.go  # Database + Redis setup
├── kafka.go           # Kafka producer setup (optional)
├── run.go             # Main orchestration (DI wiring)
└── server.go          # Gin router setup + graceful shutdown
main.go                # Entry point (calls cmd.Run())
```

### 10.2 `config.go`

```go
package cmd

import "os"

type Config struct {
    Port         string
    DatabaseURL  string
    RedisURL     string
    KafkaBrokers string
    Environment  string // dev, staging, prod
}

func loadConfig() *Config {
    return &Config{
        Port:         getEnv("PORT", "8082"),
        DatabaseURL:  getEnv("DATABASE_URL", "postgres://auron:auron_pass@products-db:5433/products_db?sslmode=disable"),
        RedisURL:     getEnv("REDIS_URL", "redis://redis:6379/0"),
        KafkaBrokers: getEnv("KAFKA_BROKERS", "kafka:29092"),
        Environment:  getEnv("ENVIRONMENT", "dev"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### 10.3 `infrastructure.go`

```go
package cmd

import (
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "log"

    "github.com/redis/go-redis/v9"
)

func setupDatabase(databaseURL string) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
    if err != nil {
        return nil, err
    }

    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }

    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)

    return db, nil
}

func setupRedis(redisURL string) (*redis.Client, error) {
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, err
    }

    client := redis.NewClient(opt)
    return client, nil
}
```

### 10.4 `run.go` (DI Wiring)

```go
package cmd

import (
    "auron/product-service/internal/cache"
    "auron/product-service/internal/handler"
    "auron/product-service/internal/repository"
    "auron/product-service/internal/service"
    "fmt"
    "log"
)

func Run() {
    cfg := loadConfig()

    // Setup infrastructure
    db, err := setupDatabase(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    redisClient, err := setupRedis(cfg.RedisURL)
    if err != nil {
        log.Fatalf("Failed to connect to Redis: %v", err)
    }

    // Run migrations (AutoMigrate for development)
    if err := runMigrations(db); err != nil {
        log.Fatalf("Failed to run migrations: %v", err)
    }

    // Setup Kafka producer (optional — can be nil for initial implementation)
    publisher := setupKafkaPublisher(cfg.KafkaBrokers)

    // Wire dependencies (Inner → Outer)
    repo := repository.NewProductRepository(db)
    cache := cache.NewProductCache(redisClient)
    svc := service.NewProductService(repo, cache, publisher)
    h := handler.NewProductHandler(svc)

    // Setup router and start server
    router := setupRouter(h)

    addr := fmt.Sprintf(":%s", cfg.Port)
    log.Printf("Starting Product Service on %s", addr)
    if err := router.Run(addr); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
```

### 10.5 `server.go`

```go
package cmd

import (
    "auron/product-service/internal/handler"
    "auron/product-service/internal/route"
    "time"

    "github.com/gin-gonic/gin"
)

func setupRouter(h *handler.ProductHandler) *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    router := gin.New()

    // Global middleware
    router.Use(gin.Logger())
    router.Use(gin.Recovery())

    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":    "healthy",
            "service":   "product-service",
            "timestamp": time.Now().UTC(),
        })
    })

    // Prometheus metrics (stub — implement later)
    router.GET("/metrics", func(c *gin.Context) {
        c.String(200, "# Prometheus metrics endpoint\n")
    })

    // Register product routes
    route.RegisterProductRoutes(router, h)

    return router
}
```

### 10.6 `main.go`

```go
package main

import "auron/product-service/cmd"

func main() {
    cmd.Run()
}
```

### 10.7 Database Migration

```go
func runMigrations(db *gorm.DB) error {
    return db.AutoMigrate(
        &domain.Product{},
        &domain.Category{},
        &domain.Inventory{},
    )
}
```

**Note:** AutoMigrate is suitable for development. For production, use `golang-migrate` with SQL migration files.

### 10.8 tsvector Index Bootstrapping

```go
func bootstrapSearchIndex(db *gorm.DB) error {
    // Create tsvector column if not exists
    db.Exec("ALTER TABLE products ADD COLUMN IF NOT EXISTS search_vector tsvector")

    // Create GIN index
    db.Exec("CREATE INDEX IF NOT EXISTS products_search_idx ON products USING GIN(search_vector)")

    // Populate search_vector for existing records
    db.Exec(`
        UPDATE products SET search_vector = to_tsvector('english', name || ' ' || COALESCE(description, ''))
        WHERE search_vector IS NULL OR search_vector = ''
    `)

    // Create trigger to auto-update search_vector on product changes
    db.Exec(`
        CREATE OR REPLACE FUNCTION products_search_vector_trigger() RETURNS trigger AS $$
        BEGIN
            NEW.search_vector := to_tsvector('english', NEW.name || ' ' || COALESCE(NEW.description, ''));
            RETURN NEW;
        END;
        $$ LANGUAGE plpgsql;

        DROP TRIGGER IF EXISTS products_search_vector_update ON products;
        CREATE TRIGGER products_search_vector_update
            BEFORE INSERT OR UPDATE ON products
            FOR EACH ROW EXECUTE FUNCTION products_search_vector_trigger();
    `)

    return nil
}
```

Call this in `runMigrations()` after `AutoMigrate`.

---

## 11. Database & Migrations

### 11.1 Tables (from Technical Plan §5)

**categories:**
```sql
CREATE TABLE categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) UNIQUE NOT NULL,
    parent_id  UUID REFERENCES categories(id),
    created_at TIMESTAMP DEFAULT NOW()
);
```

**products:**
```sql
CREATE TABLE products (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id    UUID REFERENCES categories(id),
    name           VARCHAR(500) NOT NULL,
    description    TEXT,
    price          DECIMAL(12, 2) NOT NULL,
    image_url      TEXT,
    search_vector  TSVECTOR,
    is_active      BOOLEAN DEFAULT true,
    created_at     TIMESTAMP DEFAULT NOW(),
    updated_at     TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_search ON products USING GIN(search_vector);
CREATE INDEX idx_products_price ON products(price);
```

**inventory:**
```sql
CREATE TABLE inventory (
    product_id         UUID PRIMARY KEY REFERENCES products(id),
    total_quantity     INTEGER NOT NULL DEFAULT 0,
    reserved_quantity  INTEGER NOT NULL DEFAULT 0,
    version            INTEGER NOT NULL DEFAULT 0,
    updated_at         TIMESTAMP DEFAULT NOW()
);
```

### 11.2 Migration Strategy

**Development:** GORM `AutoMigrate` (sufficient for local dev)

**Production:** SQL migration files via `golang-migrate`

```
migrations/
├── 001_create_categories.up.sql
├── 001_create_categories.down.sql
├── 002_create_products.up.sql
├── 002_create_products.down.sql
├── 003_create_inventory.up.sql
├── 003_create_inventory.down.sql
└── 004_create_search_index.up.sql
```

---

## 12. Kafka Integration

### 12.1 Events to Publish

| Event | Topic | When | Key |
|---|---|---|---|
| Product Created | `product.created` | After successful product creation | `product_id` |
| Product Updated | `product.updated` | After successful product update | `product_id` |
| Product Deleted | `product.deleted` | After successful product deletion | `product_id` |

### 12.2 Event Payload Structure

```json
{
  "event_id": "uuid",
  "event_type": "product.created",
  "timestamp": "2025-01-01T00:00:00Z",
  "payload": {
    "product_id": "uuid",
    "name": "Laptop Pro 16\"",
    "category_id": "uuid",
    "price": 1299.99,
    "is_active": true
  }
}
```

### 12.3 Integration with Shared Library

Use `shared/kafka/producer.go`:

```go
import "github.com/auron/shared/kafka"

func setupKafkaPublisher(brokers string) domain.EventPublisher {
    if brokers == "" {
        return &noopPublisher{} // Silent no-op for dev
    }

    return kafka.NewProducer(&kafka.ProducerConfig{
        Brokers: strings.Split(brokers, ","),
        Topic:   "product.created", // Default topic
    })
}
```

**Note:** Kafka integration is **optional for initial implementation**. The service should work without Kafka (use a no-op publisher for dev).

---

## 13. Configuration & Environment

### 13.1 `.env.example`

```env
# Product Service Configuration
PORT=8082
DATABASE_URL=postgres://auron:auron_pass@products-db:5433/products_db?sslmode=disable
REDIS_URL=redis://redis:6379/0
KAFKA_BROKERS=kafka:29092
ENVIRONMENT=dev

# Cache Settings
CACHE_TTL=5m
```

### 13.2 Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `PORT` | No | `8082` | HTTP port |
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `REDIS_URL` | Yes | — | Redis connection string |
| `KAFKA_BROKERS` | No | — | Comma-separated Kafka brokers |
| `ENVIRONMENT` | No | `dev` | Environment name (dev/staging/prod) |
| `CACHE_TTL` | No | `5m` | Cache time-to-live duration |

---

## 14. Implementation Checklist

### Phase 1: Foundation (Domain Layer)

- [ ] Update `internal/domain/errors.go` (expand error set)
- [ ] Create `internal/domain/repository.go` (ProductRepository interface + ProductFilter)
- [ ] Create `internal/domain/service.go` (ProductService interface)
- [ ] Create `internal/domain/cache.go` (ProductCache interface)
- [ ] Create `internal/domain/events.go` (EventPublisher interface + topic constants)
- [ ] Update `internal/domain/product.go` (add Category.ParentID field)

### Phase 2: Data Access (Repository Layer)

- [ ] Create `internal/repository/product_repository.go`
  - [ ] `CreateProduct` (with inventory row creation)
  - [ ] `GetProductByID` (with Category preload)
  - [ ] `ListProducts` (with filtering, sorting, pagination, full-text search)
  - [ ] `UpdateProduct`
  - [ ] `DeleteProduct`
  - [ ] `CreateCategory`
  - [ ] `ListCategories`
  - [ ] `GetCategoryByID`
  - [ ] `GetCategoryBySlug`

### Phase 3: Caching (Cache Layer)

- [ ] Create `internal/cache/product_cache.go`
  - [ ] `GetProduct` / `SetProduct` / `DeleteProduct`
  - [ ] `GetProductList` / `SetProductList`
  - [ ] `InvalidateProductList` (SCAN + DEL pattern)
  - [ ] Cache key generation helper

### Phase 4: Business Logic (Service Layer)

- [ ] Create `internal/service/product_service.go`
  - [ ] `CreateProduct` (validate → create → cache → publish event)
  - [ ] `GetProduct` (cache-first read)
  - [ ] `ListProducts` (cache-first read with filter hashing)
  - [ ] `UpdateProduct` (validate → update → invalidate cache → publish event)
  - [ ] `DeleteProduct` (delete → invalidate cache → publish event)
  - [ ] `CreateCategory` (validate → create)
  - [ ] `ListCategories` (direct repo call)
  - [ ] `normalizeFilter` helper (defaults + validation)

### Phase 5: HTTP Interface (Handler Layer)

- [ ] Create `internal/handler/product_handler.go`
  - [ ] `ListProducts` (parse query params → filter → respond)
  - [ ] `GetProduct` (parse UUID → respond)
  - [ ] `CreateProduct` (bind JSON → validate → respond)
  - [ ] `UpdateProduct` (bind JSON → validate → respond)
  - [ ] `DeleteProduct` (parse UUID → respond)
  - [ ] `ListCategories` (respond)
  - [ ] `CreateCategory` (bind JSON → validate → respond)
  - [ ] `handleServiceError` (error → HTTP status mapping)
  - [ ] `parseFilter` helper (query param parsing)

### Phase 6: Routing (Route Layer)

- [ ] Create `internal/route/product_route.go`
  - [ ] Register public GET routes
  - [ ] Register admin POST/PUT/DELETE routes
  - [ ] Configure middleware (gateway-forwarded headers)

### Phase 7: Bootstrap (Outer Layer)

- [ ] Create `cmd/config.go` (config struct + loading)
- [ ] Create `cmd/dotenv.go` (.env file loading)
- [ ] Create `cmd/infrastructure.go` (DB + Redis setup)
- [ ] Create `cmd/kafka.go` (Kafka publisher setup)
- [ ] Create `cmd/run.go` (DI wiring)
- [ ] Create `cmd/server.go` (Gin router + health check)
- [ ] Create `main.go` (entry point)
- [ ] Add tsvector index bootstrapping
- [ ] Add graceful shutdown

### Phase 8: Configuration & Deployment

- [ ] Create `.env.example`
- [ ] Create `Dockerfile` (multi-stage: golang builder → alpine runner)
- [ ] Update `go.mod` with required dependencies
- [ ] Verify docker-compose.yml integration (port 8082, health check)

### Phase 9: Testing & Validation

- [ ] Write unit tests for service layer (`service/product_service_test.go`)
- [ ] Write unit tests for repository layer (with test DB)
- [ ] Write integration tests (full HTTP flow with real DB + Redis)
- [ ] Smoke test: `curl http://localhost:8082/health`
- [ ] Smoke test: Create category → Create product → List products → Get product
- [ ] Cache validation test: Update product → verify cache miss on next GET
- [ ] `go test ./...` passes with 0 failures

---

## 15. File Structure

### Final Directory Layout

```
services/product-service/
├── main.go                          # Entry point
├── go.mod                           # Go module definition
├── go.sum                           # Dependency lock file
├── Dockerfile                       # Multi-stage build
├── .env.example                     # Environment template
│
├── cmd/
│   ├── config.go                    # Configuration loading
│   ├── dotenv.go                    # .env file loading
│   ├── infrastructure.go            # Database + Redis setup
│   ├── kafka.go                     # Kafka producer setup
│   ├── run.go                       # Main orchestration (DI wiring)
│   └── server.go                    # Gin router + health check
│
├── internal/
│   ├── domain/
│   │   ├── product.go               # Entities (Product, Category, Inventory) + DTOs ✅
│   │   ├── errors.go                # Error types ⚠️
│   │   ├── repository.go            # ProductRepository interface ❌
│   │   ├── service.go               # ProductService interface ❌
│   │   ├── cache.go                 # ProductCache interface ❌
│   │   └── events.go                # EventPublisher interface + topic constants ❌
│   │
│   ├── repository/
│   │   └── product_repository.go    # GORM implementation ❌
│   │
│   ├── cache/
│   │   └── product_cache.go         # Redis implementation ❌
│   │
│   ├── service/
│   │   └── product_service.go       # Business logic ❌
│   │
│   ├── handler/
│   │   └── product_handler.go       # HTTP handlers ❌
│   │
│   ├── route/
│   │   └── product_route.go         # Route registration ❌
│   │
│   ├── middleware/                   # (Optional — for future use)
│   │
│   └── events/                       # (Optional — for Kafka event structs)
│
└── migrations/                       # (Optional — for production migrations)
    ├── 001_create_categories.up.sql
    └── ...
```

**Legend:**
- ✅ = Already exists and complete
- ⚠️ = Exists but needs updates
- ❌ = Needs to be created

---

## Appendix A: Key Design Decisions

| Decision | Rationale |
|---|---|
| Cache-first reads | Reduces DB load for read-heavy product catalog traffic |
| Separate cache vs repo interfaces | Single Responsibility — each layer has one concern |
| No auth middleware in service | API Gateway handles all cross-cutting concerns |
| Async event publishing | Don't block HTTP response on Kafka availability |
| AutoMigrate for dev, SQL migrations for prod | Fast iteration in dev, auditable changes in prod |
| tsvector trigger on INSERT/UPDATE | Keeps search index in sync without application logic |
| SCAN for cache invalidation | Production-safe (non-blocking) alternative to KEYS |
| Inventory row created with product | Ensures inventory record exists before inventory-service manages it |

## Appendix B: Dependencies

From existing `go.mod`:

```go
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/google/uuid v1.6.0
    github.com/redis/go-redis/v9 v9.4.0
    github.com/segmentio/kafka-go v0.4.47
    gorm.io/driver/postgres v1.5.4
    gorm.io/gorm v1.25.5
)

replace github.com/auron/shared => ../../shared
```

## Appendix C: Health Check Configuration

Per docker-compose.yml health check pattern:

```yaml
product-service:
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8082/health"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 30s
```

The `/health` endpoint returns:

```json
{
  "status": "healthy",
  "service": "product-service",
  "timestamp": "2025-01-01T00:00:00Z"
}
```

---

*Plan created: 2025-04-14*  
*Based on: ecommerce-technical-plan.md §4.3, §5, §7, §10*