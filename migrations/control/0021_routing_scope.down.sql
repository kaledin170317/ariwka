DELETE FROM routing_policies WHERE tag = 'ru-direct-client';
DELETE FROM rule_files WHERE tag = 'ru-direct-client';
ALTER TABLE routing_policies
    DROP COLUMN apply_to_clients,
    DROP COLUMN apply_to_nodes;
