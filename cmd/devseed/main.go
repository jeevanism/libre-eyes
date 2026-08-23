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
	if err := requireDevelopmentEnvironment(cfg.Environment); err != nil {
		return err
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
	// Integration tests intentionally create isolated institutions in the shared
	// development database. Keep the interactive demo focused on its one
	// synthetic context without deleting those test records.
	if _, err := tx.Exec(ctx, `
		UPDATE institutions
		SET active = (id = $1), updated_at = now()
		WHERE active <> (id = $1)`, institutionID); err != nil {
		return fmt.Errorf("focus development login institutions: %w", err)
	}
	siteID, err := findOrInsert(ctx, tx,
		"SELECT id FROM sites WHERE institution_id = $1 AND name = $2",
		"INSERT INTO sites (institution_id, name) VALUES ($1, $2) RETURNING id",
		institutionID, "Development Eye Clinic",
	)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sites
		SET active = (id = $1), updated_at = now()
		WHERE institution_id = $2 AND active <> (id = $1)`, siteID, institutionID); err != nil {
		return fmt.Errorf("focus development login sites: %w", err)
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
			'Synthetic development-only patient, summary, and episode access',
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
		JOIN permissions p ON p.name IN (
			'patient.search', 'patient.duplicate_check', 'patient.summary.read', 'patient.clinical_summary.read',
			'patient.break_glass', 'patient.break_glass.revoke',
			'episode.read', 'episode.create', 'episode.update', 'episode.reopen',
			'event_draft.create', 'event_draft.read', 'event_draft.update', 'event_draft.abandon',
			'worklist.development_flow.manage', 'theatre.development_booking.manage', 'referral.development_appointment.manage'
		)
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
			date_of_birth = EXCLUDED.date_of_birth,
			gender = EXCLUDED.gender,
			source_system = EXCLUDED.source_system,
			source_record_id = EXCLUDED.source_record_id,
			active = TRUE, deleted_at = NULL, version = patients.version + 1,
			updated_at = now(), updated_by_user_id = EXCLUDED.updated_by_user_id
		RETURNING id`, userID).Scan(&patientID); err != nil {
		return fmt.Errorf("upsert synthetic patient: %w", err)
	}
	var patientInstitutionID int64
	// The synthetic public ID is development-only. Normalize it to this one
	// development institution so local test fixtures cannot leave a conflicting
	// active primary association behind.
	if _, err := tx.Exec(ctx, `
		UPDATE patient_institutions
		SET active = FALSE, primary_association = FALSE, effective_to = now(), updated_at = now(), updated_by_user_id = $2
		WHERE patient_id = $1 AND institution_id <> $3 AND active`, patientID, userID, institutionID); err != nil {
		return fmt.Errorf("retire non-development synthetic patient associations: %w", err)
	}
	err = tx.QueryRow(ctx, `
		SELECT id FROM patient_institutions
		WHERE patient_id = $1 AND institution_id = $2 AND active
		ORDER BY id DESC LIMIT 1`, patientID, institutionID).Scan(&patientInstitutionID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, `
			INSERT INTO patient_institutions (patient_id, institution_id, primary_association, association_source, created_by_user_id, updated_by_user_id)
			VALUES ($1, $2, TRUE, 'approved_deployment_default', $3, $3)
			RETURNING id`, patientID, institutionID, userID).Scan(&patientInstitutionID); err != nil {
			return fmt.Errorf("insert synthetic patient association: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find synthetic patient association: %w", err)
	} else if _, err := tx.Exec(ctx, `
		UPDATE patient_institutions
		SET primary_association = TRUE, updated_at = now(), updated_by_user_id = $2
		WHERE id = $1`, patientInstitutionID, userID); err != nil {
		return fmt.Errorf("refresh synthetic patient association: %w", err)
	}
	identifierTypeID, err := findOrInsert(ctx, tx,
		"SELECT id FROM patient_identifier_types WHERE institution_id = $1 AND site_id IS NULL AND stable_code = $2",
		`INSERT INTO patient_identifier_types (stable_code, institution_id, display_label, normalization_kind, maximum_canonical_length, validation_state, display_order, source_system, source_record_id)
		 VALUES ($2, $1, 'NHS number', 'nhs_number_v1', 10, 'validated', 0, 'visionopus-dev', 'identifier-type-nhs') RETURNING id`, institutionID, "nhs_number")
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
	if _, err := tx.Exec(ctx, `
		INSERT INTO episodes (
			public_id, patient_id, institution_id, patient_institution_id, site_id, firm_id, status,
			started_at, ended_at, created_by_user_id, updated_by_user_id
		) VALUES
			('22222222-2222-4222-8222-222222222222', $1, $2, $3, $4, $5, 'active', now() - interval '14 days', NULL, $6, $6),
			('33333333-3333-4333-8333-333333333333', $1, $2, $3, $4, $5, 'closed', now() - interval '60 days', now() - interval '45 days', $6, $6)
		ON CONFLICT (public_id) DO UPDATE SET
			patient_id = EXCLUDED.patient_id, institution_id = EXCLUDED.institution_id,
			patient_institution_id = EXCLUDED.patient_institution_id, site_id = EXCLUDED.site_id, firm_id = EXCLUDED.firm_id,
			status = EXCLUDED.status, started_at = EXCLUDED.started_at, ended_at = EXCLUDED.ended_at,
			deleted_at = NULL, deleted_by_user_id = NULL, deleted_reason = NULL,
			updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = now()`,
		patientID, institutionID, patientInstitutionID, siteID, firmID, userID); err != nil {
		return fmt.Errorf("upsert synthetic episodes: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO episode_audit_sequences (episode_id)
		SELECT id FROM episodes WHERE public_id IN (
			'22222222-2222-4222-8222-222222222222', '33333333-3333-4333-8333-333333333333'
		)
		ON CONFLICT (episode_id) DO NOTHING`); err != nil {
		return fmt.Errorf("initialize synthetic episode audit sequences: %w", err)
	}
	for _, event := range []struct {
		publicID  string
		episodeID string
		typeCode  string
		offset    string
	}{
		{"44444444-4444-4444-8444-444444444444", "22222222-2222-4222-8222-222222222222", "core.examination", "13 days"},
		{"55555555-5555-4555-8555-555555555555", "22222222-2222-4222-8222-222222222222", "core.follow_up", "7 days"},
		{"66666666-6666-4666-8666-666666666666", "33333333-3333-4333-8333-333333333333", "core.examination", "50 days"},
	} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO events (
				public_id, episode_id, patient_id, institution_id, patient_institution_id, site_id, firm_id,
				event_type_code, occurred_at, created_by_user_id, updated_by_user_id
			)
			SELECT $1::uuid, e.id, e.patient_id, e.institution_id, e.patient_institution_id, e.site_id, e.firm_id,
				$2, now() - $3::interval, $4, $4
			FROM episodes e WHERE e.public_id = $5::uuid
			ON CONFLICT (public_id) DO UPDATE SET
				episode_id = EXCLUDED.episode_id, patient_id = EXCLUDED.patient_id,
				institution_id = EXCLUDED.institution_id, patient_institution_id = EXCLUDED.patient_institution_id,
				site_id = EXCLUDED.site_id, firm_id = EXCLUDED.firm_id,
				event_type_code = EXCLUDED.event_type_code, occurred_at = EXCLUDED.occurred_at,
				status = 'current', deletion_reason = NULL, deleted_at = NULL, deleted_by_user_id = NULL,
				updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = now()`,
			event.publicID, event.typeCode, event.offset, userID, event.episodeID); err != nil {
			return fmt.Errorf("upsert synthetic event header: %w", err)
		}
	}
	if err := seedDevelopmentVisualAcuityCatalogue(ctx, tx, userID); err != nil {
		return err
	}
	if err := seedDevelopmentIOPCatalogue(ctx, tx, userID); err != nil {
		return err
	}
	if err := seedDevelopmentDiagnosisCatalogue(ctx, tx, userID); err != nil {
		return err
	}
	if err := seedDevelopmentClinicFlow(ctx, tx, institutionID, siteID, firmID); err != nil {
		return err
	}
	if err := seedDevelopmentTheatreBooking(ctx, tx, institutionID, siteID, firmID); err != nil {
		return err
	}
	if err := seedDevelopmentConsentCatalogue(ctx, tx, userID); err != nil {
		return err
	}
	if err := seedDevelopmentCorrespondenceCatalogue(ctx, tx); err != nil {
		return err
	}
	if err := seedDevelopmentLabResultsCatalogue(ctx, tx); err != nil {
		return err
	}
	if err := seedDevelopmentReferralAppointments(ctx, tx, institutionID, siteID, firmID, userID); err != nil {
		return err
	}
	if err := seedDevelopmentAdmin(ctx, tx, institutionID, userID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit development seed: %w", err)
	}
	_, _ = fmt.Fprintf(os.Stdout, "seeded synthetic user %q for institution %d, site %d, firm %d\n", username, institutionID, siteID, firmID)
	return nil
}

