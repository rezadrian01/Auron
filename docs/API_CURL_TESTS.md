# Auron API — curl Test Guide

All requests go through the API Gateway on `http://localhost:8080`.

> **Tested on:** 2026-05-28 against the full docker-compose stack.
> All endpoints verified working unless noted.

---

## Setup

```bash
BASE=http://localhost:8080/api
```

Run the commands below **in order** — later steps depend on tokens and IDs from earlier steps.

---

## 1. Gateway Health

```bash
curl -s http://localhost:8080/api/health | jq
```

**Response:**
```json
{ "service": "auron-api", "status": "healthy" }
```

---

## 2. Auth

### Register a customer

```bash
curl -s -X POST $BASE/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "customer@auron.test",
    "password": "password123",
    "confirm_password": "password123",
    "name": "Test Customer"
  }' | jq
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "email": "customer@auron.test", "name": "Test Customer", "role": "customer" },
  "success": true
}
```

> **Note:** `role` field in register request is ignored for security — all new accounts are `customer`.
> To create an admin, update the role directly in the DB:
> ```bash
> docker exec <users-db-container> psql -U auron -d users_db \
>   -c "UPDATE users SET role='admin' WHERE email='admin@auron.test';"
> ```

---

### Login — save tokens

```bash
CUSTOMER_TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "customer@auron.test", "password": "password123"}' \
  | jq -r '.access_token')

REFRESH_TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "customer@auron.test", "password": "password123"}' \
  | jq -r '.refresh_token')

ADMIN_TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@auron.test", "password": "password123"}' \
  | jq -r '.access_token')
```

**Response:**
```json
{
  "access_token": "eyJhbGci...",
  "refresh_token": "eyJhbGci..."
}
```

---

### Refresh token

```bash
curl -s -X POST $BASE/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}" | jq
```

**Response:**
```json
{ "access_token": "eyJhbGci...", "refresh_token": "eyJhbGci..." }
```

---

### Logout

```bash
curl -s -X POST $BASE/auth/logout \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}" | jq
```

**Response:**
```json
{ "success": true, "message": "logged out" }
```

---

## 3. User Profile

### Get profile

```bash
curl -s $BASE/users/me -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "email": "customer@auron.test", "name": "Test Customer", "role": "customer" },
  "success": true
}
```

---

### Update profile

```bash
curl -s -X PUT $BASE/users/me \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Name"}' | jq
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "email": "customer@auron.test", "name": "Updated Name", "role": "customer" },
  "success": true
}
```

---

## 4. Addresses

### Add address — save ID

```bash
ADDRESS_ID=$(curl -s -X POST $BASE/users/me/addresses \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "label": "Home",
    "street": "123 Main St",
    "city": "Jakarta",
    "country": "Indonesia",
    "postal_code": "10110",
    "is_default": true
  }' | jq -r '.data.id')
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "label": "Home", "street": "123 Main St", "city": "Jakarta", "country": "Indonesia", "postal_code": "10110", "is_default": true },
  "success": true
}
```

---

### Get all addresses

```bash
curl -s $BASE/users/me/addresses -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

**Response:**
```json
{
  "data": [{ "id": "<uuid>", "label": "Home", "street": "123 Main St", "city": "Jakarta", ... }],
  "success": true
}
```

---

### Update address

```bash
curl -s -X PUT $BASE/users/me/addresses/$ADDRESS_ID \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"city": "Surabaya"}' | jq
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "city": "Surabaya", ... },
  "success": true
}
```

---

### Delete address

```bash
curl -s -X DELETE $BASE/users/me/addresses/$ADDRESS_ID \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

**Response:**
```json
{ "success": true, "message": "address deleted" }
```

---

## 5. Categories

### Get all — public

```bash
curl -s $BASE/categories | jq
```

**Response:**
```json
{
  "data": [{ "id": "<uuid>", "name": "Electronics", "slug": "electronics", "created_at": "..." }],
  "success": true
}
```

---

### Create — admin only, save ID

```bash
CATEGORY_ID=$(curl -s -X POST $BASE/categories \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Electronics", "slug": "electronics"}' | jq -r '.data.id')
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "name": "Electronics", "slug": "electronics", "created_at": "..." },
  "success": true
}
```

