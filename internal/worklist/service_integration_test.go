//go:build integration

package worklist

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/libre-eyes/internal/auth"
)

type worklistIntegrationAuthorizer struct {
	principal auth.OperationPrincipal
}

func (a worklistIntegrationAuthorizer) AuthorizeOperation(_ context.Context, _ auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error) {
	return a.principal, nil
}

func TestServiceCommandIntegrationPreservesOneAuditPerSuccessfulTransition(t *testing.T) {
	pool := worklistTestPool(t)
	principal, ticketID := seedWorklistFixture(t, pool)
	service, err := NewService(pool, worklistIntegrationAuthorizer{principal: principal})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	authorization, err := service.Authorize(context.Background(), "token", "csrf", auth.RequestMetadata{CorrelationID: "worklist-integration", SourceIPClass: "loopback"})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}

	arrived, err := service.Command(context.Background(), authorization, CommandRequest{TicketID: ticketID, ExpectedVersion: 1, Command: CommandArrive})
	if err != nil || arrived.Status != StatusArrived || arrived.Version != 2 {
		t.Fatalf("arrive = %#v, %v", arrived, err)
	}

	results := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, commandErr := service.Command(context.Background(), authorization, CommandRequest{TicketID: ticketID, ExpectedVersion: 2, Command: CommandClaim})
			results <- commandErr
		}()
	}
	group.Wait()
	close(results)

	var successes, conflicts int
	for commandErr := range results {
		switch {
		case commandErr == nil:
			successes++
		case errors.Is(commandErr, ErrConflict):
			conflicts++
		default:
			t.Fatalf("concurrent claim error = %v", commandErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent claims successes=%d conflicts=%d, want 1 each", successes, conflicts)
	}

	var status Status
	var version, auditCount int64
	if err := pool.QueryRow(context.Background(), `SELECT status, version FROM development_flow_tickets WHERE id = $1::uuid`, ticketID).Scan(&status, &version); err != nil {
		t.Fatalf("read ticket: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM development_flow_audit WHERE ticket_id = $1::uuid`, ticketID).Scan(&auditCount); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if status != StatusInProgress || version != 3 || auditCount != 2 {
		t.Fatalf("ticket status=%q version=%d audits=%d, want in_progress/3/2", status, version, auditCount)
	}
}

func worklistTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("LIBREEYES_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LIBREEYES_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect worklist test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedWorklistFixture(t *testing.T, pool *pgxpool.Pool) (auth.OperationPrincipal, string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin worklist fixture: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fixtureID := newWorklistUUID(t)
	var institutionID, siteID, firmID, userID int64
	if err := tx.QueryRow(ctx, `INSERT INTO institutions (name) VALUES ($1) RETURNING id`, "Worklist Integration "+fixtureID).Scan(&institutionID); err != nil {
		t.Fatalf("insert institution: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO sites (institution_id, name) VALUES ($1, $2) RETURNING id`, institutionID, "Worklist Site "+fixtureID).Scan(&siteID); err != nil {
		t.Fatalf("insert site: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO firms (institution_id, name) VALUES ($1, $2) RETURNING id`, institutionID, "Worklist Firm "+fixtureID).Scan(&firmID); err != nil {
		t.Fatalf("insert firm: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO users (display_name) VALUES ($1) RETURNING id`, "Worklist Clinician "+fixtureID).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO development_flow_tickets (
			id, institution_id, site_id, firm_id, synthetic_patient_id, synthetic_patient_label, status
		) VALUES ($1::uuid, $2, $3, $4, $5, $6, 'waiting')`,
		fixtureID, institutionID, siteID, firmID, "integration-patient-"+fixtureID, "Integration synthetic patient"); err != nil {
		t.Fatalf("insert ticket: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit worklist fixture: %v", err)
	}
	return auth.OperationPrincipal{UserID: userID, InstitutionID: institutionID, SiteID: siteID, FirmID: firmID, ContextVersion: 1}, fixtureID
}

func newWorklistUUID(t *testing.T) string {
	t.Helper()
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		t.Fatalf("generate fixture UUID: %v", err)
	}
	bytes[6] = bytes[6]&0x0f | 0x40
	bytes[8] = bytes[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}
