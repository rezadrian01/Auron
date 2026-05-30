# GCS Product Image Upload — Implementation Plan

> **Scope:** Add Google Cloud Storage image support to the `product-service`, with support for **multiple images per product**.  
> **Approach:** Images are managed as a separate resource attached to a product. Admins upload images after creating a product via dedicated endpoints. Images are ordered by `position`; position 0 is the primary (display) image.

---

## What changes

| Area | Change |
|---|---|
| `product-service` | New `product_images` table (id, product_id, url, position) |
| `product-service` | Remove `image_url` from `ProductRequest` and `Product` entity |
| `product-service` | `ProductResponse` gains `images []ProductImage`; `image_url` kept as computed convenience field |
| `product-service` | New `StorageService` interface + GCS implementation |
| `product-service` | 3 new endpoints: upload, delete, reorder |
| `product-service` | `DeleteProduct` cleans up all GCS objects for the product |
| `docker-compose.yml` | Inject `GCS_BUCKET_NAME`, `GCS_CREDENTIALS_JSON` into product-service |
| `.env.example` | Document the two new variables |
| `docs/API_DOCS.md` | Document new endpoints, updated response shape |

---

## 1. GCP Setup (one-time, manual)

### 1a. Create the bucket

```bash
gcloud storage buckets create gs://auron-product-images \
  --project=YOUR_PROJECT_ID \
  --location=ASIA-SOUTHEAST1 \
  --uniform-bucket-level-access
```

### 1b. Make bucket publicly readable

Product images must be publicly accessible via URL.

```bash
gcloud storage buckets add-iam-policy-binding gs://auron-product-images \
  --member=allUsers \
  --role=roles/storage.objectViewer
```

### 1c. Create a service account

```bash
gcloud iam service-accounts create auron-product-service \
  --display-name="Auron Product Service" \
  --project=YOUR_PROJECT_ID
```

### 1d. Grant the service account Object Admin on the bucket

Needs both create (upload) and delete (cleanup on update/delete).

```bash
gcloud storage buckets add-iam-policy-binding gs://auron-product-images \
  --member=serviceAccount:auron-product-service@YOUR_PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/storage.objectAdmin
```

### 1e. Generate the key JSON

```bash
gcloud iam service-accounts keys create gcs-credentials.json \
  --iam-account=auron-product-service@YOUR_PROJECT_ID.iam.gserviceaccount.com
```

Never commit this file. See §5 for how to pass it to Docker.

---

## 2. Database — new table

No changes to the existing `products` table. A new `product_images` table is added.

```sql
-- Applied automatically via GORM AutoMigrate
CREATE TABLE product_images (
    id         UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID      NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url        TEXT      NOT NULL,
    position   INT       NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_images_product_id ON product_images(product_id);
```

`ON DELETE CASCADE` ensures images are removed from the DB if a product is hard-deleted via SQL. The service layer also deletes the GCS objects explicitly before the DB row is removed.

### Migration note — `image_url` column

The `image_url` column on the `products` table already exists but will no longer be written to. GORM AutoMigrate never drops columns, so it stays harmlessly in the DB. To clean it up manually after migration:

```sql
ALTER TABLE products DROP COLUMN image_url;
```

This is optional and can be done at any time after deploying the new code, since nothing reads or writes to it anymore.

---

## 3. New API Endpoints

All three require `Authorization: Bearer <admin-token>` (enforced by the gateway admin middleware).

### 3a. Upload image

```
POST /api/products/:id/images
Content-Type: multipart/form-data

Field: image (file)
```

Uploads the file to GCS and creates a `product_images` row. The new image is appended at the end of the existing images (highest position + 1).

**Response `201`:**
```json
{
  "success": true,
  "data": {
    "id": "a3f7c1d2-...",
    "product_id": "b9e2a4f1-...",
    "url": "https://storage.googleapis.com/auron-product-images/products/a3f7c1d2-....jpg",
    "position": 2,
    "created_at": "2026-05-30T10:00:00Z"
  }
}
```

**Constraints:**
- MIME type: `image/jpeg`, `image/png`, `image/webp` only
- Max size: 5 MB
- Product must exist; returns `404` if not

### 3b. Delete image

```
DELETE /api/products/:id/images/:image_id
```

