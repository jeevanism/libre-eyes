//go:build integration

package migrations

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPatientSearchFoundationConstraints(t *testing.T) {
	pool := patientSearchTestPool(t)

	t.Run("permission exists", func(t *testing.T) {
		var active bool
		if err := pool.QueryRow(context.Background(), `
			SELECT active FROM permissions WHERE name = 'patient.duplicate_check'`,
		).Scan(&active); err != nil {
			t.Fatalf("query duplicate-check permission: %v", err)
		}
		if !active {
			t.Fatal("patient.duplicate_check permission is inactive")
		}
	})

	t.Run("demographic search index exists", func(t *testing.T) {
		var exists bool
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public'
					AND indexname = 'patients_demographics_search_idx'
			)`,
		).Scan(&exists); err != nil {
			t.Fatalf("query demographic search index: %v", err)
		}
		if !exists {
			t.Fatal("patients_demographics_search_idx does not exist")
		}
	})

	t.Run("zero-padding configuration requires width", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			institutionID := insertInstitution(ctx, t, tx, "Padding Hospital")
			_, err := tx.Exec(ctx, `
				INSERT INTO patient_identifier_types (
					stable_code, institution_id, display_label, normalization_kind,
					validation_pattern, maximum_canonical_length, zero_pad_width,
					validation_state, display_order, source_system, source_record_id
				) VALUES (
					'hospital-number', $1, 'Hospital number',
					'left_zero_pad_ascii_digits_v1', '^[0-9]+$', 10, NULL,
					'validated', 0, 'test', 'missing-pad-width'
				)`, institutionID)
			expectPostgresCode(t, err, "23514")
		})
	})

	t.Run("public identifier must be uuid v4", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			_, err := insertPatient(ctx, tx, "11111111-1111-5111-8111-111111111111", "uuid-v5")
			expectPostgresCode(t, err, "23514")
		})
	})

	t.Run("death date cannot precede birth", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			_, err := tx.Exec(ctx, `
				INSERT INTO patients (
					public_id, given_name, given_name_normalized,
					family_name, family_name_normalized, date_of_birth,
					gender, deceased, date_of_death, source_system, source_record_id
				) VALUES (
					'11111111-1111-4111-8111-111111111112',
					'Synthetic', 'synthetic', 'Patient', 'patient',
					DATE '1980-01-02', 'unknown', TRUE, DATE '1980-01-01',
					'test', 'invalid-death-date'
				)`)
			expectPostgresCode(t, err, "23514")
		})
	})

	t.Run("death date requires deceased state", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			_, err := tx.Exec(ctx, `
				INSERT INTO patients (
					public_id, given_name, given_name_normalized,
					family_name, family_name_normalized, date_of_birth,
					gender, deceased, date_of_death, source_system, source_record_id
				) VALUES (
					'11111111-1111-4111-8111-111111111120',
					'Synthetic', 'synthetic', 'Patient', 'patient',
					DATE '1980-01-01', 'unknown', FALSE, DATE '2026-01-01',
					'test', 'death-without-deceased-state'
				)`)
			expectPostgresCode(t, err, "23514")
		})
	})

	t.Run("active institution pair is unique", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			institutionID := insertInstitution(ctx, t, tx, "Pair Hospital")
			patientID, err := insertPatient(ctx, tx, "11111111-1111-4111-8111-111111111113", "pair-patient")
			if err != nil {
				t.Fatalf("insert patient: %v", err)
			}
			insertPatientInstitution(ctx, t, tx, patientID, institutionID, false)
			_, err = tx.Exec(ctx, `
				INSERT INTO patient_institutions (
					patient_id, institution_id, association_source
				) VALUES ($1, $2, 'verified_legacy_association')`, patientID, institutionID)
			expectPostgresCode(t, err, "23505")
		})
	})

	t.Run("only one active primary institution is allowed", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			firstInstitution := insertInstitution(ctx, t, tx, "Primary Hospital")
			secondInstitution := insertInstitution(ctx, t, tx, "Secondary Hospital")
			patientID, err := insertPatient(ctx, tx, "11111111-1111-4111-8111-111111111114", "primary-patient")
			if err != nil {
				t.Fatalf("insert patient: %v", err)
			}
			insertPatientInstitution(ctx, t, tx, patientID, firstInstitution, true)
			_, err = tx.Exec(ctx, `
				INSERT INTO patient_institutions (
					patient_id, institution_id, primary_association, association_source
				) VALUES ($1, $2, TRUE, 'verified_legacy_association')`, patientID, secondInstitution)
			expectPostgresCode(t, err, "23505")
		})
	})

	t.Run("one active identifier per patient and type", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertIdentifierFixture(ctx, t, tx, "patient-type")
			insertPatientIdentifier(ctx, t, tx, fixture.patient1, fixture.identifierType, "A100", "source-a", "active-1", true)
			_, err := tx.Exec(ctx, `
				INSERT INTO patient_identifiers (
					patient_id, identifier_type_id, original_value, canonical_value,
					source_marker, active, lifecycle, source_system, source_record_id
				) VALUES ($1, $2, 'B200', 'B200', 'source-b', TRUE, 'active', 'test', 'active-2')`,
				fixture.patient1, fixture.identifierType)
			expectPostgresCode(t, err, "23505")
		})
	})

	t.Run("active type value source identity cannot span institution-associated patients", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertIdentifierFixture(ctx, t, tx, "global-identity")
			otherInstitution := insertInstitution(ctx, t, tx, "Other Identity Hospital")
			if _, err := tx.Exec(ctx, `
				UPDATE patient_institutions
				SET active = FALSE, primary_association = FALSE, effective_to = now()
				WHERE patient_id = $1 AND institution_id = $2`,
				fixture.patient2, fixture.institution); err != nil {
				t.Fatalf("replace second patient institution: %v", err)
			}
			insertPatientInstitution(ctx, t, tx, fixture.patient2, otherInstitution, true)
			insertPatientIdentifier(ctx, t, tx, fixture.patient1, fixture.identifierType, "A100", "pas", "global-1", true)
			_, err := tx.Exec(ctx, `
				INSERT INTO patient_identifiers (
					patient_id, identifier_type_id, original_value, canonical_value,
					source_marker, active, lifecycle, source_system, source_record_id
				) VALUES ($1, $2, 'A100', 'A100', 'pas', TRUE, 'active', 'test', 'global-2')`,
				fixture.patient2, fixture.identifierType)
			expectPostgresCode(t, err, "23505")
		})
	})

	t.Run("patient type value history cannot repeat", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertIdentifierFixture(ctx, t, tx, "history-identity")
			insertPatientIdentifier(ctx, t, tx, fixture.patient1, fixture.identifierType, "A100", "source-a", "history-1", true)
			_, err := tx.Exec(ctx, `
				INSERT INTO patient_identifiers (
					patient_id, identifier_type_id, original_value, canonical_value,
					source_marker, active, lifecycle, source_system, source_record_id
				) VALUES ($1, $2, 'A100', 'A100', 'source-b', FALSE, 'inactive', 'test', 'history-2')`,
				fixture.patient1, fixture.identifierType)
			expectPostgresCode(t, err, "23505")
		})
	})

	t.Run("identifier status must belong to the same type", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertIdentifierFixture(ctx, t, tx, "status-scope")
			secondType := insertIdentifierType(ctx, t, tx, fixture.institution, "secondary-id", 1, "type-secondary")
			var statusID int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO patient_identifier_statuses (
					identifier_type_id, stable_code, description, source_system, source_record_id
				) VALUES ($1, 'verified', 'Verified', 'test', 'status-primary')
				RETURNING id`, fixture.identifierType).Scan(&statusID); err != nil {
				t.Fatalf("insert identifier status: %v", err)
			}
			_, err := tx.Exec(ctx, `
				INSERT INTO patient_identifiers (
					patient_id, identifier_type_id, identifier_status_id,
					original_value, canonical_value, source_marker,
					active, lifecycle, source_system, source_record_id
				) VALUES ($1, $2, $3, 'A100', 'A100', 'source-a', TRUE, 'active', 'test', 'status-mismatch')`,
				fixture.patient1, secondType, statusID)
			expectPostgresCode(t, err, "23503")
		})
	})

	t.Run("history never occupies current identity", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertIdentifierFixture(ctx, t, tx, "separate-history")
			if _, err := tx.Exec(ctx, `
				INSERT INTO patient_identifier_history (
					patient_id, identifier_type_id, original_value, canonical_value,
					lifecycle, source_system, source_table, source_record_id
				) VALUES ($1, $2, 'A100', 'A100', 'deleted', 'test', 'archive_identifier', 'archive-1')`,
				fixture.patient1, fixture.identifierType); err != nil {
				t.Fatalf("insert identifier history: %v", err)
			}
			insertPatientIdentifier(ctx, t, tx, fixture.patient1, fixture.identifierType, "A100", "source-a", "current-1", true)
		})
	})
}

