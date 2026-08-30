package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestTherapyIntentDemoDraftRegistryValidate(t *testing.T) {
	r := TherapyIntentDemoDraftRegistry{}
	payload := func(treatment, laterality, note string) json.RawMessage {
		body, _ := json.Marshal(map[string]string{
			"recordMode": "demo_therapy_intent", "profileCode": "demo_therapy_intent_v1",
			"treatment": treatment, "laterality": laterality, "note": note,
		})
		return body
	}
	if err := r.Validate(context.Background(), therapyIntentDemoEventType, episodes.DraftIntentCreate, 1, payload("demo_anti_vegf", "right", "planning")); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if err := r.Validate(context.Background(), therapyIntentDemoEventType, episodes.DraftIntentCreate, 1, payload("clinical_drug", "right", "")); err == nil {
		t.Fatal("unsupported treatment accepted")
	}
	if err := r.Validate(context.Background(), therapyIntentDemoEventType, episodes.DraftIntentCreate, 1, payload("demo_steroid", "right", string(make([]byte, 501)))); err == nil {
		t.Fatal("oversized note accepted")
	}
}