Deletes the GCS object and removes the `product_images` row. After deletion, the remaining images are NOT automatically re-sequenced — call reorder if needed.

**Response `200`:**
```json
{ "success": true, "message": "image deleted" }
```

Returns `404` if the image_id does not belong to the given product.

### 3c. Reorder images

```
PUT /api/products/:id/images/reorder
Content-Type: application/json

{
  "image_ids": ["uuid-1", "uuid-2", "uuid-3"]
}
```

Assigns `position = 0, 1, 2, ...` to the image IDs in the order provided. The image at position 0 is the primary (display) image shown on product cards. All provided IDs must belong to the product.

**Response `200`:**
```json
{
  "success": true,
  "data": [
    { "id": "uuid-1", "url": "...", "position": 0, "created_at": "..." },
    { "id": "uuid-2", "url": "...", "position": 1, "created_at": "..." },
    { "id": "uuid-3", "url": "...", "position": 2, "created_at": "..." }
  ]
}
```

---

## 4. Updated Response Shape

`ProductResponse` gains an `images` field. The existing `image_url` field is kept as a computed convenience value (the URL of the first image by position) so that the frontend can use it for product cards without iterating the images array.

```json
{
  "id": "...",
  "category_id": "...",
  "name": "Product Name",
  "description": "...",
  "price": 99.99,
  "image_url": "https://storage.googleapis.com/auron-product-images/products/primary.jpg",
  "images": [
    { "id": "...", "product_id": "...", "url": "https://...", "position": 0, "created_at": "..." },
    { "id": "...", "product_id": "...", "url": "https://...", "position": 1, "created_at": "..." }
  ],
  "is_active": true,
  "created_at": "...",
  "updated_at": "..."
}
```

`image_url` is `""` (empty string) when a product has no images.

### `ProductRequest` change (breaking)

`image_url` is removed from the create and update request body. Images are now managed exclusively via the `/api/products/:id/images` endpoints.

**Before:**
```json
{ "category_id": "...", "name": "...", "description": "...", "price": 99.99, "image_url": "https://...", "is_active": true }
```

**After:**
```json
{ "category_id": "...", "name": "...", "description": "...", "price": 99.99, "is_active": true }
```

---

## 5. Admin Workflow (Frontend — Phase 7)

### Create product with images

```
1. Fill name, description, price, category → POST /api/products → get product_id
2. Upload images one at a time → POST /api/products/:id/images (repeat per file)
3. Reorder if needed → PUT /api/products/:id/images/reorder
```

### Edit product images

```
Existing images shown → admin can:
  ├─► Upload more      → POST  /api/products/:id/images
  ├─► Delete one       → DELETE /api/products/:id/images/:image_id
  └─► Drag to reorder  → PUT   /api/products/:id/images/reorder
```

### Primary image

The image at `position = 0` is the primary image. It shows on product cards and as the hero image on the product detail page. Dragging an image to the first slot in the admin form and saving the new order sets it as primary.

---

## 6. Backend Code Changes

### 6a. Add dependency

```bash
cd services/product-service
go get cloud.google.com/go/storage
go get google.golang.org/api/option
go mod tidy
```

### 6b. `internal/domain/storage.go` — new file

```go
package domain

import (
    "context"
    "io"
)

type StorageService interface {
    UploadImage(ctx context.Context, objectName string, r io.Reader, contentType string) (publicURL string, err error)
    DeleteImage(ctx context.Context, objectName string) error
}

// NoopStorage is used when GCS env vars are not set.
// UploadImage returns ErrStorageNotConfigured; DeleteImage is silently ignored.
type NoopStorage struct{}

func (NoopStorage) UploadImage(_ context.Context, _ string, _ io.Reader, _ string) (string, error) {
    return "", ErrStorageNotConfigured
}

func (NoopStorage) DeleteImage(_ context.Context, _ string) error {
    return nil
}
```

### 6c. `internal/domain/errors.go` — add new errors

```go
ErrStorageNotConfigured = errors.New("image storage is not configured")
ErrImageNotFound        = errors.New("image not found")
ErrImageLimitExceeded   = errors.New("product already has the maximum number of images")
```

### 6d. `internal/domain/product.go` — changes

**Remove** `ImageURL` field from the `Product` struct (GORM stops writing to the column).

