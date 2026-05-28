# Auron API Documentation

Base URL: `http://localhost:8080`  
All endpoints are prefixed with `/api`.

---

## Table of Contents

- [Authentication](#authentication)
- [Users](#users)
- [Products](#products)
- [Categories](#categories)
- [Cart](#cart)
- [Orders](#orders)
- [Payments](#payments)
- [Inventory](#inventory)
- [Health](#health)
- [Response Envelope](#response-envelope)
- [Error Codes](#error-codes)

---

## Response Envelope

All responses follow a consistent envelope format.

**Success:**
```json
{
  "success": true,
  "data": { ... }
}
```

**Paginated success:**
```json
{
  "success": true,
  "data": [ ... ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 100
  }
}
```

**Error:**
```json
{
  "success": false,
  "error": "descriptive error message"
}
```

**Exceptions:** Auth token endpoints (`/login`, `/refresh`) return `access_token` and `refresh_token` at the top level (no `data` wrapper). The Stripe webhook endpoint returns `{"received": true}`.

---

## Authentication

Rate limited to **20 requests per minute** per IP.

All auth routes are prefixed with `/api/auth`.

---

### Register

`POST /api/auth/register`

Creates a new customer account. The `role` field is ignored for security — all registrations default to `customer`.

**Request body:**
```json
{
  "email": "user@example.com",
  "password": "securepass123",
  "confirm_password": "securepass123",
  "name": "Jane Doe"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `email` | string | yes | valid email |
| `password` | string | yes | min 8 chars |
| `confirm_password` | string | yes | must match `password` |
| `name` | string | yes | — |

**Response `201`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "Jane Doe",
    "role": "customer"
  }
}
```

**Errors:** `400` invalid body · `409` email already exists

---

### Login

`POST /api/auth/login`

Returns JWT tokens. Tokens are also set as `HttpOnly` cookies (`access_token`, `refresh_token`).

**Request body:**
```json
{
  "email": "user@example.com",
  "password": "securepass123"
}
```

**Response `200`:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ..."
}
```

**Errors:** `400` invalid body · `401` invalid credentials

---

### Refresh Token

`POST /api/auth/refresh`

Exchange a refresh token for a new access token. Accepts the token from the request body or the `refresh_token` cookie.

**Request body:**
```json
{
  "refresh_token": "eyJ..."
}
```

**Response `200`:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ..."
}
```

**Errors:** `400` missing token · `401` invalid or expired token

---

### Logout

`POST /api/auth/logout`  
**Auth required.**

Revokes the refresh token. Clears both cookies. Accepts the token from the request body or the `refresh_token` cookie.

**Request body:**
```json
{
  "refresh_token": "eyJ..."
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "logged out"
}
```

**Errors:** `400` missing token · `401` invalid token

---

## Users

All routes require a valid `Authorization: Bearer <access_token>` header.

---

### Get Profile

`GET /api/users/me`

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "Jane Doe",
    "role": "customer"
  }
}
```

---

### Update Profile

`PUT /api/users/me`

All fields are optional — only provided fields are updated.

**Request body:**
```json
{
  "name": "Jane Smith",
  "email": "new@example.com",
  "password": "newpass123"
}
```

| Field | Type | Constraints |
|-------|------|-------------|
| `name` | string | — |
| `email` | string | valid email |
| `password` | string | min 8 chars |

**Response `200`:** same shape as Get Profile

**Errors:** `400` validation · `409` email taken

---

### Add Address

`POST /api/users/me/addresses`

**Request body:**
```json
{
  "label": "Home",
  "street": "123 Main St",
  "city": "Jakarta",
  "state": "DKI Jakarta",
  "country": "Indonesia",
  "postal_code": "12345",
  "is_default": true
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `street` | string | yes | — |
| `city` | string | yes | — |
| `country` | string | yes | — |
| `label` | string | no | e.g. "Home", "Office" |
| `state` | string | no | — |
| `postal_code` | string | no | — |
| `is_default` | bool | no | defaults to `false` |

**Response `201`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "label": "Home",
    "street": "123 Main St",
    "city": "Jakarta",
    "state": "DKI Jakarta",
    "country": "Indonesia",
    "postal_code": "12345",
    "is_default": true
  }
}
```

---

### List Addresses

`GET /api/users/me/addresses`

**Response `200`:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "label": "Home",
      "street": "123 Main St",
      "city": "Jakarta",
      "country": "Indonesia",
      "is_default": true
    }
  ]
}
```

---

### Update Address

`PUT /api/users/me/addresses/:id`

All fields are optional — only provided fields are updated.

**Request body:** same fields as Add Address (all optional)

**Response `200`:**
```json
{
  "success": true,
  "data": { ...address }
}
```

**Errors:** `400` invalid ID · `404` address not found

---

### Delete Address

`DELETE /api/users/me/addresses/:id`

**Response `200`:**
```json
{
  "success": true,
  "message": "address deleted"
}
```

**Errors:** `400` invalid ID · `404` address not found

---

## Products

GET endpoints are **public** (no auth required). POST, PUT, DELETE require **admin** role.

---

### List Products

`GET /api/products`

Supports full-text search, filtering by category and price range, sorting, and pagination.

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `q` | string | — | Full-text search across name and description |
| `category_id` | UUID | — | Filter by category |
| `min_price` | float | — | Minimum price |
| `max_price` | float | — | Maximum price |
| `sort` | string | — | `price_asc` · `price_desc` · `newest` · `name_asc` · `name_desc` |
| `page` | int | `1` | Page number |
| `limit` | int | `20` | Results per page |

**Response `200`:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "category_id": "uuid",
      "name": "Product Name",
      "description": "...",
      "price": 99.99,
      "image_url": "https://...",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z",
      "category": {
        "id": "uuid",
        "name": "Electronics",
        "slug": "electronics"
      }
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 42
  }
}
```

---

### Get Product

`GET /api/products/:id`

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "category_id": "uuid",
    "name": "Product Name",
    "description": "...",
    "price": 99.99,
    "image_url": "https://...",
    "is_active": true,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

**Errors:** `400` invalid UUID · `404` not found

---

### Create Product

`POST /api/products`  
**Admin only.**

**Request body:**
```json
{
  "category_id": "uuid",
  "name": "Product Name",
  "description": "Product description",
  "price": 99.99,
  "image_url": "https://example.com/image.jpg",
  "is_active": true
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `category_id` | UUID | yes | must exist |
| `name` | string | yes | max 500 chars |
| `description` | string | yes | — |
| `price` | float | yes | greater than 0 |
| `image_url` | string | no | valid URL |
| `is_active` | bool | no | defaults to `true` |

**Response `201`:** same shape as Get Product

**Errors:** `400` validation · `401` unauthenticated · `403` not admin · `404` category not found · `409` product already exists

---

### Update Product

`PUT /api/products/:id`  
**Admin only.**

**Request body:** same as Create Product

**Response `200`:** same shape as Get Product

**Errors:** `400` · `401` · `403` · `404`

---

### Delete Product

`DELETE /api/products/:id`  
**Admin only.**

**Response `200`:**
```json
{
  "success": true,
  "message": "product deleted"
}
```

**Errors:** `400` · `401` · `403` · `404`

---

## Categories

GET is **public**. POST requires **admin** role.

---

### List Categories

`GET /api/categories`

**Response `200`:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "name": "Electronics",
      "slug": "electronics",
      "parent_id": null,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### Create Category

`POST /api/categories`  
**Admin only.**

**Request body:**
```json
{
  "name": "Electronics",
  "slug": "electronics",
  "parent_id": null
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | string | yes | — |
| `slug` | string | yes | must be unique |
| `parent_id` | UUID | no | parent category UUID |

**Response `201`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Electronics",
    "slug": "electronics",
    "parent_id": null,
    "created_at": "2026-01-01T00:00:00Z"
  }
}
```

**Errors:** `400` · `401` · `403` · `409` slug already exists

---

## Cart

All routes require auth. Each user has exactly one cart; it is created automatically on first access. The cart is cleared automatically when an order is placed.

---

### Get Cart

`GET /api/cart`

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "items": [
      {
        "id": "uuid",
        "cart_id": "uuid",
        "product_id": "uuid",
        "product_name": "Product Name",
        "price": 99.99,
        "quantity": 2,
        "subtotal": 199.98,
        "created_at": "2026-01-01T00:00:00Z",
        "updated_at": "2026-01-01T00:00:00Z"
      }
    ],
    "total": 199.98,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

---

### Add Item

`POST /api/cart/items`

If the product is already in the cart, quantity is incremented.

**Request body:**
```json
{
  "product_id": "uuid",
  "quantity": 2
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `product_id` | UUID | yes | must exist and be active |
| `quantity` | int | yes | min 1 |

**Response `200`:** same shape as Get Cart

**Errors:** `400` invalid quantity · `404` product not found · `422` product inactive

---

### Update Item

`PUT /api/cart/items/:id`

`:id` is the cart item UUID (not the product UUID).

**Request body:**
```json
{
  "quantity": 3
}
```

**Response `200`:** same shape as Get Cart

**Errors:** `400` · `404` item not found

---

### Remove Item

`DELETE /api/cart/items/:id`

`:id` is the cart item UUID.

**Response `200`:**
```json
{
  "success": true,
  "message": "item removed from cart"
}
```

**Errors:** `400` · `404` item not found

---

## Orders

All routes require auth.

---

### List Orders

`GET /api/orders`

Returns orders belonging to the authenticated user, newest first.

**Query parameters:**

| Param | Type | Default |
|-------|------|---------|
| `page` | int | `1` |
| `limit` | int | `10` |

**Response `200`:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "status": "pending",
      "total_amount": 199.98,
      "shipping_name": "Jane Doe",
      "shipping_address": "123 Main St, Jakarta",
      "items": [ ... ],
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 5
  }
}
```

**Order status values:** `pending` · `confirmed` · `processing` · `shipped` · `delivered` · `cancelled`

---

### Create Order

`POST /api/orders`

Converts the user's current cart into an order. Reserves inventory, publishes `order.created` to Kafka (which triggers payment-service to create a Stripe PaymentIntent), and clears the cart.

**Request body:**
```json
{
  "shipping_name": "Jane Doe",
  "shipping_address": "123 Main St, Jakarta 12345"
}
```

**Response `201`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "status": "pending",
    "total_amount": 199.98,
    "shipping_name": "Jane Doe",
    "shipping_address": "123 Main St, Jakarta 12345",
    "items": [
      {
        "id": "uuid",
        "order_id": "uuid",
        "product_id": "uuid",
        "product_name": "Product Name",
        "price": 99.99,
        "quantity": 2,
        "subtotal": 199.98,
        "created_at": "2026-01-01T00:00:00Z"
      }
    ],
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

**Errors:** `400` cart is empty · `401` · `500` internal

---

### Get Order

`GET /api/orders/:id`

**Response `200`:** same shape as the object inside List Orders

**Errors:** `400` invalid UUID · `403` not your order · `404` not found

---

### Cancel Order

`PUT /api/orders/:id/cancel`

Only orders with status `pending`, `confirmed`, or `processing` can be cancelled. Releases reserved inventory.

**Response `200`:**
```json
{
  "success": true,
  "data": { ...order with status "cancelled" }
}
```

**Errors:** `400` invalid UUID · `403` · `404` · `409` order cannot be cancelled

---

## Payments

GET endpoints require auth. The Stripe webhook is **public** (Stripe signs its own payload).

---

### Get Payment

`GET /api/payments/:id`

Returns the payment record for the authenticated user.

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "order_id": "uuid",
    "user_id": "uuid",
    "amount": 199.98,
    "currency": "usd",
    "status": "completed",
    "stripe_payment_intent_id": "pi_...",
    "failure_reason": "",
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

**Payment status values:** `pending` · `processing` · `completed` · `failed` · `refunded`

**Errors:** `400` · `403` not your payment · `404`

---

### Get Payment by Order

`GET /api/payments/order/:order_id`

Looks up the payment for a given order. Includes `client_secret` so the frontend can confirm the Stripe PaymentIntent via Stripe.js.

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "order_id": "uuid",
    "user_id": "uuid",
    "amount": 199.98,
    "currency": "usd",
    "status": "pending",
    "stripe_payment_intent_id": "pi_...",
    "client_secret": "pi_..._secret_...",
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

**Frontend payment flow:**
1. Create order → `POST /api/orders`
2. Fetch `client_secret` → `GET /api/payments/order/:order_id`
3. Confirm payment with Stripe.js using `client_secret`
4. Stripe sends webhook → payment status updates to `completed`

**Errors:** `400` · `403` · `404`

---

### Stripe Webhook

`POST /api/payments/webhook/stripe`

Internal endpoint for Stripe event delivery. Do not call this directly.

Stripe signs every request with `Stripe-Signature`. The service verifies the signature using `STRIPE_WEBHOOK_SECRET`. Any non-2xx would cause Stripe to retry — the handler always returns `200`.

**Handled events:**
- `payment_intent.succeeded` → status → `completed`, publishes `payment.completed`
- `payment_intent.payment_failed` → status → `failed`, publishes `payment.failed`
- `payment_intent.processing` → status → `processing`

**Response `200`:**
```json
{ "received": true }
```

---

## Inventory

GET is **public**. PUT requires **admin** role.

---

### Get Inventory

`GET /api/inventory/:product_id`

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "product_id": "uuid",
    "total_quantity": 100,
    "reserved_quantity": 5,
    "available_quantity": 95,
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

`available_quantity = total_quantity - reserved_quantity`

**Errors:** `400` invalid UUID · `404` inventory not found

---

### Set Inventory

`PUT /api/inventory/:product_id`  
**Admin only.**

Sets the total stock for a product. Reserved quantity is managed automatically by the order system.

**Request body:**
```json
{
  "total_quantity": 150
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `total_quantity` | int | yes | min 0 |

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "product_id": "uuid",
    "total_quantity": 150,
    "reserved_quantity": 5,
    "available_quantity": 145,
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

**Errors:** `400` · `401` · `403` · `404`

---

## Health

### Gateway Health

`GET /api/health`

No auth required.

**Response `200`:**
```json
{
  "status": "healthy",
  "service": "auron-api"
}
```

---

## Error Codes

| HTTP Status | Meaning |
|-------------|---------|
| `400` | Bad request — invalid body or query parameters |
| `401` | Unauthenticated — missing or invalid token |
| `403` | Forbidden — authenticated but insufficient permissions |
| `404` | Resource not found |
| `409` | Conflict — duplicate resource (email, slug) or state conflict (order not cancellable) |
| `422` | Unprocessable — business rule violation (e.g. inactive product) |
| `500` | Internal server error |

---

## Authentication Header

Protected endpoints require:
```
Authorization: Bearer <access_token>
```

The gateway validates the JWT (HS256) and injects `X-User-ID` and `X-User-Role` headers before forwarding to downstream services.

---

## Service Ports (direct access, bypass gateway)

| Service | Port |
|---------|------|
| API Gateway | `8080` |
| User Service | `8081` |
| Product Service | `8082` |
| Order Service | `8083` |
| Payment Service | `8084` |
| Inventory Service | `8085` |
| Notification Service | `8086` |
