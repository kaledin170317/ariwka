CREATE TABLE node_inbounds (
    tag TEXT PRIMARY KEY,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    type INTEGER NOT NULL,
    port INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    UNIQUE(node_id, port)
);
