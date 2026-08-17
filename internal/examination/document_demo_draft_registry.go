package examination

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const documentDemoEventType = "ophthalmology.document_demo"

type DocumentDemoDraftRegistry struct{}

func NewDocumentDemoDraftRegistry(environment string) (*DocumentDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration document is enabled")
	}
	return &DocumentDemoDraftRegistry{}, nil
}

type documentDemoPayload struct {
	RecordMode   string `json:"recordMode"`
	ProfileCode  string `json:"profileCode"`
	DocumentType string `json:"documentType"`
	Title        string `json:"title"`
	Laterality   string `json:"laterality"`
	DocumentDate string `json:"documentDate"`
	Comment      string `json:"comment"`
}

func (DocumentDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != documentDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 {
		return episodes.ErrInvalidRequest
	}
	var p documentDemoPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &p) != nil {
		return episodes.ErrInvalidRequest
	}
	if p.RecordMode != "demo_document" || p.ProfileCode != "demo_document_v1" ||
		!allowedDocumentType(p.DocumentType) || !validDocumentDate(p.DocumentDate) ||
		p.Title == "" || len(p.Title) > 160 || strings.TrimSpace(p.Title) != p.Title ||
		!allowedDocumentLaterality(p.Laterality) || len(p.Comment) > 1000 || strings.TrimSpace(p.Comment) != p.Comment {
		return episodes.ErrInvalidRequest
	}
	return nil
}

func allowedDocumentType(value string) bool {
	return value == "demo_clinic_letter" || value == "demo_scan_summary" || value == "demo_external_document"
}

func allowedDocumentLaterality(value string) bool {
	return value == "" || value == "not_applicable" || value == "right" || value == "left" || value == "bilateral"
}

func validDocumentDate(value string) bool {
	date, err := time.Parse("2006-01-02", value)
	return err == nil && !date.After(time.Now().UTC().Truncate(24*time.Hour))
}
