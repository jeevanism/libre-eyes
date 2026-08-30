package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

type labCatalogueStub struct{ entry labResultCatalogueEntry }

func (s labCatalogueStub) lookup(context.Context, string) (labResultCatalogueEntry, error) {
	return s.entry, nil
}
func (s labCatalogueStub) hasDemoCodes(context.Context) (bool, error) { return false, nil }

func TestLabResultsDemoValidate(t *testing.T) {
	min, max := 0.0, 20.0
	r := &LabResultsDemoDraftRegistry{catalogue: labCatalogueStub{entry: labResultCatalogueEntry{Code: "demo_lab_hba1c", Kind: "numeric", DefaultUnit: "%", HardMin: &min, HardMax: &max}}}
	payload := func(value string) json.RawMessage {
		b, _ := json.Marshal(map[string]any{"recordMode": labResultsDemoMode, "isSynthetic": true, "resultTypeCode": "demo_lab_hba1c", "fieldKind": "numeric", "value": value, "unit": "", "observedAt": "09:30", "comment": ""})
		return b
	}
	if err := r.Validate(context.Background(), labResultsDemoEventType, episodes.DraftIntentCreate, 1, payload("5.5")); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if err := r.Validate(context.Background(), labResultsDemoEventType, episodes.DraftIntentCreate, 1, payload("21")); err != episodes.ErrInvalidRequest {
		t.Fatalf("hard bound error = %v", err)
	}
}
