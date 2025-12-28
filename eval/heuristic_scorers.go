package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ExactMatchScorer compares output to expected string with exact equality.
type ExactMatchScorer struct {
	name string
}

// NewExactMatch creates an ExactMatchScorer.
func NewExactMatch() *ExactMatchScorer {
	return &ExactMatchScorer{name: "exact_match"}
}

// Name returns the scorer's identifier.
func (s *ExactMatchScorer) Name() string {
	return s.name
}

// Score evaluates exact string equality.
func (s *ExactMatchScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	output := extractStringOutput(input.Outputs)
	expected := extractStringExpectation(input.Expectations)

	match := output == expected
	score := 0.0
	if match {
		score = 1.0
	}

	return Score{
		Value:     score,
		Rationale: fmt.Sprintf("Exact match: %v (output: %q, expected: %q)", match, output, expected),
		Metadata: map[string]any{
			"output":   output,
			"expected": expected,
		},
	}, nil
}

// ContainsScorer checks if output contains expected substring.
type ContainsScorer struct {
	name            string
	caseInsensitive bool
}

// ContainsOption configures a ContainsScorer.
type ContainsOption func(*ContainsScorer)

// WithCaseInsensitive makes the contains check case-insensitive.
func WithCaseInsensitive(v bool) ContainsOption {
	return func(s *ContainsScorer) {
		s.caseInsensitive = v
	}
}

