package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

func TestCatpromDemoDraftRegistryValidate(t *testing.T) {
	answers := make([]map[string]string, 0, 6)
	for i := 1; i <= 6; i++ {
		answers = append(answers, map[string]string{"questionCode": "demo_q" + string(rune('0'+i)), "answerCode": "demo_q" + string(rune('0'+i)) + "_a1"})
	}
	payload, _ := json.Marshal(map[string]any{"recordMode": "demo_catprom", "profileCode": "demo_catprom_v1", "laterality": "not_applicable", "answers": answers})
	if err := (CatpromDemoDraftRegistry{}).Validate(context.Background(), catpromDemoEventType, episodes.DraftIntentCreate, 1, payload); err != nil {
		t.Fatal(err)
	}
}

func TestCatpromDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := NewCatpromDemoDraftRegistry("production"); err == nil {
		t.Fatal("expected production guard")
	}
}
