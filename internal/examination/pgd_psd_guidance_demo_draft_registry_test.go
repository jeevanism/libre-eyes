package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestPGDPSDGuidanceDemoDraftRegistryValidate(t *testing.T) {
	r := PGDPSDGuidanceDemoDraftRegistry{}
	body, _ := json.Marshal(map[string]string{"recordMode": "demo_pgd_psd_guidance", "profileCode": "demo_pgd_psd_guidance_v1", "pathway": "demo_pgd", "medicationLabel": "Demo medicine", "laterality": "right", "note": "planning"})
	if err := r.Validate(context.Background(), pgdPSDGuidanceDemoEventType, episodes.DraftIntentCreate, 1, body); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	bad := append([]byte(nil), body...)
	bad = []byte(`{"recordMode":"demo_pgd_psd_guidance","profileCode":"demo_pgd_psd_guidance_v1","pathway":"demo_pgd","medicationLabel":"","laterality":"right","note":""}`)
	if err := r.Validate(context.Background(), pgdPSDGuidanceDemoEventType, episodes.DraftIntentCreate, 1, bad); err == nil {
		t.Fatal("empty medication label accepted")
	}
}
