# Payment Service — Implementation Plan

## Overview

The payment service (port **8084**) integrates with **Stripe** to handle payment processing for orders.  
It is a **Kafka consumer + HTTP API** hybrid service:

- Consumes `order.created` events → creates a Stripe PaymentIntent → stores Payment record
- Exposes HTTP endpoints for payment lookup and Stripe webhook ingestion
- Publishes `payment.created`, `payment.completed`, `payment.failed` Kafka events for downstream consumers (e.g., order-service to update order status, notification-service to email the user)

### Gateway Routes (already wired)

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET` | `/api/payments/:id` | Required | Get payment details by payment UUID |
| `POST` | `/api/payments/webhook/stripe` | None (Stripe signs) | Stripe webhook handler |

### Payment Lifecycle

```
Frontend creates order
        ↓
order-service → publishes order.created (Kafka)
        ↓
payment-service consumes order.created
        ↓
Creates Stripe PaymentIntent → stores Payment(status=pending, client_secret)
        ↓
Publishes payment.created (contains payment_id + client_secret for frontend)
        ↓
Frontend uses client_secret + Stripe.js to confirm payment
        ↓
Stripe fires POST /api/payments/webhook/stripe
        ↓
payment-service verifies webhook signature → updates status
        ↓
Publishes payment.completed or payment.failed
```

---

## Folder Structure

```
services/payment-service/
├── cmd/
│   ├── config.go          # env vars → appConfig struct
│   ├── dotenv.go          # load .env file in non-production
│   ├── infrastructure.go  # setupDatabase, setupRedis, runMigrations
│   ├── kafka.go           # setupKafkaPublisher, startKafkaConsumer
│   ├── run.go             # wire everything together
│   └── server.go          # setupRouter, registerGracefulShutdown
├── db/
│   └── 001_create_payments.up.sql
├── internal/
│   ├── cache/
│   │   └── payment_cache.go
│   ├── client/
│   │   └── stripe_client.go
│   ├── domain/
│   │   ├── payment.go     # Payment entity, PaymentStatus, DTOs
│   │   ├── errors.go      # sentinel errors
│   │   ├── repository.go  # PaymentRepository interface
│   │   ├── service.go     # PaymentService interface
│   │   ├── cache.go       # PaymentCache interface
│   │   ├── events.go      # EventPublisher interface + topic constants
│   │   └── client.go      # StripeClient interface
│   ├── events/
│   │   ├── kafka_publisher.go
│   │   └── kafka_consumer.go
│   ├── handler/
│   │   └── payment_handler.go
│   ├── middleware/
│   │   └── stripe_webhook.go  # raw body capture for signature verification
│   ├── repository/
│   │   └── payment_repository.go
│   ├── route/
│   │   └── payment_route.go
│   └── service/
│       └── payment_service.go
├── main.go
├── Dockerfile
├── go.mod
├── .env
└── .env.example
```

---

## Tasks

### Task 1 — Domain Layer

Create all files under `internal/domain/`:

**`payment.go`**
- `PaymentStatus` type (`pending`, `processing`, `completed`, `failed`, `refunded`)
- `Payment` struct with GORM tags:
  - `id uuid`, `order_id uuid` (unique index), `user_id uuid`, `amount float64`, `currency varchar(10) default 'usd'`
  - `status varchar(50) default 'pending'`, `stripe_payment_intent_id varchar(255)`, `stripe_client_secret text`
  - `failure_reason text`, `created_at`, `updated_at`
- `PaymentResponse` DTO — excludes `stripe_client_secret` for normal reads
- `PaymentInitResponse` DTO — includes `stripe_client_secret` (returned only on `payment.created` event, never via HTTP)
- `OrderCreatedEvent` struct — shape of the Kafka message from order-service: `{order_id, user_id, total_amount, items[]}`

**`errors.go`**
- `ErrPaymentNotFound`, `ErrPaymentAlreadyExists`, `ErrInvalidWebhookSignature`, `ErrForbidden`, `ErrUnauthorized`

**`repository.go`**
```go
type PaymentRepository interface {
    GetPaymentByID(id uuid.UUID) (*Payment, error)
    GetPaymentByOrderID(orderID uuid.UUID) (*Payment, error)
    CreatePayment(payment *Payment) (*Payment, error)
    UpdatePaymentStatus(id uuid.UUID, status PaymentStatus, failureReason string) (*Payment, error)
    UpdateStripePaymentIntentID(id uuid.UUID, intentID, clientSecret string) (*Payment, error)
}
```

**`service.go`**
```go
type PaymentService interface {
    GetPaymentByID(ctx context.Context, userID, paymentID uuid.UUID) (*PaymentResponse, error)
    HandleOrderCreated(ctx context.Context, event OrderCreatedEvent) error
    HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error
}
```

**`cache.go`**
```go
type PaymentCache interface {
    GetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error)
    SetPayment(ctx context.Context, payment *Payment) error
    InvalidatePayment(ctx context.Context, paymentID uuid.UUID) error
}
```

**`events.go`**
- `EventPublisher` interface with `Publish(topic string, key string, payload any) error` and `Close() error`
- Constants: `TopicPaymentCreated = "payment.created"`, `TopicPaymentCompleted = "payment.completed"`, `TopicPaymentFailed = "payment.failed"`

**`client.go`**
```go
type StripeClient interface {
    CreatePaymentIntent(ctx context.Context, amount float64, currency string, metadata map[string]string) (intentID, clientSecret string, err error)
}
```

---

### Task 2 — DB Migration

**`db/001_create_payments.up.sql`**
```sql
CREATE TABLE IF NOT EXISTS payments (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id                  UUID NOT NULL,
    user_id                   UUID NOT NULL,
    amount                    DECIMAL(12,2) NOT NULL,
    currency                  VARCHAR(10) NOT NULL DEFAULT 'usd',
    status                    VARCHAR(50) NOT NULL DEFAULT 'pending',
    stripe_payment_intent_id  VARCHAR(255),
    stripe_client_secret      TEXT,
    failure_reason            TEXT,
    created_at                TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_payments_status CHECK (
        status IN ('pending','processing','completed','failed','refunded')
    ),
    CONSTRAINT chk_payments_amount CHECK (amount > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
```

---

### Task 3 — Repository Layer

**`internal/repository/payment_repository.go`**
- GORM implementation of `domain.PaymentRepository`
- `GetPaymentByID` and `GetPaymentByOrderID` return `ErrPaymentNotFound` on GORM `ErrRecordNotFound`
- `UpdatePaymentStatus`: updates `status`, `failure_reason`, and `updated_at` in a single `db.Model().Updates()` call
- `UpdateStripePaymentIntentID`: sets `stripe_payment_intent_id` and `stripe_client_secret`

---

### Task 4 — Cache Layer

**`internal/cache/payment_cache.go`**
- `PaymentCache` struct wrapping `*redis.Client`
- Key: `payment:<uuid>` (TTL 1h)
- JSON marshal/unmarshal; miss returns `nil, nil`

---

### Task 5 — Kafka Events (Publisher + Consumer)

**`internal/events/kafka_publisher.go`**
- Same pattern as order-service: `kafkaPublisher` with `writers map[string]*kafka.Writer`
- `Publish(topic, key string, payload any) error` — JSON-marshals payload, writes message
- `Close() error`

**`internal/events/kafka_consumer.go`**
- `KafkaConsumer` struct: `reader *kafka.Reader`, `paymentService domain.PaymentService`, `logger`
- `Start(ctx context.Context)` — goroutine reading messages from `order.created` topic, group `payment-service`
- On message: unmarshal `domain.OrderCreatedEvent`, call `paymentService.HandleOrderCreated(ctx, event)`
- Log errors, commit offset, never crash — errors are non-fatal
- `Close() error`

---

### Task 6 — Stripe Client

**`internal/client/stripe_client.go`**
- `stripeClient` struct with `secretKey string`
- Implements `domain.StripeClient`
- `CreatePaymentIntent`: calls Stripe Go SDK `paymentintent.New()` with amount (converted to cents), currency, and metadata (`order_id`, `user_id`)
- Returns `intentID` and `clientSecret`

**Dependencies to add:**
```
github.com/stripe/stripe-go/v76
```

---

### Task 7 — Service Layer

**`internal/service/payment_service.go`**

**`HandleOrderCreated(ctx, event)`**
1. Check `GetPaymentByOrderID` — if already exists, return nil (idempotent)
2. Call `stripeClient.CreatePaymentIntent(ctx, event.TotalAmount, "usd", metadata)`
3. Build and `CreatePayment` record (status=pending, stripe IDs set)
4. Cache the payment
5. Publish `payment.created` event asynchronously (contains `payment_id`, `order_id`, `user_id`, `client_secret`)

**`GetPaymentByID(ctx, userID, paymentID)`**
1. Cache-aside: check cache first
2. DB fallback on miss
3. Ownership check: `payment.UserID != userID` → `ErrForbidden`
4. Return `PaymentResponse` (no client_secret)

**`HandleStripeWebhook(ctx, payload, signature)`**
1. Construct Stripe event: `webhook.ConstructEvent(payload, signature, webhookSecret)` → error → `ErrInvalidWebhookSignature`
2. Switch on event type:
   - `payment_intent.succeeded` → `UpdatePaymentStatus(completed)` → publish `payment.completed` async
   - `payment_intent.payment_failed` → `UpdatePaymentStatus(failed, failureReason)` → publish `payment.failed` async
   - `payment_intent.processing` → `UpdatePaymentStatus(processing)`
3. Invalidate and re-cache payment after status update

---

### Task 8 — Handler + Route Layers

**`internal/handler/payment_handler.go`**
- `PaymentHandler` struct with `paymentService domain.PaymentService`
- `getUserID(c *gin.Context) (uuid.UUID, error)` — reads `X-User-ID` header
- `GetPaymentByID(c *gin.Context)` — parse `:id` param, call service, return 200/404/403
- `HandleStripeWebhook(c *gin.Context)` — reads raw body (from context, set by middleware), reads `Stripe-Signature` header, calls service, always returns 200 (Stripe retries on non-200)
- `handleError(c, err)` — maps domain errors to status codes

**`internal/middleware/stripe_webhook.go`**
- Gin middleware that reads and buffers the raw request body into `c.Set("rawBody", body)` before `c.Next()`
- Required because Stripe signature verification needs the exact raw bytes, and `c.Request.Body` is consumed after `ShouldBindJSON`

**`internal/route/payment_route.go`**
```go
func RegisterPaymentRoutes(router *gin.Engine, paymentHandler *handler.PaymentHandler) {
    api := router.Group("/")
    api.GET("/payments/:id", paymentHandler.GetPaymentByID)
    api.POST("/payments/webhook/stripe", middleware.CaptureRawBody(), paymentHandler.HandleStripeWebhook)
}
```

---

### Task 9 — cmd Bootstrap

**`cmd/config.go`**
```go
type appConfig struct {
    Port                 string
    DatabaseURL          string
    RedisURL             string
    KafkaBrokers         string
    StripeSecretKey      string
    StripeWebhookSecret  string
}
```
Defaults: port 8084, localhost:5435, localhost:6379, localhost:9092

**`cmd/dotenv.go`** — identical pattern to order-service

**`cmd/infrastructure.go`**
- `setupDatabase` with connection pooling
- `runMigrations` — AutoMigrate `domain.Payment`
- `setupRedis` — ParseURL + Ping

**`cmd/kafka.go`**
- `paymentTopics`: TopicPaymentCreated, TopicPaymentCompleted, TopicPaymentFailed
- `setupKafkaPublisher(brokers string) domain.EventPublisher`
- `setupKafkaConsumer(brokers string, svc domain.PaymentService) *events.KafkaConsumer`
- `startKafkaConsumer(consumer *events.KafkaConsumer)` — launches goroutine

**`cmd/run.go`**
```go
func Run() {
    cfg := loadConfig()
    db := setupDatabase(cfg.DatabaseURL)
    runMigrations(db)
    redisClient := setupRedis(cfg.RedisURL)
    publisher := setupKafkaPublisher(cfg.KafkaBrokers)
    paymentRepo := repository.NewPaymentRepository(db)
    paymentCache := cache.NewPaymentCache(redisClient)
    stripeClient := client.NewStripeClient(cfg.StripeSecretKey)
    paymentSvc := service.NewPaymentService(paymentRepo, paymentCache, stripeClient, publisher, cfg.StripeWebhookSecret)
    consumer := setupKafkaConsumer(cfg.KafkaBrokers, paymentSvc)
    startKafkaConsumer(consumer)
    paymentHandler := handler.NewPaymentHandler(paymentSvc)
    router := setupRouter(paymentHandler)
    registerGracefulShutdown(db, redisClient, publisher, consumer)
    router.Run(fmt.Sprintf(":%s", cfg.Port))
}
```

**`cmd/server.go`**
- `setupRouter(*handler.PaymentHandler) *gin.Engine` — release mode, /health, /metrics, calls `RegisterPaymentRoutes`
- `registerGracefulShutdown` — SIGTERM/SIGINT handler closing DB, Redis, Kafka publisher and consumer

---

### Task 10 — Entry Point, Dockerfile, and Env Files

**`main.go`** — `cmd.Run()`

**`Dockerfile`** — same multi-stage pattern: `golang:1.25-alpine` builder → `alpine:3.18` runtime; binary named `payment-service`; EXPOSE 8084

**`.env`** — local dev values
```
PORT=8084
DATABASE_URL=postgres://auron:auron_pass@localhost:5435/payments_db?sslmode=disable
REDIS_URL=redis://localhost:6379/0
KAFKA_BROKERS=localhost:9092
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
GORM_LOG_LEVEL=warn
```

**`.env.example`** — same with placeholder values

---

### Task 11 — docker-compose Wiring

Update `docker-compose.yml` `payment-service` environment block:
```yaml
- REDIS_URL=redis://redis:6379/0
- KAFKA_BROKERS=kafka:29092
- STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
- STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
```

Add `kafka` to `payment-service.depends_on` (after payments-db).

---

## Key Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Stripe integration | PaymentIntents API | Supports SCA, supports card, wallet, BNPL via `automatic_payment_methods` |
| Payment initiation | Kafka consumer (`order.created`) | Decoupled — order-service doesn't need to call payment-service HTTP |
| Webhook raw body | Middleware that caches raw bytes | Stripe signature verification requires exact bytes; Gin's binding consumes the body |
| Idempotency | Check `GetPaymentByOrderID` before creating | Prevents duplicate Stripe intents if `order.created` is delivered multiple times |
| client_secret exposure | Only via `payment.created` Kafka event | Never exposed via HTTP API to avoid interception; downstream services forward to frontend |
| Stripe amount | `int64(amount * 100)` cents | Stripe API requires smallest currency unit |
| Webhook response | Always return 200 | Stripe retries on 4xx/5xx; log errors but don't fail the HTTP response |
| KafkaBrokers for consumer | `order.created` topic, group `payment-service` | Group ID ensures each message is processed exactly once per service instance |
| go.mod module | `auron/payment-service` | Matches pattern of all other services |

---

## Dependencies

```
github.com/gin-gonic/gin v1.12.0
github.com/google/uuid v1.6.0
github.com/redis/go-redis/v9 v9.19.0
github.com/segmentio/kafka-go v0.4.51
gorm.io/driver/postgres v1.6.0
gorm.io/gorm v1.31.1
github.com/stripe/stripe-go/v76 v76.x.x
github.com/joho/godotenv v1.5.1
```