---

## 6. Products

### Create — admin only, save ID

```bash
PRODUCT_ID=$(curl -s -X POST $BASE/products \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"category_id\": \"$CATEGORY_ID\",
    \"name\": \"iPhone 15 Pro\",
    \"description\": \"Apple iPhone 15 Pro 256GB\",
    \"price\": 15999000,
    \"is_active\": true
  }" | jq -r '.data.id')
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "name": "iPhone 15 Pro", "price": 15999000, "is_active": true, ... },
  "success": true
}
```

---

### List — public, with filters

```bash
# All products
curl -s "$BASE/products" | jq '.data[0] | {id,name,price}'

# Full-text search
curl -s "$BASE/products?q=iphone" | jq '{total: .meta.total, first: .data[0].name}'

# Filter + sort + paginate
curl -s "$BASE/products?category_id=$CATEGORY_ID&sort=price_asc&page=1&limit=5" | jq
```

**Response (list):**
```json
{
  "data": [{ "id": "<uuid>", "name": "iPhone 15 Pro", "price": 15999000, "is_active": true, ... }],
  "meta": { "page": 1, "limit": 20, "total": 1 },
  "success": true
}
```

---

### Get by ID — public

```bash
curl -s "$BASE/products/$PRODUCT_ID" | jq '.data | {id,name,price}'
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "name": "iPhone 15 Pro", "price": 15999000, ... },
  "success": true
}
```

---

### Update — admin only

```bash
curl -s -X PUT "$BASE/products/$PRODUCT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"category_id\":\"$CATEGORY_ID\",\"name\":\"iPhone 15 Pro Max\",\"description\":\"512GB\",\"price\":18999000,\"is_active\":true}" | jq '.data | {name,price}'
```

**Response:**
```json
{ "data": { "name": "iPhone 15 Pro Max", "price": 18999000, ... }, "success": true }
```

---

### Delete — admin only

```bash
curl -s -X DELETE "$BASE/products/$PRODUCT_ID" -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

**Response:**
```json
{ "success": true, "message": "product deleted" }
```

---

## 7. Inventory

### Get stock — **public, no auth required**

```bash
curl -s "$BASE/inventory/$PRODUCT_ID" | jq
```

**Response:**
```json
{
  "data": { "product_id": "<uuid>", "total_quantity": 50, "reserved_quantity": 0, "available_quantity": 50, "updated_at": "..." },
  "success": true
}
```

> Returns 404 if inventory has never been set for this product.

---

### Set stock — admin only

```bash
curl -s -X PUT "$BASE/inventory/$PRODUCT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"total_quantity": 50}' | jq
```

**Response:**
```json
{
  "data": { "product_id": "<uuid>", "total_quantity": 50, "reserved_quantity": 0, "available_quantity": 50, "updated_at": "..." },
  "success": true
}
```

---

## 8. Cart

### Add item

```bash
curl -s -X POST "$BASE/cart/items" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"product_id\": \"$PRODUCT_ID\", \"quantity\": 2}" | jq '.data | {total, items_count: (.items|length)}'
```

**Response:**
```json
{ "data": { "total": 31998000, "items": [...] }, "success": true }
```

---

### Get cart — save item ID

```bash
CART_ITEM_ID=$(curl -s "$BASE/cart" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq -r '.data.items[0].id')
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "user_id": "<uuid>", "items": [{ "id": "<uuid>", "product_id": "<uuid>", "quantity": 2, "price": 15999000, "subtotal": 31998000, ... }], "total": 31998000 },
  "success": true
}
```

---

### Update item quantity

```bash
curl -s -X PUT "$BASE/cart/items/$CART_ITEM_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"quantity": 1}' | jq '.data | {total, new_qty: .items[0].quantity}'
```

**Response:**
```json
{ "data": { "total": 15999000, "items": [{ "quantity": 1, ... }] }, "success": true }
```

---

### Remove item

```bash
curl -s -X DELETE "$BASE/cart/items/$CART_ITEM_ID" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

**Response:**
```json
{ "success": true, "message": "item removed from cart" }
```

---

## 9. Orders

### Place order — clears cart automatically

