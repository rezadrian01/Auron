# Auron

A production-grade e-commerce platform built with a Go microservices backend and a Next.js frontend. Each service owns its data, communicates asynchronously over Kafka, and is independently deployable via Docker.

---

## Architecture

```
                        ┌─────────────────┐
                        │    Next.js       │
                        │    Frontend      │
                        │  :3000           │
                        └────────┬────────┘
                                 │ HTTP
                                 ▼
                        ┌─────────────────┐
                        │   API Gateway   │  JWT auth · rate limiting
                        │  :8080          │  reverse proxy
                        └────────┬────────┘
                                 │
          ┌──────────┬───────────┼───────────┬──────────┬──────────┐
          ▼          ▼           ▼           ▼          ▼          ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
    │  User    │ │ Product  │ │  Order   │ │ Payment  │ │Inventory │ │Notif.    │
    │ Service  │ │ Service  │ │ Service  │ │ Service  │ │ Service  │ │ Service  │
    │  :8081   │ │  :8082   │ │  :8083   │ │  :8084   │ │  :8085   │ │  :8086   │
    └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘
         │            │            │             │             │             │
         │      ┌─────┴─────┐      │             │      ┌──────┴──────┐     │
         ▼      ▼           ▼      ▼             ▼      ▼             ▼     │
      users-db  products-db  orders-db     payments-db  products-db        │
                                                                            │
                         ┌──────────────────────────────┐                  │
                         │           Kafka               │◄─────────────────┘
                         │  (async inter-service events) │
                         └──────────────────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
                 Redis              PostgreSQL           Stripe
               (caching)          (per-service)       (payments)
```

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25 · Gin · GORM |
| Frontend | Next.js 16 · React 19 · Tailwind CSS 4 · TypeScript |
| Databases | PostgreSQL 15 (one per service) |
| Cache | Redis 7 |
| Message broker | Apache Kafka (Confluent Platform 7.6) |
| Payments | Stripe (stripe-go v76) |
| Auth | JWT HS256 (shared secret across services) |
| Containerisation | Docker · Docker Compose |
| Observability | Prometheus · Grafana |

---

## Services

| Service | Port | Responsibility |
|---------|------|----------------|
| **api-gateway** | 8080 | JWT validation, rate limiting, reverse proxy to all downstream services |
| **user-service** | 8081 | Registration, login, JWT issuance, profile, addresses |
| **product-service** | 8082 | Product catalogue, categories, PostgreSQL full-text search, Redis cache |
| **order-service** | 8083 | Cart management, order creation, inventory reservation |
| **payment-service** | 8084 | Stripe PaymentIntent lifecycle, webhook handling, payment status |
| **inventory-service** | 8085 | Stock levels, reservation/release on order events |
| **notification-service** | 8086 | Email delivery via SMTP (stateless Kafka consumer, no database) |

### Supporting infrastructure

| Service | Port | Purpose |
|---------|------|---------|
| users-db | 5432 | PostgreSQL for user-service |
| products-db | 5433 | PostgreSQL for product-service and inventory-service |
| orders-db | 5434 | PostgreSQL for order-service |
| payments-db | 5435 | PostgreSQL for payment-service |
| Redis | 6380 | Shared cache (token deny-list, product/payment caching) |
| Kafka | 9092 | External listener (services use internal port 29092) |
| Kafka UI | 8090 | Web UI for browsing topics and messages |
| Prometheus | 9090 | Metrics scraping |
| Grafana | 3001 | Dashboards (admin / admin) |

---

## Kafka Event Flow

```
user-service        ──► user.created
order-service       ──► order.created   ──► payment-service (create PaymentIntent)
                                        ──► inventory-service (reserve stock)
order-service       ──► order.cancelled ──► inventory-service (release stock)
payment-service     ──► payment.created
payment-service     ──► payment.completed ──► notification-service
payment-service     ──► payment.failed    ──► notification-service
inventory-service   ──► inventory.low_stock ──► notification-service
```

