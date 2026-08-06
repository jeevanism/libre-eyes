package patientsearch

import (
	"errors"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		maximum int
		want    string
		wantErr error
	}{
		{name: "composed", input: "  ÉLODIE  ", maximum: 100, want: "élodie"},
		{name: "decomposed", input: "E\u0301lodie", maximum: 100, want: "élodie"},
		{name: "default fold", input: "Straße", maximum: 100, want: "strasse"},
		{name: "retains punctuation and inner space", input: " O'Neil-Smith  Junior ", maximum: 100, want: "o'neil-smith  junior"},
		{name: "unicode edge whitespace", input: "\u00a0Patient\u2003", maximum: 100, want: "patient"},
		{name: "empty after trim", input: "   ", maximum: 100, wantErr: ErrInvalidName},
		{name: "control before trim", input: "\nPatient", maximum: 100, wantErr: ErrInvalidName},
		{name: "invalid utf8", input: string([]byte{0xff}), maximum: 100, wantErr: ErrInvalidName},
		{name: "too long before trim", input: " abc ", maximum: 3, wantErr: ErrNameTooLong},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeName(test.input, test.maximum)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("NormalizeName() error = %v, want %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("NormalizeName() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestIdentifierCanonicalization(t *testing.T) {
	nhs := mustCompileRule(t, IdentifierRule{Kind: NHSNumberV1, MaximumCanonicalLength: 10})
	exact := mustCompileRule(t, IdentifierRule{
		Kind:                   ExactTextV1,
		ValidationPattern:      `^[A-Za-z\x{00C9}\x{00E9}' -]+$`,
		MaximumCanonicalLength: 20,
	})
	padding := mustCompileRule(t, IdentifierRule{
		Kind:                   LeftZeroPadASCIIDigitsV1,
		ValidationPattern:      `^[0-9]{8}$`,
		MaximumCanonicalLength: 8,
		ZeroPadWidth:           8,
	})

	tests := []struct {
		name       string
		normalizer IdentifierNormalizer
		input      string
		want       string
		wantErr    error
	}{
		{name: "nhs digits", normalizer: nhs, input: "9435678904", want: "9435678904"},
		{name: "nhs spaces", normalizer: nhs, input: "943 567 8904", want: "9435678904"},
		{name: "nhs hyphens", normalizer: nhs, input: "943-567-8904", want: "9435678904"},
		{name: "nhs invalid check digit", normalizer: nhs, input: "9435678905", wantErr: ErrInvalidIdentifier},
		{name: "nhs check digit zero", normalizer: nhs, input: "943 567 8890", want: "9435678890"},
		{name: "nhs unusable check digit ten", normalizer: nhs, input: "943 567 8841", wantErr: ErrInvalidIdentifier},
		{name: "nhs unicode digit rejected", normalizer: nhs, input: "٩٤٣٤٧٦٥٩١٩", wantErr: ErrInvalidIdentifier},
		{name: "exact nfc trim preserves case", normalizer: exact, input: "  E\u0301lodie-Smith  ", want: "Élodie-Smith"},
		{name: "exact internal spacing preserved", normalizer: exact, input: "A  B", want: "A  B"},
		{name: "exact pattern rejects", normalizer: exact, input: "A_1", wantErr: ErrInvalidIdentifier},
		{name: "padding", normalizer: padding, input: "123", want: "00000123"},
		{name: "padding accepts full width", normalizer: padding, input: "12345678", want: "12345678"},
		{name: "padding rejects too wide", normalizer: padding, input: "123456789", wantErr: ErrInvalidIdentifier},
		{name: "padding rejects non ascii", normalizer: padding, input: "12٣", wantErr: ErrInvalidIdentifier},
		{name: "controls rejected", normalizer: exact, input: "ABC\n", wantErr: ErrInvalidIdentifier},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.normalizer.Canonicalize(test.input)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Canonicalize() error = %v, want %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("Canonicalize() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCompileIdentifierRuleRejectsInvalidConfiguration(t *testing.T) {
	tests := []IdentifierRule{
		{Kind: "unsupported", MaximumCanonicalLength: 10},
		{Kind: NHSNumberV1, MaximumCanonicalLength: 9},
		{Kind: NHSNumberV1, ValidationPattern: `^[0-9]+$`, MaximumCanonicalLength: 10},
		{Kind: ExactTextV1, ValidationPattern: `[0-9]+`, MaximumCanonicalLength: 10},
		{Kind: ExactTextV1, ValidationPattern: `^[$`, MaximumCanonicalLength: 10},
		{Kind: ExactTextV1, ValidationPattern: `^[0-9]+$`, MaximumCanonicalLength: 256},
		{Kind: LeftZeroPadASCIIDigitsV1, ValidationPattern: `^[0-9]+$`, MaximumCanonicalLength: 8},
		{Kind: LeftZeroPadASCIIDigitsV1, ValidationPattern: `^[0-9]+$`, MaximumCanonicalLength: 8, ZeroPadWidth: 9},
		{Kind: ExactTextV1, ValidationPattern: "^" + strings.Repeat("a", 254) + "$", MaximumCanonicalLength: 10},
		{Kind: ExactTextV1, ValidationPattern: string([]byte{'^', 0xff, '$'}), MaximumCanonicalLength: 10},
	}

	for index, rule := range tests {
		if _, err := CompileIdentifierRule(rule); !errors.Is(err, ErrInvalidIdentifierRule) {
			t.Fatalf("case %d CompileIdentifierRule() error = %v, want invalid rule", index, err)
		}
	}
}

func FuzzNormalizeName(f *testing.F) {
	f.Add(" Élodie-Smith ")
	f.Add("Patient")
	f.Fuzz(func(t *testing.T, input string) {
		got, err := NormalizeName(input, 300)
		if err != nil {
			return
		}
		if got == "" || !utf8.ValidString(got) {
			t.Fatalf("successful normalization produced invalid output")
		}
		if strings.TrimFunc(got, unicode.IsSpace) != got {
			t.Fatalf("successful normalization retained edge whitespace")
		}
		for _, character := range got {
			if unicode.IsControl(character) {
				t.Fatalf("successful normalization retained a control character")
			}
		}
	})
}

func FuzzExactIdentifierCanonicalization(f *testing.F) {
	normalizer := mustCompileRule(f, IdentifierRule{
		Kind:                   ExactTextV1,
		ValidationPattern:      `^.{1,255}$`,
		MaximumCanonicalLength: 255,
	})
	f.Add(" ABC-123 ")
	f.Add("E\u0301lodie")
	f.Fuzz(func(t *testing.T, input string) {
		got, err := normalizer.Canonicalize(input)
		if err != nil {
			return
		}
		if got == "" || !utf8.ValidString(got) || !norm.NFC.IsNormalString(got) {
			t.Fatalf("successful canonicalization produced invalid output")
		}
		if utf8.RuneCountInString(got) > 255 {
			t.Fatalf("successful canonicalization exceeded its configured maximum")
		}
		second, err := normalizer.Canonicalize(got)
		if err != nil || second != got {
			t.Fatalf("successful canonicalization is not idempotent")
		}
	})
}

type testingTB interface {
	Helper()
	Fatalf(string, ...any)
}

func mustCompileRule(t testingTB, rule IdentifierRule) IdentifierNormalizer {
	t.Helper()
	normalizer, err := CompileIdentifierRule(rule)
	if err != nil {
		t.Fatalf("CompileIdentifierRule() error = %v", err)
	}
	return normalizer
}
