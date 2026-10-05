ALTER TABLE rule_sets
    ADD COLUMN update_interval_seconds INTEGER NOT NULL DEFAULT 86400;
