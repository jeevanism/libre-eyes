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
