-- Payment Service Migration: Drop payments table
-- Down Migration

DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_payments_user_id;
DROP INDEX IF EXISTS idx_payments_stripe_pi;
DROP INDEX IF EXISTS idx_payments_order_id;

DROP TABLE IF EXISTS payments;
