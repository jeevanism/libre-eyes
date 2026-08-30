package theatrebookinghttp

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/auth"
	"github.com/jeevanism/libre-eyes/internal/theatrebooking"
)

type fakeService struct {
	readCalls, commandCalls int
	commandInvocations      int
	board                   theatrebooking.Board
	item                    theatrebooking.BookingRequest
	readErr                 error
	commandAuthorizationErr error
	commandErr              error
	command                 theatrebooking.CommandRequest
}

func (f *fakeService) AuthorizeRead(_ context.Context, _ string, _ auth.RequestMetadata) (theatrebooking.Authorization, error) {
	f.readCalls++
	return theatrebooking.Authorization{}, f.readErr
}
func (f *fakeService) AuthorizeCommand(_ context.Context, _, _ string, _ auth.RequestMetadata) (theatrebooking.Authorization, error) {
	f.commandCalls++
	return theatrebooking.Authorization{}, f.commandAuthorizationErr
}
func (f *fakeService) Board(_ context.Context, _ theatrebooking.Authorization, _, _ string) (theatrebooking.Board, error) {
	return f.board, nil
}
func (f *fakeService) Command(_ context.Context, _ theatrebooking.Authorization, request theatrebooking.CommandRequest) (theatrebooking.BookingRequest, error) {
	f.commandInvocations++
	f.command = request
	return f.item, f.commandErr
}

func TestCommandRequiresSessionAndCSRFBeforeService(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service, false)
	mux := http.NewServeMux()
	handler.Register(mux)
	path := "/api/v1/development/theatre-booking/requests/d1111111-1111-4111-8111-111111111111/schedule"

	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"expectedVersion":1,"targetSessionId":"b1111111-1111-4111-8111-111111111111"}`))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || service.commandCalls != 0 {
		t.Fatalf("missing session status/calls = %d/%d, want 401/0", response.Code, service.commandCalls)
	}

	request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"expectedVersion":1,"targetSessionId":"b1111111-1111-4111-8111-111111111111"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "token"})
	service.commandAuthorizationErr = auth.ErrCSRF
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d, want 403", response.Code)
	}
}

func TestCommandMapsConflictAndPassesVersionedRequest(t *testing.T) {
	service := &fakeService{commandErr: theatrebooking.ConflictError{Reason: "stale_version"}}
	handler := NewHandler(service, false)
	mux := http.NewServeMux()
	handler.Register(mux)
	path := "/api/v1/development/theatre-booking/requests/d1111111-1111-4111-8111-111111111111/schedule"
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"expectedVersion":2,"targetSessionId":"b1111111-1111-4111-8111-111111111111"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "token"})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d, want 409", response.Code)
	}
	if service.command.Command != theatrebooking.CommandSchedule || service.command.ExpectedVersion != 2 || service.command.RequestID != "d1111111-1111-4111-8111-111111111111" {
		t.Fatalf("command = %#v", service.command)
	}
}

func TestCommandRejectsInvalidJSONWithoutCallingService(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service, false)
	mux := http.NewServeMux()
	handler.Register(mux)
	path := "/api/v1/development/theatre-booking/requests/d1111111-1111-4111-8111-111111111111/cancel"
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"expectedVersion":"two"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "token"})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || service.commandInvocations != 0 {
		t.Fatalf("invalid JSON status/calls = %d/%d, want 400/0", response.Code, service.commandInvocations)
	}
}

func TestBoardDoesNotRequireCSRFHeader(t *testing.T) {
	service := &fakeService{board: theatrebooking.Board{DevelopmentOnly: true, SessionDate: "2026-08-10"}}
	handler := NewHandler(service, false)
	mux := http.NewServeMux()
	handler.Register(mux)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/development/theatre-booking/board", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "token"})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.readCalls != 1 {
		t.Fatalf("status=%d readCalls=%d, want 200/1", response.Code, service.readCalls)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
}
