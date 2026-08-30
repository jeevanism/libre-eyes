package patientsearchhttp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jeevanism/libre-eyes/internal/auth"
	"github.com/jeevanism/libre-eyes/internal/patientsearch"
	"github.com/jeevanism/libre-eyes/internal/platform/httpx"
)

type fakeService struct {
	authorizeErr    error
	searchErr       error
	duplicateErr    error
	authorizeCalls  int
	searchCalls     int
	duplicateCalls  int
	operation       patientsearch.Operation
	searchRequest   patientsearch.SearchRequest
	searchResult    patientsearch.SearchPage
	duplicateResult patientsearch.DuplicateResult
}

func (f *fakeService) Authorize(_ context.Context, _, _ string, operation patientsearch.Operation, _ auth.RequestMetadata) (patientsearch.Authorization, error) {
	f.authorizeCalls++
	f.operation = operation
	return patientsearch.Authorization{}, f.authorizeErr
}

func (f *fakeService) Search(_ context.Context, _ patientsearch.Authorization, request patientsearch.SearchRequest) (patientsearch.SearchPage, error) {
	f.searchCalls++
	f.searchRequest = request
	return f.searchResult, f.searchErr
}

func (f *fakeService) FindDuplicates(_ context.Context, _ patientsearch.Authorization, _ patientsearch.DuplicateRequest) (patientsearch.DuplicateResult, error) {
	f.duplicateCalls++
	return f.duplicateResult, f.duplicateErr
}
func (f *fakeService) Recent(_ context.Context, _ patientsearch.Authorization, _ int) (patientsearch.SearchPage, error) {
	return f.searchResult, f.searchErr
}

func TestSearchAuthorizesBeforeParsingBody(t *testing.T) {
	service := &fakeService{}
	server := testServer(service)
	request := authorizedRequest("/api/v1/patients/searches", `{not-json`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
	}
	if service.authorizeCalls != 1 || service.searchCalls != 0 {
		t.Fatalf("calls = authorize %d, search %d; want 1, 0", service.authorizeCalls, service.searchCalls)
	}
}

func TestSearchWithoutSessionDoesNotAuthorizeOrParse(t *testing.T) {
	service := &fakeService{}
	server := testServer(service)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/patients/searches", strings.NewReader(`{not-json`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized || service.authorizeCalls != 0 {
		t.Fatalf("status = %d, authorize calls = %d; want 401, 0", response.Code, service.authorizeCalls)
	}
	assertNoStore(t, response)
}

func TestSearchReturnsMinimumDisclosureContract(t *testing.T) {
	fullName, givenName, familyName := "Ada Lovelace", "Ada", "Lovelace"
	identifier := patientsearch.DisplayedIdentifier{TypeID: "9", Label: "Hospital number", Value: "123 456"}
	cursor := "opaque-cursor"
	service := &fakeService{searchResult: patientsearch.SearchPage{
		Items: []patientsearch.PatientResult{{
			PatientID: "018f34f6-45f2-4a57-8ac0-358cea11ec62",
			FullName:  &fullName, GivenName: &givenName, FamilyName: &familyName,
			DateOfBirth: "1980-01-02", Gender: patientsearch.GenderFemale,
			PrimaryIdentifier: &identifier,
		}},
		HasMore: true, NextCursor: &cursor,
	}}
	server := testServer(service)
	request := authorizedRequest("/api/v1/patients/searches", `{
		"criteria":{"kind":"identifier","identifierTypeId":"9","value":"123456"},
		"limit":25
	}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{`"patientId":"018f34f6-45f2-4a57-8ac0-358cea11ec62"`, `"typeId":"9"`, `"nextCursor":"opaque-cursor"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	for _, forbidden := range []string{"canonical", "normalized", "sessionId", `"id":9`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response contains forbidden internal field %q: %s", forbidden, body)
		}
	}
	if service.searchRequest.Criteria.Identifier == nil || service.searchRequest.Criteria.Identifier.IdentifierTypeID != 9 {
		t.Fatalf("service request = %#v, want parsed identifier criteria", service.searchRequest)
	}
	assertNoStore(t, response)
}

