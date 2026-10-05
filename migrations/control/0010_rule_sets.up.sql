CREATE TABLE rule_sets (
    tag TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    format INTEGER NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

INSERT INTO rule_sets (tag, url, format, active)
SELECT tag, url, CASE format WHEN 'source' THEN 1 WHEN 'binary' THEN 2 ELSE 0 END, active
FROM routing_policies WHERE url <> '';

ALTER TABLE routing_policies ADD COLUMN rule_set TEXT NOT NULL DEFAULT '';
UPDATE routing_policies SET rule_set = tag WHERE url <> '';
