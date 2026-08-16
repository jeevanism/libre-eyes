package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

func TestOperationChecklistDemoDraftRegistryValidate(t *testing.T) {
	registry := OperationChecklistDemoDraftRegistry{}
	payload := operationChecklistDemoPayload{
		RecordMode:  "demo_operation_checklist",
		ProfileCode: "demo_operation_checklist_v1",
		EyeMode:     "right",
		Questions: []operationChecklistDemoQuestion{
			{Code: "identity_check", Answer: "demo_yes"},
			{Code: "procedure_confirmed", Answer: "demo_not_recorded"},
			{Code: "escort_discussed", Answer: "demo_no"},
		},
		Note: "demo note",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Validate(context.Background(), operationChecklistDemoEventType, episodes.DraftIntentCreate, 1, raw); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*operationChecklistDemoPayload)
	}{
		{name: "missing question", mutate: func(p *operationChecklistDemoPayload) { p.Questions = p.Questions[:2] }},
		{name: "duplicate question", mutate: func(p *operationChecklistDemoPayload) { p.Questions[2].Code = p.Questions[0].Code }},
		{name: "unknown answer", mutate: func(p *operationChecklistDemoPayload) { p.Questions[0].Answer = "clinical_yes" }},
		{name: "untrimmed note", mutate: func(p *operationChecklistDemoPayload) { p.Note = " demo note" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := payload
			candidate.Questions = append([]operationChecklistDemoQuestion(nil), payload.Questions...)
			tt.mutate(&candidate)
			raw, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if err := registry.Validate(context.Background(), operationChecklistDemoEventType, episodes.DraftIntentCreate, 1, raw); err != episodes.ErrInvalidRequest {
				t.Fatalf("expected invalid request, got %v", err)
			}
		})
	}
}

func TestOperationChecklistDemoDraftRegistryRejectsProduction(t *testing.T) {
	if _, err := NewOperationChecklistDemoDraftRegistry("production"); err == nil {
		t.Fatal("expected production startup guard")
	}
}
