CREATE TABLE development_correspondence_catalogue (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category TEXT NOT NULL CHECK (category IN ('template', 'recipient_role')),
    code TEXT NOT NULL UNIQUE CHECK (code LIKE 'demo\_%'),
    display_name TEXT NOT NULL CHECK (btrim(display_name) <> '' AND char_length(display_name) <= 200),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL CHECK (display_order >= 0),
    UNIQUE (category, display_order)
);
