//go:build integration

package examination

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresIOPCatalogueDetectsDevelopmentFixture(t *testing.T) {
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

	present, err := (postgresIOPCatalogue{pool: pool}).hasDevelopmentCodes(context.Background())
	if err != nil {
		t.Fatalf("hasDevelopmentCodes() error = %v", err)
	}
	if !present {
		t.Fatal("development IOP fixture was not detected")
	}
}
