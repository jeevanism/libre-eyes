//go:build integration

package patientsearch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryIntegration(t *testing.T) {
	pool := repositoryTestPool(t)

	t.Run("loads only active validated type in institution and site scope", func(t *testing.T) {
		withRepositoryFixture(t, pool, func(ctx context.Context, _ pgx.Tx, repository *Repository, fixture repositoryFixture) {
			config, err := repository.LoadIdentifierType(ctx, fixture.institution1, fixture.site1, fixture.identifierType)
			if err != nil {
				t.Fatalf("LoadIdentifierType() error = %v", err)
			}
			if config.ID != fixture.identifierType || config.NormalizationKind != ExactTextV1 ||
				config.ValidationPattern != `^[A-Z0-9]+$` {
				t.Fatalf("LoadIdentifierType() = %#v", config)
			}
			if _, err := CompileIdentifierRule(IdentifierRule{
				Kind:                   config.NormalizationKind,
				ValidationPattern:      config.ValidationPattern,
				MaximumCanonicalLength: config.MaximumCanonicalLength,
				ZeroPadWidth:           config.ZeroPadWidth,
			}); err != nil {
				t.Fatalf("CompileIdentifierRule(loaded config) error = %v", err)
			}

			_, err = repository.LoadIdentifierType(ctx, fixture.institution2, fixture.site2, fixture.identifierType)
			if !errors.Is(err, ErrIdentifierTypeUnavailable) {
				t.Fatalf("cross-institution LoadIdentifierType() error = %v", err)
			}
			_, err = repository.LoadIdentifierType(ctx, fixture.institution1, fixture.site2, fixture.siteIdentifierType)
			if !errors.Is(err, ErrIdentifierTypeUnavailable) {
				t.Fatalf("cross-site LoadIdentifierType() error = %v", err)
			}
			_, err = repository.LoadIdentifierType(ctx, fixture.institution1, fixture.site1, fixture.unavailableType)
			if !errors.Is(err, ErrIdentifierTypeUnavailable) {
				t.Fatalf("quarantined LoadIdentifierType() error = %v", err)
			}
		})
	})

	t.Run("identifier search exposes only current institution and current rows", func(t *testing.T) {
		withRepositoryFixture(t, pool, func(ctx context.Context, tx pgx.Tx, repository *Repository, fixture repositoryFixture) {
			patient := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "30000000-0000-4000-8000-000000000001", given: pointer("Alex"), givenNormalized: pointer("alex"),
				family: pointer("Smith"), familyNormalized: pointer("smith"), dateOfBirth: date(1980, 1, 1), gender: GenderMale,
				sourceID: "identifier-current",
			})
			associateRepositoryPatient(t, ctx, tx, patient, fixture.institution1, true)
			insertRepositoryIdentifier(t, ctx, tx, patient, fixture.identifierType, "<A100>", "A100", "current", true)

			otherInstitutionPatient := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "30000000-0000-4000-8000-000000000002", given: pointer("Other"), givenNormalized: pointer("other"),
				family: pointer("Patient"), familyNormalized: pointer("patient"), dateOfBirth: date(1980, 1, 1), gender: GenderMale,
				sourceID: "identifier-other-institution",
			})
			associateRepositoryPatient(t, ctx, tx, otherInstitutionPatient, fixture.institution2, true)
			insertRepositoryIdentifier(t, ctx, tx, otherInstitutionPatient, fixture.identifierType, "A100", "A100", "other", true)

			inactiveIdentifierPatient := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "30000000-0000-4000-8000-000000000003", given: pointer("Inactive"), givenNormalized: pointer("inactive"),
				family: pointer("Identifier"), familyNormalized: pointer("identifier"), dateOfBirth: date(1980, 1, 1), gender: GenderMale,
				sourceID: "identifier-inactive",
			})
			associateRepositoryPatient(t, ctx, tx, inactiveIdentifierPatient, fixture.institution1, true)
			insertRepositoryIdentifier(t, ctx, tx, inactiveIdentifierPatient, fixture.identifierType, "A100", "A100", "inactive", false)

			inactiveAssociationPatient := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "30000000-0000-4000-8000-000000000004", given: pointer("Inactive"), givenNormalized: pointer("inactive"),
				family: pointer("Association"), familyNormalized: pointer("association"), dateOfBirth: date(1980, 1, 1), gender: GenderMale,
				sourceID: "identifier-inactive-association",
			})
			associateRepositoryPatient(t, ctx, tx, inactiveAssociationPatient, fixture.institution1, false)
			insertRepositoryIdentifier(t, ctx, tx, inactiveAssociationPatient, fixture.identifierType, "A100", "A100", "inactive-association", true)

			inactivePatient := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "30000000-0000-4000-8000-000000000005", given: pointer("Deleted"), givenNormalized: pointer("deleted"),
				family: pointer("Patient"), familyNormalized: pointer("patient"), dateOfBirth: date(1980, 1, 1), gender: GenderMale,
				sourceID: "identifier-inactive-patient",
			})
			associateRepositoryPatient(t, ctx, tx, inactivePatient, fixture.institution1, true)
			insertRepositoryIdentifier(t, ctx, tx, inactivePatient, fixture.identifierType, "A100", "A100", "inactive-patient", true)
			deactivateRepositoryPatient(t, ctx, tx, inactivePatient)

			historyOnlyPatient := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "30000000-0000-4000-8000-000000000006", given: pointer("History"), givenNormalized: pointer("history"),
				family: pointer("Only"), familyNormalized: pointer("only"), dateOfBirth: date(1980, 1, 1), gender: GenderMale,
				sourceID: "identifier-history-only",
			})
			associateRepositoryPatient(t, ctx, tx, historyOnlyPatient, fixture.institution1, true)
			insertRepositoryIdentifierHistory(t, ctx, tx, historyOnlyPatient, fixture.identifierType, "A100")

			page, err := repository.SearchByIdentifier(ctx, fixture.institution1, fixture.site1, fixture.identifierType, "A100", 25, nil)
			if err != nil {
				t.Fatalf("SearchByIdentifier() error = %v", err)
			}
			if len(page.Items) != 1 || page.Items[0].PublicID != "30000000-0000-4000-8000-000000000001" {
				t.Fatalf("SearchByIdentifier() items = %#v", page.Items)
			}
			identifier := page.Items[0].PrimaryIdentifier
			if identifier == nil || identifier.TypeID != fixture.identifierType ||
				identifier.Label != "Hospital number" || identifier.OriginalValue != "<A100>" {
				t.Fatalf("primary identifier = %#v", identifier)
			}
		})
	})

	t.Run("demographic search is exact scoped and given name narrows", func(t *testing.T) {
		withRepositoryFixture(t, pool, func(ctx context.Context, tx pgx.Tx, repository *Repository, fixture repositoryFixture) {
			insertDemographicPatient(t, ctx, tx, fixture.institution1, "30000000-0000-4000-8000-000000000010", "Anna", "anna", "Smith", "smith", GenderFemale, "demographic-anna")
			insertDemographicPatient(t, ctx, tx, fixture.institution1, "30000000-0000-4000-8000-000000000011", "Beth", "beth", "Smith", "smith", GenderFemale, "demographic-beth")
			insertDemographicPatient(t, ctx, tx, fixture.institution2, "30000000-0000-4000-8000-000000000012", "Anna", "anna", "Smith", "smith", GenderFemale, "demographic-other")
			insertDemographicPatient(t, ctx, tx, fixture.institution1, "30000000-0000-4000-8000-000000000013", "Anna", "anna", "Smith", "smith", GenderMale, "demographic-other-gender")
			inactivePatient := insertDemographicPatient(t, ctx, tx, fixture.institution1, "30000000-0000-4000-8000-000000000014", "Anna", "anna", "Smith", "smith", GenderFemale, "demographic-inactive-patient")
			deactivateRepositoryPatient(t, ctx, tx, inactivePatient)
			inactiveAssociationPatient := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "30000000-0000-4000-8000-000000000015", given: pointer("Anna"), givenNormalized: pointer("anna"),
				family: pointer("Smith"), familyNormalized: pointer("smith"), dateOfBirth: date(1980, 1, 1), gender: GenderFemale,
				sourceID: "demographic-inactive-association",
			})
			associateRepositoryPatient(t, ctx, tx, inactiveAssociationPatient, fixture.institution1, false)

			criteria := DemographicCriteria{
				FamilyNameNormalized: "smith", DateOfBirth: date(1980, 1, 1), Gender: GenderFemale,
			}
			page, err := repository.SearchByDemographics(ctx, fixture.institution1, fixture.site1, criteria, 25, nil)
			if err != nil {
				t.Fatalf("SearchByDemographics() error = %v", err)
			}
			if len(page.Items) != 2 || valueOrEmpty(page.Items[0].GivenName) != "Anna" || valueOrEmpty(page.Items[1].GivenName) != "Beth" {
				t.Fatalf("SearchByDemographics() ordered items = %#v", page.Items)
			}

			criteria.GivenNameNormalized = pointer("beth")
			page, err = repository.SearchByDemographics(ctx, fixture.institution1, fixture.site1, criteria, 25, nil)
			if err != nil {
				t.Fatalf("narrowed SearchByDemographics() error = %v", err)
			}
			if len(page.Items) != 1 || valueOrEmpty(page.Items[0].GivenName) != "Beth" {
				t.Fatalf("narrowed SearchByDemographics() items = %#v", page.Items)
			}
		})
	})

	t.Run("keyset pagination is stable and nullable names sort last", func(t *testing.T) {
		withRepositoryFixture(t, pool, func(ctx context.Context, tx pgx.Tx, repository *Repository, fixture repositoryFixture) {
			for index := 1; index <= 3; index++ {
				insertDemographicPatient(
					t, ctx, tx, fixture.institution1,
					fmt.Sprintf("40000000-0000-4000-8000-%012d", index),
					"Same", "same", "Page", "page", GenderUnknown,
					fmt.Sprintf("page-%d", index),
				)
			}
			criteria := DemographicCriteria{
				FamilyNameNormalized: "page", DateOfBirth: date(1980, 1, 1), Gender: GenderUnknown,
			}
			first, err := repository.SearchByDemographics(ctx, fixture.institution1, fixture.site1, criteria, 2, nil)
			if err != nil {
				t.Fatalf("first SearchByDemographics() error = %v", err)
			}
			if !first.HasMore || first.NextBoundary == nil || len(first.Items) != 2 {
				t.Fatalf("first page = %#v", first)
			}
			second, err := repository.SearchByDemographics(ctx, fixture.institution1, fixture.site1, criteria, 2, first.NextBoundary)
			if err != nil {
				t.Fatalf("second SearchByDemographics() error = %v", err)
			}
			if second.HasMore || second.NextBoundary != nil || len(second.Items) != 1 ||
				second.Items[0].PublicID != "40000000-0000-4000-8000-000000000003" {
				t.Fatalf("second page = %#v", second)
			}

			named := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "50000000-0000-4000-8000-000000000001", given: pointer("Named"), givenNormalized: pointer("named"),
				family: pointer("Patient"), familyNormalized: pointer("patient"), dateOfBirth: date(1980, 1, 1), gender: GenderUnknown,
				sourceID: "nullable-named",
			})
			firstNameless := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "50000000-0000-4000-8000-000000000002", dateOfBirth: date(1980, 1, 1), gender: GenderUnknown,
				sourceID: "nullable-nameless-first",
			})
			secondNameless := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "50000000-0000-4000-8000-000000000003", dateOfBirth: date(1980, 1, 1), gender: GenderUnknown,
				sourceID: "nullable-nameless-second",
			})
			associateRepositoryPatient(t, ctx, tx, named, fixture.institution1, true)
			associateRepositoryPatient(t, ctx, tx, firstNameless, fixture.institution1, true)
			associateRepositoryPatient(t, ctx, tx, secondNameless, fixture.institution1, true)
			insertRepositoryIdentifier(t, ctx, tx, named, fixture.identifierType, "Z100", "Z100", "named", true)
			insertRepositoryIdentifier(t, ctx, tx, firstNameless, fixture.identifierType, "Z100", "Z100", "nameless-first", true)
			insertRepositoryIdentifier(t, ctx, tx, secondNameless, fixture.identifierType, "Z100", "Z100", "nameless-second", true)
			identifierPage, err := repository.SearchByIdentifier(ctx, fixture.institution1, fixture.site1, fixture.identifierType, "Z100", 2, nil)
			if err != nil {
				t.Fatalf("nullable SearchByIdentifier() error = %v", err)
			}
			if !identifierPage.HasMore || identifierPage.NextBoundary == nil || len(identifierPage.Items) != 2 ||
				identifierPage.Items[0].PublicID != "50000000-0000-4000-8000-000000000001" ||
				identifierPage.Items[1].PublicID != "50000000-0000-4000-8000-000000000002" {
				t.Fatalf("nullable first page = %#v", identifierPage)
			}
			identifierPage, err = repository.SearchByIdentifier(
				ctx, fixture.institution1, fixture.site1, fixture.identifierType, "Z100", 2, identifierPage.NextBoundary,
			)
			if err != nil {
				t.Fatalf("nullable boundary SearchByIdentifier() error = %v", err)
			}
			if identifierPage.HasMore || identifierPage.NextBoundary != nil || len(identifierPage.Items) != 1 ||
				identifierPage.Items[0].PublicID != "50000000-0000-4000-8000-000000000003" {
				t.Fatalf("nullable second page = %#v", identifierPage)
			}
		})
	})

	t.Run("duplicate checks are bounded exact and read only", func(t *testing.T) {
		withRepositoryFixture(t, pool, func(ctx context.Context, tx pgx.Tx, repository *Repository, fixture repositoryFixture) {
			patient := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
				publicID: "60000000-0000-4000-8000-000000000001", given: pointer("Exact"), givenNormalized: pointer("exact"),
				family: pointer("Identifier"), familyNormalized: pointer("identifier"), dateOfBirth: date(1980, 1, 1), gender: GenderUnknown,
				sourceID: "duplicate-identifier",
			})
			associateRepositoryPatient(t, ctx, tx, patient, fixture.institution1, true)
			insertRepositoryIdentifier(t, ctx, tx, patient, fixture.identifierType, "D100", "D100", "duplicate", true)
			insertBulkIdentifierDuplicatePatients(t, ctx, tx, fixture.institution1, fixture.identifierType)

			before := loadRepositoryState(t, ctx, tx)
			identifierCandidates, err := repository.FindIdentifierDuplicates(ctx, fixture.institution1, fixture.site1, fixture.identifierType, "D100")
			if err != nil {
				t.Fatalf("FindIdentifierDuplicates() error = %v", err)
			}
			if !identifierCandidates.HardConflict || identifierCandidates.Truncated || len(identifierCandidates.Candidates) != 1 ||
				identifierCandidates.Candidates[0].Reason != DuplicateReasonIdentifier {
				t.Fatalf("identifier candidates = %#v", identifierCandidates)
			}
			truncatedIdentifiers, err := repository.FindIdentifierDuplicates(
				ctx, fixture.institution1, fixture.site1, fixture.identifierType, "I100",
			)
			if err != nil {
				t.Fatalf("truncated FindIdentifierDuplicates() error = %v", err)
			}
			if !truncatedIdentifiers.HardConflict || !truncatedIdentifiers.Truncated || len(truncatedIdentifiers.Candidates) != 100 {
				t.Fatalf("truncated identifier candidates = hard=%v truncated=%v count=%d",
					truncatedIdentifiers.HardConflict, truncatedIdentifiers.Truncated, len(truncatedIdentifiers.Candidates))
			}

			insertBulkDuplicatePatients(t, ctx, tx, fixture.institution1)
			demographicCandidates, err := repository.FindDemographicDuplicates(ctx, fixture.institution1, fixture.site1, DuplicateDemographicCriteria{
				FamilyNameNormalized: "candidate", GivenNameNormalized: "same", DateOfBirth: date(1990, 2, 3),
			})
			if err != nil {
				t.Fatalf("FindDemographicDuplicates() error = %v", err)
			}
			if demographicCandidates.HardConflict || !demographicCandidates.Truncated || len(demographicCandidates.Candidates) != 100 {
				t.Fatalf("demographic candidates = hard=%v truncated=%v count=%d",
					demographicCandidates.HardConflict, demographicCandidates.Truncated, len(demographicCandidates.Candidates))
			}
			for _, candidate := range demographicCandidates.Candidates {
				if candidate.Reason != DuplicateReasonDemographics {
					t.Fatalf("candidate reason = %q", candidate.Reason)
				}
			}
			after := loadRepositoryState(t, ctx, tx)
			if after.patients != before.patients+101 || after.identifiers != before.identifiers ||
				after.associations != before.associations+101 || after.merges != before.merges {
				t.Fatalf("duplicate checks changed state: before=%#v after=%#v", before, after)
			}
		})
	})

	t.Run("canceled context returns no partial page", func(t *testing.T) {
		withRepositoryFixture(t, pool, func(_ context.Context, _ pgx.Tx, repository *Repository, fixture repositoryFixture) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			operations := []struct {
				name string
				run  func() error
			}{
				{name: "load identifier type", run: func() error {
					_, err := repository.LoadIdentifierType(ctx, fixture.institution1, fixture.site1, fixture.identifierType)
					return err
				}},
				{name: "identifier search", run: func() error {
					_, err := repository.SearchByIdentifier(ctx, fixture.institution1, fixture.site1, fixture.identifierType, "A100", 25, nil)
					return err
				}},
				{name: "demographic search", run: func() error {
					_, err := repository.SearchByDemographics(ctx, fixture.institution1, fixture.site1, DemographicCriteria{
						FamilyNameNormalized: "patient", DateOfBirth: date(1980, 1, 1), Gender: GenderUnknown,
					}, 25, nil)
					return err
				}},
				{name: "demographic duplicates", run: func() error {
					_, err := repository.FindDemographicDuplicates(ctx, fixture.institution1, fixture.site1, DuplicateDemographicCriteria{
						FamilyNameNormalized: "patient", GivenNameNormalized: "same", DateOfBirth: date(1980, 1, 1),
					})
					return err
				}},
			}
			for _, operation := range operations {
				if err := operation.run(); !errors.Is(err, context.Canceled) {
					t.Fatalf("canceled %s error = %v", operation.name, err)
				}
			}
		})
	})
}

