# Order Service — Implementation Plan

**Branch:** `feature/order-service`  
**Port:** 8083  
**Database:** `orders_db` (PostgreSQL, orders-db:5434)  
**Stack:** Go 1.25 · Gin · GORM · Redis · Kafka  

---

## Overview

The Order Service owns two resources:

| Resource | Responsibility |
|---|---|
| **Cart** | Per-user in-flight basket; items with product snapshots |
| **Order** | Confirmed purchase; immutable after creation |

Gateway already routes these endpoints to port 8083:

```
Cart:   GET  /api/cart
        POST /api/cart/items
        PUT  /api/cart/items/:id
        DELETE /api/cart/items/:id

Orders: GET  /api/orders
        POST /api/orders
        GET  /api/orders/:id
        PUT  /api/orders/:id/cancel
```

All routes require `Authorization: Bearer <token>`. The gateway injects `X-User-ID` and `X-User-Role` headers; the service trusts these.

---

## Domain Model

### Entities

```
Cart
  id          uuid PK
  user_id     uuid  UNIQUE (one cart per user)
  created_at  timestamp
  updated_at  timestamp

CartItem
  id            uuid PK
  cart_id       uuid FK → carts
  product_id    uuid
  product_name  varchar   (snapshot at time of add)
  price         float64   (snapshot at time of add)
  quantity      int
  created_at    timestamp
  updated_at    timestamp

Order
  id               uuid PK
  user_id          uuid
  status           varchar  (pending|confirmed|processing|shipped|delivered|cancelled)
  total_amount     float64
  shipping_name    varchar  (snapshot)
  shipping_address varchar  (snapshot)
  created_at       timestamp
  updated_at       timestamp

OrderItem
  id           uuid PK
  order_id     uuid FK → orders
  product_id   uuid
  product_name varchar   (snapshot)
  price        float64   (snapshot)
  quantity     int
  subtotal     float64   (price × quantity, stored for history)
  created_at   timestamp
```

Price and product name are **snapshotted** on cart-add and order-create so historical orders are never affected by product edits.

### Order Status Flow

```
pending → confirmed → processing → shipped → delivered
       ↘ cancelled (any stage before shipped)
```

---

## External Dependencies

The service calls **Product Service** (HTTP) to:
1. Validate a product exists and is active before adding to cart
2. Get the current price and name for the snapshot

This is modelled as a `ProductClient` interface in the domain layer so the concrete HTTP implementation stays outside the domain.

---

## Redis Cache Strategy

| Key pattern | TTL | Evicted when |
|---|---|---|
| `cart:<user_id>` | 24 h | item added/updated/removed, cart cleared |
| `order:<order_id>` | 1 h | order status changes |
| `orders:user:<user_id>:page:<n>:limit:<l>` | 5 min | new order created, order cancelled |

---

## Kafka Events

| Topic | Published when |
|---|---|
| `order.created` | Order confirmed from cart |
| `order.updated` | Status changes |
| `order.cancelled` | Order cancelled |

Inventory Service and Notification Service consume these topics.

---

## Implementation Tasks

### Task 1 — Domain Layer

**Files to create:**

- `internal/domain/cart.go` — Cart, CartItem entities + TableName
- `internal/domain/order.go` — Order, OrderItem, OrderStatus entities + TableName
- `internal/domain/errors.go` — sentinel errors (ErrCartNotFound, ErrCartItemNotFound, ErrOrderNotFound, ErrOrderNotCancellable, ErrProductNotFound, ErrProductInactive, ErrInsufficientStock, ErrCartEmpty, ErrUnauthorized, ErrForbidden)
- `internal/domain/repository.go` — CartRepository + OrderRepository interfaces
- `internal/domain/service.go` — CartService + OrderService interfaces
- `internal/domain/cache.go` — CartCache + OrderCache interfaces
- `internal/domain/events.go` — EventPublisher interface + topic constants
- `internal/domain/client.go` — ProductClient interface (`GetProduct(id uuid.UUID) (*ProductSnapshot, error)`)

Key interface signatures:

