# Auron — Comprehensive Technical Plan

> **Project:** Auron E-Commerce Platform  
> **Stack:** Go (Gin) · Next.js 14 · PostgreSQL · Redis · Apache Kafka · Docker Compose  
> **Pattern:** Microservices · Event-Driven · Saga · DB-per-Service · API Gateway

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [System Architecture](#2-system-architecture)
3. [Repository Structure](#3-repository-structure)
4. [Services Specification](#4-services-specification)
   - 4.1 [API Gateway](#41-api-gateway)
   - 4.2 [User Service](#42-user-service)
   - 4.3 [Product Service](#43-product-service)
   - 4.4 [Order Service](#44-order-service)
   - 4.5 [Payment Service](#45-payment-service)
   - 4.6 [Inventory Service](#46-inventory-service)
   - 4.7 [Notification Service](#47-notification-service)
   - 4.8 [Next.js Frontend](#48-nextjs-frontend)
5. [Database Design](#5-database-design)
6. [Kafka Event Architecture](#6-kafka-event-architecture)
7. [Caching Strategy (Redis)](#7-caching-strategy-redis)
8. [Authentication & Security](#8-authentication--security)
9. [Docker Compose Infrastructure](#9-docker-compose-infrastructure)
10. [API Reference](#10-api-reference)
11. [Design Patterns](#11-design-patterns)
12. [Development Phases](#12-development-phases)
13. [Testing Strategy](#13-testing-strategy)
14. [Observability & Monitoring](#14-observability--monitoring)
15. [Environment Variables](#15-environment-variables)
16. [Makefile Commands](#16-makefile-commands)

---

## 1. Project Overview

**Auron** is a production-grade e-commerce platform built with a microservices architecture. Each service is independently deployable, owns its own database, and communicates asynchronously via Kafka for non-blocking workflows.

### Core Capabilities

| Capability | Description |
|---|---|
| User management | Registration, login, JWT auth, profile management |
| Product catalog | CRUD, full-text search, categories, image handling |
| Shopping cart | Redis-backed cart with TTL, multi-item checkout |
| Order management | Full lifecycle management with Saga-based distributed transactions |
| Payment processing | Stripe integration with webhook confirmation |
| Inventory control | Real-time stock reservation with optimistic locking |
| Notifications | Event-driven emails and SMS via Kafka consumers |
| Observability | Prometheus metrics, Grafana dashboards, structured logging |

### Architectural Principles

- **Single Responsibility** — each service owns one bounded context
- **DB-per-Service** — no shared databases; services communicate via events or APIs
- **Event-Driven** — Kafka decouples services for async workflows
- **API Gateway** — single public entry point; all auth and routing centralized
- **Fail-Safe** — Saga pattern handles distributed transaction rollbacks gracefully

---

## 2. System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     CLIENT LAYER                        │
│         Next.js 14 (SSR)          Mobile / PWA          │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTPS
┌──────────────────────▼──────────────────────────────────┐
│                   API GATEWAY :8080                      │
│         JWT Auth · Rate Limit · Routing · CORS           │
└──┬──────────┬──────────┬──────────┬──────────┬──────────┘
   │          │          │          │          │
:8081      :8082      :8083      :8084      :8085      :8086
User      Product    Order     Payment  Inventory  Notification
Service   Service   Service   Service   Service    Service
   │          │          │          │          │          │
   │          │          └──────────┴──────────┴──────────┘
   │          │                         │
   │          │              ┌──────────▼──────────┐
   │          │              │    Apache Kafka      │
   │          │              │  order.created       │
   │          │              │  payment.processed   │
   │          │              │  inventory.updated   │
   │          │              │  notification.send   │
   │          │              └─────────────────────┘
   │          │
┌──▼──┐  ┌───▼────┐  ┌───────┐  ┌────────┐
│users│  │product │  │orders │  │payment │   ← PostgreSQL DBs
│ _db │  │s_db    │  │_db    │  │s_db    │
└─────┘  └────────┘  └───────┘  └────────┘

┌─────────────────────────────────────────┐
│              Redis :6379                 │
│  Sessions · Cart · Cache · Rate Limit    │
└─────────────────────────────────────────┘
```

---

## 3. Repository Structure

```
auron/
│
├── frontend/                        # Next.js 14 application
│   ├── app/
│   │   ├── (auth)/
│   │   │   ├── login/page.tsx
│   │   │   └── register/page.tsx
│   │   ├── products/
│   │   │   ├── page.tsx             # Listing (Server Component)
│   │   │   └── [id]/page.tsx        # Detail page
│   │   ├── cart/page.tsx
│   │   ├── checkout/page.tsx
│   │   ├── orders/
│   │   │   ├── page.tsx
│   │   │   └── [id]/page.tsx
│   │   ├── profile/page.tsx
│   │   ├── layout.tsx
│   │   └── page.tsx                 # Home / landing
│   ├── components/
│   │   ├── ui/                      # shadcn/ui primitives
│   │   ├── product/
│   │   ├── cart/
│   │   └── checkout/
│   ├── lib/
│   │   ├── api.ts                   # Axios instance + interceptors
│   │   ├── store/                   # Zustand stores
│   │   └── utils.ts
│   ├── public/
│   ├── .env.local
│   ├── next.config.ts
│   ├── tailwind.config.ts
│   └── Dockerfile
│
├── services/
│   │
│   ├── api-gateway/                 # :8080
│   │   ├── middleware/
│   │   │   ├── auth.go              # JWT validation
│   │   │   ├── ratelimit.go         # Redis-based rate limiter
│   │   │   ├── cors.go
│   │   │   └── requestid.go
│   │   ├── routes/
│   │   │   └── router.go
│   │   ├── proxy/
│   │   │   └── proxy.go             # Reverse proxy handlers
│   │   ├── config/config.go
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── Dockerfile
│   │
│   ├── user-service/                # :8081
│   │   ├── handlers/
│   │   │   ├── auth.go
│   │   │   └── user.go
│   │   ├── models/
│   │   │   ├── user.go
│   │   │   └── token.go
│   │   ├── repository/
│   │   │   └── user_repository.go
│   │   ├── service/
│   │   │   └── user_service.go
│   │   ├── kafka/
│   │   │   └── producer.go
│   │   ├── migrations/
│   │   │   ├── 001_create_users.up.sql
│   │   │   └── 001_create_users.down.sql
│   │   ├── config/config.go
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── Dockerfile
│   │
│   ├── product-service/             # :8082  (same structure as above)
│   ├── order-service/               # :8083
│   ├── payment-service/             # :8084
│   ├── inventory-service/           # :8085
│   └── notification-service/        # :8086
│
├── shared/                          # Shared Go packages (internal modules)
│   ├── kafka/
│   │   ├── producer.go
│   │   └── consumer.go
│   ├── redis/
│   │   └── client.go
│   ├── middleware/
│   │   └── recovery.go
│   ├── events/
│   │   └── types.go                 # Shared event structs
│   └── go.mod
│
├── infra/
│   ├── docker-compose.yml           # Full stack
│   ├── docker-compose.dev.yml       # Dev overrides (hot reload)
│   ├── kafka/
│   │   └── topics.sh                # Topic creation script
│   ├── postgres/
│   │   └── init.sh                  # Multi-DB init script
│   ├── prometheus/
│   │   └── prometheus.yml
│   └── grafana/
│       └── dashboards/
│
├── scripts/
│   └── seed.sh                      # Seed test data
│
├── .env.example
├── Makefile
└── README.md
```

---

## 4. Services Specification

### 4.1 API Gateway

**Port:** `8080`  
**Tech:** Go, Gin, Redis  
**Role:** Single public entry point. Handles cross-cutting concerns so individual services don't have to.

#### Responsibilities

- JWT token validation on every protected route
- Rate limiting: 100 requests/minute per IP via Redis `INCR` + `EXPIRE`
- Reverse proxy routing to downstream microservices
- CORS headers and request ID injection
- Prometheus metrics endpoint at `/metrics`

#### Routing Table

| Prefix | Downstream Service | Auth Required |
|---|---|---|
| `POST /api/auth/*` | User Service :8081 | No |
| `GET /api/users/me` | User Service :8081 | Yes |
| `GET /api/products/*` | Product Service :8082 | No |
| `POST /api/products/*` | Product Service :8082 | Yes (admin) |
| `* /api/orders/*` | Order Service :8083 | Yes |
| `POST /api/payments/*` | Payment Service :8084 | Yes |
| `GET /api/inventory/*` | Inventory Service :8085 | Yes (admin) |

#### Middleware Chain

```
Request → RequestID → CORS → RateLimit → JWT → Proxy → Response
```

#### Rate Limit Implementation

```go
// Redis key: ratelimit:{ip}:{minute_window}
key := fmt.Sprintf("ratelimit:%s:%d", ip, time.Now().Unix()/60)
count, _ := redis.Incr(ctx, key)
redis.Expire(ctx, key, 2*time.Minute)
if count > 100 {
    c.AbortWithStatusJSON(429, gin.H{"error": "rate limit exceeded"})
}
```

---

### 4.2 User Service

**Port:** `8081`  
**Database:** `users_db` (PostgreSQL :5432)  
**Tech:** Go, Gin, GORM, golang-jwt, bcrypt

#### Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| `POST` | `/register` | Create account | No |
| `POST` | `/login` | Get access + refresh tokens | No |
| `POST` | `/refresh` | Rotate refresh token | No |
| `POST` | `/logout` | Revoke refresh token | Yes |
| `GET` | `/me` | Get current user profile | Yes |
| `PUT` | `/me` | Update profile | Yes |
| `POST` | `/me/address` | Add shipping address | Yes |

#### Token Strategy

| Token | TTL | Storage |
|---|---|---|
| Access Token | 15 minutes | Client memory / httpOnly cookie |
| Refresh Token | 7 days | Redis (`refresh:{hash}` → user_id) |

**Flow:**
1. Login → issue access token + refresh token
2. Access token expires → client calls `POST /refresh` with refresh token
3. Server validates refresh token in Redis → issue new pair → delete old refresh token (rotation)
4. Logout → delete refresh token from Redis

#### Kafka Events Published

```json
{
  "topic": "user.registered",
  "payload": {
    "user_id": "uuid",
    "email": "user@example.com",
    "name": "John Doe",
    "timestamp": "2025-01-01T00:00:00Z"
  }
}
```

---

### 4.3 Product Service

**Port:** `8082`  
**Database:** `products_db` (PostgreSQL :5433)  
**Tech:** Go, Gin, GORM, Redis

#### Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| `GET` | `/products` | List products (paginated, filtered) | No |
| `GET` | `/products/:id` | Get product detail | No |
| `POST` | `/products` | Create product | Admin |
| `PUT` | `/products/:id` | Update product | Admin |
| `DELETE` | `/products/:id` | Delete product | Admin |
| `GET` | `/categories` | List categories | No |
| `POST` | `/categories` | Create category | Admin |

#### Query Parameters for `GET /products`

| Param | Type | Example | Description |
|---|---|---|---|
| `q` | string | `laptop` | Full-text search |
| `category_id` | uuid | `abc-123` | Filter by category |
| `min_price` | float | `99.99` | Minimum price |
| `max_price` | float | `999.99` | Maximum price |
| `sort` | string | `price_asc` | Sort order |
| `page` | int | `1` | Page number |
| `limit` | int | `20` | Items per page |

#### Caching Strategy

```
GET /products  →  Check Redis key "products:list:{hash_of_params}"
  Hit  → Return cached JSON (TTL: 5 min)
  Miss → Query PostgreSQL → Cache result → Return

PUT/DELETE /products/:id → Invalidate "products:list:*" + "products:{id}"
```

#### Full-Text Search

```sql
-- PostgreSQL tsvector index
ALTER TABLE products ADD COLUMN search_vector tsvector;
CREATE INDEX products_search_idx ON products USING GIN(search_vector);
UPDATE products SET search_vector = to_tsvector('english', name || ' ' || description);

-- Query
SELECT * FROM products WHERE search_vector @@ plainto_tsquery('english', $1);
```

---

### 4.4 Order Service

**Port:** `8083`  
**Database:** `orders_db` (PostgreSQL :5434)  
**Tech:** Go, Gin, GORM, Kafka (producer + consumer), Redis

#### Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| `GET` | `/cart` | Get user's cart | Yes |
| `POST` | `/cart/items` | Add item to cart | Yes |
| `PUT` | `/cart/items/:id` | Update cart item quantity | Yes |
| `DELETE` | `/cart/items/:id` | Remove item from cart | Yes |
| `POST` | `/orders` | Create order (checkout) | Yes |
| `GET` | `/orders` | List user's orders | Yes |
| `GET` | `/orders/:id` | Get order detail | Yes |
| `PUT` | `/orders/:id/cancel` | Cancel order | Yes |

#### Cart Implementation (Redis)

```
Redis key: cart:{user_id}        (Hash)
Fields:    {product_id} → JSON{product_id, name, price, quantity, image}
TTL:       24 hours (reset on every cart modification)
```

#### Saga Pattern — Order Checkout Flow

The order service acts as the Saga Orchestrator. When a user checks out:

```
1. Order Service creates order (status: PENDING) → saves to orders_db
2. Publishes  →  order.created  →  Kafka
3.   Payment Service consumes  →  charges Stripe  →  publishes payment.processed | payment.failed
4.   Inventory Service consumes →  reserves stock  →  publishes inventory.updated | inventory.failed
5. Order Service consumes payment.processed + inventory.updated  →  status: CONFIRMED
6. On any failure  →  compensating transactions:
     - payment.failed    →  release inventory reservation
     - inventory.failed  →  issue Stripe refund
7. Order status: FAILED, user notified
```

#### Order Status State Machine

```
PENDING → CONFIRMED → PROCESSING → SHIPPED → DELIVERED
    ↓           ↓
  FAILED    CANCELLED
```

---

### 4.5 Payment Service

**Port:** `8084`  
**Database:** `payments_db` (PostgreSQL :5435)  
**Tech:** Go, Gin, GORM, Kafka, Stripe Go SDK

#### Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| `GET` | `/payments/:id` | Get payment detail | Yes |
| `POST` | `/webhooks/stripe` | Stripe webhook handler | No (signature verified) |

#### Kafka Events

| Consumes | Produces |
|---|---|
| `order.created` | `payment.processed` |
| | `payment.failed` |

#### Stripe Integration Flow

```
1. Consume order.created event
2. Create Stripe PaymentIntent
   - amount = order.total_cents
   - idempotency_key = order.id  (prevents double charge)
   - metadata = {order_id, user_id}
3. Store payment record (status: PENDING)
4. Stripe webhook fires: payment_intent.succeeded
5. Update payment (status: COMPLETED)
6. Publish payment.processed event
```

#### Idempotency

Every payment creation uses `order_id` as the Stripe idempotency key. If the service crashes and retries, Stripe returns the existing PaymentIntent without creating a duplicate charge.

---

### 4.6 Inventory Service

**Port:** `8085`  
**Database:** `products_db` (PostgreSQL :5433, shared read with Product Service)  
**Tech:** Go, Gin, GORM, Kafka

> **Note:** Inventory shares the `products_db` PostgreSQL instance but uses its own schema tables (`inventory`, `inventory_reservations`). This is a pragmatic choice for a v1 system; it can be split later.

#### Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| `GET` | `/inventory/:product_id` | Get stock level | Admin |
| `PUT` | `/inventory/:product_id` | Manually adjust stock | Admin |

#### Kafka Events

| Consumes | Produces |
|---|---|
| `order.created` | `inventory.updated` |
| `payment.failed` | `inventory.failed` |
| `order.cancelled` | |

#### Stock Reservation with Optimistic Locking

```sql
-- Atomic reservation prevents overselling
UPDATE inventory
SET reserved_quantity = reserved_quantity + $1,
    version = version + 1
WHERE product_id = $2
  AND version = $3                          -- optimistic lock check
  AND (available_quantity - reserved_quantity) >= $1;  -- enough stock
```

If the update affects 0 rows, the reservation fails and `inventory.failed` is published, triggering saga rollback.

---

### 4.7 Notification Service

**Port:** `8086` (internal only, no public routes)  
**Tech:** Go, Kafka (consumer only), SendGrid, Twilio, `html/template`

This service is a **pure Kafka consumer**. It has no HTTP API — it only reacts to events.

#### Event → Notification Mapping

| Event Consumed | Notification Sent | Channel |
|---|---|---|
| `user.registered` | Welcome email | Email |
| `order.created` | Order confirmation | Email |
| `payment.processed` | Payment receipt | Email + SMS |
| `payment.failed` | Payment failure alert | Email |
| `order.shipped` | Shipping update with tracking | Email + SMS |
| `order.delivered` | Delivery confirmation | Email |

#### Dead-Letter Queue

If a notification fails (e.g., SendGrid is down), the message is published to `notification.dlq` and retried with exponential backoff up to 5 attempts.

---

### 4.8 Next.js Frontend

**Port:** `3000`  
**Tech:** Next.js 14 (App Router), TypeScript, TailwindCSS, shadcn/ui, Zustand, Axios

#### Page Structure

| Route | Type | Description |
|---|---|---|
| `/` | Server Component | Landing page, featured products |
| `/products` | Server Component | Product listing with filters (SEO-friendly) |
| `/products/[id]` | Server Component | Product detail page |
| `/cart` | Client Component | Cart management |
| `/checkout` | Client Component | Checkout + Stripe Elements |
| `/orders` | Server Component | Order history |
| `/orders/[id]` | Server Component | Order detail + status |
| `/profile` | Client Component | User profile and addresses |
| `/(auth)/login` | Client Component | Login form |
| `/(auth)/register` | Client Component | Registration form |

#### Axios Instance with JWT Interceptor

```typescript
// lib/api.ts
const api = axios.create({ baseURL: process.env.NEXT_PUBLIC_API_URL });

// Attach access token to every request
api.interceptors.request.use(config => {
  const token = useAuthStore.getState().accessToken;
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// Auto-refresh on 401
api.interceptors.response.use(null, async error => {
  if (error.response?.status === 401 && !error.config._retry) {
    error.config._retry = true;
    const newToken = await refreshAccessToken();
    useAuthStore.getState().setAccessToken(newToken);
    error.config.headers.Authorization = `Bearer ${newToken}`;
    return api(error.config);
  }
  return Promise.reject(error);
});
```

#### Zustand Cart Store

```typescript
// lib/store/cartStore.ts
interface CartStore {
  items: CartItem[];
  addItem: (product: Product) => void;
  removeItem: (productId: string) => void;
  updateQuantity: (productId: string, qty: number) => void;
  clearCart: () => void;
  total: () => number;
}
```

---

## 5. Database Design

### users_db

```sql
-- users table
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) UNIQUE NOT NULL,
    password    VARCHAR(255) NOT NULL,          -- bcrypt hash
    name        VARCHAR(255) NOT NULL,
    role        VARCHAR(50) DEFAULT 'customer', -- customer | admin
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

-- addresses table
CREATE TABLE addresses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    label       VARCHAR(100),                   -- home, work, etc.
    street      TEXT NOT NULL,
    city        VARCHAR(100) NOT NULL,
    state       VARCHAR(100),
    country     VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20),
    is_default  BOOLEAN DEFAULT false,
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_addresses_user_id ON addresses(user_id);
```

### products_db

```sql
-- categories table
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) UNIQUE NOT NULL,
    parent_id   UUID REFERENCES categories(id),
    created_at  TIMESTAMP DEFAULT NOW()
);

-- products table
CREATE TABLE products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id     UUID REFERENCES categories(id),
    name            VARCHAR(500) NOT NULL,
    description     TEXT,
    price           DECIMAL(12, 2) NOT NULL,
    image_url       TEXT,
    search_vector   TSVECTOR,
    is_active       BOOLEAN DEFAULT true,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- inventory table
CREATE TABLE inventory (
    product_id          UUID PRIMARY KEY REFERENCES products(id),
    total_quantity      INTEGER NOT NULL DEFAULT 0,
    reserved_quantity   INTEGER NOT NULL DEFAULT 0,
    version             INTEGER NOT NULL DEFAULT 0,    -- for optimistic locking
    updated_at          TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_search ON products USING GIN(search_vector);
CREATE INDEX idx_products_price ON products(price);
```

### orders_db

```sql
-- orders table
CREATE TABLE orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    total_amount    DECIMAL(12, 2) NOT NULL,
    shipping_address JSONB NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- order_items table
CREATE TABLE order_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID REFERENCES orders(id) ON DELETE CASCADE,
    product_id  UUID NOT NULL,
    product_name VARCHAR(500) NOT NULL,    -- snapshot at time of order
    price       DECIMAL(12, 2) NOT NULL,  -- snapshot
    quantity    INTEGER NOT NULL,
    subtotal    DECIMAL(12, 2) NOT NULL
);

-- order_events table  (Saga state log)
CREATE TABLE order_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID REFERENCES orders(id),
    event_type  VARCHAR(100) NOT NULL,
    payload     JSONB,
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
```

### payments_db

```sql
-- payments table
CREATE TABLE payments (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id                UUID UNIQUE NOT NULL,
    user_id                 UUID NOT NULL,
    stripe_payment_intent_id VARCHAR(255) UNIQUE,
    amount                  DECIMAL(12, 2) NOT NULL,
    currency                VARCHAR(10) DEFAULT 'usd',
    status                  VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    stripe_event_ids        TEXT[],                -- for webhook idempotency
    created_at              TIMESTAMP DEFAULT NOW(),
    updated_at              TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_stripe_pi ON payments(stripe_payment_intent_id);
```

---

## 6. Kafka Event Architecture

### Topics & Partitioning

| Topic | Partitions | Retention | Key |
|---|---|---|---|
| `user.registered` | 3 | 7 days | `user_id` |
| `order.created` | 6 | 7 days | `order_id` |
| `payment.processed` | 6 | 7 days | `order_id` |
| `payment.failed` | 6 | 7 days | `order_id` |
| `inventory.updated` | 6 | 7 days | `product_id` |
| `inventory.failed` | 3 | 7 days | `order_id` |
| `notification.dlq` | 3 | 30 days | — |

### Consumer Groups

| Group ID | Topics | Service |
|---|---|---|
| `payment-service` | `order.created` | Payment Service |
| `inventory-service` | `order.created`, `payment.failed`, `order.cancelled` | Inventory Service |
| `order-service` | `payment.processed`, `payment.failed`, `inventory.updated` | Order Service |
| `notification-service` | All topics | Notification Service |

### Event Schemas

**order.created**
```json
{
  "event_id": "uuid",
  "event_type": "order.created",
  "timestamp": "2025-01-01T00:00:00Z",
  "payload": {
    "order_id": "uuid",
    "user_id": "uuid",
    "user_email": "user@example.com",
    "items": [
      { "product_id": "uuid", "quantity": 2, "price": 49.99 }
    ],
    "total_amount": 99.98,
    "shipping_address": { "street": "...", "city": "...", "country": "..." }
  }
}
```

**payment.processed**
```json
{
  "event_id": "uuid",
  "event_type": "payment.processed",
  "timestamp": "2025-01-01T00:00:00Z",
  "payload": {
    "order_id": "uuid",
    "payment_id": "uuid",
    "amount": 99.98,
    "stripe_payment_intent_id": "pi_xxx"
  }
}
```

---

## 7. Caching Strategy (Redis)

### Key Naming Convention

```
{entity}:{identifier}:{sub_key}
```

### Cache Map

| Key Pattern | Type | TTL | Populated By | Invalidated By |
|---|---|---|---|---|
| `session:{user_id}` | Hash | 7 days | User Service (login) | Logout, token revocation |
| `refresh:{token_hash}` | String | 7 days | User Service (login) | Logout, refresh rotation |
| `ratelimit:{ip}:{window}` | String | 2 min | API Gateway | Automatic (TTL) |
| `product:{id}` | String (JSON) | 5 min | Product Service | Product update/delete |
| `products:list:{params_hash}` | String (JSON) | 5 min | Product Service | Any product change |
| `cart:{user_id}` | Hash | 24 hours | Order Service | Checkout, manual clear |
| `inventory:{product_id}` | String | 1 min | Inventory Service | Stock update |

### Cart Structure in Redis

```
HSET cart:{user_id} {product_id} '{"product_id":"...","name":"...","price":49.99,"quantity":2,"image":"..."}'
EXPIRE cart:{user_id} 86400
```

---

## 8. Authentication & Security

### JWT Structure

```json
// Access Token Payload
{
  "sub": "user_id",
  "email": "user@example.com",
  "role": "customer",
  "iat": 1700000000,
  "exp": 1700000900      // +15 minutes
}
```

### Security Checklist

| Concern | Implementation |
|---|---|
| Password hashing | bcrypt with cost factor 12 |
| Token signing | RS256 (asymmetric) — private key signs, public key verifies in gateway |
| Token refresh | Refresh token rotation — old token deleted on use |
| Rate limiting | 100 req/min per IP via Redis, enforced at gateway |
| CORS | Whitelist frontend origin only |
| SQL injection | GORM parameterized queries only; never raw interpolation |
| Stripe webhooks | Signature verification via `stripe.ConstructEvent` |
| Input validation | `go-playground/validator` on all request bodies |
| HTTPS | Enforced in production via reverse proxy (Nginx/Caddy) |
| Secrets | Environment variables only; never hardcoded |

---

## 9. Docker Compose Infrastructure

### Services Overview

```yaml
# infra/docker-compose.yml

services:
  # ── Frontend ────────────────────────────────
  frontend:          # Next.js :3000

  # ── API Gateway ─────────────────────────────
  api-gateway:       # Go/Gin :8080

  # ── Microservices ───────────────────────────
  user-service:      # :8081
  product-service:   # :8082
  order-service:     # :8083
  payment-service:   # :8084
  inventory-service: # :8085
  notification-service: # :8086

  # ── Databases ───────────────────────────────
  users-db:          # PostgreSQL :5432
  products-db:       # PostgreSQL :5433
  orders-db:         # PostgreSQL :5434
  payments-db:       # PostgreSQL :5435

  # ── Message Broker ──────────────────────────
  zookeeper:         # :2181
  kafka:             # :9092

  # ── Cache ───────────────────────────────────
  redis:             # :6379

  # ── Observability ───────────────────────────
  prometheus:        # :9090
  grafana:           # :3001

networks:
  ecommerce-net:
    driver: bridge

volumes:
  users-db-data:
  products-db-data:
  orders-db-data:
  payments-db-data:
  redis-data:
  grafana-data:
```

### Health Check Pattern (per service)

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:808X/health"]
  interval: 10s
  timeout: 5s
  retries: 5
  start_period: 30s
```

### Service Startup Dependencies

```
zookeeper → kafka → [all Go services]
[postgres DBs] → [respective Go services]
redis → api-gateway, user-service, product-service, order-service
api-gateway → (started after all services are healthy)
```

---

## 10. API Reference

### Base URL
```
http://localhost:8080/api
```

### Auth Endpoints

```
POST   /auth/register        Body: {name, email, password}
POST   /auth/login           Body: {email, password}  →  {access_token, refresh_token}
POST   /auth/refresh         Body: {refresh_token}    →  {access_token, refresh_token}
POST   /auth/logout          Header: Authorization: Bearer <token>
```

### User Endpoints

```
GET    /users/me             Get profile
PUT    /users/me             Body: {name}
POST   /users/me/addresses   Body: {label, street, city, state, country, postal_code}
GET    /users/me/addresses   List addresses
DELETE /users/me/addresses/:id
```

### Product Endpoints

```
GET    /products             ?q=&category_id=&min_price=&max_price=&sort=&page=&limit=
GET    /products/:id
POST   /products             Body: {name, description, price, category_id, image_url}
PUT    /products/:id
DELETE /products/:id
GET    /categories
POST   /categories           Body: {name, slug, parent_id?}
```

### Order & Cart Endpoints

```
GET    /cart
POST   /cart/items           Body: {product_id, quantity}
PUT    /cart/items/:id       Body: {quantity}
DELETE /cart/items/:id
POST   /orders               Body: {address_id, items: [{product_id, quantity}]}
GET    /orders               ?page=&limit=&status=
GET    /orders/:id
PUT    /orders/:id/cancel
```

### Payment Endpoints

```
GET    /payments/:id
POST   /payments/webhook/stripe   (Stripe webhook — no auth, signature verified)
```

### Response Format

**Success**
```json
{
  "success": true,
  "data": { },
  "meta": { "page": 1, "limit": 20, "total": 100 }
}
```

**Error**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "email is required",
    "details": [ ]
  }
}
```

---

## 11. Design Patterns

### Saga Pattern (Order Service as Orchestrator)

```
Order Service (Orchestrator)
    │
    ├─► Publish: order.created
    │       │
    │       ├─► Payment Service consumes → charges Stripe
    │       │       ├─► Success: publish payment.processed
    │       │       └─► Failure: publish payment.failed
    │       │
    │       └─► Inventory Service consumes → reserves stock
    │               ├─► Success: publish inventory.updated
    │               └─► Failure: publish inventory.failed
    │
    ├─► On payment.processed + inventory.updated → CONFIRMED
    │
    └─► On payment.failed → cancel reservation → FAILED
        On inventory.failed → issue refund → FAILED
```

### Repository Pattern (per service)

```go
// repository/product_repository.go
type ProductRepository interface {
    FindByID(ctx context.Context, id string) (*models.Product, error)
    FindAll(ctx context.Context, filter ProductFilter) ([]models.Product, int64, error)
    Create(ctx context.Context, product *models.Product) error
    Update(ctx context.Context, product *models.Product) error
    Delete(ctx context.Context, id string) error
}
```

### Dependency Injection (per service main.go)

```go
// main.go
db := database.Connect(cfg.DatabaseURL)
redisClient := redis.NewClient(cfg.RedisURL)
kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers)

repo := repository.NewProductRepository(db)
cache := cache.NewProductCache(redisClient)
svc := service.NewProductService(repo, cache, kafkaProducer)
handler := handlers.NewProductHandler(svc)

router := gin.Default()
handler.RegisterRoutes(router)
router.Run(cfg.Port)
```

---

## 12. Development Phases

### Phase 1 — Foundation (Week 1–2)

- [ ] Initialize Go module per service
- [ ] Set up Docker Compose with all infrastructure
- [ ] PostgreSQL instances + migration tooling (`golang-migrate`)
- [ ] Redis client wrapper in `shared/`
- [ ] Kafka producer/consumer helpers in `shared/`
- [ ] Create Kafka topics script
- [ ] Base Gin server with health check, recovery, request-id middleware
- [ ] Shared event type definitions

### Phase 2 — Auth & Gateway (Week 2–3)

- [ ] User Service: register, login, refresh, logout
- [ ] JWT RS256 key pair generation
- [ ] API Gateway: JWT middleware, rate limiter, reverse proxy
- [ ] Next.js: login + register pages, Axios instance, Zustand auth store
- [ ] Token refresh interceptor in frontend
- [ ] Integration test: register → login → access protected route

### Phase 3 — Product Catalog (Week 3–4)

- [ ] Product Service: CRUD, categories
- [ ] PostgreSQL full-text search (`tsvector`)
- [ ] Redis caching layer for product listings
- [ ] Cache invalidation on mutations
- [ ] Next.js: product listing page (Server Component), product detail page
- [ ] Filtering and pagination UI

### Phase 4 — Cart & Orders (Week 4–5)

- [ ] Redis cart implementation in Order Service
- [ ] Order creation endpoint with snapshot pricing
- [ ] Inventory Service: stock management
- [ ] Kafka producers in Order Service (`order.created`)
- [ ] Kafka consumers in Inventory Service
- [ ] Saga orchestration logic
- [ ] Next.js: cart page, checkout form, order history

### Phase 5 — Payments & Notifications (Week 5–6)

- [ ] Payment Service: Stripe PaymentIntent flow
- [ ] Stripe webhook endpoint + signature verification
- [ ] Notification Service: Kafka consumer setup
- [ ] SendGrid email templates (order confirmation, receipt)
- [ ] Twilio SMS integration
- [ ] Dead-letter queue for failed notifications
- [ ] Full Saga end-to-end test (order → payment → inventory → notification)

### Phase 6 — Observability & Polish (Week 7–8)

- [ ] Prometheus metrics per service (`/metrics`)
- [ ] Grafana dashboards: request rate, error rate, latency, Kafka consumer lag
- [ ] Structured JSON logging (`zerolog` or `zap`)
- [ ] Centralized log aggregation (optional: ELK stack)
- [ ] Unit tests: service layer per microservice
- [ ] Integration tests: full order flow
- [ ] Load testing with `k6`
- [ ] GitHub Actions CI/CD pipeline

---

## 13. Testing Strategy

### Unit Tests (per service)

Focus on the **service layer** — business logic with mocked repositories and Kafka producers.

```
user-service/service/user_service_test.go
product-service/service/product_service_test.go
order-service/service/order_service_test.go
```

Tools: `testing` (stdlib), `testify`, `mockery` for interface mocks.

### Integration Tests

Test the full request path including real PostgreSQL and Redis:

```
POST /auth/register → users_db has new user → Redis has refresh token
GET /products → Redis cache miss → PostgreSQL query → Redis populated
POST /orders → order in DB → Kafka message published
```

Tools: `testcontainers-go` to spin up real PostgreSQL/Redis in tests.

### End-to-End Tests

Full saga flow verification:

```
1. Register user
2. Create products + set inventory
3. Add to cart
4. Checkout → order.created published
5. Payment processed → payment.processed published
6. Inventory reserved → inventory.updated published
7. Order status = CONFIRMED
8. Notification email sent (stub SendGrid)
```

### Load Testing

Tool: `k6`

```javascript
// scripts/load-test.js
export const options = {
  vus: 100,
  duration: '60s',
};

export default function () {
  http.get('http://localhost:8080/api/products');
  sleep(1);
}
```

Target: **500 concurrent users**, p99 latency < **200ms** for GET /products (cache hit).

---

## 14. Observability & Monitoring

### Prometheus Metrics (per service)

Each Go service exposes:

```go
// HTTP request metrics (auto via gin-prometheus)
http_requests_total{method, path, status}
http_request_duration_seconds{method, path}

// Custom business metrics
orders_created_total
payments_processed_total
kafka_messages_published_total{topic}
kafka_messages_consumed_total{topic, group}
cache_hits_total / cache_misses_total
```

### Grafana Dashboards

| Dashboard | Key Panels |
|---|---|
| System Overview | Request rate, error rate, p50/p99 latency per service |
| Kafka | Consumer lag per group, message throughput, offset lag |
| Database | Query duration, connection pool usage, slow queries |
| Business | Orders per minute, payment success rate, GMV |
| Redis | Hit rate, memory usage, evictions |

### Structured Logging

Every log line includes:

```json
{
  "timestamp": "2025-01-01T00:00:00Z",
  "level": "INFO",
  "service": "order-service",
  "request_id": "uuid",
  "user_id": "uuid",
  "message": "order created",
  "order_id": "uuid",
  "duration_ms": 12
}
```

---

## 15. Environment Variables

### Global (`.env` at root — loaded by Docker Compose)

```bash
# JWT
JWT_PRIVATE_KEY=...
JWT_PUBLIC_KEY=...
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

# Kafka
KAFKA_BROKERS=kafka:9092

# Redis
REDIS_URL=redis://redis:6379/0

# Stripe
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# SendGrid
SENDGRID_API_KEY=SG....
FROM_EMAIL=noreply@auron.shop

# Twilio
TWILIO_ACCOUNT_SID=AC...
TWILIO_AUTH_TOKEN=...
TWILIO_FROM_NUMBER=+1...

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080/api
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...
```

### Per Service

```bash
# api-gateway
PORT=8080
USER_SERVICE_URL=http://user-service:8081
PRODUCT_SERVICE_URL=http://product-service:8082
ORDER_SERVICE_URL=http://order-service:8083
PAYMENT_SERVICE_URL=http://payment-service:8084

# user-service
PORT=8081
DATABASE_URL=postgres://user:pass@users-db:5432/users_db?sslmode=disable

# product-service
PORT=8082
DATABASE_URL=postgres://user:pass@products-db:5433/products_db?sslmode=disable
CACHE_TTL=5m

# order-service
PORT=8083
DATABASE_URL=postgres://user:pass@orders-db:5434/orders_db?sslmode=disable

# payment-service
PORT=8084
DATABASE_URL=postgres://user:pass@payments-db:5435/payments_db?sslmode=disable

# inventory-service
PORT=8085
DATABASE_URL=postgres://user:pass@products-db:5433/products_db?sslmode=disable
```

---

## 16. Makefile Commands

```makefile
# Start full stack
make up           # docker compose up --build -d
make down         # docker compose down
make restart      # down + up

# Development (with hot reload via air)
make dev          # docker compose -f docker-compose.yml -f docker-compose.dev.yml up

# Database migrations
make migrate-up   # run all pending migrations across all services
make migrate-down # rollback last migration across all services

# Individual service migrations
make migrate SERVICE=user-service
make migrate SERVICE=product-service

# Testing
make test         # run unit tests across all services
make test-int     # run integration tests (requires Docker)
make test-e2e     # run end-to-end tests
make load-test    # run k6 load test

# Utilities
make logs         # docker compose logs -f
make logs-svc     # make logs SERVICE=order-service
make ps           # docker compose ps
make seed         # run ./scripts/seed.sh to populate test data
make kafka-topics # create all Kafka topics
make clean        # remove all volumes (wipes data)
```

---

## Quick Start

```bash
# 1. Clone and configure
git clone https://github.com/your-org/auron.git
cd auron
cp .env.example .env   # fill in Stripe, SendGrid, Twilio keys

# 2. Start everything
make up

# 3. Run migrations
make migrate-up

# 4. Seed test data
make seed

# 5. Open in browser
# Frontend  → http://localhost:3000
# API       → http://localhost:8080/api
# Grafana   → http://localhost:3001  (admin/admin)
# Prometheus → http://localhost:9090
```

---

*Last updated: 2025 — Auron Platform v1.0*
