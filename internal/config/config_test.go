package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("LIBREEYES_ENV", "development")
	t.Setenv("LIBREEYES_HTTP_ADDR", ":8080")
	t.Setenv("LIBREEYES_DATABASE_URL", "postgres://example")
	t.Setenv("LIBREEYES_COOKIE_SECURE", "false")
	t.Setenv("LIBREEYES_CSRF_KEY", "Jk4j37boZHsZUX1pMXgbIJ67VvHZTQCJE8ziNdJHh9E")
	t.Setenv("LIBREEYES_SESSION_IDLE_TIMEOUT", "15m")
	t.Setenv("LIBREEYES_SESSION_ABSOLUTE_TIMEOUT", "12h")
	t.Setenv("LIBREEYES_SHUTDOWN_TIMEOUT", "10s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SessionIdleTimeout != 15*time.Minute {
		t.Fatalf("SessionIdleTimeout = %s, want 15m", cfg.SessionIdleTimeout)
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure = true, want false in configured development environment")
	}
}

func TestLoadRejectsInsecureProductionCookie(t *testing.T) {
	t.Setenv("LIBREEYES_ENV", "production")
	t.Setenv("LIBREEYES_COOKIE_SECURE", "false")
	t.Setenv("LIBREEYES_CSRF_KEY", "Jk4j37boZHsZUX1pMXgbIJ67VvHZTQCJE8ziNdJHh9E")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want insecure production cookie error")
	}
}

func TestLoadRejectsIdleTimeoutBeyondAbsoluteTimeout(t *testing.T) {
	t.Setenv("LIBREEYES_CSRF_KEY", "Jk4j37boZHsZUX1pMXgbIJ67VvHZTQCJE8ziNdJHh9E")
	t.Setenv("LIBREEYES_SESSION_IDLE_TIMEOUT", "13h")
	t.Setenv("LIBREEYES_SESSION_ABSOLUTE_TIMEOUT", "12h")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want timeout ordering error")
	}
}

func TestLoadRequiresCSRFKey(t *testing.T) {
	t.Setenv("LIBREEYES_CSRF_KEY", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want missing CSRF key error")
	}
}
