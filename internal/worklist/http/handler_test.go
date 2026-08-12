package worklisthttp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/platform/httpx"
	"github.com/jeevanism/visionopus/internal/worklist"
)

type fakeService struct {
	authorizeErr  error
	listResult    []worklist.Ticket
	commandResult worklist.Ticket
	commandErr    error
	command       worklist.CommandRequest
}

func (f *fakeService) Authorize(_ context.Context, _, _ string, _ auth.RequestMetadata) (worklist.Authorization, error) {
	return worklist.Authorization{}, f.authorizeErr
}
func (f *fakeService) List(_ context.Context, _ worklist.Authorization) ([]worklist.Ticket, error) {
	return f.listResult, f.commandErr
}
func (f *fakeService) Command(_ context.Context, _ worklist.Authorization, request worklist.CommandRequest) (worklist.Ticket, error) {
	f.command = request
	return f.commandResult, f.commandErr
}

func TestListMapsMinimumDisclosureTickets(t *testing.T) {
	service := &fakeService{listResult: []worklist.Ticket{{ID: "77777777-7777-4777-8777-777777777777", SyntheticPatientLabel: "Synthetic queue patient A", Status: worklist.StatusWaiting, Version: 1}}}
	response := serve(t, service, http.MethodGet, "/api/v1/development/clinic-flow/tickets", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"syntheticPatientLabel":"Synthetic queue patient A"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	for _, excluded := range []string{"patientId", "institutionId", "firmId"} {
		if strings.Contains(response.Body.String(), excluded) {
			t.Fatalf("response disclosed %s: %s", excluded, response.Body.String())
		}
	}
}

func TestCommandUsesRouteActionAndExpectedVersion(t *testing.T) {
	service := &fakeService{commandResult: worklist.Ticket{ID: "77777777-7777-4777-8777-777777777777", SyntheticPatientLabel: "Synthetic queue patient A", Status: worklist.StatusArrived, Version: 2}}
	response := serve(t, service, http.MethodPost, "/api/v1/development/clinic-flow/tickets/77777777-7777-4777-8777-777777777777/arrive", `{"expectedVersion":1}`)
	if response.Code != http.StatusOK || service.command.Command != worklist.CommandArrive || service.command.ExpectedVersion != 1 {
		t.Fatalf("status=%d request=%#v body=%s", response.Code, service.command, response.Body.String())
	}
}

func TestCommandDoesNotDiscloseHiddenTicket(t *testing.T) {
	service := &fakeService{commandErr: worklist.ErrNotFound}
	response := serve(t, service, http.MethodPost, "/api/v1/development/clinic-flow/tickets/77777777-7777-4777-8777-777777777777/complete", `{"expectedVersion":1}`)
	if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "in_progress") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestCommandMapsStateConflictWithoutDisclosingTicketState(t *testing.T) {
	service := &fakeService{commandErr: worklist.ErrConflict}
	response := serve(t, service, http.MethodPost, "/api/v1/development/clinic-flow/tickets/77777777-7777-4777-8777-777777777777/claim", `{"expectedVersion":2}`)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"conflict"`) || strings.Contains(response.Body.String(), "arrived") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestCommandRequiresSessionBeforeParsing(t *testing.T) {
	service := &fakeService{}
	mux := http.NewServeMux()
	NewHandler(service, true).Register(mux)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/development/clinic-flow/tickets/77777777-7777-4777-8777-777777777777/arrive", strings.NewReader("{invalid"))
	response := httptest.NewRecorder()
	httpx.Middleware(slog.New(slog.NewTextHandler(io.Discard, nil)), mux).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
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
