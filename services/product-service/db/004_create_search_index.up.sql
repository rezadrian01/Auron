-- Migration: 004_create_search_index
-- Purpose: Setup PostgreSQL full-text search with tsvector and automatic triggers

-- Add search_vector column if it doesn't exist (GORM may have created it)
ALTER TABLE products ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create GIN index for full-text search
CREATE INDEX IF NOT EXISTS idx_products_search ON products USING GIN(search_vector);

-- Create function to update search_vector on product changes
CREATE OR REPLACE FUNCTION products_search_vector_trigger() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('english', COALESCE(NEW.name, '') || ' ' || COALESCE(NEW.description, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop existing trigger if it exists
DROP TRIGGER IF EXISTS products_search_vector_update ON products;

-- Create trigger to auto-populate search_vector on INSERT/UPDATE
CREATE TRIGGER products_search_vector_update
    BEFORE INSERT OR UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION products_search_vector_trigger();

-- Populate search_vector for existing records
UPDATE products
SET search_vector = to_tsvector('english', COALESCE(name, '') || ' ' || COALESCE(description, ''))
WHERE search_vector IS NULL OR search_vector = '';

-- Add index on category_id for faster filtering
CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);

-- Add index on price for range queries
CREATE INDEX IF NOT EXISTS idx_products_price ON products(price);

-- Add index on is_active for filtering
CREATE INDEX IF NOT EXISTS idx_products_is_active ON products(is_active);

-- Add index on created_at for sorting
CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC);
