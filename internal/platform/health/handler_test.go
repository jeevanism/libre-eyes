package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubPinger struct {
	err error
}

func (p stubPinger) Ping(context.Context) error {
	return p.err
}

func TestLiveness(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(stubPinger{}).Register(mux)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestReadinessReportsDatabaseFailure(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(stubPinger{err: errors.New("database unavailable")}).Register(mux)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
