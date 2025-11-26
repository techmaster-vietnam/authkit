-- Create stored procedure to delete a role
-- This procedure:
-- 1. Deletes all records from user_roles where role_id = delete_role_id
-- 2. Scans all records in rules table, removes delete_role_id from roles array if present
-- 3. Finally deletes the record from roles where id = delete_role_id

CREATE OR REPLACE FUNCTION delete_role(delete_role_id INTEGER)
RETURNS VOID AS $$
BEGIN
    -- Step 1: Delete all records from user_roles table
    DELETE FROM user_roles WHERE role_id = delete_role_id;
    
    -- Step 2: Update rules table - remove delete_role_id from roles array
    -- Using array_remove to remove all occurrences of delete_role_id from the array
    UPDATE rules 
    SET roles = array_remove(roles, delete_role_id::INTEGER)
    WHERE delete_role_id::INTEGER = ANY(roles);
    
    -- Step 3: Delete the role from roles table
    DELETE FROM roles WHERE id = delete_role_id;
END;
$$ LANGUAGE plpgsql;

