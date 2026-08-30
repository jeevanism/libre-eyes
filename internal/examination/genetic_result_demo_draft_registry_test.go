package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestGeneticResultDemoDraftRegistryValidate(t *testing.T) {
	r := GeneticResultDemoDraftRegistry{}
	body, _ := json.Marshal(map[string]string{"recordMode": "demo_genetic_result", "profileCode": "demo_genetic_result_v1", "testType": "demo_panel", "status": "demo_available", "sourceLabel": "Demo laboratory", "resultDate": "2026-08-01", "summary": "Synthetic summary"})
	if err := r.Validate(context.Background(), geneticResultDemoEventType, episodes.DraftIntentCreate, 1, body); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	bad := []byte(`{"recordMode":"demo_genetic_result","profileCode":"demo_genetic_result_v1","testType":"demo_panel","status":"demo_available","sourceLabel":"Demo laboratory","resultDate":"2999-01-01","summary":""}`)
	if err := r.Validate(context.Background(), geneticResultDemoEventType, episodes.DraftIntentCreate, 1, bad); err == nil {
		t.Fatal("future result date accepted")
	}
}
