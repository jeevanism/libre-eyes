ALTER TABLE development_prescription_catalogue
    ADD COLUMN version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1);

CREATE INDEX development_prescription_catalogue_admin_idx
    ON development_prescription_catalogue (category, active, display_order, id);
