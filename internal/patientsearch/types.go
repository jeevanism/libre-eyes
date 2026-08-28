package patientsearch

import (
	"errors"
	"strconv"
	"time"
)

const (
	defaultPageSize = 25
	dateLayout      = "2006-01-02"
)

var (
	// ErrInvalidRequest indicates a generic request or cursor validation failure.
	ErrInvalidRequest = errors.New("invalid patient search request")
	// ErrUnavailable indicates that search or its required audit cannot complete safely.
	ErrUnavailable = errors.New("patient search unavailable")
	// ErrTimeout indicates that the bounded search deadline expired.
	ErrTimeout = errors.New("patient search timeout")
)

// Operation identifies the permission and audit policy for an endpoint.
type Operation string

const (
	OperationSearch         Operation = "search"
	OperationDuplicateCheck Operation = "duplicate_check"
	OperationRecent         Operation = "recent"
)

// RateLimitError reports a bounded retry delay without exposing internal state.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string { return "patient search rate limit exceeded" }

// CriteriaKind selects one approved exact search path.
type CriteriaKind string

const (
	CriteriaIdentifier  CriteriaKind = "identifier"
	CriteriaDemographic CriteriaKind = "demographic"
)

// IdentifierCriteria contains presentation input for one explicitly selected type.
type IdentifierCriteria struct {
	IdentifierTypeID int64
	Value            string
}

// DemographicInput contains presentation input for an exact demographic search.
type DemographicInput struct {
	GivenName   *string
	FamilyName  string
	DateOfBirth string
	Gender      Gender
}

// SearchCriteria is one approved exact patient-search form.
type SearchCriteria struct {
	Kind        CriteriaKind
	Identifier  *IdentifierCriteria
	Demographic *DemographicInput
}

// SearchRequest is a bounded service-level patient search request.
type SearchRequest struct {
	Criteria SearchCriteria
	Limit    int
	Cursor   string
}

// DuplicateRequest is an exact-only duplicate candidate request.
type DuplicateRequest struct {
	Criteria SearchCriteria
}

// DisplayedIdentifier is the minimum approved identifier representation.
type DisplayedIdentifier struct {
	TypeID string
	Label  string
	Value  string
}

// PatientResult is the minimum-disclosure API-safe patient representation.
type PatientResult struct {
	PatientID         string
	FullName          *string
	GivenName         *string
	FamilyName        *string
	DateOfBirth       string
	Gender            Gender
	PrimaryIdentifier *DisplayedIdentifier
	Deceased          bool
	DateOfDeath       *string
}

// SearchPage is a bounded patient search result page.
type SearchPage struct {
	Items      []PatientResult
	HasMore    bool
	NextCursor *string
}

// DuplicateResult contains explicit limitations with exact candidates.
type DuplicateResult struct {
	HardConflict bool
	Truncated    bool
	Candidates   []DuplicateResultCandidate
}

// DuplicateResultCandidate contains one exact reason and minimum patient representation.
type DuplicateResultCandidate struct {
	Reason  DuplicateReason
	Patient PatientResult
}

func mapPatient(record PatientRecord) (PatientResult, error) {
	result := PatientResult{
		PatientID: record.PublicID,
		GivenName: cloneString(record.GivenName), FamilyName: cloneString(record.FamilyName),
		DateOfBirth: record.DateOfBirth.Format(dateLayout), Gender: record.Gender,
		Deceased: record.Deceased,
	}
	if record.GivenName != nil || record.FamilyName != nil {
		name := ""
		if record.GivenName != nil {
			name = *record.GivenName
		}
		if record.FamilyName != nil {
			if name != "" {
				name += " "
			}
			name += *record.FamilyName
		}
		if name != "" {
			result.FullName = &name
		}
	}
	if record.DateOfDeath != nil {
		value := record.DateOfDeath.Format(dateLayout)
		result.DateOfDeath = &value
	}
	if record.PrimaryIdentifier != nil {
		value, err := formatIdentifier(*record.PrimaryIdentifier)
		if err != nil {
			return PatientResult{}, err
		}
		result.PrimaryIdentifier = &DisplayedIdentifier{
			TypeID: strconv.FormatInt(record.PrimaryIdentifier.TypeID, 10),
			Label:  record.PrimaryIdentifier.Label,
			Value:  value,
		}
	}
	return result, nil
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
