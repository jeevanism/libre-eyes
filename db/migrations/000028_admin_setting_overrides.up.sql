-- Scoped overrides are separate from the existing institution defaults so the
-- demo can evolve without changing the established settings rows.
CREATE TABLE development_admin_setting_overrides (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    key TEXT NOT NULL,
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    site_id BIGINT REFERENCES sites(id),
    firm_id BIGINT REFERENCES firms(id),
    value TEXT NOT NULL CHECK (btrim(value) <> '' AND char_length(value) <= 80),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((site_id IS NULL) OR (firm_id IS NULL)),
    CHECK (site_id IS NOT NULL OR firm_id IS NOT NULL)
);

CREATE UNIQUE INDEX development_admin_setting_overrides_scope_key
    ON development_admin_setting_overrides
       (key, institution_id, COALESCE(site_id, 0), COALESCE(firm_id, 0));

CREATE INDEX development_admin_setting_overrides_lookup
    ON development_admin_setting_overrides (institution_id, site_id, firm_id, active);
