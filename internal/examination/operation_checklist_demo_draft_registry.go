package examination

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const operationChecklistDemoEventType = "ophthalmology.operation_checklist_demo"

type OperationChecklistDemoDraftRegistry struct{}

func NewOperationChecklistDemoDraftRegistry(environment string) (*OperationChecklistDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration operation checklist is enabled")
	}
	return &OperationChecklistDemoDraftRegistry{}, nil
}

type operationChecklistDemoQuestion struct {
	Code   string `json:"code"`
	Answer string `json:"answer"`
}
type operationChecklistDemoPayload struct {
	RecordMode  string                           `json:"recordMode"`
	ProfileCode string                           `json:"profileCode"`
	EyeMode     string                           `json:"eyeMode"`
	Questions   []operationChecklistDemoQuestion `json:"questions"`
	Note        string                           `json:"note"`
}

func (OperationChecklistDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != operationChecklistDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 {
		return episodes.ErrInvalidRequest
	}
	var p operationChecklistDemoPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_operation_checklist" || p.ProfileCode != "demo_operation_checklist_v1" || (p.EyeMode != "right" && p.EyeMode != "left" && p.EyeMode != "both") || len(p.Questions) != 3 || len(p.Note) > 1000 || strings.TrimSpace(p.Note) != p.Note {
		return episodes.ErrInvalidRequest
	}
	seen := map[string]bool{}
	allowed := map[string]bool{"identity_check": true, "procedure_confirmed": true, "escort_discussed": true}
	for _, q := range p.Questions {
		if !allowed[q.Code] || seen[q.Code] || (q.Answer != "demo_yes" && q.Answer != "demo_no" && q.Answer != "demo_not_recorded") {
			return episodes.ErrInvalidRequest
		}
		seen[q.Code] = true
	}
	return nil
}
