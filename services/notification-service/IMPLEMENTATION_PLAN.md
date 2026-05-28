# Notification Service — Implementation Plan

## Overview

The notification-service delivers transactional emails to users triggered by domain events
published on Kafka by other services. It consumes events, fetches any additional context it
needs, renders email content, and sends via SMTP (using Go's `net/smtp` standard library).

No database is required — all state comes from incoming events.  
No REST API endpoints — the service is entirely event-driven.  
The `/health` endpoint (required by docker-compose healthcheck) is the only HTTP surface.

**Port:** `8086`  
**Kafka consumer group:** `notification-service`

---

## Topics consumed and emails sent

| Kafka Topic          | Email sent                                       |
|----------------------|--------------------------------------------------|
| `user.created`       | Welcome email to new user                        |
| `order.created`      | Order confirmation with item summary             |
| `order.cancelled`    | Order cancellation notice                        |
| `payment.completed`  | Payment receipt / success confirmation           |
| `payment.failed`     | Payment failure notice with reason               |
| `inventory.low_stock`| (internal) low-stock alert — no user email       |

> `inventory.low_stock` is consumed but emails are suppressed for now (logged only).
> `payment.created` is skipped — that event carries Stripe client_secret; not relevant here.

---

## Directory structure

```
services/notification-service/
├── IMPLEMENTATION_PLAN.md
├── main.go
├── Dockerfile
├── .env.example
├── go.mod / go.sum
├── cmd/
│   ├── config.go          # env → Config struct
│   ├── dotenv.go          # .env loader (dev only)
│   ├── kafka.go           # setupKafkaConsumer, closeKafkaConsumer
│   ├── run.go             # Run(), registerGracefulShutdown
│   └── server.go          # setupRouter (health only)
└── internal/
    ├── domain/
    │   ├── events.go      # consumed topic constants + event structs
    │   └── service.go     # NotificationService interface
    ├── email/
    │   └── smtp_sender.go # EmailSender interface + smtpSender impl
    ├── events/
    │   └── kafka_consumer.go  # multi-topic KafkaConsumer
    ├── handler/
    │   └── health_handler.go  # GET /health
    ├── route/
    │   └── route.go           # RegisterRoutes
    └── service/
        └── notification_service.go  # NotificationService impl
```

---

## Tasks

### Task 1 — Domain: event structs + service interface

**File:** `internal/domain/events.go`

Topic constants for all consumed events:
```
TopicUserCreated      = "user.created"
TopicOrderCreated     = "order.created"
TopicOrderCancelled   = "order.cancelled"
TopicPaymentCompleted = "payment.completed"
TopicPaymentFailed    = "payment.failed"
TopicInventoryLowStock = "inventory.low_stock"
```

Event structs (match producing service JSON tags exactly):

- `UserCreatedEvent`   — `id`, `email`, `name`
- `OrderCreatedEvent`  — `id` (order ID), `user_id`, `total_amount`, `items[]` (`product_id`, `quantity`, `price`)
- `OrderCancelledEvent`— same shape as `OrderCreatedEvent`
- `PaymentEvent`       — `id`, `order_id`, `user_id`, `amount`, `currency`, `status`, `failure_reason`
- `InventoryLowStockEvent` — `product_id`, `total_quantity`, `reserved_quantity`

**File:** `internal/domain/service.go`

```go
type NotificationService interface {
    HandleUserCreated(ctx context.Context, event UserCreatedEvent) error
    HandleOrderCreated(ctx context.Context, event OrderCreatedEvent) error
    HandleOrderCancelled(ctx context.Context, event OrderCancelledEvent) error
    HandlePaymentCompleted(ctx context.Context, event PaymentEvent) error
    HandlePaymentFailed(ctx context.Context, event PaymentEvent) error
}
```

---

### Task 2 — Email sender: SMTP client

**File:** `internal/email/smtp_sender.go`

Interface:
```go
type EmailSender interface {
    Send(to, subject, body string) error
}
```

Implementation `smtpSender`:
- Config: `host`, `port`, `from`, `user`, `pass`, `secure bool`
- Uses `net/smtp` standard library
- If `user == ""` — use `smtp.SendMail` without auth (relay/MailHog mode for dev)
- Otherwise — `smtp.PlainAuth` + `smtp.SendMail`
- Body format: plain-text `Content-Type: text/plain; charset=UTF-8` (no HTML templates for now)
- Helper `buildMessage(from, to, subject, body string) []byte` — formats RFC 2822 headers

Constructor: `NewSMTPSender(host string, port int, from, user, pass string, secure bool) EmailSender`

---

### Task 3 — Service layer: notification logic

**File:** `internal/service/notification_service.go`

`notificationService` struct — fields: `sender email.EmailSender`

Each handler method:
1. Composes subject + plain-text body using Go string formatting
2. Calls `sender.Send(to, subject, body)`
3. Returns any error

Email content per event:

**HandleUserCreated** — to: `event.Email`
```
Subject: Welcome to Auron, {{Name}}!
Body:    Hi {{Name}}, your account has been created successfully. Start shopping at Auron!
```

**HandleOrderCreated** — to: derived from `user_id`; problem: no user email in event.
Resolution: embed the user email in the `OrderCreatedEvent` from the order-service side.  
For now, log a warning and skip (order-service does not include email — it would require  
a cross-service call). Store `user_id` in log; send to a no-op target.  
**Alternative (chosen):** order-service includes `user_email` in the event.  
Check if order-service event has `user_email`; if not, we skip sending and log.

```
Subject: Order Confirmed — #{{OrderID}}
Body:    Your order {{OrderID}} for ${{TotalAmount}} has been placed successfully.
         Items: (list each product_id × quantity)
```

**HandleOrderCancelled** — same resolution as above
```
Subject: Order Cancelled — #{{OrderID}}
Body:    Your order {{OrderID}} has been cancelled.
```

**HandlePaymentCompleted**
```
Subject: Payment Received — ${{Amount}} {{Currency}}
Body:    Your payment of ${{Amount}} {{Currency}} for order {{OrderID}} was successful.
         Payment ID: {{ID}}
```

**HandlePaymentFailed**
```
Subject: Payment Failed for Order #{{OrderID}}
Body:    Your payment of ${{Amount}} {{Currency}} for order {{OrderID}} failed.
         Reason: {{FailureReason}}
         Please retry or contact support.
```

> `user_id` from payment events is a UUID — no email available without a user lookup.
> Payment events from the payment-service include `user_id` (UUID) but NOT email.
> For the initial implementation: log a warning, skip sending.
> A follow-up can add a user-service HTTP call to resolve email by user_id.

---

### Task 4 — Kafka consumer: multi-topic consumer

**File:** `internal/events/kafka_consumer.go`

Same pattern as inventory-service: `[]readerEntry` with one `kafka.Reader` per topic.

Topics: `user.created`, `order.created`, `order.cancelled`, `payment.completed`, `payment.failed`, `inventory.low_stock`

Consumer group: `notification-service`

`handleMessage(topic, value []byte)`:
- Switch on topic
- Unmarshal into correct event struct
- Call appropriate `NotificationService` method
- Log errors, do NOT retry (offset always committed)

`Start(ctx context.Context)` — one goroutine per reader  
`Close() error` — close all readers

---

### Task 5 — Health handler + route

**File:** `internal/handler/health_handler.go`

```go
func (h *HealthHandler) GetHealth(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok", "service": "notification-service"})
}
```

**File:** `internal/route/route.go`

```go
func RegisterRoutes(router *gin.Engine, healthHandler *handler.HealthHandler) {
    router.GET("/health", healthHandler.GetHealth)
}
```

---

### Task 6 — cmd bootstrap: config, infrastructure, kafka, server, run

**File:** `cmd/config.go`

```go
type Config struct {
    Port         string
    SMTPHost     string
    SMTPPort     int
    SMTPFrom     string
    SMTPUser     string
    SMTPPass     string
    SMTPSecure   bool
    KafkaBrokers []string
}
```
`loadConfig()` reads from env; `KAFKA_BROKERS` splits on `,`.

**File:** `cmd/dotenv.go` — same `.env` loader pattern as other services

**File:** `cmd/kafka.go`

```go
func setupKafkaConsumer(brokers []string, svc domain.NotificationService) *events.KafkaConsumer
func startKafkaConsumer(ctx context.Context, consumer *events.KafkaConsumer)
func closeKafkaConsumer(consumer *events.KafkaConsumer)
```

**File:** `cmd/server.go` — `setupRouter` (health only, no auth middleware)

**File:** `cmd/run.go` — `Run()` wires everything + `registerGracefulShutdown`

---

### Task 7 — Entry point, Dockerfile, .env.example, docker-compose

**File:** `main.go`
```go
package main
import "auron/notification-service/cmd"
func main() { cmd.Run() }
```

**File:** `Dockerfile`
- `golang:1.25-alpine` builder → `alpine:3.18` runtime
- Binary: `/notification-service`
- EXPOSE 8086

**File:** `.env.example`
```
PORT=8086
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_FROM=noreply@auron.shop
SMTP_USER=
SMTP_PASS=
SMTP_SECURE=false
KAFKA_BROKERS=localhost:9092
```

**docker-compose.yml** — update `notification-service` stanza:
- Add `KAFKA_BROKERS=kafka:29092`
- Add `depends_on: kafka: condition: service_healthy`

---

## Dependencies (go.mod)

```
github.com/gin-gonic/gin
github.com/google/uuid
github.com/segmentio/kafka-go
github.com/joho/godotenv
```

No GORM, no Redis — this service has no database or cache.

---

## Design decisions

| Decision | Choice | Reason |
|----------|--------|--------|
| No database | Stateless | Emails are fire-and-forget; no state to persist |
| `net/smtp` not a library | Standard library | No extra dep; plain-text emails are sufficient |
| Dev mode (no SMTP auth) | `SMTP_USER=""` → no auth | Works with MailHog out of the box |
| Email for order/payment | Skip if no email in event | Cross-service user lookup adds coupling; deferrable |
| No retry on Kafka error | Log + commit | Idempotency not guaranteed; prevents consumer stall |
| Multi-reader consumer | One reader per topic | Same pattern as inventory-service; clean shutdown |
