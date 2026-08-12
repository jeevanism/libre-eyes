//go:build integration

package migrations

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestIOPDevelopmentCatalogueConstraints(t *testing.T) {
	pool := patientSearchTestPool(t)
	withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
		var firstProfileID, secondProfileID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO intraocular_pressure_profiles (code, display_name, display_order)
			VALUES ('test_iop_profile_a', 'Test IOP profile A', 0)
			RETURNING id`).Scan(&firstProfileID); err != nil {
			t.Fatalf("insert first IOP profile: %v", err)
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO intraocular_pressure_profiles (code, display_name, display_order)
			VALUES ('test_iop_profile_b', 'Test IOP profile B', 1)
			RETURNING id`).Scan(&secondProfileID); err != nil {
			t.Fatalf("insert second IOP profile: %v", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO intraocular_pressure_profile_values (profile_id, code, display_value, mmhg, display_order)
			VALUES ($1, 'test_iop_value_a', '14 mmHg', 14, 0)`, firstProfileID); err != nil {
			t.Fatalf("insert IOP value: %v", err)
		}
		expectConstraintInSavepoint(t, ctx, tx, "23505", `
			INSERT INTO intraocular_pressure_profile_values (profile_id, code, display_value, mmhg, display_order)
			VALUES ($1, 'test_iop_value_b', 'Duplicate mmHg', 14, 1)`, firstProfileID)
		expectConstraintInSavepoint(t, ctx, tx, "23514", `
			INSERT INTO intraocular_pressure_profile_values (profile_id, code, display_value, mmhg, display_order)
			VALUES ($1, 'test_iop_value_c', 'Invalid value', 100, 2)`, secondProfileID)
		expectConstraintInSavepoint(t, ctx, tx, "23001", "DELETE FROM intraocular_pressure_profiles WHERE id = $1", firstProfileID)
	})
}
