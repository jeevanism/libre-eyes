package branding

import (
	"errors"
	"math"
	"testing"
)

func TestValidateInputAcceptsAccessibleSemanticColors(t *testing.T) {
	in := DraftInput{
		OrganizationName: "Velox Group EyeCare",
		ShortName:        "Velox EyeCare",
		BrowserTitle:     "Velox EyeCare clinical workspace",
		Colors:           Colors{Primary: "#17543f", PrimaryHover: "#103d2e", SelectedSurface: "#dcefe5", Focus: "#005fcc"},
	}
	if err := validateInput(in); err != nil {
		t.Fatalf("validateInput() error = %v", err)
	}
}

func TestValidateInputRejectsUnsafeOrInaccessibleValues(t *testing.T) {
	base := DraftInput{
		OrganizationName: "Velox Group EyeCare",
		ShortName:        "Velox EyeCare",
		BrowserTitle:     "Velox EyeCare",
		Colors:           Colors{Primary: "#17543f", PrimaryHover: "#103d2e", SelectedSurface: "#dcefe5", Focus: "#005fcc"},
	}

	tests := map[string]DraftInput{
		"arbitrary CSS":       func() DraftInput { value := base; value.Colors.Primary = "red; display:none"; return value }(),
		"low contrast action": func() DraftInput { value := base; value.Colors.Primary = "#f5f5f5"; return value }(),
		"empty identity":      func() DraftInput { value := base; value.OrganizationName = " "; return value }(),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateInput(input); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("validateInput() error = %v, want ErrInvalidRequest", err)
			}
		})
	}
}

func TestValidateInputReportsEveryInaccessibleColour(t *testing.T) {
	input := DraftInput{
		OrganizationName: "Velox Group EyeCare",
		ShortName:        "Velox EyeCare",
		BrowserTitle:     "Velox EyeCare",
		Colors: Colors{
			Primary:         "#e4e651",
			PrimaryHover:    "#f03891",
			SelectedSurface: "#c953ea",
			Focus:           "#927595",
		},
	}

	err := validateInput(input)
	var contrastError *ContrastValidationError
	if !errors.As(err, &contrastError) {
		t.Fatalf("validateInput() error = %v, want ContrastValidationError", err)
	}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("validateInput() error does not wrap ErrInvalidRequest")
	}
	if len(contrastError.Issues) != 2 {
		t.Fatalf("issues = %#v, want primary and primary hover failures", contrastError.Issues)
	}
	if contrastError.Issues[0].Field != "colors.primary" || math.Abs(contrastError.Issues[0].ContrastRatio-1.33) > 0.01 {
		t.Fatalf("primary issue = %#v", contrastError.Issues[0])
	}
	if contrastError.Issues[1].Field != "colors.primaryHover" || math.Abs(contrastError.Issues[1].ContrastRatio-3.70) > 0.01 {
		t.Fatalf("hover issue = %#v", contrastError.Issues[1])
	}
}

func TestDefaultProfileIsAccessibleFallback(t *testing.T) {
	profile := DefaultProfile()
	err := validateInput(DraftInput{
		OrganizationName: profile.OrganizationName,
		ShortName:        profile.ShortName,
		BrowserTitle:     profile.BrowserTitle,
		Colors:           profile.Colors,
	})
	if err != nil {
		t.Fatalf("default profile validation error = %v", err)
	}
}
