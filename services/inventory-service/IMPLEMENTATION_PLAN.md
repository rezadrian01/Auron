# Inventory Service — Implementation Plan

## Overview

The inventory service (port **8085**) manages stock levels for products.  
It is a **Kafka consumer + HTTP API** hybrid service:

- Exposes admin-only HTTP endpoints for viewing and manually setting stock
- Consumes `order.created` → reserves stock (`ReservedQuantity += n`)
- Consumes `order.cancelled` → releases reservation (`ReservedQuantity -= n`)
- Publishes `inventory.updated` on every stock change
- Publishes `inventory.low_stock` when available stock drops below threshold

### Gateway Routes (already wired, admin-only)

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET` | `/api/inventory/:product_id` | Admin | Get stock levels for a product |
| `PUT` | `/api/inventory/:product_id` | Admin | Set total stock (restocking) |

### Database

Inventory-service **shares `products_db`** with product-service. It does **not** create the `inventory` table — product-service's AutoMigrate owns it. Inventory-service connects to the same DB and operates directly on the `inventory` table.

The table structure (from product-service):
```
inventory
├── product_id        UUID  PRIMARY KEY
├── total_quantity    INT   NOT NULL DEFAULT 0
├── reserved_quantity INT   NOT NULL DEFAULT 0
├── version           INT   NOT NULL DEFAULT 0   ← optimistic locking
└── updated_at        TIMESTAMP NOT NULL DEFAULT NOW()
```

`AvailableQuantity = TotalQuantity - ReservedQuantity`

### Stock Reservation Flow

```
order-service → publishes order.created
        ↓
inventory-service consumes order.created
        ↓
Increments ReservedQuantity for each item (optimistic lock on version)
        ↓
Publishes inventory.updated (and inventory.low_stock if available stock < threshold)

If order is cancelled:
order-service → publishes order.cancelled
        ↓
inventory-service consumes order.cancelled
        ↓
Decrements ReservedQuantity for each item
        ↓
