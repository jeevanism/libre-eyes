package examination

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const geneticResultDemoEventType = "ophthalmology.genetic_result_demo"

type GeneticResultDemoDraftRegistry struct{}

func NewGeneticResultDemoDraftRegistry(environment string) (*GeneticResultDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration genetic results are enabled")
	}
	return &GeneticResultDemoDraftRegistry{}, nil
}

type geneticResultDemoPayload struct {
	RecordMode  string `json:"recordMode"`
	ProfileCode string `json:"profileCode"`
	TestType    string `json:"testType"`
	Status      string `json:"status"`
	SourceLabel string `json:"sourceLabel"`
	ResultDate  string `json:"resultDate"`
	Summary     string `json:"summary"`
}

func (GeneticResultDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != geneticResultDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 16*1024 {
		return episodes.ErrInvalidRequest
	}
	var p geneticResultDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_genetic_result" || p.ProfileCode != "demo_genetic_result_v1" || len(p.SourceLabel) == 0 || len(p.SourceLabel) > 120 || len(p.Summary) > 500 {
		return episodes.ErrInvalidRequest
	}
	if p.TestType != "demo_panel" && p.TestType != "demo_single_gene" && p.TestType != "demo_carrier_screen" {
		return episodes.ErrInvalidRequest
	}
	if p.Status != "demo_pending" && p.Status != "demo_available" && p.Status != "demo_withdrawn" {
		return episodes.ErrInvalidRequest
	}
	d, err := time.Parse("2006-01-02", p.ResultDate)
	if err != nil || d.After(time.Now().UTC()) {
		return episodes.ErrInvalidRequest
	}
	return nil
}
