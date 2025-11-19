-- Create roles table
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    is_system BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);

