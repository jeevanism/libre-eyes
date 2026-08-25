ALTER TABLE development_admin_settings
    DROP CONSTRAINT IF EXISTS development_admin_settings_key_check;

ALTER TABLE development_admin_settings
    ADD CONSTRAINT development_admin_settings_key_allowed CHECK (key IN (
        'default_site',
        'default_firm',
        'appointment_slot_minutes',
        'demo_retention_days',
        'clinic_flow_queue_name',
        'clinic_flow_priorities',
        'theatre_default_room',
        'theatre_session_minutes',
        'theatre_capacity_minutes',
        'referral_default_recipient',
        'referral_default_priority',
        'correspondence_default_template',
        'correspondence_footer',
        'messaging_default_type',
        'messaging_mailbox'
    ));
