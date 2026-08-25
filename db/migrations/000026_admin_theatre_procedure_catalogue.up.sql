CREATE TABLE development_theatre_procedure_catalogue (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code LIKE 'development\_%'),
    display_name TEXT NOT NULL CHECK (btrim(display_name) <> '' AND char_length(display_name) <= 200),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL CHECK (display_order >= 0),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    UNIQUE (display_order)
);

INSERT INTO development_theatre_procedure_catalogue (code, display_name, display_order) VALUES
    ('development_cataract_surgery', 'Demo cataract surgery', 0),
    ('development_glaucoma_surgery', 'Demo glaucoma surgery', 1),
    ('development_intravitreal_injection', 'Demo intravitreal injection', 2);

CREATE INDEX development_theatre_procedure_catalogue_admin_idx
    ON development_theatre_procedure_catalogue (active, display_order, id);
