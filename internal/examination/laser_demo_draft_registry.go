package examination

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const laserDemoEventType = "ophthalmology.laser_demo"

type LaserDemoDraftRegistry struct{}

func NewLaserDemoDraftRegistry(environment string) (*LaserDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration laser is enabled")
	}
	return &LaserDemoDraftRegistry{}, nil
}

type laserDemoPayload struct {
	RecordMode    string        `json:"recordMode"`
	ProfileCode   string        `json:"profileCode"`
	EyeMode       string        `json:"eyeMode"`
	RightEye      *laserDemoEye `json:"rightEye"`
	LeftEye       *laserDemoEye `json:"leftEye"`
	SiteCode      string        `json:"siteCode"`
	LaserCode     string        `json:"laserCode"`
	OperatorCode  string        `json:"operatorCode"`
	TreatmentDate string        `json:"treatmentDate"`
	Comment       string        `json:"comment"`
}
type laserDemoEye struct {
	ProcedureCode string `json:"procedureCode"`
}

func (LaserDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != laserDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 {
		return episodes.ErrInvalidRequest
	}
	var p laserDemoPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_laser" || p.ProfileCode != "demo_laser_v1" || len(p.Comment) > 1000 || strings.TrimSpace(p.Comment) != p.Comment || !laserEyeMode(p.EyeMode) || (p.RightEye == nil && p.LeftEye == nil) || !demoCode(p.SiteCode, "demo_site_") || !demoCode(p.LaserCode, "demo_laser_") || !demoCode(p.OperatorCode, "demo_operator_") || !validLaserDate(p.TreatmentDate) {
		return episodes.ErrInvalidRequest
	}
	if p.EyeMode == "right" && p.RightEye == nil || p.EyeMode == "left" && p.LeftEye == nil || p.EyeMode == "both" && (p.RightEye == nil || p.LeftEye == nil) {
		return episodes.ErrInvalidRequest
	}
	if p.RightEye != nil && !demoCode(p.RightEye.ProcedureCode, "demo_procedure_") || p.LeftEye != nil && !demoCode(p.LeftEye.ProcedureCode, "demo_procedure_") {
		return episodes.ErrInvalidRequest
	}
	return nil
}
func laserEyeMode(v string) bool { return v == "right" || v == "left" || v == "both" }
func validLaserDate(v string) bool {
	d, err := time.Parse("2006-01-02", v)
	return err == nil && !d.After(time.Now().UTC().Truncate(24*time.Hour))
}
