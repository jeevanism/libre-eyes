// Package patientsearchhttp exposes exact patient search and duplicate checks over REST JSON.
package patientsearchhttp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/patientsearch"
	"github.com/jeevanism/visionopus/internal/platform/httpx"
)

const (
	defaultSessionCookieName    = "visionopus_session"
	maximumBodyBytes            = 8 * 1024
	requestTimeout              = 5 * time.Second
	preAuthorizationTokenBurst  = 20.0
	preAuthorizationTokenRate   = 2.0
	preAuthorizationSourceBurst = 100.0
	preAuthorizationSourceRate  = 20.0
)

// Service is the patient-search use-case boundary used by the HTTP adapter.
type Service interface {
	Authorize(context.Context, string, string, patientsearch.Operation, auth.RequestMetadata) (patientsearch.Authorization, error)
	Search(context.Context, patientsearch.Authorization, patientsearch.SearchRequest) (patientsearch.SearchPage, error)
	FindDuplicates(context.Context, patientsearch.Authorization, patientsearch.DuplicateRequest) (patientsearch.DuplicateResult, error)
}

// Handler serves the patient-search API.
type Handler struct {
	service       Service
	cookieName    string
	cookieSecure  bool
	tokenLimiter  *boundaryLimiter
	sourceLimiter *boundaryLimiter
}

// NewHandler constructs the patient-search HTTP adapter.
func NewHandler(service Service, cookieSecure bool) *Handler {
	return &Handler{
		service: service, cookieName: defaultSessionCookieName, cookieSecure: cookieSecure,
		tokenLimiter:  newBoundaryLimiter(preAuthorizationTokenBurst, preAuthorizationTokenRate, 8192),
		sourceLimiter: newBoundaryLimiter(preAuthorizationSourceBurst, preAuthorizationSourceRate, 4096),
	}
}

// Register adds patient-search routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/patients/searches", h.search)
	mux.HandleFunc("POST /api/v1/patients/duplicate-candidates", h.findDuplicates)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	r = r.WithContext(ctx)

	authorization, ok := h.authorize(w, r, patientsearch.OperationSearch)
	if !ok {
		return
	}
	body, err := decodeSearchRequest(w, r)
	if err != nil {
		h.writeDecodeError(w, r, err)
		return
	}
	result, err := h.service.Search(ctx, authorization, body)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSearchPage(result))
}

func (h *Handler) findDuplicates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	r = r.WithContext(ctx)

	authorization, ok := h.authorize(w, r, patientsearch.OperationDuplicateCheck)
	if !ok {
		return
	}
	body, err := decodeDuplicateRequest(w, r)
	if err != nil {
		h.writeDecodeError(w, r, err)
		return
	}
	result, err := h.service.FindDuplicates(ctx, authorization, body)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapDuplicateResult(result))
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, operation patientsearch.Operation) (patientsearch.Authorization, bool) {
	cookie, err := r.Cookie(h.cookieName)
	if err != nil || cookie.Value == "" {
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required")
		return patientsearch.Authorization{}, false
	}
	if allowed, retry := h.allowPreAuthorization(cookie.Value, r.RemoteAddr); !allowed {
		h.writeRateLimit(w, r, retry)
		return patientsearch.Authorization{}, false
	}
	authorization, err := h.service.Authorize(
		r.Context(), cookie.Value, r.Header.Get("X-CSRF-Token"), operation, requestMetadata(r),
	)
	if err != nil {
		h.writeServiceError(w, r, err)
		return patientsearch.Authorization{}, false
	}
	return authorization, true
}

func (h *Handler) allowPreAuthorization(token, remoteAddress string) (bool, time.Duration) {
	sourceAllowed, sourceRetry := h.sourceLimiter.allow(clientAddress(remoteAddress))
	if !sourceAllowed {
		return false, sourceRetry
	}
	tokenDigest := sha256.Sum256([]byte(token))
	tokenAllowed, tokenRetry := h.tokenLimiter.allow(string(tokenDigest[:]))
	if !tokenAllowed {
		return false, tokenRetry
	}
	return true, 0
}

func (h *Handler) writeRateLimit(w http.ResponseWriter, r *http.Request, retry time.Duration) {
	retrySeconds := max(int(retry.Round(time.Second)/time.Second), 1)
	w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
	writeProblem(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests")
}

func (h *Handler) writeDecodeError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeProblem(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large")
		return
	}
	writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid patient search request")
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var rateLimit *patientsearch.RateLimitError
	switch {
	case errors.Is(err, auth.ErrUnauthenticated):
		h.expireSessionCookie(w)
		writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required")
	case errors.Is(err, auth.ErrCSRF), errors.Is(err, auth.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "Operation not permitted")
	case errors.Is(err, patientsearch.ErrInvalidRequest):
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid patient search request")
	case errors.As(err, &rateLimit):
		h.writeRateLimit(w, r, rateLimit.RetryAfter)
	case errors.Is(err, patientsearch.ErrTimeout), errors.Is(err, context.DeadlineExceeded):
		writeProblem(w, r, http.StatusGatewayTimeout, "search_timeout", "Patient search timed out")
	default:
		writeProblem(w, r, http.StatusServiceUnavailable, "search_unavailable", "Patient search is unavailable")
	}
}

