// Package branding owns tenant-scoped presentation profiles.
package branding

import (
	"errors"
	"time"
)

const (
	permissionRead   = "admin.development.read"
	permissionManage = "admin.development.manage"
)

var (
	ErrInvalidRequest = errors.New("invalid branding request")
	ErrForbidden      = errors.New("branding permission denied")
	ErrConflict       = errors.New("branding profile changed")
	ErrNoDraft        = errors.New("branding draft not found")
	ErrNotFound       = errors.New("branding profile version not found")
)

// Colors contains the bounded semantic presentation tokens customers may configure.
type Colors struct {
	Primary         string `json:"primary"`
	PrimaryHover    string `json:"primaryHover"`
	SelectedSurface string `json:"selectedSurface"`
	Focus           string `json:"focus"`
}

// Profile is one immutable identity plus a mutable optimistic row version.
type Profile struct {
	InstitutionID    int64  `json:"institutionId,omitempty"`
	ProfileVersion   int64  `json:"profileVersion"`
	RowVersion       int64  `json:"rowVersion"`
	Status           string `json:"status"`
	Source           string `json:"source"`
	OrganizationName string `json:"organizationName"`
	ShortName        string `json:"shortName"`
	BrowserTitle     string `json:"browserTitle"`
	Colors           Colors `json:"colors"`
}

// State is the complete administration representation for one institution.
type State struct {
	Effective Profile      `json:"effective"`
	Published *Profile     `json:"published,omitempty"`
	Draft     *Profile     `json:"draft,omitempty"`
	History   []Summary    `json:"history"`
	Audit     []AuditEvent `json:"audit"`
}

// Summary is a bounded profile history entry.
type Summary struct {
	ProfileVersion int64      `json:"profileVersion"`
	RowVersion     int64      `json:"rowVersion"`
	Status         string     `json:"status"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	PublishedAt    *time.Time `json:"publishedAt,omitempty"`
}

// AuditEvent is an append-only record of a branding administration command.
type AuditEvent struct {
	Command          string         `json:"command"`
	ProfileVersion   int64          `json:"profileVersion"`
	ActorDisplayName string         `json:"actorDisplayName"`
	Before           map[string]any `json:"before"`
	After            map[string]any `json:"after"`
	CorrelationID    string         `json:"correlationId"`
	CreatedAt        time.Time      `json:"createdAt"`
}

// DraftInput contains all editable profile fields and the expected draft row version.
type DraftInput struct {
	OrganizationName string `json:"organizationName"`
	ShortName        string `json:"shortName"`
	BrowserTitle     string `json:"browserTitle"`
	Colors           Colors `json:"colors"`
	ExpectedVersion  int64  `json:"expectedVersion"`
}

// VersionCommand identifies the optimistic row version used by publish and rollback.
type VersionCommand struct {
	ExpectedVersion      int64 `json:"expectedVersion"`
	TargetProfileVersion int64 `json:"targetProfileVersion,omitempty"`
}
