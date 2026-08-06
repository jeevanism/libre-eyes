// Command devseed creates one synthetic development identity and context.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/config"
)

func main() {
	if err := run(context.Background()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Environment != "development" {
		return errors.New("devseed may run only when VISIONOPUS_ENV=development")
	}
	username := strings.ToLower(strings.TrimSpace(valueOrDefault("VISIONOPUS_DEV_USERNAME", "clinician")))
	displayName := strings.TrimSpace(valueOrDefault("VISIONOPUS_DEV_DISPLAY_NAME", "Synthetic Clinician"))
	password := os.Getenv("VISIONOPUS_DEV_PASSWORD")
	if username == "" || displayName == "" || password == "" {
		return errors.New("development username, display name, and VISIONOPUS_DEV_PASSWORD are required")
	}

	passwordHash, err := (auth.PasswordManager{}).Hash(password)
	if err != nil {
		return fmt.Errorf("hash development password: %w", err)
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to development database: %w", err)
	}
	defer pool.Close()

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin development seed: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	institutionID, err := findOrInsert(ctx, tx,
		"SELECT id FROM institutions WHERE name = $1 ORDER BY id LIMIT 1",
		"INSERT INTO institutions (name) VALUES ($1) RETURNING id",
		"VisionOpus Development Hospital",
	)
	if err != nil {
		return err
	}
	siteID, err := findOrInsert(ctx, tx,
		"SELECT id FROM sites WHERE institution_id = $1 AND name = $2",
		"INSERT INTO sites (institution_id, name) VALUES ($1, $2) RETURNING id",
		institutionID, "Development Eye Clinic",
	)
	if err != nil {
		return err
	}
	firmID, err := findOrInsert(ctx, tx,
		"SELECT id FROM firms WHERE institution_id = $1 AND name = $2",
		"INSERT INTO firms (institution_id, name) VALUES ($1, $2) RETURNING id",
		institutionID, "Development Ophthalmology",
	)
	if err != nil {
		return err
	}
	profileID, err := findOrInsert(ctx, tx,
		"SELECT id FROM authentication_profiles WHERE institution_id = $1 AND method = 'LOCAL' AND name = $2",
		"INSERT INTO authentication_profiles (institution_id, method, name) VALUES ($1, 'LOCAL', $2) RETURNING id",
		institutionID, "Development Local",
	)
	if err != nil {
		return err
	}

	var userID int64
	err = tx.QueryRow(ctx, "SELECT user_id FROM user_credentials WHERE canonical_username = $1", username).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, "INSERT INTO users (display_name) VALUES ($1) RETURNING id", displayName).Scan(&userID); err != nil {
			return fmt.Errorf("insert development user: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find development user: %w", err)
	} else if _, err := tx.Exec(ctx, "UPDATE users SET display_name = $2, active = TRUE, updated_at = now() WHERE id = $1", userID, displayName); err != nil {
		return fmt.Errorf("update development user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO user_credentials (
			user_id, authentication_profile_id, canonical_username,
			password_hash, hash_scheme, hash_version
		) VALUES ($1, $2, $3, $4, 'argon2id', 19)
		ON CONFLICT (canonical_username) DO UPDATE SET
			authentication_profile_id = EXCLUDED.authentication_profile_id,
			password_hash = EXCLUDED.password_hash,
			hash_scheme = EXCLUDED.hash_scheme,
			hash_version = EXCLUDED.hash_version,
			state = 'active', failed_attempts = 0, soft_locked_until = NULL,
			active = TRUE, version = user_credentials.version + 1, updated_at = now()`,
		userID, profileID, username, passwordHash); err != nil {
		return fmt.Errorf("upsert development credential: %w", err)
	}

	memberships := []struct {
		query string
		id    int64
	}{
		{"INSERT INTO user_institution_memberships (user_id, institution_id) VALUES ($1, $2) ON CONFLICT (user_id, institution_id) DO UPDATE SET active = TRUE", institutionID},
		{"INSERT INTO user_site_memberships (user_id, site_id) VALUES ($1, $2) ON CONFLICT (user_id, site_id) DO UPDATE SET active = TRUE", siteID},
		{"INSERT INTO user_firm_memberships (user_id, firm_id) VALUES ($1, $2) ON CONFLICT (user_id, firm_id) DO UPDATE SET active = TRUE", firmID},
	}
	for _, membership := range memberships {
		if _, err := tx.Exec(ctx, membership.query, userID, membership.id); err != nil {
			return fmt.Errorf("upsert development membership: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM user_firm_preferences
		WHERE user_id = $1 AND (firm_id = $2 OR position = 0)`, userID, firmID); err != nil {
		return fmt.Errorf("replace development firm preference: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_firm_preferences (user_id, firm_id, position)
		VALUES ($1, $2, 0)`, userID, firmID); err != nil {
		return fmt.Errorf("upsert development firm preference: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO roles (name, description, scope, active)
		VALUES (
			'Development Patient Search Tester',
			'Synthetic development-only patient search and duplicate-check access',
			'institution',
			TRUE
		)
		ON CONFLICT (name) DO UPDATE SET
			description = EXCLUDED.description,
			active = TRUE`); err != nil {
		return fmt.Errorf("upsert development patient-search role: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_id, active)
		SELECT r.id, p.id, TRUE
		FROM roles r
		JOIN permissions p ON p.name IN ('patient.search', 'patient.duplicate_check', 'patient.summary.read', 'patient.clinical_summary.read', 'patient.break_glass', 'patient.break_glass.revoke')
		WHERE r.name = 'Development Patient Search Tester'
		ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE`); err != nil {
		return fmt.Errorf("upsert development patient-search permissions: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_role_assignments (user_id, role_id, role_scope, institution_id)
		SELECT $1, id, scope, $2 FROM roles
		WHERE name IN ('VisionOpus User', 'Development Patient Search Tester')
		ON CONFLICT (user_id, role_id, institution_id) DO UPDATE SET active = TRUE`, userID, institutionID); err != nil {
		return fmt.Errorf("upsert development roles: %w", err)
	}

	var patientID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO patients (
			public_id, given_name, given_name_normalized, family_name,
			family_name_normalized, date_of_birth, gender, source_system,
			source_record_id, created_by_user_id, updated_by_user_id
		) VALUES ('11111111-1111-4111-8111-111111111111', 'Alice', 'alice',
			'Patient', 'patient', '1985-04-12', 'female', 'visionopus-dev',
			'patient-001', $1, $1)
		ON CONFLICT (public_id) DO UPDATE SET
			given_name = EXCLUDED.given_name,
			given_name_normalized = EXCLUDED.given_name_normalized,
			family_name = EXCLUDED.family_name,
			family_name_normalized = EXCLUDED.family_name_normalized,
			active = TRUE, deleted_at = NULL, version = patients.version + 1,
			updated_at = now(), updated_by_user_id = EXCLUDED.updated_by_user_id
		RETURNING id`, userID).Scan(&patientID); err != nil {
		return fmt.Errorf("upsert synthetic patient: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_institutions (patient_id, institution_id, primary_association, association_source, created_by_user_id, updated_by_user_id)
		VALUES ($1, $2, TRUE, 'approved_deployment_default', $3, $3)
		ON CONFLICT (patient_id, institution_id) WHERE active DO UPDATE SET active = TRUE, updated_at = now(), updated_by_user_id = EXCLUDED.updated_by_user_id`, patientID, institutionID, userID); err != nil {
		return fmt.Errorf("upsert synthetic patient association: %w", err)
	}
	identifierTypeID, err := findOrInsert(ctx, tx,
		"SELECT id FROM patient_identifier_types WHERE institution_id = $1 AND site_id IS NULL AND stable_code = $2",
		`INSERT INTO patient_identifier_types (stable_code, institution_id, display_label, normalization_kind, maximum_canonical_length, validation_state, source_system, source_record_id)
		 VALUES ($2, $1, 'NHS number', 'nhs_number_v1', 10, 'validated', 'visionopus-dev', 'identifier-type-nhs') RETURNING id`, institutionID, "nhs_number")
	if err != nil {
		return fmt.Errorf("upsert synthetic identifier type: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_identifiers (patient_id, identifier_type_id, original_value, canonical_value, source_marker, source_system, source_record_id)
		VALUES ($1, $2, '9434765919', '9434765919', 'visionopus-dev', 'visionopus-dev', 'patient-identifier-001')
		ON CONFLICT (source_system, source_record_id) DO UPDATE SET active = TRUE, lifecycle = 'active', deleted_at = NULL, updated_at = now()`, patientID, identifierTypeID); err != nil {
		return fmt.Errorf("upsert synthetic patient identifier: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_summary_warning_projections (patient_id, allergy_status, alert_status, source_revision, source_checksum, projection_state)
		VALUES ($1, 'present', 'none_known', 'visionopus-dev-1', 'synthetic-checksum-001', 'verified')
		ON CONFLICT (patient_id) DO UPDATE SET allergy_status = EXCLUDED.allergy_status, alert_status = EXCLUDED.alert_status, source_revision = EXCLUDED.source_revision, source_checksum = EXCLUDED.source_checksum, projection_state = EXCLUDED.projection_state, warning_version = patient_summary_warning_projections.warning_version + 1, updated_at = now()`, patientID); err != nil {
		return fmt.Errorf("upsert synthetic warning projection: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM patient_summary_warning_items WHERE patient_id = $1`, patientID); err != nil {
		return fmt.Errorf("reset synthetic warning items: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_summary_warning_items (patient_id, warning_kind, code, label, reaction, item_order, source_revision)
		VALUES ($1, 'allergy', 'peanuts', 'Peanut allergy', 'Urticaria', 0, 'visionopus-dev-1')`, patientID); err != nil {
		return fmt.Errorf("insert synthetic warning item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit development seed: %w", err)
	}
	_, _ = fmt.Fprintf(os.Stdout, "seeded synthetic user %q for institution %d, site %d, firm %d\n", username, institutionID, siteID, firmID)
	return nil
}

func findOrInsert(ctx context.Context, tx pgx.Tx, findQuery, insertQuery string, args ...any) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, findQuery, args...).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("find development record: %w", err)
	}
	if err := tx.QueryRow(ctx, insertQuery, args...).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert development record: %w", err)
	}
	return id, nil
}

func valueOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
