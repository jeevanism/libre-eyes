package examination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const (
	iopEventType     = "ophthalmology.intraocular_pressure"
	iopSchemaVersion = 1
	iopProfileCode   = "development_iop_manual_mmhg"
	iopRecordMode    = "development_raw_mmhg"
)

type iopCatalogueLookup interface {
	valueIsActive(context.Context, string, string) (bool, error)
	hasDevelopmentCodes(context.Context) (bool, error)
}

// IOPDraftRegistry validates the approved, development-only IOP draft schema.
type IOPDraftRegistry struct {
	catalogue iopCatalogueLookup
}

// NewIOPDraftRegistry creates the IOP validator and refuses a production startup
// whenever a development catalogue code is present, even if it is inactive.
func NewIOPDraftRegistry(ctx context.Context, pool *pgxpool.Pool, environment string) (*IOPDraftRegistry, error) {
	if pool == nil {
		return nil, errors.New("intraocular pressure catalogue database is required")
	}
	return newIOPDraftRegistry(ctx, postgresIOPCatalogue{pool: pool}, environment)
}

func newIOPDraftRegistry(ctx context.Context, catalogue iopCatalogueLookup, environment string) (*IOPDraftRegistry, error) {
	if catalogue == nil {
		return nil, errors.New("intraocular pressure catalogue is required")
	}
	if environment == "production" {
		present, err := catalogue.hasDevelopmentCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify production intraocular pressure catalogue: %w", err)
		}
		if present {
			return nil, errors.New("production startup refused: development intraocular pressure catalogue codes found")
		}
	}
	return &IOPDraftRegistry{catalogue: catalogue}, nil
}

// Validate implements episodes.DraftPayloadRegistry.
func (r *IOPDraftRegistry) Validate(ctx context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != iopEventType || intent != episodes.DraftIntentCreate || schemaVersion != iopSchemaVersion {
		return episodes.ErrInvalidRequest
	}
	parsed, err := parseIOPPayload(payload)
	if err != nil {
		return episodes.ErrInvalidRequest
	}
	for _, eye := range parsed.Eyes {
		active, err := r.catalogue.valueIsActive(ctx, parsed.ProfileCode, eye.ValueCode)
		if err != nil {
			return fmt.Errorf("look up intraocular pressure catalogue: %w", err)
		}
		if !active {
			return episodes.ErrInvalidRequest
		}
	}
	return nil
}

type iopPayload struct {
	RecordMode  string       `json:"recordMode"`
	ProfileCode string       `json:"profileCode"`
	Eyes        []iopEyeData `json:"eyes"`
}

type iopEyeData struct {
	Eye       string `json:"eye"`
	ValueCode string `json:"valueCode"`
}

func parseIOPPayload(payload json.RawMessage) (iopPayload, error) {
	var parsed iopPayload
	if err := decodeStrictJSON(payload, &parsed); err != nil || parsed.RecordMode != iopRecordMode || parsed.ProfileCode != iopProfileCode || len(parsed.Eyes) < 1 || len(parsed.Eyes) > 2 {
		return iopPayload{}, errors.New("invalid intraocular pressure payload")
	}
	eyes := make(map[string]struct{}, len(parsed.Eyes))
	for _, eye := range parsed.Eyes {
		if (eye.Eye != "left" && eye.Eye != "right") || !catalogueCodePattern.MatchString(eye.ValueCode) {
			return iopPayload{}, errors.New("invalid intraocular pressure eye")
		}
		if _, exists := eyes[eye.Eye]; exists {
			return iopPayload{}, errors.New("duplicate intraocular pressure eye")
		}
		eyes[eye.Eye] = struct{}{}
	}
	return parsed, nil
}

type postgresIOPCatalogue struct {
	pool *pgxpool.Pool
}

func (c postgresIOPCatalogue) valueIsActive(ctx context.Context, profileCode, valueCode string) (bool, error) {
	var active bool
	err := c.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM intraocular_pressure_profiles p
			JOIN intraocular_pressure_profile_values v ON v.profile_id = p.id
			WHERE p.code = $1 AND v.code = $2
				AND p.active AND v.active AND v.selectable
		)`, profileCode, valueCode).Scan(&active)
	return active, err
}

func (c postgresIOPCatalogue) hasDevelopmentCodes(ctx context.Context) (bool, error) {
	var present bool
	err := c.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM intraocular_pressure_profiles WHERE code LIKE 'development\_%'
			UNION ALL
			SELECT 1 FROM intraocular_pressure_profile_values WHERE code LIKE 'development\_%'
		)`).Scan(&present)
	return present, err
}
