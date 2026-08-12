//go:build integration

package migrations

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestDevelopmentDiagnosisCatalogueConstraints(t *testing.T) {
	pool := patientSearchTestPool(t)
	withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
		var profileID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO development_diagnosis_profiles (code, display_name, display_order)
			VALUES ('test_diagnosis_profile', 'Test diagnosis profile', 0)
			RETURNING id`).Scan(&profileID); err != nil {
			t.Fatalf("insert diagnosis profile: %v", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO development_diagnosis_profile_selections (profile_id, code, display_name, display_order)
			VALUES ($1, 'test_diagnosis_selection', 'Test diagnosis selection', 0)`, profileID); err != nil {
			t.Fatalf("insert diagnosis selection: %v", err)
		}
		expectConstraintInSavepoint(t, ctx, tx, "23505", `
			INSERT INTO development_diagnosis_profile_selections (profile_id, code, display_name, display_order)
			VALUES ($1, 'test_diagnosis_selection_two', 'Duplicate display order', 0)`, profileID)
		expectConstraintInSavepoint(t, ctx, tx, "23514", `
			INSERT INTO development_diagnosis_profile_selections (profile_id, code, display_name, display_order)
			VALUES ($1, 'InvalidCode', 'Invalid code', 1)`, profileID)
		expectConstraintInSavepoint(t, ctx, tx, "23001", "DELETE FROM development_diagnosis_profiles WHERE id = $1", profileID)
	})
}
