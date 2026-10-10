ALTER TABLE rule_files ADD COLUMN code TEXT NOT NULL DEFAULT '';
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM routing_policies
               WHERE rule_file <> '' GROUP BY rule_file
               HAVING count(DISTINCT rule_code) > 1) THEN
        RAISE EXCEPTION 'cannot roll back files used with multiple list codes';
    END IF;
END $$;
UPDATE rule_files f SET code = p.rule_code FROM routing_policies p WHERE p.rule_file = f.tag AND p.rule_code <> '';
ALTER TABLE routing_policies DROP COLUMN rule_code;
ALTER TABLE routing_policies RENAME COLUMN rule_file TO rule_set;
ALTER TABLE rule_files RENAME TO rule_sets;
