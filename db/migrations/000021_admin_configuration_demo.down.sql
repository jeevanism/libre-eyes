DROP TRIGGER IF EXISTS development_admin_audit_no_mutation ON development_admin_audit;
DROP FUNCTION IF EXISTS reject_development_admin_audit_mutation();
DROP TABLE IF EXISTS development_admin_audit;
DROP TABLE IF EXISTS development_admin_settings;
DROP TABLE IF EXISTS development_admin_users;
DELETE FROM permissions WHERE name IN ('admin.development.read','admin.development.manage');
