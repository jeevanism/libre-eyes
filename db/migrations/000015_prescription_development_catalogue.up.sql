CREATE TABLE development_prescription_catalogue (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category TEXT NOT NULL CHECK (category IN ('medication', 'route', 'frequency', 'duration', 'laterality')),
    code TEXT NOT NULL UNIQUE CHECK (code LIKE 'development\_%'),
    display_name TEXT NOT NULL CHECK (btrim(display_name) <> '' AND char_length(display_name) <= 200),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL CHECK (display_order >= 0),
    UNIQUE (category, display_order)
);

INSERT INTO development_prescription_catalogue (category, code, display_name, display_order) VALUES
('medication', 'development_lubricating_drop', 'Demo lubricating eye drop', 0),
('medication', 'development_antibiotic_drop', 'Demo antibiotic eye drop', 1),
('medication', 'development_steroid_drop', 'Demo steroid eye drop', 2),
('route', 'development_topical_eye', 'Demo topical eye', 0),
('route', 'development_oral', 'Demo oral', 1),
('frequency', 'development_once_daily', 'Demo once daily', 0),
('frequency', 'development_twice_daily', 'Demo twice daily', 1),
('frequency', 'development_four_times_daily', 'Demo four times daily', 2),
('duration', 'development_five_days', 'Demo five days', 0),
('duration', 'development_seven_days', 'Demo seven days', 1),
('duration', 'development_fourteen_days', 'Demo fourteen days', 2),
('laterality', 'development_left', 'Left eye', 0),
('laterality', 'development_right', 'Right eye', 1),
('laterality', 'development_bilateral', 'Both eyes', 2),
('laterality', 'development_not_applicable', 'Not applicable', 3);
