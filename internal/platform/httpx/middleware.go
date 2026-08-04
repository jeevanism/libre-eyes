// Package httpx contains shared HTTP boundary behavior.
package httpx

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"
)

type contextKey string

const correlationKey contextKey = "correlation-id"

var correlationPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

// Middleware adds correlation IDs, panic recovery, security headers, and request logs.
func Middleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if !correlationPattern.MatchString(correlationID) {
			correlationID = newCorrelationID()
		}
		w.Header().Set("X-Correlation-ID", correlationID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		r = r.WithContext(context.WithValue(r.Context(), correlationKey, correlationID))

		started := time.Now()
		response := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(r.Context(), "http panic",
					"panic", recovered,
					"stack", string(debug.Stack()),
					"correlation_id", correlationID,
				)
				if !response.wroteHeader {
					http.Error(response, "internal server error", http.StatusInternalServerError)
				}
			}
			logger.InfoContext(r.Context(), "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", response.status,
				"duration_ms", time.Since(started).Milliseconds(),
				"correlation_id", correlationID,
			)
		}()
		next.ServeHTTP(response, r)
	})
}

// CorrelationID returns the request correlation ID assigned by Middleware.
func CorrelationID(ctx context.Context) string {
	value, _ := ctx.Value(correlationKey).(string)
	return value
}

func newCorrelationID() string {
	value := make([]byte, 18)
	if _, err := rand.Read(value); err != nil {
		return "correlation-unavailable"
	}
	return base64.RawURLEncoding.EncodeToString(value)
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}
