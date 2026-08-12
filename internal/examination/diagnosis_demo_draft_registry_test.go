package examination

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jeevanism/visionopus/internal/episodes"
)

type diagnosisDemoCatalogueKey struct{ profile, selection string }

type fakeDiagnosisDemoCatalogue struct {
	active          map[diagnosisDemoCatalogueKey]bool
	lookupErr       error
	developmentData bool
	developmentErr  error
}

func (c fakeDiagnosisDemoCatalogue) selectionIsActive(_ context.Context, profile, selection string) (bool, error) {
	if c.lookupErr != nil {
		return false, c.lookupErr
	}
	return c.active[diagnosisDemoCatalogueKey{profile, selection}], nil
}

func (c fakeDiagnosisDemoCatalogue) hasDevelopmentCodes(context.Context) (bool, error) {
	return c.developmentData, c.developmentErr
}

func TestDiagnosisDemoDraftRegistryValidate(t *testing.T) {
	registry, err := newDiagnosisDemoDraftRegistry(context.Background(), fakeDiagnosisDemoCatalogue{active: map[diagnosisDemoCatalogueKey]bool{{diagnosisDemoProfileCode, "development_cataract"}: true}}, "development")
	if err != nil {
		t.Fatalf("newDiagnosisDemoDraftRegistry() error = %v", err)
	}
	registry.now = func() time.Time { return time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC) }
	valid := []byte(`{"recordMode":"development_synthetic_diagnosis","profileCode":"development_ophthalmology_diagnosis_v1","selectionCode":"development_cataract","laterality":"left","diagnosisDate":"2026-08-09"}`)
	if err := registry.Validate(context.Background(), diagnosisDemoEventType, episodes.DraftIntentCreate, 1, valid); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
	for name, payload := range map[string][]byte{
		"unknown field":      []byte(`{"recordMode":"development_synthetic_diagnosis","profileCode":"development_ophthalmology_diagnosis_v1","selectionCode":"development_cataract","laterality":"left","diagnosisDate":"2026-08-09","term":"cataract"}`),
		"future date":        []byte(`{"recordMode":"development_synthetic_diagnosis","profileCode":"development_ophthalmology_diagnosis_v1","selectionCode":"development_cataract","laterality":"left","diagnosisDate":"2026-08-10"}`),
		"wrong laterality":   []byte(`{"recordMode":"development_synthetic_diagnosis","profileCode":"development_ophthalmology_diagnosis_v1","selectionCode":"development_cataract","laterality":"unknown","diagnosisDate":"2026-08-09"}`),
		"inactive selection": []byte(`{"recordMode":"development_synthetic_diagnosis","profileCode":"development_ophthalmology_diagnosis_v1","selectionCode":"development_glaucoma","laterality":"left","diagnosisDate":"2026-08-09"}`),
	} {
		t.Run(name, func(t *testing.T) {
			if err := registry.Validate(context.Background(), diagnosisDemoEventType, episodes.DraftIntentCreate, 1, payload); !errors.Is(err, episodes.ErrInvalidRequest) {
				t.Fatalf("Validate() error = %v, want invalid request", err)
			}
		})
	}
}

func TestDiagnosisDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := newDiagnosisDemoDraftRegistry(context.Background(), fakeDiagnosisDemoCatalogue{developmentData: true}, "production"); err == nil {
		t.Fatal("production registry accepted development catalogue")
	}
	if _, err := newDiagnosisDemoDraftRegistry(context.Background(), fakeDiagnosisDemoCatalogue{}, "production"); err != nil {
		t.Fatalf("production registry without development catalogue error = %v", err)
	}
}
