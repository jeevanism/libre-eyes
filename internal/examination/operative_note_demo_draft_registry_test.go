package examination

import (
	"context"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

type fakeOperativeNoteCatalogue struct{ active map[string]bool }

func (f fakeOperativeNoteCatalogue) codeIsActive(_ context.Context, category, code string) (bool, error) {
	return f.active[category+":"+code], nil
}

func (f fakeOperativeNoteCatalogue) hasDevelopmentCodes(context.Context) (bool, error) {
	return false, nil
}

func TestOperativeNoteDemoDraftRegistryValidate(t *testing.T) {
	catalogue := fakeOperativeNoteCatalogue{active: map[string]bool{
		"procedure:development_cataract_extraction": true,
		"laterality:development_right_eye":          true,
		"surgeon:development_surgeon_a":             true,
		"anaesthetic:development_local_anaesthetic": true,
		"delivery:development_topical":              true,
	}}
	registry := &OperativeNoteDemoDraftRegistry{catalogue: catalogue}
	valid := []byte(`{"recordMode":"development_synthetic_operative_note","procedureCode":"development_cataract_extraction","laterality":"development_right_eye","surgeonCode":"development_surgeon_a","anaestheticCode":"development_local_anaesthetic","deliveryCodes":["development_topical"],"comment":"Synthetic demonstration"}`)
	if err := registry.Validate(context.Background(), operativeNoteDemoEventType, episodes.DraftIntentCreate, 1, valid); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	for name, payload := range map[string][]byte{
		"unknown field":          []byte(`{"recordMode":"development_synthetic_operative_note","procedureCode":"development_cataract_extraction","laterality":"development_right_eye","surgeonCode":"development_surgeon_a","anaestheticCode":"development_local_anaesthetic","deliveryCodes":["development_topical"],"unexpected":true}`),
		"missing local delivery": []byte(`{"recordMode":"development_synthetic_operative_note","procedureCode":"development_cataract_extraction","laterality":"development_right_eye","surgeonCode":"development_surgeon_a","anaestheticCode":"development_local_anaesthetic","deliveryCodes":[]}`),
		"delivery with GA":       []byte(`{"recordMode":"development_synthetic_operative_note","procedureCode":"development_cataract_extraction","laterality":"development_right_eye","surgeonCode":"development_surgeon_a","anaestheticCode":"development_general_anaesthetic","deliveryCodes":["development_topical"]}`),
	} {
		t.Run(name, func(t *testing.T) {
			if err := registry.Validate(context.Background(), operativeNoteDemoEventType, episodes.DraftIntentCreate, 1, payload); err != episodes.ErrInvalidRequest {
				t.Fatalf("Validate error = %v, want ErrInvalidRequest", err)
			}
		})
	}
}