type repositoryFixture struct {
	institution1, institution2         int64
	site1, site2                       int64
	identifierType, siteIdentifierType int64
	unavailableType                    int64
}

type repositoryPatient struct {
	publicID, sourceID       string
	given, givenNormalized   *string
	family, familyNormalized *string
	dateOfBirth              time.Time
	gender                   Gender
}

func repositoryTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("LIBREEYES_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LIBREEYES_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to repository test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func withRepositoryFixture(
	t *testing.T,
	pool *pgxpool.Pool,
	test func(context.Context, pgx.Tx, *Repository, repositoryFixture),
) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin repository test transaction: %v", err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("roll back repository test: %v", err)
		}
	})
	repository, err := NewRepository(tx)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	fixture := seedRepositoryFixture(t, ctx, tx)
	test(ctx, tx, repository, fixture)
}

func seedRepositoryFixture(t *testing.T, ctx context.Context, tx pgx.Tx) repositoryFixture {
	t.Helper()
	fixture := repositoryFixture{}
	fixture.institution1 = insertRepositoryInstitution(t, ctx, tx, "Repository Hospital One")
	fixture.institution2 = insertRepositoryInstitution(t, ctx, tx, "Repository Hospital Two")
	fixture.site1 = insertRepositorySite(t, ctx, tx, fixture.institution1, "Repository Site One")
	fixture.site2 = insertRepositorySite(t, ctx, tx, fixture.institution2, "Repository Site Two")
	fixture.identifierType = insertRepositoryIdentifierType(t, ctx, tx, fixture.institution1, nil, "hospital-number", 0, true)
	fixture.siteIdentifierType = insertRepositoryIdentifierType(t, ctx, tx, fixture.institution1, &fixture.site1, "site-number", 0, true)
	fixture.unavailableType = insertRepositoryIdentifierType(t, ctx, tx, fixture.institution1, nil, "quarantined-number", 1, false)
	return fixture
}

