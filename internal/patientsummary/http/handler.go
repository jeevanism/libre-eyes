package patientsummaryhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/patientsummary"
)

const sessionCookieName = "visionopus_session"

type Service interface {
	Authorize(context.Context, string, string, auth.RequestMetadata) (patientsummary.Authorization, error)
	GetHeader(context.Context, patientsummary.Authorization, string, string) (patientsummary.Header, error)
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
	default:
		writeProblem(w, http.StatusServiceUnavailable, "summary_unavailable", "Patient summary is unavailable")
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
		DateOfBirth: header.DateOfBirth.Format("2006-01-02"), Gender: header.Gender,
		Deceased: header.Deceased, ClinicalState: "withheld", Version: header.Version,
		WarningVersion: header.WarningVersion,
	}
	if header.DateOfDeath != nil {
		value := header.DateOfDeath.Format("2006-01-02")
		response.DateOfDeath = &value
		age := header.DateOfDeath.Year() - header.DateOfBirth.Year()
		if header.DateOfDeath.YearDay() < header.DateOfBirth.YearDay() {
			age--
		}
		if age >= 0 {
			response.AgeYears = &age
		}
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
