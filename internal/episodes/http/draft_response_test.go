package episodeshttp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestMapDraftDiagnosisDemoMatchesOverlayResponse(t *testing.T) {
	response := mapDraft(episodes.EventDraft{
		ID: "11111111-1111-4111-8111-111111111111", EpisodeID: "22222222-2222-4222-8222-222222222222",
		EventTypeCode: "ophthalmology.principal_diagnosis_demo", Intent: episodes.DraftIntentCreate,
		Mode: episodes.DraftModeManual, SchemaVersion: 1,
		Payload: json.RawMessage(`{"recordMode":"development_synthetic_diagnosis"}`),
		Version: 1, ExpiresAt: time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC),
		NewerCommittedEdits: true,
	})
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal draft response: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal draft response: %v", err)
	}
	for _, field := range []string{"targetEventId", "newerCommittedEdits"} {
		if _, present := fields[field]; present {
			t.Fatalf("diagnosis demo response includes %q", field)
		}
	}
}
