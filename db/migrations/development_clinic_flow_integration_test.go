//go:build integration

package migrations

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestDevelopmentClinicFlowMigrationConstraints(t *testing.T) {
	pool := patientSearchTestPool(t)

	t.Run("permission and supporting indexes exist", func(t *testing.T) {
		var permission, ticketIndex, auditIndex bool
		if err := pool.QueryRow(context.Background(), `
			SELECT
				EXISTS (SELECT 1 FROM permissions WHERE name = 'worklist.development_flow.manage'),
				to_regclass('public.development_flow_tickets_context_status_idx') IS NOT NULL,
				to_regclass('public.development_flow_audit_ticket_idx') IS NOT NULL`,
		).Scan(&permission, &ticketIndex, &auditIndex); err != nil {
			t.Fatalf("read development clinic-flow migration objects: %v", err)
		}
		if !permission || !ticketIndex || !auditIndex {
			t.Fatalf("permission=%t ticketIndex=%t auditIndex=%t", permission, ticketIndex, auditIndex)
		}
	})

	t.Run("completed tickets retain an owner and audit is append-only", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			institutionID := insertInstitution(ctx, t, tx, "Development Flow Hospital")
			var siteID, firmID, userID int64
			if err := tx.QueryRow(ctx, `INSERT INTO sites (institution_id, name) VALUES ($1, 'Development Flow Site') RETURNING id`, institutionID).Scan(&siteID); err != nil {
				t.Fatalf("insert site: %v", err)
			}
			if err := tx.QueryRow(ctx, `INSERT INTO firms (institution_id, name) VALUES ($1, 'Development Flow Firm') RETURNING id`, institutionID).Scan(&firmID); err != nil {
				t.Fatalf("insert firm: %v", err)
			}
			if err := tx.QueryRow(ctx, `INSERT INTO users (display_name) VALUES ('Development Flow User') RETURNING id`).Scan(&userID); err != nil {
				t.Fatalf("insert user: %v", err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO development_flow_tickets (
					id, institution_id, site_id, firm_id, synthetic_patient_id, synthetic_patient_label, status, assignee_user_id
				) VALUES ('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',$1,$2,$3,'synthetic-test','Synthetic test patient','completed',$4)`, institutionID, siteID, firmID, userID); err != nil {
				t.Fatalf("insert completed ticket: %v", err)
			}
			expectConstraintInSavepoint(t, ctx, tx, "23514", `
				INSERT INTO development_flow_tickets (
					id, institution_id, site_id, firm_id, synthetic_patient_id, synthetic_patient_label, status
				) VALUES ('bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',$1,$2,$3,'synthetic-invalid','Synthetic invalid patient','completed')`, institutionID, siteID, firmID)
			if _, err := tx.Exec(ctx, `
				INSERT INTO development_flow_audit (
					ticket_id, actor_user_id, institution_id, site_id, firm_id, command, prior_state, next_state, resulting_version, correlation_id
				) VALUES ('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',$1,$2,$3,$4,'complete','in_progress','completed',2,'development-flow-test')`, userID, institutionID, siteID, firmID); err != nil {
				t.Fatalf("insert development flow audit: %v", err)
			}
			expectConstraintInSavepoint(t, ctx, tx, "P0001", "UPDATE development_flow_audit SET command = 'claim' WHERE ticket_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'")
			expectConstraintInSavepoint(t, ctx, tx, "P0001", "DELETE FROM development_flow_audit WHERE ticket_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'")
		})
	})
}