func insertRepositoryInstitution(t *testing.T, ctx context.Context, tx pgx.Tx, name string) int64 {
	t.Helper()
	var id int64
	if err := tx.QueryRow(ctx, "INSERT INTO institutions (name) VALUES ($1) RETURNING id", name).Scan(&id); err != nil {
		t.Fatalf("insert repository institution: %v", err)
	}
	return id
}

func insertRepositorySite(t *testing.T, ctx context.Context, tx pgx.Tx, institutionID int64, name string) int64 {
	t.Helper()
	var id int64
	if err := tx.QueryRow(ctx, "INSERT INTO sites (institution_id, name) VALUES ($1, $2) RETURNING id", institutionID, name).Scan(&id); err != nil {
		t.Fatalf("insert repository site: %v", err)
	}
	return id
}

func insertRepositoryIdentifierType(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	institutionID int64,
	siteID *int64,
	stableCode string,
	displayOrder int,
	available bool,
) int64 {
	t.Helper()
	validationState := "validated"
	if !available {
		validationState = "quarantined"
	}
	var id int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO patient_identifier_types (
			stable_code, institution_id, site_id, display_label, normalization_kind,
			validation_pattern, maximum_canonical_length, validation_state,
			searchable, active, display_order, display_prefix, display_suffix,
			source_system, source_record_id
		) VALUES ($1, $2, $3, 'Hospital number', 'exact_text_v1', '^[A-Z0-9]+$', 32, $4, $5, $5, $6, 'ID: ', '', 'test', $7)
		RETURNING id`, stableCode, institutionID, siteID, validationState, available, displayOrder, "type-"+stableCode).Scan(&id); err != nil {
		t.Fatalf("insert repository identifier type: %v", err)
	}
	return id
}

func insertRepositoryPatient(t *testing.T, ctx context.Context, tx pgx.Tx, patient repositoryPatient) int64 {
	t.Helper()
	var id int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO patients (
			public_id, given_name, given_name_normalized, family_name,
			family_name_normalized, date_of_birth, gender, source_system, source_record_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'test', $8)
		RETURNING id`, patient.publicID, patient.given, patient.givenNormalized,
		patient.family, patient.familyNormalized, patient.dateOfBirth, patient.gender, patient.sourceID).Scan(&id); err != nil {
		t.Fatalf("insert repository patient: %v", err)
	}
	return id
}

