package authhttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/platform/httpx"
)

type fakeService struct {
	created        auth.CreatedSession
	loginErr       error
	loginCalls     int
	logoutCSRF     string
	replaceRequest auth.ContextRequest
	replaceCSRF    string
	replaceErr     error
	replaceCalls   int
}

func (f *fakeService) ListLoginOptions(context.Context) ([]auth.InstitutionOption, error) {
	return []auth.InstitutionOption{{
		Institution: auth.Reference{ID: 1, Name: "Vision Hospital"},
		Sites:       []auth.Reference{{ID: 2, Name: "Eye Clinic"}},
	}}, nil
}

func (f *fakeService) Login(context.Context, auth.LoginRequest, auth.RequestMetadata) (auth.CreatedSession, error) {
	f.loginCalls++
	return f.created, f.loginErr
}

func (f *fakeService) CurrentSession(context.Context, string, auth.RequestMetadata) (auth.Session, error) {
	return f.created.Session, nil
}

func (f *fakeService) Logout(_ context.Context, _, csrf string, _ auth.RequestMetadata) error {
	f.logoutCSRF = csrf
	return nil
}

func (f *fakeService) ReplaceContext(_ context.Context, _, csrf string, request auth.ContextRequest, _ auth.RequestMetadata) (auth.Session, error) {
	f.replaceCalls++
	f.replaceCSRF = csrf
	f.replaceRequest = request
	return f.created.Session, f.replaceErr
}

func TestLoginSetsOpaqueHTTPOnlyCookieAndContractResponse(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := &fakeService{created: auth.CreatedSession{
		Token: "opaque-token",
		Session: auth.Session{
			User: auth.User{ID: 7, DisplayName: "Synthetic Clinician"},
			Context: auth.UserContext{
				Institution: auth.Reference{ID: 1, Name: "Vision Hospital"},
				Site:        auth.Reference{ID: 2, Name: "Eye Clinic"},
				Firm:        auth.Reference{ID: 3, Name: "Ophthalmology"},
			},
			Permissions: []string{"session.read_self"}, CSRFToken: "csrf-token-value-with-at-least-32-characters",
			IdleExpiresAt: now.Add(15 * time.Minute), AbsoluteExpiresAt: now.Add(12 * time.Hour), ContextVersion: 1,
		},
	}}
	server := testServer(service, true)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions", strings.NewReader(`{
		"username":"clinician","password":"secret","institutionId":"1","siteId":"2","firmId":"3"
	}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != "opaque-token" || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie = %#v, want opaque secure HttpOnly SameSite=Lax cookie", cookies)
	}
	if !strings.Contains(response.Body.String(), `"id":"7"`) || strings.Contains(response.Body.String(), "opaque-token") {
		t.Fatalf("response body does not match safe contract: %s", response.Body.String())
	}
}

func TestLoginRejectsUnknownFieldsBeforeCallingService(t *testing.T) {
	service := &fakeService{}
	server := testServer(service, false)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions", strings.NewReader(`{
		"username":"clinician","password":"secret","institutionId":"1","siteId":"2","unexpected":true
	}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	if service.loginCalls != 0 {
		t.Fatalf("login calls = %d, want 0", service.loginCalls)
	}
}

