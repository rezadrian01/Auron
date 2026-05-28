# Auron — Fixes & Product Service Completion Plan

> **Scope:** JWT algorithm fix · Gateway auth wiring · Product service layer · Product service entry point · Cache key nil panic  
> **Branch:** `feature/product-service`  
> **Order:** Tasks must be done in sequence — each builds on the previous.

---

## Table of Contents

1. [Task 1 — Fix JWT Algorithm Mismatch](#task-1--fix-jwt-algorithm-mismatch)
2. [Task 2 — Wire Auth Middleware in Gateway](#task-2--wire-auth-middleware-in-gateway)
3. [Task 3 — Implement Product Service Layer](#task-3--implement-product-service-layer)
4. [Task 4 — Add Product Service Entry Point](#task-4--add-product-service-entry-point)
5. [Task 5 — Fix GenerateCacheKey Nil Panic](#task-5--fix-generatecachekey-nil-panic)
6. [Verification Checklist](#verification-checklist)

---

## Task 1 — Fix JWT Algorithm Mismatch

### Problem

The user service signs tokens with **HS256** but the API Gateway validates expecting **RS256 (RSA)**. Every token the user-service issues fails gateway validation — auth is completely broken end-to-end.

| File | Algorithm | Secret Source |
|---|---|---|
| `services/user-service/internal/service/user_service.go:369` | HS256 | `JWT_SECRET` env var |
| `services/user-service/internal/middleware/auth_middleware.go:39` | HS256 | `JWT_SECRET` env var |
| `services/api-gateway/middleware/auth.go:94` | **RS256** | PEM file at `JWT_PUBLIC_KEY` path |

**Decision: standardize on HS256.** Both user-service files already use it. RS256 is architecturally better for multi-service trust but adds operational complexity (key file management). Can be upgraded later.

### Files to Change

#### `services/api-gateway/middleware/auth.go`

- Replace `type JWTMiddleware struct { publicKey *rsa.PublicKey }` with `type JWTMiddleware struct { secret []byte }`
- Replace `NewJWTMiddleware(keyPath string)` (reads PEM file) with `NewJWTMiddleware(secret string)` (takes raw string)
- In `Auth()` and `RequireAuth()`: change the `jwt.ParseWithClaims` key func to check for `*jwt.SigningMethodHMAC` and return `j.secret`
- Update the `Claims` struct: the user-service puts the user UUID in `"sub"`, not `"user_id"`. Change `UserID string json:"user_id"` → `Sub string json:"sub"` and set `c.Set(UserIDKey, claims.Sub)` inside the middleware
- Remove now-unused imports: `crypto/rsa`, `encoding/pem`, `io/ioutil`

#### `services/api-gateway/config/config.go`

- Replace `JWTPublicKeyPath string` (env: `JWT_PUBLIC_KEY`) with `JWTSecret string` (env: `JWT_SECRET`, no default — must be set explicitly)

#### `services/api-gateway/main.go`

- Replace `middleware.NewJWTMiddleware(cfg.JWTPublicKeyPath)` with `middleware.NewJWTMiddleware(cfg.JWTSecret)`

#### `services/api-gateway/.env` and `.env.example`

- Remove `JWT_PUBLIC_KEY=...`
- Add `JWT_SECRET=<same value as in user-service .env>`

#### `docker-compose.yml`

- Add `JWT_SECRET` to the `api-gateway` environment block, sourced from the root `.env` so both containers share the identical value

---

## Task 2 — Wire Auth Middleware in Gateway

### Problem

`JWTMiddleware.RequireAuth()` and `RequireRole()` exist but are **never called** in `services/api-gateway/routes/router.go`. All routes — including write endpoints, user profile, cart, and orders — are fully unprotected.

### Route Protection Matrix

| Route | Methods | Auth | Role |
|---|---|---|---|
| `/api/auth/logout` | POST | Yes | Any |
| `/api/users/*` | All | Yes | Any |
| `/api/products` | GET | No | — |
| `/api/products/:id` | GET | No | — |
| `/api/products` | POST | Yes | `admin` |
| `/api/products/:id` | PUT, DELETE | Yes | `admin` |
| `/api/categories` | GET | No | — |
| `/api/categories` | POST | Yes | `admin` |
| `/api/cart/*` | All | Yes | Any |
| `/api/orders/*` | All | Yes | Any |
| `/api/payments/:id` | GET | Yes | Any |
| `/api/payments/webhook/stripe` | POST | **No** | — (Stripe signs its own payload) |
| `/api/inventory/*` | All | Yes | `admin` |

### Files to Change

#### `services/api-gateway/routes/router.go`

At the top of `Setup()`, instantiate the middleware using the secret from config (after Task 1 lands):

```go
jwtMiddleware, err := middleware.NewJWTMiddleware(cfg.JWTSecret)
if err != nil {
    return fmt.Errorf("create jwt middleware: %w", err)
}
requireAuth  := jwtMiddleware.RequireAuth()
requireAdmin := gin.HandlersChain{jwtMiddleware.RequireAuth(), jwtMiddleware.RequireRole("admin")}
```

Then apply per group:

- `auth` group: add `requireAuth` to the `authProtected` sub-group (logout route)
- `users` group: add `requireAuth` to the group-level `Use()`
- `products` write routes: change the three write routes to use `requireAdmin` handlers prepended
- `categories` POST: add `requireAdmin`
- `cart` group: add `requireAuth` to group-level `Use()`
- `orders` group: add `requireAuth` to group-level `Use()`
- `payments.GET("/:id")`: add `requireAuth` inline on that route only
- `inventory` group: add `requireAdmin` to group-level `Use()`

> **Note:** `proxy.go` already forwards `X-User-ID`, `X-User-Email`, `X-User-Role` headers downstream once the context keys are set by the middleware — no changes needed there.

---

## Task 3 — Implement Product Service Layer

### Problem

Every method in `services/product-service/internal/service/product_service.go` returns `nil, nil` — the file is entirely stubs. Additionally, the concrete method signatures don't match the `domain.ProductService` interface (wrong argument types, missing `context.Context`), so it won't compile.

### Sub-task 3.0 — Fix Interface Signature Mismatch First

`domain.ProductService` interface (`internal/domain/service.go`) uses `context.Context` + `uuid.UUID`:

```go
GetProductByID(ctx context.Context, id uuid.UUID) (*Product, error)
```

But the concrete struct uses bare `string` with no context:

```go
GetProductByID(id string) (*Product, error)   // ← won't satisfy interface
```

**Fix:** Update every method signature in `product_service.go` to match `domain.ProductService` exactly — add `ctx context.Context` as first param and use `uuid.UUID` (not `string`) for IDs.

### Sub-task 3.1 — Read Methods (cache-aside pattern)

**`GetProductByID(ctx, id)`**
1. `cache.GetProduct(ctx, id.String())`
2. On cache miss → `repo.GetProductByID(id)`
3. `cache.SetProduct(ctx, product)` — log error, don't fail
4. Return product

**`GetProducts(ctx, filter)`**
1. Validate filter (page ≥ 1, limit 1–100, sort in `ValidSorts`)
2. Build cache key via `GenerateCacheKey(filter)` (fixed in Task 5)
3. `cache.GetProductList(ctx, cacheKey)`
4. On cache miss → `repo.GetProducts(filter)`
5. `cache.SetProductList(ctx, cacheKey, result)` — log error, don't fail
6. Return result

**`GetCategories(ctx)`, `GetCategoryByID(ctx, id)`, `GetCategoryBySlug(ctx, slug)`**
- Direct repo calls — no caching needed at this stage

### Sub-task 3.2 — Write Methods (invalidate cache + publish event)

**`CreateProduct(ctx, req)`**
1. Verify category exists: `repo.GetCategoryByID(req.CategoryID)` → `ErrCategoryNotFound` if missing
2. Build `domain.Product` from request; set `ID = uuid.New()`, timestamps
3. `repo.CreateProduct(&product)`
4. `cache.SetProduct(ctx, product)` + `cache.InvalidateProductList(ctx)`
5. `publisher.Publish(ctx, TopicProductCreated, product)` — log error, don't fail request
6. Return product

**`UpdateProduct(ctx, id, req)`**
1. Fetch existing: `repo.GetProductByID(id)` → propagate `ErrProductNotFound`
2. If `CategoryID` changed, verify new category exists
3. Apply fields from req, update `UpdatedAt`
4. `repo.UpdateProduct(&product)`
5. `cache.SetProduct(ctx, product)` + `cache.InvalidateProductList(ctx)`
6. `publisher.Publish(ctx, TopicProductUpdated, product)`
7. Return product

**`DeleteProduct(ctx, id)`**
1. Verify exists: `repo.GetProductByID(id)`
2. `repo.DeleteProduct(id)`
3. `cache.DeleteProduct(ctx, id.String())` + `cache.InvalidateProductList(ctx)`
4. `publisher.Publish(ctx, TopicProductDeleted, gin.H{"product_id": id})`
5. Return nil

**`CreateCategory(ctx, req)`**
1. Check slug uniqueness: `repo.GetCategoryBySlug(req.Slug)` → if found, return `ErrCategorySlugExists`
2. Build `domain.Category`; set `ID = uuid.New()`
3. `repo.CreateCategory(&category)`
4. Return category

---

## Task 4 — Add Product Service Entry Point

### Problem

The product service has no `main.go`, no `Dockerfile`, no config loader, no HTTP handler, no route layer. It cannot be built or run.

### Files to Create

```
services/product-service/
├── main.go
├── Dockerfile
├── .env
├── .env.example
├── cmd/
│   ├── config.go
│   ├── dotenv.go
│   ├── infrastructure.go
│   ├── kafka.go
│   ├── run.go
│   └── server.go
└── internal/
    ├── handler/
    │   └── product_handler.go
    └── route/
        └── product_route.go
```

### `cmd/config.go`

```go
type Config struct {
    Port         string
    DatabaseURL  string
    RedisURL     string
    KafkaBrokers string
}
```

Env vars: `PORT` (default `"8082"`), `DATABASE_URL` (required), `REDIS_URL` (default `"redis://localhost:6379/0"`), `KAFKA_BROKERS` (default `"localhost:9092"`).

### `cmd/dotenv.go`

Same pattern as user-service: silently skip if `.env` is absent.

### `cmd/infrastructure.go`

- **DB:** GORM + `gorm.io/driver/postgres`. Run `AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Inventory{})`. Also run the tsvector trigger SQL from `db/004_create_search_index.up.sql` via `db.Exec(...)` after AutoMigrate.
- **Redis:** `redis.ParseURL(cfg.RedisURL)` → `redis.NewClient(opt)`
- **Kafka:** One `kafka.Writer` per topic (`product.created`, `product.updated`, `product.deleted`) — same pattern as `services/user-service/cmd/kafka.go`

### `cmd/run.go`

Wire the dependency graph in order:

```
config → db, redis, kafka
db     → NewProductRepository(db)
redis  → NewProductCache(redisClient)
kafka  → NewKafkaPublisher(writers)
repo + cache + publisher → NewProductService(...)
service → NewProductHandler(service)
handler → RegisterProductRoutes(router, handler)
```

Register graceful shutdown (close DB, Redis, Kafka writers on SIGINT/SIGTERM).

### `cmd/server.go`

Same pattern as user-service `cmd/server.go`:
- `gin.SetMode(gin.ReleaseMode)`
- `gin.New()` + `gin.Logger()` + `gin.Recovery()`
- `GET /health` → `{"status":"healthy","service":"product-service"}`
- `GET /metrics` → stub (Prometheus later)
- Call `route.RegisterProductRoutes(router, h)`

### `main.go`

```go
package main

import "auron/product-service/cmd"

func main() { cmd.Run() }
```

### `internal/handler/product_handler.go`

One handler per endpoint. Keep handlers thin — only HTTP binding and error mapping, no business logic.

| Handler | HTTP | Domain call |
|---|---|---|
| `GetProducts` | `GET /products` | `service.GetProducts(ctx, filter)` |
| `GetProductByID` | `GET /products/:id` | `service.GetProductByID(ctx, id)` |
| `CreateProduct` | `POST /products` | `service.CreateProduct(ctx, req)` |
| `UpdateProduct` | `PUT /products/:id` | `service.UpdateProduct(ctx, id, req)` |
| `DeleteProduct` | `DELETE /products/:id` | `service.DeleteProduct(ctx, id)` |
| `GetCategories` | `GET /categories` | `service.GetCategories(ctx)` |
| `CreateCategory` | `POST /categories` | `service.CreateCategory(ctx, req)` |

**`GetProducts` query param parsing:**

| Param | Type | Default | Validation |
|---|---|---|---|
| `q` | string | `""` | none |
| `category_id` | UUID string | nil | `uuid.Parse` → 400 on invalid |
| `min_price` | float64 | nil | `strconv.ParseFloat` → 400 on invalid |
| `max_price` | float64 | nil | same |
| `sort` | string | `"newest"` | validated in service layer |
| `page` | int | 1 | validated in service layer |
| `limit` | int | 20 | validated in service layer |

**`handleServiceError` mapping:**

| Domain Error | HTTP Status |
|---|---|
| `ErrProductNotFound`, `ErrCategoryNotFound` | 404 |
| `ErrCategorySlugExists`, `ErrProductAlreadyExists` | 409 |
| `ErrInvalidSortParam`, `ErrInvalidPageParam`, `ErrInvalidLimitParam`, `ErrPriceMustBePositive` | 400 |
| `ErrUnauthorized` | 401 |
| `ErrForbidden` | 403 |
| everything else | 500 |

### `internal/route/product_route.go`

```go
func RegisterProductRoutes(router *gin.Engine, h *handler.ProductHandler) {
    api := router.Group("/")
    api.GET("/products",       h.GetProducts)
    api.GET("/products/:id",   h.GetProductByID)
    api.POST("/products",      h.CreateProduct)
    api.PUT("/products/:id",   h.UpdateProduct)
    api.DELETE("/products/:id",h.DeleteProduct)
    api.GET("/categories",     h.GetCategories)
    api.POST("/categories",    h.CreateCategory)
}
```

Auth enforcement lives at the **gateway** (Task 2). The product service trusts `X-User-Role` injected by the gateway.

### `Dockerfile`

Multi-stage build identical to `services/user-service/Dockerfile`, changing:
- Binary output name: `/product-service`
- `EXPOSE 8082`
- `CMD ["./product-service"]`

### `.env` / `.env.example`

```env
PORT=8082
DATABASE_URL=postgres://auron:auron_pass@localhost:5433/products_db?sslmode=disable
REDIS_URL=redis://localhost:6380/0
KAFKA_BROKERS=localhost:9092
```

### `go.mod` — missing dependencies to add

```
github.com/gin-gonic/gin
gorm.io/driver/postgres
github.com/segmentio/kafka-go
```

Run `go mod tidy` after editing.

---

## Task 5 — Fix GenerateCacheKey Nil Panic

### Problem

`services/product-service/internal/cache/product_cache.go:117` unconditionally dereferences three optional pointer fields:

```go
// Current code — panics when any filter field is nil
hash := fmt.Sprintf("%s_%s_%s_%s_%d_%d",
    filter.Q,
    filter.CategoryID.String(),             // nil pointer panic
    fmt.Sprintf("%.2f", *filter.MinPrice),  // nil pointer panic
    fmt.Sprintf("%.2f", *filter.MaxPrice),  // nil pointer panic
    filter.Page,
    filter.Limit,
)
```

Every `GET /products` request without all three filters panics and crashes the service.

### Fix

Replace with nil-safe guards before formatting:

```go
func GenerateCacheKey(filter domain.ProductFilter) string {
    categoryID := ""
    if filter.CategoryID != nil {
        categoryID = filter.CategoryID.String()
    }

    minPrice := "0.00"
    if filter.MinPrice != nil {
        minPrice = fmt.Sprintf("%.2f", *filter.MinPrice)
    }

    maxPrice := "0.00"
    if filter.MaxPrice != nil {
        maxPrice = fmt.Sprintf("%.2f", *filter.MaxPrice)
    }

    key := fmt.Sprintf("%s_%s_%s_%s_%d_%d",
        filter.Q, categoryID, minPrice, maxPrice,
        filter.Page, filter.Limit,
    )
    return ProductListPrefix + ":" + key
}
```

The `InvalidateProductList` scan pattern `"product:list*"` still matches after adding the `:` separator — no other changes needed.

---

## Verification Checklist

### After Task 1
- [ ] `go build ./...` passes inside `services/api-gateway`
- [ ] `NewJWTMiddleware` no longer reads any file
- [ ] `JWT_SECRET` present in `api-gateway/.env` matching `user-service/.env`
- [ ] `docker-compose.yml` passes `JWT_SECRET` to `api-gateway`

### After Task 2
- [ ] `POST /api/products` without token → `401`
- [ ] `GET /api/products` without token → `200`
- [ ] `POST /api/payments/webhook/stripe` without token → `200` (not 401)
- [ ] `GET /api/users/me` without token → `401`
- [ ] Login via user-service → use returned token → `GET /api/users/me` → `200`

### After Task 3
- [ ] `go build ./...` passes inside `services/product-service`
- [ ] `ProductService` struct satisfies `domain.ProductService` interface (compiler verifies)
- [ ] Cache miss path calls repository
- [ ] Cache hit path does NOT call repository
- [ ] Write methods publish to Kafka; if Kafka is unavailable, request still succeeds

### After Task 4
- [ ] `go build ./...` passes inside `services/product-service`
- [ ] `docker build -t product-service .` succeeds
- [ ] `GET /health` → `200 {"status":"healthy"}`
- [ ] `GET /products` → `200` with paginated list
- [ ] `POST /products` with valid body → `201` with created product
- [ ] `GET /products?q=laptop` → FTS results
- [ ] `GET /products?sort=invalid` → `400`
- [ ] `GET /products/:nonexistent-id` → `404`

### After Task 5
- [ ] `GET /products` (no filters) does not panic
- [ ] `GET /products?category_id=<uuid>` does not panic
- [ ] `GET /products?min_price=10` without `max_price` does not panic
- [ ] Two requests with different filters produce different cache keys
- [ ] Two requests with identical filters produce the same cache key