func associateRepositoryPatient(t *testing.T, ctx context.Context, tx pgx.Tx, patientID, institutionID int64, active bool) {
	t.Helper()
	var effectiveTo *time.Time
	if !active {
		now := time.Now().UTC()
		effectiveTo = &now
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_institutions (
			patient_id, institution_id, active, primary_association,
			association_source, effective_to
		) VALUES ($1, $2, $3, $3, 'verified_legacy_association', $4)`, patientID, institutionID, active, effectiveTo); err != nil {
		t.Fatalf("associate repository patient: %v", err)
	}
}

func insertRepositoryIdentifier(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	patientID, identifierTypeID int64,
	originalValue, canonicalValue, sourceMarker string,
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'test', $8)`,
		patientID, identifierTypeID, originalValue, canonicalValue, sourceMarker,
		active, lifecycle, fmt.Sprintf("identifier-%d-%s", patientID, sourceMarker)); err != nil {
		t.Fatalf("insert repository identifier: %v", err)
	}
}

func deactivateRepositoryPatient(t *testing.T, ctx context.Context, tx pgx.Tx, patientID int64) {
	t.Helper()
	if _, err := tx.Exec(ctx, `
		UPDATE patients
		SET active = FALSE, deleted_at = now(), updated_at = now()
		WHERE id = $1`, patientID); err != nil {
		t.Fatalf("deactivate repository patient: %v", err)
	}
}