func (h *Handler) expireSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: h.cookieName, Value: "", Path: "/", MaxAge: -1,
		Expires: time.Unix(1, 0).UTC(), HttpOnly: true,
		Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

type searchEnvelope struct {
	Criteria json.RawMessage `json:"criteria"`
	Limit    int             `json:"limit,omitempty"`
	Cursor   *string         `json:"cursor,omitempty"`
}

type duplicateEnvelope struct {
	Criteria json.RawMessage `json:"criteria"`
}

type criteriaKind struct {
	Kind string `json:"kind"`
}

type identifierCriteria struct {
	Kind             string `json:"kind"`
	IdentifierTypeID string `json:"identifierTypeId"`
	Value            string `json:"value"`
}

type demographicCriteria struct {
	Kind        string  `json:"kind"`
	GivenName   *string `json:"givenName,omitempty"`
	FamilyName  string  `json:"familyName"`
	DateOfBirth string  `json:"dateOfBirth"`
	Gender      string  `json:"gender,omitempty"`
}

type duplicateDemographicCriteria struct {
	Kind        string `json:"kind"`
	GivenName   string `json:"givenName"`
	FamilyName  string `json:"familyName"`
	DateOfBirth string `json:"dateOfBirth"`
}

func decodeSearchRequest(w http.ResponseWriter, r *http.Request) (patientsearch.SearchRequest, error) {
	var envelope searchEnvelope
	if err := decodeJSON(w, r, &envelope); err != nil {
		return patientsearch.SearchRequest{}, err
	}
	criteria, err := decodeCriteria(envelope.Criteria, false)
	if err != nil {
		return patientsearch.SearchRequest{}, err
	}
	cursor := ""
	if envelope.Cursor != nil {
		if *envelope.Cursor == "" || len(*envelope.Cursor) > 512 {
			return patientsearch.SearchRequest{}, patientsearch.ErrInvalidRequest
		}
		cursor = *envelope.Cursor
	}
	return patientsearch.SearchRequest{Criteria: criteria, Limit: envelope.Limit, Cursor: cursor}, nil
}

func decodeDuplicateRequest(w http.ResponseWriter, r *http.Request) (patientsearch.DuplicateRequest, error) {
	var envelope duplicateEnvelope
	if err := decodeJSON(w, r, &envelope); err != nil {
		return patientsearch.DuplicateRequest{}, err
	}
	criteria, err := decodeCriteria(envelope.Criteria, true)
	if err != nil {
		return patientsearch.DuplicateRequest{}, err
	}
	return patientsearch.DuplicateRequest{Criteria: criteria}, nil
}

func decodeCriteria(raw json.RawMessage, duplicate bool) (patientsearch.SearchCriteria, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return patientsearch.SearchCriteria{}, patientsearch.ErrInvalidRequest
	}
	var selector criteriaKind
	if err := json.Unmarshal(raw, &selector); err != nil {
		return patientsearch.SearchCriteria{}, err
	}
	switch selector.Kind {
	case string(patientsearch.CriteriaIdentifier):
		var input identifierCriteria
		if err := decodeRaw(raw, &input); err != nil {
			return patientsearch.SearchCriteria{}, err
		}
		typeID, ok := parseIdentifier(input.IdentifierTypeID)
		if !ok || input.Value == "" {
			return patientsearch.SearchCriteria{}, patientsearch.ErrInvalidRequest
		}
		return patientsearch.SearchCriteria{
			Kind:       patientsearch.CriteriaIdentifier,
			Identifier: &patientsearch.IdentifierCriteria{IdentifierTypeID: typeID, Value: input.Value},
		}, nil
	case string(patientsearch.CriteriaDemographic):
		if duplicate {
			var input duplicateDemographicCriteria
			if err := decodeRaw(raw, &input); err != nil {
				return patientsearch.SearchCriteria{}, err
			}
			if input.GivenName == "" || input.FamilyName == "" || input.DateOfBirth == "" {
				return patientsearch.SearchCriteria{}, patientsearch.ErrInvalidRequest
			}
			return patientsearch.SearchCriteria{
				Kind: patientsearch.CriteriaDemographic,
				Demographic: &patientsearch.DemographicInput{
					GivenName: &input.GivenName, FamilyName: input.FamilyName,
					DateOfBirth: input.DateOfBirth,
				},
			}, nil
		}
		var input demographicCriteria
		if err := decodeRaw(raw, &input); err != nil {
			return patientsearch.SearchCriteria{}, err
		}
		if input.FamilyName == "" || input.DateOfBirth == "" {
			return patientsearch.SearchCriteria{}, patientsearch.ErrInvalidRequest
		}
		return patientsearch.SearchCriteria{
			Kind: patientsearch.CriteriaDemographic,
			Demographic: &patientsearch.DemographicInput{
				GivenName: input.GivenName, FamilyName: input.FamilyName,
				DateOfBirth: input.DateOfBirth, Gender: patientsearch.Gender(input.Gender),
			},
		}, nil
	default:
		return patientsearch.SearchCriteria{}, patientsearch.ErrInvalidRequest
	}
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

