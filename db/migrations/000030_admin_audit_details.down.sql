DROP INDEX IF EXISTS development_admin_audit_filter_idx;
ALTER TABLE development_admin_audit
    DROP COLUMN IF EXISTS before_values,
    DROP COLUMN IF EXISTS after_values,
    DROP COLUMN IF EXISTS scope;