// NewContains creates a ContainsScorer.
func NewContains(opts ...ContainsOption) *ContainsScorer {
	s := &ContainsScorer{name: "contains"}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the scorer's identifier.
func (s *ContainsScorer) Name() string {
	return s.name
}

// Score evaluates substring containment.
func (s *ContainsScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	output := extractStringOutput(input.Outputs)
	expected := extractStringExpectation(input.Expectations)

	var match bool
	if s.caseInsensitive {
		match = strings.Contains(strings.ToLower(output), strings.ToLower(expected))
	} else {
		match = strings.Contains(output, expected)
	}

	score := 0.0
	if match {
		score = 1.0
	}

	return Score{
		Value:     score,
		Rationale: fmt.Sprintf("Contains match: %v (output: %q, expected substring: %q, case_insensitive: %v)", match, output, expected, s.caseInsensitive),
		Metadata: map[string]any{
			"output":           output,
			"expected":         expected,
			"case_insensitive": s.caseInsensitive,
		},
	}, nil
}

// RegexScorer checks if output matches a regex pattern.
type RegexScorer struct {
	name    string
	pattern *regexp.Regexp
}

// NewRegex creates a RegexScorer with a compiled pattern.
func NewRegex(pattern string) (*RegexScorer, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	return &RegexScorer{
		name:    "regex",
		pattern: re,
	}, nil
}

// Name returns the scorer's identifier.
func (s *RegexScorer) Name() string {
	return s.name
}

// Score evaluates regex pattern matching.
func (s *RegexScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	output := extractStringOutput(input.Outputs)

	match := s.pattern.MatchString(output)
	score := 0.0
	if match {
		score = 1.0
	}

	return Score{
		Value:     score,
		Rationale: fmt.Sprintf("Regex match: %v (output: %q, pattern: %q)", match, output, s.pattern.String()),
		Metadata: map[string]any{
			"output":  output,
			"pattern": s.pattern.String(),
		},
	}, nil
}

// JSONMatchOptions configures JSON comparison behavior.
type JSONMatchOptions struct {
	IgnoreOrder       bool // Ignore array element order
	IgnoreExtraFields bool // Allow extra fields in output not in expected
	IgnoreTypes       bool // Compare values loosely (e.g., 1 == "1")
}

// JSONMatchScorer compares JSON structures.
type JSONMatchScorer struct {
	name    string
	options JSONMatchOptions
}

// JSONMatchOption configures a JSONMatchScorer.
type JSONMatchOption func(*JSONMatchScorer)

// WithIgnoreOrder ignores array element order.
func WithIgnoreOrder(v bool) JSONMatchOption {
	return func(s *JSONMatchScorer) {
		s.options.IgnoreOrder = v
	}
}

// WithIgnoreExtraFields allows extra fields in output.
func WithIgnoreExtraFields(v bool) JSONMatchOption {
	return func(s *JSONMatchScorer) {
		s.options.IgnoreExtraFields = v
	}
}

// WithIgnoreTypes compares values loosely.
func WithIgnoreTypes(v bool) JSONMatchOption {
	return func(s *JSONMatchScorer) {
		s.options.IgnoreTypes = v
	}
}

// NewJSONMatch creates a JSONMatchScorer.
func NewJSONMatch(opts ...JSONMatchOption) *JSONMatchScorer {
	s := &JSONMatchScorer{name: "json_match"}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the scorer's identifier.
func (s *JSONMatchScorer) Name() string {
	return s.name
}

// Score evaluates JSON structural comparison.
func (s *JSONMatchScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract output
	outputRaw := input.Outputs["output"]
	if outputRaw == nil {
		outputRaw = input.Outputs["response"]
	}

	// Extract expected
	expectedRaw := input.Expectations["output"]
	if expectedRaw == nil {
		expectedRaw = input.Expectations["expected"]
	}

	// Parse JSON if strings
	var outputJSON, expectedJSON any
	var err error

	outputJSON, err = parseJSONValue(outputRaw)
	if err != nil {
		return Score{}, fmt.Errorf("failed to parse output JSON: %w", err)
	}

	expectedJSON, err = parseJSONValue(expectedRaw)
	if err != nil {
		return Score{}, fmt.Errorf("failed to parse expected JSON: %w", err)
	}

	// Compare
	match := s.compareJSON(outputJSON, expectedJSON)
	score := 0.0
	if match {
		score = 1.0
	}

	return Score{
		Value:     score,
		Rationale: fmt.Sprintf("JSON match: %v", match),
		Metadata: map[string]any{
			"output":   outputJSON,
			"expected": expectedJSON,
			"options":  s.options,
		},
	}, nil
}

// parseJSONValue parses a value as JSON if it's a string, otherwise returns as-is.
func parseJSONValue(v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	// If it's already a structured type, return it
	switch v.(type) {
	case map[string]any, []any:
		return v, nil
	}

	// If it's a string, try to parse it as JSON
	if str, ok := v.(string); ok {
		var result any
		if err := json.Unmarshal([]byte(str), &result); err != nil {
			// Not valid JSON, return the string itself
			return str, nil
		}
		return result, nil
	}

	// For other types, return as-is
	return v, nil
}

// compareJSON compares two JSON values according to options.
func (s *JSONMatchScorer) compareJSON(output, expected any) bool {
	// Handle nil cases
	if output == nil && expected == nil {
		return true
	}
	if output == nil || expected == nil {
		return false
	}

	// Type conversion if IgnoreTypes is enabled
	if s.options.IgnoreTypes {
		output = normalizeType(output)
		expected = normalizeType(expected)
	}

	// Handle maps
	if outMap, outOk := output.(map[string]any); outOk {
		expMap, expOk := expected.(map[string]any)
		if !expOk {
			return false
		}
		return s.compareMaps(outMap, expMap)
	}

	// Handle slices
	if outSlice, outOk := output.([]any); outOk {
		expSlice, expOk := expected.([]any)
		if !expOk {
			return false
		}
		return s.compareSlices(outSlice, expSlice)
	}

	// Handle interface{} slices (from JSON unmarshaling)
	if outSlice, outOk := output.([]interface{}); outOk {
		expSlice, expOk := expected.([]interface{})
		if !expOk {
			return false
		}
		return s.compareSlices(outSlice, expSlice)
	}

	// Direct comparison for primitives
	return reflect.DeepEqual(output, expected)
}

// compareMaps compares two maps according to options.
func (s *JSONMatchScorer) compareMaps(output, expected map[string]any) bool {
	// Check all expected keys exist in output
	for key, expVal := range expected {
		outVal, exists := output[key]
		if !exists {
			return false
		}
		if !s.compareJSON(outVal, expVal) {
			return false
		}
	}

	// If not ignoring extra fields, check output has no extra keys
	if !s.options.IgnoreExtraFields {
		if len(output) != len(expected) {
			return false
		}
	}

	return true
}

// compareSlices compares two slices according to options.
func (s *JSONMatchScorer) compareSlices(output, expected []any) bool {
	if len(output) != len(expected) {
		return false
	}

	if s.options.IgnoreOrder {
		// Create a copy of output to mark matched elements
		matched := make([]bool, len(output))
		for _, expElem := range expected {
			found := false
			for i, outElem := range output {
				if !matched[i] && s.compareJSON(outElem, expElem) {
					matched[i] = true
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

	// Order matters
	for i := range expected {
		if !s.compareJSON(output[i], expected[i]) {
			return false
		}
	}

	return true
}

// normalizeType converts values to a common type for loose comparison.
func normalizeType(v any) any {
	if v == nil {
		return nil
	}

	// Try to convert to string first
	str := fmt.Sprintf("%v", v)

	// Try to parse as number
	if f, err := strconv.ParseFloat(str, 64); err == nil {
		return f
	}

	// Try to parse as bool
	if b, err := strconv.ParseBool(str); err == nil {
		return b
	}

	// Return as string
	return str
}

// NumericRangeScorer checks if a numeric value is within a range.
type NumericRangeScorer struct {
	name         string
	min          float64
	max          float64
	minInclusive bool
	maxInclusive bool
}

// NumericRangeOption configures a NumericRangeScorer.
type NumericRangeOption func(*NumericRangeScorer)

// WithMinInclusive sets whether the minimum bound is inclusive.
func WithMinInclusive(v bool) NumericRangeOption {
	return func(s *NumericRangeScorer) {
		s.minInclusive = v
	}
}

// WithMaxInclusive sets whether the maximum bound is inclusive.
func WithMaxInclusive(v bool) NumericRangeOption {
	return func(s *NumericRangeScorer) {
		s.maxInclusive = v
	}
}

// NewNumericRange creates a NumericRangeScorer.
// Default: inclusive on both bounds.
func NewNumericRange(min, max float64, opts ...NumericRangeOption) *NumericRangeScorer {
	s := &NumericRangeScorer{
		name:         "numeric_range",
		min:          min,
		max:          max,
		minInclusive: true,
		maxInclusive: true,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the scorer's identifier.
func (s *NumericRangeScorer) Name() string {
	return s.name
}

// Score evaluates if numeric output is within range.
func (s *NumericRangeScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract numeric value
	outputRaw := input.Outputs["output"]
	if outputRaw == nil {
		outputRaw = input.Outputs["response"]
	}

	value, err := extractNumeric(outputRaw)
	if err != nil {
		return Score{}, fmt.Errorf("failed to extract numeric value: %w", err)
	}

	// Check bounds
	var inRange bool
	if s.minInclusive && s.maxInclusive {
		inRange = value >= s.min && value <= s.max
	} else if s.minInclusive && !s.maxInclusive {
		inRange = value >= s.min && value < s.max
	} else if !s.minInclusive && s.maxInclusive {
		inRange = value > s.min && value <= s.max
	} else {
		inRange = value > s.min && value < s.max
	}

	score := 0.0
	if inRange {
		score = 1.0
	}

	minOp := ">"
	if s.minInclusive {
		minOp = ">="
	}
	maxOp := "<"
	if s.maxInclusive {
		maxOp = "<="
	}

	return Score{
		Value:     score,
		Rationale: fmt.Sprintf("Numeric range check: %v (value: %v, range: %v %s x %s %v)", inRange, value, s.min, minOp, maxOp, s.max),
		Metadata: map[string]any{
			"value":         value,
			"min":           s.min,
			"max":           s.max,
			"min_inclusive": s.minInclusive,
			"max_inclusive": s.maxInclusive,
		},
	}, nil
}

// extractNumeric extracts a float64 from various value types.
func extractNumeric(v any) (float64, error) {
	if v == nil {
		return 0, fmt.Errorf("nil value")
	}

	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse string as number: %w", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported type for numeric extraction: %T", v)
	}
}

// ToolCallTrajectoryOptions configures tool call trajectory validation.
type ToolCallTrajectoryOptions struct {
	ExactMatch bool // Require exact match vs. contains
}

// ToolCallTrajectoryScorer validates tool call sequences.
type ToolCallTrajectoryScorer struct {
	name    string
	options ToolCallTrajectoryOptions
}

// ToolCallTrajectoryOption configures a ToolCallTrajectoryScorer.
type ToolCallTrajectoryOption func(*ToolCallTrajectoryScorer)

// WithExactMatch requires exact sequence match.
func WithExactMatch(v bool) ToolCallTrajectoryOption {
	return func(s *ToolCallTrajectoryScorer) {
		s.options.ExactMatch = v
	}
}

// NewToolCallTrajectory creates a ToolCallTrajectoryScorer.
func NewToolCallTrajectory(opts ...ToolCallTrajectoryOption) *ToolCallTrajectoryScorer {
	s := &ToolCallTrajectoryScorer{name: "tool_call_trajectory"}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the scorer's identifier.
func (s *ToolCallTrajectoryScorer) Name() string {
	return s.name
}

// Score evaluates tool call trajectory.
func (s *ToolCallTrajectoryScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract tool calls from outputs
	toolCallsRaw := input.Outputs["tool_calls"]
	if toolCallsRaw == nil {
		return Score{}, fmt.Errorf("no tool_calls in outputs")
	}

	// Extract expected trajectory
	expectedRaw := input.Expectations["tool_calls"]
	if expectedRaw == nil {
		expectedRaw = input.Expectations["expected"]
	}
	if expectedRaw == nil {
		return Score{}, fmt.Errorf("no expected tool_calls in expectations")
	}

	// Convert to string slices (tool names)
	actualCalls := extractToolNames(toolCallsRaw)
	expectedCalls := extractToolNames(expectedRaw)

	var match bool
	if s.options.ExactMatch {
		match = reflect.DeepEqual(actualCalls, expectedCalls)
	} else {
		// Contains: check if expected is a subsequence of actual
		match = isSubsequence(expectedCalls, actualCalls)
	}

	score := 0.0
	if match {
		score = 1.0
	}

	matchType := "contains"
	if s.options.ExactMatch {
		matchType = "exact"
	}

	return Score{
		Value:     score,
		Rationale: fmt.Sprintf("Tool call trajectory %s match: %v (actual: %v, expected: %v)", matchType, match, actualCalls, expectedCalls),
		Metadata: map[string]any{
			"actual":      actualCalls,
			"expected":    expectedCalls,
			"exact_match": s.options.ExactMatch,
		},
	}, nil
}

// extractToolNames extracts tool names from various formats.
func extractToolNames(v any) []string {
	if v == nil {
		return nil
	}

	// Handle string slice directly
	if strSlice, ok := v.([]string); ok {
		return strSlice
	}

	// Handle []any with strings
	if anySlice, ok := v.([]any); ok {
		var result []string
		for _, item := range anySlice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			} else if m, ok := item.(map[string]any); ok {
				// Handle map with "name" field
				if name, ok := m["name"].(string); ok {
					result = append(result, name)
				}
			}
		}
		return result
	}

	// Handle []interface{} (from JSON unmarshaling)
	if ifaceSlice, ok := v.([]any); ok {
		var result []string
		for _, item := range ifaceSlice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			} else if m, ok := item.(map[string]any); ok {
				// Handle map with "name" field
				if name, ok := m["name"].(string); ok {
					result = append(result, name)
				}
			}
		}
		return result
	}

	return nil
}

// isSubsequence checks if expected is a subsequence of actual.
func isSubsequence(expected, actual []string) bool {
	if len(expected) == 0 {
		return true
	}
	if len(actual) == 0 {
		return false
	}

	expIdx := 0
	for _, actualItem := range actual {
		if actualItem == expected[expIdx] {
			expIdx++
			if expIdx == len(expected) {
				return true
			}
		}
	}

	return expIdx == len(expected)
}

// StepValidationOptions configures step validation.
type StepValidationOptions struct {
	MinSteps     int      // Minimum number of steps
	MaxSteps     int      // Maximum number of steps (0 = no limit)
	StepPatterns []string // Regex patterns that steps should match
	AllMatch     bool     // All steps must match patterns vs. at least one
}

// StepValidationScorer validates step count and content.
type StepValidationScorer struct {
	name     string
	options  StepValidationOptions
	patterns []*regexp.Regexp
}

// StepValidationOption configures a StepValidationScorer.
type StepValidationOption func(*StepValidationScorer)

// WithMinSteps sets minimum step count.
func WithMinSteps(n int) StepValidationOption {
	return func(s *StepValidationScorer) {
		s.options.MinSteps = n
	}
}

// WithMaxSteps sets maximum step count.
func WithMaxSteps(n int) StepValidationOption {
	return func(s *StepValidationScorer) {
		s.options.MaxSteps = n
	}
}

// WithStepPatterns sets regex patterns for step content.
func WithStepPatterns(patterns []string) StepValidationOption {
	return func(s *StepValidationScorer) {
		s.options.StepPatterns = patterns
	}
}

// WithAllMatch requires all steps to match patterns.
func WithAllMatch(v bool) StepValidationOption {
	return func(s *StepValidationScorer) {
		s.options.AllMatch = v
	}
}

// NewStepValidation creates a StepValidationScorer.
func NewStepValidation(opts ...StepValidationOption) (*StepValidationScorer, error) {
	s := &StepValidationScorer{name: "step_validation"}
	for _, opt := range opts {
		opt(s)
	}

	// Compile patterns
	for _, pattern := range s.options.StepPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid step pattern %q: %w", pattern, err)
		}
		s.patterns = append(s.patterns, re)
	}

	return s, nil
}

// Name returns the scorer's identifier.
func (s *StepValidationScorer) Name() string {
	return s.name
}

// Score evaluates step validation.
func (s *StepValidationScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract steps
	stepsRaw := input.Outputs["steps"]
	if stepsRaw == nil {
		return Score{}, fmt.Errorf("no steps in outputs")
	}

	steps := extractSteps(stepsRaw)
	stepCount := len(steps)

	// Validate count
	countValid := true
	var countReason string

	if s.options.MinSteps > 0 && stepCount < s.options.MinSteps {
		countValid = false
		countReason = fmt.Sprintf("too few steps: %d < %d", stepCount, s.options.MinSteps)
	}

	if s.options.MaxSteps > 0 && stepCount > s.options.MaxSteps {
		countValid = false
		countReason = fmt.Sprintf("too many steps: %d > %d", stepCount, s.options.MaxSteps)
	}

	// Validate patterns
	patternValid := true
	var patternReason string

	if len(s.patterns) > 0 {
		if s.options.AllMatch {
			// All steps must match at least one pattern
			for i, step := range steps {
				matched := false
				for _, pattern := range s.patterns {
					if pattern.MatchString(step) {
						matched = true
						break
					}
				}
				if !matched {
					patternValid = false
					patternReason = fmt.Sprintf("step %d does not match any pattern: %q", i, step)
					break
				}
			}
		} else {
			// At least one step must match
			matched := false
			for _, step := range steps {
				for _, pattern := range s.patterns {
					if pattern.MatchString(step) {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				patternValid = false
				patternReason = "no steps match any pattern"
			}
		}
	}

	// Overall validity
	valid := countValid && patternValid
	score := 0.0
	if valid {
		score = 1.0
	}

	rationale := fmt.Sprintf("Step validation: %v (count: %d)", valid, stepCount)
	if !countValid {
		rationale += fmt.Sprintf(" [%s]", countReason)
	}
	if !patternValid {
		rationale += fmt.Sprintf(" [%s]", patternReason)
	}

	return Score{
		Value:     score,
		Rationale: rationale,
		Metadata: map[string]any{
			"step_count":    stepCount,
			"count_valid":   countValid,
			"pattern_valid": patternValid,
			"steps":         steps,
		},
	}, nil
}

// extractSteps extracts step strings from various formats.
func extractSteps(v any) []string {
	if v == nil {
		return nil
	}

	// Handle string slice directly
	if strSlice, ok := v.([]string); ok {
		return strSlice
	}

	// Handle []any
	if anySlice, ok := v.([]any); ok {
		var result []string
		for _, item := range anySlice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			} else {
				// Convert to string
				result = append(result, fmt.Sprintf("%v", item))
			}
		}
		return result
	}

	// Handle []interface{} (from JSON unmarshaling)
	if ifaceSlice, ok := v.([]interface{}); ok {
		var result []string
		for _, item := range ifaceSlice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			} else {
				// Convert to string
				result = append(result, fmt.Sprintf("%v", item))
			}
		}
		return result
	}

	return nil
}

// Helper functions for extracting common fields

// extractStringOutput extracts string output from Outputs map.
func extractStringOutput(outputs map[string]any) string {
	if v := outputs["output"]; v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	if v := outputs["response"]; v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// extractStringExpectation extracts expected string from Expectations map.
func extractStringExpectation(expectations map[string]any) string {
	if v := expectations["output"]; v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	if v := expectations["expected"]; v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}
