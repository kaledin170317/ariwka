UPDATE routing_policies AS p
SET url = r.url,
    format = CASE r.format WHEN 1 THEN 'source' WHEN 2 THEN 'binary' ELSE '' END,
    update_interval_seconds = 86400
FROM rule_sets AS r
WHERE p.rule_set = r.tag;

ALTER TABLE routing_policies DROP COLUMN rule_set;
DROP TABLE rule_sets;
