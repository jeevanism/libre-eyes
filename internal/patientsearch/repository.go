package patientsearch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	maximumPageSize      = 100
	duplicateResultLimit = 100
)

var (
	// ErrIdentifierTypeUnavailable indicates an inactive, invalid, or out-of-scope type.
	ErrIdentifierTypeUnavailable = errors.New("identifier type unavailable")
	// ErrInvalidRepositoryRequest indicates invalid internal repository arguments.
	ErrInvalidRepositoryRequest = errors.New("invalid patient search repository request")
)

// DBTX is the pgx query surface used by Repository and transactional tests.
type DBTX interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Repository executes institution-scoped, read-only patient-search queries.
type Repository struct {
	db DBTX
}

// NewRepository constructs a patient-search repository.
func NewRepository(db DBTX) (*Repository, error) {
	if db == nil {
		return nil, errors.New("patient search database is required")
	}
	return &Repository{db: db}, nil
}

// Gender is an approved administrative gender value.
type Gender string

const (
	GenderFemale  Gender = "female"
	GenderMale    Gender = "male"
	GenderOther   Gender = "other"
	GenderUnknown Gender = "unknown"
)

// IdentifierTypeConfig is an active configured identifier type.
type IdentifierTypeConfig struct {
	ID                     int64
	InstitutionID          int64
	SiteID                 *int64
	StableCode             string
	DisplayLabel           string
	NormalizationKind      IdentifierNormalizationKind
	ValidationPattern      string
	MaximumCanonicalLength int
	ZeroPadWidth           int
	ConfigurationVersion   int64
}

// PrimaryIdentifierRecord contains only fields needed for approved display formatting.
type PrimaryIdentifierRecord struct {
	TypeID        int64
	Label         string
	OriginalValue string
	DisplayPrefix string
	DisplaySuffix string
	SpacingRule   *string
}

// PatientRecord is the bounded repository representation of a current patient.
type PatientRecord struct {
	PublicID             string
	GivenName            *string
	FamilyName           *string
	GivenNameNormalized  *string
	FamilyNameNormalized *string
	DateOfBirth          time.Time
	Gender               Gender
	Deceased             bool
	DateOfDeath          *time.Time
	PrimaryIdentifier    *PrimaryIdentifierRecord
}

// PageBoundary is the internal fixed-order keyset boundary.
type PageBoundary struct {
	FamilyNameNormalized *string
	GivenNameNormalized  *string
	DateOfBirth          time.Time
	PublicID             string
}

// PatientPage is a bounded keyset page.
type PatientPage struct {
	Items        []PatientRecord
	HasMore      bool
	NextBoundary *PageBoundary
}

// DemographicCriteria contains already-normalized exact search values.
type DemographicCriteria struct {
	FamilyNameNormalized string
	GivenNameNormalized  *string
	DateOfBirth          time.Time
	Gender               Gender
}

// DuplicateDemographicCriteria contains the approved exact-only duplicate values.
type DuplicateDemographicCriteria struct {
	FamilyNameNormalized string
	GivenNameNormalized  string
	DateOfBirth          time.Time
}

// DuplicateReason identifies the exact matching path that produced a candidate.
type DuplicateReason string

const (
	DuplicateReasonIdentifier   DuplicateReason = "exact_identifier"
	DuplicateReasonDemographics DuplicateReason = "exact_demographics"
)

// DuplicateCandidate contains one current-institution patient and exact reason.
type DuplicateCandidate struct {
	Reason  DuplicateReason
	Patient PatientRecord
}

// DuplicateCandidates is an explicitly incomplete, bounded candidate set.
type DuplicateCandidates struct {
	Candidates   []DuplicateCandidate
	HardConflict bool
	Truncated    bool
}

