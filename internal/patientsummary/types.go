package patientsummary

import "time"

// WarningStatus is the clinically meaningful projection state.
type WarningStatus string

const (
	WarningPresent     WarningStatus = "present"
	WarningNoneKnown   WarningStatus = "none_known"
	WarningUnknown     WarningStatus = "unknown"
	WarningUnavailable WarningStatus = "unavailable"
)

// Header is the bounded identity and warning snapshot returned by the repository.
type Header struct {
	PublicID          string
	GivenName         *string
	FamilyName        *string
	DateOfBirth       time.Time
	Gender            string
	Deceased          bool
	DateOfDeath       *time.Time
	Version           int64
	WarningVersion    *int64
	AllergyStatus     *WarningStatus
	AlertStatus       *WarningStatus
	AllergyAssessedAt *time.Time
	AlertAssessedAt   *time.Time
	ProjectionState   *string
}
