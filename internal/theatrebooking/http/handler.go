// Package theatrebookinghttp exposes the development-only theatre-booking API.
package theatrebookinghttp

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

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/platform/httpx"
	"github.com/jeevanism/visionopus/internal/theatrebooking"
)

const (
	sessionCookieName = "visionopus_session"
	maximumBodyBytes  = 2048
	requestTimeout    = 5 * time.Second
)

type Service interface {
	AuthorizeRead(context.Context, string, auth.RequestMetadata) (theatrebooking.Authorization, error)
	AuthorizeCommand(context.Context, string, string, auth.RequestMetadata) (theatrebooking.Authorization, error)
	Board(context.Context, theatrebooking.Authorization, string, string) (theatrebooking.Board, error)
	Command(context.Context, theatrebooking.Authorization, theatrebooking.CommandRequest) (theatrebooking.BookingRequest, error)
}

type Handler struct {
	service      Service
	cookieSecure bool
}

func NewHandler(service Service, cookieSecure bool) *Handler {
	return &Handler{service: service, cookieSecure: cookieSecure}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/development/theatre-booking/board", h.board)
	mux.HandleFunc("POST /api/v1/development/theatre-booking/requests/{requestId}/schedule", h.command(theatrebooking.CommandSchedule))
	mux.HandleFunc("POST /api/v1/development/theatre-booking/requests/{requestId}/reschedule", h.command(theatrebooking.CommandReschedule))
	mux.HandleFunc("POST /api/v1/development/theatre-booking/requests/{requestId}/cancel", h.command(theatrebooking.CommandCancel))
}

func (h *Handler) board(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorizeRead(w, r, ctx)
		if !ok {
			return
		}
		board, err := h.service.Board(ctx, authorization, r.URL.Query().Get("sessionDate"), r.URL.Query().Get("theatreRoomId"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, mapBoard(board))
	})
}

func (h *Handler) command(command theatrebooking.Command) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.withTimeout(w, r, func(ctx context.Context) {
			authorization, ok := h.authorizeCommand(w, r, ctx)
			if !ok {
				return
			}
			var body struct {
				ExpectedVersion int64  `json:"expectedVersion"`
				TargetSessionID string `json:"targetSessionId"`
			}
			if err := decodeJSON(w, r, &body); err != nil {
				h.writeError(w, r, err)
				return
			}
			item, err := h.service.Command(ctx, authorization, theatrebooking.CommandRequest{RequestID: r.PathValue("requestId"), ExpectedVersion: body.ExpectedVersion, TargetSessionID: body.TargetSessionID, Command: command})
			if err != nil {
				h.writeError(w, r, err)
				return
			}
			writeJSON(w, http.StatusOK, mapBookingRequest(item))
		})
	}
}

func (h *Handler) withTimeout(w http.ResponseWriter, r *http.Request, operation func(context.Context)) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	operation(ctx)
}
func (h *Handler) authorizeRead(w http.ResponseWriter, r *http.Request, ctx context.Context) (theatrebooking.Authorization, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		h.writeError(w, r, auth.ErrUnauthenticated)
		return theatrebooking.Authorization{}, false
	}
	value, err := h.service.AuthorizeRead(ctx, cookie.Value, requestMetadata(r))
	if err != nil {
		h.writeError(w, r, err)
		return theatrebooking.Authorization{}, false
	}
	return value, true
}
func (h *Handler) authorizeCommand(w http.ResponseWriter, r *http.Request, ctx context.Context) (theatrebooking.Authorization, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		h.writeError(w, r, auth.ErrUnauthenticated)
		return theatrebooking.Authorization{}, false
	}
	value, err := h.service.AuthorizeCommand(ctx, cookie.Value, r.Header.Get("X-CSRF-Token"), requestMetadata(r))
	if err != nil {
		h.writeError(w, r, err)
		return theatrebooking.Authorization{}, false
	}
	return value, true
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maximumBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: %w", theatrebooking.ErrInvalidRequest, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return theatrebooking.ErrInvalidRequest
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
	if address.IsLoopback() {
		return "loopback"
	}
	if address.IsPrivate() {
		return "private"
	}
	return "public"
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	var conflict theatrebooking.ConflictError
	switch {
	case errors.As(err, &tooLarge):
		writeProblem(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large")
	case errors.Is(err, auth.ErrUnauthenticated):
		h.expireSessionCookie(w)
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required")
	case errors.Is(err, auth.ErrCSRF), errors.Is(err, auth.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "The theatre-booking demonstration is unavailable in this context")
	case errors.Is(err, theatrebooking.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "booking_request_not_found", "Booking request not found")
	case errors.As(err, &conflict):
		writeProblem(w, r, http.StatusConflict, conflict.Reason, "The synthetic booking request changed")
	case errors.Is(err, theatrebooking.ErrInvalidRequest):
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid development theatre-booking request")
	case errors.Is(err, context.DeadlineExceeded):
		writeProblem(w, r, http.StatusGatewayTimeout, "theatre_booking_timeout", "Development theatre-booking request timed out")
	default:
		writeProblem(w, r, http.StatusServiceUnavailable, "theatre_booking_unavailable", "Development theatre booking is unavailable")
	}
}
func (h *Handler) expireSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(1, 0).UTC(), HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode})
}

