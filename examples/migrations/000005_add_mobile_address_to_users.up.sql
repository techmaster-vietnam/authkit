-- Add mobile and address columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS mobile VARCHAR(15);
ALTER TABLE users ADD COLUMN IF NOT EXISTS address VARCHAR(200);

