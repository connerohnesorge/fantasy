package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// ExactMatchScorer returns a scorer that checks for exact string match.
func ExactMatchScorer(field string) Scorer {
	return &exactMatchScorer{field: field}
}

type exactMatchScorer struct {
	field string
}

func (s *exactMatchScorer) Name() string {
	return "exact_match"
}

func (s *exactMatchScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	expected, ok := input.Expectations[s.field]
	if !ok {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("expected field %q not found", s.field),
		}, nil
	}

	actual, ok := input.Outputs[s.field]
	if !ok {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("output field %q not found", s.field),
		}, nil
	}

	expectedStr := fmt.Sprintf("%v", expected)
	actualStr := fmt.Sprintf("%v", actual)

	if expectedStr == actualStr {
		return Score{
			Value:     true,
			Rationale: "exact match",
		}, nil
	}

	return Score{
		Value:     false,
		Rationale: fmt.Sprintf("expected %q, got %q", expectedStr, actualStr),
	}, nil
}

// ContainsScorer returns a scorer that checks if output contains expected substring.
func ContainsScorer(field string) Scorer {
	return &containsScorer{field: field}
}

type containsScorer struct {
	field string
}

func (s *containsScorer) Name() string {
	return "contains"
}

func (s *containsScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	expected, ok := input.Expectations[s.field]
	if !ok {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("expected field %q not found", s.field),
		}, nil
	}

	actual, ok := input.Outputs[s.field]
	if !ok {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("output field %q not found", s.field),
		}, nil
	}

	expectedStr := fmt.Sprintf("%v", expected)
	actualStr := fmt.Sprintf("%v", actual)

	if strings.Contains(actualStr, expectedStr) {
		return Score{
			Value:     true,
			Rationale: fmt.Sprintf("output contains %q", expectedStr),
		}, nil
	}

	return Score{
		Value:     false,
		Rationale: fmt.Sprintf("output does not contain %q", expectedStr),
	}, nil
}

// RegexScorer returns a scorer that checks if output matches a regex pattern.
func RegexScorer(field string) Scorer {
	return &regexScorer{field: field}
}

type regexScorer struct {
	field string
}

func (s *regexScorer) Name() string {
	return "regex"
}

func (s *regexScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	pattern, ok := input.Expectations[s.field]
	if !ok {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("expected field %q not found", s.field),
		}, nil
	}

	actual, ok := input.Outputs[s.field]
	if !ok {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("output field %q not found", s.field),
		}, nil
	}

	patternStr := fmt.Sprintf("%v", pattern)
	actualStr := fmt.Sprintf("%v", actual)

	re, err := regexp.Compile(patternStr)
	if err != nil {
		return Score{}, fmt.Errorf("invalid regex pattern: %w", err)
	}

	if re.MatchString(actualStr) {
		return Score{
			Value:     true,
			Rationale: fmt.Sprintf("output matches pattern %q", patternStr),
		}, nil
	}

	return Score{
		Value:     false,
		Rationale: fmt.Sprintf("output does not match pattern %q", patternStr),
	}, nil
}

// JSONMatchOptions configures JSONMatch scorer behavior.
type JSONMatchOptions struct {
	NullEqualsMissing bool    // If true, null values are considered equal to missing keys
	IgnoreArrayOrder  bool    // If true, array elements are compared as sets
	FloatTolerance    float64 // Tolerance for float comparisons
}

// JSONMatchScorer returns a scorer that checks for semantic JSON equivalence.
func JSONMatchScorer(field string, opts ...JSONMatchOptions) Scorer {
	var options JSONMatchOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	return &jsonMatchScorer{field: field, opts: options}
}

type jsonMatchScorer struct {
	field string
	opts  JSONMatchOptions
}

func (s *jsonMatchScorer) Name() string {
	return "json_match"
}

func (s *jsonMatchScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	expected, ok := input.Expectations[s.field]
	if !ok {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("expected field %q not found", s.field),
		}, nil
	}

	actual, ok := input.Outputs[s.field]
	if !ok {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("output field %q not found", s.field),
		}, nil
	}

	// Normalize both values to JSON
	expectedJSON, err := s.normalizeJSON(expected)
	if err != nil {
		return Score{}, fmt.Errorf("failed to normalize expected value: %w", err)
	}

	actualJSON, err := s.normalizeJSON(actual)
	if err != nil {
		return Score{}, fmt.Errorf("failed to normalize actual value: %w", err)
	}

	if s.jsonEqual(expectedJSON, actualJSON) {
		return Score{
			Value:     true,
			Rationale: "JSON structures are equivalent",
		}, nil
	}

	return Score{
		Value:     false,
		Rationale: "JSON structures differ",
	}, nil
}

