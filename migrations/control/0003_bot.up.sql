INSERT INTO users (id, username, role) VALUES ('telegram-bot', 'telegram-bot', 'admin')
ON CONFLICT(username) DO NOTHING;

INSERT INTO identities (user_id, provider, subject, secret_hash)
SELECT id, 'password', 'telegram-bot', '$2a$10$g/.dylbcJGDpYyJY313pqOtNlcU3ADXYsY5wRa0jaxWj0jtZezjY.'
FROM users WHERE username = 'telegram-bot' AND role = 'admin'
ON CONFLICT(provider, subject) DO UPDATE SET secret_hash = excluded.secret_hash;
