ALTER TABLE routing_policies
    ADD COLUMN apply_to_nodes BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN apply_to_clients BOOLEAN NOT NULL DEFAULT FALSE;

-- This is intentionally a separate client policy: existing (potentially
-- very large) node policies must not be exported to mobile JSON subscriptions.
INSERT INTO rule_files (tag, geoip_url, geosite_url, active)
VALUES ('ru-direct-client',
        'https://raw.githubusercontent.com/kyoresuas/ru-direct/release/geoip.dat',
        'https://raw.githubusercontent.com/kyoresuas/ru-direct/release/geosite.dat', TRUE);

INSERT INTO routing_policies
    (tag, rule_file, rule_code, source_country, source_not_country,
     target_country, target_not_country, active, apply_to_nodes, apply_to_clients)
VALUES ('ru-direct-client', 'ru-direct-client', 'ru-standard', '', 'RU',
        'RU', '', TRUE, FALSE, TRUE);
