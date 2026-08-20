package examination

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const therapyIntentDemoEventType = "ophthalmology.therapy_intent_demo"

type TherapyIntentDemoDraftRegistry struct{}

func NewTherapyIntentDemoDraftRegistry(environment string) (*TherapyIntentDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration therapy intent is enabled")
	}
	return &TherapyIntentDemoDraftRegistry{}, nil
}

type therapyIntentDemoPayload struct {
	RecordMode  string `json:"recordMode"`
	ProfileCode string `json:"profileCode"`
	Treatment   string `json:"treatment"`
	Laterality  string `json:"laterality"`
	Note        string `json:"note"`
}

func (TherapyIntentDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != therapyIntentDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 16*1024 {
		return episodes.ErrInvalidRequest
	}
	var p therapyIntentDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_therapy_intent" || p.ProfileCode != "demo_therapy_intent_v1" || len(p.Note) > 500 {
		return episodes.ErrInvalidRequest
	}
	if p.Treatment != "demo_anti_vegf" && p.Treatment != "demo_steroid" && p.Treatment != "demo_observation" {
		return episodes.ErrInvalidRequest
	}
	if p.Laterality != "right" && p.Laterality != "left" && p.Laterality != "bilateral" && p.Laterality != "not_applicable" {
		return episodes.ErrInvalidRequest
	}
	return nil
}
