INSERT INTO users (id, username, role) VALUES ('admin', 'admin', 'admin')
ON CONFLICT(username) DO NOTHING;

INSERT INTO identities (user_id, provider, subject, secret_hash)
SELECT id, 'password', 'admin', '$2a$10$mRJjVEg5rlRM1v/yUybnluOgWLAaGkl1dwRvt7PLHjollZNZY77D6'
FROM users WHERE username = 'admin' AND role = 'admin'
ON CONFLICT(provider, subject) DO UPDATE SET secret_hash = excluded.secret_hash;