// LoadIdentifierType loads one active, searchable, validated type in the current scope.
func (r *Repository) LoadIdentifierType(
	ctx context.Context,
	institutionID int64,
	siteID int64,
	identifierTypeID int64,
) (IdentifierTypeConfig, error) {
	if institutionID < 1 || siteID < 1 || identifierTypeID < 1 {
		return IdentifierTypeConfig{}, ErrInvalidRepositoryRequest
	}

	var config IdentifierTypeConfig
	var kind string
	var validationPattern *string
	var zeroPadWidth *int
	err := r.db.QueryRow(ctx, `
		SELECT id, institution_id, site_id, stable_code, display_label,
			normalization_kind, validation_pattern, maximum_canonical_length,
			zero_pad_width, configuration_version
		FROM patient_identifier_types
		WHERE id = $3
			AND institution_id = $1
			AND (site_id IS NULL OR site_id = $2)
			AND active
			AND searchable
			AND validation_state = 'validated'`,
		institutionID, siteID, identifierTypeID,
	).Scan(
		&config.ID, &config.InstitutionID, &config.SiteID, &config.StableCode,
		&config.DisplayLabel, &kind, &validationPattern,
		&config.MaximumCanonicalLength, &zeroPadWidth, &config.ConfigurationVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return IdentifierTypeConfig{}, ErrIdentifierTypeUnavailable
	}
	if err != nil {
		return IdentifierTypeConfig{}, fmt.Errorf("load identifier type: %w", err)
	}
	config.NormalizationKind = IdentifierNormalizationKind(kind)
	if validationPattern != nil {
		config.ValidationPattern = *validationPattern
	}
	if zeroPadWidth != nil {
		config.ZeroPadWidth = *zeroPadWidth
	}
	return config, nil
}

// SearchByIdentifier returns active current-institution patients for one canonical identity.
func (r *Repository) SearchByIdentifier(
	ctx context.Context,
	institutionID int64,
	siteID int64,
	identifierTypeID int64,
	canonicalValue string,
	limit int,
	boundary *PageBoundary,
) (PatientPage, error) {
	if institutionID < 1 || siteID < 1 || identifierTypeID < 1 || canonicalValue == "" {
		return PatientPage{}, ErrInvalidRepositoryRequest
	}
	if err := validatePage(limit, boundary); err != nil {
		return PatientPage{}, err
	}

	boundaryArgs := pageBoundaryArgs(boundary)
	args := []any{
		institutionID, siteID, identifierTypeID, canonicalValue,
		boundary != nil, boundaryArgs.familyName, boundaryArgs.givenName,
		boundaryArgs.dateOfBirth, boundaryArgs.publicID, limit + 1,
	}
	return r.queryPatientPage(ctx, identifierSearchQuery, limit, args...)
}

// SearchByDemographics returns exact normalized current-institution patients.
func (r *Repository) SearchByDemographics(
	ctx context.Context,
	institutionID int64,
	siteID int64,
	criteria DemographicCriteria,
	limit int,
	boundary *PageBoundary,
) (PatientPage, error) {
	if institutionID < 1 || siteID < 1 || criteria.FamilyNameNormalized == "" ||
		criteria.DateOfBirth.IsZero() || !validGender(criteria.Gender) {
		return PatientPage{}, ErrInvalidRepositoryRequest
	}
	if criteria.GivenNameNormalized != nil && *criteria.GivenNameNormalized == "" {
		return PatientPage{}, ErrInvalidRepositoryRequest
	}
	if err := validatePage(limit, boundary); err != nil {
		return PatientPage{}, err
	}

	boundaryArgs := pageBoundaryArgs(boundary)
	args := []any{
		institutionID, siteID, criteria.FamilyNameNormalized, criteria.DateOfBirth,
		string(criteria.Gender), criteria.GivenNameNormalized, boundary != nil,
		boundaryArgs.familyName, boundaryArgs.givenName, boundaryArgs.dateOfBirth,
		boundaryArgs.publicID, limit + 1,
	}
	return r.queryPatientPage(ctx, demographicSearchQuery, limit, args...)
}

// FindIdentifierDuplicates returns bounded exact active-identifier candidates.
func (r *Repository) FindIdentifierDuplicates(
	ctx context.Context,
	institutionID int64,
	siteID int64,
	identifierTypeID int64,
	canonicalValue string,
) (DuplicateCandidates, error) {
	page, err := r.SearchByIdentifier(
		ctx, institutionID, siteID, identifierTypeID, canonicalValue,
		duplicateResultLimit, nil,
	)
	if err != nil {
		return DuplicateCandidates{}, err
	}
	return duplicateCandidates(page, DuplicateReasonIdentifier, len(page.Items) > 0), nil
}