type identifierFixture struct {
	institution    int64
	patient1       int64
	patient2       int64
	identifierType int64
}

func patientSearchTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("VISIONOPUS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VISIONOPUS_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func withPatientSearchTx(t *testing.T, pool *pgxpool.Pool, test func(context.Context, pgx.Tx)) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin test transaction: %v", err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("roll back test transaction: %v", err)
		}
	})
	test(ctx, tx)
}

func insertIdentifierFixture(ctx context.Context, t *testing.T, tx pgx.Tx, key string) identifierFixture {
	t.Helper()
	fixture := identifierFixture{}
	fixture.institution = insertInstitution(ctx, t, tx, "Synthetic Identifier Hospital")
	var err error
	fixture.patient1, err = insertPatient(ctx, tx, "11111111-1111-4111-8111-111111111121", key+"-patient-1")
	if err != nil {
		t.Fatalf("insert first patient: %v", err)
	}
	fixture.patient2, err = insertPatient(ctx, tx, "11111111-1111-4111-8111-111111111122", key+"-patient-2")
	if err != nil {
		t.Fatalf("insert second patient: %v", err)
	}
	insertPatientInstitution(ctx, t, tx, fixture.patient1, fixture.institution, true)
	insertPatientInstitution(ctx, t, tx, fixture.patient2, fixture.institution, true)
	fixture.identifierType = insertIdentifierType(ctx, t, tx, fixture.institution, "hospital-number", 0, key+"-type")
	return fixture
}

