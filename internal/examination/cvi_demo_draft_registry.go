package examination

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const cviDemoEventType = "ophthalmology.cvi_demo"

type CviDemoDraftRegistry struct{}

func NewCviDemoDraftRegistry(environment string) (*CviDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration CVI is enabled")
	}
	return &CviDemoDraftRegistry{}, nil
}

type cviDemoPayload struct {
	RecordMode      string `json:"recordMode"`
	ProfileCode     string `json:"profileCode"`
	Status          string `json:"status"`
	PreferredFormat string `json:"preferredFormat"`
	Note            string `json:"note"`
}

func (CviDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != cviDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 16*1024 {
		return episodes.ErrInvalidRequest
	}
	var p cviDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_cvi" || p.ProfileCode != "demo_cvi_v1" || len(p.Note) > 500 {
		return episodes.ErrInvalidRequest
	}
	if p.Status != "demo_new" && p.Status != "demo_in_review" && p.Status != "demo_ready_for_discussion" {
		return episodes.ErrInvalidRequest
	}
	if p.PreferredFormat != "demo_large_print" && p.PreferredFormat != "demo_audio" && p.PreferredFormat != "demo_digital" {
		return episodes.ErrInvalidRequest
	}
	return nil
}
