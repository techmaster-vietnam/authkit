-- Remove mobile and address columns from users table
-- Note: This is a rollback migration for version 5
ALTER TABLE users DROP COLUMN IF EXISTS mobile;
ALTER TABLE users DROP COLUMN IF EXISTS address;

