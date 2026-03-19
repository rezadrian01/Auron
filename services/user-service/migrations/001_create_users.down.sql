-- User Service Migration: Drop users and addresses tables
-- Down Migration

DROP INDEX IF EXISTS idx_addresses_user_id;
DROP INDEX IF EXISTS idx_users_email;

DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS users;
