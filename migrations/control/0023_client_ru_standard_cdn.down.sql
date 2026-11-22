UPDATE routing_policies
SET rule_code = 'ru-lite'
WHERE tag = 'ru-direct-client'
  AND rule_file = 'ru-direct-client'
  AND rule_code = 'ru-standard'
  AND apply_to_clients = TRUE
  AND apply_to_nodes = FALSE;

UPDATE rule_files
SET geoip_url = 'https://raw.githubusercontent.com/kyoresuas/ru-direct/release/geoip.dat',
    geosite_url = 'https://raw.githubusercontent.com/kyoresuas/ru-direct/release/geosite.dat'
WHERE tag = 'ru-direct-client'
  AND geoip_url = 'https://cdn.jsdelivr.net/gh/kyoresuas/ru-direct@release/geoip.dat'
  AND geosite_url = 'https://cdn.jsdelivr.net/gh/kyoresuas/ru-direct@release/geosite.dat';