func seedDevelopmentAdmin(ctx context.Context, tx pgx.Tx, institutionID, actorID int64) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_id, active)
		SELECT r.id, p.id, TRUE FROM roles r CROSS JOIN permissions p
		WHERE r.name = 'Development Patient Search Tester'
		  AND p.name IN ('admin.development.read','admin.development.manage')
		ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE`); err != nil {
		return fmt.Errorf("grant development admin permissions: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO development_admin_users (public_id, institution_id, user_id, username, display_name, role_code)
		VALUES
		 ('90000000-0000-4000-8000-000000000001',$1,$2,'admin.demo','Demo Institution Administrator','institution_administrator'),
		 ('90000000-0000-4000-8000-000000000002',$1,$2,'clinical.demo','Demo Clinical User','clinical_user')
		ON CONFLICT (public_id) DO UPDATE SET institution_id=EXCLUDED.institution_id,user_id=EXCLUDED.user_id,display_name=EXCLUDED.display_name,active=TRUE,version=development_admin_users.version+1,updated_at=now()`, institutionID, actorID); err != nil {
		return fmt.Errorf("seed development admin users: %w", err)
	}
	for key, value := range map[string]string{"default_site": "Development Eye Clinic", "default_firm": "Development Ophthalmology", "appointment_slot_minutes": "30", "demo_retention_days": "7"} {
		if _, err := tx.Exec(ctx, `INSERT INTO development_admin_settings (key,institution_id,value) VALUES ($1,$2,$3) ON CONFLICT (key) DO UPDATE SET institution_id=EXCLUDED.institution_id,value=EXCLUDED.value,updated_at=now()`, key, institutionID, value); err != nil {
			return fmt.Errorf("seed development admin setting: %w", err)
		}
	}
	return nil
}