func TestDuplicateResponseStatesIncompleteCoverage(t *testing.T) {
	service := &fakeService{duplicateResult: patientsearch.DuplicateResult{Candidates: []patientsearch.DuplicateResultCandidate{}}}
	server := testServer(service)
	request := authorizedRequest("/api/v1/patients/duplicate-candidates", `{
		"criteria":{"kind":"demographic","givenName":"Ada","familyName":"Lovelace","dateOfBirth":"1980-01-02"}
	}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	for _, expected := range []string{`"coverage":"exact_only"`, `"populationCoverage":"current_institution_only"`, `"complete":false`, `"noCandidatesDoesNotExcludeDuplicate":true`, `"candidates":[]`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("response missing %s: %s", expected, response.Body.String())
		}
	}
	if service.operation != patientsearch.OperationDuplicateCheck || service.duplicateCalls != 1 {
		t.Fatalf("operation = %q, duplicate calls = %d", service.operation, service.duplicateCalls)
	}
}

func TestSearchRejectsUnknownFieldsAndOversizeBodiesAfterAuthorization(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{name: "unknown envelope field", body: `{"criteria":{"kind":"identifier","identifierTypeId":"1","value":"x"},"unexpected":true}`, status: http.StatusBadRequest},
		{name: "unknown criteria field", body: `{"criteria":{"kind":"identifier","identifierTypeId":"1","value":"x","unexpected":true}}`, status: http.StatusBadRequest},
		{name: "signed identifier type", body: `{"criteria":{"kind":"identifier","identifierTypeId":"+9","value":"x"}}`, status: http.StatusBadRequest},
		{name: "empty cursor", body: `{"criteria":{"kind":"identifier","identifierTypeId":"1","value":"x"},"cursor":""}`, status: http.StatusBadRequest},
		{name: "oversize", body: `{"criteria":{"kind":"identifier","identifierTypeId":"1","value":"` + strings.Repeat("x", maximumBodyBytes) + `"}}`, status: http.StatusRequestEntityTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeService{}
			server := testServer(service)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, authorizedRequest("/api/v1/patients/searches", test.body))
			if response.Code != test.status || service.authorizeCalls != 1 || service.searchCalls != 0 {
				t.Fatalf("status=%d authorize=%d search=%d body=%s", response.Code, service.authorizeCalls, service.searchCalls, response.Body.String())
			}
		})
	}
}

func TestDuplicateRejectsSearchOnlyGenderField(t *testing.T) {
	service := &fakeService{}
	server := testServer(service)
	request := authorizedRequest("/api/v1/patients/duplicate-candidates", `{
		"criteria":{"kind":"demographic","givenName":"Ada","familyName":"Lovelace","dateOfBirth":"1980-01-02","gender":"female"}
	}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || service.authorizeCalls != 1 || service.duplicateCalls != 0 {
		t.Fatalf("status=%d authorize=%d duplicate=%d body=%s", response.Code, service.authorizeCalls, service.duplicateCalls, response.Body.String())
	}
}

func TestPreAuthorizationLimiterBoundsInvalidCSRFDatabaseWork(t *testing.T) {
	service := &fakeService{authorizeErr: auth.ErrCSRF}
	server := testServer(service)
	for requestNumber := 1; requestNumber <= int(preAuthorizationTokenBurst)+1; requestNumber++ {
		request := authorizedRequest("/api/v1/patients/searches", `{}`)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if requestNumber <= int(preAuthorizationTokenBurst) && response.Code != http.StatusForbidden {
			t.Fatalf("request %d status=%d, want 403", requestNumber, response.Code)
		}
		if requestNumber == int(preAuthorizationTokenBurst)+1 {
			if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") == "" {
				t.Fatalf("bounded request status=%d Retry-After=%q body=%s", response.Code, response.Header().Get("Retry-After"), response.Body.String())
			}
		}
	}
	if service.authorizeCalls != int(preAuthorizationTokenBurst) {
		t.Fatalf("Authorize() calls=%d, want %d bounded DB attempts", service.authorizeCalls, int(preAuthorizationTokenBurst))
	}
}

func TestPreAuthorizationSourceLimiterBoundsRotatingGarbageTokens(t *testing.T) {
	service := &fakeService{authorizeErr: auth.ErrUnauthenticated}
	server := testServer(service)
	for requestNumber := 1; requestNumber <= int(preAuthorizationSourceBurst)+1; requestNumber++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/patients/searches", strings.NewReader(`{}`))
		request.AddCookie(&http.Cookie{
			Name: defaultSessionCookieName, Value: "rotating-garbage-" + strconv.Itoa(requestNumber),
		})
		request.RemoteAddr = "192.0.2.50:12345"
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if requestNumber == int(preAuthorizationSourceBurst)+1 && response.Code != http.StatusTooManyRequests {
			t.Fatalf("bounded rotating-token request status=%d body=%s", response.Code, response.Body.String())
		}
	}
	if service.authorizeCalls != int(preAuthorizationSourceBurst) {
		t.Fatalf("Authorize() calls=%d, want %d source-bounded DB attempts", service.authorizeCalls, int(preAuthorizationSourceBurst))
	}
}

func TestSearchMapsBoundaryErrorsWithoutSensitiveData(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		code       string
		retryAfter string
	}{
		{name: "unauthenticated", err: auth.ErrUnauthenticated, status: 401, code: "unauthenticated"},
		{name: "csrf", err: auth.ErrCSRF, status: 403, code: "forbidden"},
		{name: "permission", err: auth.ErrForbidden, status: 403, code: "forbidden"},
		{name: "invalid", err: patientsearch.ErrInvalidRequest, status: 400, code: "invalid_request"},
		{name: "limited", err: &patientsearch.RateLimitError{RetryAfter: time.Second}, status: 429, code: "rate_limited", retryAfter: "1"},
		{name: "unavailable", err: patientsearch.ErrUnavailable, status: 503, code: "search_unavailable"},
		{name: "timeout", err: patientsearch.ErrTimeout, status: 504, code: "search_timeout"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeService{authorizeErr: test.err}
			server := testServer(service)
			request := authorizedRequest("/api/v1/patients/searches", `{"criteria":{"kind":"identifier","identifierTypeId":"1","value":"SECRET"}}`)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "SECRET") {
				t.Fatalf("problem echoed criteria: %s", response.Body.String())
			}
			if response.Header().Get("Retry-After") != test.retryAfter {
				t.Fatalf("Retry-After=%q, want %q", response.Header().Get("Retry-After"), test.retryAfter)
			}
			assertNoStore(t, response)
		})
	}
}

func testServer(service *fakeService) http.Handler {
	mux := http.NewServeMux()
	NewHandler(service, true).Register(mux)
	return httpx.Middleware(discardLogger(), mux)
}

func authorizedRequest(path, body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.AddCookie(&http.Cookie{Name: defaultSessionCookieName, Value: "opaque-session"})
	request.Header.Set("X-CSRF-Token", "synthetic-csrf-token")
	return request
}

func assertNoStore(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", response.Header().Get("Cache-Control"))
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
