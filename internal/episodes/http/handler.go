// Package episodeshttp exposes approved episode lifecycle operations over REST JSON.
package episodeshttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/jeevanism/libre-eyes/internal/auth"
	"github.com/jeevanism/libre-eyes/internal/episodes"
	"github.com/jeevanism/libre-eyes/internal/platform/httpx"
)

const (
	sessionCookieName = "libreeyes_session"
	maximumBodyBytes  = 8 * 1024
	requestTimeout    = 5 * time.Second
)

// Service is the episode use-case boundary used by this HTTP adapter.
type Service interface {
	Authorize(context.Context, string, string, string, auth.RequestMetadata) (episodes.Authorization, error)
	Create(context.Context, episodes.Authorization, string, episodes.CreateRequest) (episodes.Episode, error)
	List(context.Context, episodes.Authorization, episodes.ListRequest) (episodes.EpisodePage, error)
	Get(context.Context, episodes.Authorization, string) (episodes.Episode, error)
	ListEvents(context.Context, episodes.Authorization, episodes.EventListRequest) (episodes.EventPage, error)
	GetEventHeader(context.Context, episodes.Authorization, string) (episodes.EventHeader, error)
	CreateDraft(context.Context, episodes.Authorization, string, episodes.DraftCreateRequest) (episodes.EventDraft, error)
	GetDraft(context.Context, episodes.Authorization, string) (episodes.EventDraft, error)
	UpdateDraft(context.Context, episodes.Authorization, string, episodes.DraftUpdateRequest) (episodes.EventDraft, error)
	AbandonDraft(context.Context, episodes.Authorization, string, int64) error
	Activate(context.Context, episodes.Authorization, episodes.LifecycleRequest) (episodes.Episode, error)
	Close(context.Context, episodes.Authorization, episodes.LifecycleRequest) (episodes.Episode, error)
	Reopen(context.Context, episodes.Authorization, episodes.LifecycleRequest) (episodes.Episode, error)
}

// Handler serves the approved episode lifecycle endpoints.
type Handler struct {
	service      Service
	cookieSecure bool
}

// NewHandler constructs the episode HTTP adapter.
func NewHandler(service Service, cookieSecure bool) *Handler {
	return &Handler{service: service, cookieSecure: cookieSecure}
}

