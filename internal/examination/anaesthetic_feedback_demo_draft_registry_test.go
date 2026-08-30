package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestAnaestheticFeedbackDemoDraftRegistryValidate(t *testing.T) {
	r := AnaestheticFeedbackDemoDraftRegistry{}
	body, _ := json.Marshal(map[string]string{"recordMode": "demo_anaesthetic_feedback", "profileCode": "demo_anaesthetic_feedback_v1", "anaesthetic": "demo_local", "satisfaction": "demo_satisfied", "note": "Synthetic feedback"})
	if err := r.Validate(context.Background(), anaestheticFeedbackDemoEventType, episodes.DraftIntentCreate, 1, body); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	bad := []byte(`{"recordMode":"demo_anaesthetic_feedback","profileCode":"demo_anaesthetic_feedback_v1","anaesthetic":"real","satisfaction":"demo_satisfied","note":""}`)
	if err := r.Validate(context.Background(), anaestheticFeedbackDemoEventType, episodes.DraftIntentCreate, 1, bad); err == nil {
		t.Fatal("unsupported anaesthetic accepted")
	}
}
