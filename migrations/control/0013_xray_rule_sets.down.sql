DELETE FROM routing_policies;
DELETE FROM rule_sets;
ALTER TABLE rule_sets DROP COLUMN geoip_url;
ALTER TABLE rule_sets DROP COLUMN geosite_url;
ALTER TABLE rule_sets DROP COLUMN code;
ALTER TABLE rule_sets ADD COLUMN url TEXT NOT NULL DEFAULT '';
ALTER TABLE rule_sets ADD COLUMN format INTEGER NOT NULL DEFAULT 0;