// FindDemographicDuplicates returns bounded exact current-institution candidates.
func (r *Repository) FindDemographicDuplicates(
	ctx context.Context,
	institutionID int64,
	siteID int64,
	criteria DuplicateDemographicCriteria,
) (DuplicateCandidates, error) {
	if institutionID < 1 || siteID < 1 || criteria.FamilyNameNormalized == "" ||
		criteria.GivenNameNormalized == "" || criteria.DateOfBirth.IsZero() {
		return DuplicateCandidates{}, ErrInvalidRepositoryRequest
	}

	page, err := r.queryPatientPage(
		ctx,
		demographicDuplicateQuery,
		duplicateResultLimit,
		institutionID, siteID, criteria.FamilyNameNormalized,
		criteria.GivenNameNormalized, criteria.DateOfBirth, duplicateResultLimit+1,
	)
	if err != nil {
		return DuplicateCandidates{}, err
	}
	return duplicateCandidates(page, DuplicateReasonDemographics, false), nil
}

func (r *Repository) queryPatientPage(
	ctx context.Context,
	query string,
	limit int,
	args ...any,
) (PatientPage, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return PatientPage{}, fmt.Errorf("query patient search: %w", err)
	}
	defer rows.Close()

	items := make([]PatientRecord, 0, limit+1)
	for rows.Next() {
		record, err := scanPatientRecord(rows)
		if err != nil {
			return PatientPage{}, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return PatientPage{}, fmt.Errorf("iterate patient search: %w", err)
	}

	page := PatientPage{Items: items}
	if len(page.Items) > limit {
		page.HasMore = true
		page.Items = page.Items[:limit]
	}
	if page.HasMore {
		last := page.Items[len(page.Items)-1]
		page.NextBoundary = &PageBoundary{
			FamilyNameNormalized: last.FamilyNameNormalized,
			GivenNameNormalized:  last.GivenNameNormalized,
			DateOfBirth:          last.DateOfBirth,
			PublicID:             last.PublicID,
		}
	}
	return page, nil
}

func scanPatientRecord(row pgx.Row) (PatientRecord, error) {
	var record PatientRecord
	var gender string
	var identifierTypeID *int64
	var identifierLabel, originalValue, displayPrefix, displaySuffix, spacingRule *string
	if err := row.Scan(
		&record.PublicID, &record.GivenName, &record.FamilyName,
		&record.GivenNameNormalized, &record.FamilyNameNormalized,
		&record.DateOfBirth, &gender, &record.Deceased, &record.DateOfDeath,
		&identifierTypeID, &identifierLabel, &originalValue,
		&displayPrefix, &displaySuffix, &spacingRule,
	); err != nil {
		return PatientRecord{}, fmt.Errorf("scan patient search result: %w", err)
	}
	record.Gender = Gender(gender)
	if identifierTypeID != nil {
		record.PrimaryIdentifier = &PrimaryIdentifierRecord{
			TypeID:        *identifierTypeID,
			Label:         valueOrEmpty(identifierLabel),
			OriginalValue: valueOrEmpty(originalValue),
			DisplayPrefix: valueOrEmpty(displayPrefix),
			DisplaySuffix: valueOrEmpty(displaySuffix),
			SpacingRule:   spacingRule,
		}
	}
	return record, nil
}

func validatePage(limit int, boundary *PageBoundary) error {
	if limit < 1 || limit > maximumPageSize {
		return ErrInvalidRepositoryRequest
	}
	if boundary != nil && (boundary.DateOfBirth.IsZero() || boundary.PublicID == "") {
		return ErrInvalidRepositoryRequest
	}
	return nil
}

type boundaryArguments struct {
	familyName, givenName *string
	dateOfBirth           *time.Time
	publicID              *string
}

func pageBoundaryArgs(boundary *PageBoundary) boundaryArguments {
	if boundary == nil {
		return boundaryArguments{}
	}
	dateOfBirth := boundary.DateOfBirth
	publicID := boundary.PublicID
	return boundaryArguments{
		familyName:  boundary.FamilyNameNormalized,
		givenName:   boundary.GivenNameNormalized,
		dateOfBirth: &dateOfBirth,
		publicID:    &publicID,
	}
}

func duplicateCandidates(page PatientPage, reason DuplicateReason, hardConflict bool) DuplicateCandidates {
	candidates := make([]DuplicateCandidate, len(page.Items))
	for index, patient := range page.Items {
		candidates[index] = DuplicateCandidate{Reason: reason, Patient: patient}
	}
	return DuplicateCandidates{
		Candidates:   candidates,
		HardConflict: hardConflict,
		Truncated:    page.HasMore,
	}
}

