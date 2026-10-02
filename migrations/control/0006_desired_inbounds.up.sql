ALTER TABLE node_inbounds ADD COLUMN server_name TEXT NOT NULL DEFAULT '';
ALTER TABLE node_inbounds ADD COLUMN transport_type INTEGER NOT NULL DEFAULT 0;
ALTER TABLE node_inbounds ADD COLUMN transport_host TEXT NOT NULL DEFAULT '';
ALTER TABLE node_inbounds ADD COLUMN transport_path TEXT NOT NULL DEFAULT '';
ALTER TABLE node_inbounds ADD COLUMN service_name TEXT NOT NULL DEFAULT '';
ALTER TABLE node_inbounds ADD COLUMN private_key TEXT NOT NULL DEFAULT '';
ALTER TABLE node_inbounds ADD COLUMN short_id TEXT NOT NULL DEFAULT '';
ALTER TABLE profiles ADD COLUMN password TEXT NOT NULL DEFAULT '';
