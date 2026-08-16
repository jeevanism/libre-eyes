package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

func TestVisualFieldsDemoDraftRegistryValidate(t *testing.T) {
	r := &VisualFieldsDemoDraftRegistry{}
	payload := func(eyes any) json.RawMessage {
		value, _ := json.Marshal(map[string]any{"recordMode": "demo_visual_fields", "strategyCode": "demo_sita_standard", "patternCode": "demo_24_2", "eyes": eyes, "comment": "demo"})
		return value
	}
	if err := r.Validate(context.Background(), visualFieldsDemoEventType, episodes.DraftIntentCreate, 1, payload([]map[string]string{{"eye": "right", "resultCode": "demo_normal"}, {"eye": "left", "resultCode": "demo_field_defect"}})); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if err := r.Validate(context.Background(), visualFieldsDemoEventType, episodes.DraftIntentCreate, 1, payload([]map[string]string{{"eye": "right", "resultCode": "demo_unknown"}})); err == nil {
		t.Fatal("unknown result accepted")
	}
	if err := r.Validate(context.Background(), visualFieldsDemoEventType, episodes.DraftIntentCreate, 1, payload([]map[string]string{{"eye": "right", "resultCode": "demo_normal"}, {"eye": "right", "resultCode": "demo_normal"}})); err == nil {
		t.Fatal("duplicate eye accepted")
	}
}

func TestVisualFieldsDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := NewVisualFieldsDemoDraftRegistry("production"); err == nil {
		t.Fatal("production registry accepted")
	}
}