```go
// CartService
GetCart(ctx, userID uuid.UUID) (*Cart, error)
AddItem(ctx, userID uuid.UUID, req AddItemRequest) (*Cart, error)
UpdateItem(ctx, userID, itemID uuid.UUID, qty int) (*Cart, error)
RemoveItem(ctx, userID, itemID uuid.UUID) error

// OrderService
GetOrders(ctx, userID uuid.UUID, page, limit int) (*OrderListResponse, error)
CreateOrder(ctx, userID uuid.UUID, req CreateOrderRequest) (*Order, error)
GetOrderByID(ctx, userID, orderID uuid.UUID) (*Order, error)
CancelOrder(ctx, userID, orderID uuid.UUID) (*Order, error)
```

---

### Task 2 — Database Migrations

**Files to create:**

- `db/001_create_carts.up.sql` — `carts` + `cart_items` tables
- `db/002_create_orders.up.sql` — `orders` + `order_items` tables

GORM AutoMigrate will handle the actual schema apply at startup (same pattern as product-service). The SQL files serve as documentation / manual fallback.

---

### Task 3 — Repository Layer

**Files to create:**

- `internal/repository/cart_repository.go` — implements `domain.CartRepository`
  - `GetCartByUserID(userID)` — preloads CartItems
  - `GetCartItemByID(cartID, itemID)` — single item lookup
  - `UpsertCart(cart)` — create or save
  - `UpsertCartItem(item)` — create or save
  - `DeleteCartItem(cartID, itemID)` — hard delete
  - `ClearCart(cartID)` — delete all items (after order created)

- `internal/repository/order_repository.go` — implements `domain.OrderRepository`
  - `GetOrdersByUserID(userID, offset, limit)` — preloads OrderItems
  - `GetOrderByID(orderID)` — preloads OrderItems
  - `CreateOrder(order)` — creates order + items in a single transaction
  - `UpdateOrderStatus(orderID, status)` — targeted update

---

### Task 4 — Cache Layer

**Files to create:**

- `internal/cache/cart_cache.go`
  - `GetCart(ctx, userID) (*domain.Cart, error)`
  - `SetCart(ctx, cart) error`
  - `InvalidateCart(ctx, userID) error`

- `internal/cache/order_cache.go`
  - `GetOrder(ctx, orderID) (*domain.Order, error)`
  - `SetOrder(ctx, order) error`
  - `InvalidateOrder(ctx, orderID) error`
  - `GetOrderList(ctx, userID, page, limit) (*domain.OrderListResponse, error)`
  - `SetOrderList(ctx, userID, page, limit, resp) error`
  - `InvalidateOrderList(ctx, userID) error` — scans `orders:user:<userID>:*`

---

### Task 5 — Kafka Publisher

**File to create:**

- `internal/events/kafka_publisher.go` — implements `domain.EventPublisher`
  - One `kafka.Writer` per topic (same pattern as product-service)
  - JSON-serialises the payload, publishes with context + key = order ID

---

### Task 6 — Product HTTP Client

**File to create:**

- `internal/client/product_client.go` — implements `domain.ProductClient`
  - `GET {PRODUCT_SERVICE_URL}/products/{id}`
  - Returns `domain.ProductSnapshot{ID, Name, Price, IsActive}`
  - Returns `domain.ErrProductNotFound` on 404, `domain.ErrProductInactive` if `is_active == false`
  - 5-second timeout

---

### Task 7 — Service Layer

**Files to create:**

- `internal/service/cart_service.go` — implements `domain.CartService`
  - `AddItem`: validate product via ProductClient → snapshot price/name → upsert cart + item → invalidate cache
  - `UpdateItem`: validate item belongs to user's cart → update qty → invalidate cache
  - `RemoveItem`: validate ownership → delete item → invalidate cache
  - `GetCart`: cache-aside (cache → DB)

- `internal/service/order_service.go` — implements `domain.OrderService`
  - `CreateOrder`: load cart → validate not empty → build Order + OrderItems from cart snapshots → DB create in transaction → clear cart → cache order → invalidate order list → publish `order.created`
  - `CancelOrder`: validate order belongs to user + status allows cancellation → update status → cache → publish `order.cancelled`
  - `GetOrderByID`: cache-aside
  - `GetOrders`: cache-aside (list cache, short TTL)

