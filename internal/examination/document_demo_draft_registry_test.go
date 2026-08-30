package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestDocumentDemoDraftRegistry(t *testing.T) {
	registry := DocumentDemoDraftRegistry{}
	valid := func() json.RawMessage {
		return json.RawMessage(`{"recordMode":"demo_document","profileCode":"demo_document_v1","documentType":"demo_scan_summary","title":"OCT summary","laterality":"right","documentDate":"2026-08-17","comment":"Demo only"}`)
	}
	tests := []struct {
		name    string
		payload json.RawMessage
		wantErr bool
	}{
		{name: "valid", payload: valid()},
		{name: "future date", payload: json.RawMessage(`{"recordMode":"demo_document","profileCode":"demo_document_v1","documentType":"demo_scan_summary","title":"OCT","documentDate":"2999-01-01"}`), wantErr: true},
		{name: "unknown type", payload: json.RawMessage(`{"recordMode":"demo_document","profileCode":"demo_document_v1","documentType":"real_pdf","title":"OCT","documentDate":"2026-08-17"}`), wantErr: true},
		{name: "blank title", payload: json.RawMessage(`{"recordMode":"demo_document","profileCode":"demo_document_v1","documentType":"demo_scan_summary","title":" ","documentDate":"2026-08-17"}`), wantErr: true},
		{name: "unknown field", payload: json.RawMessage(`{"recordMode":"demo_document","profileCode":"demo_document_v1","documentType":"demo_scan_summary","title":"OCT","documentDate":"2026-08-17","file":"x"}`), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Validate(context.Background(), documentDemoEventType, episodes.DraftIntentCreate, 1, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewDocumentDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := NewDocumentDemoDraftRegistry("production"); err == nil {
		t.Fatal("expected production guard")
	}
}
