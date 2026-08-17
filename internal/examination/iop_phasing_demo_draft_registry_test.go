package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

func TestIOPPhasingDemoDraftRegistryValidate(t *testing.T) {
	payload := map[string]any{"recordMode": "demo_iop_phasing", "profileCode": "demo_iop_phasing_v1", "eyeMode": "right", "rightEye": map[string]any{"instrumentCode": "demo_goldmann", "dilated": false, "comment": "", "readings": []map[string]any{{"value": 18, "measurementTime": "09:00"}}}}
	raw, _ := json.Marshal(payload)
	if err := (IOPPhasingDemoDraftRegistry{}).Validate(context.Background(), iopPhasingDemoEventType, episodes.DraftIntentCreate, 1, raw); err != nil { t.Fatal(err) }
	payload["rightEye"].(map[string]any)["readings"] = []map[string]any{{"value": 101, "measurementTime": "09:00"}}
	raw, _ = json.Marshal(payload)
	if err := (IOPPhasingDemoDraftRegistry{}).Validate(context.Background(), iopPhasingDemoEventType, episodes.DraftIntentCreate, 1, raw); err != episodes.ErrInvalidRequest { t.Fatalf("got %v", err) }
}

func TestIOPPhasingDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := NewIOPPhasingDemoDraftRegistry("production"); err == nil { t.Fatal("expected production guard") }
}
