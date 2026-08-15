DROP INDEX IF EXISTS development_referral_expiry_idx;
ALTER TABLE development_referral_appointments DROP COLUMN IF EXISTS retention_kind;
