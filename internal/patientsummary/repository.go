package patientsummary

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidRequest = errors.New("invalid patient summary request")
	ErrNotFound       = errors.New("patient summary not found")
)

// DBTX is the read surface used by the summary repository and its tests.
type DBTX interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Repository loads institution-scoped, read-only summary snapshots.
type Repository struct{ db DBTX }

func NewRepository(db DBTX) (*Repository, error) {
	if db == nil {
		return nil, errors.New("patient summary database is required")
	}
	return &Repository{db: db}, nil
}

// LoadHeader returns only an active patient associated with the current institution.
// The public UUID is the only caller-supplied patient locator.
func (r *Repository) LoadHeader(ctx context.Context, institutionID int64, publicID string) (Header, error) {
	if institutionID < 1 || publicID == "" {
		return Header{}, ErrInvalidRequest
	}
	var header Header
	var gender string
	var allergyStatus, alertStatus *string
	err := r.db.QueryRow(ctx, `
		SELECT p.public_id, p.given_name, p.family_name, p.date_of_birth,
			p.gender::text, p.deceased, p.date_of_death, p.version,
			wp.warning_version, wp.allergy_status::text, wp.alert_status::text,
			wp.allergy_assessed_at, wp.alert_assessed_at, wp.projection_state
		FROM patients p
		JOIN patient_institutions pi
			ON pi.patient_id = p.id
			AND pi.institution_id = $1
			AND pi.active
		LEFT JOIN patient_summary_warning_projections wp
			ON wp.patient_id = p.id
		WHERE p.public_id = $2
			AND p.active
		LIMIT 1`, institutionID, publicID,
	).Scan(
		&header.PublicID, &header.GivenName, &header.FamilyName,
		&header.DateOfBirth, &gender, &header.Deceased, &header.DateOfDeath,
		&header.Version, &header.WarningVersion, &allergyStatus, &alertStatus,
		&header.AllergyAssessedAt, &header.AlertAssessedAt, &header.ProjectionState,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Header{}, ErrNotFound
	}
	if err != nil {
		return Header{}, fmt.Errorf("load patient summary header: %w", err)
	}
	header.Gender = gender
	if allergyStatus != nil {
		value := WarningStatus(*allergyStatus)
		header.AllergyStatus = &value
	}
	if alertStatus != nil {
		value := WarningStatus(*alertStatus)
		header.AlertStatus = &value
	}
	return header, nil
}

func (r *Repository) LoadWarningDetails(ctx context.Context, institutionID int64, publicID string) (WarningDetails, error) {
	if institutionID < 1 || publicID == "" {
		return WarningDetails{}, ErrInvalidRequest
	}
	var result WarningDetails
	var allergyStatus, alertStatus string
	var patientID int64
	err := r.db.QueryRow(ctx, `
		SELECT p.id, p.public_id, wp.allergy_status::text, wp.alert_status::text,
			wp.warning_version
		FROM patients p
		JOIN patient_institutions pi ON pi.patient_id = p.id
			AND pi.institution_id = $1 AND pi.active
		JOIN patient_summary_warning_projections wp ON wp.patient_id = p.id
		WHERE p.public_id = $2 AND p.active AND wp.projection_state = 'verified'
		LIMIT 1`, institutionID, publicID,
	).Scan(&patientID, &result.PatientID, &allergyStatus, &alertStatus, &result.WarningVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return WarningDetails{}, ErrNotFound
	}
	if err != nil {
		return WarningDetails{}, fmt.Errorf("load patient warning projection: %w", err)
	}
	rows, err := r.db.(interface {
		Query(context.Context, string, ...any) (pgx.Rows, error)
	}).Query(ctx, `
		SELECT warning_kind::text, code, label, reaction, comment
		FROM patient_summary_warning_items
		WHERE patient_id = $1
		ORDER BY item_order, id
		LIMIT 201`, patientID)
	if err != nil {
		return WarningDetails{}, fmt.Errorf("load patient warning items: %w", err)
	}
	defer rows.Close()
	result.AllergyStatus = WarningStatus(allergyStatus)
	result.AlertStatus = WarningStatus(alertStatus)
	for rows.Next() {
		if len(result.Items) >= 200 {
			return WarningDetails{}, ErrWarningOverflow
		}
		var item WarningItem
		if err := rows.Scan(&item.Kind, &item.Code, &item.Label, &item.Reaction, &item.Comment); err != nil {
			return WarningDetails{}, fmt.Errorf("scan patient warning item: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return WarningDetails{}, fmt.Errorf("iterate patient warning items: %w", err)
	}
	result.Complete = true
	return result, nil
}
