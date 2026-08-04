// Package authhttp exposes authentication use cases over REST JSON.
package authhttp

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
	"strings"
	"time"

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/platform/httpx"
)

const (
	sessionCookieName          = "visionopus_session"
	maximumBodyBytes           = 16 * 1024
	loginLimitPerClient        = 10
	loginLimitGlobal           = 500
	loginOptionsLimitPerClient = 120
	loginOptionsLimitGlobal    = 5000
	publicRateLimitWindow      = time.Minute
)

// Service is the authentication use-case boundary used by the HTTP adapter.
type Service interface {
	ListLoginOptions(context.Context) ([]auth.InstitutionOption, error)
	Login(context.Context, auth.LoginRequest, auth.RequestMetadata) (auth.CreatedSession, error)
	CurrentSession(context.Context, string, auth.RequestMetadata) (auth.Session, error)
	Logout(context.Context, string, string, auth.RequestMetadata) error
	ReplaceContext(context.Context, string, string, auth.ContextRequest, auth.RequestMetadata) (auth.Session, error)
}

// Handler serves the authentication API.
type Handler struct {
	service       Service
	cookieSecure  bool
	cookieMaxAge  int
	clientLimiter *fixedWindowLimiter
	globalLimiter *fixedWindowLimiter
	now           func() time.Time
}

// NewHandler constructs the authentication HTTP adapter.
func NewHandler(service Service, cookieSecure bool, absoluteTimeout time.Duration) *Handler {
	return &Handler{
		service:       service,
		cookieSecure:  cookieSecure,
		cookieMaxAge:  int(absoluteTimeout.Seconds()),
		clientLimiter: newFixedWindowLimiter(4096),
		globalLimiter: newFixedWindowLimiter(8),
		now:           time.Now,
	}
}

// Register adds authentication routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/login-options", h.loginOptions)
	mux.HandleFunc("POST /api/v1/auth/sessions", h.login)
	mux.HandleFunc("GET /api/v1/auth/session", h.currentSession)
	mux.HandleFunc("DELETE /api/v1/auth/session", h.logout)
	mux.HandleFunc("PUT /api/v1/auth/session/context", h.replaceContext)
}

func (h *Handler) loginOptions(w http.ResponseWriter, r *http.Request) {
	if !h.allowPublicRequest(w, r, "login-options", loginOptionsLimitPerClient, loginOptionsLimitGlobal) {
		return
	}
	options, err := h.service.ListLoginOptions(r.Context())
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Unable to load login options")
		return
	}
	institutions := make([]institutionOptionResponse, 0, len(options))
	for _, option := range options {
		sites := make([]referenceResponse, 0, len(option.Sites))
		for _, site := range option.Sites {
			sites = append(sites, mapReference(site))
		}
		institutions = append(institutions, institutionOptionResponse{
			ID: strconv.FormatInt(option.Institution.ID, 10), Name: option.Institution.Name, Sites: sites,
		})
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, loginOptionsResponse{Institutions: institutions})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if !h.allowPublicRequest(w, r, "login", loginLimitPerClient, loginLimitGlobal) {
		return
	}
	var body loginRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid login request")
		return
	}
	institutionID, ok := parseIdentifier(body.InstitutionID)
	if !ok {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid login request")
		return
	}
	siteID, ok := parseIdentifier(body.SiteID)
	if !ok || len(body.Username) < 1 || len(body.Username) > 255 || len(body.Password) < 1 || len(body.Password) > 1024 {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid login request")
		return
	}
	var firmID *int64
	if body.FirmID != nil {
		parsed, valid := parseIdentifier(*body.FirmID)
		if !valid {
			writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid login request")
			return
		}
		firmID = &parsed
	}

	created, err := h.service.Login(r.Context(), auth.LoginRequest{
		Username: body.Username, Password: body.Password,
		InstitutionID: institutionID, SiteID: siteID, FirmID: firmID,
	}, requestMetadata(r))
	if err != nil {
		if errors.Is(err, auth.ErrAuthenticationFailed) {
			writeProblem(w, r, http.StatusUnauthorized, "authentication_failed", "Authentication failed")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Unable to create session")
		return
	}
	h.setSessionCookie(w, created.Token)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, mapSession(created.Session))
}

func (h *Handler) allowPublicRequest(w http.ResponseWriter, r *http.Request, scope string, perClientLimit, globalLimit int) bool {
	now := h.now().UTC()
	clientAllowed, clientRetry := h.clientLimiter.allow(scope+":"+clientAddress(r.RemoteAddr), perClientLimit, publicRateLimitWindow, now)
	if !clientAllowed {
		h.writeRateLimit(w, r, clientRetry)
		return false
	}
	globalAllowed, globalRetry := h.globalLimiter.allow(scope, globalLimit, publicRateLimitWindow, now)
	if globalAllowed {
		return true
	}
	h.writeRateLimit(w, r, globalRetry)
	return false
}

