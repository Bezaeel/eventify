-- Assign all permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE name = 'admin'),
    id
FROM permissions;

-- Assign event permissions to event_manager role
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE name = 'event_manager'),
    id
FROM permissions
WHERE name LIKE 'events.%';

-- Assign user permissions to user_manager role
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE name = 'user_manager'),
    id
FROM permissions
WHERE name LIKE 'users.%';

-- Assign read permissions to viewer role
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE name = 'viewer'),
    id
FROM permissions
WHERE name LIKE '%.read'; 