```bash
# Re-add item first
curl -s -X POST "$BASE/cart/items" -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"product_id\": \"$PRODUCT_ID\", \"quantity\": 1}" > /dev/null

ORDER_ID=$(curl -s -X POST "$BASE/orders" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"shipping_name": "Test Customer", "shipping_address": "123 Main St, Jakarta 10110"}' \
  | jq -r '.data.id')
```

**Response:**
```json
{
  "data": {
    "id": "<uuid>", "status": "pending", "total_amount": 15999000,
    "items": [{ "product_id": "<uuid>", "product_name": "iPhone 15 Pro", "quantity": 1, "price": 15999000, "subtotal": 15999000 }],
    "shipping_name": "Test Customer", "shipping_address": "123 Main St, Jakarta 10110"
  },
  "success": true
}
```

---

### Verify cart was cleared

```bash
curl -s "$BASE/cart" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.data | {total, items_count: (.items|length)}'
```

**Response:** `{ "total": 0, "items_count": 0 }` ✅

---

### Get all orders

```bash
curl -s "$BASE/orders" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '{total: .meta.total}'
```

**Response:**
```json
{ "data": [...], "meta": { "page": 1, "limit": 10, "total": 1 }, "success": true }
```

---

### Get order by ID

```bash
curl -s "$BASE/orders/$ORDER_ID" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.data | {id,status,total_amount}'
```

**Response:**
```json
{ "data": { "id": "<uuid>", "status": "pending", "total_amount": 15999000 }, "success": true }
```

---

### Cancel order

```bash
curl -s -X PUT "$BASE/orders/$ORDER_ID/cancel" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.data | {id,status}'
```

**Response:**
```json
{ "data": { "id": "<uuid>", "status": "cancelled" }, "success": true }
```

---

## 10. Payments

> Payment is created **asynchronously** after `POST /orders` via Kafka (`order.created` → payment-service).
> The payment-service calls Stripe to create a PaymentIntent and stores the `client_secret`.

### Get payment by order ID — includes Stripe `client_secret`

```bash
# Place a fresh order first, then wait ~1-3s for Kafka
sleep 3

curl -s "$BASE/payments/order/$ORDER_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.data | {id,status,amount,currency,stripe_payment_intent_id,client_secret}'
```

**Response:**
```json
{
  "data": {
    "id": "<payment-uuid>",
    "order_id": "<order-uuid>",
    "user_id": "<user-uuid>",
    "status": "pending",
    "amount": 15999000,
    "currency": "usd",
    "stripe_payment_intent_id": "pi_3Tc0zrRr2KxYotum0gxE3rT4",
    "client_secret": "pi_3Tc0zrRr2KxYotum0gxE3rT4_secret_...",
    "created_at": "..."
  },
  "success": true
}
```

> Use `client_secret` in the frontend with `stripe.confirmPayment()` to complete the payment.
> Requesting before Kafka processes → `404 payment not found`.

---

### Get payment by ID — no `client_secret`

```bash
curl -s "$BASE/payments/$PAYMENT_ID" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.data | {id,status,amount}'
```

**Response:**
```json
{
  "data": { "id": "<uuid>", "status": "pending", "amount": 15999000, "currency": "usd", ... },
  "success": true
}
```

> `client_secret` is omitted from this response. Use `/payments/order/:order_id` for checkout.

---

### Stripe webhook — full end-to-end test

```bash
# 1. Start the Stripe CLI listener (run once in a separate terminal)
stripe listen --forward-to localhost:8080/api/payments/webhook/stripe
# Copy the whsec_... secret into STRIPE_WEBHOOK_SECRET env var and restart payment-service

# 2. Place an order and get the PaymentIntent ID
curl -s -X POST "$BASE/cart/items" -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" -d "{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}" > /dev/null

ORDER_ID=$(curl -s -X POST "$BASE/orders" -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"shipping_name":"Test","shipping_address":"123 St"}' | jq -r '.data.id')

sleep 3  # wait for Kafka

PAYMENT=$(curl -s "$BASE/payments/order/$ORDER_ID" -H "Authorization: Bearer $CUSTOMER_TOKEN")
PI=$(echo $PAYMENT | jq -r '.data.stripe_payment_intent_id')
PAYMENT_ID=$(echo $PAYMENT | jq -r '.data.id')

# 3. Confirm the PaymentIntent using the Stripe CLI with a test card
stripe payment_intents confirm $PI --payment-method=pm_card_visa

# 4. Verify status updated to completed
sleep 3
curl -s "$BASE/payments/$PAYMENT_ID" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.data | {id, status}'
```

