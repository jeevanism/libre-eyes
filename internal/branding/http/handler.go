// Package brandinghttp exposes presentation profiles over REST JSON.
package brandinghttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jeevanism/libre-eyes/internal/auth"
	"github.com/jeevanism/libre-eyes/internal/branding"
	"github.com/jeevanism/libre-eyes/internal/platform/httpx"
)

const maximumBodyBytes = 8 * 1024

// Service is the branding use-case boundary used by the HTTP adapter.
type Service interface {
	PublicProfile(context.Context, int64) (branding.Profile, error)
	Authorize(context.Context, string, string, auth.RequestMetadata, bool) (branding.Authorization, error)
	State(context.Context, branding.Authorization) (branding.State, error)
	SaveDraft(context.Context, branding.Authorization, branding.DraftInput) (branding.State, error)
	Publish(context.Context, branding.Authorization, branding.VersionCommand) (branding.State, error)
	Rollback(context.Context, branding.Authorization, branding.VersionCommand) (branding.State, error)
}

// Handler serves public presentation reads and protected branding commands.
type Handler struct {
	service Service
	logger  *slog.Logger
}

// NewHandler constructs the branding HTTP adapter.
func NewHandler(service Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{service: service, logger: logger}
}

// Register adds branding routes to the application mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/presentation/branding", h.publicProfile)
	mux.HandleFunc("GET /api/v1/admin/branding", h.state)
	mux.HandleFunc("PUT /api/v1/admin/branding/draft", h.saveDraft)
	mux.HandleFunc("POST /api/v1/admin/branding/publish", h.publish)
	mux.HandleFunc("POST /api/v1/admin/branding/rollback", h.rollback)
}

func (h *Handler) publicProfile(w http.ResponseWriter, r *http.Request) {
	var institutionID int64
	if raw := r.URL.Query().Get("institutionId"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 1 {
			h.writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Choose a valid institution before loading its presentation profile.", branding.ErrInvalidRequest)
			return
		}
		institutionID = parsed
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	profile, err := h.service.PublicProfile(ctx, institutionID)
	if err != nil {
		h.writeProblem(w, r, http.StatusInternalServerError, "branding_unavailable", "The presentation profile is temporarily unavailable. LibreEyes defaults remain available.", err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, profile)
}

func (h *Handler) state(w http.ResponseWriter, r *http.Request) {
	authorization, ok := h.authorize(w, r, false)
	if !ok {
		return
	}
	state, err := h.service.State(r.Context(), authorization)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, state)
}

func (h *Handler) saveDraft(w http.ResponseWriter, r *http.Request) {
	authorization, ok := h.authorize(w, r, true)
	if !ok {
		return
	}
	var input branding.DraftInput
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Review the branding values and try again.", branding.ErrInvalidRequest)
		return
	}
	state, err := h.service.SaveDraft(r.Context(), authorization, input)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	h.logger.InfoContext(r.Context(), "branding draft saved", "institution_id", state.Effective.InstitutionID, "correlation_id", httpx.CorrelationID(r.Context()))
	writeJSON(w, state)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.service.Publish, "branding profile published")
}

func (h *Handler) rollback(w http.ResponseWriter, r *http.Request) {
	h.command(w, r, h.service.Rollback, "branding profile rolled back")
}

