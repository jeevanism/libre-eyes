package episodeshttp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/episodes"
	"github.com/jeevanism/visionopus/internal/platform/httpx"
)

type fakeService struct {
	authorizeErr  error
	operation     string
	csrfToken     string
	createCalls   int
	listCalls     int
	getCalls      int
	activateCalls int
	closeCalls    int
	reopenCalls   int
	listRequest   episodes.ListRequest
	lifecycle     episodes.LifecycleRequest
	createResult  episodes.Episode
	listResult    episodes.EpisodePage
	serviceErr    error
}

func (f *fakeService) Authorize(_ context.Context, _, csrf string, permission string, _ auth.RequestMetadata) (episodes.Authorization, error) {
	f.operation = permission
	f.csrfToken = csrf
	return episodes.Authorization{}, f.authorizeErr
}
func (f *fakeService) Create(_ context.Context, _ episodes.Authorization, _ string, _ episodes.CreateRequest) (episodes.Episode, error) {
	f.createCalls++
	return f.createResult, f.serviceErr
}
func (f *fakeService) List(_ context.Context, _ episodes.Authorization, request episodes.ListRequest) (episodes.EpisodePage, error) {
	f.listCalls++
	f.listRequest = request
	return f.listResult, f.serviceErr
}
func (f *fakeService) Get(_ context.Context, _ episodes.Authorization, _ string) (episodes.Episode, error) {
	f.getCalls++
	return f.createResult, f.serviceErr
}
func (f *fakeService) Activate(_ context.Context, _ episodes.Authorization, request episodes.LifecycleRequest) (episodes.Episode, error) {
	f.activateCalls++
	f.lifecycle = request
	return f.createResult, f.serviceErr
}
func (f *fakeService) Close(_ context.Context, _ episodes.Authorization, request episodes.LifecycleRequest) (episodes.Episode, error) {
	f.closeCalls++
	f.lifecycle = request
	return f.createResult, f.serviceErr
}
func (f *fakeService) Reopen(_ context.Context, _ episodes.Authorization, request episodes.LifecycleRequest) (episodes.Episode, error) {
	f.reopenCalls++
	f.lifecycle = request
	return f.createResult, f.serviceErr
}

func TestListMapsScopeBoundPageAndOpaqueCursor(t *testing.T) {
	next := "opaque.cursor"
	service := &fakeService{listResult: episodes.EpisodePage{Items: []episodes.Episode{{
		ID: "11111111-1111-4111-8111-111111111111", PatientID: "22222222-2222-4222-8222-222222222222",
		Status: episodes.StatusActive, Version: 3,
	}}, NextCursor: &next}}
	response := serve(t, service, http.MethodGet, "/api/v1/patients/22222222-2222-4222-8222-222222222222/episodes?limit=20&cursor=opaque.cursor", "")
	if response.Code != http.StatusOK || service.operation != episodes.PermissionRead || service.listCalls != 1 {
		t.Fatalf("status=%d operation=%q list=%d body=%s", response.Code, service.operation, service.listCalls, response.Body.String())
	}
	if service.listRequest.PatientID != "22222222-2222-4222-8222-222222222222" || service.listRequest.Limit != 20 || service.listRequest.Cursor != "opaque.cursor" {
		t.Fatalf("list request = %#v", service.listRequest)
	}
	for _, expected := range []string{`"nextCursor":"opaque.cursor"`, `"supportServices":false`, `"changeTracker":false`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("response missing %s: %s", expected, response.Body.String())
		}
	}
	assertNoStore(t, response)
}

func TestCreateAuthorizesBeforeParsingAndRejectsUnknownFields(t *testing.T) {
	service := &fakeService{}
	response := serve(t, service, http.MethodPost, "/api/v1/patients/22222222-2222-4222-8222-222222222222/episodes", `{"status":"open","unexpected":true}`)
	if response.Code != http.StatusBadRequest || service.operation != episodes.PermissionCreate || service.createCalls != 0 {
		t.Fatalf("status=%d operation=%q creates=%d body=%s", response.Code, service.operation, service.createCalls, response.Body.String())
	}
}

func TestUpdateUsesLeastPrivilegePermissionForLifecycleAction(t *testing.T) {
	service := &fakeService{createResult: episodes.Episode{ID: "11111111-1111-4111-8111-111111111111", PatientID: "22222222-2222-4222-8222-222222222222", Status: episodes.StatusActive, Version: 2}}
	response := serve(t, service, http.MethodPatch, "/api/v1/episodes/11111111-1111-4111-8111-111111111111", `{"expectedVersion":1,"lifecycleAction":"reopen","reopenReason":"Corrected closure"}`)
	if response.Code != http.StatusOK || service.operation != episodes.PermissionReopen || service.reopenCalls != 1 || service.lifecycle.Reason != "Corrected closure" {
		t.Fatalf("status=%d operation=%q reopen=%d lifecycle=%#v body=%s", response.Code, service.operation, service.reopenCalls, service.lifecycle, response.Body.String())
	}
}

