package examination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const (
	labResultsDemoEventType = "ophthalmology.lab_result_demo"
	labResultsDemoSchema    = 1
	labResultsDemoMode      = "demo_lab_result"
)

type labResultCatalogue interface {
	lookup(context.Context, string) (labResultCatalogueEntry, error)
	hasDemoCodes(context.Context) (bool, error)
}

type labResultCatalogueEntry struct {
	Code, Kind, DefaultUnit                string
	HardMin, HardMax, NormalMin, NormalMax *float64
	Choices                                []string
}

type LabResultsDemoDraftRegistry struct{ catalogue labResultCatalogue }

func NewLabResultsDemoDraftRegistry(ctx context.Context, pool *pgxpool.Pool, environment string) (*LabResultsDemoDraftRegistry, error) {
	if pool == nil {
		return nil, errors.New("lab results catalogue database is required")
	}
	c := postgresLabResultCatalogue{pool: pool}
	if environment == "production" {
		present, err := c.hasDemoCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("verify production lab results catalogue: %w", err)
		}
		if present {
			return nil, errors.New("production startup refused: demonstration lab results catalogue codes found")
		}
	}
	return &LabResultsDemoDraftRegistry{catalogue: c}, nil
}

func (r *LabResultsDemoDraftRegistry) Validate(ctx context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if r == nil || eventTypeCode != labResultsDemoEventType || intent != episodes.DraftIntentCreate || schemaVersion != labResultsDemoSchema {
		return episodes.ErrInvalidRequest
	}
	p, err := parseLabResultPayload(payload)
	if err != nil {
		return episodes.ErrInvalidRequest
	}
	c, err := r.catalogue.lookup(ctx, p.ResultTypeCode)
	if err != nil {
		return fmt.Errorf("look up lab result catalogue: %w", err)
	}
	if c.Code == "" || !strings.HasPrefix(c.Code, "demo_lab_") || c.Kind != p.FieldKind || (p.Unit != "" && len(p.Unit) > 30) {
		return episodes.ErrInvalidRequest
	}
	if p.Unit == "" {
		p.Unit = c.DefaultUnit
	}
	if c.Kind == "numeric" {
		value, parseErr := strconv.ParseFloat(p.Value, 64)
		if parseErr != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return episodes.ErrInvalidRequest
		}
		if c.HardMin != nil && value < *c.HardMin || c.HardMax != nil && value > *c.HardMax {
			return episodes.ErrInvalidRequest
		}
	} else {
		found := false
		for _, choice := range c.Choices {
			if p.Value == choice {
				found = true
				break
			}
		}
		if !found {
			return episodes.ErrInvalidRequest
		}
	}
	return nil
}

type labResultPayload struct {
	RecordMode, ResultTypeCode, FieldKind, Value, Unit, ObservedAt, Comment string
	IsSynthetic                                                             bool
}

func parseLabResultPayload(payload json.RawMessage) (labResultPayload, error) {
	var p labResultPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &p) != nil || p.RecordMode != labResultsDemoMode || !p.IsSynthetic || !catalogueCodePattern.MatchString(p.ResultTypeCode) || (p.FieldKind != "numeric" && p.FieldKind != "choice") || strings.TrimSpace(p.Value) != p.Value || p.Value == "" || len(p.Value) > 120 || len(p.Comment) > 1000 || strings.ContainsAny(p.Comment, "<>") || len(p.ObservedAt) != 5 {
		return labResultPayload{}, errors.New("invalid lab result payload")
	}
	if _, err := time.Parse("15:04", p.ObservedAt); err != nil {
		return labResultPayload{}, errors.New("invalid observed time")
	}
	return p, nil
}

type postgresLabResultCatalogue struct{ pool *pgxpool.Pool }

func (c postgresLabResultCatalogue) lookup(ctx context.Context, code string) (labResultCatalogueEntry, error) {
	var e labResultCatalogueEntry
	var choices []byte
	err := c.pool.QueryRow(ctx, `SELECT code, field_kind, default_unit, hard_min, hard_max, normal_min, normal_max, choices FROM development_lab_results_catalogue WHERE code=$1 AND active AND institution_id IS NULL`, code).Scan(&e.Code, &e.Kind, &e.DefaultUnit, &e.HardMin, &e.HardMax, &e.NormalMin, &e.NormalMax, &choices)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return e, nil
		}
		return e, err
	}
	_ = json.Unmarshal(choices, &e.Choices)
	return e, nil
}
func (c postgresLabResultCatalogue) hasDemoCodes(ctx context.Context) (bool, error) {
	var present bool
	err := c.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM development_lab_results_catalogue WHERE code LIKE 'demo_lab_%')`).Scan(&present)
	return present, err
}
