DELETE FROM identities WHERE user_id = 'admin' AND provider = 'password' AND subject = 'admin';
DELETE FROM users WHERE id = 'admin' AND username = 'admin' AND role = 'admin';