// Register adds episode lifecycle routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/patients/{patientId}/episodes", h.list)
	mux.HandleFunc("POST /api/v1/patients/{patientId}/episodes", h.create)
	mux.HandleFunc("GET /api/v1/episodes/{episodeId}", h.get)
	mux.HandleFunc("PATCH /api/v1/episodes/{episodeId}", h.update)
	mux.HandleFunc("GET /api/v1/episodes/{episodeId}/events", h.listEvents)
	mux.HandleFunc("GET /api/v1/events/{eventId}", h.getEventHeader)
	mux.HandleFunc("POST /api/v1/episodes/{episodeId}/event-drafts", h.createDraft)
	mux.HandleFunc("GET /api/v1/event-drafts/{draftId}", h.getDraft)
	mux.HandleFunc("PATCH /api/v1/event-drafts/{draftId}", h.updateDraft)
	mux.HandleFunc("DELETE /api/v1/event-drafts/{draftId}", h.abandonDraft)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx, episodes.PermissionRead)
		if !ok {
			return
		}
		request, err := listRequest(r)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		request.PatientID = r.PathValue("patientId")
		page, err := h.service.List(ctx, authorization, request)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, mapPage(page))
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx, episodes.PermissionCreate)
		if !ok {
			return
		}
		var body struct {
			Status episodes.Status `json:"status"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			h.writeError(w, r, err)
			return
		}
		episode, err := h.service.Create(ctx, authorization, r.PathValue("patientId"), episodes.CreateRequest{Status: body.Status})
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, mapEpisode(episode))
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx, episodes.PermissionRead)
		if !ok {
			return
		}
		episode, err := h.service.Get(ctx, authorization, r.PathValue("episodeId"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, mapEpisode(episode))
	})
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx, episodes.PermissionRead)
		if !ok {
			return
		}
		request, err := eventListRequest(r)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		request.EpisodeID = r.PathValue("episodeId")
		page, err := h.service.ListEvents(ctx, authorization, request)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, mapEventPage(page))
	})
}

func (h *Handler) getEventHeader(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx, episodes.PermissionRead)
		if !ok {
			return
		}
		event, err := h.service.GetEventHeader(ctx, authorization, r.PathValue("eventId"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, mapEventHeader(event))
	})
}

func (h *Handler) createDraft(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx, episodes.PermissionDraftCreate)
		if !ok {
			return
		}
		var body draftCreateRequest
		if err := decodeDraftJSON(w, r, &body); err != nil {
			h.writeError(w, r, err)
			return
		}
		draft, err := h.service.CreateDraft(ctx, authorization, r.PathValue("episodeId"), episodes.DraftCreateRequest{EventTypeCode: body.EventTypeCode, TargetEventID: body.TargetEventID, Intent: body.Intent, Mode: body.Mode, SchemaVersion: body.SchemaVersion, Payload: body.Payload})
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, mapDraft(draft))
	})
}

func (h *Handler) getDraft(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx, episodes.PermissionDraftRead)
		if !ok {
			return
		}
		draft, err := h.service.GetDraft(ctx, authorization, r.PathValue("draftId"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, mapDraft(draft))
	})
}

func (h *Handler) updateDraft(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx, episodes.PermissionDraftUpdate)
		if !ok {
			return
		}
		var body draftUpdateRequest
		if err := decodeDraftJSON(w, r, &body); err != nil {
			h.writeError(w, r, err)
			return
		}
		draft, err := h.service.UpdateDraft(ctx, authorization, r.PathValue("draftId"), episodes.DraftUpdateRequest{ExpectedVersion: body.ExpectedVersion, SchemaVersion: body.SchemaVersion, Payload: body.Payload})
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, mapDraft(draft))
	})
}

func (h *Handler) abandonDraft(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		authorization, ok := h.authorize(w, r, ctx, episodes.PermissionDraftAbandon)
		if !ok {
			return
		}
		expected, err := strconv.ParseInt(r.URL.Query().Get("expectedVersion"), 10, 64)
		if err != nil || expected < 1 {
			h.writeError(w, r, episodes.ErrInvalidRequest)
			return
		}
		if err := h.service.AbandonDraft(ctx, authorization, r.PathValue("draftId"), expected); err != nil {
			h.writeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	h.withTimeout(w, r, func(ctx context.Context) {
		if !h.hasSessionCookie(w, r) {
			return
		}
		var body updateRequest
		if err := decodeJSON(w, r, &body); err != nil {
			h.writeError(w, r, err)
			return
		}
		permission, ok := updatePermission(body.LifecycleAction)
		if !ok {
			h.writeError(w, r, episodes.ErrInvalidRequest)
			return
		}
		authorization, authorized := h.authorize(w, r, ctx, permission)
		if !authorized {
			return
		}
		request := episodes.LifecycleRequest{
			EpisodeID: r.PathValue("episodeId"), ExpectedVersion: body.ExpectedVersion, Reason: body.ReopenReason,
		}
		var (
			episode episodes.Episode
			err     error
		)
		switch body.LifecycleAction {
		case "activate":
			episode, err = h.service.Activate(ctx, authorization, request)
		case "close":
			episode, err = h.service.Close(ctx, authorization, request)
		case "reopen":
			episode, err = h.service.Reopen(ctx, authorization, request)
		}
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, mapEpisode(episode))
	})
}

func updatePermission(action string) (string, bool) {
	switch action {
	case "activate", "close":
		return episodes.PermissionUpdate, true
	case "reopen":
		return episodes.PermissionReopen, true
	default:
		return "", false
	}
}

func (h *Handler) withTimeout(w http.ResponseWriter, r *http.Request, handle func(context.Context)) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	handle(ctx)
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, ctx context.Context, permission string) (episodes.Authorization, bool) {
	if !h.hasSessionCookie(w, r) {
		return episodes.Authorization{}, false
	}
	cookie, _ := r.Cookie(sessionCookieName)
	authorization, err := h.service.Authorize(ctx, cookie.Value, r.Header.Get("X-CSRF-Token"), permission, requestMetadata(r))
	if err != nil {
		h.writeError(w, r, err)
		return episodes.Authorization{}, false
	}
	return authorization, true
}

func (h *Handler) hasSessionCookie(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		h.writeError(w, r, auth.ErrUnauthenticated)
		return false
	}
	return true
}

type updateRequest struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	LifecycleAction string `json:"lifecycleAction"`
	ReopenReason    string `json:"reopenReason,omitempty"`
}

type draftCreateRequest struct {
	EventTypeCode string               `json:"eventTypeCode"`
	TargetEventID *string              `json:"targetEventId"`
	Intent        episodes.DraftIntent `json:"intent"`
	Mode          episodes.DraftMode   `json:"mode"`
	SchemaVersion int64                `json:"schemaVersion"`
	Payload       json.RawMessage      `json:"payload"`
}

type draftUpdateRequest struct {
	ExpectedVersion int64           `json:"expectedVersion"`
	SchemaVersion   int64           `json:"schemaVersion"`
	Payload         json.RawMessage `json:"payload"`
}

func listRequest(r *http.Request) (episodes.ListRequest, error) {
	request := episodes.ListRequest{Cursor: r.URL.Query().Get("cursor")}
	if len(request.Cursor) > 512 {
		return episodes.ListRequest{}, episodes.ErrInvalidRequest
	}
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return episodes.ListRequest{}, episodes.ErrInvalidRequest
		}
		if limit < 1 || limit > 100 {
			return episodes.ListRequest{}, episodes.ErrInvalidRequest
		}
		request.Limit = limit
	}
	return request, nil
}

func eventListRequest(r *http.Request) (episodes.EventListRequest, error) {
	request, err := listRequest(r)
	if err != nil {
		return episodes.EventListRequest{}, err
	}
	return episodes.EventListRequest{Limit: request.Limit, Cursor: request.Cursor}, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maximumBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: %w", episodes.ErrInvalidRequest, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return episodes.ErrInvalidRequest
	}
	return nil
}

func decodeDraftJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: %w", episodes.ErrInvalidRequest, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return episodes.ErrInvalidRequest
	}
	return nil
}

func requestMetadata(r *http.Request) auth.RequestMetadata {
	return auth.RequestMetadata{
		CorrelationID: httpx.CorrelationID(r.Context()),
		SourceIPClass: classifyRemoteIP(r.RemoteAddr),
	}
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

type episodeResponse struct {
	ID              string          `json:"id"`
	PatientID       string          `json:"patientId"`
	Status          episodes.Status `json:"status"`
	StartedAt       *time.Time      `json:"startedAt"`
	EndedAt         *time.Time      `json:"endedAt"`
	SupportServices bool            `json:"supportServices"`
	ChangeTracker   bool            `json:"changeTracker"`
	Version         int64           `json:"version"`
}

type pageResponse struct {
	Items      []episodeResponse `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}

