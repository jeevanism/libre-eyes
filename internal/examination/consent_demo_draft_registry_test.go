package examination

import (
	"context"
	"errors"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

type fakeConsentCatalogue struct {
	active      map[string]bool
	development bool
}

func (f fakeConsentCatalogue) codeIsActive(_ context.Context, category, code string) (bool, error) {
	return f.active[category+":"+code], nil
}
func (f fakeConsentCatalogue) hasDevelopmentCodes(context.Context) (bool, error) {
	return f.development, nil
}

func TestConsentDemoDraftRegistryValidate(t *testing.T) {
	r := &ConsentDemoDraftRegistry{catalogue: fakeConsentCatalogue{active: map[string]bool{
		"form_type:development_form_type_1": true, "procedure:development_cataract_extraction": true,
		"laterality:development_right_eye": true, "anaesthetic:development_local_anaesthetic": true,
	}}}
	valid := []byte(`{"recordMode":"development_synthetic_consent","formTypeCode":"development_form_type_1","procedureCode":"development_cataract_extraction","laterality":"development_right_eye","anaestheticCode":"development_local_anaesthetic","comment":"demo"}`)
	if err := r.Validate(context.Background(), consentDemoEventType, episodes.DraftIntentCreate, 1, valid); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
	for _, payload := range [][]byte{
		[]byte(`{"recordMode":"development_synthetic_consent","formTypeCode":"development_form_type_1","procedureCode":"development_cataract_extraction","laterality":"development_right_eye","anaestheticCode":"development_local_anaesthetic","extra":true}`),
		[]byte(`{"recordMode":"development_synthetic_consent","formTypeCode":"production_form","procedureCode":"development_cataract_extraction","laterality":"development_right_eye","anaestheticCode":"development_local_anaesthetic"}`),
	} {
		if !errors.Is(r.Validate(context.Background(), consentDemoEventType, episodes.DraftIntentCreate, 1, payload), episodes.ErrInvalidRequest) {
			t.Fatal("invalid consent payload accepted")
		}
	}
}

func TestConsentDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := newConsentDemoDraftRegistry(context.Background(), fakeConsentCatalogue{development: true}, "production"); err == nil {
		t.Fatal("production accepted demo catalogue")
	}
}