**Add** `ProductImage` entity:
```go
type ProductImage struct {
    ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    ProductID uuid.UUID `json:"product_id" gorm:"type:uuid;not null;index"`
    URL       string    `json:"url" gorm:"type:text;not null"`
    Position  int       `json:"position" gorm:"not null;default:0"`
    CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (ProductImage) TableName() string { return "product_images" }
```

**Update** `Product` to include the association:
```go
type Product struct {
    ID          uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    CategoryID  uuid.UUID      `json:"category_id" gorm:"type:uuid;not null;index"`
    Name        string         `json:"name" gorm:"type:varchar(500);not null"`
    Description string         `json:"description" gorm:"type:text"`
    Price       float64        `json:"price" gorm:"type:numeric(12,2);not null;index"`
    Images      []ProductImage `json:"images,omitempty" gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:CASCADE"`
    SearchVector string        `json:"-" gorm:"-"`
    IsActive    bool           `json:"is_active" gorm:"not null;default:true;index"`
    CreatedAt   time.Time      `json:"created_at" gorm:"not null;default:now()"`
    UpdatedAt   time.Time      `json:"updated_at" gorm:"not null;default:now()"`
    Category    *Category      `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
}
```

**Update** `ProductRequest` — remove `ImageURL`:
```go
type ProductRequest struct {
    CategoryID  uuid.UUID `json:"category_id" binding:"required"`
    Name        string    `json:"name" binding:"required,max=500"`
    Description string    `json:"description" binding:"required"`
    Price       float64   `json:"price" binding:"required,gt=0"`
    IsActive    *bool     `json:"is_active"`
}
```

**Update** `ProductResponse` — add `Images`, computed `ImageURL`:
```go
type ProductResponse struct {
    ID          uuid.UUID      `json:"id"`
    CategoryID  uuid.UUID      `json:"category_id"`
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Price       float64        `json:"price"`
    ImageURL    string         `json:"image_url"`     // computed: images[0].URL or ""
    Images      []ProductImage `json:"images"`
    IsActive    bool           `json:"is_active"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    Category    *Category      `json:"category,omitempty"`
}
```

**Update** `ToResponse()`:
```go
func (p *Product) ToResponse() *ProductResponse {
    resp := &ProductResponse{
        ID:          p.ID,
        CategoryID:  p.CategoryID,
        Name:        p.Name,
        Description: p.Description,
        Price:       p.Price,
        Images:      p.Images,
        IsActive:    p.IsActive,
        CreatedAt:   p.CreatedAt,
        UpdatedAt:   p.UpdatedAt,
        Category:    p.Category,
    }
    if resp.Images == nil {
        resp.Images = []ProductImage{} // always return array, never null
    }
    if len(p.Images) > 0 {
        resp.ImageURL = p.Images[0].URL // position 0 = primary
    }
    return resp
}
```

**Add** `AddImageRequest` and `ReorderRequest` DTOs:
```go
type AddImageRequest struct {
    URL      string `json:"url"`
    Position int    `json:"position"`
}

type ReorderImagesRequest struct {
    ImageIDs []uuid.UUID `json:"image_ids" binding:"required,min=1"`
}
```

### 6e. `internal/domain/service.go` — add image methods

```go
type ProductService interface {
    // ... existing product and category methods ...

    // Image operations
    AddProductImage(ctx context.Context, productID uuid.UUID, url string) (*ProductImage, error)
    DeleteProductImage(ctx context.Context, productID, imageID uuid.UUID) (*ProductImage, error)
    ReorderProductImages(ctx context.Context, productID uuid.UUID, imageIDs []uuid.UUID) ([]ProductImage, error)
    GetProductImages(ctx context.Context, productID uuid.UUID) ([]ProductImage, error)
}
```

`DeleteProductImage` returns the deleted image so the caller (handler) can delete the GCS object.

### 6f. `internal/domain/repository.go` — add image methods

```go
type ProductRepository interface {
    // ... existing methods ...

    AddProductImage(ctx context.Context, image *ProductImage) error
    GetProductImage(ctx context.Context, productID, imageID uuid.UUID) (*ProductImage, error)
    GetProductImages(ctx context.Context, productID uuid.UUID) ([]ProductImage, error)
    DeleteProductImage(ctx context.Context, productID, imageID uuid.UUID) error
    UpdateProductImagePositions(ctx context.Context, images []ProductImage) error
    GetAllProductImages(ctx context.Context, productID uuid.UUID) ([]ProductImage, error)
}
```

### 6g. `internal/storage/gcs.go` — new file

```go
package storage

