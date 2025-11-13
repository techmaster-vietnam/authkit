-- Create rules table
CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(255) PRIMARY KEY,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    type VARCHAR(20) NOT NULL,
    roles TEXT, -- JSON array stored as text
    UNIQUE(method, path)
);

CREATE INDEX IF NOT EXISTS idx_method_path ON rules(method, path);
