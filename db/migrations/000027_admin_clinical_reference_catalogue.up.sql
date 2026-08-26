CREATE TABLE development_admin_clinical_catalogue (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    domain TEXT NOT NULL CHECK (domain IN ('biometry', 'laser', 'intravitreal', 'lab', 'genetics', 'dna', 'consent', 'examination')),
    code TEXT NOT NULL UNIQUE CHECK (code LIKE 'development\_%'),
    display_name TEXT NOT NULL CHECK (btrim(display_name) <> '' AND char_length(display_name) <= 200),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL CHECK (display_order >= 0),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    UNIQUE (domain, display_order)
);

INSERT INTO development_admin_clinical_catalogue (domain, code, display_name, display_order) VALUES
 ('biometry','development_biometry_standard','Demo standard lens',0),
 ('biometry','development_biometry_toric','Demo toric lens',1),
 ('laser','development_laser_grid','Demo grid laser',0),
 ('laser','development_laser_focal','Demo focal laser',1),
 ('intravitreal','development_injection_antivegf','Demo anti-VEGF injection',0),
 ('intravitreal','development_injection_steroid','Demo steroid injection',1),
 ('lab','development_lab_hba1c','Demo HbA1c',0),
 ('lab','development_lab_iop','Demo pressure result',1),
 ('genetics','development_genetic_panel','Demo gene panel',0),
 ('genetics','development_genetic_variant','Demo variant result',1),
 ('dna','development_dna_box_a','Demo DNA box A',0),
 ('dna','development_dna_box_b','Demo DNA box B',1),
 ('consent','development_consent_standard','Demo standard consent',0),
 ('consent','development_consent_extended','Demo extended consent',1),
 ('examination','development_exam_general','Demo general examination',0),
 ('examination','development_exam_followup','Demo follow-up examination',1);

CREATE INDEX development_admin_clinical_catalogue_admin_idx
 ON development_admin_clinical_catalogue (domain, active, display_order, id);
