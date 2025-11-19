-- Remove mobile and address columns from users table
ALTER TABLE users DROP COLUMN IF EXISTS mobile;
ALTER TABLE users DROP COLUMN IF EXISTS address;

