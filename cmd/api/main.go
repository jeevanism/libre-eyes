package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
	authhttp "github.com/jeevanism/visionopus/internal/auth/http"
	"github.com/jeevanism/visionopus/internal/config"
	"github.com/jeevanism/visionopus/internal/platform/httpserver"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	authentication, err := auth.NewService(database, auth.ServiceConfig{
		CSRFKey: cfg.CSRFKey, SessionIdleTimeout: cfg.SessionIdleTimeout,
		SessionAbsoluteTimeout: cfg.SessionAbsoluteTimeout,
		LoginFailureLimit:      cfg.LoginFailureLimit, SoftLockDuration: cfg.SoftLockDuration,
	})
	if err != nil {
		return err
	}
	authenticationHTTP := authhttp.NewHandler(authentication, cfg.CookieSecure, cfg.SessionAbsoluteTimeout)
	server := httpserver.New(cfg.HTTPAddr, logger, database, authenticationHTTP)
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("api listening", "address", cfg.HTTPAddr, "environment", cfg.Environment)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
