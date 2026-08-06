CREATE TYPE patient_summary_warning_status AS ENUM (
    'present',
    'none_known',
    'unknown',
    'unavailable'
);

CREATE TYPE patient_summary_warning_kind AS ENUM (
    'allergy',
    'risk',
    'diabetes'
);

INSERT INTO permissions (name, description) VALUES
    ('patient.clinical_summary.read', 'Read patient summary warning status and details'),
    ('patient.break_glass', 'Request bounded audited patient break-glass access'),
    ('patient.break_glass.revoke', 'Revoke a bounded patient break-glass grant')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE patient_break_glass_grants (
    grant_id TEXT PRIMARY KEY CHECK (char_length(grant_id) BETWEEN 32 AND 128),
    patient_id BIGINT NOT NULL REFERENCES patients(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    session_id BIGINT NOT NULL REFERENCES sessions(id),
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    site_id BIGINT NOT NULL REFERENCES sites(id),
    firm_id BIGINT NOT NULL REFERENCES firms(id),
    reason_code TEXT NOT NULL CHECK (btrim(reason_code) <> ''),
    reason_detail TEXT CHECK (reason_detail IS NULL OR char_length(reason_detail) <= 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    CHECK (expires_at > created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX patient_break_glass_grants_lookup_idx
    ON patient_break_glass_grants (user_id, session_id, patient_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE patient_break_glass_alert_outbox (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    grant_id TEXT NOT NULL REFERENCES patient_break_glass_grants(grant_id),
    event_type TEXT NOT NULL CHECK (event_type IN ('created', 'revoked')),
    queued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at TIMESTAMPTZ,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    UNIQUE (grant_id, event_type)
);

CREATE TABLE patient_summary_warning_projections (
    patient_id BIGINT PRIMARY KEY REFERENCES patients(id),
    allergy_status patient_summary_warning_status NOT NULL,
    alert_status patient_summary_warning_status NOT NULL,
    allergy_assessed_at TIMESTAMPTZ,
    alert_assessed_at TIMESTAMPTZ,
    source_revision TEXT NOT NULL CHECK (btrim(source_revision) <> ''),
    source_checksum TEXT NOT NULL CHECK (btrim(source_checksum) <> ''),
    warning_version BIGINT NOT NULL DEFAULT 1 CHECK (warning_version >= 1),
    projected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    projection_state TEXT NOT NULL DEFAULT 'verified' CHECK (
        projection_state IN ('verified', 'quarantined', 'unavailable')
    ),
    quarantine_reason TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (projection_state <> 'quarantined' OR quarantine_reason IS NOT NULL),
    CHECK (projection_state = 'quarantined' OR quarantine_reason IS NULL)
);

CREATE TABLE patient_summary_warning_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    patient_id BIGINT NOT NULL REFERENCES patients(id),
    warning_kind patient_summary_warning_kind NOT NULL,
    code TEXT,
    label TEXT NOT NULL CHECK (btrim(label) <> '' AND char_length(label) <= 500),
    reaction TEXT CHECK (reaction IS NULL OR char_length(reaction) <= 1000),
    comment TEXT CHECK (comment IS NULL OR char_length(comment) <= 2000),
    item_order INTEGER NOT NULL CHECK (item_order >= 0 AND item_order < 200),
    source_revision TEXT NOT NULL CHECK (btrim(source_revision) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (patient_id, source_revision, warning_kind, item_order)
);

CREATE INDEX patient_summary_warning_items_patient_order_idx
    ON patient_summary_warning_items (patient_id, item_order, id);

CREATE INDEX patient_summary_warning_projection_state_idx
    ON patient_summary_warning_projections (projection_state, updated_at);
