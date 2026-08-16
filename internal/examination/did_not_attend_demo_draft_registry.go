package examination

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const didNotAttendDemoEventType = "ophthalmology.did_not_attend_demo"

type DidNotAttendDemoDraftRegistry struct{}

func NewDidNotAttendDemoDraftRegistry(environment string) (*DidNotAttendDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration did-not-attend is enabled")
	}
	return &DidNotAttendDemoDraftRegistry{}, nil
}

type didNotAttendDemoPayload struct {
	RecordMode  string `json:"recordMode"`
	ProfileCode string `json:"profileCode"`
	EventDate   string `json:"eventDate"`
	Source      string `json:"source"`
	Comment     string `json:"comment"`
}

func (DidNotAttendDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != didNotAttendDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 {
		return episodes.ErrInvalidRequest
	}
	var p didNotAttendDemoPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_did_not_attend" || p.ProfileCode != "demo_did_not_attend_v1" || p.Source != "demo_clinic_flow" || len(p.Comment) > 255 || strings.TrimSpace(p.Comment) != p.Comment || !validDidNotAttendDate(p.EventDate) {
		return episodes.ErrInvalidRequest
	}
	return nil
}

func validDidNotAttendDate(value string) bool {
	date, err := time.Parse("2006-01-02", value)
	return err == nil && !date.After(time.Now().UTC().Truncate(24*time.Hour))
}
