package examination

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const intravitrealInjectionDemoEventType = "ophthalmology.intravitreal_injection_demo"

type IntravitrealInjectionDemoDraftRegistry struct{}

func NewIntravitrealInjectionDemoDraftRegistry(environment string) (*IntravitrealInjectionDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration intravitreal injection is enabled")
	}
	return &IntravitrealInjectionDemoDraftRegistry{}, nil
}

type intravitrealInjectionDemoPayload struct {
	RecordMode  string                        `json:"recordMode"`
	ProfileCode string                        `json:"profileCode"`
	EyeMode     string                        `json:"eyeMode"`
	RightEye    *intravitrealInjectionDemoEye `json:"rightEye"`
	LeftEye     *intravitrealInjectionDemoEye `json:"leftEye"`
	Note        string                        `json:"note"`
}
type intravitrealInjectionDemoEye struct {
	DrugCode        string `json:"drugCode"`
	SiteCode        string `json:"siteCode"`
	AnaestheticCode string `json:"anaestheticCode"`
	InjectionNumber int    `json:"injectionNumber"`
	PlannedDate     string `json:"plannedDate"`
	PostCheck       string `json:"postCheck"`
}

func (IntravitrealInjectionDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != intravitrealInjectionDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 {
		return episodes.ErrInvalidRequest
	}
	var p intravitrealInjectionDemoPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_intravitreal_injection" || p.ProfileCode != "demo_intravitreal_injection_v1" || len(p.Note) > 1000 || strings.TrimSpace(p.Note) != p.Note {
		return episodes.ErrInvalidRequest
	}
	if (p.EyeMode != "right" && p.EyeMode != "left" && p.EyeMode != "both") || (p.RightEye == nil && p.LeftEye == nil) || (p.EyeMode == "right" && p.RightEye == nil) || (p.EyeMode == "left" && p.LeftEye == nil) || (p.EyeMode == "both" && (p.RightEye == nil || p.LeftEye == nil)) {
		return episodes.ErrInvalidRequest
	}
	if p.RightEye != nil && !validIntravitrealEye(*p.RightEye) || p.LeftEye != nil && !validIntravitrealEye(*p.LeftEye) {
		return episodes.ErrInvalidRequest
	}
	return nil
}
func validIntravitrealEye(e intravitrealInjectionDemoEye) bool {
	return demoCode(e.DrugCode, "demo_drug_") && demoCode(e.SiteCode, "demo_site_") && demoCode(e.AnaestheticCode, "demo_anaesthetic_") && e.InjectionNumber >= 1 && e.InjectionNumber <= 10 && validDemoDate(e.PlannedDate) && (e.PostCheck == "demo_not_recorded" || e.PostCheck == "demo_clear" || e.PostCheck == "demo_review")
}
func demoCode(value, prefix string) bool {
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix) && len(value) <= 64 && strings.Trim(value, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_") == ""
}
func validDemoDate(value string) bool {
	d, err := time.Parse("2006-01-02", value)
	return err == nil && !d.After(time.Now().UTC().Truncate(24*time.Hour))
}
