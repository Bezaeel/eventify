-- Create admin role
INSERT INTO roles (id, name, description)
VALUES (gen_random_uuid(), 'admin', 'Administrator with all permissions');

-- Create event manager role
INSERT INTO roles (id, name, description)
VALUES (gen_random_uuid(), 'event_manager', 'Can manage events');

-- Create user manager role
INSERT INTO roles (id, name, description)
VALUES (gen_random_uuid(), 'user_manager', 'Can manage users');

-- Create read-only role
INSERT INTO roles (id, name, description)
VALUES (gen_random_uuid(), 'viewer', 'Read-only access'); 