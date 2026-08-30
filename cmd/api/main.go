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

	"github.com/jeevanism/libre-eyes/internal/admin"
	adminhttp "github.com/jeevanism/libre-eyes/internal/admin/http"
	"github.com/jeevanism/libre-eyes/internal/auth"
	authhttp "github.com/jeevanism/libre-eyes/internal/auth/http"
	"github.com/jeevanism/libre-eyes/internal/branding"
	brandinghttp "github.com/jeevanism/libre-eyes/internal/branding/http"
	"github.com/jeevanism/libre-eyes/internal/config"
	"github.com/jeevanism/libre-eyes/internal/episodes"
	episodeshttp "github.com/jeevanism/libre-eyes/internal/episodes/http"
	"github.com/jeevanism/libre-eyes/internal/examination"
	"github.com/jeevanism/libre-eyes/internal/patientsearch"
	patientsearchhttp "github.com/jeevanism/libre-eyes/internal/patientsearch/http"
	"github.com/jeevanism/libre-eyes/internal/patientsummary"
	patientsummaryhttp "github.com/jeevanism/libre-eyes/internal/patientsummary/http"
	"github.com/jeevanism/libre-eyes/internal/platform/httpserver"
	"github.com/jeevanism/libre-eyes/internal/referralappointment"
	referralappointmenthttp "github.com/jeevanism/libre-eyes/internal/referralappointment/http"
	"github.com/jeevanism/libre-eyes/internal/theatrebooking"
	theatrebookinghttp "github.com/jeevanism/libre-eyes/internal/theatrebooking/http"
	"github.com/jeevanism/libre-eyes/internal/worklist"
	worklisthttp "github.com/jeevanism/libre-eyes/internal/worklist/http"
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
	patientSearch, err := patientsearch.NewService(database, authentication, patientsearch.ServiceConfig{})
	if err != nil {
		return err
	}
	patientSearchHTTP := patientsearchhttp.NewHandler(patientSearch, cfg.CookieSecure)
	patientSummary, err := patientsummary.NewService(database, authentication)
	if err != nil {
		return err
	}
	patientSummaryHTTP := patientsummaryhttp.NewHandler(patientSummary, cfg.CookieSecure)
	draftRegistry, err := examination.NewDraftRegistry(ctx, database, cfg.Environment)
	if err != nil {
		return err
	}
	episodeService, err := episodes.NewServiceWithDraftRegistry(database, authentication, draftRegistry)
	if err != nil {
		return err
	}
	episodesHTTP := episodeshttp.NewHandler(episodeService, cfg.CookieSecure)
	developmentFlow, err := worklist.NewService(database, authentication)
	if err != nil {
		return err
	}
	developmentFlowHTTP := worklisthttp.NewHandler(developmentFlow, cfg.CookieSecure)
	developmentTheatreBooking, err := theatrebooking.NewService(database, authentication)
	if err != nil {
		return err
	}
	developmentTheatreBookingHTTP := theatrebookinghttp.NewHandler(developmentTheatreBooking, cfg.CookieSecure)
	developmentReferral, err := referralappointment.NewService(database, authentication)
	if err != nil {
		return err
	}
	developmentReferralHTTP := referralappointmenthttp.NewHandler(developmentReferral, cfg.CookieSecure)
	adminService, err := admin.NewService(database, authentication)
	if err != nil {
		return err
	}
	adminHTTP := adminhttp.NewHandler(adminService, cfg.CookieSecure, logger)
	brandingService, err := branding.NewService(database, authentication)
	if err != nil {
		return err
	}
	brandingHTTP := brandinghttp.NewHandler(brandingService, logger)
	server := httpserver.New(cfg.HTTPAddr, logger, database, cfg.StaticDir, authenticationHTTP, patientSearchHTTP, patientSummaryHTTP, episodesHTTP, developmentFlowHTTP, developmentTheatreBookingHTTP, developmentReferralHTTP, adminHTTP, brandingHTTP)
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
