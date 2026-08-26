CREATE TABLE development_admin_capabilities (
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    capability_key TEXT NOT NULL CHECK (capability_key ~ '^[a-z][a-z0-9_]*$'),
    display_name TEXT NOT NULL CHECK (btrim(display_name) <> '' AND char_length(display_name) <= 120),
    description TEXT NOT NULL CHECK (char_length(description) <= 300),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (institution_id, capability_key)
);

INSERT INTO development_admin_capabilities (institution_id, capability_key, display_name, description)
SELECT i.id, v.capability_key, v.display_name, v.description
FROM institutions i
CROSS JOIN (VALUES
 ('patient_search','Patient search','Search and open synthetic patient summaries.'),
 ('examination','Examination tools','Visual acuity, pressure, fields, and examination drafts.'),
 ('clinic_flow','Clinic flow','Synthetic clinic queue and ticket movement.'),
 ('theatre_booking','Theatre schedule','Synthetic theatre booking board.'),
 ('referrals','Referrals and appointments','Synthetic referral and appointment queue.'),
 ('prescribing','Medication orders','Synthetic medication-order drafts and catalogue.'),
 ('correspondence','Correspondence and messaging','Synthetic correspondence and internal messaging drafts.'),
 ('consent','Consent forms','Synthetic consent-form drafts.'),
 ('laboratory','Laboratory and genetics','Synthetic laboratory and genetic-result drafts.'),
 ('administration','Administration','Synthetic users, contexts, settings, and catalogues.')
) AS v(capability_key, display_name, description)
WHERE i.name = 'VisionOpus Development Hospital'
ON CONFLICT (institution_id, capability_key) DO NOTHING;
