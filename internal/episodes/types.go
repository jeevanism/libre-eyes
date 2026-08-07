// Package episodes implements the approved episode lifecycle boundary.
package episodes

import (
	"errors"
	"time"
)

const (
	// PermissionRead permits scoped episode-header disclosure.
	PermissionRead = "episode.read"
	// PermissionCreate permits creation in the server-derived clinical context.
	PermissionCreate = "episode.create"
	// PermissionUpdate permits activation and closure of scoped episodes.
	PermissionUpdate = "episode.update"
	// PermissionReopen permits reopening a scoped closed episode with a reason.
	PermissionReopen = "episode.reopen"

	permissionRead   = PermissionRead
	permissionCreate = PermissionCreate
	permissionUpdate = PermissionUpdate
	permissionReopen = PermissionReopen

	defaultPageSize = 25
	maximumPageSize = 100
)

var (
	// ErrInvalidRequest is returned for malformed requests and unsupported state changes.
	ErrInvalidRequest = errors.New("invalid episode request")
	// ErrInvalidTransition is returned when the requested lifecycle transition is not approved.
	ErrInvalidTransition = errors.New("invalid episode lifecycle transition")
	// ErrUnavailable means the required data or audit transaction could not complete safely.
	ErrUnavailable = errors.New("episode service unavailable")
)

// Status is the allowed episode lifecycle state.
type Status string

const (
	StatusOpen   Status = "open"
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

// Episode is the API-safe episode header. The legacy flags are read-only.
type Episode struct {
	ID              string
	PatientID       string
	Status          Status
	StartedAt       *time.Time
	EndedAt         *time.Time
	SupportServices bool
	ChangeTracker   bool
	Version         int64
}

// CreateRequest contains the only approved client-controlled create field.
type CreateRequest struct {
	Status Status
}

// ListRequest bounds a patient timeline query.
type ListRequest struct {
	PatientID string
	Limit     int
	Cursor    string
}

// EpisodePage is a bounded, scope-bound timeline page.
type EpisodePage struct {
	Items      []Episode
	NextCursor *string
}

// LifecycleRequest identifies a resource and proves the caller observed its current version.
type LifecycleRequest struct {
	EpisodeID       string
	ExpectedVersion int64
	Reason          string
}
