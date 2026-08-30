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
	consentDemoEventType = "ophthalmology.consent_demo"
	consentDemoSchema    = 1
	consentDemoMode      = "development_synthetic_consent"
)

type consentDemoCatalogue interface {
	codeIsActive(context.Context, string, string) (bool, error)
	hasDevelopmentCodes(context.Context) (bool, error)
}

type ConsentDemoDraftRegistry struct{ catalogue consentDemoCatalogue }

func NewConsentDemoDraftRegistry(ctx context.Context, pool *pgxpool.Pool, environment string) (*ConsentDemoDraftRegistry, error) {
	if pool == nil {
		return nil, errors.New("consent demonstration catalogue database is required")
	}
	catalogue := postgresConsentDemoCatalogue{pool: pool}
	if environment == "production" {
		present, err := catalogue.hasDevelopmentCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify production consent catalogue: %w", err)
		}
		if present {
			return nil, errors.New("production startup refused: development consent catalogue codes found")
		}
	}
	return newConsentDemoDraftRegistry(ctx, catalogue, environment)
}

func newConsentDemoDraftRegistry(ctx context.Context, catalogue consentDemoCatalogue, environment string) (*ConsentDemoDraftRegistry, error) {
	if catalogue == nil {
		return nil, errors.New("consent demonstration catalogue is required")
	}
	if environment == "production" {
		present, err := catalogue.hasDevelopmentCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify production consent catalogue: %w", err)
		}
		if present {
			return nil, errors.New("production startup refused: development consent catalogue codes found")
		}
	}
	return &ConsentDemoDraftRegistry{catalogue: catalogue}, nil
}

func (r *ConsentDemoDraftRegistry) Validate(ctx context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if r == nil || eventTypeCode != consentDemoEventType || intent != episodes.DraftIntentCreate || schemaVersion != consentDemoSchema {
		return episodes.ErrInvalidRequest
	}
	parsed, err := parseConsentDemoPayload(payload)
	if err != nil {
		return episodes.ErrInvalidRequest
	}
	for category, code := range map[string]string{"form_type": parsed.FormTypeCode, "procedure": parsed.ProcedureCode, "laterality": parsed.Laterality, "anaesthetic": parsed.AnaestheticCode} {
		active, lookupErr := r.catalogue.codeIsActive(ctx, category, code)
		if lookupErr != nil {
			return fmt.Errorf("look up consent catalogue: %w", lookupErr)
		}
		if !active {
			return episodes.ErrInvalidRequest
		}
	}
	return nil
}

type consentDemoPayload struct {
	RecordMode      string `json:"recordMode"`
	FormTypeCode    string `json:"formTypeCode"`
	ProcedureCode   string `json:"procedureCode"`
	Laterality      string `json:"laterality"`
	AnaestheticCode string `json:"anaestheticCode"`
	Comment         string `json:"comment"`
}

func parseConsentDemoPayload(payload json.RawMessage) (consentDemoPayload, error) {
	var parsed consentDemoPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &parsed) != nil || parsed.RecordMode != consentDemoMode || len(parsed.Comment) > 2000 || strings.TrimSpace(parsed.Comment) != parsed.Comment {
		return consentDemoPayload{}, errors.New("invalid consent payload")
	}
	for _, code := range []string{parsed.FormTypeCode, parsed.ProcedureCode, parsed.Laterality, parsed.AnaestheticCode} {
		if !catalogueCodePattern.MatchString(code) {
			return consentDemoPayload{}, errors.New("invalid consent catalogue code")
		}
	}
	return parsed, nil
}

type postgresConsentDemoCatalogue struct{ pool *pgxpool.Pool }

func (c postgresConsentDemoCatalogue) codeIsActive(ctx context.Context, category, code string) (bool, error) {
	var active bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM development_consent_catalogue WHERE category = $1 AND code = $2 AND active)`, category, code).Scan(&active)
	return active, err
}
func (c postgresConsentDemoCatalogue) hasDevelopmentCodes(ctx context.Context) (bool, error) {
	var present bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM development_consent_catalogue)`).Scan(&present)
	return present, err
}
