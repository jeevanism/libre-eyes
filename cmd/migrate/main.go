package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/db/migrations"
	"github.com/jeevanism/visionopus/internal/config"
	"github.com/jeevanism/visionopus/internal/platform/database"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) != 1 || (args[0] != "up" && args[0] != "down") {
		return fmt.Errorf("usage: migrate up|down")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	runner := database.NewMigrator(pool, migrations.Files)
	if args[0] == "up" {
		return runner.Up(ctx)
	}
	return runner.Down(ctx)
}