func (h *Handler) writeRateLimit(w http.ResponseWriter, r *http.Request, retry time.Duration) {
	retrySeconds := int(retry.Round(time.Second) / time.Second)
	if retrySeconds < 1 {
		retrySeconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
	writeProblem(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests")
}

func (h *Handler) currentSession(w http.ResponseWriter, r *http.Request) {
	token, ok := sessionToken(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	session, err := h.service.CurrentSession(r.Context(), token, requestMetadata(r))
	if err != nil {
		h.writeSessionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, mapSession(session))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token, ok := sessionToken(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	if err := h.service.Logout(r.Context(), token, r.Header.Get("X-CSRF-Token"), requestMetadata(r)); err != nil {
		h.writeSessionError(w, r, err)
		return
	}
	h.expireSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) replaceContext(w http.ResponseWriter, r *http.Request) {
	token, ok := sessionToken(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return
	}
	version, ok := parseETag(r.Header.Get("If-Match"))
	if !ok {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "A valid context version is required")
		return
	}
	var body replaceContextRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid context request")
		return
	}
	institutionID, institutionOK := parseIdentifier(body.InstitutionID)
	siteID, siteOK := parseIdentifier(body.SiteID)
	firmID, firmOK := parseIdentifier(body.FirmID)
	if !institutionOK || !siteOK || !firmOK {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid context request")
		return
	}

	session, err := h.service.ReplaceContext(r.Context(), token, r.Header.Get("X-CSRF-Token"), auth.ContextRequest{
		InstitutionID: institutionID, SiteID: siteID, FirmID: firmID, Version: version,
	}, requestMetadata(r))
	if err != nil {
		h.writeSessionError(w, r, err)
		return
	}
	w.Header().Set("ETag", fmt.Sprintf("\"%d\"", session.ContextVersion))
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, mapSession(session))
}

func (h *Handler) writeSessionError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, auth.ErrUnauthenticated):
		h.expireSessionCookie(w)
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required")
	case errors.Is(err, auth.ErrCSRF):
		writeProblem(w, r, http.StatusForbidden, "csrf_rejected", "Request verification failed")
	case errors.Is(err, auth.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "Operation not permitted")
	case errors.Is(err, auth.ErrConflict):
		writeProblem(w, r, http.StatusConflict, "context_conflict", "The context was changed by another request")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Unable to process session request")
	}
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", MaxAge: h.cookieMaxAge,
		HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) expireSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1,
		Expires: time.Unix(1, 0).UTC(), HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func sessionToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	return valueOrEmpty(cookie), err == nil && cookie.Value != ""
}

func valueOrEmpty(cookie *http.Cookie) string {
	if cookie == nil {
		return ""
	}
	return cookie.Value
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maximumBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func parseIdentifier(value string) (int64, bool) {
	if value == "" || value[0] == '0' || strings.TrimSpace(value) != value {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil && parsed > 0
}

func parseETag(value string) (int64, bool) {
	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
		return 0, false
	}
	return parseIdentifier(value[1 : len(value)-1])
}

func requestMetadata(r *http.Request) auth.RequestMetadata {
	return auth.RequestMetadata{
		CorrelationID: httpx.CorrelationID(r.Context()),
		SourceIPClass: classifyRemoteIP(r.RemoteAddr),
	}
}

func classifyRemoteIP(remote string) string {
	address, err := netip.ParseAddr(clientAddress(remote))
	if err != nil {
		return "unknown"
	}
	switch {
	case address.IsLoopback():
		return "loopback"
	case address.IsPrivate():
		return "private"
	default:
		return "public"
	}
}

func clientAddress(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return "unknown"
	}
	return address.Unmap().String()
}

type loginRequest struct {
	Username      string  `json:"username"`
	Password      string  `json:"password"`
	InstitutionID string  `json:"institutionId"`
	SiteID        string  `json:"siteId"`
	FirmID        *string `json:"firmId,omitempty"`
}

type replaceContextRequest struct {
	InstitutionID string `json:"institutionId"`
	SiteID        string `json:"siteId"`
	FirmID        string `json:"firmId"`
}

type referenceResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type institutionOptionResponse struct {
	ID    string              `json:"id"`
	Name  string              `json:"name"`
	Sites []referenceResponse `json:"sites"`
}

type loginOptionsResponse struct {
	Institutions []institutionOptionResponse `json:"institutions"`
}

type userResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type contextResponse struct {
	Institution referenceResponse `json:"institution"`
	Site        referenceResponse `json:"site"`
	Firm        referenceResponse `json:"firm"`
}

type sessionResponse struct {
	User              userResponse    `json:"user"`
	Context           contextResponse `json:"context"`
	Permissions       []string        `json:"permissions"`
	CSRFToken         string          `json:"csrfToken"`
	IdleExpiresAt     time.Time       `json:"idleExpiresAt"`
	AbsoluteExpiresAt time.Time       `json:"absoluteExpiresAt"`
	ContextVersion    int64           `json:"contextVersion"`
}

func mapReference(value auth.Reference) referenceResponse {
	return referenceResponse{ID: strconv.FormatInt(value.ID, 10), Name: value.Name}
}

func mapSession(value auth.Session) sessionResponse {
	return sessionResponse{
		User: userResponse{ID: strconv.FormatInt(value.User.ID, 10), DisplayName: value.User.DisplayName},
		Context: contextResponse{
			Institution: mapReference(value.Context.Institution),
			Site:        mapReference(value.Context.Site),
			Firm:        mapReference(value.Context.Firm),
		},
		Permissions: value.Permissions, CSRFToken: value.CSRFToken,
		IdleExpiresAt: value.IdleExpiresAt, AbsoluteExpiresAt: value.AbsoluteExpiresAt,
		ContextVersion: value.ContextVersion,
	}
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
	writeJSON(w, status, problem{
		Type: "about:blank", Title: title, Status: status, Code: code,
		CorrelationID: httpx.CorrelationID(r.Context()),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