---

### Task 8 — Handler + Route Layers

**Files to create:**

- `internal/handler/cart_handler.go`
  - Reads `X-User-ID` header (set by gateway) to identify the caller
  - `GetCart`, `AddItem`, `UpdateItem`, `RemoveItem`

- `internal/handler/order_handler.go`
  - `GetOrders`, `CreateOrder`, `GetOrderByID`, `CancelOrder`
  - Request body for CreateOrder: `{ shipping_name, shipping_address }`

- `internal/route/order_route.go`
  - Registers all 8 routes on the Gin engine

---

### Task 9 — cmd Bootstrap

**Files to create** (same structure as product-service):

- `cmd/config.go` — `appConfig` struct; loads PORT, DATABASE_URL, REDIS_URL, KAFKA_BROKERS, PRODUCT_SERVICE_URL from env
- `cmd/dotenv.go` — silent `.env` loader
- `cmd/infrastructure.go` — GORM setup + AutoMigrate (Cart, CartItem, Order, OrderItem) + Redis setup
- `cmd/kafka.go` — creates `kafka.Writer` per topic (`order.created`, `order.updated`, `order.cancelled`), `ensureTopics`, `closeKafkaPublisher`
- `cmd/server.go` — Gin engine, `/health`, `/metrics`, calls `RegisterOrderRoutes`
- `cmd/run.go` — wires full dependency graph (repo → cache → client → publisher → service → handler → routes), registers graceful shutdown (DB, Redis, Kafka)

---

### Task 10 — Entry Point + Dockerfile

**Files to create:**

- `main.go` — calls `cmd.Run()`
- `Dockerfile` — multi-stage build (golang:1.25-alpine builder → alpine:3.18 runtime), port 8083
- `.env` — local dev values
- `.env.example`

---

### Task 11 — docker-compose Update

Add missing env vars to the `order-service` block in `docker-compose.yml`:

```yaml
- REDIS_URL=redis://redis:6379/0
- KAFKA_BROKERS=kafka:29092
- PRODUCT_SERVICE_URL=http://product-service:8082
```

Also add `depends_on: redis` and `depends_on: kafka` conditions.

---

## File Map

```
services/order-service/
├── main.go
├── Dockerfile
├── .env
├── .env.example
├── go.mod
├── go.sum
├── cmd/
│   ├── config.go
│   ├── dotenv.go
│   ├── infrastructure.go
│   ├── kafka.go
│   ├── run.go
│   └── server.go
├── db/
│   ├── 001_create_carts.up.sql
│   └── 002_create_orders.up.sql
└── internal/
    ├── cache/
    │   ├── cart_cache.go
    │   └── order_cache.go
    ├── client/
    │   └── product_client.go
    ├── domain/
    │   ├── cart.go
    │   ├── order.go
    │   ├── errors.go
    │   ├── repository.go
    │   ├── service.go
    │   ├── cache.go
    │   ├── events.go
    │   └── client.go
    ├── events/
    │   └── kafka_publisher.go
    ├── handler/
    │   ├── cart_handler.go
    │   └── order_handler.go
    ├── repository/
    │   ├── cart_repository.go
    │   └── order_repository.go
    ├── route/
    │   └── order_route.go
    └── service/
        ├── cart_service.go
        └── order_service.go
```

---

## Key Decisions

| Decision | Rationale |
|---|---|
| Price snapshot on cart-add | Historical orders stay accurate when product prices change |
| ProductClient interface in domain | Keeps domain testable; HTTP impl detail lives in `internal/client` |
| Cart cleared after order creation | Cart is single-use; users start a new one after checkout |
| No cart service auth check | Gateway already enforces auth; service trusts `X-User-ID` header |
| Kafka publish is async goroutine | Kafka unavailability never blocks HTTP response (same pattern as product-service) |
| `float64` for price | Consistent with product-service; avoids genproto/decimal GORM incompatibility |
