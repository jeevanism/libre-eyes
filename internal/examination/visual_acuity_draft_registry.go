// Package examination owns approved clinical payload validation.
package examination

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const (
	visualAcuityEventType     = "ophthalmology.visual_acuity"
	visualAcuitySchemaVersion = 1
)

var catalogueCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

type catalogueLookup interface {
	readingIsActive(context.Context, string, string, string) (bool, error)
	hasActiveDevelopmentCodes(context.Context) (bool, error)
}

// VisualAcuityDraftRegistry validates the approved, draft-only Visual Acuity schema.
type VisualAcuityDraftRegistry struct {
	catalogue catalogueLookup
}

// NewVisualAcuityDraftRegistry creates the draft registry and rejects a production
// database contaminated with the explicitly development-only catalogue fixture.
func NewVisualAcuityDraftRegistry(ctx context.Context, pool *pgxpool.Pool, environment string) (*VisualAcuityDraftRegistry, error) {
	if pool == nil {
		return nil, errors.New("visual acuity catalogue database is required")
	}
	return newVisualAcuityDraftRegistry(ctx, postgresCatalogue{pool: pool}, environment)
}

func newVisualAcuityDraftRegistry(ctx context.Context, catalogue catalogueLookup, environment string) (*VisualAcuityDraftRegistry, error) {
	if catalogue == nil {
		return nil, errors.New("visual acuity catalogue is required")
	}
	if environment == "production" {
		hasDevelopmentCodes, err := catalogue.hasActiveDevelopmentCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify production visual acuity catalogue: %w", err)
		}
		if hasDevelopmentCodes {
			return nil, errors.New("production startup refused: active development visual acuity catalogue codes found")
		}
	}
	return &VisualAcuityDraftRegistry{catalogue: catalogue}, nil
}

// Validate implements episodes.DraftPayloadRegistry. Unexpected catalogue failures
// are deliberately returned separately so the HTTP boundary reports only a generic
// unavailable response and never echoes clinical payload values.
func (r *VisualAcuityDraftRegistry) Validate(ctx context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != visualAcuityEventType || intent != episodes.DraftIntentCreate || schemaVersion != visualAcuitySchemaVersion {
		return episodes.ErrInvalidRequest
	}
	parsed, err := parseVisualAcuityPayload(payload)
	if err != nil {
		return episodes.ErrInvalidRequest
	}
	for _, eye := range parsed.Eyes {
		if eye.Assessment != "recorded" {
			continue
		}
		for _, reading := range eye.readings {
			active, err := r.catalogue.readingIsActive(ctx, reading.UnitCode, reading.ValueCode, reading.MethodCode)
			if err != nil {
				return fmt.Errorf("look up visual acuity catalogue: %w", err)
			}
			if !active {
				return episodes.ErrInvalidRequest
			}
		}
	}
	return nil
}

type visualAcuityPayload struct {
	RecordMode string                `json:"recordMode"`
	Eyes       []visualAcuityEyeData `json:"eyes"`
}

type visualAcuityEyeData struct {
	Eye        string          `json:"eye"`
	Assessment string          `json:"assessment"`
	Readings   json.RawMessage `json:"readings"`
	readings   []visualAcuityReadingData
}

type visualAcuityReadingData struct {
	UnitCode   string `json:"unitCode"`
	ValueCode  string `json:"valueCode"`
	MethodCode string `json:"methodCode"`
}

func parseVisualAcuityPayload(payload json.RawMessage) (visualAcuityPayload, error) {
	var parsed visualAcuityPayload
	if err := decodeStrictJSON(payload, &parsed); err != nil || parsed.RecordMode != "simple" || len(parsed.Eyes) < 1 || len(parsed.Eyes) > 2 {
		return visualAcuityPayload{}, errors.New("invalid visual acuity payload")
	}
	eyes := make(map[string]struct{}, len(parsed.Eyes))
	for index := range parsed.Eyes {
		eye := &parsed.Eyes[index]
		if (eye.Eye != "left" && eye.Eye != "right") || eye.Assessment == "" {
			return visualAcuityPayload{}, errors.New("invalid visual acuity eye")
		}
		if _, exists := eyes[eye.Eye]; exists {
			return visualAcuityPayload{}, errors.New("duplicate visual acuity eye")
		}
		eyes[eye.Eye] = struct{}{}
		switch eye.Assessment {
		case "recorded":
			var readings []visualAcuityReadingData
			if len(eye.Readings) == 0 || decodeStrictJSON(eye.Readings, &readings) != nil || len(readings) < 1 || len(readings) > 20 {
				return visualAcuityPayload{}, errors.New("invalid visual acuity readings")
			}
			methods := make(map[string]struct{}, len(readings))
			for _, reading := range readings {
				if !catalogueCodePattern.MatchString(reading.UnitCode) || !catalogueCodePattern.MatchString(reading.ValueCode) || !catalogueCodePattern.MatchString(reading.MethodCode) {
					return visualAcuityPayload{}, errors.New("invalid visual acuity catalogue code")
				}
				if _, exists := methods[reading.MethodCode]; exists {
					return visualAcuityPayload{}, errors.New("duplicate visual acuity method")
				}
				methods[reading.MethodCode] = struct{}{}
			}
			eye.readings = readings
		case "unable_to_assess", "eye_missing":
			if len(eye.Readings) != 0 {
				return visualAcuityPayload{}, errors.New("non-recorded visual acuity eye has readings")
			}
		default:
			return visualAcuityPayload{}, errors.New("invalid visual acuity assessment")
		}
	}
	return parsed, nil
}

func decodeStrictJSON(payload json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}

type postgresCatalogue struct {
	pool *pgxpool.Pool
}

func (c postgresCatalogue) readingIsActive(ctx context.Context, unitCode, valueCode, methodCode string) (bool, error) {
	var active bool
	err := c.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM visual_acuity_units u
			JOIN visual_acuity_unit_values v ON v.unit_id = u.id
			JOIN visual_acuity_methods m ON m.code = $3
			WHERE u.code = $1 AND v.code = $2
				AND u.active AND v.active AND v.selectable AND m.active
		)`, unitCode, valueCode, methodCode).Scan(&active)
	return active, err
}

func (c postgresCatalogue) hasActiveDevelopmentCodes(ctx context.Context) (bool, error) {
	var present bool
	err := c.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM visual_acuity_units WHERE active AND code LIKE 'development\_%'
			UNION ALL
			SELECT 1 FROM visual_acuity_unit_values WHERE active AND code LIKE 'development\_%'
			UNION ALL
			SELECT 1 FROM visual_acuity_methods WHERE active AND code LIKE 'development\_%'
		)`).Scan(&present)
	return present, err
}