import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
)

type GCSStorage struct {
    client     *storage.Client
    bucketName string
}

func NewGCSStorage(ctx context.Context, bucketName, credJSON string) (*GCSStorage, error) {
    var opts []option.ClientOption
    if credJSON != "" {
        opts = append(opts, option.WithCredentialsJSON([]byte(credJSON)))
    }
    // Falls back to GOOGLE_APPLICATION_CREDENTIALS or metadata server if credJSON is empty.
    client, err := storage.NewClient(ctx, opts...)
    if err != nil {
        return nil, fmt.Errorf("gcs: failed to create client: %w", err)
    }
    return &GCSStorage{client: client, bucketName: bucketName}, nil
}

func (g *GCSStorage) UploadImage(ctx context.Context, objectName string, r io.Reader, contentType string) (string, error) {
    obj := g.client.Bucket(g.bucketName).Object(objectName)
    w := obj.NewWriter(ctx)
    w.ContentType = contentType
    w.CacheControl = "public, max-age=31536000" // 1 year — UUID-named, never stale

    if _, err := io.Copy(w, r); err != nil {
        _ = w.Close()
        return "", fmt.Errorf("gcs: upload failed: %w", err)
    }
    if err := w.Close(); err != nil {
        return "", fmt.Errorf("gcs: finalise failed: %w", err)
    }
    return fmt.Sprintf("https://storage.googleapis.com/%s/%s", g.bucketName, objectName), nil
}

func (g *GCSStorage) DeleteImage(ctx context.Context, objectName string) error {
    err := g.client.Bucket(g.bucketName).Object(objectName).Delete(ctx)
    if err == storage.ErrObjectNotExist {
        return nil // already gone — treat as success
    }
    return err
}

func (g *GCSStorage) BucketName() string { return g.bucketName }
```

### 6h. `internal/handler/product_handler.go` — new image handlers

Update struct to include storage:
```go
type ProductHandler struct {
    service domain.ProductService
    storage domain.StorageService
    bucket  string // needed to extract object name for GCS delete
}

func NewProductHandler(service domain.ProductService, storage domain.StorageService, bucket string) *ProductHandler {
    return &ProductHandler{service: service, storage: storage, bucket: bucket}
}
```

**UploadProductImage:**
```go
var (
    allowedMIME   = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}
    maxImageBytes = int64(5 << 20) // 5 MB
)

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

    buf := make([]byte, 512)
    n, _ := file.Read(buf)
    contentType := http.DetectContentType(buf[:n])
    ext, ok := allowedMIME[contentType]
    if !ok {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "only JPEG, PNG, and WebP are accepted"})
        return
    }
    if seeker, ok := file.(io.Seeker); ok {
        seeker.Seek(0, io.SeekStart)
    }

    objectName := fmt.Sprintf("products/%s%s", uuid.New().String(), ext)
    url, err := h.storage.UploadImage(c.Request.Context(), objectName, file, contentType)
    if err != nil {
        if errors.Is(err, domain.ErrStorageNotConfigured) {
            c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "image storage is not configured"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to upload image"})
        return
    }

    image, err := h.service.AddProductImage(c.Request.Context(), productID, url)
    if err != nil {
        // GCS upload succeeded but DB insert failed — clean up the orphan object
        _ = h.storage.DeleteImage(c.Request.Context(), objectName)
        h.handleError(c, err)
        return
    }

    c.JSON(http.StatusCreated, gin.H{"success": true, "data": image})
}
```

**DeleteProductImage:**
```go
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

    // Service deletes the DB row and returns the deleted image (for GCS cleanup)
    deleted, err := h.service.DeleteProductImage(c.Request.Context(), productID, imageID)
    if err != nil {
        h.handleError(c, err)
        return
    }

    // Delete the GCS object — best-effort, don't fail the request if GCS is slow
    if objectName, ok := extractGCSObjectName(deleted.URL, h.bucket); ok {
        _ = h.storage.DeleteImage(c.Request.Context(), objectName)
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "image deleted"})
}
```

**ReorderProductImages:**
```go
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
```

**Shared helper:**
```go
// extractGCSObjectName extracts the GCS object path from a full public URL.
// Returns ("", false) for non-GCS or non-bucket URLs.
func extractGCSObjectName(imageURL, bucketName string) (string, bool) {
    prefix := fmt.Sprintf("https://storage.googleapis.com/%s/", bucketName)
    if strings.HasPrefix(imageURL, prefix) {
        return strings.TrimPrefix(imageURL, prefix), true
    }
    return "", false
}
```

**handleError** — add new cases:
```go
case errors.Is(err, domain.ErrImageNotFound):
    c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
