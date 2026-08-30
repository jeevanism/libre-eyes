package referralappointmenthttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jeevanism/libre-eyes/internal/auth"
	"github.com/jeevanism/libre-eyes/internal/platform/httpx"
	"github.com/jeevanism/libre-eyes/internal/referralappointment"
	"io"
	"net/http"
	"time"
)

const cookieName = "libreeyes_session"

type Service interface {
	Authorize(context.Context, string, string, auth.RequestMetadata) (referralappointment.Authorization, error)
	List(context.Context, referralappointment.Authorization) ([]referralappointment.Request, error)
	Create(context.Context, referralappointment.Authorization, referralappointment.CreateRequest) (referralappointment.Request, error)
	Command(context.Context, referralappointment.Authorization, referralappointment.CommandRequest) (referralappointment.Request, error)
}
type Handler struct {
	service      Service
	cookieSecure bool
}

func NewHandler(s Service, secure bool) *Handler { return &Handler{service: s, cookieSecure: secure} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/development/referral-appointments/requests", h.list)
	mux.HandleFunc("POST /api/v1/development/referral-appointments/requests", h.create)
	for _, c := range []referralappointment.Command{referralappointment.CommandSchedule, referralappointment.CommandArrive, referralappointment.CommandComplete, referralappointment.CommandAbandon} {
		mux.HandleFunc("POST /api/v1/development/referral-appointments/requests/{requestId}/"+string(c), h.command(c))
	}
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func(ctx context.Context, a referralappointment.Authorization) {
		items, e := h.service.List(ctx, a)
		if e != nil {
			h.err(w, r, e)
			return
		}
		out := make([]response, len(items))
		for i := range items {
			out[i] = mapRequest(items[i])
		}
		jsonResp(w, 200, map[string]any{"items": out})
	})
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func(ctx context.Context, a referralappointment.Authorization) {
		var body referralappointment.CreateRequest
		if e := decode(w, r, &body); e != nil {
			h.err(w, r, e)
			return
		}
		item, e := h.service.Create(ctx, a, body)
		if e != nil {
			h.err(w, r, e)
			return
		}
		jsonResp(w, 201, mapRequest(item))
	})
}
func (h *Handler) command(c referralappointment.Command) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.run(w, r, func(ctx context.Context, a referralappointment.Authorization) {
			var body struct {
				ExpectedVersion int64 `json:"expectedVersion"`
			}
			if e := decode(w, r, &body); e != nil {
				h.err(w, r, e)
				return
			}
			item, e := h.service.Command(ctx, a, referralappointment.CommandRequest{RequestID: r.PathValue("requestId"), ExpectedVersion: body.ExpectedVersion, Command: c})
			if e != nil {
				h.err(w, r, e)
				return
			}
			jsonResp(w, 200, mapRequest(item))
		})
	}
}
func (h *Handler) run(w http.ResponseWriter, r *http.Request, fn func(context.Context, referralappointment.Authorization)) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cookie, e := r.Cookie(cookieName)
	if e != nil {
		h.err(w, r, auth.ErrUnauthenticated)
		return
	}
	a, e := h.service.Authorize(ctx, cookie.Value, r.Header.Get("X-CSRF-Token"), auth.RequestMetadata{CorrelationID: httpx.CorrelationID(r.Context())})
	if e != nil {
		h.err(w, r, e)
		return
	}
	fn(ctx, a)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	if e := d.Decode(&struct{}{}); !errors.Is(e, io.EOF) {
		return fmt.Errorf("invalid request")
	}
	return nil
}

type response struct {
	ID                    string                     `json:"id"`
	SyntheticPatientLabel string                     `json:"syntheticPatientLabel"`
	RecipientRole         string                     `json:"recipientRole"`
	ClinicCode            string                     `json:"clinicCode"`
	AppointmentDate       string                     `json:"appointmentDate"`
	AppointmentTime       string                     `json:"appointmentTime"`
	Priority              string                     `json:"priority"`
	Notes                 string                     `json:"notes"`
	Status                referralappointment.Status `json:"status"`
	Version               int64                      `json:"version"`
}

func mapRequest(r referralappointment.Request) response {
	return response{ID: r.ID, SyntheticPatientLabel: r.SyntheticPatientLabel, RecipientRole: r.RecipientRole, ClinicCode: r.ClinicCode, AppointmentDate: r.AppointmentDate, AppointmentTime: r.AppointmentTime, Priority: r.Priority, Notes: r.Notes, Status: r.Status, Version: r.Version}
}
func jsonResp(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (h *Handler) err(w http.ResponseWriter, r *http.Request, e error) {
	code, status, title := "referral_unavailable", 503, "Development referral service unavailable"
	switch {
	case errors.Is(e, auth.ErrUnauthenticated):
		code, status, title = "unauthenticated", 401, "Authentication required"
	case errors.Is(e, auth.ErrCSRF), errors.Is(e, auth.ErrForbidden), errors.Is(e, referralappointment.ErrNotFound):
		code, status, title = "not_found", 404, "Referral appointment not found"
	case errors.Is(e, referralappointment.ErrConflict), errors.Is(e, referralappointment.ErrInvalidTransition):
		code, status, title = "conflict", 409, "Referral appointment changed; refresh and try again"
	case errors.Is(e, referralappointment.ErrInvalidRequest):
		code, status, title = "invalid_request", 400, "Invalid development referral appointment request"
	}
	jsonResp(w, status, map[string]any{"type": "about:blank", "title": title, "status": status, "code": code, "correlationId": httpx.CorrelationID(r.Context())})
}
