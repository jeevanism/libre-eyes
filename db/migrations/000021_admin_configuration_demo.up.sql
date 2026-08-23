INSERT INTO permissions (name, description) VALUES
    ('admin.development.read', 'Read the synthetic administration workspace'),
    ('admin.development.manage', 'Manage synthetic users and settings')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE development_admin_users (
    public_id UUID PRIMARY KEY,
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id),
    username TEXT NOT NULL UNIQUE CHECK (btrim(username) <> '' AND char_length(username) <= 80),
    display_name TEXT NOT NULL CHECK (btrim(display_name) <> '' AND char_length(display_name) <= 120),
    role_code TEXT NOT NULL CHECK (role_code IN ('system_administrator','institution_administrator','clinical_user')),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE development_admin_settings (
    key TEXT PRIMARY KEY CHECK (key IN ('default_site','default_firm','appointment_slot_minutes','demo_retention_days')),
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    value TEXT NOT NULL CHECK (btrim(value) <> '' AND char_length(value) <= 80),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE development_admin_audit (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_user_id BIGINT NOT NULL REFERENCES users(id),
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    command TEXT NOT NULL CHECK (btrim(command) <> ''),
    target_type TEXT NOT NULL CHECK (btrim(target_type) <> ''),
    target_public_id UUID,
    changed_fields JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(changed_fields) = 'array'),
    outcome TEXT NOT NULL CHECK (outcome IN ('success','failure','denied')),
    correlation_id TEXT NOT NULL CHECK (btrim(correlation_id) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX development_admin_users_scope_idx ON development_admin_users (institution_id, active, display_name);
CREATE INDEX development_admin_audit_scope_idx ON development_admin_audit (institution_id, created_at DESC, id DESC);

CREATE FUNCTION reject_development_admin_audit_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'development_admin_audit is append-only'; END;
$$;
CREATE TRIGGER development_admin_audit_no_mutation
BEFORE UPDATE OR DELETE OR TRUNCATE ON development_admin_audit
FOR EACH STATEMENT EXECUTE FUNCTION reject_development_admin_audit_mutation();
