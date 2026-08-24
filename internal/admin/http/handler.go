package adminhttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jeevanism/visionopus/internal/admin"
	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/platform/httpx"
)

type Service interface {
	Authorize(context.Context, string, string, auth.RequestMetadata, bool) (admin.Authorization, error)
	Users(context.Context, admin.Authorization) ([]admin.User, error)
	CreateUser(context.Context, admin.Authorization, admin.UserUpsert) (admin.User, error)
	UpdateUser(context.Context, admin.Authorization, admin.UserUpsert) (admin.User, error)
	Contexts(context.Context, admin.Authorization) (admin.Contexts, error)
	Settings(context.Context, admin.Authorization) ([]admin.Setting, error)
	Audit(context.Context, admin.Authorization) ([]admin.AuditEvent, error)
	SetUserActive(context.Context, admin.Authorization, admin.UserCommand) (admin.User, error)
	UpdateSetting(context.Context, admin.Authorization, admin.SettingUpdate) (admin.Setting, error)
	UpsertSite(context.Context, admin.Authorization, admin.ContextUpsert) (admin.Reference, error)
	UpsertFirm(context.Context, admin.Authorization, admin.ContextUpsert) (admin.Reference, error)
}
type Handler struct {
	service      Service
	cookieSecure bool
	logger       *slog.Logger
}

func NewHandler(s Service, secure bool, loggers ...*slog.Logger) *Handler {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &Handler{service: s, cookieSecure: secure, logger: logger}
}
func (h *Handler) Register(m *http.ServeMux) {
	m.HandleFunc("GET /api/v1/admin/users", h.users)
	m.HandleFunc("POST /api/v1/admin/users", h.createUser)
	m.HandleFunc("PATCH /api/v1/admin/users/{userId}", h.updateUser)
	m.HandleFunc("GET /api/v1/admin/contexts", h.contexts)
	m.HandleFunc("GET /api/v1/admin/settings", h.settings)
	m.HandleFunc("GET /api/v1/admin/audit", h.audit)
	m.HandleFunc("POST /api/v1/admin/users/{userId}/deactivate", h.userCommand(false))
	m.HandleFunc("POST /api/v1/admin/users/{userId}/reactivate", h.userCommand(true))
	m.HandleFunc("PATCH /api/v1/admin/settings", h.updateSetting)
	m.HandleFunc("POST /api/v1/admin/sites", h.createSite)
	m.HandleFunc("PATCH /api/v1/admin/sites/{siteId}", h.updateSite)
	m.HandleFunc("POST /api/v1/admin/sites/{siteId}/deactivate", h.deactivateSite)
	m.HandleFunc("POST /api/v1/admin/sites/{siteId}/reactivate", h.reactivateSite)
	m.HandleFunc("POST /api/v1/admin/firms", h.createFirm)
	m.HandleFunc("PATCH /api/v1/admin/firms/{firmId}", h.updateFirm)
	m.HandleFunc("POST /api/v1/admin/firms/{firmId}/deactivate", h.deactivateFirm)
	m.HandleFunc("POST /api/v1/admin/firms/{firmId}/reactivate", h.reactivateFirm)
}

type userBody struct {
	Username        string  `json:"username"`
	DisplayName     string  `json:"displayName"`
	Password        string  `json:"password"`
	Role            string  `json:"role"`
	SiteIDs         []int64 `json:"siteIds"`
	FirmIDs         []int64 `json:"firmIds"`
	ExpectedVersion int64   `json:"expectedVersion"`
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	a, ok := h.auth(w, r, true)
	if !ok {
		return
	}
	var b userBody
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&b) != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	u, err := h.service.CreateUser(r.Context(), a, admin.UserUpsert{Username: b.Username, DisplayName: b.DisplayName, Password: b.Password, Role: b.Role, SiteIDs: b.SiteIDs, FirmIDs: b.FirmIDs})
	if err == admin.ErrConflict {
		h.writeProblem(w, r, http.StatusConflict, "The user could not be created because another change won the update.", "conflict", err)
		return
	}
	if err != nil {
		h.writeProblem(w, r, adminStatus(err), adminMessage(err), adminCode(err), err)
		return
	}
	writeJSON(w, u)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	a, ok := h.auth(w, r, true)
	if !ok {
		return
	}
	var b userBody
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&b) != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	u, err := h.service.UpdateUser(r.Context(), a, admin.UserUpsert{PublicID: r.PathValue("userId"), Username: b.Username, DisplayName: b.DisplayName, Password: b.Password, Role: b.Role, SiteIDs: b.SiteIDs, FirmIDs: b.FirmIDs, ExpectedVersion: b.ExpectedVersion})
	if err == admin.ErrConflict {
		h.writeProblem(w, r, http.StatusConflict, "The user was changed by someone else. Reload the list and try again.", "conflict", err)
		return
	}
	if err != nil {
		h.writeProblem(w, r, adminStatus(err), adminMessage(err), adminCode(err), err)
		return
	}
	writeJSON(w, u)
}

