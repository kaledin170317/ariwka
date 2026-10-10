-- Old sing-box rule sets and their policies cannot be applied to Xray.
DELETE FROM routing_policies;
DELETE FROM rule_sets;

ALTER TABLE rule_sets DROP COLUMN url;
ALTER TABLE rule_sets DROP COLUMN format;
ALTER TABLE rule_sets ADD COLUMN geoip_url TEXT NOT NULL DEFAULT '';
ALTER TABLE rule_sets ADD COLUMN geosite_url TEXT NOT NULL DEFAULT '';
ALTER TABLE rule_sets ADD COLUMN code TEXT NOT NULL DEFAULT '';
