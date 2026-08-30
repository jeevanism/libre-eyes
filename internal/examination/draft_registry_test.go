package examination

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestDraftRegistryRejectsUnregisteredEventType(t *testing.T) {
	registry := &DraftRegistry{}
	err := registry.Validate(context.Background(), "unregistered.event", episodes.DraftIntentCreate, 1, []byte(`{}`))
	if !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate() error = %v, want invalid request", err)
	}
}

func TestDraftRegistryDispatchesIOPValidation(t *testing.T) {
	registry := &DraftRegistry{
		visualAcuity: &VisualAcuityDraftRegistry{catalogue: fakeCatalogue{}},
		iop: &IOPDraftRegistry{catalogue: fakeIOPCatalogue{active: map[iopCatalogueKey]bool{
			{iopProfileCode, "development_iop_14"}: true,
		}}},
		diagnosisDemo: &DiagnosisDemoDraftRegistry{catalogue: fakeDiagnosisDemoCatalogue{active: map[diagnosisDemoCatalogueKey]bool{
			{diagnosisDemoProfileCode, "development_cataract"}: true,
		}}, now: func() time.Time { return time.Date(2026, time.August, 9, 12, 0, 0, 0, time.Local) }},
	}
	valid := []byte(`{"recordMode":"development_raw_mmhg","profileCode":"development_iop_manual_mmhg","eyes":[{"eye":"right","valueCode":"development_iop_14"}]}`)
	if err := registry.Validate(context.Background(), iopEventType, episodes.DraftIntentCreate, 1, valid); err != nil {
		t.Fatalf("Validate(IOP) error = %v", err)
	}
	invalid := []byte(`{"recordMode":"development_raw_mmhg","profileCode":"development_iop_manual_mmhg","eyes":[{"eye":"right","valueCode":"development_iop_99"}]}`)
	if err := registry.Validate(context.Background(), iopEventType, episodes.DraftIntentCreate, 1, invalid); !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate(inactive IOP) error = %v, want invalid request", err)
	}
}

func TestDraftRegistryDispatchesDiagnosisDemoValidation(t *testing.T) {
	registry := &DraftRegistry{diagnosisDemo: &DiagnosisDemoDraftRegistry{catalogue: fakeDiagnosisDemoCatalogue{active: map[diagnosisDemoCatalogueKey]bool{{diagnosisDemoProfileCode, "development_cataract"}: true}}, now: func() time.Time { return time.Date(2026, time.August, 9, 12, 0, 0, 0, time.Local) }}}
	payload := []byte(`{"recordMode":"development_synthetic_diagnosis","profileCode":"development_ophthalmology_diagnosis_v1","selectionCode":"development_cataract","laterality":"left","diagnosisDate":"2026-08-09"}`)
	if err := registry.Validate(context.Background(), diagnosisDemoEventType, episodes.DraftIntentCreate, 1, payload); err != nil {
		t.Fatalf("Validate(diagnosis demo) error = %v", err)
	}
}

func TestDraftRegistryDispatchesEyeDrawDemoValidation(t *testing.T) {
	registry := &DraftRegistry{eyeDrawDemo: &EyeDrawDemoDraftRegistry{}}
	payload := []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"development_exam_ant_seg_v1","laterality":"left","drawing":[{"className":"AntSeg"}]}`)
	if err := registry.Validate(context.Background(), eyeDrawDemoEventType, episodes.DraftIntentCreate, 1, payload); err != nil {
		t.Fatalf("Validate(EyeDraw demo) error = %v", err)
	}
}
