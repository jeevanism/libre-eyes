CREATE OR REPLACE FUNCTION reject_disposed_event_draft_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.deleted_at IS NOT NULL THEN
        RAISE EXCEPTION 'disposed event_drafts are immutable';
    END IF;

    IF NEW.owner_user_id IS DISTINCT FROM OLD.owner_user_id
        OR NEW.episode_id IS DISTINCT FROM OLD.episode_id
        OR NEW.patient_id IS DISTINCT FROM OLD.patient_id
        OR NEW.institution_id IS DISTINCT FROM OLD.institution_id
        OR NEW.site_id IS DISTINCT FROM OLD.site_id
        OR NEW.firm_id IS DISTINCT FROM OLD.firm_id
        OR NEW.event_type_code IS DISTINCT FROM OLD.event_type_code
        OR NEW.target_event_id IS DISTINCT FROM OLD.target_event_id
        OR NEW.intent IS DISTINCT FROM OLD.intent
        OR NEW.mode IS DISTINCT FROM OLD.mode THEN
        RAISE EXCEPTION 'event_draft identity fields are immutable';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS event_drafts_no_mutation_after_disposal ON event_drafts;
CREATE TRIGGER event_drafts_no_mutation_after_disposal
BEFORE UPDATE ON event_drafts
FOR EACH ROW EXECUTE FUNCTION reject_disposed_event_draft_mutation();

CREATE OR REPLACE FUNCTION reject_event_draft_hard_deletion() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'hard deletion of event_drafts is prohibited';
END;
$$;

DROP TRIGGER IF EXISTS event_drafts_no_delete ON event_drafts;
CREATE TRIGGER event_drafts_no_delete
BEFORE DELETE ON event_drafts
FOR EACH STATEMENT EXECUTE FUNCTION reject_event_draft_hard_deletion();
