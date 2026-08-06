//go:build integration

package patientsearch

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
)

type staticAuthorizer struct {
	principal auth.OperationPrincipal
	err       error
	last      auth.OperationAuthorizationRequest
}

func (a *staticAuthorizer) AuthorizeOperation(_ context.Context, request auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error) {
	a.last = request
	return a.principal, a.err
}

func TestServiceIntegrationSearchAuditCursorAndDuplicate(t *testing.T) {
	pool := repositoryTestPool(t)
	fixture := seedServiceFixture(t, pool)
	authorizer := &staticAuthorizer{principal: fixture.principal}
	service, err := NewService(pool, authorizer, ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	metadata := auth.RequestMetadata{CorrelationID: "patient-search-integration", SourceIPClass: "loopback"}

	authorization, err := service.Authorize(context.Background(), "token", "csrf", OperationSearch, metadata)
	if err != nil {
		t.Fatalf("Authorize(search) error = %v", err)
	}
	if authorizer.last.Permission != "patient.search" || authorizer.last.DeniedEventType != "patient_search.denied" {
		t.Fatalf("authorization policy = %#v", authorizer.last)
	}
	first, err := service.Search(context.Background(), authorization, SearchRequest{
		Criteria: SearchCriteria{Kind: CriteriaDemographic, Demographic: &DemographicInput{
			FamilyName: "  Sensitive Family  ", DateOfBirth: "1980-01-01", Gender: GenderFemale,
		}},
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("Search(first) error = %v", err)
	}
	if len(first.Items) != 1 || !first.HasMore || first.NextCursor == nil {
		t.Fatalf("first page = %#v, want one item and protected next cursor", first)
	}
	if strings.Contains(*first.NextCursor, "sensitive") || len(*first.NextCursor) > 512 {
		t.Fatalf("next cursor is not bounded opaque output: %q", *first.NextCursor)
	}

	authorization, err = service.Authorize(context.Background(), "token", "csrf", OperationSearch, metadata)
	if err != nil {
		t.Fatalf("Authorize(second page) error = %v", err)
	}
	second, err := service.Search(context.Background(), authorization, SearchRequest{
		Criteria: SearchCriteria{Kind: CriteriaDemographic, Demographic: &DemographicInput{
			FamilyName: "Sensitive Family", DateOfBirth: "1980-01-01", Gender: GenderFemale,
		}},
		Limit: 1, Cursor: *first.NextCursor,
	})
	if err != nil {
		t.Fatalf("Search(second) error = %v", err)
	}
	if len(second.Items) != 1 || second.HasMore || second.NextCursor != nil || second.Items[0].PatientID == first.Items[0].PatientID {
		t.Fatalf("second page = %#v, want distinct final item", second)
	}

	wrongPrincipal := fixture.principal
	wrongPrincipal.ContextVersion++
	authorizer.principal = wrongPrincipal
	wrongAuthorization, err := service.Authorize(context.Background(), "token", "csrf", OperationSearch, metadata)
	if err != nil {
		t.Fatalf("Authorize(wrong context) error = %v", err)
	}
	_, err = service.Search(context.Background(), wrongAuthorization, SearchRequest{
		Criteria: SearchCriteria{Kind: CriteriaDemographic, Demographic: &DemographicInput{
			FamilyName: "Sensitive Family", DateOfBirth: "1980-01-01", Gender: GenderFemale,
		}},
		Limit: 1, Cursor: *first.NextCursor,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Search(context-bound cursor) error = %v, want ErrInvalidRequest", err)
	}
	authorizer.principal = fixture.principal

	duplicateAuthorization, err := service.Authorize(context.Background(), "token", "csrf", OperationDuplicateCheck, metadata)
	if err != nil {
		t.Fatalf("Authorize(duplicate) error = %v", err)
	}
	duplicate, err := service.FindDuplicates(context.Background(), duplicateAuthorization, DuplicateRequest{
		Criteria: SearchCriteria{Kind: CriteriaIdentifier, Identifier: &IdentifierCriteria{
			IdentifierTypeID: fixture.identifierType, Value: "A100",
		}},
	})
	if err != nil {
		t.Fatalf("FindDuplicates() error = %v", err)
	}
	if !duplicate.HardConflict || duplicate.Truncated || len(duplicate.Candidates) != 1 ||
		duplicate.Candidates[0].Patient.PrimaryIdentifier == nil ||
		duplicate.Candidates[0].Patient.PrimaryIdentifier.Value != "ID: A100" {
		t.Fatalf("duplicate result = %#v", duplicate)
	}

	var executedSearches, executedDuplicates int
	var attributes string
	if err := pool.QueryRow(context.Background(), `
		SELECT
			count(*) FILTER (WHERE event_type = 'patient_search.executed'),
			count(*) FILTER (WHERE event_type = 'patient_duplicate_check.executed'),
			COALESCE(string_agg(attributes::text, ' '), '')
		FROM audit_events
		WHERE event_type IN ('patient_search.executed', 'patient_duplicate_check.executed')`,
	).Scan(&executedSearches, &executedDuplicates, &attributes); err != nil {
		t.Fatalf("query executed audit: %v", err)
	}
	if executedSearches != 2 || executedDuplicates != 1 {
		t.Fatalf("executed audit counts = search:%d duplicate:%d", executedSearches, executedDuplicates)
	}
	for _, forbidden := range []string{"Sensitive", "sensitive", "A100", first.Items[0].PatientID, second.Items[0].PatientID} {
		if strings.Contains(attributes, forbidden) {
			t.Fatalf("audit attributes contain forbidden criteria or patient identity %q: %s", forbidden, attributes)
		}
	}
	for _, required := range []string{"permissionOutcome", "resultCountBucket", "operationOutcome", "latencyMs"} {
		if !strings.Contains(attributes, required) {
			t.Fatalf("audit attributes missing %q: %s", required, attributes)
		}
	}

	before := servicePatientState(t, pool)
	forcePatientAuditFailure(t, pool)
	authorization, err = service.Authorize(context.Background(), "token", "csrf", OperationSearch, metadata)
	if err != nil {
		t.Fatalf("Authorize(audit failure) error = %v", err)
	}
	_, err = service.Search(context.Background(), authorization, SearchRequest{
		Criteria: SearchCriteria{Kind: CriteriaIdentifier, Identifier: &IdentifierCriteria{
			IdentifierTypeID: fixture.identifierType, Value: "A100",
		}},
	})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Search(audit failure) error = %v, want ErrUnavailable", err)
	}
	after := servicePatientState(t, pool)
	if before != after {
		t.Fatalf("search changed patient state: before=%#v after=%#v", before, after)
	}
}

func TestServicePrincipalRateLimitIsCombinedAndAudited(t *testing.T) {
	pool := repositoryTestPool(t)
	fixture := seedServiceFixture(t, pool)
	authorizer := &staticAuthorizer{principal: fixture.principal}
	service, err := NewService(pool, authorizer, ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	metadata := auth.RequestMetadata{CorrelationID: "patient-rate-integration", SourceIPClass: "loopback"}
	for requestNumber := 1; requestNumber <= int(principalBurst); requestNumber++ {
		operation := OperationSearch
		if requestNumber%2 == 0 {
			operation = OperationDuplicateCheck
		}
		if _, err := service.Authorize(context.Background(), "token", "csrf", operation, metadata); err != nil {
			t.Fatalf("Authorize() request %d error = %v", requestNumber, err)
		}
	}
	_, err = service.Authorize(context.Background(), "token", "csrf", OperationDuplicateCheck, metadata)
	var rateLimit *RateLimitError
	if !errors.As(err, &rateLimit) || rateLimit.RetryAfter != time.Second {
		t.Fatalf("exhausted Authorize() error = %v, want 1s RateLimitError", err)
	}
	var denialCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM audit_events
		WHERE event_type = 'patient_duplicate_check.denied'
			AND reason_code = 'rate_limited' AND outcome = 'denied'`,
	).Scan(&denialCount); err != nil {
		t.Fatalf("query rate-limit denial audit: %v", err)
	}
	if denialCount != 1 {
		t.Fatalf("rate-limit denial audit count = %d, want 1", denialCount)
	}
}

type serviceFixture struct {
	principal      auth.OperationPrincipal
	identifierType int64
}

func seedServiceFixture(t *testing.T, pool *pgxpool.Pool) serviceFixture {
	t.Helper()
	resetServiceData(t, pool)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin service fixture: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	institutionID := insertRepositoryInstitution(t, ctx, tx, "Patient Search Service Hospital")
	otherInstitutionID := insertRepositoryInstitution(t, ctx, tx, "Other Patient Search Hospital")
	siteID := insertRepositorySite(t, ctx, tx, institutionID, "Patient Search Site")
	otherSiteID := insertRepositorySite(t, ctx, tx, otherInstitutionID, "Other Search Site")
	_ = otherSiteID
	var firmID, userID, sessionID int64
	if err := tx.QueryRow(ctx, "INSERT INTO firms (institution_id, name) VALUES ($1, 'Ophthalmology') RETURNING id", institutionID).Scan(&firmID); err != nil {
		t.Fatalf("insert service firm: %v", err)
	}
	if err := tx.QueryRow(ctx, "INSERT INTO users (display_name) VALUES ('Synthetic Search Clinician') RETURNING id").Scan(&userID); err != nil {
		t.Fatalf("insert service user: %v", err)
	}
	now := time.Now().UTC()
	if err := tx.QueryRow(ctx, `
		INSERT INTO sessions (
			token_digest, csrf_digest, user_id, institution_id, site_id, firm_id,
			created_at, last_seen_at, idle_expires_at, absolute_expires_at, authorization_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8, $9, 1)
		RETURNING id`, make([]byte, 32), append([]byte{1}, make([]byte, 31)...), userID,
		institutionID, siteID, firmID, now, now.Add(15*time.Minute), now.Add(12*time.Hour),
	).Scan(&sessionID); err != nil {
		t.Fatalf("insert service session: %v", err)
	}
	identifierTypeID := insertRepositoryIdentifierType(t, ctx, tx, institutionID, nil, "service-hospital-number", 0, true)

	first := insertDemographicPatient(t, ctx, tx, institutionID, "90000000-0000-4000-8000-000000000001", "Ada", "ada", "Sensitive Family", "sensitive family", GenderFemale, "service-first")
	second := insertDemographicPatient(t, ctx, tx, institutionID, "90000000-0000-4000-8000-000000000002", "Grace", "grace", "Sensitive Family", "sensitive family", GenderFemale, "service-second")
	other := insertDemographicPatient(t, ctx, tx, otherInstitutionID, "90000000-0000-4000-8000-000000000003", "Other", "other", "Sensitive Family", "sensitive family", GenderFemale, "service-other")
	insertRepositoryIdentifier(t, ctx, tx, first, identifierTypeID, "A100", "A100", "service-first", true)
	_ = second
	_ = other

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit service fixture: %v", err)
	}
	return serviceFixture{
		principal: auth.OperationPrincipal{
			UserID: userID, SessionID: sessionID, InstitutionID: institutionID,
			SiteID: siteID, FirmID: firmID, ContextVersion: 1,
		},
		identifierType: identifierTypeID,
	}
}

func resetServiceData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin service reset: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DROP TRIGGER IF EXISTS patient_search_audit_failure ON audit_events`); err != nil {
		t.Fatalf("drop service audit failure trigger: %v", err)
	}
	if _, err := tx.Exec(ctx, `DROP FUNCTION IF EXISTS fail_patient_search_audit()`); err != nil {
		t.Fatalf("drop service audit failure function: %v", err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE audit_events DISABLE TRIGGER audit_events_no_truncate`); err != nil {
		t.Fatalf("disable service audit guard: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		TRUNCATE audit_events, sessions, user_role_assignments, user_credentials,
			authentication_profiles, user_firm_preferences, user_firm_memberships,
			user_site_memberships, user_institution_memberships, users, firms, sites,
			institutions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate service data: %v", err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE audit_events ENABLE TRIGGER audit_events_no_truncate`); err != nil {
		t.Fatalf("enable service audit guard: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit service reset: %v", err)
	}
}

func forcePatientAuditFailure(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION fail_patient_search_audit() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.event_type IN ('patient_search.executed', 'patient_duplicate_check.executed') THEN
				RAISE EXCEPTION 'synthetic patient search audit failure';
			END IF;
			RETURN NEW;
		END;
		$$`); err != nil {
		t.Fatalf("create audit failure function: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE TRIGGER patient_search_audit_failure
		BEFORE INSERT ON audit_events
		FOR EACH ROW EXECUTE FUNCTION fail_patient_search_audit()`); err != nil {
		t.Fatalf("create audit failure trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS patient_search_audit_failure ON audit_events`)
		_, _ = pool.Exec(context.Background(), `DROP FUNCTION IF EXISTS fail_patient_search_audit()`)
	})
}

func servicePatientState(t *testing.T, pool *pgxpool.Pool) repositoryState {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin state query: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return loadRepositoryState(t, ctx, tx)
}
