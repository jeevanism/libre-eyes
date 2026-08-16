package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

func TestIntravitrealInjectionDemoRegistryValidate(t *testing.T) {
	r := IntravitrealInjectionDemoDraftRegistry{}
	payload := map[string]any{
		"recordMode": "demo_intravitreal_injection", "profileCode": "demo_intravitreal_injection_v1", "eyeMode": "right",
		"rightEye": map[string]any{"drugCode": "demo_drug_a", "siteCode": "demo_site_superotemporal", "anaestheticCode": "demo_anaesthetic_topical", "injectionNumber": 1, "plannedDate": "2026-08-16", "postCheck": "demo_not_recorded"}, "leftEye": nil, "note": "demo",
	}
	b, _ := json.Marshal(payload)
	if err := r.Validate(context.Background(), intravitrealInjectionDemoEventType, episodes.DraftIntentCreate, 1, b); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	payload["rightEye"].(map[string]any)["injectionNumber"] = 11
	b, _ = json.Marshal(payload)
	if err := r.Validate(context.Background(), intravitrealInjectionDemoEventType, episodes.DraftIntentCreate, 1, b); err == nil {
		t.Fatal("out-of-range injection accepted")
	}
}

func TestIntravitrealInjectionDemoProductionGuard(t *testing.T) {
	if _, err := NewIntravitrealInjectionDemoDraftRegistry("production"); err == nil {
		t.Fatal("production registry unexpectedly available")
	}
}
