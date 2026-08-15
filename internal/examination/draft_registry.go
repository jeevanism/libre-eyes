package examination

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/episodes"
)

// DraftRegistry dispatches approved examination draft schemas by event type.
// The generic episode service remains responsible for authorization and storage.
type DraftRegistry struct {
	visualAcuity      *VisualAcuityDraftRegistry
	iop               *IOPDraftRegistry
	diagnosisDemo     *DiagnosisDemoDraftRegistry
	eyeDrawDemo       *EyeDrawDemoDraftRegistry
	operativeNoteDemo *OperativeNoteDemoDraftRegistry
	prescriptionDemo  *PrescriptionDemoDraftRegistry
	consentDemo       *ConsentDemoDraftRegistry
}

func NewDraftRegistry(ctx context.Context, pool *pgxpool.Pool, environment string) (*DraftRegistry, error) {
	visualAcuity, err := NewVisualAcuityDraftRegistry(ctx, pool, environment)
	if err != nil {
		return nil, err
	}
	iop, err := NewIOPDraftRegistry(ctx, pool, environment)
	if err != nil {
		return nil, err
	}
	diagnosisDemo, err := NewDiagnosisDemoDraftRegistry(ctx, pool, environment)
	if err != nil {
		return nil, err
	}
	eyeDrawDemo, err := NewEyeDrawDemoDraftRegistry(environment)
	if err != nil {
		return nil, err
	}
	operativeNoteDemo, err := NewOperativeNoteDemoDraftRegistry(ctx, pool, environment)
	if err != nil {
		return nil, err
	}
	prescriptionDemo, err := NewPrescriptionDemoDraftRegistry(ctx, pool, environment)
	if err != nil {
		return nil, err
	}
	consentDemo, err := NewConsentDemoDraftRegistry(ctx, pool, environment)
	if err != nil {
		return nil, err
	}
	return &DraftRegistry{visualAcuity: visualAcuity, iop: iop, diagnosisDemo: diagnosisDemo, eyeDrawDemo: eyeDrawDemo, operativeNoteDemo: operativeNoteDemo, prescriptionDemo: prescriptionDemo, consentDemo: consentDemo}, nil
}

func (r *DraftRegistry) Validate(ctx context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if r == nil {
		return errors.New("examination draft registry is required")
	}
	switch eventTypeCode {
	case visualAcuityEventType:
		return r.visualAcuity.Validate(ctx, eventTypeCode, intent, schemaVersion, payload)
	case iopEventType:
		return r.iop.Validate(ctx, eventTypeCode, intent, schemaVersion, payload)
	case diagnosisDemoEventType:
		return r.diagnosisDemo.Validate(ctx, eventTypeCode, intent, schemaVersion, payload)
	case eyeDrawDemoEventType:
		return r.eyeDrawDemo.Validate(ctx, eventTypeCode, intent, schemaVersion, payload)
	case operativeNoteDemoEventType:
		return r.operativeNoteDemo.Validate(ctx, eventTypeCode, intent, schemaVersion, payload)
	case prescriptionDemoEventType:
		return r.prescriptionDemo.Validate(ctx, eventTypeCode, intent, schemaVersion, payload)
	case consentDemoEventType:
		return r.consentDemo.Validate(ctx, eventTypeCode, intent, schemaVersion, payload)
	default:
		return episodes.ErrInvalidRequest
	}
}
