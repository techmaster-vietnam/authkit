-- Create rules table
CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(255) PRIMARY KEY,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    type VARCHAR(20) NOT NULL,
    roles INTEGER[] DEFAULT '{}',
    fixed BOOLEAN DEFAULT FALSE,
    description TEXT,
    service_name VARCHAR(20)
);

-- Create index on method and path for faster lookups
CREATE INDEX IF NOT EXISTS idx_rules_method_path ON rules(method, path);

-- Create index for faster filtering by service_name
CREATE INDEX IF NOT EXISTS idx_rules_service_name ON rules(service_name);

-- Create partial unique index for rules without service_name (single-app mode)
-- This ensures (method, path) is unique when service_name IS NULL
CREATE UNIQUE INDEX IF NOT EXISTS idx_method_path_null_service 
    ON rules(method, path) 
    WHERE service_name IS NULL;

-- Create unique index for rules with service_name (microservice mode)
-- This ensures (service_name, method, path) is unique when service_name IS NOT NULL
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_method_path 
    ON rules(service_name, method, path) 
    WHERE service_name IS NOT NULL;

