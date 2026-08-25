ALTER TABLE development_admin_settings
    DROP CONSTRAINT IF EXISTS development_admin_settings_key_allowed;

DELETE FROM development_admin_settings
WHERE key NOT IN ('default_site', 'default_firm', 'appointment_slot_minutes', 'demo_retention_days');

ALTER TABLE development_admin_settings
    ADD CONSTRAINT development_admin_settings_key_check CHECK (key IN (
        'default_site',
        'default_firm',
        'appointment_slot_minutes',
    'demo_retention_days'
    ));