Publishes inventory.updated
```

> **Why not deduct TotalQuantity on order.created?**  
> Reserved stock stays visible to admins as "committed but not shipped." Total stock only changes when an admin performs a restock (`PUT /inventory/:product_id`). This gives accurate available-stock visibility without needing inter-service HTTP calls.

---

## Folder Structure

```
services/inventory-service/
├── cmd/
│   ├── config.go          # env vars → appConfig
│   ├── dotenv.go          # load .env in non-production
│   ├── infrastructure.go  # setupDatabase, setupRedis (no AutoMigrate)
│   ├── kafka.go           # setupKafkaPublisher, setupKafkaConsumer, startKafkaConsumer
│   ├── run.go             # wire everything together
│   └── server.go          # setupRouter, registerGracefulShutdown
├── db/
│   └── NOTE.md            # explains that product-service owns the inventory table
├── internal/
│   ├── cache/
│   │   └── inventory_cache.go
│   ├── domain/
│   │   ├── inventory.go   # Inventory entity, DTOs, event structs
│   │   ├── errors.go      # sentinel errors
│   │   ├── repository.go  # InventoryRepository interface
│   │   ├── service.go     # InventoryService interface
│   │   ├── cache.go       # InventoryCache interface
│   │   └── events.go      # EventPublisher + topic constants
│   ├── events/
│   │   ├── kafka_publisher.go
│   │   └── kafka_consumer.go  # multi-topic consumer (2 topics)
│   ├── handler/
│   │   └── inventory_handler.go
│   ├── repository/
│   │   └── inventory_repository.go
│   ├── route/
│   │   └── inventory_route.go
│   └── service/
│       └── inventory_service.go
├── main.go
├── Dockerfile
├── go.mod
├── .env
└── .env.example
```

> `internal/middleware/` exists in the scaffold but is not used — the gateway enforces admin auth via `X-User-Role` header before proxying.

---

## Tasks

### Task 1 — Domain Layer

**`internal/domain/inventory.go`**

```go
type Inventory struct {
    ProductID        uuid.UUID `json:"product_id" gorm:"type:uuid;primaryKey"`
    TotalQuantity    int       `json:"total_quantity" gorm:"not null;default:0"`
    ReservedQuantity int       `json:"reserved_quantity" gorm:"not null;default:0"`
    Version          int       `json:"version" gorm:"not null;default:0"`
    UpdatedAt        time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (Inventory) TableName() string { return "inventory" }

func (i *Inventory) AvailableQuantity() int {
    return i.TotalQuantity - i.ReservedQuantity
}
```

- `InventoryResponse` DTO — adds computed `available_quantity` field
- `UpdateInventoryRequest` — `{ total_quantity int, binding:"required,min=0" }`
- `LowStockThreshold = 10` — constant; triggers `inventory.low_stock` event when available stock drops at or below this
- `OrderCreatedEvent` / `OrderCancelledEvent` — shape of messages from order-service:
  ```go
  type OrderCreatedEvent struct {
      OrderID uuid.UUID        `json:"id"`     // matches Order.ID json tag
      UserID  uuid.UUID        `json:"user_id"`
      Items   []OrderEventItem `json:"items"`
  }
  type OrderEventItem struct {
      ProductID uuid.UUID `json:"product_id"`
      Quantity  int       `json:"quantity"`
  }
  ```
  `OrderCancelledEvent` is identical in shape — same `Order` struct published by order-service.

**`internal/domain/errors.go`**
- `ErrInventoryNotFound`, `ErrInsufficientStock`, `ErrInvalidQuantity`

**`internal/domain/repository.go`**
```go
type InventoryRepository interface {
    GetByProductID(productID uuid.UUID) (*Inventory, error)
    SetTotalQuantity(productID uuid.UUID, quantity int) (*Inventory, error)
    ReserveStock(productID uuid.UUID, quantity int) (*Inventory, error)
    ReleaseStock(productID uuid.UUID, quantity int) (*Inventory, error)
}
```

**`internal/domain/service.go`**
```go
type InventoryService interface {
    GetInventory(ctx context.Context, productID uuid.UUID) (*InventoryResponse, error)
    SetInventory(ctx context.Context, productID uuid.UUID, req UpdateInventoryRequest) (*InventoryResponse, error)
    HandleOrderCreated(ctx context.Context, event OrderCreatedEvent) error
    HandleOrderCancelled(ctx context.Context, event OrderCreatedEvent) error
}
```

**`internal/domain/cache.go`**
```go
type InventoryCache interface {
    GetInventory(ctx context.Context, productID uuid.UUID) (*Inventory, error)
    SetInventory(ctx context.Context, inv *Inventory) error
    InvalidateInventory(ctx context.Context, productID uuid.UUID) error
}
```

**`internal/domain/events.go`**
- `EventPublisher` interface: `Publish(ctx, topic string, payload any) error` + `Close() error`
- Consumed topics (constants, not published):
  - `TopicOrderCreated = "order.created"`
  - `TopicOrderCancelled = "order.cancelled"`
- Published topics:
  - `TopicInventoryUpdated = "inventory.updated"`
  - `TopicInventoryLowStock = "inventory.low_stock"`

---

### Task 2 — Repository Layer

**`internal/repository/inventory_repository.go`**

- `GetByProductID` — returns `ErrInventoryNotFound` on GORM `ErrRecordNotFound`
- `SetTotalQuantity` — admin restock: uses `db.Save()` with upsert semantics (creates if not exists, updates if exists); bumps `Version`
- `ReserveStock` — uses optimistic locking:
  ```go
  result := db.Model(&Inventory{}).
      Where("product_id = ? AND version = ? AND (total_quantity - reserved_quantity) >= ?",
            productID, current.Version, quantity).
      Updates(map[string]any{
          "reserved_quantity": gorm.Expr("reserved_quantity + ?", quantity),
          "version":           gorm.Expr("version + 1"),
          "updated_at":        time.Now(),
      })
  if result.RowsAffected == 0 {
      return nil, domain.ErrInsufficientStock
  }
  ```
  Returns `ErrInsufficientStock` if the WHERE clause misses (concurrent update or not enough stock).
- `ReleaseStock` — similar pattern; clamps `reserved_quantity` to minimum 0 via `GREATEST`:
  ```go
  Updates(map[string]any{
      "reserved_quantity": gorm.Expr("GREATEST(reserved_quantity - ?, 0)", quantity),
      "version":           gorm.Expr("version + 1"),
      "updated_at":        time.Now(),
  })
  ```

> **No AutoMigrate** — the `inventory` table is owned by product-service. Inventory-service connects to the same DB and reads/writes the table without managing its schema.

---

### Task 3 — Cache Layer

**`internal/cache/inventory_cache.go`**
- Key: `inventory:<product_id>` (TTL **5 minutes** — shorter than other caches since stock changes frequently)
- JSON marshal/unmarshal; Redis miss returns `nil, nil`

---

### Task 4 — Kafka Events (Publisher + Consumer)

**`internal/events/kafka_publisher.go`**
- Same pattern as other services: `kafkaPublisher` with `writers map[string]*kafka.Writer`

**`internal/events/kafka_consumer.go`**

Multi-topic consumer — subscribes to `order.created` AND `order.cancelled` with a single struct, two readers:

```go
type KafkaConsumer struct {
    readers []readerEntry
    service domain.InventoryService
}

type readerEntry struct {
    reader *kafka.Reader
    topic  string
}

func NewKafkaConsumer(brokers []string, service domain.InventoryService) *KafkaConsumer
```

- `Start(ctx)` — launches one goroutine per reader; each goroutine calls `handleMessage(topic, payload)`
- `handleMessage` — switches on topic, unmarshals the appropriate event type, calls the right service method
- `Close()` — closes all readers

Group IDs:
- `order.created` → group `inventory-service-orders`
- `order.cancelled` → group `inventory-service-orders` (same group, different topic)

---

### Task 5 — Service Layer

**`internal/service/inventory_service.go`**

**`GetInventory(ctx, productID)`**
1. Cache-aside: check `inventoryCache.GetInventory(ctx, productID)`
2. DB fallback on miss
3. Return `InventoryResponse` with computed `available_quantity`

**`SetInventory(ctx, productID, req)`**
1. Call `repo.SetTotalQuantity(productID, req.TotalQuantity)` (upsert)
2. Invalidate + re-cache
3. Publish `inventory.updated` async
4. Check if available stock ≤ `LowStockThreshold` → publish `inventory.low_stock` async

**`HandleOrderCreated(ctx, event)`**
- For each `event.Items`:
  1. Call `repo.ReserveStock(item.ProductID, item.Quantity)`
  2. On `ErrInsufficientStock`: log error, continue with remaining items (partial reservation is acceptable — could block order fulfillment in a real system, but acceptable for portfolio scope)
  3. Invalidate cache for affected product
  4. Publish `inventory.updated` async; if available stock ≤ threshold, also publish `inventory.low_stock`

**`HandleOrderCancelled(ctx, event)`**
- For each `event.Items`:
  1. Call `repo.ReleaseStock(item.ProductID, item.Quantity)`
  2. Invalidate cache
  3. Publish `inventory.updated` async

---

### Task 6 — Handler + Route Layers

**`internal/handler/inventory_handler.go`**
- `InventoryHandler` struct with `service domain.InventoryService`
- `GetInventory(c)` — parse `:product_id` UUID param, call service, return 200/404
- `SetInventory(c)` — parse `:product_id`, bind `UpdateInventoryRequest`, call service, return 200
- `handleError(c, err)` — maps `ErrInventoryNotFound` → 404, `ErrInvalidQuantity` → 400, default → 500
- No `getUserID` helper needed — admin identity is verified at the gateway; inventory-service trusts the request is admin

**`internal/route/inventory_route.go`**
```go
func RegisterInventoryRoutes(router *gin.Engine, inventoryHandler *handler.InventoryHandler) {
    api := router.Group("/")
    api.GET("/inventory/:product_id", inventoryHandler.GetInventory)
    api.PUT("/inventory/:product_id", inventoryHandler.SetInventory)
}
```

---

### Task 7 — cmd Bootstrap

**`cmd/config.go`**
```go
type appConfig struct {
    Port         string
    DatabaseURL  string
    RedisURL     string
    KafkaBrokers string
}
```
Defaults: port 8085, `localhost:5433/products_db`, `localhost:6379`, `localhost:9092`

> Note: default DATABASE_URL uses port 5433 (host-mapped port for products-db) for local dev.

**`cmd/dotenv.go`** — identical pattern to all other services

**`cmd/infrastructure.go`**
- `setupDatabase` with connection pooling
- **No `runMigrations`** — product-service owns the `inventory` table; inventory-service skips AutoMigrate
- `setupRedis` with 5s ping timeout
- `resolveGormLogLevel`

**`cmd/kafka.go`**
- `inventoryPublishedTopics`: `TopicInventoryUpdated`, `TopicInventoryLowStock`
- `setupKafkaPublisher(brokers string) domain.EventPublisher`
- `setupKafkaConsumer(brokers string, svc domain.InventoryService) *events.KafkaConsumer`
  - ensures `order.created` and `order.cancelled` topics exist (non-fatal if they already exist)
- `startKafkaConsumer(ctx, consumer)`
- `closeKafkaPublisher`, `parseBrokers`, `ensureTopics` — same helpers as other services

**`cmd/run.go`**
```go
func Run() {
    cfg := loadConfig()
    db := setupDatabase(cfg.DatabaseURL)
    redisClient := setupRedis(cfg.RedisURL)
    publisher := setupKafkaPublisher(cfg.KafkaBrokers)
    inventoryRepo := repository.NewInventoryRepository(db)
    inventoryCache := cache.NewInventoryCache(redisClient)
    inventorySvc := service.NewInventoryService(inventoryRepo, inventoryCache, publisher)
    ctx, cancel := context.WithCancel(context.Background())
    consumer := setupKafkaConsumer(cfg.KafkaBrokers, inventorySvc)
    startKafkaConsumer(ctx, consumer)
    inventoryHandler := handler.NewInventoryHandler(inventorySvc)
    router := setupRouter(inventoryHandler)
    registerGracefulShutdown(db, redisClient, publisher, consumer, cancel)
    router.Run(fmt.Sprintf(":%s", cfg.Port))
}
```

**`cmd/server.go`**
- `setupRouter(*handler.InventoryHandler) *gin.Engine` — release mode, `/health`, `/metrics`, calls `RegisterInventoryRoutes`

---

### Task 8 — Entry Point, Dockerfile, Env Files, docker-compose

**`main.go`** — `cmd.Run()`

**`Dockerfile`** — same multi-stage pattern: `golang:1.25-alpine` builder → `alpine:3.18` runtime; binary `inventory-service`; EXPOSE 8085

**`.env`** — local dev values:
```
PORT=8085
DATABASE_URL=postgres://auron:auron_pass@localhost:5433/products_db?sslmode=disable
REDIS_URL=redis://localhost:6379/0
KAFKA_BROKERS=localhost:9092
GORM_LOG_LEVEL=warn
```

**`.env.example`** — same with documented placeholders

**`docker-compose.yml`** — update `inventory-service` block:
```yaml
environment:
  - PORT=8085
  - DATABASE_URL=postgres://auron:auron_pass@products-db:5432/products_db?sslmode=disable
  - REDIS_URL=redis://redis:6379/0
  - KAFKA_BROKERS=kafka:29092
depends_on:
  products-db:
    condition: service_healthy
  kafka:
    condition: service_healthy
```

---

## Key Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Shared database | inventory-service reads/writes `products_db` | Product and inventory are tightly coupled; avoids extra DB; consistent with docker-compose original design |
| No AutoMigrate | inventory-service skips it | product-service owns the `inventory` table schema; running AutoMigrate in both is safe but unnecessary |
| Reservation vs deduction | Reserve on `order.created`, release on `order.cancelled` | Available stock stays accurate; no HTTP call to order-service needed for `payment.completed` |
| Optimistic locking | `version` field incremented on every update | Prevents lost updates under concurrent order placement; `ErrInsufficientStock` returned if version mismatches |
| Multi-topic consumer | Single struct with two readers, one goroutine each | kafka-go Reader only supports one topic; two readers is the idiomatic approach |
| Low-stock threshold | Constant `LowStockThreshold = 10` | Simple; notification-service can consume `inventory.low_stock` to alert admins |
| go.mod module | `auron/inventory-service` | Matches pattern of all other services |

---

## Dependencies

```
github.com/gin-gonic/gin v1.12.0
github.com/google/uuid v1.6.0
github.com/redis/go-redis/v9 v9.19.0
github.com/segmentio/kafka-go v0.4.51
gorm.io/driver/postgres v1.6.0
gorm.io/gorm v1.31.1
```

No Stripe SDK needed. Simpler dependency set than payment-service.