func (h *Handler) command(w http.ResponseWriter, r *http.Request, command func(context.Context, branding.Authorization, branding.VersionCommand) (branding.State, error), message string) {
	authorization, ok := h.authorize(w, r, true)
	if !ok {
		return
	}
	var input branding.VersionCommand
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeProblem(w, r, http.StatusBadRequest, "invalid_request", "A valid branding version is required.", branding.ErrInvalidRequest)
		return
	}
	state, err := command(r.Context(), authorization, input)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	h.logger.InfoContext(r.Context(), message, "institution_id", state.Effective.InstitutionID, "profile_version", state.Effective.ProfileVersion, "correlation_id", httpx.CorrelationID(r.Context()))
	writeJSON(w, state)
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, write bool) (branding.Authorization, bool) {
	cookie, err := r.Cookie("libreeyes_session")
	if err != nil {
		h.writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Sign in before opening branding administration.", err)
		return branding.Authorization{}, false
	}
	metadata := auth.RequestMetadata{CorrelationID: httpx.CorrelationID(r.Context()), SourceIPClass: "local"}
	if metadata.CorrelationID == "" {
		metadata.CorrelationID = "branding-admin"
	}
	authorization, err := h.service.Authorize(r.Context(), cookie.Value, r.Header.Get("X-CSRF-Token"), metadata, write)
	if err != nil {
		h.writeProblem(w, r, http.StatusForbidden, "forbidden", "Your current role cannot manage presentation profiles.", err)
		return branding.Authorization{}, false
	}
	return authorization, true
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var contrastError *branding.ContrastValidationError
	switch {
	case errors.As(err, &contrastError):
		h.writeProblemWithFields(
			w,
			r,
			http.StatusBadRequest,
			"branding_contrast_invalid",
			contrastProblemTitle(contrastError.Issues),
			err,
			contrastError.Issues,
		)
	case errors.Is(err, branding.ErrInvalidRequest):
		h.writeProblem(w, r, http.StatusBadRequest, "invalid_branding", "Use valid names and accessible six-digit colour values. Action colours must remain readable with white text.", err)
	case errors.Is(err, branding.ErrConflict):
		h.writeProblem(w, r, http.StatusConflict, "branding_conflict", "The branding profile was changed by someone else. Reload it before trying again.", err)
	case errors.Is(err, branding.ErrNoDraft):
		h.writeProblem(w, r, http.StatusConflict, "branding_draft_missing", "Save a branding draft before publishing it.", err)
	case errors.Is(err, branding.ErrNotFound):
		h.writeProblem(w, r, http.StatusNotFound, "branding_version_missing", "The selected branding version is no longer available for rollback.", err)
	case errors.Is(err, branding.ErrForbidden):
		h.writeProblem(w, r, http.StatusForbidden, "forbidden", "Your current role cannot manage presentation profiles.", err)
	default:
		h.writeProblem(w, r, http.StatusInternalServerError, "branding_failure", "The branding change could not be completed. Use the correlation ID when reporting this problem.", err)
	}
}

func (h *Handler) writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title string, err error) {
	h.writeProblemWithFields(w, r, status, code, title, err, nil)
}

func (h *Handler) writeProblemWithFields(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	code string,
	title string,
	err error,
	fieldErrors []branding.ContrastIssue,
) {
	h.logger.ErrorContext(
		r.Context(),
		"branding operation failed",
		"operation", r.Method+" "+r.URL.Path,
		"status", status,
		"code", code,
		"error", err,
		"correlation_id", httpx.CorrelationID(r.Context()),
	)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	problem := map[string]any{
		"type": "about:blank", "title": title, "status": status, "code": code,
		"correlationId": httpx.CorrelationID(r.Context()),
	}
	if len(fieldErrors) > 0 {
		problem["fieldErrors"] = fieldErrors
	}
	_ = json.NewEncoder(w).Encode(problem)
}

func contrastProblemTitle(issues []branding.ContrastIssue) string {
	labels := make([]string, 0, len(issues))
	for _, issue := range issues {
		labels = append(labels, issue.Label)
	}
	if len(labels) == 1 {
		return labels[0] + " colour does not meet its accessibility contrast requirement."
	}
	if len(labels) > 1 {
		prefix := strings.Join(labels[:len(labels)-1], ", ")
		return prefix + " and " + labels[len(labels)-1] +
			" colours do not meet their accessibility contrast requirements."
	}
	return "The branding colours do not meet accessibility contrast requirements."
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maximumBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body contains multiple JSON values")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
