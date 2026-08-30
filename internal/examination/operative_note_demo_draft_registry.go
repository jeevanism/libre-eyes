package examination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const (
	operativeNoteDemoEventType = "ophthalmology.operative_note_demo"
	operativeNoteDemoSchema    = 1
	operativeNoteDemoMode      = "development_synthetic_operative_note"
)

type operativeNoteCatalogue interface {
	codeIsActive(context.Context, string, string) (bool, error)
	hasDevelopmentCodes(context.Context) (bool, error)
}

// OperativeNoteDemoDraftRegistry validates only the uncommitted demo payload.
type OperativeNoteDemoDraftRegistry struct{ catalogue operativeNoteCatalogue }

func NewOperativeNoteDemoDraftRegistry(ctx context.Context, pool *pgxpool.Pool, environment string) (*OperativeNoteDemoDraftRegistry, error) {
	if pool == nil {
		return nil, errors.New("operative-note demonstration catalogue database is required")
	}
	catalogue := postgresOperativeNoteCatalogue{pool: pool}
	if environment == "production" {
		present, err := catalogue.hasDevelopmentCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify production operative-note catalogue: %w", err)
		}
		if present {
			return nil, errors.New("production startup refused: development operative-note catalogue codes found")
		}
	}
	return &OperativeNoteDemoDraftRegistry{catalogue: catalogue}, nil
}

func (r *OperativeNoteDemoDraftRegistry) Validate(ctx context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if r == nil || eventTypeCode != operativeNoteDemoEventType || intent != episodes.DraftIntentCreate || schemaVersion != operativeNoteDemoSchema {
		return episodes.ErrInvalidRequest
	}
	parsed, err := parseOperativeNoteDemoPayload(payload)
	if err != nil {
		return episodes.ErrInvalidRequest
	}
	for category, code := range map[string]string{
		"procedure": parsed.ProcedureCode, "laterality": parsed.Laterality,
		"surgeon": parsed.SurgeonCode, "anaesthetic": parsed.AnaestheticCode,
	} {
		active, lookupErr := r.catalogue.codeIsActive(ctx, category, code)
		if lookupErr != nil {
			return fmt.Errorf("look up operative-note catalogue: %w", lookupErr)
		}
		if !active {
			return episodes.ErrInvalidRequest
		}
	}
	for _, code := range parsed.DeliveryCodes {
		active, lookupErr := r.catalogue.codeIsActive(ctx, "delivery", code)
		if lookupErr != nil {
			return fmt.Errorf("look up operative-note delivery catalogue: %w", lookupErr)
		}
		if !active {
			return episodes.ErrInvalidRequest
		}
	}
	return nil
}

type operativeNoteDemoPayload struct {
	RecordMode      string   `json:"recordMode"`
	ProcedureCode   string   `json:"procedureCode"`
	Laterality      string   `json:"laterality"`
	SurgeonCode     string   `json:"surgeonCode"`
	AnaestheticCode string   `json:"anaestheticCode"`
	DeliveryCodes   []string `json:"deliveryCodes"`
	Comment         string   `json:"comment"`
}

func parseOperativeNoteDemoPayload(payload json.RawMessage) (operativeNoteDemoPayload, error) {
	var parsed operativeNoteDemoPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &parsed) != nil {
		return operativeNoteDemoPayload{}, errors.New("invalid operative-note payload")
	}
	if parsed.RecordMode != operativeNoteDemoMode || !catalogueCodePattern.MatchString(parsed.ProcedureCode) ||
		!catalogueCodePattern.MatchString(parsed.Laterality) || !catalogueCodePattern.MatchString(parsed.SurgeonCode) ||
		!catalogueCodePattern.MatchString(parsed.AnaestheticCode) || len(parsed.DeliveryCodes) > 3 || len(parsed.Comment) > 2000 {
		return operativeNoteDemoPayload{}, errors.New("invalid operative-note fields")
	}
	seen := make(map[string]struct{}, len(parsed.DeliveryCodes))
	for _, code := range parsed.DeliveryCodes {
		if !catalogueCodePattern.MatchString(code) {
			return operativeNoteDemoPayload{}, errors.New("invalid operative-note delivery")
		}
		if _, exists := seen[code]; exists {
			return operativeNoteDemoPayload{}, errors.New("duplicate operative-note delivery")
		}
		seen[code] = struct{}{}
	}
	if parsed.AnaestheticCode == "development_local_anaesthetic" && len(parsed.DeliveryCodes) == 0 {
		return operativeNoteDemoPayload{}, errors.New("local anaesthetic requires a delivery selection")
	}
	if parsed.AnaestheticCode != "development_local_anaesthetic" && len(parsed.DeliveryCodes) != 0 {
		return operativeNoteDemoPayload{}, errors.New("delivery is limited to local anaesthetic in the demo")
	}
	if strings.TrimSpace(parsed.Comment) != parsed.Comment {
		return operativeNoteDemoPayload{}, errors.New("operative-note comment must not have surrounding whitespace")
	}
	return parsed, nil
}

type postgresOperativeNoteCatalogue struct{ pool *pgxpool.Pool }

func (c postgresOperativeNoteCatalogue) codeIsActive(ctx context.Context, category, code string) (bool, error) {
	var active bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM development_operative_note_catalogue WHERE category = $1 AND code = $2 AND active)`, category, code).Scan(&active)
	return active, err
}

func (c postgresOperativeNoteCatalogue) hasDevelopmentCodes(ctx context.Context) (bool, error) {
	var present bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM development_operative_note_catalogue)`).Scan(&present)
	return present, err
}
