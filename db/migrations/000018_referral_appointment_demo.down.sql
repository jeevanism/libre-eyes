DROP TRIGGER IF EXISTS development_referral_audit_no_truncate ON development_referral_audit;
DROP TRIGGER IF EXISTS development_referral_audit_no_update_or_delete ON development_referral_audit;
DROP FUNCTION IF EXISTS reject_development_referral_audit_mutation();
DROP TABLE IF EXISTS development_referral_audit;
DROP TABLE IF EXISTS development_referral_appointments;
DROP TYPE IF EXISTS development_referral_status;
DELETE FROM permissions WHERE name = 'referral.development_appointment.manage';
