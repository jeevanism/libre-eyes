package admin

import (
	"errors"
	"time"
)

const (
	PermissionRead   = "admin.development.read"
	PermissionManage = "admin.development.manage"
)

var (
	ErrInvalidRequest = errors.New("invalid administration request")
	ErrForbidden      = errors.New("administration permission denied")
	ErrConflict       = errors.New("administration row changed")
)

type User struct {
	ID          string      `json:"id"`
	Username    string      `json:"username"`
	DisplayName string      `json:"displayName"`
	Role        string      `json:"role"`
	Active      bool        `json:"active"`
	Version     int64       `json:"version"`
	Permissions []string    `json:"permissions"`
	Sites       []Reference `json:"sites"`
	Firms       []Reference `json:"firms"`
}

type Contexts struct {
	Institution Reference   `json:"institution"`
	Sites       []Reference `json:"sites"`
	Firms       []Reference `json:"firms"`
}

type Reference struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Active  bool   `json:"active,omitempty"`
	Version int64  `json:"version,omitempty"`
}
type Setting struct {
	Key     string       `json:"key"`
	Value   string       `json:"value"`
	Version int64        `json:"version"`
	Scope   SettingScope `json:"scope"`
	Source  string       `json:"source"`
}
type AuditEvent struct {
	ActorUserID       int64     `json:"actorUserId"`
	ActorDisplayName  string    `json:"actorDisplayName"`
	Command           string    `json:"command"`
	TargetType        string    `json:"targetType"`
	TargetPublicID    *string   `json:"targetPublicId,omitempty"`
	TargetKey         *string   `json:"targetKey,omitempty"`
	TargetDisplayName *string   `json:"targetDisplayName,omitempty"`
	ChangedFields     []string  `json:"changedFields"`
	Outcome           string    `json:"outcome"`
	CorrelationID     string    `json:"correlationId"`
	CreatedAt         time.Time `json:"createdAt"`
}

type UserCommand struct {
	PublicID        string
	ExpectedVersion int64
	Active          bool
}
type UserUpsert struct {
	PublicID        string
	Username        string
	DisplayName     string
	Password        string
	Role            string
	SiteIDs         []int64
	FirmIDs         []int64
	ExpectedVersion int64
}
type SettingUpdate struct {
	Key, Value      string
	ExpectedVersion int64
}

type Capability struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Version     int64  `json:"version"`
}

type CapabilityUpdate struct {
	Key             string
	Enabled         bool
	ExpectedVersion int64
}

type ContextUpsert struct {
	ID              int64
	Name            string
	Active          bool
	ExpectedVersion int64
}

type CatalogueItem struct {
	ID           int64  `json:"id"`
	Category     string `json:"category"`
	Code         string `json:"code"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
	DisplayOrder int    `json:"displayOrder"`
	Version      int64  `json:"version"`
}

type CatalogueUpsert struct {
	ID              int64
	Category        string
	Code            string
	DisplayName     string
	Active          bool
	DisplayOrder    int
	ExpectedVersion int64
}
