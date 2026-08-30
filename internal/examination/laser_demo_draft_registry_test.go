package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestLaserDemoDraftRegistryValidate(t *testing.T) {
	r := LaserDemoDraftRegistry{}
	payload := map[string]any{"recordMode": "demo_laser", "profileCode": "demo_laser_v1", "eyeMode": "both", "rightEye": map[string]any{"procedureCode": "demo_procedure_grid"}, "leftEye": map[string]any{"procedureCode": "demo_procedure_grid"}, "siteCode": "demo_site_main", "laserCode": "demo_laser_green", "operatorCode": "demo_operator_a", "treatmentDate": "2026-08-16", "comment": "demo"}
	b, _ := json.Marshal(payload)
	if err := r.Validate(context.Background(), laserDemoEventType, episodes.DraftIntentCreate, 1, b); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	payload["rightEye"].(map[string]any)["procedureCode"] = "real_procedure"
	b, _ = json.Marshal(payload)
	if err := r.Validate(context.Background(), laserDemoEventType, episodes.DraftIntentCreate, 1, b); err == nil {
		t.Fatal("non-demo procedure accepted")
	}
}

func TestLaserDemoProductionGuard(t *testing.T) {
	if _, err := NewLaserDemoDraftRegistry("production"); err == nil {
		t.Fatal("production registry unexpectedly available")
	}
}
