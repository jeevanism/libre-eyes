DROP INDEX IF EXISTS development_admin_audit_target_key_idx;
ALTER TABLE development_admin_audit DROP COLUMN IF EXISTS target_key;
ALTER TABLE development_admin_settings DROP CONSTRAINT development_admin_settings_pkey;
ALTER TABLE development_admin_settings ADD PRIMARY KEY (key);
