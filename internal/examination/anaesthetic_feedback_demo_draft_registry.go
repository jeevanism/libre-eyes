package examination

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const anaestheticFeedbackDemoEventType = "ophthalmology.anaesthetic_feedback_demo"

type AnaestheticFeedbackDemoDraftRegistry struct{}

func NewAnaestheticFeedbackDemoDraftRegistry(environment string) (*AnaestheticFeedbackDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration anaesthetic feedback is enabled")
	}
	return &AnaestheticFeedbackDemoDraftRegistry{}, nil
}

type anaestheticFeedbackDemoPayload struct {
	RecordMode   string `json:"recordMode"`
	ProfileCode  string `json:"profileCode"`
	Anaesthetic  string `json:"anaesthetic"`
	Satisfaction string `json:"satisfaction"`
	Note         string `json:"note"`
}

func (AnaestheticFeedbackDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != anaestheticFeedbackDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 16*1024 {
		return episodes.ErrInvalidRequest
	}
	var p anaestheticFeedbackDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_anaesthetic_feedback" || p.ProfileCode != "demo_anaesthetic_feedback_v1" || len(p.Note) > 500 {
		return episodes.ErrInvalidRequest
	}
	if p.Anaesthetic != "demo_general" && p.Anaesthetic != "demo_local" && p.Anaesthetic != "demo_none" {
		return episodes.ErrInvalidRequest
	}
	if p.Satisfaction != "demo_very_satisfied" && p.Satisfaction != "demo_satisfied" && p.Satisfaction != "demo_neutral" && p.Satisfaction != "demo_dissatisfied" {
		return episodes.ErrInvalidRequest
	}
	return nil
}
