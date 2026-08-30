package patientsummaryhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jeevanism/libre-eyes/internal/auth"
	"github.com/jeevanism/libre-eyes/internal/patientsummary"
)

const sessionCookieName = "libreeyes_session"

type Service interface {
	Authorize(context.Context, string, string, auth.RequestMetadata) (patientsummary.Authorization, error)
	AuthorizeClinical(context.Context, string, string, auth.RequestMetadata) (patientsummary.Authorization, error)
	GetHeader(context.Context, patientsummary.Authorization, string, string) (patientsummary.Header, error)
	GetWarningDetails(context.Context, patientsummary.Authorization, string, string) (patientsummary.WarningDetails, error)
	CreateBreakGlassGrant(context.Context, string, string, auth.RequestMetadata, string, string, patientsummary.BreakGlassRequest) (patientsummary.BreakGlassGrant, error)
	RevokeBreakGlassGrant(context.Context, string, string, auth.RequestMetadata, string, string, string) error
}

type Handler struct {
	service      Service
	cookieSecure bool
}

func NewHandler(service Service, cookieSecure bool) *Handler {
	return &Handler{service: service, cookieSecure: cookieSecure}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/patients/{patientId}/summary-header", h.getHeader)
	mux.HandleFunc("GET /api/v1/patients/{patientId}/summary-header/warnings", h.getWarnings)
	mux.HandleFunc("POST /api/v1/patients/{patientId}/break-glass-grants", h.createGrant)
	mux.HandleFunc("DELETE /api/v1/patients/{patientId}/break-glass-grants/{grantId}", h.revokeGrant)
}

func (h *Handler) createGrant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	var body struct {
		ReasonCode   string  `json:"reasonCode"`
		ReasonDetail *string `json:"reasonDetail"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2048))
	if err := decoder.Decode(&body); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "Invalid break-glass request")
		return
	}
	version := r.Header.Get("X-Context-Version")
	if version == "" {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "A context version is required")
		return
	}
	grant, err := h.service.CreateBreakGlassGrant(r.Context(), cookie.Value, r.Header.Get("X-CSRF-Token"), auth.RequestMetadata{CorrelationID: correlationID(r), SourceIPClass: "private"}, r.PathValue("patientId"), version, patientsummary.BreakGlassRequest{ReasonCode: body.ReasonCode, ReasonDetail: body.ReasonDetail})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, grant)
}

func (h *Handler) revokeGrant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	version := r.Header.Get("X-Context-Version")
	if version == "" {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "A context version is required")
		return
	}
	err = h.service.RevokeBreakGlassGrant(r.Context(), cookie.Value, r.Header.Get("X-CSRF-Token"), auth.RequestMetadata{CorrelationID: correlationID(r), SourceIPClass: "private"}, r.PathValue("patientId"), r.PathValue("grantId"), version)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getWarnings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	version := r.Header.Get("X-Context-Version")
	if version == "" {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "A context version is required")
		return
	}
	authorization, err := h.service.AuthorizeClinical(ctx, cookie.Value, r.Header.Get("X-CSRF-Token"), auth.RequestMetadata{CorrelationID: correlationID(r), SourceIPClass: "private"})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	details, err := h.service.GetWarningDetails(ctx, authorization, r.PathValue("patientId"), version)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapWarningDetails(details))
}

func (h *Handler) getHeader(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	version := r.Header.Get("X-Context-Version")
	if version == "" {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "A context version is required")
		return
	}
	authorization, err := h.service.Authorize(ctx, cookie.Value, r.Header.Get("X-CSRF-Token"), auth.RequestMetadata{CorrelationID: correlationID(r), SourceIPClass: "private"})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	header, err := h.service.GetHeader(ctx, authorization, r.PathValue("patientId"), version)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapHeader(header))
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrUnauthenticated):
		writeProblem(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
	case errors.Is(err, auth.ErrCSRF), errors.Is(err, auth.ErrForbidden):
		writeProblem(w, http.StatusNotFound, "patient_not_found", "Patient not found")
	case errors.Is(err, auth.ErrConflict):
		writeProblem(w, http.StatusConflict, "context_changed", "Clinical context changed")
	case errors.Is(err, patientsummary.ErrNotFound):
		writeProblem(w, http.StatusNotFound, "patient_not_found", "Patient not found")
	case errors.Is(err, patientsummary.ErrWarningOverflow):
		writeProblem(w, http.StatusServiceUnavailable, "summary_unavailable", "Patient summary is unavailable")
	default:
		writeProblem(w, http.StatusServiceUnavailable, "summary_unavailable", "Patient summary is unavailable")
	}
}

func mapWarningDetails(details patientsummary.WarningDetails) map[string]any {
	return map[string]any{
		"patientId":      details.PatientID,
		"allergies":      map[string]any{"status": details.AllergyStatus},
		"alerts":         map[string]any{"status": details.AlertStatus},
		"items":          details.Items,
		"complete":       details.Complete,
		"warningVersion": details.WarningVersion,
	}
}

type headerResponse struct {
	PatientID      string  `json:"patientId"`
	GivenName      *string `json:"givenName"`
	FamilyName     *string `json:"familyName"`
	DateOfBirth    string  `json:"dateOfBirth"`
	AgeYears       *int    `json:"ageYears"`
	Gender         string  `json:"gender"`
	Deceased       bool    `json:"deceased"`
	DateOfDeath    *string `json:"dateOfDeath"`
	ClinicalState  string  `json:"clinicalDisclosure"`
	AllergyStatus  *string `json:"allergyStatus,omitempty"`
	AlertStatus    *string `json:"alertStatus,omitempty"`
	Version        int64   `json:"patientVersion"`
	WarningVersion *int64  `json:"warningVersion,omitempty"`
}

func mapHeader(header patientsummary.Header) headerResponse {
	response := headerResponse{
		PatientID: header.PublicID, GivenName: header.GivenName, FamilyName: header.FamilyName,
		DateOfBirth: header.DateOfBirth.Format("2006-01-02"), AgeYears: header.AgeYears, Gender: header.Gender,
		Deceased: header.Deceased, ClinicalState: "withheld", Version: header.Version,
		WarningVersion: header.WarningVersion,
	}
	if header.DateOfDeath != nil {
		value := header.DateOfDeath.Format("2006-01-02")
		response.DateOfDeath = &value
	}
	return response
}

func correlationID(r *http.Request) string {
	if value := r.Header.Get("X-Request-ID"); value != "" {
		return value
	}
	return "summary-request"
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, map[string]any{"type": "about:blank", "title": "Request failed", "status": status, "code": code, "detail": detail, "correlationId": "summary-request"})
}