func validGender(gender Gender) bool {
	return gender == GenderFemale || gender == GenderMale ||
		gender == GenderOther || gender == GenderUnknown
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

const patientResultColumns = `
	p.public_id::text,
	p.given_name,
	p.family_name,
	p.given_name_normalized,
	p.family_name_normalized,
	p.date_of_birth,
	p.gender::text,
	p.deceased,
	p.date_of_death,
	primary_identifier.identifier_type_id,
	primary_identifier.display_label,
	primary_identifier.original_value,
	primary_identifier.display_prefix,
	primary_identifier.display_suffix,
	primary_identifier.spacing_rule`

const patientScopeAndDisplayJoin = `
	JOIN patient_institutions patient_scope
		ON patient_scope.patient_id = p.id
		AND patient_scope.institution_id = $1
		AND patient_scope.active
	LEFT JOIN LATERAL (
		SELECT
			pi.identifier_type_id,
			pit.display_label,
			pi.original_value,
			pit.display_prefix,
			pit.display_suffix,
			pit.spacing_rule
		FROM patient_identifiers pi
		JOIN patient_identifier_types pit
			ON pit.id = pi.identifier_type_id
			AND pit.institution_id = $1
			AND (pit.site_id IS NULL OR pit.site_id = $2)
			AND pit.active
			AND pit.searchable
			AND pit.validation_state = 'validated'
		WHERE pi.patient_id = p.id
			AND pi.active
			AND pi.lifecycle = 'active'
		ORDER BY pit.display_order, pit.id, pi.id
		LIMIT 1
	) primary_identifier ON TRUE`

const patientStableOrder = `
	ORDER BY
		p.family_name_normalized ASC NULLS LAST,
		p.given_name_normalized ASC NULLS LAST,
		p.date_of_birth ASC,
		p.public_id ASC`

const identifierSearchQuery = `
	SELECT ` + patientResultColumns + `
	FROM patients p
	` + patientScopeAndDisplayJoin + `
	JOIN patient_identifiers searched_identifier
		ON searched_identifier.patient_id = p.id
		AND searched_identifier.identifier_type_id = $3
		AND searched_identifier.canonical_value = $4
		AND searched_identifier.active
		AND searched_identifier.lifecycle = 'active'
	JOIN patient_identifier_types searched_type
		ON searched_type.id = searched_identifier.identifier_type_id
		AND searched_type.institution_id = $1
		AND (searched_type.site_id IS NULL OR searched_type.site_id = $2)
		AND searched_type.active
		AND searched_type.searchable
		AND searched_type.validation_state = 'validated'
	WHERE p.active
		AND (
			NOT $5::boolean
			OR ROW(
				p.family_name_normalized IS NULL,
				COALESCE(p.family_name_normalized, ''),
				p.given_name_normalized IS NULL,
				COALESCE(p.given_name_normalized, ''),
				p.date_of_birth,
				p.public_id
			) > ROW(
				$6::text IS NULL,
				COALESCE($6::text, ''),
				$7::text IS NULL,
				COALESCE($7::text, ''),
				$8::date,
				$9::uuid
			)
		)
	` + patientStableOrder + `
	LIMIT $10`

const demographicSearchQuery = `
	SELECT ` + patientResultColumns + `
	FROM patients p
	` + patientScopeAndDisplayJoin + `
	WHERE p.active
		AND p.family_name_normalized = $3
		AND p.date_of_birth = $4
		AND p.gender = $5::patient_gender
		AND ($6::text IS NULL OR p.given_name_normalized = $6)
		AND (
			NOT $7::boolean
			OR ROW(
				p.family_name_normalized IS NULL,
				COALESCE(p.family_name_normalized, ''),
				p.given_name_normalized IS NULL,
				COALESCE(p.given_name_normalized, ''),
				p.date_of_birth,
				p.public_id
			) > ROW(
				$8::text IS NULL,
				COALESCE($8::text, ''),
				$9::text IS NULL,
				COALESCE($9::text, ''),
				$10::date,
				$11::uuid
			)
		)
	` + patientStableOrder + `
	LIMIT $12`

const demographicDuplicateQuery = `
	SELECT ` + patientResultColumns + `
	FROM patients p
	` + patientScopeAndDisplayJoin + `
	WHERE p.active
		AND p.family_name_normalized = $3
		AND p.given_name_normalized = $4
		AND p.date_of_birth = $5
	` + patientStableOrder + `
	LIMIT $6`
