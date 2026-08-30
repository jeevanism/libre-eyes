package examination

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

type fakePrescriptionCatalogue struct {
	active      map[string]bool
	development bool
}

func (f fakePrescriptionCatalogue) codeIsActive(_ context.Context, category, code string) (bool, error) {
	return f.active[category+":"+code], nil
}
func (f fakePrescriptionCatalogue) hasDevelopmentCodes(context.Context) (bool, error) {
	return f.development, nil
}

func TestPrescriptionDemoDraftRegistryValidate(t *testing.T) {
	r := &PrescriptionDemoDraftRegistry{catalogue: fakePrescriptionCatalogue{active: map[string]bool{
		"medication:development_lubricating_drop": true, "route:development_topical_eye": true, "frequency:development_once_daily": true,
		"duration:development_five_days": true, "laterality:development_right": true,
	}}}
	payload := json.RawMessage(`{"recordMode":"development_synthetic_medication_order","items":[{"medicationCode":"development_lubricating_drop","dose":"1","doseUnit":"drop","routeCode":"development_topical_eye","frequencyCode":"development_once_daily","durationCode":"development_five_days","laterality":"development_right","startDate":"2026-08-12","comment":"","taper":null}]}`)
	if err := r.Validate(context.Background(), prescriptionDemoEventType, episodes.DraftIntentCreate, 1, payload); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
}

func TestPrescriptionDemoDraftRegistryRejectsUnknownAndOversizedPayloads(t *testing.T) {
	r := &PrescriptionDemoDraftRegistry{catalogue: fakePrescriptionCatalogue{active: map[string]bool{}}}
	for name, payload := range map[string]json.RawMessage{
		"wrong mode":   json.RawMessage(`{"recordMode":"issued","items":[]}`),
		"unknown code": json.RawMessage(`{"recordMode":"development_synthetic_medication_order","items":[{"medicationCode":"development_unknown","dose":"1","doseUnit":"drop","routeCode":"development_topical_eye","frequencyCode":"development_once_daily","durationCode":"development_five_days","laterality":"development_right","startDate":"2026-08-12"}]}`),
	} {
		t.Run(name, func(t *testing.T) {
			if err := r.Validate(context.Background(), prescriptionDemoEventType, episodes.DraftIntentCreate, 1, payload); !errors.Is(err, episodes.ErrInvalidRequest) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	items := make([]map[string]any, 4)
	for i := range items {
		items[i] = map[string]any{"medicationCode": "development_lubricating_drop", "dose": "1", "doseUnit": "drop", "routeCode": "development_topical_eye", "frequencyCode": "development_once_daily", "durationCode": "development_five_days", "laterality": "development_right", "startDate": "2026-08-12"}
	}
	payload, _ := json.Marshal(map[string]any{"recordMode": prescriptionDemoMode, "items": items})
	if err := r.Validate(context.Background(), prescriptionDemoEventType, episodes.DraftIntentCreate, 1, payload); !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("oversized error = %v", err)
	}
}

func TestPrescriptionDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := NewPrescriptionDemoDraftRegistry(context.Background(), nil, "production"); err == nil {
		t.Fatal("nil pool should fail")
	}
}