type eventHeaderResponse struct {
	ID            string    `json:"id"`
	EpisodeID     *string   `json:"episodeId"`
	EventTypeCode string    `json:"eventTypeCode"`
	OccurredAt    time.Time `json:"occurredAt"`
	Status        string    `json:"status"`
	Version       int64     `json:"version"`
}

type eventPageResponse struct {
	Items      []eventHeaderResponse `json:"items"`
	NextCursor *string               `json:"nextCursor"`
}

type draftResponse struct {
	ID                  string               `json:"id"`
	EpisodeID           string               `json:"episodeId"`
	EventTypeCode       string               `json:"eventTypeCode"`
	TargetEventID       *string              `json:"targetEventId,omitempty"`
	Intent              episodes.DraftIntent `json:"intent"`
	Mode                episodes.DraftMode   `json:"mode"`
	SchemaVersion       int64                `json:"schemaVersion"`
	Payload             json.RawMessage      `json:"payload"`
	Version             int64                `json:"version"`
	ExpiresAt           time.Time            `json:"expiresAt"`
	NewerCommittedEdits *bool                `json:"newerCommittedEdits,omitempty"`
}

func mapEpisode(episode episodes.Episode) episodeResponse {
	return episodeResponse{
		ID: episode.ID, PatientID: episode.PatientID, Status: episode.Status,
		StartedAt: episode.StartedAt, EndedAt: episode.EndedAt,
		SupportServices: episode.SupportServices, ChangeTracker: episode.ChangeTracker, Version: episode.Version,
	}
}

func mapPage(page episodes.EpisodePage) pageResponse {
	items := make([]episodeResponse, len(page.Items))
	for index, episode := range page.Items {
		items[index] = mapEpisode(episode)
	}
	return pageResponse{Items: items, NextCursor: page.NextCursor}
}

func mapEventHeader(event episodes.EventHeader) eventHeaderResponse {
	return eventHeaderResponse{
		ID: event.ID, EpisodeID: event.EpisodeID, EventTypeCode: event.EventTypeCode,
		OccurredAt: event.OccurredAt, Status: event.Status, Version: event.Version,
	}
}

func mapEventPage(page episodes.EventPage) eventPageResponse {
	items := make([]eventHeaderResponse, len(page.Items))
	for index, event := range page.Items {
		items[index] = mapEventHeader(event)
	}
	return eventPageResponse{Items: items, NextCursor: page.NextCursor}
}

func mapDraft(draft episodes.EventDraft) draftResponse {
	response := draftResponse{ID: draft.ID, EpisodeID: draft.EpisodeID, EventTypeCode: draft.EventTypeCode, TargetEventID: draft.TargetEventID, Intent: draft.Intent, Mode: draft.Mode, SchemaVersion: draft.SchemaVersion, Payload: draft.Payload, Version: draft.Version, ExpiresAt: draft.ExpiresAt}
	if draft.EventTypeCode != "ophthalmology.visual_acuity" && draft.EventTypeCode != "ophthalmology.principal_diagnosis_demo" {
		response.NewerCommittedEdits = &draft.NewerCommittedEdits
	}
	return response
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		writeProblem(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large")
	case errors.Is(err, auth.ErrUnauthenticated):
		h.expireSessionCookie(w)
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required")
	case errors.Is(err, auth.ErrCSRF), errors.Is(err, auth.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "Operation not permitted")
	case errors.Is(err, auth.ErrConflict):
		writeProblem(w, r, http.StatusConflict, "conflict", "Episode changed; refresh and try again")
	case errors.Is(err, episodes.ErrInvalidRequest), errors.Is(err, episodes.ErrInvalidTransition):
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid episode request")
	case errors.Is(err, context.DeadlineExceeded):
		writeProblem(w, r, http.StatusGatewayTimeout, "episode_timeout", "Episode request timed out")
	default:
		writeProblem(w, r, http.StatusServiceUnavailable, "episode_unavailable", "Episode service is unavailable")
	}
}

func (h *Handler) expireSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1,
		Expires: time.Unix(1, 0).UTC(), HttpOnly: true,
		Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
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
