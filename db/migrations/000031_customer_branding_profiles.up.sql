CREATE TABLE development_branding_profile_versions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    profile_version BIGINT NOT NULL CHECK (profile_version >= 1),
    status TEXT NOT NULL CHECK (status IN ('draft', 'published', 'superseded')),
    organization_name TEXT NOT NULL CHECK (btrim(organization_name) <> '' AND char_length(organization_name) <= 120),
    short_name TEXT NOT NULL CHECK (btrim(short_name) <> '' AND char_length(short_name) <= 60),
    browser_title TEXT NOT NULL CHECK (btrim(browser_title) <> '' AND char_length(browser_title) <= 80),
    primary_color TEXT NOT NULL CHECK (primary_color ~ '^#[0-9A-Fa-f]{6}$'),
    primary_hover_color TEXT NOT NULL CHECK (primary_hover_color ~ '^#[0-9A-Fa-f]{6}$'),
    selected_surface_color TEXT NOT NULL CHECK (selected_surface_color ~ '^#[0-9A-Fa-f]{6}$'),
    focus_color TEXT NOT NULL CHECK (focus_color ~ '^#[0-9A-Fa-f]{6}$'),
    row_version BIGINT NOT NULL DEFAULT 1 CHECK (row_version >= 1),
    created_by_user_id BIGINT REFERENCES users(id),
    published_by_user_id BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    UNIQUE (institution_id, profile_version)
);

CREATE UNIQUE INDEX development_branding_one_draft_idx
    ON development_branding_profile_versions (institution_id)
    WHERE status = 'draft';

CREATE UNIQUE INDEX development_branding_one_published_idx
    ON development_branding_profile_versions (institution_id)
    WHERE status = 'published';

CREATE INDEX development_branding_history_idx
    ON development_branding_profile_versions (institution_id, profile_version DESC);

CREATE TABLE development_branding_audit (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_user_id BIGINT NOT NULL REFERENCES users(id),
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    command TEXT NOT NULL CHECK (command IN ('branding.draft_saved', 'branding.published', 'branding.rolled_back')),
    profile_version BIGINT NOT NULL CHECK (profile_version >= 1),
    before_values JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(before_values) = 'object'),
    after_values JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(after_values) = 'object'),
    correlation_id TEXT NOT NULL CHECK (btrim(correlation_id) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX development_branding_audit_scope_idx
    ON development_branding_audit (institution_id, created_at DESC, id DESC);

CREATE FUNCTION reject_development_branding_audit_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'development_branding_audit is append-only';
END;
$$;

CREATE TRIGGER development_branding_audit_no_mutation
BEFORE UPDATE OR DELETE OR TRUNCATE ON development_branding_audit
FOR EACH STATEMENT EXECUTE FUNCTION reject_development_branding_audit_mutation();

INSERT INTO development_branding_profile_versions (
    institution_id,
    profile_version,
    status,
    organization_name,
    short_name,
    browser_title,
    primary_color,
    primary_hover_color,
    selected_surface_color,
    focus_color,
    published_at
)
SELECT
    id,
    1,
    'published',
    name,
    'LibreEyes',
    'LibreEyes',
    '#116466',
    '#0c5355',
    '#deefee',
    '#0b6fcc',
    now()
FROM institutions
WHERE active
ON CONFLICT (institution_id, profile_version) DO NOTHING;
