DROP TRIGGER IF EXISTS event_drafts_no_delete ON event_drafts;
DROP FUNCTION IF EXISTS reject_event_draft_hard_deletion();

CREATE OR REPLACE FUNCTION reject_disposed_event_draft_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.deleted_at IS NOT NULL THEN
        RAISE EXCEPTION 'disposed event_drafts are immutable';
    END IF;
    RETURN NEW;
END;
$$;
