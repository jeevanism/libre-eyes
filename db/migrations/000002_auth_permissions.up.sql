INSERT INTO permissions (name, description) VALUES
    ('session.read_self', 'Read the current authenticated session'),
    ('session.revoke_self', 'Revoke the current authenticated session'),
    ('context.switch', 'Change the current institution, site, and firm context'),
    ('patient.search', 'Search for patients in the current context'),
    ('clinical.view', 'View clinical information in an authorized patient context'),
    ('user.admin.institution', 'Administer users in an authorized institution'),
    ('role.assign.institution', 'Assign permitted roles in an authorized institution'),
    ('user.admin.global', 'Administer users globally'),
    ('role.assign.global', 'Assign global roles')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (name, description, scope) VALUES
    ('LibreEyes User', 'Base authenticated-user capabilities', 'institution'),
    ('Clinical Viewer', 'Read-only clinical access', 'institution'),
    ('Institution Administrator', 'Institution-scoped user and role administration', 'institution'),
    ('Global Administrator', 'Explicit global user and role administration', 'global')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name = ANY (CASE r.name
    WHEN 'LibreEyes User' THEN ARRAY['session.read_self', 'session.revoke_self', 'context.switch']
    WHEN 'Clinical Viewer' THEN ARRAY['patient.search', 'clinical.view']
    WHEN 'Institution Administrator' THEN ARRAY['user.admin.institution', 'role.assign.institution']
    WHEN 'Global Administrator' THEN ARRAY['user.admin.global', 'role.assign.global']
END)
WHERE r.name IN ('LibreEyes User', 'Clinical Viewer', 'Institution Administrator', 'Global Administrator')
ON CONFLICT (role_id, permission_id) DO NOTHING;