func (h *Handler) writeProblem(w http.ResponseWriter, r *http.Request, status int, title, code string, err error) {
	h.logger.ErrorContext(r.Context(), "admin operation failed", "operation", r.Method+" "+r.URL.Path, "status", status, "code", code, "error", err, "correlation_id", httpx.CorrelationID(r.Context()))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": title, "status": status, "code": code, "correlationId": httpx.CorrelationID(r.Context())})
}

func adminStatus(err error) int {
	if errors.Is(err, admin.ErrInvalidRequest) {
		return http.StatusBadRequest
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
func adminCode(err error) string {
	if errors.Is(err, admin.ErrInvalidRequest) {
		return "invalid_request"
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return "duplicate"
	}
	return "admin_failure"
}
func adminMessage(err error) string {
	if errors.Is(err, admin.ErrInvalidRequest) {
		return "Review the submitted administration values and try again."
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if strings.Contains(pgErr.ConstraintName, "site") || strings.Contains(pgErr.ConstraintName, "firm") {
			return "That site or firm name is already in use in this institution. Choose another name."
		}
		return "That username is already in use. Choose another username."
	}
	return "The administration change could not be completed. Use the correlation ID when reporting this problem."
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

type contextBody struct {
	Name            string `json:"name"`
	Active          bool   `json:"active"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

func (h *Handler) createSite(w http.ResponseWriter, r *http.Request) {
	h.contextMutation(w, r, false, 0, false)
}
func (h *Handler) updateSite(w http.ResponseWriter, r *http.Request) {
	h.contextMutation(w, r, false, 0, true)
}
func (h *Handler) deactivateSite(w http.ResponseWriter, r *http.Request) {
	h.contextMutation(w, r, false, 0, false)
}
func (h *Handler) reactivateSite(w http.ResponseWriter, r *http.Request) {
	h.contextMutation(w, r, false, 0, false)
}
func (h *Handler) createFirm(w http.ResponseWriter, r *http.Request) {
	h.contextMutation(w, r, true, 0, false)
}
func (h *Handler) updateFirm(w http.ResponseWriter, r *http.Request) {
	h.contextMutation(w, r, true, 0, true)
}
func (h *Handler) deactivateFirm(w http.ResponseWriter, r *http.Request) {
	h.contextMutation(w, r, true, 0, false)
}
func (h *Handler) reactivateFirm(w http.ResponseWriter, r *http.Request) {
	h.contextMutation(w, r, true, 0, false)
}

func (h *Handler) contextMutation(w http.ResponseWriter, r *http.Request, firm bool, ignoredID int64, update bool) {
	a, ok := h.auth(w, r, true)
	if !ok {
		return
	}
	var b contextBody
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&b) != nil {
		h.writeProblem(w, r, http.StatusBadRequest, "Review the context name and version.", "invalid_request", admin.ErrInvalidRequest)
		return
	}
	id := ignoredID
	param := "siteId"
	if firm {
		param = "firmId"
	}
	if update || r.PathValue(param) != "" {
		parsed, err := strconv.ParseInt(r.PathValue(param), 10, 64)
		if err != nil || parsed < 1 {
			h.writeProblem(w, r, http.StatusBadRequest, "The context identifier is invalid.", "invalid_request", admin.ErrInvalidRequest)
			return
		}
		id = parsed
	}
	if strings.HasSuffix(r.URL.Path, "/deactivate") {
		b.Active = false
	}
	if strings.HasSuffix(r.URL.Path, "/reactivate") {
		b.Active = true
	}
	in := admin.ContextUpsert{ID: id, Name: b.Name, Active: b.Active, ExpectedVersion: b.ExpectedVersion}
	var value admin.Reference
	var err error
	if firm {
		value, err = h.service.UpsertFirm(r.Context(), a, in)
	} else {
		value, err = h.service.UpsertSite(r.Context(), a, in)
	}
	if err == admin.ErrConflict {
		h.writeProblem(w, r, http.StatusConflict, "The context was changed by someone else. Reload and try again.", "conflict", err)
		return
	}
	if err != nil {
		h.writeProblem(w, r, adminStatus(err), adminMessage(err), adminCode(err), err)
		return
	}
	writeJSON(w, value)
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
	meta := auth.RequestMetadata{CorrelationID: httpx.CorrelationID(r.Context()), SourceIPClass: "local"}
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