func insertInstitution(ctx context.Context, t *testing.T, tx pgx.Tx, name string) int64 {
	t.Helper()
	var id int64
	if err := tx.QueryRow(ctx, "INSERT INTO institutions (name) VALUES ($1) RETURNING id", name).Scan(&id); err != nil {
		t.Fatalf("insert institution: %v", err)
	}
	return id
}

func insertPatient(ctx context.Context, tx pgx.Tx, publicID, sourceRecordID string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO patients (
			public_id, given_name, given_name_normalized,
			family_name, family_name_normalized, date_of_birth,
			gender, source_system, source_record_id
		) VALUES ($1, 'Synthetic', 'synthetic', 'Patient', 'patient', DATE '1980-01-01', 'unknown', 'test', $2)
		RETURNING id`, publicID, sourceRecordID).Scan(&id)
	return id, err
}

func insertPatientInstitution(
	ctx context.Context,
	t *testing.T,
	tx pgx.Tx,
	patientID int64,
	institutionID int64,
	primary bool,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_institutions (
			patient_id, institution_id, primary_association, association_source
		) VALUES ($1, $2, $3, 'verified_legacy_association')`, patientID, institutionID, primary); err != nil {
		t.Fatalf("insert patient institution: %v", err)
	}
}

func insertIdentifierType(
	ctx context.Context,
	t *testing.T,
	tx pgx.Tx,
	institutionID int64,
	stableCode string,
	displayOrder int,
	sourceRecordID string,
) int64 {
	t.Helper()
	var id int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO patient_identifier_types (
			stable_code, institution_id, display_label, normalization_kind,
			validation_pattern, maximum_canonical_length, validation_state,
			display_order, source_system, source_record_id
		) VALUES ($1, $2, 'Hospital number', 'exact_text_v1', '^[A-Z0-9]+$', 32, 'validated', $3, 'test', $4)
		RETURNING id`, stableCode, institutionID, displayOrder, sourceRecordID).Scan(&id); err != nil {
		t.Fatalf("insert identifier type: %v", err)
	}
	return id
}

func insertPatientIdentifier(
	ctx context.Context,
	t *testing.T,
	tx pgx.Tx,
	patientID int64,
	identifierTypeID int64,
	value string,
	sourceMarker string,
	sourceRecordID string,
	active bool,
) {
	t.Helper()
	lifecycle := "inactive"
	if active {
		lifecycle = "active"
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_identifiers (
			patient_id, identifier_type_id, original_value, canonical_value,
			source_marker, active, lifecycle, source_system, source_record_id
		) VALUES ($1, $2, $3, $3, $4, $5, $6, 'test', $7)`,
		patientID, identifierTypeID, value, sourceMarker, active, lifecycle, sourceRecordID); err != nil {
		t.Fatalf("insert patient identifier: %v", err)
	}
}

func expectPostgresCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("database operation succeeded, want PostgreSQL error %s", code)
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		t.Fatalf("database error = %T %v, want PostgreSQL error %s", err, err, code)
	}
	if postgresError.Code != code {
		t.Fatalf("PostgreSQL error code = %s, want %s: %v", postgresError.Code, code, err)
	}
}
