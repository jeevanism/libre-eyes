package examination

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const visualFieldsDemoEventType = "ophthalmology.visual_fields_demo"

// VisualFieldsDemoDraftRegistry validates a synthetic, uncommitted field-test draft.
type VisualFieldsDemoDraftRegistry struct{}

func NewVisualFieldsDemoDraftRegistry(environment string) (*VisualFieldsDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration visual fields is enabled")
	}
	return &VisualFieldsDemoDraftRegistry{}, nil
}

func (VisualFieldsDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != visualFieldsDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 {
		return episodes.ErrInvalidRequest
	}
	var p struct {
		RecordMode string `json:"recordMode"`
		Strategy   string `json:"strategyCode"`
		Pattern    string `json:"patternCode"`
		Eyes       []struct {
			Eye    string `json:"eye"`
			Result string `json:"resultCode"`
		} `json:"eyes"`
		Comment string `json:"comment"`
	}
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_visual_fields" || p.Strategy != "demo_sita_standard" || p.Pattern != "demo_24_2" || len(p.Comment) > 2000 || strings.TrimSpace(p.Comment) != p.Comment {
		return episodes.ErrInvalidRequest
	}
	if len(p.Eyes) != 1 && len(p.Eyes) != 2 {
		return episodes.ErrInvalidRequest
	}
	seen := map[string]bool{}
	for _, eye := range p.Eyes {
		if seen[eye.Eye] || (eye.Eye != "left" && eye.Eye != "right") || (eye.Result != "demo_normal" && eye.Result != "demo_generalised_reduction" && eye.Result != "demo_field_defect") {
			return episodes.ErrInvalidRequest
		}
		seen[eye.Eye] = true
	}
	return nil
}
