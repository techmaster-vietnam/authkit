-- Drop refresh_tokens table
DROP INDEX IF EXISTS idx_refresh_tokens_deleted_at;
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP INDEX IF EXISTS idx_refresh_tokens_token;
DROP TABLE IF EXISTS refresh_tokens;

