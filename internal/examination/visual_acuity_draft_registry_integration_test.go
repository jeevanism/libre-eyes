//go:build integration

package examination

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCatalogueDetectsDevelopmentFixture(t *testing.T) {
	if os.Getenv("LIBREEYES_TEST_DEVELOPMENT_FIXTURE") != "true" {
		t.Skip("LIBREEYES_TEST_DEVELOPMENT_FIXTURE is not true")
	}
	databaseURL := os.Getenv("LIBREEYES_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LIBREEYES_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)

	present, err := (postgresCatalogue{pool: pool}).hasActiveDevelopmentCodes(context.Background())
	if err != nil {
		t.Fatalf("hasActiveDevelopmentCodes() error = %v", err)
	}
	if !present {
		t.Fatal("active development Visual Acuity fixture was not detected")
	}
}
