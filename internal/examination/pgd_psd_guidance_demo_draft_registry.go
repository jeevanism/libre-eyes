package examination

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const pgdPSDGuidanceDemoEventType = "ophthalmology.pgd_psd_guidance_demo"

type PGDPSDGuidanceDemoDraftRegistry struct{}

func NewPGDPSDGuidanceDemoDraftRegistry(environment string) (*PGDPSDGuidanceDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration PGD/PSD guidance is enabled")
	}
	return &PGDPSDGuidanceDemoDraftRegistry{}, nil
}

type pgdPSDGuidanceDemoPayload struct {
	RecordMode      string `json:"recordMode"`
	ProfileCode     string `json:"profileCode"`
	Pathway         string `json:"pathway"`
	MedicationLabel string `json:"medicationLabel"`
	Laterality      string `json:"laterality"`
	Note            string `json:"note"`
}

func (PGDPSDGuidanceDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != pgdPSDGuidanceDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 16*1024 {
		return episodes.ErrInvalidRequest
	}
	var p pgdPSDGuidanceDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_pgd_psd_guidance" || p.ProfileCode != "demo_pgd_psd_guidance_v1" || len(p.MedicationLabel) == 0 || len(p.MedicationLabel) > 120 || len(p.Note) > 500 {
		return episodes.ErrInvalidRequest
	}
	if p.Pathway != "demo_pgd" && p.Pathway != "demo_psd" {
		return episodes.ErrInvalidRequest
	}
	if p.Laterality != "right" && p.Laterality != "left" && p.Laterality != "bilateral" && p.Laterality != "not_applicable" {
		return episodes.ErrInvalidRequest
	}
	return nil
}
