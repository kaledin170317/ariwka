CREATE TABLE traffic_usage (
    profile_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id TEXT REFERENCES devices(id) ON DELETE CASCADE,
    default_network BOOLEAN NOT NULL,
    uploaded BIGINT NOT NULL DEFAULT 0,
    downloaded BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX traffic_usage_device_id_idx ON traffic_usage(device_id);
CREATE INDEX traffic_usage_user_id_idx ON traffic_usage(user_id);
