//go:build integration

package migrations

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestVisualAcuityDevelopmentCatalogueConstraints(t *testing.T) {
	pool := patientSearchTestPool(t)
	withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
		var firstUnitID, secondUnitID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO visual_acuity_units (code, display_name, display_order)
			VALUES ('test_distance_scale_a', 'Test distance scale A', 0)
			RETURNING id`).Scan(&firstUnitID); err != nil {
			t.Fatalf("insert first visual acuity unit: %v", err)
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO visual_acuity_units (code, display_name, display_order)
			VALUES ('test_distance_scale_b', 'Test distance scale B', 1)
			RETURNING id`).Scan(&secondUnitID); err != nil {
			t.Fatalf("insert second visual acuity unit: %v", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO visual_acuity_unit_values (unit_id, code, display_value, base_value, display_order)
			VALUES ($1, 'test_value_unique', 'Test value', 0.1000, 0)`, firstUnitID); err != nil {
			t.Fatalf("insert visual acuity value: %v", err)
		}
		expectConstraintInSavepoint(t, ctx, tx, "23505", `
			INSERT INTO visual_acuity_unit_values (unit_id, code, display_value, base_value, display_order)
			VALUES ($1, 'test_value_unique', 'Duplicate test value', 0.2000, 1)`, secondUnitID)
		expectConstraintInSavepoint(t, ctx, tx, "23001", "DELETE FROM visual_acuity_units WHERE id = $1", firstUnitID)
		expectConstraintInSavepoint(t, ctx, tx, "23514", `
			INSERT INTO visual_acuity_methods (code, display_name, display_order)
			VALUES ('Invalid-Method', 'Invalid method', 0)`)
	})
}
