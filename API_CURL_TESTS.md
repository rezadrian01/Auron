# Auron API — curl Test Guide

All requests go through the API Gateway on `http://localhost:8080`.

---

## Setup

```bash
BASE=http://localhost:8080/api
```

Run the commands below **in order** — later steps depend on tokens and IDs from earlier steps.

---

## 1. Auth

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

Expected: `{ "success": true, "data": { "id", "email", "name", "role": "customer" } }`

---

### Register an admin

```bash
curl -s -X POST $BASE/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@auron.test",
    "password": "password123",
    "confirm_password": "password123",
    "name": "Test Admin",
    "role": "admin"
  }' | jq
```

---

### Login as customer — save token

```bash
CUSTOMER_TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "customer@auron.test",
    "password": "password123"
  }' | jq -r '.access_token')

echo "Customer token: $CUSTOMER_TOKEN"
```

---

### Login as admin — save token

```bash
ADMIN_TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@auron.test",
    "password": "password123"
  }' | jq -r '.access_token')

echo "Admin token: $ADMIN_TOKEN"
```

---

### Refresh token

```bash
REFRESH_TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "customer@auron.test",
    "password": "password123"
  }' | jq -r '.refresh_token')

curl -s -X POST $BASE/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}" | jq
```

Expected: `{ "access_token": "...", "refresh_token": "..." }`

---

### Logout

```bash
curl -s -X POST $BASE/auth/logout \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}" | jq
```

Expected: `{ "success": true, "message": "logged out" }`

---

## 2. User Profile

### Get profile

```bash
curl -s -X GET $BASE/users/me \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

Expected: `{ "success": true, "data": { "id", "email", "name", "role" } }`

---

### Update profile

```bash
curl -s -X PUT $BASE/users/me \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ "name": "Updated Customer Name" }' | jq
```

Expected: `{ "success": true, "data": { ... } }`

---

## 3. Addresses

### Add address

```bash
curl -s -X POST $BASE/users/me/addresses \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "label": "Home",
    "street": "123 Main St",
    "city": "Jakarta",
    "country": "Indonesia",
    "postal_code": "10110",
    "is_default": true
  }' | jq
```

Expected: `{ "success": true, "data": { "id", "label", "street", "city", ... } }`

```bash
# Save address ID for later
ADDRESS_ID=$(curl -s -X POST $BASE/users/me/addresses \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "label": "Office",
    "street": "456 Business Ave",
    "city": "Bandung",
    "country": "Indonesia",
    "is_default": false
  }' | jq -r '.data.id')

echo "Address ID: $ADDRESS_ID"
```

---

### Get all addresses

```bash
curl -s -X GET $BASE/users/me/addresses \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

Expected: `{ "success": true, "data": [ ... ] }`

---

### Update address

```bash
curl -s -X PUT $BASE/users/me/addresses/$ADDRESS_ID \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ "city": "Surabaya" }' | jq
```

---

### Delete address

```bash
curl -s -X DELETE $BASE/users/me/addresses/$ADDRESS_ID \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

Expected: `{ "success": true, "message": "address deleted" }`

---

## 4. Categories (admin only for write)

### Get all categories — public

```bash
curl -s -X GET $BASE/categories | jq
```

---

### Create category — admin

```bash
CATEGORY_ID=$(curl -s -X POST $BASE/categories \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Electronics",
    "slug": "electronics"
  }' | jq -r '.data.id')

echo "Category ID: $CATEGORY_ID"
```

### Create sub-category

```bash
curl -s -X POST $BASE/categories \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Smartphones\",
    \"slug\": \"smartphones\",
    \"parent_id\": \"$CATEGORY_ID\"
  }" | jq
```

---

## 5. Products (admin only for write)

### Create product — admin

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

echo "Product ID: $PRODUCT_ID"
```

---

### List products — public

```bash
curl -s -X GET "$BASE/products" | jq
```

---

### List products with filters

```bash
# Search by name
curl -s -X GET "$BASE/products?q=iphone" | jq

# Filter by category
curl -s -X GET "$BASE/products?category_id=$CATEGORY_ID" | jq

# Price range
curl -s -X GET "$BASE/products?min_price=10000000&max_price=20000000" | jq

# Sort + paginate
curl -s -X GET "$BASE/products?sort=price_asc&page=1&limit=5" | jq
```

---

### Get single product — public

```bash
curl -s -X GET "$BASE/products/$PRODUCT_ID" | jq
```

---

### Update product — admin

```bash
curl -s -X PUT "$BASE/products/$PRODUCT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"category_id\": \"$CATEGORY_ID\",
    \"name\": \"iPhone 15 Pro Max\",
    \"description\": \"Apple iPhone 15 Pro Max 512GB\",
    \"price\": 18999000,
    \"is_active\": true
  }" | jq
```

---

### Delete product — admin

```bash
curl -s -X DELETE "$BASE/products/$PRODUCT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

> Re-create the product after deletion for subsequent tests:

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
echo "Product ID: $PRODUCT_ID"
```

---

## 6. Inventory (GET public, PUT admin)

### Get stock — public

```bash
curl -s -X GET "$BASE/inventory/$PRODUCT_ID" | jq
```

Expected: `{ "success": true, "data": { "product_id", "total_quantity", "reserved_quantity", "available_quantity" } }`

---

### Set stock — admin

