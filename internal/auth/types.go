package auth

import (
	"errors"
	"time"
)

var (
	// ErrAuthenticationFailed is deliberately generic to prevent account enumeration.
	ErrAuthenticationFailed = errors.New("authentication failed")
	// ErrUnauthenticated indicates that a session is missing, expired, or revoked.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrForbidden indicates an authenticated authorization failure.
	ErrForbidden = errors.New("forbidden")
	// ErrCSRF indicates a failed anti-CSRF check.
	ErrCSRF = errors.New("csrf validation failed")
	// ErrConflict indicates an optimistic concurrency conflict.
	ErrConflict = errors.New("conflict")
)

// Reference is a stable identifier and display name.
type Reference struct {
	ID   int64
	Name string
}

// InstitutionOption is a safe anonymous login choice.
type InstitutionOption struct {
	Institution Reference
	Sites       []Reference
}

// LoginRequest contains untrusted local-login input.
type LoginRequest struct {
	Username      string
	Password      string
	InstitutionID int64
	SiteID        int64
	FirmID        *int64
}

// ContextRequest contains an authenticated context replacement.
type ContextRequest struct {
	InstitutionID int64
	SiteID        int64
	FirmID        int64
	Version       int64
}

// RequestMetadata contains non-secret audit context.
type RequestMetadata struct {
	CorrelationID string
	SourceIPClass string
}

// OperationAuthorizationRequest describes one authenticated, CSRF-protected permission check.
type OperationAuthorizationRequest struct {
	Token           string
	CSRFToken       string
	Permission      string
	DeniedEventType string
	Metadata        RequestMetadata
}

// OperationPrincipal is the server-validated identity and clinical context for one operation.
type OperationPrincipal struct {
	UserID         int64
	SessionID      int64
	InstitutionID  int64
	SiteID         int64
	FirmID         int64
	ContextVersion int64
}

// User is the authenticated user's safe representation.
type User struct {
	ID          int64
	DisplayName string
}

// UserContext is the complete clinical working context.
type UserContext struct {
	Institution Reference
	Site        Reference
	Firm        Reference
}

// Session is the safe session representation returned by the API.
type Session struct {
	User              User
	Context           UserContext
	Permissions       []string
	CSRFToken         string
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	ContextVersion    int64
}

// CreatedSession includes the opaque credential that is written only to the cookie.
type CreatedSession struct {
	Session Session
	Token   string
}
