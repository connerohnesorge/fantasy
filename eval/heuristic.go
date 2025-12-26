package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
)

// ExactMatchScorer returns a scorer that checks for exact string equality.
type ExactMatchScorer struct {
	name         string
	outputKey    string
	expectedKey  string
	caseSensitive bool
}

// NewExactMatch creates an ExactMatchScorer with default options.
// It compares the output value at outputKey with the expectation at expectedKey.
func NewExactMatch(outputKey, expectedKey string) *ExactMatchScorer {
	return &ExactMatchScorer{
		name:         "ExactMatch",
		outputKey:    outputKey,
		expectedKey:  expectedKey,
		caseSensitive: true,
	}
}

// WithCaseSensitive sets whether the comparison is case-sensitive.
func (s *ExactMatchScorer) WithCaseSensitive(sensitive bool) *ExactMatchScorer {
	s.caseSensitive = sensitive
	return s
}

// WithName sets a custom name for the scorer.
func (s *ExactMatchScorer) WithName(name string) *ExactMatchScorer {
	s.name = name
	return s
}

// Name returns the scorer's identifier.
func (s *ExactMatchScorer) Name() string {
	return s.name
}

// Score evaluates whether the output exactly matches the expected value.
func (s *ExactMatchScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract values
	output, ok := input.Outputs[s.outputKey]
	if !ok {
		return Score{
			Error:     fmt.Errorf("output key %q not found", s.outputKey),
			Rationale: fmt.Sprintf("Missing output key: %s", s.outputKey),
		}, nil
	}

	expected, ok := input.Expectations[s.expectedKey]
	if !ok {
		return Score{
			Error:     fmt.Errorf("expectation key %q not found", s.expectedKey),
			Rationale: fmt.Sprintf("Missing expectation key: %s", s.expectedKey),
		}, nil
	}

	// Convert to strings
	outputStr := fmt.Sprintf("%v", output)
	expectedStr := fmt.Sprintf("%v", expected)

	// Compare
	var matches bool
	if s.caseSensitive {
		matches = outputStr == expectedStr
	} else {
		matches = strings.EqualFold(outputStr, expectedStr)
	}

	rationale := fmt.Sprintf("Output %q vs Expected %q", outputStr, expectedStr)
	if matches {
		rationale = "Output exactly matches expected value"
	}

	return Score{
		Value:     matches,
		Rationale: rationale,
		Metadata: map[string]any{
			"output":   outputStr,
			"expected": expectedStr,
		},
	}, nil
}

// ContainsScorer checks if the output contains a substring.
type ContainsScorer struct {
	name          string
	outputKey     string
	expectedKey   string
	caseSensitive bool
}

// NewContains creates a ContainsScorer with default options.
func NewContains(outputKey, expectedKey string) *ContainsScorer {
	return &ContainsScorer{
		name:          "Contains",
		outputKey:     outputKey,
		expectedKey:   expectedKey,
		caseSensitive: true,
	}
}

// WithCaseSensitive sets whether the comparison is case-sensitive.
func (s *ContainsScorer) WithCaseSensitive(sensitive bool) *ContainsScorer {
	s.caseSensitive = sensitive
	return s
}

// WithName sets a custom name for the scorer.
func (s *ContainsScorer) WithName(name string) *ContainsScorer {
	s.name = name
	return s
}

// Name returns the scorer's identifier.
func (s *ContainsScorer) Name() string {
	return s.name
}

// Score evaluates whether the output contains the expected substring.
func (s *ContainsScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract values
	output, ok := input.Outputs[s.outputKey]
	if !ok {
		return Score{
			Error:     fmt.Errorf("output key %q not found", s.outputKey),
			Rationale: fmt.Sprintf("Missing output key: %s", s.outputKey),
		}, nil
	}

	expected, ok := input.Expectations[s.expectedKey]
	if !ok {
		return Score{
			Error:     fmt.Errorf("expectation key %q not found", s.expectedKey),
			Rationale: fmt.Sprintf("Missing expectation key: %s", s.expectedKey),
		}, nil
	}

	// Convert to strings
	outputStr := fmt.Sprintf("%v", output)
	expectedStr := fmt.Sprintf("%v", expected)

	// Check substring
	var contains bool
	if s.caseSensitive {
		contains = strings.Contains(outputStr, expectedStr)
	} else {
		contains = strings.Contains(strings.ToLower(outputStr), strings.ToLower(expectedStr))
	}

	rationale := fmt.Sprintf("Output does not contain %q", expectedStr)
	if contains {
		rationale = fmt.Sprintf("Output contains %q", expectedStr)
	}

	return Score{
		Value:     contains,
		Rationale: rationale,
		Metadata: map[string]any{
			"output":   outputStr,
			"expected": expectedStr,
		},
	}, nil
}

