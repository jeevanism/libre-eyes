package examination

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const dnaExtractionDemoEventType = "ophthalmology.dna_extraction_demo"

type DNAExtractionDemoDraftRegistry struct{}

func NewDNAExtractionDemoDraftRegistry(environment string) (*DNAExtractionDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration DNA extraction is enabled")
	}
	return &DNAExtractionDemoDraftRegistry{}, nil
}

type dnaExtractionDemoPayload struct {
	RecordMode     string `json:"recordMode"`
	ProfileCode    string `json:"profileCode"`
	SampleLabel    string `json:"sampleLabel"`
	Status         string `json:"status"`
	StorageAddress string `json:"storageAddress"`
	ExtractionDate string `json:"extractionDate"`
	Volume         string `json:"volume"`
	Comment        string `json:"comment"`
}

func (DNAExtractionDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != dnaExtractionDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 16*1024 {
		return episodes.ErrInvalidRequest
	}
	var p dnaExtractionDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_dna_extraction" || p.ProfileCode != "demo_dna_extraction_v1" {
		return episodes.ErrInvalidRequest
	}
	if p.Status != "prepared" && p.Status != "extracted" && p.Status != "stored" || len(p.SampleLabel) < 1 || len(p.SampleLabel) > 80 || len(p.StorageAddress) < 1 || len(p.StorageAddress) > 80 || len(p.Volume) > 16 || len(p.Comment) > 500 {
		return episodes.ErrInvalidRequest
	}
	if p.ExtractionDate != "" {
		if _, err := time.Parse("2006-01-02", p.ExtractionDate); err != nil {
			return episodes.ErrInvalidRequest
		}
	}
	if strings.ContainsAny(p.SampleLabel+p.StorageAddress, "\r\n") {
		return episodes.ErrInvalidRequest
	}
	return nil
}
