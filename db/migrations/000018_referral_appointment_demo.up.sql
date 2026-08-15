INSERT INTO permissions (name, description) VALUES
  ('referral.development_appointment.manage', 'Operate the synthetic referral and appointment demonstration')
ON CONFLICT (name) DO NOTHING;

CREATE TYPE development_referral_status AS ENUM ('requested', 'scheduled', 'arrived', 'completed', 'abandoned');

CREATE TABLE development_referral_appointments (
  id UUID PRIMARY KEY,
  institution_id BIGINT NOT NULL REFERENCES institutions(id),
  site_id BIGINT NOT NULL,
  firm_id BIGINT NOT NULL REFERENCES firms(id),
  synthetic_patient_id TEXT NOT NULL CHECK (btrim(synthetic_patient_id) <> '' AND char_length(synthetic_patient_id) <= 100),
  synthetic_patient_label TEXT NOT NULL CHECK (btrim(synthetic_patient_label) <> '' AND char_length(synthetic_patient_label) <= 200),
  recipient_role TEXT NOT NULL CHECK (recipient_role IN ('demo_gp', 'demo_optometrist', 'demo_consultant')),
  clinic_code TEXT NOT NULL CHECK (clinic_code IN ('demo_general_eye_clinic', 'demo_glaucoma_clinic', 'demo_retina_clinic')),
  appointment_date DATE NOT NULL,
  appointment_time TIME,
  priority TEXT NOT NULL CHECK (priority IN ('routine', 'soon', 'urgent')),
  notes TEXT NOT NULL DEFAULT '' CHECK (char_length(notes) <= 1000 AND position('<' IN notes) = 0),
  status development_referral_status NOT NULL DEFAULT 'requested',
  owner_user_id BIGINT NOT NULL REFERENCES users(id),
  version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (site_id, institution_id) REFERENCES sites(id, institution_id)
);
CREATE INDEX development_referral_context_status_idx ON development_referral_appointments (institution_id, site_id, firm_id, status, appointment_date, id);

CREATE TABLE development_referral_audit (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  referral_id UUID NOT NULL REFERENCES development_referral_appointments(id),
  actor_user_id BIGINT NOT NULL REFERENCES users(id),
  institution_id BIGINT NOT NULL REFERENCES institutions(id),
  site_id BIGINT NOT NULL,
  firm_id BIGINT NOT NULL REFERENCES firms(id),
  command TEXT NOT NULL CHECK (command IN ('create', 'schedule', 'arrive', 'complete', 'abandon')),
  prior_state development_referral_status,
  next_state development_referral_status NOT NULL,
  resulting_version BIGINT NOT NULL CHECK (resulting_version >= 1),
  correlation_id TEXT NOT NULL CHECK (btrim(correlation_id) <> '' AND char_length(correlation_id) <= 200),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (site_id, institution_id) REFERENCES sites(id, institution_id)
);
CREATE INDEX development_referral_audit_referral_idx ON development_referral_audit (referral_id, created_at DESC, id DESC);
CREATE FUNCTION reject_development_referral_audit_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'development_referral_audit is append-only'; END; $$;
CREATE TRIGGER development_referral_audit_no_update_or_delete BEFORE UPDATE OR DELETE ON development_referral_audit FOR EACH ROW EXECUTE FUNCTION reject_development_referral_audit_mutation();
CREATE TRIGGER development_referral_audit_no_truncate BEFORE TRUNCATE ON development_referral_audit FOR EACH STATEMENT EXECUTE FUNCTION reject_development_referral_audit_mutation();