// RegexScorer checks if the output matches a regex pattern.
type RegexScorer struct {
	name        string
	outputKey   string
	expectedKey string
	pattern     *regexp.Regexp
}

// NewRegex creates a RegexScorer that compiles the pattern from expectations.
func NewRegex(outputKey, expectedKey string) *RegexScorer {
	return &RegexScorer{
		name:        "Regex",
		outputKey:   outputKey,
		expectedKey: expectedKey,
	}
}

// WithPattern sets a pre-compiled regex pattern.
func (s *RegexScorer) WithPattern(pattern *regexp.Regexp) *RegexScorer {
	s.pattern = pattern
	return s
}

// WithName sets a custom name for the scorer.
func (s *RegexScorer) WithName(name string) *RegexScorer {
	s.name = name
	return s
}

// Name returns the scorer's identifier.
func (s *RegexScorer) Name() string {
	return s.name
}

// Score evaluates whether the output matches the regex pattern.
func (s *RegexScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract output
	output, ok := input.Outputs[s.outputKey]
	if !ok {
		return Score{
			Error:     fmt.Errorf("output key %q not found", s.outputKey),
			Rationale: fmt.Sprintf("Missing output key: %s", s.outputKey),
		}, nil
	}

	outputStr := fmt.Sprintf("%v", output)

	// Get pattern
	var pattern *regexp.Regexp
	if s.pattern != nil {
		pattern = s.pattern
	} else {
		// Get pattern from expectations
		expectedPattern, ok := input.Expectations[s.expectedKey]
		if !ok {
			return Score{
				Error:     fmt.Errorf("expectation key %q not found", s.expectedKey),
				Rationale: fmt.Sprintf("Missing expectation key: %s", s.expectedKey),
			}, nil
		}

		patternStr := fmt.Sprintf("%v", expectedPattern)
		var err error
		pattern, err = regexp.Compile(patternStr)
		if err != nil {
			return Score{
				Error:     fmt.Errorf("invalid regex pattern %q: %w", patternStr, err),
				Rationale: fmt.Sprintf("Failed to compile regex pattern: %v", err),
			}, nil
		}
	}

	// Test pattern
	matches := pattern.MatchString(outputStr)

	rationale := fmt.Sprintf("Output does not match pattern %q", pattern.String())
	if matches {
		rationale = fmt.Sprintf("Output matches pattern %q", pattern.String())
	}

	return Score{
		Value:     matches,
		Rationale: rationale,
		Metadata: map[string]any{
			"output":  outputStr,
			"pattern": pattern.String(),
		},
	}, nil
}

// JSONMatchOptions configures JSON comparison behavior.
type JSONMatchOptions struct {
	// NullEqualsMissing: if true, null values are considered equal to missing keys
	// Default: false (null and missing are different)
	NullEqualsMissing bool

	// IgnoreArrayOrder: if true, array elements are compared as sets (order doesn't matter)
	// Default: false (array order matters)
	IgnoreArrayOrder bool

	// FloatTolerance: tolerance for float comparisons (absolute difference)
	// Default: 0 (exact match required)
	FloatTolerance float64
}

// JSONMatchScorer compares JSON structures semantically.
type JSONMatchScorer struct {
	name        string
	outputKey   string
	expectedKey string
	options     JSONMatchOptions
}

// NewJSONMatch creates a JSONMatchScorer with default options.
func NewJSONMatch(outputKey, expectedKey string) *JSONMatchScorer {
	return &JSONMatchScorer{
		name:        "JSONMatch",
		outputKey:   outputKey,
		expectedKey: expectedKey,
		options:     JSONMatchOptions{},
	}
}

// WithOptions sets JSON comparison options.
func (s *JSONMatchScorer) WithOptions(opts JSONMatchOptions) *JSONMatchScorer {
	s.options = opts
	return s
}

// WithName sets a custom name for the scorer.
func (s *JSONMatchScorer) WithName(name string) *JSONMatchScorer {
	s.name = name
	return s
}