func decodeRaw(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("criteria must contain one JSON value")
	}
	return nil
}

func parseIdentifier(value string) (int64, bool) {
	if value == "" || value[0] < '1' || value[0] > '9' {
		return 0, false
	}
	for index := 1; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil && parsed > 0
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

type displayedIdentifierResponse struct {
	TypeID string `json:"typeId"`
	Label  string `json:"label"`
	Value  string `json:"value"`
}

type patientResponse struct {
	PatientID         string                       `json:"patientId"`
	FullName          *string                      `json:"fullName"`
	GivenName         *string                      `json:"givenName"`
	FamilyName        *string                      `json:"familyName"`
	DateOfBirth       string                       `json:"dateOfBirth"`
	Gender            patientsearch.Gender         `json:"gender"`
	PrimaryIdentifier *displayedIdentifierResponse `json:"primaryIdentifier"`
	Deceased          bool                         `json:"deceased"`
	DateOfDeath       *string                      `json:"dateOfDeath"`
}

type cursorPageResponse struct {
	HasMore    bool    `json:"hasMore"`
	NextCursor *string `json:"nextCursor"`
}

type searchPageResponse struct {
	Items []patientResponse  `json:"items"`
	Page  cursorPageResponse `json:"page"`
}

type duplicateCandidateResponse struct {
	Reason  patientsearch.DuplicateReason `json:"reason"`
	Patient patientResponse               `json:"patient"`
}

type duplicateResultResponse struct {
	Coverage                            string                       `json:"coverage"`
	PopulationCoverage                  string                       `json:"populationCoverage"`
	Complete                            bool                         `json:"complete"`
	NoCandidatesDoesNotExcludeDuplicate bool                         `json:"noCandidatesDoesNotExcludeDuplicate"`
	HardConflict                        bool                         `json:"hardConflict"`
	Truncated                           bool                         `json:"truncated"`
	Candidates                          []duplicateCandidateResponse `json:"candidates"`
}

func mapSearchPage(page patientsearch.SearchPage) searchPageResponse {
	items := make([]patientResponse, len(page.Items))
	for index, item := range page.Items {
		items[index] = mapPatient(item)
	}
	return searchPageResponse{Items: items, Page: cursorPageResponse{HasMore: page.HasMore, NextCursor: page.NextCursor}}
}

func mapDuplicateResult(result patientsearch.DuplicateResult) duplicateResultResponse {
	candidates := make([]duplicateCandidateResponse, len(result.Candidates))
	for index, candidate := range result.Candidates {
		candidates[index] = duplicateCandidateResponse{Reason: candidate.Reason, Patient: mapPatient(candidate.Patient)}
	}
	return duplicateResultResponse{
		Coverage: "exact_only", PopulationCoverage: "current_institution_only",
		Complete: false, NoCandidatesDoesNotExcludeDuplicate: true,
		HardConflict: result.HardConflict, Truncated: result.Truncated, Candidates: candidates,
	}
}

func mapPatient(patient patientsearch.PatientResult) patientResponse {
	result := patientResponse{
		PatientID: patient.PatientID, FullName: patient.FullName,
		GivenName: patient.GivenName, FamilyName: patient.FamilyName,
		DateOfBirth: patient.DateOfBirth, Gender: patient.Gender,
		Deceased: patient.Deceased, DateOfDeath: patient.DateOfDeath,
	}
	if patient.PrimaryIdentifier != nil {
		result.PrimaryIdentifier = &displayedIdentifierResponse{
			TypeID: patient.PrimaryIdentifier.TypeID,
			Label:  patient.PrimaryIdentifier.Label,
			Value:  patient.PrimaryIdentifier.Value,
		}
	}
	return result
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