type sessionResponse struct {
	ID                 string `json:"id"`
	SyntheticRoomLabel string `json:"syntheticRoomLabel"`
	StartsAt           string `json:"startsAt"`
	EndsAt             string `json:"endsAt"`
	CapacityMinutes    int    `json:"capacityMinutes"`
	AllocatedMinutes   int    `json:"allocatedMinutes"`
	Version            int64  `json:"version"`
}
type bookingResponse struct {
	ID                       string                `json:"id"`
	SyntheticLabel           string                `json:"syntheticLabel"`
	RequestedDurationMinutes int                   `json:"requestedDurationMinutes"`
	Status                   theatrebooking.Status `json:"status"`
	AssignedSessionID        *string               `json:"assignedSessionId"`
	Version                  int64                 `json:"version"`
}
type whiteboardResponse struct {
	SessionID          string                    `json:"sessionId"`
	SyntheticRoomLabel string                    `json:"syntheticRoomLabel"`
	Entries            []whiteboardEntryResponse `json:"entries"`
}
type whiteboardEntryResponse struct {
	RequestID                string `json:"requestId"`
	SyntheticLabel           string `json:"syntheticLabel"`
	RequestedDurationMinutes int    `json:"requestedDurationMinutes"`
}

func mapBoard(value theatrebooking.Board) any {
	sessions := make([]sessionResponse, len(value.TheatreSessions))
	for i, session := range value.TheatreSessions {
		sessions[i] = sessionResponse{session.ID, session.SyntheticRoomLabel, session.StartsAt, session.EndsAt, session.CapacityMinutes, session.AllocatedMinutes, session.Version}
	}
	requests := make([]bookingResponse, len(value.BookingRequests))
	for i, item := range value.BookingRequests {
		requests[i] = mapBookingRequest(item)
	}
	whiteboard := make([]whiteboardResponse, len(value.Whiteboard))
	for i, group := range value.Whiteboard {
		entries := make([]whiteboardEntryResponse, len(group.Entries))
		for j, entry := range group.Entries {
			entries[j] = whiteboardEntryResponse{entry.RequestID, entry.SyntheticLabel, entry.RequestedDurationMinutes}
		}
		whiteboard[i] = whiteboardResponse{group.SessionID, group.SyntheticRoomLabel, entries}
	}
	return struct {
		DevelopmentOnly bool                 `json:"developmentOnly"`
		SessionDate     string               `json:"sessionDate"`
		TheatreSessions []sessionResponse    `json:"theatreSessions"`
		BookingRequests []bookingResponse    `json:"bookingRequests"`
		Whiteboard      []whiteboardResponse `json:"whiteboard"`
	}{value.DevelopmentOnly, value.SessionDate, sessions, requests, whiteboard}
}
func mapBookingRequest(item theatrebooking.BookingRequest) bookingResponse {
	return bookingResponse{item.ID, item.SyntheticLabel, item.RequestedDurationMinutes, item.Status, item.AssignedSessionID, item.Version}
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
