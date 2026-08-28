package branding

import (
	"errors"
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
