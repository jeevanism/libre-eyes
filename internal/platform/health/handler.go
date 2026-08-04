// Package health exposes process liveness and dependency readiness endpoints.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger checks whether a required dependency is available.
type Pinger interface {
	Ping(context.Context) error
}

// Handler serves health endpoints.
type Handler struct {
	database Pinger
}

// NewHandler creates a health handler.
func NewHandler(database Pinger) *Handler {
	return &Handler{database: database}
}

// Register adds health routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health/live", h.live)
	mux.HandleFunc("GET /health/ready", h.ready)
}

func (h *Handler) live(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.database.Ping(ctx); err != nil {
		writeStatus(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	writeStatus(w, http.StatusOK, "ready")
}

func writeStatus(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
	}{Status: value})
}
