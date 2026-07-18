-- Remove admin user and their role
DELETE FROM user_roles WHERE user_id = (SELECT id FROM users WHERE email = 'admin@example.com');
DELETE FROM users WHERE email = 'admin@example.com'; 