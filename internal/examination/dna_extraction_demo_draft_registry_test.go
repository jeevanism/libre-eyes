package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestDNAExtractionDemoDraftRegistryAcceptsBoundedPayload(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"recordMode": "demo_dna_extraction", "profileCode": "demo_dna_extraction_v1",
		"sampleLabel": "Demo sample A", "status": "stored", "storageAddress": "Demo box A / slot 01",
		"extractionDate": "2026-08-19", "volume": "20", "comment": "Synthetic only",
	})
	registry := DNAExtractionDemoDraftRegistry{}
	if err := registry.Validate(context.Background(), dnaExtractionDemoEventType, episodes.DraftIntentCreate, 1, payload); err != nil {
		t.Fatalf("expected payload to validate: %v", err)
	}
}

func TestDNAExtractionDemoDraftRegistryRejectsFutureShapeAndUnknownStatus(t *testing.T) {
	payload := json.RawMessage(`{"recordMode":"demo_dna_extraction","profileCode":"demo_dna_extraction_v1","sampleLabel":"x","status":"result_available","storageAddress":"x"}`)
	if err := (DNAExtractionDemoDraftRegistry{}).Validate(context.Background(), dnaExtractionDemoEventType, episodes.DraftIntentCreate, 1, payload); err == nil {
		t.Fatal("expected unknown status to be rejected")
	}
}
