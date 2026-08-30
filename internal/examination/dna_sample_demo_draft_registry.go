package examination

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const dnaSampleDemoEventType = "ophthalmology.dna_sample_demo"

type DNASampleDemoDraftRegistry struct{}

func NewDNASampleDemoDraftRegistry(environment string) (*DNASampleDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration DNA sample is enabled")
	}
	return &DNASampleDemoDraftRegistry{}, nil
}

type dnaSampleDemoPayload struct {
	RecordMode  string `json:"recordMode"`
	ProfileCode string `json:"profileCode"`
	SampleType  string `json:"sampleType"`
	ConsentedBy string `json:"consentedBy"`
	SampleDate  string `json:"sampleDate"`
	Volume      int    `json:"volume"`
	Comment     string `json:"comment"`
}

func (DNASampleDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != dnaSampleDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 16*1024 {
		return episodes.ErrInvalidRequest
	}
	var p dnaSampleDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_dna_sample" || p.ProfileCode != "demo_dna_sample_v1" || p.Volume < 1 || p.Volume > 99 || len(p.Comment) > 500 {
		return episodes.ErrInvalidRequest
	}
	if p.SampleType != "demo_blood" && p.SampleType != "demo_saliva" && p.SampleType != "demo_other" {
		return episodes.ErrInvalidRequest
	}
	if p.ConsentedBy != "demo_clinician" && p.ConsentedBy != "demo_patient" && p.ConsentedBy != "demo_guardian" {
		return episodes.ErrInvalidRequest
	}
	if _, err := time.Parse("2006-01-02", p.SampleDate); err != nil {
		return episodes.ErrInvalidRequest
	}
	return nil
}
