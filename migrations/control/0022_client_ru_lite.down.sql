UPDATE routing_policies
SET rule_code = 'ru-standard'
WHERE tag = 'ru-direct-client'
  AND rule_file = 'ru-direct-client'
  AND rule_code = 'ru-lite'
  AND apply_to_clients = TRUE
  AND apply_to_nodes = FALSE;
