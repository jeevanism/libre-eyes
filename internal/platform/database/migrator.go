// Package database provides PostgreSQL lifecycle helpers.
package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationLockName = "libreeyes.schema_migrations"

// Migrator executes embedded, versioned SQL migrations.
type Migrator struct {
	pool  *pgxpool.Pool
	files fs.FS
}

// NewMigrator creates a migration runner.
func NewMigrator(pool *pgxpool.Pool, files fs.FS) *Migrator {
	return &Migrator{pool: pool, files: files}
}

// Up applies all unapplied migrations in version order.
func (m *Migrator) Up(ctx context.Context) error {
	migrations, err := loadMigrations(m.files)
	if err != nil {
		return err
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", migrationLockName); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	for _, migration := range migrations {
		var applied bool
		if err := tx.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)",
			migration.version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", migration.version, err)
		}
		if applied {
			continue
		}
		if _, err := tx.Exec(ctx, migration.upSQL); err != nil {
			return fmt.Errorf("apply migration %06d_%s: %w", migration.version, migration.name, err)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			migration.version,
			migration.name,
		); err != nil {
			return fmt.Errorf("record migration %d: %w", migration.version, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}

// Down rolls back the latest applied migration.
func (m *Migrator) Down(ctx context.Context) error {
	migrations, err := loadMigrations(m.files)
	if err != nil {
		return err
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration rollback: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", migrationLockName); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}

	var version int64
	if err := tx.QueryRow(ctx, "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("find latest migration: %w", err)
	}

	index := slices.IndexFunc(migrations, func(m migration) bool { return m.version == version })
	if index < 0 {
		return fmt.Errorf("migration %d is applied but not embedded", version)
	}
	if _, err := tx.Exec(ctx, migrations[index].downSQL); err != nil {
		return fmt.Errorf("roll back migration %d: %w", version, err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
		return fmt.Errorf("remove migration record %d: %w", version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration rollback: %w", err)
	}
	return nil
}

type migration struct {
	version int64
	name    string
	upSQL   string
	downSQL string
}

func loadMigrations(files fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	byVersion := make(map[int64]*migration)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := strings.Split(entry.Name(), ".")
		if len(parts) != 3 || (parts[1] != "up" && parts[1] != "down") {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		identity := strings.SplitN(parts[0], "_", 2)
		if len(identity) != 2 {
			return nil, fmt.Errorf("invalid migration identity %q", entry.Name())
		}
		version, err := strconv.ParseInt(identity[0], 10, 64)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("invalid migration version in %q", entry.Name())
		}
		contents, err := fs.ReadFile(files, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}

		item := byVersion[version]
		if item == nil {
			item = &migration{version: version, name: identity[1]}
			byVersion[version] = item
		}
		if item.name != identity[1] {
			return nil, fmt.Errorf("migration %d has inconsistent names", version)
		}
		if parts[1] == "up" {
			item.upSQL = string(contents)
		} else {
			item.downSQL = string(contents)
		}
	}

	migrations := make([]migration, 0, len(byVersion))
	for _, item := range byVersion {
		if item.upSQL == "" || item.downSQL == "" {
			return nil, fmt.Errorf("migration %d must have up and down files", item.version)
		}
		migrations = append(migrations, *item)
	}
	slices.SortFunc(migrations, func(a, b migration) int { return int(a.version - b.version) })
	return migrations, nil
}
