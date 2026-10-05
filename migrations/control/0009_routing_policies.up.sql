CREATE TABLE routing_policies (
    tag TEXT PRIMARY KEY,
    url TEXT NOT NULL DEFAULT '',
    format TEXT NOT NULL DEFAULT '',
    update_interval_seconds INTEGER NOT NULL DEFAULT 0,
    source_country TEXT NOT NULL DEFAULT '',
    source_not_country TEXT NOT NULL DEFAULT '',
    target_country TEXT NOT NULL DEFAULT '',
    target_not_country TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE
);
