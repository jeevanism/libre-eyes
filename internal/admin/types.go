package admin

import "errors"

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
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Active      bool   `json:"active"`
	Version     int64  `json:"version"`
}

type Contexts struct {
	Institution Reference   `json:"institution"`
	Sites       []Reference `json:"sites"`
	Firms       []Reference `json:"firms"`
}

type Reference struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type Setting struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int64  `json:"version"`
}
type AuditEvent struct {
	Command       string   `json:"command"`
	TargetType    string   `json:"targetType"`
	ChangedFields []string `json:"changedFields"`
	Outcome       string   `json:"outcome"`
}

type UserCommand struct {
	PublicID        string
	ExpectedVersion int64
	Active          bool
}
type SettingUpdate struct {
	Key, Value      string
	ExpectedVersion int64
}
