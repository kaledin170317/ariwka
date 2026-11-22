-- Use the same ru-direct geo files, served through jsDelivr for Happ clients.
UPDATE rule_files
SET geoip_url = 'https://cdn.jsdelivr.net/gh/kyoresuas/ru-direct@release/geoip.dat',
    geosite_url = 'https://cdn.jsdelivr.net/gh/kyoresuas/ru-direct@release/geosite.dat'
WHERE tag = 'ru-direct-client';

UPDATE routing_policies
SET rule_code = 'ru-standard'
WHERE tag = 'ru-direct-client'
  AND rule_file = 'ru-direct-client'
  AND rule_code = 'ru-lite'
  AND apply_to_clients = TRUE
  AND apply_to_nodes = FALSE;
