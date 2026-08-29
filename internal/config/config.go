// Package config loads and validates runtime configuration.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultHTTPAddr               = ":8080"
	defaultDatabaseURL            = "postgres://visionopus:visionopus_dev@localhost:5432/visionopus?sslmode=disable"
	defaultSessionIdleTimeout     = 15 * time.Minute
	defaultSessionAbsoluteTimeout = 12 * time.Hour
	defaultLoginFailureLimit      = 5
	defaultSoftLockDuration       = 15 * time.Minute
	defaultShutdownTimeout        = 10 * time.Second
)

// Config contains validated process configuration.
type Config struct {
	Environment            string
	HTTPAddr               string
	StaticDir              string
	DatabaseURL            string
	CookieSecure           bool
	CSRFKey                []byte
	SessionIdleTimeout     time.Duration
	SessionAbsoluteTimeout time.Duration
	LoginFailureLimit      int
	SoftLockDuration       time.Duration
	ShutdownTimeout        time.Duration
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		Environment:            valueOrDefault("VISIONOPUS_ENV", "development"),
		HTTPAddr:               valueOrDefault("VISIONOPUS_HTTP_ADDR", defaultHTTPAddr),
		StaticDir:              os.Getenv("VISIONOPUS_STATIC_DIR"),
		DatabaseURL:            valueOrDefault("VISIONOPUS_DATABASE_URL", defaultDatabaseURL),
		SessionIdleTimeout:     defaultSessionIdleTimeout,
		SessionAbsoluteTimeout: defaultSessionAbsoluteTimeout,
		LoginFailureLimit:      defaultLoginFailureLimit,
		SoftLockDuration:       defaultSoftLockDuration,
		ShutdownTimeout:        defaultShutdownTimeout,
	}

	var err error
	if cfg.CookieSecure, err = boolValue("VISIONOPUS_COOKIE_SECURE", cfg.Environment != "development"); err != nil {
		return Config{}, err
	}
	if cfg.SessionIdleTimeout, err = durationValue("VISIONOPUS_SESSION_IDLE_TIMEOUT", defaultSessionIdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.SessionAbsoluteTimeout, err = durationValue("VISIONOPUS_SESSION_ABSOLUTE_TIMEOUT", defaultSessionAbsoluteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = durationValue("VISIONOPUS_SHUTDOWN_TIMEOUT", defaultShutdownTimeout); err != nil {
		return Config{}, err
	}
	if cfg.SoftLockDuration, err = durationValue("VISIONOPUS_SOFT_LOCK_DURATION", defaultSoftLockDuration); err != nil {
		return Config{}, err
	}
	if cfg.LoginFailureLimit, err = intValue("VISIONOPUS_LOGIN_FAILURE_LIMIT", defaultLoginFailureLimit); err != nil {
		return Config{}, err
	}
	if cfg.CSRFKey, err = secretValue("VISIONOPUS_CSRF_KEY", 32); err != nil {
		return Config{}, err
	}

	if cfg.HTTPAddr == "" {
		return Config{}, errors.New("VISIONOPUS_HTTP_ADDR must not be empty")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("VISIONOPUS_DATABASE_URL must not be empty")
	}
	if cfg.SessionIdleTimeout <= 0 || cfg.SessionAbsoluteTimeout <= 0 || cfg.SoftLockDuration <= 0 || cfg.ShutdownTimeout <= 0 {
		return Config{}, errors.New("configured timeouts must be positive")
	}
	if cfg.LoginFailureLimit < 1 {
		return Config{}, errors.New("login failure limit must be positive")
	}
	if cfg.SessionIdleTimeout > cfg.SessionAbsoluteTimeout {
		return Config{}, errors.New("session idle timeout must not exceed absolute timeout")
	}
	if cfg.Environment == "production" && !cfg.CookieSecure {
		return Config{}, errors.New("secure cookies are required in production")
	}

	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func boolValue(key string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func durationValue(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func intValue(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func secretValue(key string, minimumBytes int) ([]byte, error) {
	value := os.Getenv(key)
	if value == "" {
		return nil, fmt.Errorf("%s is required", key)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", key, err)
	}
	if len(decoded) < minimumBytes {
		return nil, fmt.Errorf("%s must contain at least %d bytes", key, minimumBytes)
	}
	return decoded, nil
}
