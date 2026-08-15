package examination

import (
	"context"
	"errors"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

type fakeCorrespondenceCatalogue struct {
	active map[string]bool
	demo   bool
}

func (f fakeCorrespondenceCatalogue) codeIsActive(_ context.Context, category, code string) (bool, error) {
	return f.active[category+":"+code], nil
}
func (f fakeCorrespondenceCatalogue) hasDemoCodes(context.Context) (bool, error) { return f.demo, nil }

func TestCorrespondenceDemoDraftRegistryValidate(t *testing.T) {
	r := &CorrespondenceDemoDraftRegistry{catalogue: fakeCorrespondenceCatalogue{active: map[string]bool{
		"template:demo_clinic_update": true, "recipient_role:demo_gp": true,
	}}}
	valid := []byte(`{"recordMode":"demo_correspondence","templateCode":"demo_clinic_update","recipientRole":"demo_gp","subject":"Clinic update","body":"A plain text demonstration letter.\nSecond paragraph.","footer":"VisionOpus demo\nClinic team","clinicDate":"2026-08-01"}`)
	if err := r.Validate(context.Background(), correspondenceDemoEventType, episodes.DraftIntentCreate, 1, valid); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
	for _, payload := range [][]byte{
		[]byte(`{"recordMode":"demo_correspondence","templateCode":"demo_clinic_update","recipientRole":"demo_gp","subject":"<b>bad</b>","body":"body"}`),
		[]byte(`{"recordMode":"demo_correspondence","templateCode":"demo_clinic_update","recipientRole":"demo_gp","subject":"subject","body":"body","clinicDate":"2999-01-01"}`),
		[]byte(`{"recordMode":"demo_correspondence","templateCode":"production_letter","recipientRole":"demo_gp","subject":"subject","body":"body"}`),
	} {
		if !errors.Is(r.Validate(context.Background(), correspondenceDemoEventType, episodes.DraftIntentCreate, 1, payload), episodes.ErrInvalidRequest) {
			t.Fatal("invalid correspondence payload accepted")
		}
	}
}

func TestCorrespondenceDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := newCorrespondenceDemoDraftRegistry(context.Background(), fakeCorrespondenceCatalogue{demo: true}, "production"); err == nil {
		t.Fatal("production accepted demo correspondence catalogue")
	}
}
