ALTER TABLE rule_sets RENAME TO rule_files;
ALTER TABLE routing_policies RENAME COLUMN rule_set TO rule_file;
ALTER TABLE routing_policies ADD COLUMN rule_code TEXT NOT NULL DEFAULT '';

-- Transfer existing per-file list codes into policies, including old codes
-- that differ from the file tag. Handle installations where 0014 already ran.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'rule_files' AND column_name = 'code') THEN
        UPDATE routing_policies p SET rule_code = f.code
        FROM rule_files f WHERE p.rule_file = f.tag AND p.rule_file <> '';
        ALTER TABLE rule_files DROP COLUMN code;
    ELSE
        UPDATE routing_policies SET rule_code = rule_file WHERE rule_file <> '';
    END IF;
END $$;
