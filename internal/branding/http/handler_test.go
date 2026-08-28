package brandinghttp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jeevanism/visionopus/internal/auth"
	"github.com/jeevanism/visionopus/internal/branding"
)

type fakeService struct {
	publicInstitution int64
	publicProfile     branding.Profile
	authorizeWrite    bool
	draft             branding.DraftInput
}

func (f *fakeService) PublicProfile(_ context.Context, institutionID int64) (branding.Profile, error) {
	f.publicInstitution = institutionID
	return f.publicProfile, nil
}

func (f *fakeService) Authorize(_ context.Context, _, _ string, _ auth.RequestMetadata, write bool) (branding.Authorization, error) {
	f.authorizeWrite = write
	return branding.Authorization{}, nil
}

func (f *fakeService) State(context.Context, branding.Authorization) (branding.State, error) {
	return branding.State{Effective: f.publicProfile}, nil
}

func (f *fakeService) SaveDraft(_ context.Context, _ branding.Authorization, input branding.DraftInput) (branding.State, error) {
	f.draft = input
	return branding.State{Effective: f.publicProfile}, nil
}

func (f *fakeService) Publish(context.Context, branding.Authorization, branding.VersionCommand) (branding.State, error) {
	return branding.State{Effective: f.publicProfile}, nil
}

func (f *fakeService) Rollback(context.Context, branding.Authorization, branding.VersionCommand) (branding.State, error) {
	return branding.State{Effective: f.publicProfile}, nil
}

func TestPublicProfileReturnsPublishedPresentation(t *testing.T) {
	service := &fakeService{publicProfile: branding.DefaultProfile()}
	handler := NewHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	handler.Register(mux)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/presentation/branding?institutionId=42", nil)
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK || service.publicInstitution != 42 {
		t.Fatalf("response status = %d, institution = %d", response.Code, service.publicInstitution)
	}
	if cache := response.Header().Get("Cache-Control"); cache != "public, max-age=300" {
		t.Fatalf("Cache-Control = %q", cache)
	}
}

func TestSaveDraftRequiresSessionAndForwardsBoundedInput(t *testing.T) {
	service := &fakeService{publicProfile: branding.DefaultProfile()}
	handler := NewHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	handler.Register(mux)
	body := `{"organizationName":"Velox Group EyeCare","shortName":"Velox","browserTitle":"Velox clinical workspace","colors":{"primary":"#17543f","primaryHover":"#103d2e","selectedSurface":"#dcefe5","focus":"#005fcc"},"expectedVersion":0}`
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/branding/draft", strings.NewReader(body))
	request.AddCookie(&http.Cookie{Name: "visionopus_session", Value: "opaque"})
	request.Header.Set("X-CSRF-Token", "csrf")
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !service.authorizeWrite {
		t.Fatalf("response status = %d, write authorization = %v", response.Code, service.authorizeWrite)
	}
	if service.draft.OrganizationName != "Velox Group EyeCare" || service.draft.Colors.Primary != "#17543f" {
		t.Fatalf("draft = %#v", service.draft)
	}
}

func TestSaveDraftRejectsUnknownFields(t *testing.T) {
	service := &fakeService{publicProfile: branding.DefaultProfile()}
	handler := NewHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	handler.Register(mux)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/branding/draft", strings.NewReader(`{"arbitraryCss":"body{display:none}"}`))
	request.AddCookie(&http.Cookie{Name: "visionopus_session", Value: "opaque"})
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestSaveDraftRejectsTrailingContent(t *testing.T) {
	service := &fakeService{publicProfile: branding.DefaultProfile()}
	handler := NewHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	handler.Register(mux)
	body := `{"organizationName":"Velox Group EyeCare"} trailing`
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/branding/draft", strings.NewReader(body))
	request.AddCookie(&http.Cookie{Name: "visionopus_session", Value: "opaque"})
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
