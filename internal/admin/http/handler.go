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
	SetUserActive(context.Context, admin.Authorization, admin.UserCommand) (admin.User, error)
	UpdateSetting(context.Context, admin.Authorization, admin.SettingUpdate) (admin.Setting, error)
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
	m.HandleFunc("POST /api/v1/admin/users/{userId}/deactivate", h.userCommand(false))
	m.HandleFunc("POST /api/v1/admin/users/{userId}/reactivate", h.userCommand(true))
	m.HandleFunc("PATCH /api/v1/admin/settings", h.updateSetting)
}

func (h *Handler) userCommand(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, ok := h.auth(w, r, true)
		if !ok {
			return
		}
		var b struct {
			ExpectedVersion int64 `json:"expectedVersion"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&b) != nil {
			http.Error(w, "invalid request", 400)
			return
		}
		u, err := h.service.SetUserActive(r.Context(), a, admin.UserCommand{PublicID: r.PathValue("userId"), ExpectedVersion: b.ExpectedVersion, Active: active})
		if err == admin.ErrConflict {
			http.Error(w, "conflict", 409)
			return
		}
		if err != nil {
			http.Error(w, "request failed", 400)
			return
		}
		writeJSON(w, u)
	}
}
func (h *Handler) updateSetting(w http.ResponseWriter, r *http.Request) {
	a, ok := h.auth(w, r, true)
	if !ok {
		return
	}
	var b struct {
		Key             string `json:"key"`
		Value           string `json:"value"`
		ExpectedVersion int64  `json:"expectedVersion"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&b) != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	v, err := h.service.UpdateSetting(r.Context(), a, admin.SettingUpdate{Key: b.Key, Value: b.Value, ExpectedVersion: b.ExpectedVersion})
	if err == admin.ErrConflict {
		http.Error(w, "conflict", 409)
		return
	}
	if err != nil {
		http.Error(w, "request failed", 400)
		return
	}
	writeJSON(w, v)
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
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
