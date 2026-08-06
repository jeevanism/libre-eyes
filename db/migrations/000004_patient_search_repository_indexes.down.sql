CREATE INDEX patient_identifiers_patient_active_idx
    ON patient_identifiers (patient_id, identifier_type_id)
    WHERE active;

DROP INDEX patients_demographics_duplicate_idx;
