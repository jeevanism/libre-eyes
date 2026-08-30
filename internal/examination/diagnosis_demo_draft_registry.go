package examination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const (
	diagnosisDemoEventType   = "ophthalmology.principal_diagnosis_demo"
	diagnosisDemoSchema      = 1
	diagnosisDemoProfileCode = "development_ophthalmology_diagnosis_v1"
	diagnosisDemoRecordMode  = "development_synthetic_diagnosis"
	diagnosisDemoTimeZone    = "Europe/London"
)

type diagnosisDemoCatalogue interface {
	selectionIsActive(context.Context, string, string) (bool, error)
	hasDevelopmentCodes(context.Context) (bool, error)
}

// DiagnosisDemoDraftRegistry validates the approved synthetic diagnosis draft.
type DiagnosisDemoDraftRegistry struct {
	catalogue diagnosisDemoCatalogue
	now       func() time.Time
}

func NewDiagnosisDemoDraftRegistry(ctx context.Context, pool *pgxpool.Pool, environment string) (*DiagnosisDemoDraftRegistry, error) {
	if pool == nil {
		return nil, errors.New("development diagnosis catalogue database is required")
	}
	return newDiagnosisDemoDraftRegistry(ctx, postgresDiagnosisDemoCatalogue{pool: pool}, environment)
}

func newDiagnosisDemoDraftRegistry(ctx context.Context, catalogue diagnosisDemoCatalogue, environment string) (*DiagnosisDemoDraftRegistry, error) {
	if catalogue == nil {
		return nil, errors.New("development diagnosis catalogue is required")
	}
	if environment == "production" {
		present, err := catalogue.hasDevelopmentCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify production development diagnosis catalogue: %w", err)
		}
		if present {
			return nil, errors.New("production startup refused: development diagnosis catalogue codes found")
		}
	}
	return &DiagnosisDemoDraftRegistry{catalogue: catalogue, now: time.Now}, nil
}

func (r *DiagnosisDemoDraftRegistry) Validate(ctx context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != diagnosisDemoEventType || intent != episodes.DraftIntentCreate || schemaVersion != diagnosisDemoSchema {
		return episodes.ErrInvalidRequest
	}
	parsed, err := parseDiagnosisDemoPayload(payload, r.now)
	if err != nil {
		return episodes.ErrInvalidRequest
	}
	active, err := r.catalogue.selectionIsActive(ctx, parsed.ProfileCode, parsed.SelectionCode)
	if err != nil {
		return fmt.Errorf("look up development diagnosis catalogue: %w", err)
	}
	if !active {
		return episodes.ErrInvalidRequest
	}
	return nil
}

type diagnosisDemoPayload struct {
	RecordMode    string `json:"recordMode"`
	ProfileCode   string `json:"profileCode"`
	SelectionCode string `json:"selectionCode"`
	Laterality    string `json:"laterality"`
	DiagnosisDate string `json:"diagnosisDate"`
}

func parseDiagnosisDemoPayload(payload json.RawMessage, now func() time.Time) (diagnosisDemoPayload, error) {
	var parsed diagnosisDemoPayload
	if err := decodeStrictJSON(payload, &parsed); err != nil || parsed.RecordMode != diagnosisDemoRecordMode || parsed.ProfileCode != diagnosisDemoProfileCode {
		return diagnosisDemoPayload{}, errors.New("invalid development diagnosis payload")
	}
	if !catalogueCodePattern.MatchString(parsed.SelectionCode) || (parsed.Laterality != "left" && parsed.Laterality != "right" && parsed.Laterality != "bilateral") {
		return diagnosisDemoPayload{}, errors.New("invalid development diagnosis selection")
	}
	if _, err := time.Parse("2006-01-02", parsed.DiagnosisDate); err != nil {
		return diagnosisDemoPayload{}, errors.New("invalid development diagnosis date")
	}
	location, err := time.LoadLocation(diagnosisDemoTimeZone)
	if err != nil {
		return diagnosisDemoPayload{}, fmt.Errorf("load development diagnosis timezone: %w", err)
	}
	if parsed.DiagnosisDate > now().In(location).Format("2006-01-02") {
		return diagnosisDemoPayload{}, errors.New("future development diagnosis date")
	}
	return parsed, nil
}

type postgresDiagnosisDemoCatalogue struct{ pool *pgxpool.Pool }

func (c postgresDiagnosisDemoCatalogue) selectionIsActive(ctx context.Context, profileCode, selectionCode string) (bool, error) {
	var active bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM development_diagnosis_profiles p
		JOIN development_diagnosis_profile_selections s ON s.profile_id = p.id
		WHERE p.code = $1 AND s.code = $2 AND p.active AND s.active AND s.selectable
	)`, profileCode, selectionCode).Scan(&active)
	return active, err
}

func (c postgresDiagnosisDemoCatalogue) hasDevelopmentCodes(ctx context.Context) (bool, error) {
	var present bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM development_diagnosis_profiles
		UNION ALL
		SELECT 1 FROM development_diagnosis_profile_selections
	)`).Scan(&present)
	return present, err
}