func seedDevelopmentLabResultsCatalogue(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM development_lab_results_catalogue WHERE code LIKE 'demo_lab\_%'`); err != nil {
		return fmt.Errorf("reset lab results catalogue: %w", err)
	}
	rows := []struct {
		code, name, kind, unit, choices        string
		hardMin, hardMax, normalMin, normalMax *float64
		order                                  int
	}{
		{code: "demo_lab_hba1c", name: "Demo HbA1c", kind: "numeric", unit: "%", hardMin: ptrFloat(0), hardMax: ptrFloat(20), normalMin: ptrFloat(4), normalMax: ptrFloat(6), order: 0},
		{code: "demo_lab_creatinine", name: "Demo serum creatinine", kind: "numeric", unit: "umol/L", hardMin: ptrFloat(0), hardMax: ptrFloat(2000), normalMin: ptrFloat(45), normalMax: ptrFloat(110), order: 1},
		{code: "demo_lab_status", name: "Demo laboratory status", kind: "choice", choices: `["pending","complete","not available"]`, order: 2},
	}
	for _, row := range rows {
		if _, err := tx.Exec(ctx, `INSERT INTO development_lab_results_catalogue (code,display_name,field_kind,default_unit,hard_min,hard_max,normal_min,normal_max,choices,display_order) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10)`, row.code, row.name, row.kind, row.unit, row.hardMin, row.hardMax, row.normalMin, row.normalMax, valueOrDefault(row.choices, `[]`), row.order); err != nil {
			return fmt.Errorf("seed lab result catalogue: %w", err)
		}
	}
	return nil
}

func ptrFloat(value float64) *float64 { return &value }

func seedDevelopmentReferralAppointments(ctx context.Context, tx pgx.Tx, institutionID, siteID, firmID, userID int64) error {
	rows := []struct{ id, label, role, clinic, date, priority string }{
		{"77777777-7777-4777-8777-777777777771", "Demo referral to GP", "demo_gp", "demo_general_eye_clinic", "2026-08-18", "routine"},
		{"77777777-7777-4777-8777-777777777772", "Demo optometry follow-up", "demo_optometrist", "demo_glaucoma_clinic", "2026-08-19", "soon"},
		{"77777777-7777-4777-8777-777777777773", "Demo consultant review", "demo_consultant", "demo_retina_clinic", "2026-08-20", "urgent"},
	}
	for _, r := range rows {
		if _, err := tx.Exec(ctx, `INSERT INTO development_referral_appointments (id,institution_id,site_id,firm_id,synthetic_patient_id,synthetic_patient_label,recipient_role,clinic_code,appointment_date,appointment_time,priority,notes,owner_user_id,expires_at) VALUES ($1::uuid,$2,$3,$4,'11111111-1111-4111-8111-111111111111',$5,$6,$7,$8::date,'09:00',$9,'Synthetic demonstration referral',$10,now()+interval '7 days') ON CONFLICT (id) DO UPDATE SET institution_id=EXCLUDED.institution_id,site_id=EXCLUDED.site_id,firm_id=EXCLUDED.firm_id,synthetic_patient_label=EXCLUDED.synthetic_patient_label,recipient_role=EXCLUDED.recipient_role,clinic_code=EXCLUDED.clinic_code,appointment_date=EXCLUDED.appointment_date,appointment_time=EXCLUDED.appointment_time,priority=EXCLUDED.priority,notes=EXCLUDED.notes,owner_user_id=EXCLUDED.owner_user_id,status='requested',version=1,expires_at=EXCLUDED.expires_at,updated_at=now()`, r.id, institutionID, siteID, firmID, r.label, r.role, r.clinic, r.date, r.priority, userID); err != nil {
			return fmt.Errorf("seed synthetic referral appointment: %w", err)
		}
	}
	return nil
}

func seedDevelopmentCorrespondenceCatalogue(ctx context.Context, tx pgx.Tx) error {
	rows := []struct {
		category, code, name string
		order                int
	}{
		{"template", "demo_clinic_update", "Demo clinic update", 0},
		{"template", "demo_referral_summary", "Demo referral summary", 1},
		{"template", "demo_follow_up", "Demo follow-up letter", 2},
		{"recipient_role", "demo_gp", "Demo GP", 0},
		{"recipient_role", "demo_optometrist", "Demo optometrist", 1},
		{"recipient_role", "demo_consultant", "Demo consultant", 2},
	}
	if _, err := tx.Exec(ctx, `DELETE FROM development_correspondence_catalogue WHERE code LIKE 'demo\_%'`); err != nil {
		return fmt.Errorf("reset development correspondence catalogue: %w", err)
	}
	for _, row := range rows {
		if _, err := tx.Exec(ctx, `INSERT INTO development_correspondence_catalogue (category, code, display_name, display_order) VALUES ($1,$2,$3,$4) ON CONFLICT (code) DO UPDATE SET category=EXCLUDED.category, display_name=EXCLUDED.display_name, display_order=EXCLUDED.display_order, active=TRUE`, row.category, row.code, row.name, row.order); err != nil {
			return fmt.Errorf("upsert correspondence catalogue: %w", err)
		}
	}
	return nil
}

func seedDevelopmentConsentCatalogue(ctx context.Context, tx pgx.Tx, userID int64) error {
	rows := []struct {
		category, code, name string
		order                int
	}{
		{"form_type", "development_form_type_1", "Demo consent form type 1", 0},
		{"form_type", "development_form_type_2", "Demo consent form type 2", 1},
		{"form_type", "development_form_type_3", "Demo consent form type 3", 2},
		{"form_type", "development_form_type_4", "Demo consent form type 4", 3},
		{"procedure", "development_cataract_extraction", "Demo cataract extraction", 0},
		{"procedure", "development_trabeculectomy", "Demo trabeculectomy", 1},
		{"laterality", "development_left_eye", "Left eye", 0},
		{"laterality", "development_right_eye", "Right eye", 1},
		{"laterality", "development_both_eyes", "Both eyes", 2},
		{"anaesthetic", "development_local_anaesthetic", "Demo local anaesthetic", 0},
		{"anaesthetic", "development_general_anaesthetic", "Demo general anaesthetic", 1},
		{"anaesthetic", "development_no_anaesthetic", "Demo no anaesthetic", 2},
	}
	if _, err := tx.Exec(ctx, `DELETE FROM development_consent_catalogue WHERE code LIKE 'development\_%'`); err != nil {
		return fmt.Errorf("reset development consent catalogue: %w", err)
	}
	for _, row := range rows {
		if _, err := tx.Exec(ctx, `INSERT INTO development_consent_catalogue (category, code, display_name, display_order) VALUES ($1,$2,$3,$4) ON CONFLICT (code) DO UPDATE SET category=EXCLUDED.category, display_name=EXCLUDED.display_name, display_order=EXCLUDED.display_order, active=TRUE`, row.category, row.code, row.name, row.order); err != nil {
			return fmt.Errorf("upsert consent catalogue: %w", err)
		}
	}
	_ = userID
	return nil
}

func requireDevelopmentEnvironment(environment string) error {
	if environment != "development" {
		return errors.New("devseed may run only when VISIONOPUS_ENV=development")
	}
	return nil
}

func seedDevelopmentVisualAcuityCatalogue(ctx context.Context, tx pgx.Tx, userID int64) error {
	var unitID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO visual_acuity_units (code, display_name, active, display_order, created_by_user_id, updated_by_user_id)
		VALUES ('development_distance_scale', 'Development LogMAR distance 4 m demonstration', TRUE, 0, $1, $1)
		ON CONFLICT (code) DO UPDATE SET
			display_name = EXCLUDED.display_name, active = TRUE, display_order = EXCLUDED.display_order,
			updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = now()
		RETURNING id`, userID).Scan(&unitID); err != nil {
		return fmt.Errorf("upsert development visual acuity unit: %w", err)
	}
	for _, value := range developmentVisualAcuityCatalogueValues() {
		if _, err := tx.Exec(ctx, `
			INSERT INTO visual_acuity_unit_values (unit_id, code, display_value, base_value, selectable, active, display_order, created_by_user_id, updated_by_user_id)
			VALUES ($1, $2, $3, $4::numeric, TRUE, TRUE, $5, $6, $6)
			ON CONFLICT (code) DO UPDATE SET
				unit_id = EXCLUDED.unit_id, display_value = EXCLUDED.display_value, base_value = EXCLUDED.base_value,
				selectable = TRUE, active = TRUE, display_order = EXCLUDED.display_order,
				updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = now()`,
			unitID, value.code, value.displayValue, value.baseValue, value.order, userID); err != nil {
			return fmt.Errorf("upsert development visual acuity value: %w", err)
		}
	}
	for _, method := range []struct {
		code, displayName, category string
		order                       int
	}{
		{"development_unaided", "Development unaided", "unaided", 0},
		{"development_habitual", "Development habitual correction", "habitual", 1},
		{"development_best_corrected", "Development best-corrected", "best_corrected", 2},
		{"development_pinhole", "Development pinhole", "pinhole", 3},
	} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO visual_acuity_methods (code, display_name, correction_category, active, display_order, created_by_user_id, updated_by_user_id)
			VALUES ($1, $2, $3, TRUE, $4, $5, $5)
			ON CONFLICT (code) DO UPDATE SET
				display_name = EXCLUDED.display_name, correction_category = EXCLUDED.correction_category,
				active = TRUE, display_order = EXCLUDED.display_order,
				updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = now()`,
			method.code, method.displayName, method.category, method.order, userID); err != nil {
			return fmt.Errorf("upsert development visual acuity method: %w", err)
		}
	}
	return nil
}

type developmentVisualAcuityValue struct {
	code         string
	displayValue string
	baseValue    string
	order        int
}

func developmentVisualAcuityCatalogueValues() []developmentVisualAcuityValue {
	const (
		firstHundredths = -30
		lastHundredths  = 150
		stepHundredths  = 2
	)

	values := make([]developmentVisualAcuityValue, 0, (lastHundredths-firstHundredths)/stepHundredths+1)
	for hundredths := firstHundredths; hundredths <= lastHundredths; hundredths += stepHundredths {
		values = append(values, developmentVisualAcuityValue{
			code:         developmentVisualAcuityValueCode(hundredths),
			displayValue: formatVisualAcuityHundredths(hundredths, 2),
			baseValue:    formatVisualAcuityHundredths(hundredths, 4),
			order:        len(values),
		})
	}
	return values
}

func developmentVisualAcuityValueCode(hundredths int) string {
	if hundredths < 0 {
		return fmt.Sprintf("development_value_m%03d", -hundredths)
	}
	return fmt.Sprintf("development_value_%03d", hundredths)
}

func formatVisualAcuityHundredths(hundredths, decimalPlaces int) string {
	sign := ""
	if hundredths < 0 {
		sign = "-"
		hundredths = -hundredths
	}
	whole := hundredths / 100
	fraction := hundredths % 100
	if decimalPlaces == 2 {
		return fmt.Sprintf("%s%d.%02d", sign, whole, fraction)
	}
	return fmt.Sprintf("%s%d.%04d", sign, whole, fraction*100)
}

func seedDevelopmentIOPCatalogue(ctx context.Context, tx pgx.Tx, userID int64) error {
	var profileID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO intraocular_pressure_profiles (code, display_name, active, display_order, created_by_user_id, updated_by_user_id)
		VALUES ('development_iop_manual_mmhg', 'Development manual IOP demonstration', TRUE, 0, $1, $1)
		ON CONFLICT (code) DO UPDATE SET
			display_name = EXCLUDED.display_name, active = TRUE, display_order = EXCLUDED.display_order,
			updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = now()
		RETURNING id`, userID).Scan(&profileID); err != nil {
		return fmt.Errorf("upsert development intraocular pressure profile: %w", err)
	}
	for _, value := range developmentIOPCatalogueValues() {
		if _, err := tx.Exec(ctx, `
			INSERT INTO intraocular_pressure_profile_values (
				profile_id, code, display_value, mmhg, selectable, active, display_order, created_by_user_id, updated_by_user_id
			) VALUES ($1, $2, $3, $4::smallint, TRUE, TRUE, $4::integer, $5, $5)
			ON CONFLICT (code) DO UPDATE SET
				profile_id = EXCLUDED.profile_id, display_value = EXCLUDED.display_value, mmhg = EXCLUDED.mmhg,
				selectable = TRUE, active = TRUE, display_order = EXCLUDED.display_order,
				updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = now()`,
			profileID, value.code, value.displayValue, value.mmhg, userID); err != nil {
			return fmt.Errorf("upsert development intraocular pressure value: %w", err)
		}
	}
	return nil
}

