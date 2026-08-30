package examination

import (
	"context"
	"encoding/json"
	"errors"
)

import "github.com/jeevanism/libre-eyes/internal/episodes"

const (
	eyeDrawDemoEventType         = "ophthalmology.eyedraw_anterior_segment_demo"
	eyeDrawDemoSchemaVersion     = 1
	eyeDrawDemoRecordMode        = "development_eyedraw_anterior_segment"
	eyeDrawDemoCanvasCode        = "development_exam_ant_seg_v1"
	eyeDrawDemoMaximumBytes      = 16 * 1024
	eyeDrawDemoMaximumDoodles    = 32
	eyeDrawDemoMaximumProperties = 48
)

var allowedEyeDrawDemoClasses = map[string]struct{}{
	"AntSeg":        {},
	"PCIOL":         {},
	"PhakoIncision": {},
	"SidePort":      {},
}

// EyeDrawDemoDraftRegistry validates the approved, uncommitted EyeDraw demo
// payload. It deliberately recognizes only concrete EyeDraw runtime classes.
type EyeDrawDemoDraftRegistry struct{}

func NewEyeDrawDemoDraftRegistry(environment string) (*EyeDrawDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: development EyeDraw demo is enabled")
	}
	return &EyeDrawDemoDraftRegistry{}, nil
}

// Validate implements episodes.DraftPayloadRegistry.
func (EyeDrawDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != eyeDrawDemoEventType || intent != episodes.DraftIntentCreate || schemaVersion != eyeDrawDemoSchemaVersion {
		return episodes.ErrInvalidRequest
	}
	if _, err := parseEyeDrawDemoPayload(payload); err != nil {
		return episodes.ErrInvalidRequest
	}
	return nil
}

type eyeDrawDemoPayload struct {
	RecordMode string            `json:"recordMode"`
	CanvasCode string            `json:"canvasCode"`
	Laterality string            `json:"laterality"`
	Drawing    []json.RawMessage `json:"drawing"`
}

func parseEyeDrawDemoPayload(payload json.RawMessage) (eyeDrawDemoPayload, error) {
	var parsed eyeDrawDemoPayload
	if len(payload) == 0 || len(payload) > eyeDrawDemoMaximumBytes {
		return eyeDrawDemoPayload{}, errors.New("invalid EyeDraw development payload size")
	}
	if err := decodeStrictJSON(payload, &parsed); err != nil || parsed.RecordMode != eyeDrawDemoRecordMode || parsed.CanvasCode != eyeDrawDemoCanvasCode || (parsed.Laterality != "left" && parsed.Laterality != "right") || len(parsed.Drawing) > eyeDrawDemoMaximumDoodles {
		return eyeDrawDemoPayload{}, errors.New("invalid EyeDraw development payload")
	}
	for _, doodle := range parsed.Drawing {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(doodle, &object); err != nil || len(object) == 0 || len(object) > eyeDrawDemoMaximumProperties {
			return eyeDrawDemoPayload{}, errors.New("invalid EyeDraw doodle")
		}
		var className string
		if err := json.Unmarshal(object["className"], &className); err != nil {
			return eyeDrawDemoPayload{}, errors.New("invalid EyeDraw doodle class")
		}
		if _, allowed := allowedEyeDrawDemoClasses[className]; !allowed {
			return eyeDrawDemoPayload{}, errors.New("unapproved EyeDraw doodle class")
		}
		if rawSubclass, supplied := object["subclass"]; supplied {
			var subclass string
			if err := json.Unmarshal(rawSubclass, &subclass); err != nil || subclass != className {
				return eyeDrawDemoPayload{}, errors.New("EyeDraw runtime class does not match approved class")
			}
		}
	}
	return parsed, nil
}
