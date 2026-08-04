package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("VISIONOPUS_ENV", "development")
	t.Setenv("VISIONOPUS_HTTP_ADDR", ":8080")
	t.Setenv("VISIONOPUS_DATABASE_URL", "postgres://example")
	t.Setenv("VISIONOPUS_COOKIE_SECURE", "false")
	t.Setenv("VISIONOPUS_CSRF_KEY", "Jk4j37boZHsZUX1pMXgbIJ67VvHZTQCJE8ziNdJHh9E")
	t.Setenv("VISIONOPUS_SESSION_IDLE_TIMEOUT", "15m")
	t.Setenv("VISIONOPUS_SESSION_ABSOLUTE_TIMEOUT", "12h")
	t.Setenv("VISIONOPUS_SHUTDOWN_TIMEOUT", "10s")

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
	t.Setenv("VISIONOPUS_ENV", "production")
	t.Setenv("VISIONOPUS_COOKIE_SECURE", "false")
	t.Setenv("VISIONOPUS_CSRF_KEY", "Jk4j37boZHsZUX1pMXgbIJ67VvHZTQCJE8ziNdJHh9E")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want insecure production cookie error")
	}
}

func TestLoadRejectsIdleTimeoutBeyondAbsoluteTimeout(t *testing.T) {
	t.Setenv("VISIONOPUS_CSRF_KEY", "Jk4j37boZHsZUX1pMXgbIJ67VvHZTQCJE8ziNdJHh9E")
	t.Setenv("VISIONOPUS_SESSION_IDLE_TIMEOUT", "13h")
	t.Setenv("VISIONOPUS_SESSION_ABSOLUTE_TIMEOUT", "12h")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want timeout ordering error")
	}
}

func TestLoadRequiresCSRFKey(t *testing.T) {
	t.Setenv("VISIONOPUS_CSRF_KEY", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want missing CSRF key error")
	}
}
