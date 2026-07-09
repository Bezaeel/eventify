-- Create an initial admin user (password: admin123)
INSERT INTO users (id, email, password, first_name, last_name)
VALUES (
    gen_random_uuid(),
    'admin@example.com',
    '$2a$10$9Vvi8tkw5ZeLkoKb0MhTuu2sgnp86XplbVwwfybIy374juKEOLnhC', -- bcrypt hash for 'admin123'
    'Admin',
    'User'
);

-- Assign admin role to admin user
INSERT INTO user_roles (user_id, role_id)
SELECT
    (SELECT id FROM users WHERE email = 'admin@example.com'),
    (SELECT id FROM roles WHERE name = 'admin'); 