case errors.Is(err, domain.ErrStorageNotConfigured):
    c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": err.Error()})
```

### 6i. `internal/route/product_route.go` — register new routes

```go
func RegisterProductRoutes(router *gin.Engine, h *handler.ProductHandler) {
    api := router.Group("/")

    api.GET("/products", h.GetProducts)
    api.GET("/products/:id", h.GetProductByID)
    api.POST("/products", h.CreateProduct)
    api.PUT("/products/:id", h.UpdateProduct)
    api.DELETE("/products/:id", h.DeleteProduct)

    // Image management
    api.POST("/products/:id/images", h.UploadProductImage)
    api.DELETE("/products/:id/images/:image_id", h.DeleteProductImage)
    api.PUT("/products/:id/images/reorder", h.ReorderProductImages)

    api.GET("/categories", h.GetCategories)
    api.POST("/categories", h.CreateCategory)
}
```

### 6j. `internal/repository/product_repository.go` — add image methods

Implement `AddProductImage`, `GetProductImage`, `GetProductImages`, `DeleteProductImage`, `UpdateProductImagePositions`, and update `GetProductByID` to preload images:

```go
// GetProductByID — add Preload for images, ordered by position
db.Preload("Images", func(db *gorm.DB) *gorm.DB {
    return db.Order("position ASC")
}).Where("id = ? AND is_active = true", id).First(&product)

// GetProducts — similarly preload Images ordered by position
```

### 6k. `internal/service/product_service.go` — implement image methods

`AddProductImage`: verify product exists, get current max position, insert with `position = max + 1`.

`DeleteProductImage`: verify image belongs to product, delete DB row, return the deleted image.

`ReorderProductImages`: verify all IDs belong to the product, run a bulk UPDATE setting position = array index within a transaction.

`DeleteProduct`: before deleting the product, fetch all images, delete GCS objects for each, then delete the DB row (CASCADE handles `product_images`).

### 6l. `cmd/infrastructure.go` — add `setupGCS`

```go
func setupGCS(ctx context.Context, bucketName, credJSON string) (domain.StorageService, error) {
    if bucketName == "" {
        log.Println("GCS_BUCKET_NAME not set — image upload disabled")
        return domain.NoopStorage{}, nil
    }
    svc, err := gcsStorage.NewGCSStorage(ctx, bucketName, credJSON)
    if err != nil {
        return nil, fmt.Errorf("failed to initialise GCS: %w", err)
    }
    log.Printf("GCS storage initialised (bucket: %s)", bucketName)
    return svc, nil
}
```

### 6m. `cmd/config.go` — add GCS fields

```go
type appConfig struct {
    Port           string
    DatabaseURL    string
    RedisURL       string
    KafkaBrokers   string
    GCSBucketName  string
    GCSCredentials string
}
```

### 6n. `cmd/run.go` — wire everything up

```go
storageSvc, err := setupGCS(context.Background(), cfg.GCSBucketName, cfg.GCSCredentials)
if err != nil {
    log.Fatalf("failed to set up GCS: %v", err)
}

// Pass storage to both service (for DeleteProduct cleanup) and handler (for upload/delete)
svc := service.NewProductService(repo, productCache, publisher, storageSvc, cfg.GCSBucketName)
h := handler.NewProductHandler(svc, storageSvc, cfg.GCSBucketName)
```

### 6o. `cmd/infrastructure.go` — add `ProductImage` to AutoMigrate

```go
if err := db.AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Inventory{}, &domain.ProductImage{}); err != nil {
    return err
}
```

---

## 7. Environment Variables

### `.env.example`

```env
# ── GCS Image Storage ──────────────────────────────────────────────────────────
GCS_BUCKET_NAME=auron-product-images

