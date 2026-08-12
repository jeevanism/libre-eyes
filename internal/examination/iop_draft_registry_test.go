package examination

import (
	"context"
	"errors"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

type iopCatalogueKey struct {
	profile, value string
}

type fakeIOPCatalogue struct {
	active          map[iopCatalogueKey]bool
	lookupErr       error
	developmentData bool
	developmentErr  error
}

func (c fakeIOPCatalogue) valueIsActive(_ context.Context, profile, value string) (bool, error) {
	if c.lookupErr != nil {
		return false, c.lookupErr
	}
	return c.active[iopCatalogueKey{profile, value}], nil
}

func (c fakeIOPCatalogue) hasDevelopmentCodes(context.Context) (bool, error) {
	return c.developmentData, c.developmentErr
}

func TestIOPDraftRegistryValidate(t *testing.T) {
	valid := []byte(`{"recordMode":"development_raw_mmhg","profileCode":"development_iop_manual_mmhg","eyes":[{"eye":"left","valueCode":"development_iop_14"},{"eye":"right","valueCode":"development_iop_18"}]}`)
	registry, err := newIOPDraftRegistry(context.Background(), fakeIOPCatalogue{active: map[iopCatalogueKey]bool{
		{iopProfileCode, "development_iop_14"}: true,
		{iopProfileCode, "development_iop_18"}: true,
	}}, "development")
	if err != nil {
		t.Fatalf("newIOPDraftRegistry() error = %v", err)
	}
	if err := registry.Validate(context.Background(), iopEventType, episodes.DraftIntentCreate, 1, valid); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}

	for name, payload := range map[string][]byte{
		"unknown property": []byte(`{"recordMode":"development_raw_mmhg","profileCode":"development_iop_manual_mmhg","eyes":[],"extra":true}`),
		"wrong mode":       []byte(`{"recordMode":"raw_mmhg","profileCode":"development_iop_manual_mmhg","eyes":[{"eye":"left","valueCode":"development_iop_14"}]}`),
		"wrong profile":    []byte(`{"recordMode":"development_raw_mmhg","profileCode":"other_profile","eyes":[{"eye":"left","valueCode":"development_iop_14"}]}`),
		"duplicate eye":    []byte(`{"recordMode":"development_raw_mmhg","profileCode":"development_iop_manual_mmhg","eyes":[{"eye":"left","valueCode":"development_iop_14"},{"eye":"left","valueCode":"development_iop_18"}]}`),
		"numeric input":    []byte(`{"recordMode":"development_raw_mmhg","profileCode":"development_iop_manual_mmhg","eyes":[{"eye":"left","valueCode":"development_iop_14","mmhg":14}]}`),
		"inactive value":   []byte(`{"recordMode":"development_raw_mmhg","profileCode":"development_iop_manual_mmhg","eyes":[{"eye":"left","valueCode":"development_iop_99"}]}`),
	} {
		t.Run(name, func(t *testing.T) {
			if err := registry.Validate(context.Background(), iopEventType, episodes.DraftIntentCreate, 1, payload); !errors.Is(err, episodes.ErrInvalidRequest) {
				t.Fatalf("Validate() error = %v, want invalid request", err)
			}
		})
	}
	if err := registry.Validate(context.Background(), iopEventType, episodes.DraftIntentUpdate, 1, valid); !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate(update) error = %v, want invalid request", err)
	}
}

func TestIOPDraftRegistryCatalogueFailureIsNotInvalidRequest(t *testing.T) {
	registry, err := newIOPDraftRegistry(context.Background(), fakeIOPCatalogue{lookupErr: errors.New("database unavailable")}, "development")
	if err != nil {
		t.Fatalf("newIOPDraftRegistry() error = %v", err)
	}
	err = registry.Validate(context.Background(), iopEventType, episodes.DraftIntentCreate, 1, []byte(`{"recordMode":"development_raw_mmhg","profileCode":"development_iop_manual_mmhg","eyes":[{"eye":"left","valueCode":"development_iop_14"}]}`))
	if err == nil || errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate(catalogue failure) error = %v, want unavailable error", err)
	}
}

func TestIOPDraftRegistryProductionGuard(t *testing.T) {
	if _, err := newIOPDraftRegistry(context.Background(), fakeIOPCatalogue{developmentData: true}, "production"); err == nil {
		t.Fatal("production registry accepted development catalogue")
	}
	if _, err := newIOPDraftRegistry(context.Background(), fakeIOPCatalogue{developmentErr: errors.New("query failed")}, "production"); err == nil {
		t.Fatal("production registry accepted an unreadable catalogue")
	}
	if _, err := newIOPDraftRegistry(context.Background(), fakeIOPCatalogue{}, "production"); err != nil {
		t.Fatalf("production registry without development catalogue error = %v", err)
	}
}
