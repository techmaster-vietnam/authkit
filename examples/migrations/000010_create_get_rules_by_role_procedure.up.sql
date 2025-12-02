-- Create stored procedure to get rules by role
-- This procedure returns all rules that a specific role can access
-- Parameters:
--   p_role_id: Role ID (INTEGER)
--   p_role_name: Role name (VARCHAR) - if 'super_admin', returns all rules
--   p_service_name: Service name filter (VARCHAR, optional) - NULL for single-app mode
-- Returns: TABLE with all columns from rules table

CREATE OR REPLACE FUNCTION get_rules_by_role(
    p_role_id INTEGER,
    p_role_name VARCHAR,
    p_service_name VARCHAR DEFAULT NULL
)
RETURNS TABLE (
    id VARCHAR(255),
    method VARCHAR(10),
    path VARCHAR(500),
    type VARCHAR(20),
    roles INTEGER[],
    fixed BOOLEAN,
    description TEXT,
    service_name VARCHAR(20)
) AS $$
BEGIN
    -- If role is super_admin, return all rules (with service_name filter if provided)
    IF p_role_name = 'super_admin' THEN
        IF p_service_name IS NULL OR p_service_name = '' THEN
            RETURN QUERY
            SELECT 
                r.id::VARCHAR(255),
                r.method::VARCHAR(10),
                r.path::VARCHAR(500),
                r.type::VARCHAR(20),
                r.roles,
                r.fixed,
                r.description,
                r.service_name::VARCHAR(20)
            FROM rules r
            WHERE r.service_name IS NULL OR r.service_name = '';
        ELSE
            RETURN QUERY
            SELECT 
                r.id::VARCHAR(255),
                r.method::VARCHAR(10),
                r.path::VARCHAR(500),
                r.type::VARCHAR(20),
                r.roles,
                r.fixed,
                r.description,
                r.service_name::VARCHAR(20)
            FROM rules r
            WHERE r.service_name = p_service_name;
        END IF;
        RETURN;
    END IF;
    
    -- Otherwise, return:
    -- 1. All rules with type = 'PUBLIC'
    -- 2. All rules with type = 'ALLOW' and roles = [] (empty array)
    -- 3. All rules with type = 'ALLOW' and roles contains p_role_id
    IF p_service_name IS NULL OR p_service_name = '' THEN
        RETURN QUERY
        SELECT 
            r.id::VARCHAR(255),
            r.method::VARCHAR(10),
            r.path::VARCHAR(500),
            r.type::VARCHAR(20),
            r.roles,
            r.fixed,
            r.description,
            r.service_name::VARCHAR(20)
        FROM rules r
        WHERE (r.service_name IS NULL OR r.service_name = '')
          AND (
              r.type = 'PUBLIC'
              OR (r.type = 'ALLOW' AND (cardinality(r.roles) = 0 OR p_role_id = ANY(r.roles)))
          );
    ELSE
        RETURN QUERY
        SELECT 
            r.id::VARCHAR(255),
            r.method::VARCHAR(10),
            r.path::VARCHAR(500),
            r.type::VARCHAR(20),
            r.roles,
            r.fixed,
            r.description,
            r.service_name::VARCHAR(20)
        FROM rules r
        WHERE r.service_name = p_service_name
          AND (
              r.type = 'PUBLIC'
              OR (r.type = 'ALLOW' AND (cardinality(r.roles) = 0 OR p_role_id = ANY(r.roles)))
          );
    END IF;
END;
$$ LANGUAGE plpgsql;

