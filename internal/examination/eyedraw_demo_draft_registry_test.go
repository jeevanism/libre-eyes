package examination

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jeevanism/visionopus/internal/episodes"
)

func TestEyeDrawDemoDraftRegistryValidate(t *testing.T) {
	registry, err := NewEyeDrawDemoDraftRegistry("development")
	if err != nil {
		t.Fatalf("NewEyeDrawDemoDraftRegistry() error = %v", err)
	}
	valid := []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"development_exam_ant_seg_v1","laterality":"right","drawing":[{"className":"AntSeg"},{"className":"PCIOL"},{"className":"PhakoIncision"},{"className":"SidePort"}]}`)
	if err := registry.Validate(context.Background(), eyeDrawDemoEventType, episodes.DraftIntentCreate, 1, valid); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}

	for name, payload := range map[string][]byte{
		"unknown root property": []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"development_exam_ant_seg_v1","laterality":"right","drawing":[],"extra":true}`),
		"wrong canvas":          []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"other","laterality":"right","drawing":[]}`),
		"standalone pupil":      []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"development_exam_ant_seg_v1","laterality":"right","drawing":[{"className":"Pupil"}]}`),
		"unapproved doodle":     []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"development_exam_ant_seg_v1","laterality":"right","drawing":[{"className":"Fundus"}]}`),
		"mismatched subclass":   []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"development_exam_ant_seg_v1","laterality":"right","drawing":[{"className":"AntSeg","subclass":"Fundus"}]}`),
		"array item":            []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"development_exam_ant_seg_v1","laterality":"right","drawing":[[]]}`),
		"too many properties":   []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"development_exam_ant_seg_v1","laterality":"right","drawing":[{"className":"AntSeg","a":1,"b":2,"c":3,"d":4,"e":5,"f":6,"g":7,"h":8,"i":9,"j":10,"k":11,"l":12,"m":13,"n":14,"o":15,"p":16,"q":17,"r":18,"s":19,"t":20,"u":21,"v":22,"w":23,"x":24,"y":25,"z":26,"aa":27,"ab":28,"ac":29,"ad":30,"ae":31,"af":32,"ag":33,"ah":34,"ai":35,"aj":36,"ak":37,"al":38,"am":39,"an":40,"ao":41,"ap":42,"aq":43,"ar":44,"as":45,"at":46,"au":47,"av":48}]}`),
	} {
		t.Run(name, func(t *testing.T) {
			if err := registry.Validate(context.Background(), eyeDrawDemoEventType, episodes.DraftIntentCreate, 1, payload); !errors.Is(err, episodes.ErrInvalidRequest) {
				t.Fatalf("Validate() error = %v, want invalid request", err)
			}
		})
	}

	oversized := []byte(`{"recordMode":"development_eyedraw_anterior_segment","canvasCode":"development_exam_ant_seg_v1","laterality":"right","drawing":[{"className":"AntSeg","note":"` + strings.Repeat("x", eyeDrawDemoMaximumBytes) + `"}]}`)
	if err := registry.Validate(context.Background(), eyeDrawDemoEventType, episodes.DraftIntentCreate, 1, oversized); !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate(oversized) error = %v, want invalid request", err)
	}
	if err := registry.Validate(context.Background(), eyeDrawDemoEventType, episodes.DraftIntentUpdate, 1, valid); !errors.Is(err, episodes.ErrInvalidRequest) {
		t.Fatalf("Validate(update intent) error = %v, want invalid request", err)
	}
}

func TestEyeDrawDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := NewEyeDrawDemoDraftRegistry("production"); err == nil {
		t.Fatal("production registry accepted development EyeDraw demo")
	}
}
