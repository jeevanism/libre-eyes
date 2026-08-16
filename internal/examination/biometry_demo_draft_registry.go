package examination

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jeevanism/visionopus/internal/episodes"
	"math"
	"strconv"
	"strings"
	"time"
)

const biometryDemoEventType = "ophthalmology.biometry_demo"

type BiometryDemoDraftRegistry struct{}

func NewBiometryDemoDraftRegistry(environment string) (*BiometryDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration biometry is enabled")
	}
	return &BiometryDemoDraftRegistry{}, nil
}

type biometryDemoPayload struct {
	RecordMode      string           `json:"recordMode"`
	ProfileCode     string           `json:"profileCode"`
	DeviceCode      string           `json:"deviceCode"`
	LensCode        string           `json:"lensCode"`
	MeasurementDate string           `json:"measurementDate"`
	RightEye        *biometryDemoEye `json:"rightEye"`
	LeftEye         *biometryDemoEye `json:"leftEye"`
	Comment         string           `json:"comment"`
}
type biometryDemoEye struct {
	AxialLength string `json:"axialLength"`
	R1          string `json:"r1"`
	R2          string `json:"r2"`
	R1Axis      int    `json:"r1Axis"`
	R2Axis      int    `json:"r2Axis"`
	ACD         string `json:"acd"`
	WTW         string `json:"wtw"`
}

func (BiometryDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != biometryDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 {
		return episodes.ErrInvalidRequest
	}
	var p biometryDemoPayload
	if len(payload) == 0 || len(payload) > 16*1024 || decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_biometry" || p.ProfileCode != "demo_biometry_v1" || (p.DeviceCode != "demo_manual" && p.DeviceCode != "demo_iolmaster") || !validBiometryLens(p.LensCode) || len(p.Comment) > 2000 || strings.TrimSpace(p.Comment) != p.Comment {
		return episodes.ErrInvalidRequest
	}
	date, err := time.Parse("2006-01-02", p.MeasurementDate)
	if err != nil || date.After(time.Now().UTC().Truncate(24*time.Hour)) || (p.RightEye == nil && p.LeftEye == nil) {
		return episodes.ErrInvalidRequest
	}
	if p.RightEye != nil && !validBiometryEye(*p.RightEye) || p.LeftEye != nil && !validBiometryEye(*p.LeftEye) {
		return episodes.ErrInvalidRequest
	}
	return nil
}
func validBiometryLens(v string) bool {
	switch v {
	case "demo_none", "demo_ma60ac", "demo_sn60wf", "demo_sa60at", "demo_mta3uo":
		return true
	}
	return false
}
func validBiometryEye(e biometryDemoEye) bool {
	return validBiometryDecimal(e.AxialLength, 10, 40) && validBiometryDecimal(e.R1, 1, 20) && validBiometryDecimal(e.R2, 1, 20) && validBiometryDecimal(e.ACD, 0, 10) && validBiometryDecimal(e.WTW, 5, 20) && e.R1Axis >= 0 && e.R1Axis <= 180 && e.R2Axis >= 0 && e.R2Axis <= 180
}
func validBiometryDecimal(v string, min, max float64) bool {
	if v == "" || len(v) > 12 || strings.Count(v, ".") > 1 {
		return false
	}
	if dot := strings.IndexByte(v, '.'); dot >= 0 && len(v)-dot-1 > 2 {
		return false
	}
	n, err := strconv.ParseFloat(v, 64)
	return err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) && n >= min && n <= max
}