func (s *jsonMatchScorer) normalizeJSON(v any) (any, error) {
	// If already a map or slice, use directly
	switch v := v.(type) {
	case map[string]any, []any:
		return v, nil
	case string:
		// Try to parse as JSON
		var parsed any
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			// Not valid JSON, return as string
			return v, nil
		}
		return parsed, nil
	default:
		// Convert to JSON and back to normalize
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var parsed any
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	}
}

func (s *jsonMatchScorer) jsonEqual(a, b any) bool {
	// Handle nil
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		if s.opts.NullEqualsMissing {
			return true
		}
		return false
	}

	// Compare based on type
	switch aVal := a.(type) {
	case map[string]any:
		bVal, ok := b.(map[string]any)
		if !ok {
			return false
		}
		return s.mapsEqual(aVal, bVal)

	case []any:
		bVal, ok := b.([]any)
		if !ok {
			return false
		}
		return s.slicesEqual(aVal, bVal)

	case float64:
		bVal, ok := b.(float64)
		if !ok {
			return false
		}
		if s.opts.FloatTolerance > 0 {
			diff := aVal - bVal
			if diff < 0 {
				diff = -diff
			}
			return diff <= s.opts.FloatTolerance
		}
		return aVal == bVal

	default:
		return reflect.DeepEqual(a, b)
	}
}

func (s *jsonMatchScorer) mapsEqual(a, b map[string]any) bool {
	// Check all keys in a
	for k, aVal := range a {
		bVal, ok := b[k]
		if !ok {
			if s.opts.NullEqualsMissing && aVal == nil {
				continue
			}
			return false
		}
		if !s.jsonEqual(aVal, bVal) {
			return false
		}
	}

	// Check for extra keys in b
	for k, bVal := range b {
		if _, ok := a[k]; !ok {
			if s.opts.NullEqualsMissing && bVal == nil {
				continue
			}
			return false
		}
	}

	return true
}

