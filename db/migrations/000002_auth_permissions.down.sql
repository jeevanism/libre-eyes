DELETE FROM user_role_assignments
WHERE role_id IN (
    SELECT id FROM roles
    WHERE name IN ('VisionOpus User', 'Clinical Viewer', 'Institution Administrator', 'Global Administrator')
);

DELETE FROM role_permissions
WHERE role_id IN (
    SELECT id FROM roles
    WHERE name IN ('VisionOpus User', 'Clinical Viewer', 'Institution Administrator', 'Global Administrator')
);

DELETE FROM roles
WHERE name IN ('VisionOpus User', 'Clinical Viewer', 'Institution Administrator', 'Global Administrator');

DELETE FROM permissions
WHERE name IN (
    'session.read_self',
    'session.revoke_self',
    'context.switch',
    'patient.search',
    'clinical.view',
    'user.admin.institution',
    'role.assign.institution',
    'user.admin.global',
    'role.assign.global'
);