func TestUpdateDoesNotAuthorizeUnsupportedAction(t *testing.T) {
	service := &fakeService{}
	response := serve(t, service, http.MethodPatch, "/api/v1/episodes/11111111-1111-4111-8111-111111111111", `{"expectedVersion":1,"lifecycleAction":"delete"}`)
	if response.Code != http.StatusBadRequest || service.operation != "" || service.activateCalls+service.closeCalls+service.reopenCalls != 0 {
		t.Fatalf("status=%d operation=%q body=%s", response.Code, service.operation, response.Body.String())
	}
}

func TestUpdateWithoutSessionDoesNotParseBody(t *testing.T) {
	service := &fakeService{}
	mux := http.NewServeMux()
	NewHandler(service, true).Register(mux)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/episodes/11111111-1111-4111-8111-111111111111", strings.NewReader(`{invalid`))
	response := httptest.NewRecorder()
	httpx.Middleware(slog.New(slog.NewTextHandler(io.Discard, nil)), mux).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || service.operation != "" {
		t.Fatalf("status=%d operation=%q body=%s", response.Code, service.operation, response.Body.String())
	}
}

func TestGetUsesReadPermissionAndMapsHeader(t *testing.T) {
	service := &fakeService{createResult: episodes.Episode{
		ID: "11111111-1111-4111-8111-111111111111", PatientID: "22222222-2222-4222-8222-222222222222",
		Status: episodes.StatusClosed, Version: 2,
	}}
	response := serve(t, service, http.MethodGet, "/api/v1/episodes/11111111-1111-4111-8111-111111111111", "")
	if response.Code != http.StatusOK || service.operation != episodes.PermissionRead || service.getCalls != 1 {
		t.Fatalf("status=%d operation=%q gets=%d body=%s", response.Code, service.operation, service.getCalls, response.Body.String())
	}
}

func TestBoundaryErrorsAreGenericAndNonDisclosing(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"unauthenticated", auth.ErrUnauthenticated, http.StatusUnauthorized, "unauthenticated"},
		{"csrf", auth.ErrCSRF, http.StatusForbidden, "forbidden"},
		{"conflict", auth.ErrConflict, http.StatusConflict, "conflict"},
		{"invalid transition", episodes.ErrInvalidTransition, http.StatusBadRequest, "invalid_request"},
		{"unavailable", episodes.ErrUnavailable, http.StatusServiceUnavailable, "episode_unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeService{authorizeErr: test.err}
			response := serve(t, service, http.MethodGet, "/api/v1/patients/22222222-2222-4222-8222-222222222222/episodes", "")
			if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) || strings.Contains(response.Body.String(), "22222222") {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			assertNoStore(t, response)
		})
	}
}

func TestListRejectsInvalidLimitWithoutCallingService(t *testing.T) {
	for _, value := range []string{"invalid", "0", "-1", "101"} {
		t.Run(value, func(t *testing.T) {
			service := &fakeService{}
			response := serve(t, service, http.MethodGet, "/api/v1/patients/22222222-2222-4222-8222-222222222222/episodes?limit="+value, "")
			if response.Code != http.StatusBadRequest || service.listCalls != 0 {
				t.Fatalf("status=%d lists=%d body=%s", response.Code, service.listCalls, response.Body.String())
			}
		})
	}
}

func TestReadWithoutCSRFIsForbidden(t *testing.T) {
	service := &fakeService{authorizeErr: auth.ErrCSRF}
	mux := http.NewServeMux()
	NewHandler(service, true).Register(mux)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/patients/22222222-2222-4222-8222-222222222222/episodes", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-session"})
	response := httptest.NewRecorder()
	httpx.Middleware(slog.New(slog.NewTextHandler(io.Discard, nil)), mux).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || service.csrfToken != "" || service.listCalls != 0 {
		t.Fatalf("status=%d csrf=%q lists=%d body=%s", response.Code, service.csrfToken, service.listCalls, response.Body.String())
	}
}

func serve(t *testing.T, service *fakeService, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	NewHandler(service, true).Register(mux)
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.RemoteAddr = "127.0.0.1:1234"
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-session"})
	request.Header.Set("X-CSRF-Token", "synthetic-csrf-token")
	response := httptest.NewRecorder()
	httpx.Middleware(slog.New(slog.NewTextHandler(io.Discard, nil)), mux).ServeHTTP(response, request)
	return response
}

func assertNoStore(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control=%q, want no-store", response.Header().Get("Cache-Control"))
	}
}

func TestWriteErrorHandlesDeadline(t *testing.T) {
	service := &fakeService{authorizeErr: context.DeadlineExceeded}
	response := serve(t, service, http.MethodGet, "/api/v1/patients/22222222-2222-4222-8222-222222222222/episodes", "")
	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestDecodeJSONRejectsMoreThanOneValue(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"status":"open"} {}`))
	response := httptest.NewRecorder()
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(response, request, &body); !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("decodeJSON() error = %v", err)
	}
}

var _ = time.Second
