//go:build integration

package theatrebooking

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
)

type integrationAuthorizer struct{ principal auth.OperationPrincipal }

func (a integrationAuthorizer) AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error) {
	return a.principal, nil
}
func (a integrationAuthorizer) AuthorizeRead(context.Context, auth.ReadAuthorizationRequest) (auth.OperationPrincipal, error) {
	return a.principal, nil
}

func TestCommandIntegrationPreservesCapacityAndAudit(t *testing.T) {
	pool := theatreTestPool(t)
	principal, sessionID, firstRequestID, secondRequestID := seedTheatreFixture(t, pool)
	service, err := NewService(pool, integrationAuthorizer{principal: principal})
	if err != nil {
		t.Fatal(err)
	}
	authorization, err := service.AuthorizeCommand(context.Background(), "token", "csrf", auth.RequestMetadata{CorrelationID: "theatre-integration", SourceIPClass: "loopback"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Command(context.Background(), authorization, CommandRequest{RequestID: firstRequestID, ExpectedVersion: 1, TargetSessionID: sessionID, Command: CommandSchedule})
	if err != nil || updated.Status != StatusScheduled || updated.Version != 2 {
		t.Fatalf("schedule = %#v, %v", updated, err)
	}
	_, err = service.Command(context.Background(), authorization, CommandRequest{RequestID: secondRequestID, ExpectedVersion: 1, TargetSessionID: sessionID, Command: CommandSchedule})
	var conflict ConflictError
	if !errors.As(err, &conflict) || conflict.Reason != "insufficient_capacity" {
		t.Fatalf("capacity error = %v", err)
	}
	var firstAudit, secondAudit int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM development_theatre_booking_audit WHERE request_id = $1::uuid`, firstRequestID).Scan(&firstAudit); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM development_theatre_booking_audit WHERE request_id = $1::uuid`, secondRequestID).Scan(&secondAudit); err != nil {
		t.Fatal(err)
	}
	if firstAudit != 1 || secondAudit != 0 {
		t.Fatalf("audit counts = %d/%d, want 1/0", firstAudit, secondAudit)
	}
}

func theatreTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("VISIONOPUS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("VISIONOPUS_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
func seedTheatreFixture(t *testing.T, pool *pgxpool.Pool) (auth.OperationPrincipal, string, string, string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	key := newTheatreUUID(t)
	roomID, sessionID, firstID, secondID := newTheatreUUID(t), newTheatreUUID(t), newTheatreUUID(t), newTheatreUUID(t)
	var institutionID, siteID, firmID, userID int64
	if err := tx.QueryRow(ctx, `INSERT INTO institutions (name) VALUES ($1) RETURNING id`, "Theatre Integration "+key).Scan(&institutionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO sites (institution_id, name) VALUES ($1,$2) RETURNING id`, institutionID, "Theatre Site "+key).Scan(&siteID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO firms (institution_id, name) VALUES ($1,$2) RETURNING id`, institutionID, "Theatre Firm "+key).Scan(&firmID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO users (display_name) VALUES ($1) RETURNING id`, "Theatre Clinician "+key).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO development_theatre_rooms (id, institution_id, site_id, firm_id, synthetic_label) VALUES ($1::uuid,$2,$3,$4,'Integration theatre')`, roomID, institutionID, siteID, firmID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO development_theatre_sessions (id, room_id, institution_id, site_id, firm_id, starts_at, ends_at, capacity_minutes) VALUES ($1::uuid,$2::uuid,$3,$4,$5,'2026-08-10T08:00:00Z','2026-08-10T09:00:00Z',60)`, sessionID, roomID, institutionID, siteID, firmID); err != nil {
		t.Fatal(err)
	}
	for _, requestID := range []string{firstID, secondID} {
		if _, err := tx.Exec(ctx, `INSERT INTO development_booking_requests (id, institution_id, site_id, firm_id, synthetic_label, requested_duration_minutes) VALUES ($1::uuid,$2,$3,$4,$5,60)`, requestID, institutionID, siteID, firmID, "Integration request "+requestID); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return auth.OperationPrincipal{UserID: userID, InstitutionID: institutionID, SiteID: siteID, FirmID: firmID, ContextVersion: 1}, sessionID, firstID, secondID
}
func newTheatreUUID(t *testing.T) string {
	t.Helper()
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		t.Fatal(err)
	}
	bytes[6] = bytes[6]&0x0f | 0x40
	bytes[8] = bytes[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}
