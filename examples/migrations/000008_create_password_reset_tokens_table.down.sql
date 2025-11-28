-- Drop password_reset_tokens table
DROP INDEX IF EXISTS idx_password_reset_tokens_deleted_at;
DROP INDEX IF EXISTS idx_password_reset_tokens_used;
DROP INDEX IF EXISTS idx_password_reset_tokens_expires_at;
DROP INDEX IF EXISTS idx_password_reset_tokens_user_id;
DROP TABLE IF EXISTS password_reset_tokens;

