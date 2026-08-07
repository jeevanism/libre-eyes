DROP TRIGGER IF EXISTS event_drafts_no_mutation_after_disposal ON event_drafts;
DROP FUNCTION IF EXISTS reject_disposed_event_draft_mutation();
DROP INDEX IF EXISTS event_drafts_expiry_idx;
DROP INDEX IF EXISTS event_drafts_active_owner_idx;
DROP INDEX IF EXISTS event_drafts_active_update_unique;
DROP INDEX IF EXISTS event_drafts_active_autosave_create_unique;
DROP TABLE IF EXISTS event_drafts;

-- Refuse a rollback that would erase clinically significant version history.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM event_versions LIMIT 1) THEN
        RAISE EXCEPTION 'rollback blocked: event_versions contains rows';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS event_versions_no_truncate ON event_versions;
DROP TRIGGER IF EXISTS event_versions_no_update_or_delete ON event_versions;
DROP FUNCTION IF EXISTS reject_event_version_mutation();
DROP INDEX IF EXISTS event_versions_event_idx;
DROP TABLE IF EXISTS event_versions;

DROP INDEX IF EXISTS events_scope_header_idx;
DROP INDEX IF EXISTS events_patient_timeline_idx;
DROP INDEX IF EXISTS events_episode_timeline_idx;
DROP TABLE IF EXISTS events;

DROP TABLE IF EXISTS episode_audit_sequences;
DROP INDEX IF EXISTS episodes_scope_status_idx;
DROP INDEX IF EXISTS episodes_patient_timeline_idx;
DROP TABLE IF EXISTS episodes;

ALTER TABLE patient_institutions
    DROP CONSTRAINT IF EXISTS patient_institutions_id_patient_institution_unique;

DROP TYPE IF EXISTS event_draft_mode;
DROP TYPE IF EXISTS event_draft_intent;
DROP TYPE IF EXISTS episode_event_status;
DROP TYPE IF EXISTS episode_lifecycle_status;

-- Permission rows are retained on rollback because deployments may already
-- reference them from role assignments; see the approved migration direction
-- in doc/approvals/0005-episodes-events-security-migration.md.
