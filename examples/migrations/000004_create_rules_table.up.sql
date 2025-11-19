-- Create rules table
CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(255) PRIMARY KEY,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    type VARCHAR(20) NOT NULL,
    roles INTEGER[] DEFAULT '{}',
    fixed BOOLEAN DEFAULT FALSE,
    description TEXT,
    CONSTRAINT idx_method_path UNIQUE (method, path)
);

-- Create index on method and path for faster lookups
CREATE INDEX IF NOT EXISTS idx_rules_method_path ON rules(method, path);

