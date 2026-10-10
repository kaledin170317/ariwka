CREATE TABLE devices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL
);

ALTER TABLE profiles ADD COLUMN device_id TEXT REFERENCES devices(id) ON DELETE CASCADE;
CREATE INDEX profiles_device_id_idx ON profiles(device_id);
CREATE INDEX devices_user_id_idx ON devices(user_id);
