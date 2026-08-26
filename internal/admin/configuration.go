package admin

import "errors"

// SettingScope describes where a configuration value applies.
type SettingScope string

const (
	ScopeSystem      SettingScope = "system"
	ScopeInstitution SettingScope = "institution"
	ScopeSite        SettingScope = "site"
	ScopeFirm        SettingScope = "firm"
)

var ErrAmbiguousSetting = errors.New("configuration has multiple equally specific values")

type SettingCandidate struct {
	Key           string
	Value         string
	Version       int64
	Scope         SettingScope
	InstitutionID int64
	SiteID        int64
	FirmID        int64
}

type EffectiveSetting struct {
	Key           string
	Value         string
	Version       int64
	Scope         SettingScope
	InstitutionID int64
	SiteID        int64
	FirmID        int64
}

// ResolveSetting selects the most specific candidate for a context. Candidates
// outside the current institution or context are ignored.
func ResolveSetting(candidates []SettingCandidate, key string, institutionID, siteID, firmID int64) (EffectiveSetting, error) {
	best := EffectiveSetting{}
	bestRank := -1
	for _, candidate := range candidates {
		if candidate.Key != key || candidate.InstitutionID != 0 && candidate.InstitutionID != institutionID {
			continue
		}
		rank, matches := settingRank(candidate, institutionID, siteID, firmID)
		if !matches {
			continue
		}
		if rank == bestRank {
			return EffectiveSetting{}, ErrAmbiguousSetting
		}
		if rank > bestRank {
			bestRank = rank
			best = EffectiveSetting{Key: candidate.Key, Value: candidate.Value, Version: candidate.Version, Scope: candidate.Scope, InstitutionID: candidate.InstitutionID, SiteID: candidate.SiteID, FirmID: candidate.FirmID}
		}
	}
	if bestRank < 0 {
		return EffectiveSetting{}, ErrInvalidRequest
	}
	return best, nil
}

func settingRank(candidate SettingCandidate, institutionID, siteID, firmID int64) (int, bool) {
	switch candidate.Scope {
	case ScopeSystem:
		return 0, candidate.InstitutionID == 0
	case ScopeInstitution:
		return 1, candidate.InstitutionID == institutionID && candidate.SiteID == 0 && candidate.FirmID == 0
	case ScopeSite:
		return 2, candidate.InstitutionID == institutionID && candidate.SiteID == siteID && candidate.SiteID != 0 && candidate.FirmID == 0
	case ScopeFirm:
		return 3, candidate.InstitutionID == institutionID && candidate.FirmID == firmID && candidate.FirmID != 0 && candidate.SiteID == 0
	default:
		return -1, false
	}
}
