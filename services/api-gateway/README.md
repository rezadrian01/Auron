# API Gateway

## Service Routing Model

The gateway supports two ways to route to downstream services:

1. **Explicit routes** in `routes/router.go` (best for stable public API paths)
2. **Generic pass-through route**: `/api/:service/*proxyPath`

Configured services are loaded from environment variables in this order:

1. Legacy specific vars (`USER_SERVICE_URL`, `PRODUCT_SERVICE_URL`, etc.)
2. `SERVICE_URLS` (comma-separated `name=url` pairs)
3. `SERVICE_URL_<NAME>` overrides/additions (highest precedence)

Example:

```env
SERVICE_URLS=user=http://user-service:8081,product=http://product-service:8082
SERVICE_URL_NOTIFICATION=http://notification-service:8086
```

## Add a New Service (Checklist)

### 1) Register upstream URL (no code)

Choose one:

- Add to `SERVICE_URLS`
- Or add dedicated `SERVICE_URL_<NAME>`

Example:

```env
SERVICE_URL_INVOICE=http://invoice-service:8087
```

In Docker Compose (`api-gateway.environment`):

```yaml
- SERVICE_URL_INVOICE=http://invoice-service:8087
```

### 2) Decide endpoint style

#### Option A — Fastest (no route code): generic route

Use:

- `GET /api/invoice/health` -> forwards to `invoice-service` path `/health`
- `POST /api/invoice/generate` -> forwards to `/generate`

#### Option B — Stable public API contract (recommended for core APIs)

Add explicit route(s) in `routes/router.go`:

```go
toInvoiceService := proxyHandler.ProxyTo(config.ServiceInvoice)
api.GET("/invoices/:id", toInvoiceService)
```

If your downstream service expects a different prefix, use `ProxyToWithStrip(service, prefix)`.

Example:

```go
toUserService := proxyHandler.ProxyToWithStrip(config.ServiceUser, "/api/users")
```

### 3) Add config constant (only if using explicit typed constant)

In `config/config.go`:

```go
const ServiceInvoice = "invoice"
```

This is optional for generic-only routing, but recommended for explicit route readability.

### 4) Smoke test

```bash
curl -i http://localhost:8080/api/invoice/health
```

Expected:

- `200` from downstream service if configured and reachable
- `502` with `{"error":"service unavailable","service":"invoice"}` if missing config

## Notes

- User-service path mapping is intentionally split:
  - `/api/auth/*` -> user-service `/*`
  - `/api/users/*` -> user-service `/*`
- This preserves clean gateway URLs while matching user-service handlers.
- Backward-compatible auth aliases are also supported:
  - `/api/login` and `/api/login/`
  - `/api/register` and `/api/register/`
  - `/api/refresh` and `/api/refresh/`
  - `/api/logout` and `/api/logout/`
- Canonical auth endpoints remain under `/api/auth/*`.