```bash
curl -s -X PUT "$BASE/inventory/$PRODUCT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ "total_quantity": 50 }' | jq
```

---

## 7. Cart

### Add item to cart

```bash
curl -s -X POST "$BASE/cart/items" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"product_id\": \"$PRODUCT_ID\",
    \"quantity\": 2
  }" | jq
```

---

### Get cart

```bash
curl -s -X GET "$BASE/cart" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

Expected: `{ "success": true, "data": { "id", "items": [...], "total" } }`

```bash
# Save cart item ID
CART_ITEM_ID=$(curl -s -X GET "$BASE/cart" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq -r '.data.items[0].id')
echo "Cart item ID: $CART_ITEM_ID"
```

---

### Update cart item quantity

```bash
curl -s -X PUT "$BASE/cart/items/$CART_ITEM_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ "quantity": 1 }' | jq
```

---

### Remove cart item

```bash
curl -s -X DELETE "$BASE/cart/items/$CART_ITEM_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

> Re-add item for checkout test:

```bash
curl -s -X POST "$BASE/cart/items" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"product_id\": \"$PRODUCT_ID\",
    \"quantity\": 1
  }" | jq
```

---

## 8. Orders

### Place order (clears cart automatically)

```bash
ORDER_ID=$(curl -s -X POST "$BASE/orders" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "shipping_name": "Test Customer",
    "shipping_address": "123 Main St, Jakarta 10110"
  }' | jq -r '.data.id')

echo "Order ID: $ORDER_ID"
```

Expected: `{ "success": true, "data": { "id", "status": "pending", "total_amount", "items": [...] } }`

---

### Get all orders

```bash
curl -s -X GET "$BASE/orders" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq

# With pagination
curl -s -X GET "$BASE/orders?page=1&limit=10" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

---

### Get single order

```bash
curl -s -X GET "$BASE/orders/$ORDER_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

---

### Verify cart was cleared after order

```bash
curl -s -X GET "$BASE/cart" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

Expected: empty items array or cart not found.

---

### Cancel order

```bash
curl -s -X PUT "$BASE/orders/$ORDER_ID/cancel" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

Expected: `{ "success": true, "data": { "status": "cancelled" } }`

> Place a new order for payment tests:

```bash
curl -s -X POST "$BASE/cart/items" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"product_id\": \"$PRODUCT_ID\", \"quantity\": 1}" | jq

ORDER_ID=$(curl -s -X POST "$BASE/orders" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"shipping_name":"Test Customer","shipping_address":"123 Main St"}' \
  | jq -r '.data.id')
echo "New Order ID: $ORDER_ID"
```

---

## 9. Payments

> Payment is created **asynchronously** after the order is placed via Kafka (`order.created` → `payment-service`).
> Wait a few seconds after placing an order before querying payment.

### Get payment by order ID (includes Stripe client_secret)

```bash
sleep 3  # Wait for Kafka event processing

curl -s -X GET "$BASE/payments/order/$ORDER_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

Expected:
```json
{
  "success": true,
  "data": {
    "id": "...",
    "order_id": "...",
    "amount": 15999000,
    "currency": "usd",
    "status": "pending",
    "stripe_payment_intent_id": "pi_...",
    "client_secret": "pi_..._secret_..."
  }
}
```

```bash
# Save payment ID
PAYMENT_ID=$(curl -s -X GET "$BASE/payments/order/$ORDER_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq -r '.data.id')
echo "Payment ID: $PAYMENT_ID"
```

---

### Get payment by ID

```bash
curl -s -X GET "$BASE/payments/$PAYMENT_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

> Note: This endpoint uses `PaymentResponse` (no `client_secret`). Use `/payments/order/:order_id` to get the secret.

---

### Stripe webhook (dev mode — no signature required when STRIPE_WEBHOOK_SECRET is empty)

```bash
curl -s -X POST "$BASE/payments/webhook/stripe" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"evt_test_001\",
    \"type\": \"payment_intent.succeeded\",
    \"data\": {
      \"object\": {
        \"id\": \"pi_test\",
        \"object\": \"payment_intent\",
        \"amount\": 15999000,
        \"currency\": \"usd\",
        \"status\": \"succeeded\",
        \"metadata\": {
          \"payment_id\": \"$PAYMENT_ID\",
          \"order_id\": \"$ORDER_ID\"
        }
      }
    }
  }" | jq
```

Expected: `{ "received": true }`

---

### Verify payment status updated to completed

```bash
curl -s -X GET "$BASE/payments/$PAYMENT_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.data.status'
```

Expected: `"completed"`

---

## 10. Gateway health

```bash
curl -s http://localhost:8080/api/health | jq
```

Expected: `{ "status": "healthy", "service": "auron-api" }`

---

## Error cases

### Unauthenticated request to protected endpoint

```bash
curl -s -X GET $BASE/users/me | jq
```

Expected: 401

---

### Customer accessing admin endpoint

```bash
curl -s -X POST $BASE/categories \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Hack","slug":"hack"}' | jq
```

Expected: 403

---

### Place order with empty cart

```bash
curl -s -X POST "$BASE/orders" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"shipping_name":"Test","shipping_address":"Somewhere"}' | jq
```

Expected: 400 / cart empty error

---

### Get payment before Kafka has processed it

```bash
curl -s -X GET "$BASE/payments/order/$ORDER_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

Expected: 404 (payment not yet created)
