// Package worklist implements the development-only clinic-flow demonstration.
package worklist

import "errors"

const PermissionManage = "worklist.development_flow.manage"

var (
	ErrInvalidRequest    = errors.New("invalid development clinic-flow request")
	ErrNotFound          = errors.New("development clinic-flow ticket not found")
	ErrConflict          = errors.New("development clinic-flow ticket changed")
	ErrInvalidTransition = errors.New("invalid development clinic-flow transition")
)

type Status string

const (
	StatusWaiting    Status = "waiting"
	StatusArrived    Status = "arrived"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
)

type Command string

const (
	CommandArrive   Command = "arrive"
	CommandClaim    Command = "claim"
	CommandRelease  Command = "release"
	CommandComplete Command = "complete"
)

// Ticket is the minimum-disclosure synthetic queue representation.
type Ticket struct {
	ID                    string
	SyntheticPatientLabel string
	Status                Status
	AssigneeDisplayName   *string
	Version               int64
}

// CommandRequest proves the caller observed one ticket version.
type CommandRequest struct {
	TicketID        string
	ExpectedVersion int64
	Command         Command
}
