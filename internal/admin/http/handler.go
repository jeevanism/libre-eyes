package adminhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jeevanism/visionopus/internal/admin"
	"github.com/jeevanism/visionopus/internal/auth"
)

type Service interface {
	Authorize(context.Context, string, string, auth.RequestMetadata, bool) (admin.Authorization, error)
	Users(context.Context, admin.Authorization) ([]admin.User, error)
	Contexts(context.Context, admin.Authorization) (admin.Contexts, error)
	Settings(context.Context, admin.Authorization) ([]admin.Setting, error)
	Audit(context.Context, admin.Authorization) ([]admin.AuditEvent, error)
}
type Handler struct {
	service      Service
	cookieSecure bool
}

func NewHandler(s Service, secure bool) *Handler { return &Handler{service: s, cookieSecure: secure} }
func (h *Handler) Register(m *http.ServeMux) {
	m.HandleFunc("GET /api/v1/admin/users", h.users)
	m.HandleFunc("GET /api/v1/admin/contexts", h.contexts)
	m.HandleFunc("GET /api/v1/admin/settings", h.settings)
	m.HandleFunc("GET /api/v1/admin/audit", h.audit)
}
func (h *Handler) auth(w http.ResponseWriter, r *http.Request, write bool) (admin.Authorization, bool) {
	c, err := r.Cookie("visionopus_session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return admin.Authorization{}, false
	}
	meta := auth.RequestMetadata{CorrelationID: r.Header.Get("X-Correlation-ID"), SourceIPClass: "local"}
	if meta.CorrelationID == "" {
		meta.CorrelationID = "admin-demo"
	}
	a, err := h.service.Authorize(r.Context(), c.Value, r.Header.Get("X-CSRF-Token"), meta, write)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return admin.Authorization{}, false
	}
	return a, true
}
func (h *Handler) users(w http.ResponseWriter, r *http.Request) {
	h.read(w, r, func(ctx context.Context, a admin.Authorization) (any, error) { return h.service.Users(ctx, a) })
}
func (h *Handler) contexts(w http.ResponseWriter, r *http.Request) {
	h.read(w, r, func(ctx context.Context, a admin.Authorization) (any, error) { return h.service.Contexts(ctx, a) })
}
func (h *Handler) settings(w http.ResponseWriter, r *http.Request) {
	h.read(w, r, func(ctx context.Context, a admin.Authorization) (any, error) { return h.service.Settings(ctx, a) })
}
func (h *Handler) audit(w http.ResponseWriter, r *http.Request) {
	h.read(w, r, func(ctx context.Context, a admin.Authorization) (any, error) { return h.service.Audit(ctx, a) })
}
func (h *Handler) read(w http.ResponseWriter, r *http.Request, fn func(context.Context, admin.Authorization) (any, error)) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	a, ok := h.auth(w, r, false)
	if !ok {
		return
	}
	v, err := fn(ctx, a)
	if err != nil {
		http.Error(w, "request failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
