INSERT INTO permissions (name, description) VALUES
    ('worklist.development_flow.manage', 'Operate the synthetic development clinic-flow queue')
ON CONFLICT (name) DO NOTHING;

CREATE TYPE development_flow_ticket_status AS ENUM (
    'waiting', 'arrived', 'in_progress', 'completed'
);

CREATE TABLE development_flow_tickets (
    id UUID PRIMARY KEY,
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    site_id BIGINT NOT NULL,
    firm_id BIGINT NOT NULL REFERENCES firms(id),
    synthetic_patient_id TEXT NOT NULL CHECK (btrim(synthetic_patient_id) <> '' AND char_length(synthetic_patient_id) <= 100),
    synthetic_patient_label TEXT NOT NULL CHECK (btrim(synthetic_patient_label) <> '' AND char_length(synthetic_patient_label) <= 200),
    status development_flow_ticket_status NOT NULL DEFAULT 'waiting',
    assignee_user_id BIGINT REFERENCES users(id),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (site_id, institution_id) REFERENCES sites(id, institution_id),
    CHECK ((status IN ('in_progress', 'completed')) = (assignee_user_id IS NOT NULL)),
    CHECK (updated_at >= created_at)
);

CREATE INDEX development_flow_tickets_context_status_idx
    ON development_flow_tickets (institution_id, site_id, firm_id, status, created_at, id);

CREATE TABLE development_flow_audit (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES development_flow_tickets(id),
    actor_user_id BIGINT NOT NULL REFERENCES users(id),
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    site_id BIGINT NOT NULL,
    firm_id BIGINT NOT NULL REFERENCES firms(id),
    command TEXT NOT NULL CHECK (command IN ('arrive', 'claim', 'release', 'complete')),
    prior_state development_flow_ticket_status NOT NULL,
    next_state development_flow_ticket_status NOT NULL,
    resulting_version BIGINT NOT NULL CHECK (resulting_version >= 1),
    correlation_id TEXT NOT NULL CHECK (btrim(correlation_id) <> '' AND char_length(correlation_id) <= 200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (site_id, institution_id) REFERENCES sites(id, institution_id)
);

CREATE INDEX development_flow_audit_ticket_idx
    ON development_flow_audit (ticket_id, created_at DESC, id DESC);

CREATE FUNCTION reject_development_flow_audit_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'development_flow_audit is append-only';
END;
$$;

CREATE TRIGGER development_flow_audit_no_update_or_delete
BEFORE UPDATE OR DELETE ON development_flow_audit
FOR EACH ROW EXECUTE FUNCTION reject_development_flow_audit_mutation();

CREATE TRIGGER development_flow_audit_no_truncate
BEFORE TRUNCATE ON development_flow_audit
FOR EACH STATEMENT EXECUTE FUNCTION reject_development_flow_audit_mutation();
