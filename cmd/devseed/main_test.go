package main

import (
	"reflect"
	"testing"
)

func TestRequireDevelopmentEnvironment(t *testing.T) {
	if err := requireDevelopmentEnvironment("development"); err != nil {
		t.Fatalf("requireDevelopmentEnvironment(development) error = %v", err)
	}
	if err := requireDevelopmentEnvironment("production"); err == nil {
		t.Fatal("requireDevelopmentEnvironment(production) succeeded")
	}
	if err := requireDevelopmentEnvironment(""); err == nil {
		t.Fatal("requireDevelopmentEnvironment(empty) succeeded")
	}
}

func TestDevelopmentIOPCatalogueValuesAreDeterministic(t *testing.T) {
	values := developmentIOPCatalogueValues()
	if len(values) != 100 {
		t.Fatalf("development IOP value count = %d, want 100", len(values))
	}
	if values[0] != (developmentIOPValue{code: "development_iop_00", displayValue: "0 mmHg", mmhg: 0}) {
		t.Fatalf("first development IOP value = %#v", values[0])
	}
	if values[99] != (developmentIOPValue{code: "development_iop_99", displayValue: "99 mmHg", mmhg: 99}) {
		t.Fatalf("last development IOP value = %#v", values[99])
	}
	for index, value := range values {
		if value.mmhg != index {
			t.Fatalf("value %d mmhg = %d", index, value.mmhg)
		}
	}
}

func TestDevelopmentVisualAcuityCatalogueValuesAreDeterministic(t *testing.T) {
	values := developmentVisualAcuityCatalogueValues()
	if len(values) != 91 {
		t.Fatalf("development Visual Acuity value count = %d, want 91", len(values))
	}
	if values[0] != (developmentVisualAcuityValue{code: "development_value_m030", displayValue: "-0.30", baseValue: "-0.3000", order: 0}) {
		t.Fatalf("first development Visual Acuity value = %#v", values[0])
	}
	if values[1] != (developmentVisualAcuityValue{code: "development_value_m028", displayValue: "-0.28", baseValue: "-0.2800", order: 1}) {
		t.Fatalf("second development Visual Acuity value = %#v", values[1])
	}
	if values[90] != (developmentVisualAcuityValue{code: "development_value_150", displayValue: "1.50", baseValue: "1.5000", order: 90}) {
		t.Fatalf("last development Visual Acuity value = %#v", values[90])
	}
}

func TestDevelopmentDiagnosisSelectionsAreDeterministic(t *testing.T) {
	want := []developmentDiagnosisSelection{
		{code: "development_cataract", displayName: "Development cataract example", order: 0},
		{code: "development_glaucoma", displayName: "Development glaucoma example", order: 1},
		{code: "development_macular_condition", displayName: "Development macular-condition example", order: 2},
	}
	if got := developmentDiagnosisSelections(); !reflect.DeepEqual(got, want) {
		t.Fatalf("development diagnosis selections = %#v, want %#v", got, want)
	}
}
