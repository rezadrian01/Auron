-- Inventory Service Migration: Drop inventory table
-- Down Migration

DROP TRIGGER IF EXISTS update_inventory_updated_at ON inventory;
DROP FUNCTION IF EXISTS update_inventory_updated_at_column();

DROP INDEX IF EXISTS idx_inventory_product_id;

-- Note: We don't drop the inventory table here as it's shared with product-service
-- In a real scenario, you'd coordinate with product-service or use a migration marker
