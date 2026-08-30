//go:build integration

package auth

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthenticationLifecycleIntegration(t *testing.T) {
	service, pool := integrationService(t, 5)
	seed := seedIntegrationUser(t, pool)
	metadata := RequestMetadata{CorrelationID: "integration-lifecycle", SourceIPClass: "loopback"}

	options, err := service.ListLoginOptions(context.Background())
	if err != nil {
		t.Fatalf("ListLoginOptions() error = %v", err)
	}
	if len(options) != 2 {
		t.Fatalf("login option count = %d, want 2", len(options))
	}

	created, err := service.Login(context.Background(), LoginRequest{
		Username: "  CLINICIAN  ", Password: "synthetic-password",
		InstitutionID: seed.institution1, SiteID: seed.site1,
	}, metadata)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if created.Token == "" || created.Session.Context.Firm.ID != seed.firm1 {
		t.Fatalf("created session = %#v", created)
	}
	if !slices.Contains(created.Session.Permissions, permissionSwitchContext) {
		t.Fatalf("permissions = %v, want %s", created.Session.Permissions, permissionSwitchContext)
	}

	current, err := service.CurrentSession(context.Background(), created.Token, metadata)
	if err != nil {
		t.Fatalf("CurrentSession() error = %v", err)
	}
	if current.User.DisplayName != "Synthetic Clinician" || current.CSRFToken != created.Session.CSRFToken {
		t.Fatalf("current session = %#v", current)
	}

	replaced, err := service.ReplaceContext(context.Background(), created.Token, current.CSRFToken, ContextRequest{
		InstitutionID: seed.institution2, SiteID: seed.site2, FirmID: seed.firm2, Version: current.ContextVersion,
	}, metadata)
	if err != nil {
		t.Fatalf("ReplaceContext() error = %v", err)
	}
	if replaced.Context.Institution.ID != seed.institution2 || replaced.ContextVersion != 2 {
		t.Fatalf("replaced context = %#v", replaced)
	}

	_, err = service.ReplaceContext(context.Background(), created.Token, current.CSRFToken, ContextRequest{
		InstitutionID: seed.institution1, SiteID: seed.site1, FirmID: seed.firm1, Version: 1,
	}, metadata)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale ReplaceContext() error = %v, want ErrConflict", err)
	}

	if err := service.Logout(context.Background(), created.Token, replaced.CSRFToken, metadata); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := service.CurrentSession(context.Background(), created.Token, metadata); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("CurrentSession() after logout error = %v, want ErrUnauthenticated", err)
	}

	var successes, contextChanges, logouts int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			count(*) FILTER (WHERE event_type = 'auth.login_succeeded'),
			count(*) FILTER (WHERE event_type = 'auth.context_changed'),
			count(*) FILTER (WHERE event_type = 'auth.logout')
		FROM audit_events`).Scan(&successes, &contextChanges, &logouts); err != nil {
		t.Fatalf("query audit counts: %v", err)
	}
	if successes != 1 || contextChanges != 1 || logouts != 1 {
		t.Fatalf("audit counts = success:%d context:%d logout:%d, want 1 each", successes, contextChanges, logouts)
	}
}

func TestConcurrentFailureAccountingLocksCredential(t *testing.T) {
	service, pool := integrationService(t, 2)
	seed := seedIntegrationUser(t, pool)
	metadata := RequestMetadata{CorrelationID: "integration-lockout", SourceIPClass: "loopback"}
	request := LoginRequest{
		Username: "clinician", Password: "wrong-password",
		InstitutionID: seed.institution1, SiteID: seed.site1,
	}

	errorsChannel := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := service.Login(context.Background(), request, metadata)
			errorsChannel <- err
		}()
	}
	for range 2 {
		if err := <-errorsChannel; !errors.Is(err, ErrAuthenticationFailed) {
			t.Fatalf("Login() error = %v, want ErrAuthenticationFailed", err)
		}
	}

	var state string
	var failedAttempts int
	if err := pool.QueryRow(context.Background(), `
		SELECT state::text, failed_attempts FROM user_credentials WHERE canonical_username = 'clinician'`,
	).Scan(&state, &failedAttempts); err != nil {
		t.Fatalf("query credential state: %v", err)
	}
	if state != "soft_locked" || failedAttempts != 2 {
		t.Fatalf("credential = state:%s attempts:%d, want soft_locked/2", state, failedAttempts)
	}
}

func TestAuthorizationDenialAuditIsBounded(t *testing.T) {
	service, pool := integrationService(t, 5)
	seed := seedIntegrationUser(t, pool)
	metadata := RequestMetadata{CorrelationID: "integration-denial", SourceIPClass: "loopback"}
	created, err := service.Login(context.Background(), LoginRequest{
		Username: "clinician", Password: "synthetic-password",
		InstitutionID: seed.institution1, SiteID: seed.site1,
	}, metadata)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		DELETE FROM user_role_assignments WHERE user_id = $1`, created.Session.User.ID); err != nil {
		t.Fatalf("remove role assignment: %v", err)
	}

	for attempt := 0; attempt < 2; attempt++ {
		if _, err := service.CurrentSession(context.Background(), created.Token, metadata); !errors.Is(err, ErrForbidden) {
			t.Fatalf("CurrentSession() attempt %d error = %v, want ErrForbidden", attempt+1, err)
		}
	}

	var denials int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM audit_events
		WHERE event_type = 'auth.authorization_denied'
			AND reason_code = 'missing_session_read_permission'`).Scan(&denials); err != nil {
		t.Fatalf("query authorization denials: %v", err)
	}
	if denials != 1 {
		t.Fatalf("authorization denial count = %d, want 1", denials)
	}
}

func TestAuthorizeOperationEnforcesCSRFContextAndPermission(t *testing.T) {
	service, pool := integrationService(t, 5)
	seed := seedIntegrationUser(t, pool)
	metadata := RequestMetadata{CorrelationID: "integration-operation-auth", SourceIPClass: "loopback"}
	created, err := service.Login(context.Background(), LoginRequest{
		Username: "clinician", Password: "synthetic-password",
		InstitutionID: seed.institution1, SiteID: seed.site1,
	}, metadata)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	request := OperationAuthorizationRequest{
		Token: created.Token, CSRFToken: created.Session.CSRFToken,
		Permission: "patient.search", DeniedEventType: "patient_search.denied",
		Metadata: metadata,
	}

	if _, err := service.AuthorizeOperation(context.Background(), request); !errors.Is(err, ErrForbidden) {
		t.Fatalf("AuthorizeOperation() without permission error = %v, want ErrForbidden", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO user_role_assignments (user_id, role_id, role_scope, institution_id)
		SELECT $1, id, scope, $2 FROM roles WHERE name = 'Clinical Viewer'`,
		created.Session.User.ID, seed.institution1,
	); err != nil {
		t.Fatalf("assign Clinical Viewer: %v", err)
	}
	principal, err := service.AuthorizeOperation(context.Background(), request)
	if err != nil {
		t.Fatalf("AuthorizeOperation() error = %v", err)
	}
	if principal.UserID != created.Session.User.ID || principal.InstitutionID != seed.institution1 ||
		principal.SiteID != seed.site1 || principal.FirmID != seed.firm1 || principal.SessionID < 1 {
		t.Fatalf("principal = %#v, want current validated session context", principal)
	}

	badCSRF := request
	badCSRF.CSRFToken = "incorrect-csrf"
	if _, err := pool.Exec(context.Background(), `
		CREATE FUNCTION fail_operation_denial_audit() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.event_type = 'patient_search.denied' THEN
				RAISE EXCEPTION 'synthetic denial audit failure';
			END IF;
			RETURN NEW;
		END;
		$$`); err != nil {
		t.Fatalf("create denial audit failure function: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		CREATE TRIGGER operation_denial_audit_failure
		BEFORE INSERT ON audit_events
		FOR EACH ROW EXECUTE FUNCTION fail_operation_denial_audit()`); err != nil {
		t.Fatalf("create denial audit failure trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS operation_denial_audit_failure ON audit_events`)
		_, _ = pool.Exec(context.Background(), `DROP FUNCTION IF EXISTS fail_operation_denial_audit()`)
	})
	if _, err := service.AuthorizeOperation(context.Background(), badCSRF); !errors.Is(err, ErrCSRF) {
		t.Fatalf("AuthorizeOperation() unavailable denial audit error = %v, want ErrCSRF", err)
	}
	if _, err := pool.Exec(context.Background(), `DROP TRIGGER operation_denial_audit_failure ON audit_events`); err != nil {
		t.Fatalf("drop denial audit failure trigger: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DROP FUNCTION fail_operation_denial_audit()`); err != nil {
		t.Fatalf("drop denial audit failure function: %v", err)
	}
	if _, err := service.AuthorizeOperation(context.Background(), badCSRF); !errors.Is(err, ErrCSRF) {
		t.Fatalf("AuthorizeOperation() bad CSRF error = %v, want ErrCSRF", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE user_site_memberships SET active = FALSE
		WHERE user_id = $1 AND site_id = $2`, created.Session.User.ID, seed.site1); err != nil {
		t.Fatalf("deactivate site membership: %v", err)
	}
	if _, err := service.AuthorizeOperation(context.Background(), request); !errors.Is(err, ErrForbidden) {
		t.Fatalf("AuthorizeOperation() inaccessible context error = %v, want ErrForbidden", err)
	}

	var permissionDenied, csrfDenied, contextDenied int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			count(*) FILTER (WHERE reason_code = 'permission_denied'),
			count(*) FILTER (WHERE reason_code = 'csrf_rejected'),
			count(*) FILTER (WHERE reason_code = 'context_inaccessible')
		FROM audit_events WHERE event_type = 'patient_search.denied'`,
	).Scan(&permissionDenied, &csrfDenied, &contextDenied); err != nil {
		t.Fatalf("query operation denial audit: %v", err)
	}
	if permissionDenied != 1 || csrfDenied != 1 || contextDenied != 1 {
		t.Fatalf("denial audit counts = permission:%d csrf:%d context:%d, want 1 each", permissionDenied, csrfDenied, contextDenied)
	}
}

func TestCurrentSessionBoundsRefreshWrites(t *testing.T) {
	service, pool := integrationService(t, 5)
	seed := seedIntegrationUser(t, pool)
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return base }
	metadata := RequestMetadata{CorrelationID: "integration-refresh", SourceIPClass: "loopback"}
	created, err := service.Login(context.Background(), LoginRequest{
		Username: "clinician", Password: "synthetic-password",
		InstitutionID: seed.institution1, SiteID: seed.site1,
	}, metadata)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	tokenHash := tokenDigest(created.Token)
	var initialVersion int64
	var initialLastSeen, initialIdleExpiry time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT version, last_seen_at, idle_expires_at
		FROM sessions WHERE token_digest = $1`, tokenHash[:]).Scan(
		&initialVersion, &initialLastSeen, &initialIdleExpiry,
	); err != nil {
		t.Fatalf("query initial session refresh state: %v", err)
	}

	service.now = func() time.Time { return base.Add(30 * time.Second) }
	current, err := service.CurrentSession(context.Background(), created.Token, metadata)
	if err != nil {
		t.Fatalf("CurrentSession() before refresh interval error = %v", err)
	}
	var earlyVersion int64
	var earlyLastSeen, earlyIdleExpiry time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT version, last_seen_at, idle_expires_at
		FROM sessions WHERE token_digest = $1`, tokenHash[:]).Scan(
		&earlyVersion, &earlyLastSeen, &earlyIdleExpiry,
	); err != nil {
		t.Fatalf("query early session refresh state: %v", err)
	}
	if earlyVersion != initialVersion || !earlyLastSeen.Equal(initialLastSeen) || !earlyIdleExpiry.Equal(initialIdleExpiry) {
		t.Fatalf("early session refresh changed storage: version=%d lastSeen=%s idle=%s", earlyVersion, earlyLastSeen, earlyIdleExpiry)
	}
	if !current.IdleExpiresAt.Equal(initialIdleExpiry) {
		t.Fatalf("early CurrentSession() idle expiry = %s, want %s", current.IdleExpiresAt, initialIdleExpiry)
	}

	refreshTime := base.Add(61 * time.Second)
	service.now = func() time.Time { return refreshTime }
	current, err = service.CurrentSession(context.Background(), created.Token, metadata)
	if err != nil {
		t.Fatalf("CurrentSession() after refresh interval error = %v", err)
	}
	var refreshedVersion int64
	var refreshedLastSeen, refreshedIdleExpiry time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT version, last_seen_at, idle_expires_at
		FROM sessions WHERE token_digest = $1`, tokenHash[:]).Scan(
		&refreshedVersion, &refreshedLastSeen, &refreshedIdleExpiry,
	); err != nil {
		t.Fatalf("query refreshed session state: %v", err)
	}
	if refreshedVersion != initialVersion+1 || !refreshedLastSeen.Equal(refreshTime) {
		t.Fatalf("refreshed session = version:%d lastSeen:%s, want version:%d lastSeen:%s", refreshedVersion, refreshedLastSeen, initialVersion+1, refreshTime)
	}
	wantIdleExpiry := refreshTime.Add(15 * time.Minute)
	if !refreshedIdleExpiry.Equal(wantIdleExpiry) || !current.IdleExpiresAt.Equal(wantIdleExpiry) {
		t.Fatalf("refreshed idle expiry = stored:%s returned:%s, want %s", refreshedIdleExpiry, current.IdleExpiresAt, wantIdleExpiry)
	}
}

func TestInactiveUserSessionIsRevoked(t *testing.T) {
	service, pool := integrationService(t, 5)
	seed := seedIntegrationUser(t, pool)
	metadata := RequestMetadata{CorrelationID: "integration-disabled-user", SourceIPClass: "loopback"}
	created, err := service.Login(context.Background(), LoginRequest{
		Username: "clinician", Password: "synthetic-password",
		InstitutionID: seed.institution1, SiteID: seed.site1,
	}, metadata)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE users SET active = FALSE WHERE id = $1`, created.Session.User.ID); err != nil {
		t.Fatalf("disable user: %v", err)
	}

	if _, err := service.CurrentSession(context.Background(), created.Token, metadata); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("CurrentSession() disabled user error = %v, want ErrUnauthenticated", err)
	}

	tokenHash := tokenDigest(created.Token)
	var reason string
	if err := pool.QueryRow(context.Background(), `
		SELECT revocation_reason FROM sessions WHERE token_digest = $1`, tokenHash[:]).Scan(&reason); err != nil {
		t.Fatalf("query disabled-user revocation: %v", err)
	}
	if reason != "account_disabled" {
		t.Fatalf("revocation reason = %q, want account_disabled", reason)
	}
	var auditCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM audit_events
		WHERE event_type = 'auth.session_revoked' AND reason_code = 'account_disabled'`).Scan(&auditCount); err != nil {
		t.Fatalf("query disabled-user audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("disabled-user audit count = %d, want 1", auditCount)
	}
}

func TestRoleAssignmentScopeIsEnforced(t *testing.T) {
	_, pool := integrationService(t, 5)
	seed := seedIntegrationUser(t, pool)
	ctx := context.Background()
	var userID int64
	if err := pool.QueryRow(ctx, `
		SELECT user_id FROM user_credentials WHERE canonical_username = 'clinician'`).Scan(&userID); err != nil {
		t.Fatalf("query integration user: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO user_role_assignments (user_id, role_id, role_scope, institution_id)
		SELECT $1, id, scope, NULL FROM roles WHERE name = 'LibreEyes User'`, userID); err == nil {
		t.Fatal("institution-scoped role assignment without institution succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO user_role_assignments (user_id, role_id, role_scope, institution_id)
		SELECT $1, id, scope, $2 FROM roles WHERE name = 'Global Administrator'`, userID, seed.institution1); err == nil {
		t.Fatal("global role assignment with institution succeeded")
	}
}

func TestAuditEventsCannotBeTruncated(t *testing.T) {
	_, pool := integrationService(t, 5)
	seedIntegrationUser(t, pool)
	if _, err := pool.Exec(context.Background(), `TRUNCATE audit_events`); err == nil {
		t.Fatal("TRUNCATE audit_events succeeded")
	}
}

type integrationSeed struct {
	institution1 int64
	institution2 int64
	site1        int64
	site2        int64
	firm1        int64
	firm2        int64
}

func integrationService(t *testing.T, failureLimit int) (*Service, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("LIBREEYES_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("LIBREEYES_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	service, err := NewService(pool, ServiceConfig{
		CSRFKey:            []byte("01234567890123456789012345678901"),
		SessionIdleTimeout: 15 * time.Minute, SessionAbsoluteTimeout: 12 * time.Hour,
		LoginFailureLimit: failureLimit, SoftLockDuration: 15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service, pool
}

func seedIntegrationUser(t *testing.T, pool *pgxpool.Pool) integrationSeed {
	t.Helper()
	ctx := context.Background()
	resetIntegrationData(t, pool)

	seed := integrationSeed{}
	if err := pool.QueryRow(ctx, "INSERT INTO institutions (name) VALUES ('Vision Hospital') RETURNING id").Scan(&seed.institution1); err != nil {
		t.Fatalf("insert institution 1: %v", err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO institutions (name) VALUES ('Community Eye Centre') RETURNING id").Scan(&seed.institution2); err != nil {
		t.Fatalf("insert institution 2: %v", err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO sites (institution_id, name) VALUES ($1, 'Main Clinic') RETURNING id", seed.institution1).Scan(&seed.site1); err != nil {
		t.Fatalf("insert site 1: %v", err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO sites (institution_id, name) VALUES ($1, 'Satellite Clinic') RETURNING id", seed.institution2).Scan(&seed.site2); err != nil {
		t.Fatalf("insert site 2: %v", err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO firms (institution_id, name) VALUES ($1, 'Ophthalmology') RETURNING id", seed.institution1).Scan(&seed.firm1); err != nil {
		t.Fatalf("insert firm 1: %v", err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO firms (institution_id, name) VALUES ($1, 'Retina Service') RETURNING id", seed.institution2).Scan(&seed.firm2); err != nil {
		t.Fatalf("insert firm 2: %v", err)
	}
	var userID, profileID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users (display_name) VALUES ('Synthetic Clinician') RETURNING id").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO authentication_profiles (institution_id, method, name) VALUES ($1, 'LOCAL', 'Local') RETURNING id", seed.institution1).Scan(&profileID); err != nil {
		t.Fatalf("insert profile: %v", err)
	}
	hash, err := (PasswordManager{}).Hash("synthetic-password")
	if err != nil {
		t.Fatalf("hash synthetic password: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO user_credentials (
			user_id, authentication_profile_id, canonical_username,
			password_hash, hash_scheme, hash_version
		) VALUES ($1, $2, 'clinician', $3, 'argon2id', 19)`, userID, profileID, hash); err != nil {
		t.Fatalf("insert credential: %v", err)
	}
	membershipStatements := []struct {
		query string
		args  []any
	}{
		{"INSERT INTO user_institution_memberships (user_id, institution_id) VALUES ($1, $2), ($1, $3)", []any{userID, seed.institution1, seed.institution2}},
		{"INSERT INTO user_site_memberships (user_id, site_id) VALUES ($1, $2), ($1, $3)", []any{userID, seed.site1, seed.site2}},
		{"INSERT INTO user_firm_memberships (user_id, firm_id) VALUES ($1, $2), ($1, $3)", []any{userID, seed.firm1, seed.firm2}},
		{"INSERT INTO user_firm_preferences (user_id, firm_id, position) VALUES ($1, $2, 0), ($1, $3, 1)", []any{userID, seed.firm1, seed.firm2}},
	}
	for _, statement := range membershipStatements {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("insert membership: %v", err)
		}
	}
	for _, institutionID := range []int64{seed.institution1, seed.institution2} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO user_role_assignments (user_id, role_id, role_scope, institution_id)
			SELECT $1, id, scope, $2 FROM roles WHERE name = 'LibreEyes User'`, userID, institutionID); err != nil {
			t.Fatalf("insert role assignment: %v", err)
		}
	}
	return seed
}

func resetIntegrationData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin test-data reset: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `ALTER TABLE audit_events DISABLE TRIGGER audit_events_no_truncate`); err != nil {
		t.Fatalf("disable audit truncate guard for test reset: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		TRUNCATE audit_events, sessions, user_role_assignments, user_credentials,
			authentication_profiles, user_firm_preferences, user_firm_memberships,
			user_site_memberships, user_institution_memberships, users, firms, sites,
			institutions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate test data: %v", err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE audit_events ENABLE TRIGGER audit_events_no_truncate`); err != nil {
		t.Fatalf("enable audit truncate guard after test reset: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit test-data reset: %v", err)
	}
}
