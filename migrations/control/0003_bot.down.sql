DELETE FROM identities WHERE user_id = 'telegram-bot' AND provider = 'password';
DELETE FROM users WHERE id = 'telegram-bot' AND username = 'telegram-bot' AND role = 'admin';
