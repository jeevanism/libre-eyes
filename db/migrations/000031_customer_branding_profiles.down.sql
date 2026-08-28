DROP TRIGGER IF EXISTS development_branding_audit_no_mutation ON development_branding_audit;
DROP FUNCTION IF EXISTS reject_development_branding_audit_mutation();
DROP TABLE IF EXISTS development_branding_audit;
DROP TABLE IF EXISTS development_branding_profile_versions;
