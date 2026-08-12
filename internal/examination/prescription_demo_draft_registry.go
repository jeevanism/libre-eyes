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
	prescriptionDemoEventType = "ophthalmology.prescription_demo"
	prescriptionDemoSchema    = 1
	prescriptionDemoMode      = "development_synthetic_medication_order"
)

type prescriptionDemoCatalogue interface {
	codeIsActive(context.Context, string, string) (bool, error)
	hasDevelopmentCodes(context.Context) (bool, error)
}

type PrescriptionDemoDraftRegistry struct{ catalogue prescriptionDemoCatalogue }

func NewPrescriptionDemoDraftRegistry(ctx context.Context, pool *pgxpool.Pool, environment string) (*PrescriptionDemoDraftRegistry, error) {
	if pool == nil {
		return nil, errors.New("prescription demonstration catalogue database is required")
	}
	catalogue := postgresPrescriptionDemoCatalogue{pool: pool}
	if environment == "production" {
		present, err := catalogue.hasDevelopmentCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify production prescription catalogue: %w", err)
		}
		if present {
			return nil, errors.New("production startup refused: development prescription catalogue codes found")
		}
	}
	return &PrescriptionDemoDraftRegistry{catalogue: catalogue}, nil
}

func (r *PrescriptionDemoDraftRegistry) Validate(ctx context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if r == nil || eventTypeCode != prescriptionDemoEventType || intent != episodes.DraftIntentCreate || schemaVersion != prescriptionDemoSchema {
		return episodes.ErrInvalidRequest
	}
	parsed, err := parsePrescriptionDemoPayload(payload)
	if err != nil {
		return episodes.ErrInvalidRequest
	}
	for _, item := range parsed.Items {
		for category, code := range map[string]string{"medication": item.MedicationCode, "route": item.RouteCode, "frequency": item.FrequencyCode, "duration": item.DurationCode, "laterality": item.Laterality} {
			active, lookupErr := r.catalogue.codeIsActive(ctx, category, code)
			if lookupErr != nil {
				return fmt.Errorf("look up prescription catalogue: %w", lookupErr)
			}
			if !active {
				return episodes.ErrInvalidRequest
			}
		}
		if item.Taper != nil {
			for category, code := range map[string]string{"frequency": item.Taper.FrequencyCode, "duration": item.Taper.DurationCode} {
				active, lookupErr := r.catalogue.codeIsActive(ctx, category, code)
				if lookupErr != nil {
					return fmt.Errorf("look up prescription taper catalogue: %w", lookupErr)
				}
				if !active {
					return episodes.ErrInvalidRequest
				}
			}
		}
	}
	return nil
}

type prescriptionDemoPayload struct {
	RecordMode    string                 `json:"recordMode"`
	HeaderComment string                 `json:"headerComment"`
	Items         []prescriptionDemoItem `json:"items"`
}
type prescriptionDemoItem struct {
	MedicationCode string                 `json:"medicationCode"`
	Dose           string                 `json:"dose"`
	DoseUnit       string                 `json:"doseUnit"`
	RouteCode      string                 `json:"routeCode"`
	FrequencyCode  string                 `json:"frequencyCode"`
	DurationCode   string                 `json:"durationCode"`
	Laterality     string                 `json:"laterality"`
	StartDate      string                 `json:"startDate"`
	Comment        string                 `json:"comment"`
	Taper          *prescriptionDemoTaper `json:"taper"`
}
type prescriptionDemoTaper struct {
	Dose          string `json:"dose"`
	FrequencyCode string `json:"frequencyCode"`
	DurationCode  string `json:"durationCode"`
}

func parsePrescriptionDemoPayload(payload json.RawMessage) (prescriptionDemoPayload, error) {
	var parsed prescriptionDemoPayload
	if len(payload) == 0 || len(payload) > 32*1024 || decodeStrictJSON(payload, &parsed) != nil || parsed.RecordMode != prescriptionDemoMode || len(parsed.Items) < 1 || len(parsed.Items) > 3 || len(parsed.HeaderComment) > 2000 {
		return prescriptionDemoPayload{}, errors.New("invalid prescription payload")
	}
	for _, item := range parsed.Items {
		if !catalogueCodePattern.MatchString(item.MedicationCode) || !catalogueCodePattern.MatchString(item.RouteCode) || !catalogueCodePattern.MatchString(item.FrequencyCode) || !catalogueCodePattern.MatchString(item.DurationCode) || !catalogueCodePattern.MatchString(item.Laterality) || strings.TrimSpace(item.Dose) != item.Dose || item.Dose == "" || len(item.Dose) > 40 || item.DoseUnit == "" || len(item.DoseUnit) > 45 || len(item.Comment) > 1000 {
			return prescriptionDemoPayload{}, errors.New("invalid prescription item")
		}
		if item.Taper != nil && (item.Taper.Dose == "" || len(item.Taper.Dose) > 40 || !catalogueCodePattern.MatchString(item.Taper.FrequencyCode) || !catalogueCodePattern.MatchString(item.Taper.DurationCode)) {
			return prescriptionDemoPayload{}, errors.New("invalid prescription taper")
		}
		if _, err := parseISODate(item.StartDate); err != nil {
			return prescriptionDemoPayload{}, errors.New("invalid prescription start date")
		}
	}
	return parsed, nil
}

func parseISODate(value string) (string, error) {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return "", err
	}
	return value, nil
}

type postgresPrescriptionDemoCatalogue struct{ pool *pgxpool.Pool }

func (c postgresPrescriptionDemoCatalogue) codeIsActive(ctx context.Context, category, code string) (bool, error) {
	var active bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM development_prescription_catalogue WHERE category = $1 AND code = $2 AND active)`, category, code).Scan(&active)
	return active, err
}
func (c postgresPrescriptionDemoCatalogue) hasDevelopmentCodes(ctx context.Context) (bool, error) {
	var present bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM development_prescription_catalogue)`).Scan(&present)
	return present, err
}
