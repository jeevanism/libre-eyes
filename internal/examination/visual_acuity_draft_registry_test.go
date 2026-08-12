package examination

import (
	"context"
	"errors"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

type catalogueKey struct {
	unit, value, method string
}

type fakeCatalogue struct {
	active          map[catalogueKey]bool
	lookupErr       error
	developmentData bool
	developmentErr  error
}

func (c fakeCatalogue) readingIsActive(_ context.Context, unit, value, method string) (bool, error) {
	if c.lookupErr != nil {
		return false, c.lookupErr
	}
	return c.active[catalogueKey{unit, value, method}], nil
}

func (c fakeCatalogue) hasActiveDevelopmentCodes(context.Context) (bool, error) {
	return c.developmentData, c.developmentErr
}

func TestVisualAcuityDraftRegistryValidate(t *testing.T) {
	valid := []byte(`{"recordMode":"simple","eyes":[{"eye":"left","assessment":"recorded","readings":[{"unitCode":"development_distance_scale","valueCode":"development_value_010","methodCode":"development_unaided"}]},{"eye":"right","assessment":"unable_to_assess"}]}`)
	registry, err := newVisualAcuityDraftRegistry(context.Background(), fakeCatalogue{active: map[catalogueKey]bool{
		{"development_distance_scale", "development_value_010", "development_unaided"}: true,
	}}, "development")
	if err != nil {
		t.Fatalf("newVisualAcuityDraftRegistry() error = %v", err)
	}
	if err := registry.Validate(context.Background(), visualAcuityEventType, episodes.DraftIntentCreate, 1, valid); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}

	for name, payload := range map[string][]byte{
		"unknown root property":      []byte(`{"recordMode":"simple","eyes":[],"extra":true}`),
		"duplicate eye":              []byte(`{"recordMode":"simple","eyes":[{"eye":"left","assessment":"unable_to_assess"},{"eye":"left","assessment":"eye_missing"}]}`),
		"recorded without readings":  []byte(`{"recordMode":"simple","eyes":[{"eye":"left","assessment":"recorded"}]}`),
		"non-recorded with readings": []byte(`{"recordMode":"simple","eyes":[{"eye":"left","assessment":"eye_missing","readings":[]}]}`),
		"duplicate method":           []byte(`{"recordMode":"simple","eyes":[{"eye":"left","assessment":"recorded","readings":[{"unitCode":"development_distance_scale","valueCode":"development_value_010","methodCode":"development_unaided"},{"unitCode":"development_distance_scale","valueCode":"development_value_010","methodCode":"development_unaided"}]}]}`),
		"inactive code":              []byte(`{"recordMode":"simple","eyes":[{"eye":"left","assessment":"recorded","readings":[{"unitCode":"development_distance_scale","valueCode":"development_value_020","methodCode":"development_unaided"}]}]}`),
	} {
		t.Run(name, func(t *testing.T) {
			if err := registry.Validate(context.Background(), visualAcuityEventType, episodes.DraftIntentCreate, 1, payload); !errors.Is(err, episodes.ErrInvalidRequest) {
				t.Fatalf("Validate() error = %v, want invalid request", err)
			}
		})
	}
	if err := registry.Validate(context.Background(), "other.type", episodes.DraftIntentCreate, 1, valid); !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate(other type) error = %v, want invalid request", err)
	}
	if err := registry.Validate(context.Background(), visualAcuityEventType, episodes.DraftIntentCreate, 2, valid); !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate(other schema) error = %v, want invalid request", err)
	}
	if err := registry.Validate(context.Background(), visualAcuityEventType, episodes.DraftIntentUpdate, 1, valid); !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate(update intent) error = %v, want invalid request", err)
	}
}

func TestVisualAcuityDraftRegistryCatalogueFailureIsNotInvalidRequest(t *testing.T) {
	registry, err := newVisualAcuityDraftRegistry(context.Background(), fakeCatalogue{lookupErr: errors.New("database unavailable")}, "development")
	if err != nil {
		t.Fatalf("newVisualAcuityDraftRegistry() error = %v", err)
	}
	err = registry.Validate(context.Background(), visualAcuityEventType, episodes.DraftIntentCreate, 1, []byte(`{"recordMode":"simple","eyes":[{"eye":"left","assessment":"recorded","readings":[{"unitCode":"development_distance_scale","valueCode":"development_value_010","methodCode":"development_unaided"}]}]}`))
	if err == nil || errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate(catalogue failure) error = %v, want unavailable error", err)
	}
}

func TestVisualAcuityDraftRegistryProductionGuard(t *testing.T) {
	if _, err := newVisualAcuityDraftRegistry(context.Background(), fakeCatalogue{developmentData: true}, "production"); err == nil {
		t.Fatal("production registry accepted active development catalogue")
	}
	if _, err := newVisualAcuityDraftRegistry(context.Background(), fakeCatalogue{developmentErr: errors.New("query failed")}, "production"); err == nil {
		t.Fatal("production registry accepted an unreadable catalogue")
	}
	if _, err := newVisualAcuityDraftRegistry(context.Background(), fakeCatalogue{}, "production"); err != nil {
		t.Fatalf("production registry without development catalogue error = %v", err)
	}
}
