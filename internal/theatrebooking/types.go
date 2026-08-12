// Package theatrebooking implements the development-only synthetic theatre board.
package theatrebooking

import "errors"

const PermissionManage = "theatre.development_booking.manage"

var (
	ErrInvalidRequest = errors.New("invalid development theatre-booking request")
	ErrNotFound       = errors.New("development theatre-booking resource not found")
	ErrConflict       = errors.New("development theatre-booking conflict")
)

type Status string

const (
	StatusWaiting   Status = "waiting"
	StatusScheduled Status = "scheduled"
	StatusCancelled Status = "cancelled"
)

type Command string

const (
	CommandSchedule   Command = "schedule"
	CommandReschedule Command = "reschedule"
	CommandCancel     Command = "cancel"
)

// Session is a fixed-capacity synthetic theatre session.
type Session struct {
	ID                 string
	SyntheticRoomLabel string
	StartsAt           string
	EndsAt             string
	CapacityMinutes    int
	AllocatedMinutes   int
	Version            int64
}

// BookingRequest intentionally contains no patient, procedure, or clinical data.
type BookingRequest struct {
	ID                       string
	SyntheticLabel           string
	RequestedDurationMinutes int
	Status                   Status
	AssignedSessionID        *string
	Version                  int64
}

type WhiteboardEntry struct {
	RequestID                string
	SyntheticLabel           string
	RequestedDurationMinutes int
}

type WhiteboardGroup struct {
	SessionID          string
	SyntheticRoomLabel string
	Entries            []WhiteboardEntry
}

type Board struct {
	DevelopmentOnly bool
	SessionDate     string
	TheatreSessions []Session
	BookingRequests []BookingRequest
	Whiteboard      []WhiteboardGroup
}

type CommandRequest struct {
	RequestID       string
	ExpectedVersion int64
	TargetSessionID string
	Command         Command
}

type ConflictError struct{ Reason string }

func (e ConflictError) Error() string        { return e.Reason }
func (e ConflictError) Is(target error) bool { return target == ErrConflict }
