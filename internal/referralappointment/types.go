package referralappointment

import "errors"

const PermissionManage = "referral.development_appointment.manage"

var (
	ErrInvalidRequest    = errors.New("invalid development referral appointment request")
	ErrNotFound          = errors.New("development referral appointment not found")
	ErrConflict          = errors.New("development referral appointment changed")
	ErrInvalidTransition = errors.New("invalid development referral appointment transition")
)

type Status string

const (
	StatusRequested Status = "requested"
	StatusScheduled Status = "scheduled"
	StatusArrived   Status = "arrived"
	StatusCompleted Status = "completed"
	StatusAbandoned Status = "abandoned"
)

type Command string

const (
	CommandSchedule Command = "schedule"
	CommandArrive   Command = "arrive"
	CommandComplete Command = "complete"
	CommandAbandon  Command = "abandon"
)

type Request struct {
	ID, SyntheticPatientLabel, RecipientRole, ClinicCode, AppointmentDate, AppointmentTime, Priority, Notes string
	Status                                                                                                  Status
	Version                                                                                                 int64
}

type CreateRequest struct {
	SyntheticPatientID, SyntheticPatientLabel, RecipientRole, ClinicCode, AppointmentDate, AppointmentTime, Priority, Notes string
	RetentionKind                                                                                                           string
}
type CommandRequest struct {
	RequestID       string
	ExpectedVersion int64
	Command         Command
}
type Authorization struct {
	principalUserID, institutionID, siteID, firmID int64
	correlationID                                  string
	service                                        *Service
}