func insertRepositoryIdentifierHistory(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	patientID, identifierTypeID int64,
	canonicalValue string,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_identifier_history (
			patient_id, identifier_type_id, original_value, canonical_value,
			lifecycle, source_system, source_table, source_record_id
		) VALUES ($1, $2, $3, $3, 'deleted', 'test', 'archive_identifier', $4)`,
		patientID, identifierTypeID, canonicalValue, fmt.Sprintf("history-%d", patientID)); err != nil {
		t.Fatalf("insert repository identifier history: %v", err)
	}
}

func insertDemographicPatient(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	institutionID int64,
	publicID, given, givenNormalized, family, familyNormalized string,
	gender Gender,
	sourceID string,
) int64 {
	t.Helper()
	patientID := insertRepositoryPatient(t, ctx, tx, repositoryPatient{
		publicID: publicID, given: &given, givenNormalized: &givenNormalized,
		family: &family, familyNormalized: &familyNormalized,
		dateOfBirth: date(1980, 1, 1), gender: gender, sourceID: sourceID,
	})
	associateRepositoryPatient(t, ctx, tx, patientID, institutionID, true)
	return patientID
}

func insertBulkDuplicatePatients(t *testing.T, ctx context.Context, tx pgx.Tx, institutionID int64) {
	t.Helper()
	if _, err := tx.Exec(ctx, `
		INSERT INTO patients (
			public_id, given_name, given_name_normalized, family_name,
			family_name_normalized, date_of_birth, gender, source_system, source_record_id
		)
		SELECT
			('70000000-0000-4000-8000-' || lpad(series::text, 12, '0'))::uuid,
			'Same', 'same', 'Candidate', 'candidate', DATE '1990-02-03',
			'unknown', 'test', 'bulk-candidate-' || series
		FROM generate_series(1, 101) series`); err != nil {
		t.Fatalf("insert bulk duplicate patients: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_institutions (
			patient_id, institution_id, primary_association, association_source
		)
		SELECT id, $1, TRUE, 'verified_legacy_association'
		FROM patients
		WHERE source_system = 'test' AND source_record_id LIKE 'bulk-candidate-%'`, institutionID); err != nil {
		t.Fatalf("associate bulk duplicate patients: %v", err)
	}
}

