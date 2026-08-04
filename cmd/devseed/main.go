// Command devseed creates one synthetic development identity and context.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/config"
)

func main() {
	if err := run(context.Background()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Environment != "development" {
		return errors.New("devseed may run only when VISIONOPUS_ENV=development")
	}
	username := strings.ToLower(strings.TrimSpace(valueOrDefault("VISIONOPUS_DEV_USERNAME", "clinician")))
	displayName := strings.TrimSpace(valueOrDefault("VISIONOPUS_DEV_DISPLAY_NAME", "Synthetic Clinician"))
	password := os.Getenv("VISIONOPUS_DEV_PASSWORD")
	if username == "" || displayName == "" || password == "" {
		return errors.New("development username, display name, and VISIONOPUS_DEV_PASSWORD are required")
	}

	passwordHash, err := (auth.PasswordManager{}).Hash(password)
	if err != nil {
		return fmt.Errorf("hash development password: %w", err)
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to development database: %w", err)
	}
	defer pool.Close()

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin development seed: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	institutionID, err := findOrInsert(ctx, tx,
		"SELECT id FROM institutions WHERE name = $1 ORDER BY id LIMIT 1",
		"INSERT INTO institutions (name) VALUES ($1) RETURNING id",
		"VisionOpus Development Hospital",
	)
	if err != nil {
		return err
	}
	siteID, err := findOrInsert(ctx, tx,
		"SELECT id FROM sites WHERE institution_id = $1 AND name = $2",
		"INSERT INTO sites (institution_id, name) VALUES ($1, $2) RETURNING id",
		institutionID, "Development Eye Clinic",
	)
	if err != nil {
		return err
	}
	firmID, err := findOrInsert(ctx, tx,
		"SELECT id FROM firms WHERE institution_id = $1 AND name = $2",
		"INSERT INTO firms (institution_id, name) VALUES ($1, $2) RETURNING id",
		institutionID, "Development Ophthalmology",
	)
	if err != nil {
		return err
	}
	profileID, err := findOrInsert(ctx, tx,
		"SELECT id FROM authentication_profiles WHERE institution_id = $1 AND method = 'LOCAL' AND name = $2",
		"INSERT INTO authentication_profiles (institution_id, method, name) VALUES ($1, 'LOCAL', $2) RETURNING id",
		institutionID, "Development Local",
	)
	if err != nil {
		return err
	}

	var userID int64
	err = tx.QueryRow(ctx, "SELECT user_id FROM user_credentials WHERE canonical_username = $1", username).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, "INSERT INTO users (display_name) VALUES ($1) RETURNING id", displayName).Scan(&userID); err != nil {
			return fmt.Errorf("insert development user: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find development user: %w", err)
	} else if _, err := tx.Exec(ctx, "UPDATE users SET display_name = $2, active = TRUE, updated_at = now() WHERE id = $1", userID, displayName); err != nil {
		return fmt.Errorf("update development user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO user_credentials (
			user_id, authentication_profile_id, canonical_username,
			password_hash, hash_scheme, hash_version
		) VALUES ($1, $2, $3, $4, 'argon2id', 19)
		ON CONFLICT (canonical_username) DO UPDATE SET
			authentication_profile_id = EXCLUDED.authentication_profile_id,
			password_hash = EXCLUDED.password_hash,
			hash_scheme = EXCLUDED.hash_scheme,
			hash_version = EXCLUDED.hash_version,
			state = 'active', failed_attempts = 0, soft_locked_until = NULL,
			active = TRUE, version = user_credentials.version + 1, updated_at = now()`,
		userID, profileID, username, passwordHash); err != nil {
		return fmt.Errorf("upsert development credential: %w", err)
	}

	memberships := []struct {
		query string
		id    int64
	}{
		{"INSERT INTO user_institution_memberships (user_id, institution_id) VALUES ($1, $2) ON CONFLICT (user_id, institution_id) DO UPDATE SET active = TRUE", institutionID},
		{"INSERT INTO user_site_memberships (user_id, site_id) VALUES ($1, $2) ON CONFLICT (user_id, site_id) DO UPDATE SET active = TRUE", siteID},
		{"INSERT INTO user_firm_memberships (user_id, firm_id) VALUES ($1, $2) ON CONFLICT (user_id, firm_id) DO UPDATE SET active = TRUE", firmID},
	}
	for _, membership := range memberships {
		if _, err := tx.Exec(ctx, membership.query, userID, membership.id); err != nil {
			return fmt.Errorf("upsert development membership: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM user_firm_preferences
		WHERE user_id = $1 AND (firm_id = $2 OR position = 0)`, userID, firmID); err != nil {
		return fmt.Errorf("replace development firm preference: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_firm_preferences (user_id, firm_id, position)
		VALUES ($1, $2, 0)`, userID, firmID); err != nil {
		return fmt.Errorf("upsert development firm preference: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_role_assignments (user_id, role_id, role_scope, institution_id)
		SELECT $1, id, scope, $2 FROM roles WHERE name = 'VisionOpus User'
		ON CONFLICT (user_id, role_id, institution_id) DO UPDATE SET active = TRUE`, userID, institutionID); err != nil {
		return fmt.Errorf("upsert development role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit development seed: %w", err)
	}
	_, _ = fmt.Fprintf(os.Stdout, "seeded synthetic user %q for institution %d, site %d, firm %d\n", username, institutionID, siteID, firmID)
	return nil
}

func findOrInsert(ctx context.Context, tx pgx.Tx, findQuery, insertQuery string, args ...any) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, findQuery, args...).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("find development record: %w", err)
	}
	if err := tx.QueryRow(ctx, insertQuery, args...).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert development record: %w", err)
	}
	return id, nil
}

func valueOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
