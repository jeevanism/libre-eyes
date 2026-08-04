// Package httpserver constructs the VisionOpus HTTP server.
package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	authhttp "github.com/jeevanism/visionopus/internal/auth/http"
	"github.com/jeevanism/visionopus/internal/platform/health"
	"github.com/jeevanism/visionopus/internal/platform/httpx"
)

// New creates the HTTP server and registers foundation routes.
func New(addr string, logger *slog.Logger, database health.Pinger, authentication *authhttp.Handler) *http.Server {
	mux := http.NewServeMux()
	health.NewHandler(database).Register(mux)
	authentication.Register(mux)

	return &http.Server{
		Addr:              addr,
		Handler:           httpx.Middleware(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