type developmentIOPValue struct {
	code, displayValue string
	mmhg               int
}

func developmentIOPCatalogueValues() []developmentIOPValue {
	values := make([]developmentIOPValue, 0, 100)
	for mmhg := 0; mmhg <= 99; mmhg++ {
		values = append(values, developmentIOPValue{
			code:         fmt.Sprintf("development_iop_%02d", mmhg),
			displayValue: fmt.Sprintf("%d mmHg", mmhg),
			mmhg:         mmhg,
		})
	}
	return values
}

func seedDevelopmentDiagnosisCatalogue(ctx context.Context, tx pgx.Tx, userID int64) error {
	var profileID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO development_diagnosis_profiles (code, display_name, active, display_order, created_by_user_id, updated_by_user_id)
		VALUES ('development_ophthalmology_diagnosis_v1', 'Development ophthalmology selection demonstration', TRUE, 0, $1, $1)
		ON CONFLICT (code) DO UPDATE SET display_name = EXCLUDED.display_name, active = TRUE, display_order = EXCLUDED.display_order, updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = now()
		RETURNING id`, userID).Scan(&profileID); err != nil {
		return fmt.Errorf("upsert development diagnosis profile: %w", err)
	}
	for _, selection := range developmentDiagnosisSelections() {
		if _, err := tx.Exec(ctx, `
			INSERT INTO development_diagnosis_profile_selections (profile_id, code, display_name, selectable, active, display_order, created_by_user_id, updated_by_user_id)
			VALUES ($1, $2, $3, TRUE, TRUE, $4, $5, $5)
			ON CONFLICT (code) DO UPDATE SET profile_id = EXCLUDED.profile_id, display_name = EXCLUDED.display_name, selectable = TRUE, active = TRUE, display_order = EXCLUDED.display_order, updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = now()`,
			profileID, selection.code, selection.displayName, selection.order, userID); err != nil {
			return fmt.Errorf("upsert development diagnosis selection: %w", err)
		}
	}
	return nil
}

type developmentDiagnosisSelection struct {
	code, displayName string
	order             int
}

func developmentDiagnosisSelections() []developmentDiagnosisSelection {
	return []developmentDiagnosisSelection{
		{code: "development_cataract", displayName: "Development cataract example", order: 0},
		{code: "development_glaucoma", displayName: "Development glaucoma example", order: 1},
		{code: "development_macular_condition", displayName: "Development macular-condition example", order: 2},
	}
}

func seedDevelopmentClinicFlow(ctx context.Context, tx pgx.Tx, institutionID, siteID, firmID int64) error {
	for _, ticket := range []struct {
		id, patientID, label string
	}{
		{"77777777-7777-4777-8777-777777777777", "development-flow-patient-001", "Synthetic queue patient A"},
		{"88888888-8888-4888-8888-888888888888", "development-flow-patient-002", "Synthetic queue patient B"},
		{"99999999-9999-4999-8999-999999999999", "development-flow-patient-003", "Synthetic queue patient C"},
	} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO development_flow_tickets (
				id, institution_id, site_id, firm_id, synthetic_patient_id, synthetic_patient_label, status
			) VALUES ($1::uuid,$2,$3,$4,$5,$6,'waiting')
			ON CONFLICT (id) DO UPDATE SET
				institution_id = EXCLUDED.institution_id, site_id = EXCLUDED.site_id, firm_id = EXCLUDED.firm_id,
				synthetic_patient_id = EXCLUDED.synthetic_patient_id, synthetic_patient_label = EXCLUDED.synthetic_patient_label,
				status = 'waiting', assignee_user_id = NULL,
				version = CASE WHEN development_flow_tickets.status <> 'waiting' OR development_flow_tickets.assignee_user_id IS NOT NULL THEN development_flow_tickets.version + 1 ELSE development_flow_tickets.version END,
				updated_at = now()`,
			ticket.id, institutionID, siteID, firmID, ticket.patientID, ticket.label); err != nil {
			return fmt.Errorf("upsert synthetic development clinic-flow ticket: %w", err)
		}
	}
	return nil
}