# Paste the raw JSON content of your service account key here.
# To prepare: cat gcs-credentials.json (copy the entire JSON, single or multi-line)
# Leave empty to disable image upload (service still starts without it).
GCS_CREDENTIALS_JSON=
```

### `docker-compose.yml`

```yaml
product-service:
  environment:
    - PORT=8082
    - DATABASE_URL=postgres://auron:auron_pass@products-db:5432/products_db?sslmode=disable
    - REDIS_URL=redis://redis:6379/0
    - KAFKA_BROKERS=kafka:29092
    - GCS_BUCKET_NAME=${GCS_BUCKET_NAME}
    - GCS_CREDENTIALS_JSON=${GCS_CREDENTIALS_JSON}
```

---

## 8. Gateway — confirm admin routing

The gateway already routes `/products*` to the product-service. Confirm that `POST /products/:id/images`, `DELETE /products/:id/images/:image_id`, and `PUT /products/:id/images/reorder` are covered by the admin middleware (require `role = admin`). Check `services/api-gateway/routes/router.go`.

---

## 9. Frontend Impact (Phase 7 — Admin Panel)

### Updated `types/product.ts`

```typescript
export interface ProductImage {
  id: string
  product_id: string
  url: string
  position: number
  created_at: string
}

export interface Product {
  // ... existing fields ...
  image_url: string      // computed: images[0].url or ""
  images: ProductImage[] // always an array, never null
}
```

Remove `image_url` from `productSchema` in `lib/validations/product.ts` — it's no longer a form field.

### `lib/api/products.ts` additions

```typescript
uploadProductImage(productId: string, file: File): Promise<ProductImage>
deleteProductImage(productId: string, imageId: string): Promise<void>
reorderProductImages(productId: string, imageIds: string[]): Promise<ProductImage[]>
```

### Admin product form image section

The product form splits into two steps:
1. Save product details → get `product_id`
2. Image manager panel: upload (file input), preview thumbnails, drag-to-reorder, delete button per image

---

## 10. Implementation Order

1. GCP setup (bucket, service account, key) — manual
2. `go get cloud.google.com/go/storage google.golang.org/api/option`
3. `internal/domain/storage.go` (interface + NoopStorage)
4. `internal/domain/errors.go` (add new errors)
5. `internal/domain/product.go` (add ProductImage, update Product, remove ImageURL, update DTOs)
6. `internal/domain/service.go` (add image methods)
7. `internal/domain/repository.go` (add image methods)
8. `internal/storage/gcs.go`
9. `internal/repository/product_repository.go` (implement image methods + update preloads)
10. `internal/service/product_service.go` (implement image methods + update DeleteProduct)
11. `internal/handler/product_handler.go` (add image handlers, update constructor)
12. `internal/route/product_route.go` (register new routes)
13. `cmd/config.go`, `cmd/infrastructure.go`, `cmd/run.go` (wire GCS)
14. `docker-compose.yml` + `.env.example`
15. Test with curl:
    ```bash
    # 1. Create a product (no image_url in body anymore)
    curl -X POST http://localhost:8080/api/products \
      -H "Authorization: Bearer <admin-token>" \
      -H "Content-Type: application/json" \
      -d '{"category_id":"...","name":"Test","description":"...","price":99.99}'

    # 2. Upload images
    curl -X POST http://localhost:8080/api/products/<id>/images \
      -H "Authorization: Bearer <admin-token>" \
      -F "image=@photo1.jpg"

    curl -X POST http://localhost:8080/api/products/<id>/images \
      -H "Authorization: Bearer <admin-token>" \
      -F "image=@photo2.jpg"

    # 3. Reorder (first ID becomes primary)
    curl -X PUT http://localhost:8080/api/products/<id>/images/reorder \
      -H "Authorization: Bearer <admin-token>" \
      -H "Content-Type: application/json" \
      -d '{"image_ids":["<img-id-2>","<img-id-1>"]}'

    # 4. Get product — confirm images array + image_url
    curl http://localhost:8080/api/products/<id>
    ```

---

## 11. What Stays Unchanged

- `GET /api/products` and `GET /api/products/:id` — same paths, enriched response (adds `images` array)
- `POST /api/categories`, `GET /api/categories` — not affected
- Inventory service, order service, payment service — not affected
- `image_url` in `ProductResponse` — still present, computed from `images[0]`