func TestLoginRateLimitReturnsContractResponse(t *testing.T) {
	service := &fakeService{}
	server := testServer(service, false)

	for attempt := 1; attempt <= loginLimitPerClient+1; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions", strings.NewReader(`{
			"username":"clinician","password":"secret","institutionId":"1","siteId":"2"
		}`))
		request.RemoteAddr = "192.0.2.10:12345"
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)

		if attempt <= loginLimitPerClient && response.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d was rate limited early", attempt)
		}
		if attempt == loginLimitPerClient+1 {
			if response.Code != http.StatusTooManyRequests {
				t.Fatalf("status = %d, want 429; body=%s", response.Code, response.Body.String())
			}
			if response.Header().Get("Retry-After") != "60" {
				t.Fatalf("Retry-After = %q, want 60", response.Header().Get("Retry-After"))
			}
			if !strings.Contains(response.Body.String(), `"code":"rate_limited"`) {
				t.Fatalf("problem body = %s, want rate_limited code", response.Body.String())
			}
		}
	}
	if service.loginCalls != loginLimitPerClient {
		t.Fatalf("Login() calls = %d, want %d", service.loginCalls, loginLimitPerClient)
	}

	for range loginLimitGlobal {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions", strings.NewReader(`{}`))
		request.RemoteAddr = "192.0.2.10:12345"
		server.ServeHTTP(httptest.NewRecorder(), request)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions", strings.NewReader(`{
		"username":"clinician","password":"secret","institutionId":"1","siteId":"2"
	}`))
	request.RemoteAddr = "192.0.2.11:12345"
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code == http.StatusTooManyRequests {
		t.Fatal("one rate-limited client exhausted the global login allowance")
	}
}

func TestLoginGlobalRateLimitSurvivesClientMapSaturation(t *testing.T) {
	service := &fakeService{}
	server := testServer(service, false)

	for attempt := 1; attempt <= 4100; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions", strings.NewReader(`{
			"username":"clinician","password":"secret","institutionId":"1","siteId":"2"
		}`))
		request.RemoteAddr = fmt.Sprintf("192.0.%d.%d:12345", attempt/256, attempt%256)
		server.ServeHTTP(httptest.NewRecorder(), request)
	}
	if service.loginCalls != loginLimitGlobal {
		t.Fatalf("Login() calls = %d, want global limit %d", service.loginCalls, loginLimitGlobal)
	}
}

func TestLoginFailureUsesGenericProblem(t *testing.T) {
	service := &fakeService{loginErr: auth.ErrAuthenticationFailed}
	server := testServer(service, false)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions", strings.NewReader(`{
		"username":"unknown","password":"wrong","institutionId":"1","siteId":"2"
	}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"code":"authentication_failed"`) || strings.Contains(response.Body.String(), "unknown") {
		t.Fatalf("problem body is not generic: %s", response.Body.String())
	}
}

func TestLogoutPassesCSRFAndExpiresCookie(t *testing.T) {
	service := &fakeService{}
	server := testServer(service, true)
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/session", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-token"})
	request.Header.Set("X-CSRF-Token", "provided-csrf")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if service.logoutCSRF != "provided-csrf" {
		t.Fatalf("logout CSRF = %q, want provided-csrf", service.logoutCSRF)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != -1 {
		t.Fatalf("logout cookie = %#v, want expired cookie", cookies)
	}
}

func TestReplaceContextPassesContractAndReturnsETag(t *testing.T) {
	service := &fakeService{created: auth.CreatedSession{Session: auth.Session{ContextVersion: 2}}}
	server := testServer(service, true)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/auth/session/context", strings.NewReader(`{
		"institutionId":"11","siteId":"12","firmId":"13"
	}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-token"})
	request.Header.Set("X-CSRF-Token", "provided-csrf")
	request.Header.Set("If-Match", `"1"`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("ETag") != `"2"` {
		t.Fatalf("ETag = %q, want %q", response.Header().Get("ETag"), `"2"`)
	}
	if service.replaceCalls != 1 || service.replaceCSRF != "provided-csrf" {
		t.Fatalf("ReplaceContext() calls = %d, CSRF = %q", service.replaceCalls, service.replaceCSRF)
	}
	want := auth.ContextRequest{InstitutionID: 11, SiteID: 12, FirmID: 13, Version: 1}
	if service.replaceRequest != want {
		t.Fatalf("ReplaceContext() request = %#v, want %#v", service.replaceRequest, want)
	}
}

func TestReplaceContextRejectsInvalidIfMatchBeforeCallingService(t *testing.T) {
	tests := []struct {
		name    string
		ifMatch string
	}{
		{name: "missing"},
		{name: "unquoted", ifMatch: "1"},
		{name: "weak ETag", ifMatch: `W/"1"`},
		{name: "zero", ifMatch: `"0"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeService{}
			server := testServer(service, true)
			request := httptest.NewRequest(http.MethodPut, "/api/v1/auth/session/context", strings.NewReader(`{
				"institutionId":"11","siteId":"12","firmId":"13"
			}`))
			request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-token"})
			request.Header.Set("If-Match", test.ifMatch)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
			}
			if service.replaceCalls != 0 {
				t.Fatalf("ReplaceContext() calls = %d, want 0", service.replaceCalls)
			}
		})
	}
}

func TestReplaceContextMapsAuthorizationAndConflictErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "forbidden", err: auth.ErrForbidden, status: http.StatusForbidden, code: "forbidden"},
		{name: "conflict", err: auth.ErrConflict, status: http.StatusConflict, code: "context_conflict"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeService{replaceErr: test.err}
			server := testServer(service, true)
			request := httptest.NewRequest(http.MethodPut, "/api/v1/auth/session/context", strings.NewReader(`{
				"institutionId":"11","siteId":"12","firmId":"13"
			}`))
			request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-token"})
			request.Header.Set("X-CSRF-Token", "provided-csrf")
			request.Header.Set("If-Match", `"1"`)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("problem body = %s, want code %q", response.Body.String(), test.code)
			}
		})
	}
}

func testServer(service Service, secure bool) http.Handler {
	mux := http.NewServeMux()
	NewHandler(service, secure, 12*time.Hour).Register(mux)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httpx.Middleware(logger, mux)
}

func TestWriteSessionErrorMappings(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{auth.ErrUnauthenticated, http.StatusUnauthorized},
		{auth.ErrCSRF, http.StatusForbidden},
		{auth.ErrForbidden, http.StatusForbidden},
		{auth.ErrConflict, http.StatusConflict},
		{errors.New("database"), http.StatusInternalServerError},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodGet, "/", bytes.NewReader(nil))
		request = request.WithContext(context.WithValue(request.Context(), struct{}{}, "unused"))
		response := httptest.NewRecorder()
		(&Handler{}).writeSessionError(response, request, test.err)
		if response.Code != test.status {
			t.Fatalf("error %v status = %d, want %d", test.err, response.Code, test.status)
		}
	}
}
