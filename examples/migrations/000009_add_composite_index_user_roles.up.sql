-- Add composite index on (role_id, user_id) for optimized role-based filtering
-- This index helps optimize queries that filter users by role name
-- Example: SELECT users.* FROM users 
--          INNER JOIN user_roles ON users.id = user_roles.user_id
--          INNER JOIN roles ON user_roles.role_id = roles.id
--          WHERE roles.name LIKE '%admin%'
-- The index allows database to efficiently:
-- 1. Filter roles by name (using UNIQUE index on roles.name)
-- 2. Use this composite index to quickly find all user_ids for matching role_ids
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id_user_id ON user_roles(role_id, user_id);