func (s *jsonMatchScorer) slicesEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}

	if s.opts.IgnoreArrayOrder {
		// Compare as sets - O(n^2) but simple
		used := make([]bool, len(b))
		for _, aVal := range a {
			found := false
			for j, bVal := range b {
				if !used[j] && s.jsonEqual(aVal, bVal) {
					used[j] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	// Compare in order
	for i := range a {
		if !s.jsonEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// NumericRangeScorer returns a scorer that checks if a numeric value is within a range.
func NumericRangeScorer(field string, min, max float64) Scorer {
	return &numericRangeScorer{field: field, min: min, max: max}
}

type numericRangeScorer struct {
	field string
	min   float64
	max   float64
}

func (s *numericRangeScorer) Name() string {
	return "numeric_range"
}

func (s *numericRangeScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	actual, ok := input.Outputs[s.field]
	if !ok {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("output field %q not found", s.field),
		}, nil
	}

	var value float64
	switch v := actual.(type) {
	case float64:
		value = v
	case int:
		value = float64(v)
	case int64:
		value = float64(v)
	case json.Number:
		var err error
		value, err = v.Float64()
		if err != nil {
			return Score{}, fmt.Errorf("invalid numeric value: %w", err)
		}
	default:
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("field %q is not numeric", s.field),
		}, nil
	}

	if value >= s.min && value <= s.max {
		return Score{
			Value:     true,
			Rationale: fmt.Sprintf("%.2f is within range [%.2f, %.2f]", value, s.min, s.max),
		}, nil
	}

	return Score{
		Value:     false,
		Rationale: fmt.Sprintf("%.2f is outside range [%.2f, %.2f]", value, s.min, s.max),
	}, nil
}

// ToolCallTrajectoryOptions configures ToolCallTrajectory scorer behavior.
type ToolCallTrajectoryOptions struct {
	Strict      bool // If true, requires exact match (no extra tools allowed)
	IgnoreOrder bool // If true, checks that all expected tools were called (in any order)
}

// ToolCallTrajectoryScorer returns a scorer that checks tool call sequences.
// Requires a non-nil trace in ScorerInput.
func ToolCallTrajectoryScorer(opts ...ToolCallTrajectoryOptions) Scorer {
	var options ToolCallTrajectoryOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	return &toolCallTrajectoryScorer{opts: options}
}

type toolCallTrajectoryScorer struct {
	opts ToolCallTrajectoryOptions
}

func (s *toolCallTrajectoryScorer) Name() string {
	return "tool_call_trajectory"
}

func (s *toolCallTrajectoryScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	if input.Trace == nil {
		return Score{
			Error:     fmt.Errorf("ToolCallTrajectory scorer requires a non-nil trace"),
			Rationale: "no trace available",
		}, nil
	}

	expectedSeq, ok := input.Expectations["tool_sequence"]
	if !ok {
		return Score{
			Value:     false,
			Rationale: "expected tool_sequence not found in expectations",
		}, nil
	}

	expected, ok := expectedSeq.([]string)
	if !ok {
		// Try to convert from []any
		if anySlice, ok := expectedSeq.([]any); ok {
			expected = make([]string, len(anySlice))
			for i, v := range anySlice {
				expected[i] = fmt.Sprintf("%v", v)
			}
		} else {
			return Score{}, fmt.Errorf("tool_sequence must be a string array")
		}
	}

	// Extract tool names from trace
	actual := s.extractToolNames(input.Trace)

	if s.opts.IgnoreOrder {
		// Check that all expected tools were called
		return s.checkAllCalled(expected, actual), nil
	}

	if s.opts.Strict {
		// Check exact sequence match
		return s.checkExactMatch(expected, actual), nil
	}

	// Check subsequence match
	return s.checkSubsequence(expected, actual), nil
}

func (s *toolCallTrajectoryScorer) extractToolNames(trace any) []string {
	// This would need to be adapted based on the actual trace structure
	// For now, return empty slice - actual implementation depends on tracing package
	return []string{}
}

func (s *toolCallTrajectoryScorer) checkExactMatch(expected, actual []string) Score {
	if len(expected) != len(actual) {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("expected %d tools, got %d", len(expected), len(actual)),
		}
	}

	for i := range expected {
		if expected[i] != actual[i] {
			return Score{
				Value:     false,
				Rationale: fmt.Sprintf("at position %d: expected %q, got %q", i, expected[i], actual[i]),
			}
		}
	}

	return Score{
		Value:     true,
		Rationale: "tool sequence matches exactly",
	}
}

func (s *toolCallTrajectoryScorer) checkSubsequence(expected, actual []string) Score {
	j := 0
	for i := 0; i < len(actual) && j < len(expected); i++ {
		if actual[i] == expected[j] {
			j++
		}
	}

	if j == len(expected) {
		return Score{
			Value:     true,
			Rationale: "expected tool sequence found",
		}
	}

	return Score{
		Value:     false,
		Rationale: fmt.Sprintf("expected sequence not found, missing %v", expected[j:]),
	}
}

func (s *toolCallTrajectoryScorer) checkAllCalled(expected, actual []string) Score {
	actualSet := make(map[string]bool)
	for _, t := range actual {
		actualSet[t] = true
	}

	var missing []string
	for _, t := range expected {
		if !actualSet[t] {
			missing = append(missing, t)
		}
	}

	if len(missing) == 0 {
		return Score{
			Value:     true,
			Rationale: "all expected tools were called",
		}
	}

	sort.Strings(missing)
	return Score{
		Value:     false,
		Rationale: fmt.Sprintf("missing tools: %v", missing),
	}
}

// StepValidationOptions configures StepValidation scorer behavior.
type StepValidationOptions struct {
	MinSteps        int            // Minimum number of steps required
	MaxSteps        int            // Maximum number of steps allowed
	ContentPatterns map[int]string // Regex patterns for specific step outputs
}

// StepValidationScorer returns a scorer that validates step counts and content.
// Requires a non-nil trace in ScorerInput.
func StepValidationScorer(opts ...StepValidationOptions) Scorer {
	var options StepValidationOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	return &stepValidationScorer{opts: options}
}

type stepValidationScorer struct {
	opts StepValidationOptions
}

func (s *stepValidationScorer) Name() string {
	return "step_validation"
}

func (s *stepValidationScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	if input.Trace == nil {
		return Score{
			Error:     fmt.Errorf("StepValidation scorer requires a non-nil trace"),
			Rationale: "no trace available",
		}, nil
	}

	// Extract step count from trace
	stepCount := s.extractStepCount(input.Trace)

	// Check min steps
	if s.opts.MinSteps > 0 && stepCount < s.opts.MinSteps {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("too few steps: got %d, expected at least %d", stepCount, s.opts.MinSteps),
		}, nil
	}

	// Check max steps
	if s.opts.MaxSteps > 0 && stepCount > s.opts.MaxSteps {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("too many steps: got %d, expected at most %d", stepCount, s.opts.MaxSteps),
		}, nil
	}

	// Check content patterns
	for stepIdx, pattern := range s.opts.ContentPatterns {
		content := s.extractStepContent(input.Trace, stepIdx)
		if content == "" {
			return Score{
				Value:     false,
				Rationale: fmt.Sprintf("step %d not found", stepIdx),
			}, nil
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return Score{}, fmt.Errorf("invalid pattern for step %d: %w", stepIdx, err)
		}

		if !re.MatchString(content) {
			return Score{
				Value:     false,
				Rationale: fmt.Sprintf("step %d content does not match pattern", stepIdx),
			}, nil
		}
	}

	return Score{
		Value:     true,
		Rationale: fmt.Sprintf("step validation passed (%d steps)", stepCount),
	}, nil
}

func (s *stepValidationScorer) extractStepCount(trace any) int {
	// Actual implementation depends on tracing package
	return 0
}

func (s *stepValidationScorer) extractStepContent(trace any, stepIdx int) string {
	// Actual implementation depends on tracing package
	return ""
}
