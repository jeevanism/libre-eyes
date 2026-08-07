//go:build integration

package episodes

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
)

type integrationAuthorizer struct {
	principal auth.OperationPrincipal
	last      auth.OperationAuthorizationRequest
}

func (a *integrationAuthorizer) AuthorizeOperation(_ context.Context, request auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error) {
	a.last = request
	return a.principal, nil
}

func TestServiceIntegrationLifecycleAuditAndScope(t *testing.T) {
	pool := episodesTestPool(t)
	fixture := seedEpisodesFixture(t, pool)
	authorizer := &integrationAuthorizer{principal: fixture.principal}
	service, err := NewService(pool, authorizer)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.now = func() time.Time { return time.Date(2026, 8, 7, 9, 30, 0, 0, time.UTC) }
	metadata := auth.RequestMetadata{CorrelationID: "episodes-integration", SourceIPClass: "loopback"}

	createAuthorization, err := service.Authorize(context.Background(), "token", "csrf", permissionCreate, metadata)
	if err != nil {
		t.Fatalf("Authorize(create) error = %v", err)
	}
	if authorizer.last.Permission != permissionCreate || authorizer.last.DeniedEventType != "episode.create_denied" {
		t.Fatalf("create authorization policy = %#v", authorizer.last)
	}
	if _, err := service.Create(context.Background(), createAuthorization, fixture.patientPublicID, CreateRequest{Status: StatusClosed}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Create(closed) error = %v, want invalid request", err)
	}
	created, err := service.Create(context.Background(), createAuthorization, fixture.patientPublicID, CreateRequest{Status: StatusOpen})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != StatusOpen || created.Version != 1 || created.StartedAt == nil || !created.StartedAt.Equal(service.now()) || created.EndedAt != nil {
		t.Fatalf("created episode = %#v", created)
	}
	second, err := service.Create(context.Background(), createAuthorization, fixture.patientPublicID, CreateRequest{Status: StatusActive})
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	if second.Status != StatusActive || second.Version != 1 {
		t.Fatalf("created second episode = %#v", second)
	}

	readAuthorization := authorizeEpisode(t, service, permissionRead, metadata)
	if _, err := service.List(context.Background(), readAuthorization, ListRequest{PatientID: fixture.patientPublicID, Limit: maximumPageSize + 1}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("List(oversized limit) error = %v, want invalid request", err)
	}
	listed, err := service.List(context.Background(), readAuthorization, ListRequest{PatientID: fixture.patientPublicID})
	if err != nil || len(listed.Items) != 2 || listed.NextCursor != nil {
		t.Fatalf("List() = %#v, %v", listed, err)
	}
	firstPage, err := service.List(context.Background(), readAuthorization, ListRequest{PatientID: fixture.patientPublicID, Limit: 1})
	if err != nil || len(firstPage.Items) != 1 || firstPage.NextCursor == nil {
		t.Fatalf("List(first page) = %#v, %v", firstPage, err)
	}
	if _, err := service.List(context.Background(), readAuthorization, ListRequest{PatientID: "11111111-1111-4111-8111-111111111112", Limit: 1, Cursor: *firstPage.NextCursor}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("List(cross patient cursor) error = %v, want invalid request", err)
	}
	secondPage, err := service.List(context.Background(), readAuthorization, ListRequest{PatientID: fixture.patientPublicID, Limit: 1, Cursor: *firstPage.NextCursor})
	if err != nil || len(secondPage.Items) != 1 || secondPage.NextCursor != nil || secondPage.Items[0].ID == firstPage.Items[0].ID {
		t.Fatalf("List(second page) = %#v, %v", secondPage, err)
	}
	loaded, err := service.Get(context.Background(), readAuthorization, created.ID)
	if err != nil || loaded.Version != 1 || loaded.PatientID != fixture.patientPublicID {
		t.Fatalf("Get() = %#v, %v", loaded, err)
	}

	updateAuthorization := authorizeEpisode(t, service, permissionUpdate, metadata)
	if _, err := service.Activate(context.Background(), updateAuthorization, LifecycleRequest{EpisodeID: created.ID}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Activate(no version) error = %v, want invalid request", err)
	}
	if _, err := service.Activate(context.Background(), updateAuthorization, LifecycleRequest{EpisodeID: created.ID, ExpectedVersion: 1, Reason: "not allowed"}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Activate(reason) error = %v, want invalid request", err)
	}
	active, err := service.Activate(context.Background(), updateAuthorization, LifecycleRequest{EpisodeID: created.ID, ExpectedVersion: 1})
	if err != nil || active.Status != StatusActive || active.Version != 2 || active.EndedAt != nil {
		t.Fatalf("Activate() = %#v, %v", active, err)
	}
	if _, err := service.Activate(context.Background(), updateAuthorization, LifecycleRequest{EpisodeID: created.ID, ExpectedVersion: 1}); !errors.Is(err, auth.ErrConflict) {
		t.Fatalf("Activate(stale) error = %v, want conflict", err)
	}
	closed, err := service.Close(context.Background(), updateAuthorization, LifecycleRequest{EpisodeID: created.ID, ExpectedVersion: 2})
	if err != nil || closed.Status != StatusClosed || closed.Version != 3 || closed.EndedAt == nil {
		t.Fatalf("Close() = %#v, %v", closed, err)
	}
	reopenAuthorization := authorizeEpisode(t, service, permissionReopen, metadata)
	if _, err := service.Reopen(context.Background(), reopenAuthorization, LifecycleRequest{EpisodeID: created.ID, ExpectedVersion: 3}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Reopen(no reason) error = %v, want invalid request", err)
	}
	reopened, err := service.Reopen(context.Background(), reopenAuthorization, LifecycleRequest{EpisodeID: created.ID, ExpectedVersion: 3, Reason: "corrected closure"})
	if err != nil || reopened.Status != StatusActive || reopened.Version != 4 || reopened.EndedAt != nil {
		t.Fatalf("Reopen() = %#v, %v", reopened, err)
	}

	otherPrincipal := fixture.principal
	otherPrincipal.FirmID = fixture.otherFirmID
	authorizer.principal = otherPrincipal
	otherRead := authorizeEpisode(t, service, permissionRead, metadata)
	if _, err := service.Get(context.Background(), otherRead, created.ID); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("Get(cross firm) error = %v, want forbidden", err)
	}
	otherPrincipal = fixture.otherInstitutionPrincipal
	authorizer.principal = otherPrincipal
	otherInstitutionRead := authorizeEpisode(t, service, permissionRead, metadata)
	if _, err := service.Get(context.Background(), otherInstitutionRead, created.ID); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("Get(cross institution) error = %v, want forbidden", err)
	}
	if _, err := service.List(context.Background(), otherInstitutionRead, ListRequest{PatientID: fixture.patientPublicID}); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("List(cross institution) error = %v, want forbidden", err)
	}
	authorizer.principal = fixture.principal
	if _, err := pool.Exec(context.Background(), `UPDATE patients SET active = FALSE, deleted_at = now() WHERE public_id = $1`, fixture.patientPublicID); err != nil {
		t.Fatalf("deactivate patient: %v", err)
	}
	activeScopeRead := authorizeEpisode(t, service, permissionRead, metadata)
	if _, err := service.Get(context.Background(), activeScopeRead, created.ID); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("Get(deactivated patient) error = %v, want forbidden", err)
	}

	var audits int
	var reopenReason string
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*), COALESCE(max(attributes->>'reason') FILTER (WHERE event_type = 'episode.reopened'), '')
		FROM audit_events
		WHERE event_type = ANY($1::text[])`, []string{"episode.created", "episode.listed", "episode.read", "episode.activated", "episode.closed", "episode.reopened"},
	).Scan(&audits, &reopenReason); err != nil {
		t.Fatalf("query episode audits: %v", err)
	}
	if audits != 9 || reopenReason != "corrected closure" {
		t.Fatalf("episode audit summary = count:%d reopenReason:%q", audits, reopenReason)
	}
}

func authorizeEpisode(t *testing.T, service *Service, permission string, metadata auth.RequestMetadata) Authorization {
	t.Helper()
	authorization, err := service.Authorize(context.Background(), "token", "csrf", permission, metadata)
	if err != nil {
		t.Fatalf("Authorize(%s) error = %v", permission, err)
	}
	return authorization
}

type episodesFixture struct {
	principal                 auth.OperationPrincipal
	otherInstitutionPrincipal auth.OperationPrincipal
	patientPublicID           string
	otherFirmID               int64
}

func episodesTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("VISIONOPUS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VISIONOPUS_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect episodes test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedEpisodesFixture(t *testing.T, pool *pgxpool.Pool) episodesFixture {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin episodes fixture: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	resetEpisodesData(t, ctx, tx)
	var institutionID, siteID, firmID, otherFirmID, userID, sessionID, patientID int64
	if err := tx.QueryRow(ctx, `INSERT INTO institutions (name) VALUES ('Episodes Service Hospital') RETURNING id`).Scan(&institutionID); err != nil {
		t.Fatalf("insert institution: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO sites (institution_id, name) VALUES ($1, 'Episodes Service Site') RETURNING id`, institutionID).Scan(&siteID); err != nil {
		t.Fatalf("insert site: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO firms (institution_id, name) VALUES ($1, 'Episodes Firm') RETURNING id`, institutionID).Scan(&firmID); err != nil {
		t.Fatalf("insert firm: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO firms (institution_id, name) VALUES ($1, 'Other Episodes Firm') RETURNING id`, institutionID).Scan(&otherFirmID); err != nil {
		t.Fatalf("insert other firm: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO users (display_name) VALUES ('Episodes Clinician') RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	now := time.Now().UTC()
	if err := tx.QueryRow(ctx, `
		INSERT INTO sessions (
			token_digest, csrf_digest, user_id, institution_id, site_id, firm_id,
			created_at, last_seen_at, idle_expires_at, absolute_expires_at, authorization_version
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$7,$8,$9,1) RETURNING id`,
		make([]byte, 32), append([]byte{1}, make([]byte, 31)...), userID, institutionID, siteID, firmID,
		now, now.Add(time.Hour), now.Add(12*time.Hour)).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO patients (
			public_id, given_name, given_name_normalized, family_name, family_name_normalized,
			date_of_birth, gender, source_system, source_record_id
		) VALUES ('11111111-1111-4111-8111-111111111111', 'Episode', 'episode', 'Patient', 'patient', DATE '1980-01-01', 'unknown', 'test', 'episodes-service')
		RETURNING id`).Scan(&patientID); err != nil {
		t.Fatalf("insert patient: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO patient_institutions (patient_id, institution_id, primary_association, association_source) VALUES ($1,$2,TRUE,'visionopus')`, patientID, institutionID); err != nil {
		t.Fatalf("associate patient: %v", err)
	}
	var otherInstitutionID, otherSiteID, otherInstitutionFirmID int64
	if err := tx.QueryRow(ctx, `INSERT INTO institutions (name) VALUES ('Other Episodes Service Hospital') RETURNING id`).Scan(&otherInstitutionID); err != nil {
		t.Fatalf("insert other institution: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO sites (institution_id, name) VALUES ($1, 'Other Episodes Service Site') RETURNING id`, otherInstitutionID).Scan(&otherSiteID); err != nil {
		t.Fatalf("insert other site: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO firms (institution_id, name) VALUES ($1, 'Other Episodes Institution Firm') RETURNING id`, otherInstitutionID).Scan(&otherInstitutionFirmID); err != nil {
		t.Fatalf("insert other institution firm: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit episodes fixture: %v", err)
	}
	return episodesFixture{
		principal:                 auth.OperationPrincipal{UserID: userID, SessionID: sessionID, InstitutionID: institutionID, SiteID: siteID, FirmID: firmID, ContextVersion: 1},
		otherInstitutionPrincipal: auth.OperationPrincipal{UserID: userID, SessionID: sessionID, InstitutionID: otherInstitutionID, SiteID: otherSiteID, FirmID: otherInstitutionFirmID, ContextVersion: 1},
		patientPublicID:           "11111111-1111-4111-8111-111111111111",
		otherFirmID:               otherFirmID,
	}
}

func resetEpisodesData(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()
	if _, err := tx.Exec(ctx, `ALTER TABLE audit_events DISABLE TRIGGER audit_events_no_truncate`); err != nil {
		t.Fatalf("disable audit truncate guard: %v", err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE event_versions DISABLE TRIGGER event_versions_no_truncate`); err != nil {
		t.Fatalf("disable event-version truncate guard: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		TRUNCATE audit_events, sessions, user_role_assignments, user_credentials,
			authentication_profiles, user_firm_preferences, user_firm_memberships,
			user_site_memberships, user_institution_memberships, users, firms, sites,
			institutions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate episodes fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE event_versions ENABLE TRIGGER event_versions_no_truncate`); err != nil {
		t.Fatalf("enable event-version truncate guard: %v", err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE audit_events ENABLE TRIGGER audit_events_no_truncate`); err != nil {
		t.Fatalf("enable audit truncate guard: %v", err)
	}
}
