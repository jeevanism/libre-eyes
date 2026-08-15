CREATE TABLE development_consent_catalogue (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category TEXT NOT NULL CHECK (category IN ('form_type', 'procedure', 'laterality', 'anaesthetic', 'benefit_risk')),
    code TEXT NOT NULL UNIQUE CHECK (code LIKE 'development\_%'),
    display_name TEXT NOT NULL CHECK (btrim(display_name) <> '' AND char_length(display_name) <= 200),
    detail TEXT NOT NULL DEFAULT '' CHECK (char_length(detail) <= 2000),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL CHECK (display_order >= 0),
    UNIQUE (category, display_order)
);
