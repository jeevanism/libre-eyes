// Package worklisthttp exposes the development-only clinic-flow API.
package worklisthttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/jeevanism/libre-eyes/internal/auth"
	"github.com/jeevanism/libre-eyes/internal/platform/httpx"
	"github.com/jeevanism/libre-eyes/internal/worklist"
)

const (
	sessionCookieName = "libreeyes_session"
	maximumBodyBytes  = 2048
	requestTimeout    = 5 * time.Second
)

type Service interface {
	Authorize(context.Context, string, string, auth.RequestMetadata) (worklist.Authorization, error)
	List(context.Context, worklist.Authorization) ([]worklist.Ticket, error)
	Command(context.Context, worklist.Authorization, worklist.CommandRequest) (worklist.Ticket, error)
}

type Handler struct {
	service      Service
	cookieSecure bool
}

func NewHandler(service Service, cookieSecure bool) *Handler {
	return &Handler{service: service, cookieSecure: cookieSecure}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/development/clinic-flow/tickets", h.list)
	mux.HandleFunc("POST /api/v1/development/clinic-flow/tickets/{ticketId}/arrive", h.command(worklist.CommandArrive))
	mux.HandleFunc("POST /api/v1/development/clinic-flow/tickets/{ticketId}/claim", h.command(worklist.CommandClaim))
	mux.HandleFunc("POST /api/v1/development/clinic-flow/tickets/{ticketId}/release", h.command(worklist.CommandRelease))
	mux.HandleFunc("POST /api/v1/development/clinic-flow/tickets/{ticketId}/complete", h.command(worklist.CommandComplete))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx)
		if !ok {
			return
		}
		items, err := h.service.List(ctx, authorization)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		response := make([]ticketResponse, len(items))
		for index, ticket := range items {
			response[index] = mapTicket(ticket)
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": response})
	})
}

func (h *Handler) command(command worklist.Command) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.withTimeout(w, r, func(ctx context.Context) {
			authorization, ok := h.authorize(w, r, ctx)
			if !ok {
				return
			}
			var body struct {
				ExpectedVersion int64 `json:"expectedVersion"`
			}
			if err := decodeJSON(w, r, &body); err != nil {
				h.writeError(w, r, err)
				return
			}
			ticket, err := h.service.Command(ctx, authorization, worklist.CommandRequest{
				TicketID: r.PathValue("ticketId"), ExpectedVersion: body.ExpectedVersion, Command: command,
			})
			if err != nil {
				h.writeError(w, r, err)
				return
			}
			writeJSON(w, http.StatusOK, mapTicket(ticket))
		})
	}
}

func (h *Handler) withTimeout(w http.ResponseWriter, r *http.Request, operation func(context.Context)) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	operation(ctx)
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, ctx context.Context) (worklist.Authorization, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		h.writeError(w, r, auth.ErrUnauthenticated)
		return worklist.Authorization{}, false
	}
	authorization, err := h.service.Authorize(ctx, cookie.Value, r.Header.Get("X-CSRF-Token"), requestMetadata(r))
	if err != nil {
		h.writeError(w, r, err)
		return worklist.Authorization{}, false
	}
	return authorization, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maximumBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: %w", worklist.ErrInvalidRequest, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return worklist.ErrInvalidRequest
	}
	return nil
}

func requestMetadata(r *http.Request) auth.RequestMetadata {
	return auth.RequestMetadata{CorrelationID: httpx.CorrelationID(r.Context()), SourceIPClass: classifyRemoteIP(r.RemoteAddr)}
}

func classifyRemoteIP(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return "unknown"
	}
	address = address.Unmap()
	switch {
	case address.IsLoopback():
		return "loopback"
	case address.IsPrivate():
		return "private"
	default:
		return "public"
	}
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		writeProblem(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large")
	case errors.Is(err, auth.ErrUnauthenticated):
		h.expireSessionCookie(w)
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required")
	case errors.Is(err, auth.ErrCSRF), errors.Is(err, auth.ErrForbidden), errors.Is(err, worklist.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "ticket_not_found", "Ticket not found")
	case errors.Is(err, auth.ErrConflict), errors.Is(err, worklist.ErrConflict), errors.Is(err, worklist.ErrInvalidTransition):
		writeProblem(w, r, http.StatusConflict, "conflict", "Ticket changed; refresh and try again")
	case errors.Is(err, worklist.ErrInvalidRequest):
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid development clinic-flow request")
	case errors.Is(err, context.DeadlineExceeded):
		writeProblem(w, r, http.StatusGatewayTimeout, "clinic_flow_timeout", "Development clinic-flow request timed out")
	default:
		writeProblem(w, r, http.StatusServiceUnavailable, "clinic_flow_unavailable", "Development clinic-flow is unavailable")
	}
}

func (h *Handler) expireSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(1, 0).UTC(), HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode})
}

type ticketResponse struct {
	ID                    string          `json:"id"`
	SyntheticPatientLabel string          `json:"syntheticPatientLabel"`
	Status                worklist.Status `json:"status"`
	AssigneeDisplayName   *string         `json:"assigneeDisplayName"`
	Version               int64           `json:"version"`
}

func mapTicket(ticket worklist.Ticket) ticketResponse {
	return ticketResponse{ID: ticket.ID, SyntheticPatientLabel: ticket.SyntheticPatientLabel, Status: ticket.Status, AssigneeDisplayName: ticket.AssigneeDisplayName, Version: ticket.Version}
}

type problem struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	Status        int    `json:"status"`
	Code          string `json:"code"`
	CorrelationID string `json:"correlationId"`
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, problem{Type: "about:blank", Title: title, Status: status, Code: code, CorrelationID: httpx.CorrelationID(r.Context())})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
