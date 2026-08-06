CREATE INDEX patients_demographics_duplicate_idx
    ON patients (
        family_name_normalized,
        given_name_normalized,
        date_of_birth,
        public_id
    )
    WHERE active;

DROP INDEX patient_identifiers_patient_active_idx;