func seedDevelopmentTheatreBooking(ctx context.Context, tx pgx.Tx, institutionID, siteID, firmID int64) error {
	const roomID = "a1111111-1111-4111-8111-111111111111"
	if _, err := tx.Exec(ctx, `
		INSERT INTO development_theatre_rooms (id, institution_id, site_id, firm_id, synthetic_label)
		VALUES ($1::uuid,$2,$3,$4,'Development Theatre One')
		ON CONFLICT (id) DO UPDATE SET institution_id = EXCLUDED.institution_id, site_id = EXCLUDED.site_id,
			firm_id = EXCLUDED.firm_id, synthetic_label = EXCLUDED.synthetic_label`, roomID, institutionID, siteID, firmID); err != nil {
		return fmt.Errorf("upsert synthetic theatre room: %w", err)
	}
	for _, session := range []struct {
		id, startsAt, endsAt string
		capacity             int
	}{
		{"b1111111-1111-4111-8111-111111111111", "2026-08-10T08:00:00Z", "2026-08-10T12:00:00Z", 180},
		{"c1111111-1111-4111-8111-111111111111", "2026-08-10T13:00:00Z", "2026-08-10T17:00:00Z", 180},
	} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO development_theatre_sessions (id, room_id, institution_id, site_id, firm_id, starts_at, ends_at, capacity_minutes)
			VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6::timestamptz,$7::timestamptz,$8)
			ON CONFLICT (id) DO UPDATE SET room_id = EXCLUDED.room_id, institution_id = EXCLUDED.institution_id,
				site_id = EXCLUDED.site_id, firm_id = EXCLUDED.firm_id, starts_at = EXCLUDED.starts_at,
				ends_at = EXCLUDED.ends_at, capacity_minutes = EXCLUDED.capacity_minutes,
				updated_at = development_theatre_sessions.updated_at`, session.id, roomID, institutionID, siteID, firmID, session.startsAt, session.endsAt, session.capacity); err != nil {
			return fmt.Errorf("upsert synthetic theatre session: %w", err)
		}
	}
	for _, request := range []struct {
		id, label string
		duration  int
	}{
		{"d1111111-1111-4111-8111-111111111111", "Synthetic booking request A", 60},
		{"e1111111-1111-4111-8111-111111111111", "Synthetic booking request B", 90},
		{"f1111111-1111-4111-8111-111111111111", "Synthetic booking request C", 120},
	} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO development_booking_requests (id, institution_id, site_id, firm_id, synthetic_label, requested_duration_minutes)
			VALUES ($1::uuid,$2,$3,$4,$5,$6)
			ON CONFLICT (id) DO UPDATE SET institution_id = EXCLUDED.institution_id, site_id = EXCLUDED.site_id,
				firm_id = EXCLUDED.firm_id, synthetic_label = EXCLUDED.synthetic_label,
				requested_duration_minutes = EXCLUDED.requested_duration_minutes,
				status = 'waiting', assigned_session_id = NULL,
				version = CASE WHEN development_booking_requests.status <> 'waiting' OR development_booking_requests.assigned_session_id IS NOT NULL THEN development_booking_requests.version + 1 ELSE development_booking_requests.version END,
				updated_at = now()`, request.id, institutionID, siteID, firmID, request.label, request.duration); err != nil {
			return fmt.Errorf("upsert synthetic booking request: %w", err)
		}
	}
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
