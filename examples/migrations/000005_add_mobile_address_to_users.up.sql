-- Add mobile and address columns to users table
-- Note: These columns are now included in migration 000001, but this migration
-- is kept for databases that were migrated before 000001 was updated.
-- Using IF NOT EXISTS to make it safe for re-running.
ALTER TABLE users ADD COLUMN IF NOT EXISTS mobile VARCHAR(15);
ALTER TABLE users ADD COLUMN IF NOT EXISTS address VARCHAR(200);

