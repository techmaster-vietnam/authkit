-- Create password_reset_tokens table
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,
    user_id VARCHAR(12) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE NOT NULL
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_used ON password_reset_tokens(used);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_deleted_at ON password_reset_tokens(deleted_at);

-- Add foreign key constraint (optional, depends on your DB setup)
-- ALTER TABLE password_reset_tokens ADD CONSTRAINT fk_password_reset_tokens_user_id 
--     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

