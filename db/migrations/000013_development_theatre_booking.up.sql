INSERT INTO permissions (name, description) VALUES
    ('theatre.development_booking.manage', 'Operate the synthetic development theatre-booking demonstration')
ON CONFLICT (name) DO NOTHING;

CREATE TYPE development_booking_request_status AS ENUM ('waiting', 'scheduled', 'cancelled');

CREATE TABLE development_theatre_rooms (
    id UUID PRIMARY KEY,
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    site_id BIGINT NOT NULL,
    firm_id BIGINT NOT NULL REFERENCES firms(id),
    synthetic_label TEXT NOT NULL CHECK (btrim(synthetic_label) <> '' AND char_length(synthetic_label) <= 80),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (institution_id, site_id, firm_id, synthetic_label),
    FOREIGN KEY (site_id, institution_id) REFERENCES sites(id, institution_id)
);

CREATE TABLE development_theatre_sessions (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES development_theatre_rooms(id),
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    site_id BIGINT NOT NULL,
    firm_id BIGINT NOT NULL REFERENCES firms(id),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    capacity_minutes INTEGER NOT NULL CHECK (capacity_minutes BETWEEN 1 AND 1440),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at),
    FOREIGN KEY (site_id, institution_id) REFERENCES sites(id, institution_id)
);

CREATE INDEX development_theatre_sessions_context_start_idx
    ON development_theatre_sessions (institution_id, site_id, firm_id, starts_at, id);

CREATE TABLE development_booking_requests (
    id UUID PRIMARY KEY,
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    site_id BIGINT NOT NULL,
    firm_id BIGINT NOT NULL REFERENCES firms(id),
    synthetic_label TEXT NOT NULL CHECK (btrim(synthetic_label) <> '' AND char_length(synthetic_label) <= 120),
    requested_duration_minutes INTEGER NOT NULL CHECK (requested_duration_minutes BETWEEN 5 AND 480),
    status development_booking_request_status NOT NULL DEFAULT 'waiting',
    assigned_session_id UUID REFERENCES development_theatre_sessions(id),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (site_id, institution_id) REFERENCES sites(id, institution_id),
    CHECK ((status = 'scheduled') = (assigned_session_id IS NOT NULL))
);

CREATE INDEX development_booking_requests_context_status_idx
    ON development_booking_requests (institution_id, site_id, firm_id, status, created_at, id);
CREATE INDEX development_booking_requests_session_idx
    ON development_booking_requests (assigned_session_id) WHERE status = 'scheduled';

CREATE TABLE development_theatre_booking_audit (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES development_booking_requests(id),
    actor_user_id BIGINT NOT NULL REFERENCES users(id),
    institution_id BIGINT NOT NULL REFERENCES institutions(id),
    site_id BIGINT NOT NULL,
    firm_id BIGINT NOT NULL REFERENCES firms(id),
    command TEXT NOT NULL CHECK (command IN ('schedule', 'reschedule', 'cancel')),
    prior_state development_booking_request_status NOT NULL,
    next_state development_booking_request_status NOT NULL,
    prior_session_id UUID,
    next_session_id UUID,
    resulting_version BIGINT NOT NULL CHECK (resulting_version >= 1),
    correlation_id TEXT NOT NULL CHECK (btrim(correlation_id) <> '' AND char_length(correlation_id) <= 200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (site_id, institution_id) REFERENCES sites(id, institution_id)
);

CREATE INDEX development_theatre_booking_audit_request_idx
    ON development_theatre_booking_audit (request_id, created_at DESC, id DESC);

CREATE FUNCTION reject_development_theatre_booking_audit_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'development_theatre_booking_audit is append-only';
END;
$$;

CREATE TRIGGER development_theatre_booking_audit_no_update_or_delete
BEFORE UPDATE OR DELETE ON development_theatre_booking_audit
FOR EACH ROW EXECUTE FUNCTION reject_development_theatre_booking_audit_mutation();

CREATE TRIGGER development_theatre_booking_audit_no_truncate
BEFORE TRUNCATE ON development_theatre_booking_audit
FOR EACH STATEMENT EXECUTE FUNCTION reject_development_theatre_booking_audit_mutation();
