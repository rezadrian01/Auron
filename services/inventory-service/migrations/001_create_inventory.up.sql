-- Inventory Service Migration: Ensure inventory table exists (shares products_db)
-- Up Migration

-- Create inventory table if it doesn't exist (may be created by product-service)
CREATE TABLE IF NOT EXISTS inventory (
    product_id UUID PRIMARY KEY REFERENCES products(id),
    total_quantity INTEGER NOT NULL DEFAULT 0,
    reserved_quantity INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index for efficient stock queries
CREATE INDEX IF NOT EXISTS idx_inventory_product_id ON inventory(product_id);

-- Create trigger function for updated_at if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc WHERE proname = 'update_inventory_updated_at_column'
    ) THEN
        CREATE OR REPLACE FUNCTION update_inventory_updated_at_column()
        RETURNS TRIGGER AS $$
        BEGIN
            NEW.updated_at = NOW();
            RETURN NEW;
        END;
        $$ language 'plpgsql';
    END IF;
END $$;

-- Create trigger
CREATE TRIGGER IF NOT EXISTS update_inventory_updated_at BEFORE UPDATE ON inventory
    FOR EACH ROW EXECUTE FUNCTION update_inventory_updated_at_column();
