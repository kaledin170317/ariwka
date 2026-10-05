ALTER TABLE rule_sets
    ALTER COLUMN format TYPE INTEGER USING CASE format::text
        WHEN 'source' THEN 1
        WHEN 'binary' THEN 2
        WHEN '' THEN 0
        ELSE format::text::integer
    END;

ALTER TABLE rule_sets DROP COLUMN IF EXISTS update_interval_seconds;
