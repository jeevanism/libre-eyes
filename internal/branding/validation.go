package branding

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// DefaultProfile is the safe embedded fallback used when no published tenant profile is available.
func DefaultProfile() Profile {
	return Profile{
		ProfileVersion:   1,
		RowVersion:       1,
		Status:           "default",
		Source:           "default",
		OrganizationName: "LibreEyes",
		ShortName:        "LibreEyes",
		BrowserTitle:     "LibreEyes",
		Colors: Colors{
			Primary:         "#116466",
			PrimaryHover:    "#0c5355",
			SelectedSurface: "#deefee",
			Focus:           "#0b6fcc",
		},
	}
}

func normalizeInput(in DraftInput) DraftInput {
	in.OrganizationName = strings.TrimSpace(in.OrganizationName)
	in.ShortName = strings.TrimSpace(in.ShortName)
	in.BrowserTitle = strings.TrimSpace(in.BrowserTitle)
	in.Colors.Primary = strings.ToLower(strings.TrimSpace(in.Colors.Primary))
	in.Colors.PrimaryHover = strings.ToLower(strings.TrimSpace(in.Colors.PrimaryHover))
	in.Colors.SelectedSurface = strings.ToLower(strings.TrimSpace(in.Colors.SelectedSurface))
	in.Colors.Focus = strings.ToLower(strings.TrimSpace(in.Colors.Focus))
	return in
}

func validateInput(in DraftInput) error {
	in = normalizeInput(in)
	if len(in.OrganizationName) < 2 || len(in.OrganizationName) > 120 || len(in.ShortName) < 2 || len(in.ShortName) > 60 || len(in.BrowserTitle) < 2 || len(in.BrowserTitle) > 80 {
		return ErrInvalidRequest
	}
	colors := []string{in.Colors.Primary, in.Colors.PrimaryHover, in.Colors.SelectedSurface, in.Colors.Focus}
	for _, color := range colors {
		if !hexColorPattern.MatchString(color) {
			return ErrInvalidRequest
		}
	}
	checks := []struct {
		field      string
		label      string
		color      string
		background string
		against    string
		minimum    float64
	}{
		{
			field: "colors.primary", label: "Primary action", color: in.Colors.Primary,
			background: "#ffffff", against: "white text", minimum: 4.5,
		},
		{
			field: "colors.primaryHover", label: "Primary hover", color: in.Colors.PrimaryHover,
			background: "#ffffff", against: "white text", minimum: 4.5,
		},
		{
			field: "colors.selectedSurface", label: "Selected surface", color: in.Colors.SelectedSurface,
			background: "#172124", against: "interface text", minimum: 4.5,
		},
		{
			field: "colors.focus", label: "Focus indicator", color: in.Colors.Focus,
			background: "#ffffff", against: "white surface", minimum: 3,
		},
	}
	issues := make([]ContrastIssue, 0, len(checks))
	for _, check := range checks {
		ratio := contrastRatio(check.color, check.background)
		if ratio >= check.minimum {
			continue
		}
		message := fmt.Sprintf(
			"%s colour has %.2f:1 contrast against %s; at least %.1f:1 is required.",
			check.label,
			ratio,
			check.against,
			check.minimum,
		)
		issues = append(issues, ContrastIssue{
			Field:           check.field,
			Label:           check.label,
			Against:         check.against,
			Message:         message,
			ContrastRatio:   ratio,
			MinimumContrast: check.minimum,
		})
	}
	if len(issues) > 0 {
		return &ContrastValidationError{Issues: issues}
	}
	return nil
}

func contrastRatio(first, second string) float64 {
	left := relativeLuminance(first)
	right := relativeLuminance(second)
	if left < right {
		left, right = right, left
	}
	return (left + 0.05) / (right + 0.05)
}

func relativeLuminance(color string) float64 {
	channel := func(value string) float64 {
		parsed, err := strconv.ParseUint(value, 16, 8)
		if err != nil {
			return 0
		}
		normalized := float64(parsed) / 255
		if normalized <= 0.04045 {
			return normalized / 12.92
		}
		return math.Pow((normalized+0.055)/1.055, 2.4)
	}
	if !hexColorPattern.MatchString(color) {
		return 0
	}
	return 0.2126*channel(color[1:3]) + 0.7152*channel(color[3:5]) + 0.0722*channel(color[5:7])
}
