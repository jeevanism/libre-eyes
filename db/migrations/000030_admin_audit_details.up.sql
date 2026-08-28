ALTER TABLE development_admin_audit
    ADD COLUMN IF NOT EXISTS before_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS after_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS scope TEXT NOT NULL DEFAULT 'institution';

CREATE INDEX IF NOT EXISTS development_admin_audit_filter_idx
    ON development_admin_audit (institution_id, command, outcome, target_type, created_at DESC, id DESC);
