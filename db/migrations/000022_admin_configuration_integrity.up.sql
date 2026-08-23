ALTER TABLE development_admin_settings DROP CONSTRAINT development_admin_settings_pkey;
ALTER TABLE development_admin_settings ADD PRIMARY KEY (institution_id, key);
ALTER TABLE development_admin_audit ADD COLUMN target_key TEXT;
CREATE INDEX development_admin_audit_target_key_idx ON development_admin_audit (institution_id, target_key) WHERE target_key IS NOT NULL;
