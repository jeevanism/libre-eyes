CREATE TABLE development_lab_results_catalogue (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code LIKE 'demo_lab_%'),
    display_name TEXT NOT NULL CHECK (btrim(display_name) <> '' AND char_length(display_name) <= 200),
    field_kind TEXT NOT NULL CHECK (field_kind IN ('numeric', 'choice')),
    default_unit TEXT NOT NULL DEFAULT '' CHECK (char_length(default_unit) <= 30),
    hard_min NUMERIC,
    hard_max NUMERIC,
    normal_min NUMERIC,
    normal_max NUMERIC,
    choices JSONB NOT NULL DEFAULT '[]'::jsonb,
    institution_id BIGINT REFERENCES institutions(id),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL CHECK (display_order >= 0),
    CHECK (hard_min IS NULL OR hard_max IS NULL OR hard_min <= hard_max),
    CHECK (normal_min IS NULL OR normal_max IS NULL OR normal_min <= normal_max),
    CHECK (normal_min IS NULL OR hard_min IS NULL OR normal_min >= hard_min),
    CHECK (normal_max IS NULL OR hard_max IS NULL OR normal_max <= hard_max)
);
CREATE INDEX development_lab_results_catalogue_scope_idx ON development_lab_results_catalogue (institution_id, active, display_order);
