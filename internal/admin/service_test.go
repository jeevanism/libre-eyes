package admin

import "testing"

func TestValidCatalogueItem(t *testing.T) {
	base := CatalogueUpsert{Category: "medication", Code: "development_demo_drop", DisplayName: "Demo drop", Active: true, DisplayOrder: 1}
	if !validCatalogueItem(base) {
		t.Fatal("expected development catalogue item to be valid")
	}
	for name, item := range map[string]CatalogueUpsert{
		"production code":  {Category: "medication", Code: "production_drop", DisplayName: "Drop"},
		"unknown category": {Category: "drug", Code: "development_drop", DisplayName: "Drop"},
		"negative order":   {Category: "medication", Code: "development_drop", DisplayName: "Drop", DisplayOrder: -1},
	} {
		t.Run(name, func(t *testing.T) {
			if validCatalogueItem(item) {
				t.Fatal("expected catalogue item to be rejected")
			}
		})
	}
}

func TestValidSettingValue(t *testing.T) {
	if !validSettingValue("theatre_capacity_minutes", "240") {
		t.Fatal("expected positive theatre capacity to be valid")
	}
	if validSettingValue("theatre_capacity_minutes", "0") {
		t.Fatal("expected zero capacity to be rejected")
	}
	if !validSettingValue("clinic_flow_priorities", "routine,urgent") {
		t.Fatal("expected supported priorities to be valid")
	}
	if validSettingValue("clinic_flow_priorities", "routine,critical") {
		t.Fatal("expected unsupported priority to be rejected")
	}
}