All topics use the prefix convention `<domain>.<event>` and 6 partitions by default.

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose v2
- A [Stripe](https://dashboard.stripe.com/register) account (for payment testing)
- [Stripe CLI](https://stripe.com/docs/stripe-cli) (optional, for local webhook forwarding)

### 1. Clone and configure environment

```bash
git clone https://github.com/rezadrian01/auron.git
cd auron
cp .env.example .env
```

Open `.env` and fill in the required values:

```env
# Required
JWT_SECRET=your-32-char-minimum-secret-here
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Optional — SMTP for notification emails
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_FROM=noreply@auron.shop
```

> **JWT_SECRET** must be the same value used by all services. It is shared via the root `.env` file and injected by Docker Compose into the gateway and each service.

### 2. Start the stack

```bash
make up
```

This builds all Docker images and starts every service. On first boot, each service runs `AutoMigrate` to create its database schema.

Wait ~30 seconds for Kafka and all databases to report healthy, then verify:

```bash
make health
```

### 3. (Optional) Set up Stripe webhook forwarding for local testing

```bash
stripe listen --forward-to http://localhost:8080/api/payments/webhook/stripe
```

Copy the `whsec_...` secret printed by the CLI and set it as `STRIPE_WEBHOOK_SECRET` in your `.env`, then restart the payment-service:

```bash
docker compose restart payment-service
```

### 4. Create an admin account

All registrations default to the `customer` role. To promote a user to admin:

```bash
docker compose exec users-db psql -U auron -d users_db \
  -c "UPDATE users SET role='admin' WHERE email='your@email.com';"
```

---

## Make Commands

```
make up            Build images and start the full stack
make down          Stop and remove all containers
make restart       down + up
make infra-up      Start only databases, Kafka, Redis, and observability tools
make build         Compile all Go services locally
make build-docker  Build all Docker images without starting containers
make test          Run go test ./... across all services
make logs          Tail logs from all containers
make logs-svc SERVICE=order-service   Tail logs from one service
make ps            Show running containers
make health        Check HTTP health endpoints for all services
make kafka-topics  Create all Kafka topics manually
make clean         Remove all containers and volumes (destructive)
make deps          Download Go module dependencies for all services
make tidy          Run go mod tidy across all services
```

---

## Payment Flow

The checkout flow works as follows:

```
1. Add items to cart        POST /api/cart/items
2. Create order             POST /api/orders
                            └─► Kafka: order.created
                                └─► payment-service creates Stripe PaymentIntent
3. Fetch client_secret      GET  /api/payments/order/:order_id
4. Confirm payment          Stripe.js confirmCardPayment(client_secret)
                            └─► Stripe webhook: payment_intent.succeeded
                                └─► payment-service sets status = completed
                                └─► Kafka: payment.completed
5. Poll payment status      GET  /api/payments/:payment_id
```

> There is a short async delay between step 2 and when the `client_secret` is available (payment-service must process the Kafka event and call the Stripe API). Polling `GET /api/payments/order/:order_id` until `status` is no longer `pending` is the recommended pattern.

---

## API Reference

Full endpoint documentation is in [API_DOCS.md](./API_DOCS.md).

A ready-to-import Postman collection is at [Auron.postman_collection.json](./Auron.postman_collection.json). It includes:
- Collection-level Bearer auth using `{{access_token}}`
- Test scripts that auto-save tokens, IDs, and UUIDs after each request
- All 30 endpoints organised into folders

### Quick reference

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | — | Register |
| POST | `/api/auth/login` | — | Login |
| POST | `/api/auth/refresh` | — | Refresh token |
| POST | `/api/auth/logout` | ✓ | Logout |
| GET | `/api/users/me` | ✓ | Get profile |
| PUT | `/api/users/me` | ✓ | Update profile |
| GET/POST | `/api/users/me/addresses` | ✓ | Addresses |
| GET | `/api/products` | — | List products (search, filter, sort) |
| GET | `/api/products/:id` | — | Get product |
| POST/PUT/DELETE | `/api/products` | admin | Manage products |
| GET | `/api/categories` | — | List categories |
| POST | `/api/categories` | admin | Create category |
| GET | `/api/cart` | ✓ | Get cart |
| POST | `/api/cart/items` | ✓ | Add item |
| PUT/DELETE | `/api/cart/items/:id` | ✓ | Update/remove item |
| GET/POST | `/api/orders` | ✓ | List / create order |
| GET | `/api/orders/:id` | ✓ | Get order |
| PUT | `/api/orders/:id/cancel` | ✓ | Cancel order |
| GET | `/api/payments/:id` | ✓ | Get payment |
| GET | `/api/payments/order/:id` | ✓ | Get payment by order (includes `client_secret`) |
| POST | `/api/payments/webhook/stripe` | — | Stripe webhook |
| GET | `/api/inventory/:product_id` | — | Get stock |
| PUT | `/api/inventory/:product_id` | admin | Set stock |
| GET | `/api/health` | — | Gateway health |

---

## Project Structure

```
auron/
├── docker-compose.yml          # Full stack definition
├── Makefile                    # Developer commands
├── .env.example                # Environment variable template
├── API_DOCS.md                 # Full API reference
├── Auron.postman_collection.json
│
├── services/
│   ├── api-gateway/            # Gin · JWT middleware · httputil.ReverseProxy
│   ├── user-service/           # Gin · GORM · Redis · Kafka producer
│   ├── product-service/        # Gin · GORM · Redis · full-text search
│   ├── order-service/          # Gin · GORM · Redis · Kafka producer
│   ├── payment-service/        # Gin · GORM · Redis · stripe-go · Kafka
│   ├── inventory-service/      # Gin · GORM · Redis · Kafka consumer+producer
│   └── notification-service/   # Gin (health only) · net/smtp · Kafka consumer
│
├── shared/                     # Shared Go modules (Kafka helpers, Redis client)
│
├── frontend/                   # Next.js 16 · React 19 · Tailwind CSS 4
│   ├── app/                    # App Router pages and layouts
│   └── components/             # Cart, checkout, product, UI components
│
└── infra/
    ├── kafka/topics.sh         # Topic creation script
    ├── postgres/               # Database init SQL
    ├── prometheus/             # Scrape config
    └── grafana/                # Dashboard definitions
```

Each Go service follows the same internal layout:

```
<service>/
├── main.go
├── cmd/           # Wiring: config, database, redis, kafka, HTTP server
└── internal/
    ├── domain/    # Entities, DTOs, repository and service interfaces
    ├── handler/   # HTTP handlers (Gin)
    ├── route/     # Route registration
    ├── service/   # Business logic
    ├── repository/# GORM implementations
    ├── cache/     # Redis implementations
    └── events/    # Kafka producers / consumers
```

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `JWT_SECRET` | yes | HS256 signing secret, shared across all services |
| `STRIPE_SECRET_KEY` | yes | Stripe secret key (`sk_test_...` or `sk_live_...`) |
| `STRIPE_WEBHOOK_SECRET` | yes | Stripe webhook signing secret (`whsec_...`) |
| `SMTP_HOST` | no | SMTP server hostname (defaults to no-op logging) |
| `SMTP_PORT` | no | SMTP port (default `587`) |
| `SMTP_FROM` | no | From address for notification emails |
| `SMTP_USER` | no | SMTP username (omit for unauthenticated relay, e.g. MailHog) |
| `SMTP_PASS` | no | SMTP password |
| `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | no | Stripe publishable key for frontend Stripe.js |

Database URLs and internal service URLs are pre-configured in `docker-compose.yml` and do not need to be set in `.env`.

---

## License

MIT — see [LICENSE](./LICENSE).  
Copyright © 2026 Ahmad Reza Adrian.
