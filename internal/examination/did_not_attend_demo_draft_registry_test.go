package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestDidNotAttendDemoDraftRegistryValidate(t *testing.T) {
	registry := DidNotAttendDemoDraftRegistry{}
	payload := didNotAttendDemoPayload{RecordMode: "demo_did_not_attend", ProfileCode: "demo_did_not_attend_v1", EventDate: "2026-08-15", Source: "demo_clinic_flow", Comment: "Synthetic note"}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Validate(context.Background(), didNotAttendDemoEventType, episodes.DraftIntentCreate, 1, raw); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	for name, mutate := range map[string]func(*didNotAttendDemoPayload){
		"future date":       func(p *didNotAttendDemoPayload) { p.EventDate = "2999-01-01" },
		"unknown source":    func(p *didNotAttendDemoPayload) { p.Source = "real_schedule" },
		"untrimmed comment": func(p *didNotAttendDemoPayload) { p.Comment = " note" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := payload
			mutate(&candidate)
			raw, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if err := registry.Validate(context.Background(), didNotAttendDemoEventType, episodes.DraftIntentCreate, 1, raw); err != episodes.ErrInvalidRequest {
				t.Fatalf("expected invalid request, got %v", err)
			}
		})
	}
}

func TestDidNotAttendDemoDraftRegistryRejectsProduction(t *testing.T) {
	if _, err := NewDidNotAttendDemoDraftRegistry("production"); err == nil {
		t.Fatal("expected production startup guard")
	}
}
