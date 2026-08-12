//go:build integration

package migrations

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestDevelopmentOperativeNoteCatalogueConstraints(t *testing.T) {
	pool := patientSearchTestPool(t)
	withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
		if _, err := tx.Exec(ctx, `INSERT INTO development_operative_note_catalogue (category, code, display_name, display_order) VALUES ('procedure', 'development_test_procedure', 'Test procedure', 99)`); err != nil {
			t.Fatalf("insert operative-note catalogue code: %v", err)
		}
		expectConstraintInSavepoint(t, ctx, tx, "23505", `INSERT INTO development_operative_note_catalogue (category, code, display_name, display_order) VALUES ('procedure', 'development_test_procedure_two', 'Duplicate order', 99)`)
		expectConstraintInSavepoint(t, ctx, tx, "23514", `INSERT INTO development_operative_note_catalogue (category, code, display_name, display_order) VALUES ('procedure', 'production_procedure', 'Production code', 100)`)
		expectConstraintInSavepoint(t, ctx, tx, "23514", `INSERT INTO development_operative_note_catalogue (category, code, display_name, display_order) VALUES ('unknown', 'test_development_unknown', 'Unknown category', 100)`)
	})
}
