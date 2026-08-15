ALTER TABLE development_referral_appointments
  ADD COLUMN retention_kind TEXT NOT NULL DEFAULT 'manual'
    CHECK (retention_kind IN ('autosave', 'manual'));
CREATE INDEX development_referral_expiry_idx ON development_referral_appointments (expires_at);
