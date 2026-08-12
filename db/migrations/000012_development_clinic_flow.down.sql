DROP TRIGGER IF EXISTS development_flow_audit_no_truncate ON development_flow_audit;
DROP TRIGGER IF EXISTS development_flow_audit_no_update_or_delete ON development_flow_audit;
DROP FUNCTION IF EXISTS reject_development_flow_audit_mutation();
DROP TABLE IF EXISTS development_flow_audit;
DROP TABLE IF EXISTS development_flow_tickets;
DROP TYPE IF EXISTS development_flow_ticket_status;
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'worklist.development_flow.manage');
DELETE FROM permissions WHERE name = 'worklist.development_flow.manage';