// Name returns the scorer's identifier.
func (s *JSONMatchScorer) Name() string {
	return s.name
}

// Score evaluates whether the output JSON matches the expected JSON structure.
func (s *JSONMatchScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract values
	output, ok := input.Outputs[s.outputKey]
	if !ok {
		return Score{
			Error:     fmt.Errorf("output key %q not found", s.outputKey),
			Rationale: fmt.Sprintf("Missing output key: %s", s.outputKey),
		}, nil
	}

	expected, ok := input.Expectations[s.expectedKey]
	if !ok {
		return Score{
			Error:     fmt.Errorf("expectation key %q not found", s.expectedKey),
			Rationale: fmt.Sprintf("Missing expectation key: %s", s.expectedKey),
		}, nil
	}

	// Parse JSON if strings
	outputVal := s.parseJSON(output)
	expectedVal := s.parseJSON(expected)

	// Compare
	matches, diff := s.compareJSON(outputVal, expectedVal)

	rationale := "JSON structures match"
	if !matches {
		rationale = fmt.Sprintf("JSON structures differ: %s", diff)
	}

	return Score{
		Value:     matches,
		Rationale: rationale,
		Metadata: map[string]any{
			"diff": diff,
		},
	}, nil
}

// parseJSON attempts to parse a value as JSON if it's a string.
func (s *JSONMatchScorer) parseJSON(v any) any {
	str, ok := v.(string)
	if !ok {
		return v
	}

	var parsed any
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		// Not valid JSON, return original string
		return v
	}
	return parsed
}

// compareJSON recursively compares two JSON values.
func (s *JSONMatchScorer) compareJSON(output, expected any) (bool, string) {
	// Handle nil
	if output == nil && expected == nil {
		return true, ""
	}
	if output == nil {
		if s.options.NullEqualsMissing {
			return true, ""
		}
		return false, "output is nil but expected is not"
	}
	if expected == nil {
		if s.options.NullEqualsMissing {
			return true, ""
		}
		return false, "expected is nil but output is not"
	}

	// Get types
	outputType := reflect.TypeOf(output).Kind()
	expectedType := reflect.TypeOf(expected).Kind()

	if outputType != expectedType {
		return false, fmt.Sprintf("type mismatch: output is %v, expected is %v", outputType, expectedType)
	}

	switch expectedType {
	case reflect.Map:
		return s.compareMaps(output, expected)
	case reflect.Slice:
		return s.compareArrays(output, expected)
	case reflect.Float64, reflect.Float32:
		return s.compareFloats(output, expected)
	default:
		if output == expected {
			return true, ""
		}
		return false, fmt.Sprintf("values differ: %v != %v", output, expected)
	}
}

// compareMaps compares two map values.
func (s *JSONMatchScorer) compareMaps(output, expected any) (bool, string) {
	outMap, ok1 := output.(map[string]any)
	expMap, ok2 := expected.(map[string]any)
	if !ok1 || !ok2 {
		return false, "map type assertion failed"
	}

	// Check all expected keys
	for key, expVal := range expMap {
		outVal, exists := outMap[key]
		if !exists {
			if expVal == nil && s.options.NullEqualsMissing {
				continue
			}
			return false, fmt.Sprintf("missing key: %s", key)
		}

		matches, diff := s.compareJSON(outVal, expVal)
		if !matches {
			return false, fmt.Sprintf("key %s: %s", key, diff)
		}
	}

	// Check for extra keys in output (not in expected)
	for key := range outMap {
		if _, exists := expMap[key]; !exists {
			// Extra keys are allowed - we only check that expected keys match
			continue
		}
	}

	return true, ""
}

// compareArrays compares two array values.
func (s *JSONMatchScorer) compareArrays(output, expected any) (bool, string) {
	outArr, ok1 := output.([]any)
	expArr, ok2 := expected.([]any)
	if !ok1 || !ok2 {
		return false, "array type assertion failed"
	}

	if len(outArr) != len(expArr) {
		return false, fmt.Sprintf("array length mismatch: %d != %d", len(outArr), len(expArr))
	}

	if s.options.IgnoreArrayOrder {
		// Compare as sets - each expected element must have a match in output
		matched := make([]bool, len(outArr))
		for _, expVal := range expArr {
			found := false
			for i, outVal := range outArr {
				if matched[i] {
					continue
				}
				if matches, _ := s.compareJSON(outVal, expVal); matches {
					matched[i] = true
					found = true
					break
				}
			}
			if !found {
				return false, fmt.Sprintf("expected element not found in output: %v", expVal)
			}
		}
		return true, ""
	} else {
		// Compare element by element
		for i := range expArr {
			matches, diff := s.compareJSON(outArr[i], expArr[i])
			if !matches {
				return false, fmt.Sprintf("index %d: %s", i, diff)
			}
		}
		return true, ""
	}
}