**Response after confirmation:**
```json
{ "data": { "id": "<uuid>", "status": "completed" }, "success": true }
```

**Notes:**
- Always returns `{ "received": true }` with HTTP 200 (Stripe retries on non-2xx)
- Handles: `payment_intent.succeeded` → status `completed`, `payment_intent.payment_failed` → status `failed`
- PaymentIntent created with `allow_redirects: never` so no `return_url` is needed at confirmation
- stripe-go v76 uses API `2023-10-16`; `IgnoreAPIVersionMismatch: true` set so CLI events (API `2024-04-10`) are accepted
- In dev: set `STRIPE_WEBHOOK_SECRET=` (empty) to skip signature verification entirely

---

## 11. Error Cases

### No auth token → 401

```bash
curl -s $BASE/users/me | jq
```

```json
{ "success": false, "error": { "code": "UNAUTHORIZED", "message": "Authorization header is required" } }
```

---

### Customer accessing admin endpoint → 403

```bash
curl -s -X POST $BASE/categories \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Hack","slug":"hack"}' | jq
```

```json
{ "success": false, "error": { "code": "FORBIDDEN", "message": "Insufficient permissions" } }
```

---

### Order with empty cart → 400

```bash
curl -s -X POST "$BASE/orders" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"shipping_name":"Test","shipping_address":"Somewhere"}' | jq
```

```json
{ "success": false, "error": "cart is empty" }
```

---

### Payment not yet processed by Kafka → 404

```bash
# Immediately after POST /orders (before Kafka processes)
curl -s "$BASE/payments/order/$ORDER_ID" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

```json
{ "success": false, "error": "payment not found" }
```

---

## Summary — Endpoint Status

| Endpoint | Method | Auth | Status |
|----------|--------|------|--------|
| `/api/health` | GET | ❌ | ✅ |
| `/api/auth/register` | POST | ❌ | ✅ |
| `/api/auth/login` | POST | ❌ | ✅ |
| `/api/auth/refresh` | POST | ❌ | ✅ |
| `/api/auth/logout` | POST | JWT | ✅ |
| `/api/users/me` | GET | JWT | ✅ |
| `/api/users/me` | PUT | JWT | ✅ |
| `/api/users/me/addresses` | GET | JWT | ✅ |
| `/api/users/me/addresses` | POST | JWT | ✅ |
| `/api/users/me/addresses/:id` | PUT | JWT | ✅ |
| `/api/users/me/addresses/:id` | DELETE | JWT | ✅ |
| `/api/categories` | GET | ❌ | ✅ |
| `/api/categories` | POST | Admin | ✅ |
| `/api/products` | GET | ❌ | ✅ |
| `/api/products` | POST | Admin | ✅ |
| `/api/products/:id` | GET | ❌ | ✅ |
| `/api/products/:id` | PUT | Admin | ✅ |
| `/api/products/:id` | DELETE | Admin | ✅ |
| `/api/inventory/:product_id` | GET | ❌ | ✅ |
| `/api/inventory/:product_id` | PUT | Admin | ✅ |
| `/api/cart` | GET | JWT | ✅ |
| `/api/cart/items` | POST | JWT | ✅ |
| `/api/cart/items/:id` | PUT | JWT | ✅ |
| `/api/cart/items/:id` | DELETE | JWT | ✅ |
| `/api/orders` | GET | JWT | ✅ |
| `/api/orders` | POST | JWT | ✅ |
| `/api/orders/:id` | GET | JWT | ✅ |
| `/api/orders/:id/cancel` | PUT | JWT | ✅ |
| `/api/payments/order/:order_id` | GET | JWT | ✅ |
| `/api/payments/:id` | GET | JWT | ✅ |
| `/api/payments/webhook/stripe` | POST | Stripe-signed | ✅ (verified end-to-end with CLI confirm) |
