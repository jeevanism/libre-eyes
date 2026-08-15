package examination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jeevanism/visionopus/internal/episodes"
)

const (
	correspondenceDemoEventType = "ophthalmology.correspondence_demo"
	correspondenceDemoSchema    = 1
	correspondenceDemoMode      = "demo_correspondence"
)

type correspondenceDemoCatalogue interface {
	codeIsActive(context.Context, string, string) (bool, error)
	hasDemoCodes(context.Context) (bool, error)
}

type CorrespondenceDemoDraftRegistry struct{ catalogue correspondenceDemoCatalogue }

func NewCorrespondenceDemoDraftRegistry(ctx context.Context, pool *pgxpool.Pool, environment string) (*CorrespondenceDemoDraftRegistry, error) {
	if pool == nil {
		return nil, errors.New("correspondence demonstration catalogue database is required")
	}
	return newCorrespondenceDemoDraftRegistry(ctx, postgresCorrespondenceDemoCatalogue{pool: pool}, environment)
}

func newCorrespondenceDemoDraftRegistry(ctx context.Context, catalogue correspondenceDemoCatalogue, environment string) (*CorrespondenceDemoDraftRegistry, error) {
	if catalogue == nil {
		return nil, errors.New("correspondence demonstration catalogue is required")
	}
	if environment == "production" {
		present, err := catalogue.hasDemoCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify production correspondence catalogue: %w", err)
		}
		if present {
			return nil, errors.New("production startup refused: demo correspondence catalogue codes found")
		}
	}
	return &CorrespondenceDemoDraftRegistry{catalogue: catalogue}, nil
}

func (r *CorrespondenceDemoDraftRegistry) Validate(ctx context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if r == nil || eventTypeCode != correspondenceDemoEventType || intent != episodes.DraftIntentCreate || schemaVersion != correspondenceDemoSchema {
		return episodes.ErrInvalidRequest
	}
	parsed, err := parseCorrespondenceDemoPayload(payload)
	if err != nil {
		return episodes.ErrInvalidRequest
	}
	for category, code := range map[string]string{"template": parsed.TemplateCode, "recipient_role": parsed.RecipientRole} {
		active, lookupErr := r.catalogue.codeIsActive(ctx, category, code)
		if lookupErr != nil {
			return fmt.Errorf("look up correspondence catalogue: %w", lookupErr)
		}
		if !active {
			return episodes.ErrInvalidRequest
		}
	}
	return nil
}

type correspondenceDemoPayload struct {
	RecordMode    string `json:"recordMode"`
	TemplateCode  string `json:"templateCode"`
	RecipientRole string `json:"recipientRole"`
	Subject       string `json:"subject"`
	Body          string `json:"body"`
	Footer        string `json:"footer"`
	ClinicDate    string `json:"clinicDate"`
}

func parseCorrespondenceDemoPayload(payload json.RawMessage) (correspondenceDemoPayload, error) {
	var parsed correspondenceDemoPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &parsed) != nil || parsed.RecordMode != correspondenceDemoMode {
		return correspondenceDemoPayload{}, errors.New("invalid correspondence payload")
	}
	if !catalogueCodePattern.MatchString(parsed.TemplateCode) || !catalogueCodePattern.MatchString(parsed.RecipientRole) {
		return correspondenceDemoPayload{}, errors.New("invalid correspondence catalogue code")
	}
	if err := plainTextField(parsed.Subject, 200, true); err != nil {
		return correspondenceDemoPayload{}, err
	}
	if err := plainTextField(parsed.Body, 5000, true); err != nil {
		return correspondenceDemoPayload{}, err
	}
	if err := plainTextField(parsed.Footer, 1000, false); err != nil {
		return correspondenceDemoPayload{}, err
	}
	if parsed.ClinicDate != "" {
		date, err := time.Parse("2006-01-02", parsed.ClinicDate)
		if err != nil || date.After(time.Now()) {
			return correspondenceDemoPayload{}, errors.New("invalid or future clinic date")
		}
	}
	return parsed, nil
}

func plainTextField(value string, max int, required bool) error {
	if strings.TrimSpace(value) != value || (required && value == "") || len(value) > max || strings.ContainsAny(value, "<>") {
		return errors.New("invalid plain-text correspondence field")
	}
	for _, r := range value {
		if r < 0x20 && r != '\n' && r != '\r' && r != '\t' {
			return errors.New("invalid control character in correspondence field")
		}
	}
	return nil
}

type postgresCorrespondenceDemoCatalogue struct{ pool *pgxpool.Pool }

func (c postgresCorrespondenceDemoCatalogue) codeIsActive(ctx context.Context, category, code string) (bool, error) {
	var active bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM development_correspondence_catalogue WHERE category = $1 AND code = $2 AND active)`, category, code).Scan(&active)
	return active, err
}

func (c postgresCorrespondenceDemoCatalogue) hasDemoCodes(ctx context.Context) (bool, error) {
	var present bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM development_correspondence_catalogue)`).Scan(&present)
	return present, err
}