func insertBulkIdentifierDuplicatePatients(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	institutionID, identifierTypeID int64,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, `
		INSERT INTO patients (
			public_id, given_name, given_name_normalized, family_name,
			family_name_normalized, date_of_birth, gender, source_system, source_record_id
		)
		SELECT
			('80000000-0000-4000-8000-' || lpad(series::text, 12, '0'))::uuid,
			'Identifier', 'identifier', 'Candidate', 'candidate', DATE '1990-02-03',
			'unknown', 'test', 'bulk-identifier-' || series
		FROM generate_series(1, 101) series`); err != nil {
		t.Fatalf("insert bulk identifier duplicate patients: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_institutions (
			patient_id, institution_id, primary_association, association_source
		)
		SELECT id, $1, TRUE, 'verified_legacy_association'
		FROM patients
		WHERE source_system = 'test' AND source_record_id LIKE 'bulk-identifier-%'`, institutionID); err != nil {
		t.Fatalf("associate bulk identifier duplicate patients: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO patient_identifiers (
			patient_id, identifier_type_id, original_value, canonical_value,
			source_marker, source_system, source_record_id
		)
		SELECT id, $1, 'I100', 'I100', source_record_id, 'test', 'identifier-' || source_record_id
		FROM patients
		WHERE source_system = 'test' AND source_record_id LIKE 'bulk-identifier-%'`, identifierTypeID); err != nil {
		t.Fatalf("insert bulk identifier duplicates: %v", err)
	}
}

type repositoryState struct {
	patients, identifiers, associations, merges int64
}

func loadRepositoryState(t *testing.T, ctx context.Context, tx pgx.Tx) repositoryState {
	t.Helper()
	var state repositoryState
	if err := tx.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM patients),
			(SELECT count(*) FROM patient_identifiers),
			(SELECT count(*) FROM patient_institutions),
			(SELECT count(*) FROM patient_merge_lineage)`,
	).Scan(&state.patients, &state.identifiers, &state.associations, &state.merges); err != nil {
		t.Fatalf("load repository state: %v", err)
	}
	return state
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func pointer(value string) *string {
	return &value
}
