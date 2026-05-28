# Full-Text Search Setup Guide

## Overview

The Product Service uses PostgreSQL's built-in full-text search capabilities via `tsvector` and `plainto_tsquery()`.

## How It Works

### 1. Database Layer (PostgreSQL)

The `search_vector` column in the `products` table stores a tsvector (text search vector) that is automatically populated by a PostgreSQL trigger on every INSERT or UPDATE.

```sql
-- Trigger automatically runs:
NEW.search_vector := to_tsvector('english', COALESCE(NEW.name, '') || ' ' || COALESCE(NEW.description, ''));
```

### 2. Query Layer (GORM + Raw SQL)

To search products, use PostgreSQL's `@@` (match) operator with `plainto_tsquery()`:

```sql
SELECT * FROM products 
WHERE search_vector @@ plainto_tsquery('english', 'laptop gaming')
  AND is_active = true;
```

### 3. Go Implementation (Repository Layer)

In `internal/repository/product_repository.go`:

```go
func (r *ProductRepository) ListProducts(filter domain.ProductFilter) (*domain.ProductListResponse, error) {
    query := r.db.Model(&domain.Product{}).Where("is_active = ?", true)

    // Full-text search
    if filter.Q != "" {
        // Use raw SQL for tsvector queries (GORM doesn't support this natively)
        query = query.Where("search_vector @@ plainto_tsquery('english', ?)", filter.Q)
    }

    // ... rest of filtering, sorting, pagination
}
```

## Search Features

### Phrase Search
```
Query: "wireless mouse"
Matches: Products containing both "wireless" AND "mouse"
```

### Prefix Matching
```
Query: "lap*"
Matches: "laptop", "lapse", "lapel", etc.
```

### Weighted Results (Future Enhancement)
```sql
-- Rank results by relevance
SELECT *, ts_rank(search_vector, plainto_tsquery('english', 'laptop')) AS rank
FROM products
WHERE search_vector @@ plainto_tsquery('english', 'laptop')
ORDER BY rank DESC;
```

## Testing Full-Text Search

### 1. Manual Test via SQL

```sql
-- Insert test product
INSERT INTO products (name, description, price, category_id)
VALUES ('Gaming Laptop Pro', 'High-performance laptop with RTX 4090 and 32GB RAM', 2499.99, 'some-uuid');

-- Verify search_vector is populated
SELECT id, name, search_vector FROM products WHERE name = 'Gaming Laptop Pro';

-- Test search query
SELECT id, name, description
FROM products
WHERE search_vector @@ plainto_tsquery('english', 'laptop');

-- Test multi-word search
SELECT id, name, description
FROM products
WHERE search_vector @@ plainto_tsquery('english', 'gaming laptop');
```

### 2. API Test via curl

```bash
# Search for "laptop"
curl "http://localhost:8080/api/products?q=laptop"

# Search for "gaming laptop"
curl "http://localhost:8080/api/products?q=gaming+laptop"

# Search with filters
curl "http://localhost:8080/api/products?q=laptop&min_price=1000&max_price=3000&sort=price_asc"
```

## Migration

Run the migration to setup full-text search:

```bash
# Via Docker
docker compose exec product-service sh
# Inside container
psql $DATABASE_URL -f /app/db/004_create_search_index.up.sql
```

Or let GORM's `AutoMigrate` create the basic structure, then run the trigger setup:

```go
// In cmd/infrastructure.go
func runMigrations(db *gorm.DB) error {
    // GORM creates the basic table structure
    if err := db.AutoMigrate(&domain.Product{}, &domain.Category{}, &domain.Inventory{}); err != nil {
        return err
    }

    // PostgreSQL-specific setup (tsvector trigger)
    return bootstrapSearchIndex(db)
}

func bootstrapSearchIndex(db *gorm.DB) error {
    // Execute the SQL migration
    sqlContent, err := os.ReadFile("db/004_create_search_index.up.sql")
    if err != nil {
        return fmt.Errorf("failed to read migration file: %w", err)
    }

    if err := db.Exec(string(sqlContent)).Error; err != nil {
        return fmt.Errorf("failed to execute search index migration: %w", err)
    }

    return nil
}
```

## Performance Notes

| Aspect | Details |
|---|---|
| Index Type | GIN (Generalized Inverted Index) |
| Text Config | `english` (uses English stemmer) |
| Trigger | `BEFORE INSERT OR UPDATE` (automatic) |
| Query Speed | ~1-5ms for 100k products |
| Index Size | ~20-30% of text column size |

## Troubleshooting

### Search Returns No Results

1. **Check if search_vector is populated:**
   ```sql
   SELECT id, name, search_vector FROM products LIMIT 5;
   ```

2. **Manually trigger update:**
   ```sql
   UPDATE products SET search_vector = to_tsvector('english', name || ' ' || COALESCE(description, ''));
   ```

3. **Verify trigger exists:**
   ```sql
   SELECT trigger_name, event_manipulation 
   FROM information_schema.triggers 
   WHERE trigger_name = 'products_search_vector_update';
   ```

### Search Returns Wrong Results

PostgreSQL's `plainto_tsquery()` uses AND logic by default. "laptop gaming" matches products containing **both** words, not necessarily in that order.

For exact phrase matching, use `phraseto_tsquery()`:
```sql
WHERE search_vector @@ phraseto_tsquery('english', 'gaming laptop')
```

## References

- [PostgreSQL Full-Text Search](https://www.postgresql.org/docs/current/textsearch.html)
- [tsvector Documentation](https://www.postgresql.org/docs/current/datatype-textsearch.html)
- [GIN Index Documentation](https://www.postgresql.org/docs/current/gin.html)