// compareFloats compares two float values with tolerance.
func (s *JSONMatchScorer) compareFloats(output, expected any) (bool, string) {
	outFloat, ok1 := toFloat64(output)
	expFloat, ok2 := toFloat64(expected)
	if !ok1 || !ok2 {
		return false, "float conversion failed"
	}

	diff := math.Abs(outFloat - expFloat)
	if diff <= s.options.FloatTolerance {
		return true, ""
	}
	return false, fmt.Sprintf("float difference %v exceeds tolerance %v", diff, s.options.FloatTolerance)
}

// toFloat64 converts a value to float64.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

// NumericRangeScorer checks if a numeric value is within bounds.
type NumericRangeScorer struct {
	name      string
	outputKey string
	min       float64
	max       float64
	minInclusive bool
	maxInclusive bool
}

// NewNumericRange creates a NumericRangeScorer with inclusive bounds.
func NewNumericRange(outputKey string, min, max float64) *NumericRangeScorer {
	return &NumericRangeScorer{
		name:         "NumericRange",
		outputKey:    outputKey,
		min:          min,
		max:          max,
		minInclusive: true,
		maxInclusive: true,
	}
}

// WithExclusiveBounds sets bounds to be exclusive.
func (s *NumericRangeScorer) WithExclusiveBounds() *NumericRangeScorer {
	s.minInclusive = false
	s.maxInclusive = false
	return s
}

// WithMinInclusive sets whether the minimum bound is inclusive.
func (s *NumericRangeScorer) WithMinInclusive(inclusive bool) *NumericRangeScorer {
	s.minInclusive = inclusive
	return s
}

// WithMaxInclusive sets whether the maximum bound is inclusive.
func (s *NumericRangeScorer) WithMaxInclusive(inclusive bool) *NumericRangeScorer {
	s.maxInclusive = inclusive
	return s
}

// WithName sets a custom name for the scorer.
func (s *NumericRangeScorer) WithName(name string) *NumericRangeScorer {
	s.name = name
	return s
}

// Name returns the scorer's identifier.
func (s *NumericRangeScorer) Name() string {
	return s.name
}

// Score evaluates whether the output is within the numeric range.
func (s *NumericRangeScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract output
	output, ok := input.Outputs[s.outputKey]
	if !ok {
		return Score{
			Error:     fmt.Errorf("output key %q not found", s.outputKey),
			Rationale: fmt.Sprintf("Missing output key: %s", s.outputKey),
		}, nil
	}

	// Convert to float64
	value, ok := toFloat64(output)
	if !ok {
		return Score{
			Error:     fmt.Errorf("output %v is not numeric", output),
			Rationale: fmt.Sprintf("Output is not a numeric value: %v", output),
		}, nil
	}

	// Check bounds
	var inRange bool
	if s.minInclusive {
		if value < s.min {
			inRange = false
		} else if s.maxInclusive {
			inRange = value <= s.max
		} else {
			inRange = value < s.max
		}
	} else {
		if value <= s.min {
			inRange = false
		} else if s.maxInclusive {
			inRange = value <= s.max
		} else {
			inRange = value < s.max
		}
	}

	minBracket := "["
	if !s.minInclusive {
		minBracket = "("
	}
	maxBracket := "]"
	if !s.maxInclusive {
		maxBracket = ")"
	}

	rangeStr := fmt.Sprintf("%s%v, %v%s", minBracket, s.min, s.max, maxBracket)
	rationale := fmt.Sprintf("Value %v is within range %s", value, rangeStr)
	if !inRange {
		rationale = fmt.Sprintf("Value %v is outside range %s", value, rangeStr)
	}

	return Score{
		Value:     inRange,
		Rationale: rationale,
		Metadata: map[string]any{
			"value": value,
			"min":   s.min,
			"max":   s.max,
			"range": rangeStr,
		},
	}, nil
}
