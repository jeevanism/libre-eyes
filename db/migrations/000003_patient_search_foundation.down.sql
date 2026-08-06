DROP TABLE IF EXISTS patient_merge_lineage;
DROP TABLE IF EXISTS patient_identifier_history;
DROP TABLE IF EXISTS patient_identifiers;
DROP TABLE IF EXISTS patient_identifier_statuses;
DROP TABLE IF EXISTS patient_identifier_types;
DROP TABLE IF EXISTS patient_institutions;
DROP TABLE IF EXISTS patients;

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name = 'patient.duplicate_check'
);

DELETE FROM permissions WHERE name = 'patient.duplicate_check';

DROP TYPE IF EXISTS patient_identifier_lifecycle;
DROP TYPE IF EXISTS patient_gender;
