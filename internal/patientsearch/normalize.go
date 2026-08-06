// Package patientsearch implements the patient-search domain contract.
package patientsearch

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	// NameNormalizationVersion identifies the persisted name-normalization algorithm.
	NameNormalizationVersion       int16 = 1
	maxIdentifierCodePoints              = 255
	maxValidationPatternCodePoints       = 255
)

var (
	// ErrInvalidName indicates that a submitted name cannot be normalized.
	ErrInvalidName = errors.New("invalid name")
	// ErrNameTooLong indicates that a submitted name exceeds its field limit.
	ErrNameTooLong = errors.New("name exceeds maximum length")
	// ErrInvalidIdentifier indicates that a value fails its configured rule.
	ErrInvalidIdentifier = errors.New("invalid identifier")
	// ErrIdentifierTooLong indicates that an identifier exceeds an approved limit.
	ErrIdentifierTooLong = errors.New("identifier exceeds maximum length")
	// ErrInvalidIdentifierRule indicates unusable identifier-type configuration.
	ErrInvalidIdentifierRule = errors.New("invalid identifier rule")
)

// NormalizeName applies the approved version-one search normalization.
func NormalizeName(value string, maximumCodePoints int) (string, error) {
	if maximumCodePoints < 1 || !utf8.ValidString(value) {
		return "", ErrInvalidName
	}
	if utf8.RuneCountInString(value) > maximumCodePoints {
		return "", ErrNameTooLong
	}
	if containsControl(value) {
		return "", ErrInvalidName
	}

	normalized := norm.NFC.String(value)
	normalized = strings.TrimFunc(normalized, unicode.IsSpace)
	if normalized == "" {
		return "", ErrInvalidName
	}

	return cases.Fold().String(normalized), nil
}

// IdentifierNormalizationKind names a versioned identifier algorithm.
type IdentifierNormalizationKind string

const (
	// NHSNumberV1 selects ten-digit NHS modulo-11 canonicalization.
	NHSNumberV1 IdentifierNormalizationKind = "nhs_number_v1"
	// ExactTextV1 selects NFC, edge trim, and full configured-pattern matching.
	ExactTextV1 IdentifierNormalizationKind = "exact_text_v1"
	// LeftZeroPadASCIIDigitsV1 selects bounded ASCII-digit left padding.
	LeftZeroPadASCIIDigitsV1 IdentifierNormalizationKind = "left_zero_pad_ascii_digits_v1"
)

// IdentifierRule is validated once before it is used by migration or runtime code.
type IdentifierRule struct {
	Kind                   IdentifierNormalizationKind
	ValidationPattern      string
	MaximumCanonicalLength int
	ZeroPadWidth           int
}

// IdentifierNormalizer is an immutable compiled identifier rule.
type IdentifierNormalizer struct {
	kind                   IdentifierNormalizationKind
	validationPattern      *regexp.Regexp
	maximumCanonicalLength int
	zeroPadWidth           int
}

// CompileIdentifierRule validates configuration and compiles its RE2 pattern.
func CompileIdentifierRule(rule IdentifierRule) (IdentifierNormalizer, error) {
	if rule.MaximumCanonicalLength < 1 || rule.MaximumCanonicalLength > maxIdentifierCodePoints {
		return IdentifierNormalizer{}, ErrInvalidIdentifierRule
	}

	normalizer := IdentifierNormalizer{
		kind:                   rule.Kind,
		maximumCanonicalLength: rule.MaximumCanonicalLength,
		zeroPadWidth:           rule.ZeroPadWidth,
	}

	switch rule.Kind {
	case NHSNumberV1:
		if rule.MaximumCanonicalLength != 10 || rule.ValidationPattern != "" || rule.ZeroPadWidth != 0 {
			return IdentifierNormalizer{}, ErrInvalidIdentifierRule
		}
	case ExactTextV1, LeftZeroPadASCIIDigitsV1:
		pattern, err := compileValidationPattern(rule.ValidationPattern)
		if err != nil {
			return IdentifierNormalizer{}, err
		}
		normalizer.validationPattern = pattern
		if rule.Kind == ExactTextV1 && rule.ZeroPadWidth != 0 {
			return IdentifierNormalizer{}, ErrInvalidIdentifierRule
		}
		if rule.Kind == LeftZeroPadASCIIDigitsV1 &&
			(rule.ZeroPadWidth < 1 || rule.ZeroPadWidth > rule.MaximumCanonicalLength) {
			return IdentifierNormalizer{}, ErrInvalidIdentifierRule
		}
	default:
		return IdentifierNormalizer{}, ErrInvalidIdentifierRule
	}

	return normalizer, nil
}

// Canonicalize returns the configured canonical identity without type fallback.
func (n IdentifierNormalizer) Canonicalize(value string) (string, error) {
	if !utf8.ValidString(value) || containsControl(value) {
		return "", ErrInvalidIdentifier
	}
	if utf8.RuneCountInString(value) > maxIdentifierCodePoints {
		return "", ErrIdentifierTooLong
	}

	canonical := norm.NFC.String(value)
	canonical = strings.TrimFunc(canonical, unicode.IsSpace)

	switch n.kind {
	case NHSNumberV1:
		canonical = strings.NewReplacer(" ", "", "-", "").Replace(canonical)
		if !validNHSNumber(canonical) {
			return "", ErrInvalidIdentifier
		}
	case ExactTextV1:
		if canonical == "" {
			return "", ErrInvalidIdentifier
		}
	case LeftZeroPadASCIIDigitsV1:
		if !isASCIIDigits(canonical) || len(canonical) > n.zeroPadWidth {
			return "", ErrInvalidIdentifier
		}
		canonical = strings.Repeat("0", n.zeroPadWidth-len(canonical)) + canonical
	default:
		return "", ErrInvalidIdentifierRule
	}

	if utf8.RuneCountInString(canonical) > n.maximumCanonicalLength {
		return "", ErrIdentifierTooLong
	}
	if n.validationPattern != nil && !matchesEntireString(n.validationPattern, canonical) {
		return "", ErrInvalidIdentifier
	}

	return canonical, nil
}

func compileValidationPattern(pattern string) (*regexp.Regexp, error) {
	if pattern == "" || !utf8.ValidString(pattern) ||
		utf8.RuneCountInString(pattern) > maxValidationPatternCodePoints ||
		!strings.HasPrefix(pattern, "^") ||
		!(strings.HasSuffix(pattern, "$") || strings.HasSuffix(pattern, `\z`)) {
		return nil, ErrInvalidIdentifierRule
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, ErrInvalidIdentifierRule
	}
	return compiled, nil
}

func matchesEntireString(pattern *regexp.Regexp, value string) bool {
	location := pattern.FindStringIndex(value)
	return location != nil && location[0] == 0 && location[1] == len(value)
}

func validNHSNumber(value string) bool {
	if len(value) != 10 || !isASCIIDigits(value) {
		return false
	}

	sum := 0
	for index := 0; index < 9; index++ {
		sum += int(value[index]-'0') * (10 - index)
	}
	checkDigit := 11 - sum%11
	if checkDigit == 11 {
		checkDigit = 0
	}
	return checkDigit != 10 && checkDigit == int(value[9]-'0')
}

func isASCIIDigits(value string) bool {
	if value == "" {
		return false
	}
	for index := range len(value) {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}
