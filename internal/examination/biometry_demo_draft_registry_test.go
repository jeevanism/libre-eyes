package examination

import (
	"context"
	"encoding/json"
	"github.com/jeevanism/visionopus/internal/episodes"
	"testing"
)

func TestBiometryDemoDraftRegistryValidate(t *testing.T) {
	r := BiometryDemoDraftRegistry{}
	eye := map[string]any{"axialLength": "23.50", "r1": "7.80", "r2": "7.70", "r1Axis": 90, "r2Axis": 0, "acd": "3.20", "wtw": "11.80"}
	raw, _ := json.Marshal(map[string]any{"recordMode": "demo_biometry", "profileCode": "demo_biometry_v1", "deviceCode": "demo_manual", "lensCode": "demo_none", "measurementDate": "2026-08-16", "rightEye": eye, "leftEye": nil, "comment": "demo"})
	if err := r.Validate(context.Background(), biometryDemoEventType, episodes.DraftIntentCreate, 1, raw); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if err := r.Validate(context.Background(), biometryDemoEventType, episodes.DraftIntentUpdate, 1, raw); err != nil {
		t.Fatalf("valid update rejected: %v", err)
	}
	bad := append([]byte(nil), raw...)
	bad = []byte(`{"recordMode":"demo_biometry","profileCode":"demo_biometry_v1","deviceCode":"demo_manual","lensCode":"demo_none","measurementDate":"2999-01-01","rightEye":null,"leftEye":null}`)
	if err := r.Validate(context.Background(), biometryDemoEventType, episodes.DraftIntentCreate, 1, bad); err == nil {
		t.Fatal("invalid payload accepted")
	}
}
func TestBiometryDemoProductionGuard(t *testing.T) {
	if _, err := NewBiometryDemoDraftRegistry("production"); err == nil {
		t.Fatal("production registry accepted")
	}
}
