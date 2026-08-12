//go:build integration

package examination

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresDiagnosisDemoCatalogueDetectsDevelopmentFixture(t *testing.T) {
	if os.Getenv("VISIONOPUS_TEST_DEVELOPMENT_FIXTURE") != "true" {
		t.Skip("VISIONOPUS_TEST_DEVELOPMENT_FIXTURE is not true")
	}
	databaseURL := os.Getenv("VISIONOPUS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VISIONOPUS_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)

	present, err := (postgresDiagnosisDemoCatalogue{pool: pool}).hasDevelopmentCodes(context.Background())
	if err != nil {
		t.Fatalf("hasDevelopmentCodes() error = %v", err)
	}
	if !present {
		t.Fatal("development diagnosis fixture was not detected")
	}
}

func TestPostgresDiagnosisDemoCatalogueRejectsAnyProductionRow(t *testing.T) {
	databaseURL := os.Getenv("VISIONOPUS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("VISIONOPUS_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO development_diagnosis_profiles (code, display_name, display_order) VALUES ('test_production_guard_profile', 'Test production guard profile', 999) ON CONFLICT (code) DO NOTHING`); err != nil {
		t.Fatalf("insert non-development-prefix profile: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM development_diagnosis_profiles WHERE code = 'test_production_guard_profile'")
	})
	present, err := (postgresDiagnosisDemoCatalogue{pool: pool}).hasDevelopmentCodes(ctx)
	if err != nil {
		t.Fatalf("hasDevelopmentCodes() error = %v", err)
	}
	if !present {
		t.Fatal("non-development-prefix row was not detected by production guard")
	}
}
