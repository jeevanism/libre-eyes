package examination

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const iopPhasingDemoEventType = "ophthalmology.iop_phasing_demo"

var phasingTimePattern = regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9]$`)

type IOPPhasingDemoDraftRegistry struct{}

func NewIOPPhasingDemoDraftRegistry(environment string) (*IOPPhasingDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration IOP phasing is enabled")
	}
	return &IOPPhasingDemoDraftRegistry{}, nil
}

type iopPhasingDemoEye struct {
	InstrumentCode string `json:"instrumentCode"`
	Dilated        bool   `json:"dilated"`
	Comment        string `json:"comment"`
	Readings       []struct {
		Value           int    `json:"value"`
		MeasurementTime string `json:"measurementTime"`
	} `json:"readings"`
}
type iopPhasingDemoPayload struct {
	RecordMode  string             `json:"recordMode"`
	ProfileCode string             `json:"profileCode"`
	EyeMode     string             `json:"eyeMode"`
	RightEye    *iopPhasingDemoEye `json:"rightEye"`
	LeftEye     *iopPhasingDemoEye `json:"leftEye"`
}

func (IOPPhasingDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != iopPhasingDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 24*1024 {
		return episodes.ErrInvalidRequest
	}
	var p iopPhasingDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_iop_phasing" || p.ProfileCode != "demo_iop_phasing_v1" || (p.EyeMode != "right" && p.EyeMode != "left" && p.EyeMode != "both") {
		return episodes.ErrInvalidRequest
	}
	if (p.EyeMode == "right" || p.EyeMode == "both") && !validPhasingEye(p.RightEye) {
		return episodes.ErrInvalidRequest
	}
	if (p.EyeMode == "left" || p.EyeMode == "both") && !validPhasingEye(p.LeftEye) {
		return episodes.ErrInvalidRequest
	}
	if p.EyeMode == "right" && p.LeftEye != nil || p.EyeMode == "left" && p.RightEye != nil {
		return episodes.ErrInvalidRequest
	}
	return nil
}

func validPhasingEye(eye *iopPhasingDemoEye) bool {
	if eye == nil || !allowedPhasingInstrument(eye.InstrumentCode) || len(eye.Readings) < 1 || len(eye.Readings) > 12 || len(eye.Comment) > 500 {
		return false
	}
	for _, reading := range eye.Readings {
		if reading.Value < 0 || reading.Value > 100 || !phasingTimePattern.MatchString(reading.MeasurementTime) {
			return false
		}
		if _, err := time.Parse("15:04", reading.MeasurementTime); err != nil {
			return false
		}
	}
	return true
}

func allowedPhasingInstrument(value string) bool {
	return value == "demo_goldmann" || value == "demo_tono_pen" || value == "demo_i_care" || value == "demo_perkins" || value == "demo_other"
}
