package admin

import "testing"

func TestResolveSettingUsesMostSpecificScope(t *testing.T) {
	got, err := ResolveSetting([]SettingCandidate{
		{Key: "queue", Value: "system", Scope: ScopeSystem},
		{Key: "queue", Value: "institution", Scope: ScopeInstitution, InstitutionID: 5},
		{Key: "queue", Value: "site", Scope: ScopeSite, InstitutionID: 5, SiteID: 7},
		{Key: "queue", Value: "firm", Scope: ScopeFirm, InstitutionID: 5, FirmID: 9},
	}, "queue", 5, 7, 9)
	if err != nil || got.Value != "firm" || got.Scope != ScopeFirm {
		t.Fatalf("got %#v, err %v", got, err)
	}
}

func TestResolveSettingIgnoresOtherContexts(t *testing.T) {
	got, err := ResolveSetting([]SettingCandidate{
		{Key: "queue", Value: "other", Scope: ScopeInstitution, InstitutionID: 6},
		{Key: "queue", Value: "current", Scope: ScopeInstitution, InstitutionID: 5},
	}, "queue", 5, 7, 9)
	if err != nil || got.Value != "current" {
		t.Fatalf("got %#v, err %v", got, err)
	}
}

func TestResolveSettingRejectsAmbiguousCandidates(t *testing.T) {
	_, err := ResolveSetting([]SettingCandidate{
		{Key: "queue", Value: "one", Scope: ScopeSite, InstitutionID: 5, SiteID: 7},
		{Key: "queue", Value: "two", Scope: ScopeSite, InstitutionID: 5, SiteID: 7},
	}, "queue", 5, 7, 9)
	if err != ErrAmbiguousSetting {
		t.Fatalf("expected ambiguity, got %v", err)
	}
}
