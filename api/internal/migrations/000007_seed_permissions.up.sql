-- Insert event permissions
INSERT INTO permissions (id, name, description)
VALUES 
    (gen_random_uuid(), 'events.create', 'Permission to create events'),
    (gen_random_uuid(), 'events.read', 'Permission to read events'),
    (gen_random_uuid(), 'events.update', 'Permission to update events'),
    (gen_random_uuid(), 'events.delete', 'Permission to delete events'),
    (gen_random_uuid(), 'events.admin', 'Permission for events admin');

-- Insert user permissions
INSERT INTO permissions (id, name, description)
VALUES 
    (gen_random_uuid(), 'users.create', 'Permission to create users'),
    (gen_random_uuid(), 'users.read', 'Permission to read users'),
    (gen_random_uuid(), 'users.update', 'Permission to update users'),
    (gen_random_uuid(), 'users.delete', 'Permission to delete users'),
    (gen_random_uuid(), 'users.admin', 'Permission for users admin');