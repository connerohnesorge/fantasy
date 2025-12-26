package eval

import (
	"context"
	"fmt"
	"regexp"
)

// ToolCallTrajectoryOptions configures tool trajectory validation.
type ToolCallTrajectoryOptions struct {
	// Strict: if true, requires exact match (no extra tools allowed)
	// Default: false (extra tools are allowed as long as expected sequence is present)
	Strict bool

	// IgnoreOrder: if true, checks that all expected tools were called (in any order)
	// Default: false (order matters)
	IgnoreOrder bool
}

// ToolCallTrajectoryScorer validates that tool calls match an expected sequence.
type ToolCallTrajectoryScorer struct {
	name        string
	expectedKey string
	options     ToolCallTrajectoryOptions
}

// NewToolCallTrajectory creates a ToolCallTrajectoryScorer.
// expectedKey should point to a []string in expectations containing tool names.
func NewToolCallTrajectory(expectedKey string) *ToolCallTrajectoryScorer {
	return &ToolCallTrajectoryScorer{
		name:        "ToolCallTrajectory",
		expectedKey: expectedKey,
		options:     ToolCallTrajectoryOptions{},
	}
}

// WithOptions sets trajectory validation options.
func (s *ToolCallTrajectoryScorer) WithOptions(opts ToolCallTrajectoryOptions) *ToolCallTrajectoryScorer {
	s.options = opts
	return s
}

// WithName sets a custom name for the scorer.
func (s *ToolCallTrajectoryScorer) WithName(name string) *ToolCallTrajectoryScorer {
	s.name = name
	return s
}

// Name returns the scorer's identifier.
func (s *ToolCallTrajectoryScorer) Name() string {
	return s.name
}

// Score evaluates whether the tool call sequence matches expectations.
func (s *ToolCallTrajectoryScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Check for trace
	if input.Trace == nil {
		return Score{
			Error:     fmt.Errorf("trace is required for ToolCallTrajectory scorer"),
			Rationale: "No trace available - ToolCallTrajectory requires tracing to be enabled",
		}, nil
	}

	// Get expected sequence
	expectedRaw, ok := input.Expectations[s.expectedKey]
	if !ok {
		return Score{
			Error:     fmt.Errorf("expectation key %q not found", s.expectedKey),
			Rationale: fmt.Sprintf("Missing expectation key: %s", s.expectedKey),
		}, nil
	}

	// Convert to string slice
	expectedSequence, err := toStringSlice(expectedRaw)
	if err != nil {
		return Score{
			Error:     fmt.Errorf("expectation %q must be []string: %w", s.expectedKey, err),
			Rationale: fmt.Sprintf("Invalid expectation format: %v", err),
		}, nil
	}

	// Extract tool calls from trace
	// NOTE: This is a placeholder implementation since the tracing package doesn't exist yet.
	// When the tracing package is implemented, this should extract tool spans by SpanType == "TOOL"
	// and return tool names in chronological order.
	actualSequence := s.extractToolCalls(input.Trace)

	// Compare sequences
	matches, rationale := s.compareSequences(actualSequence, expectedSequence)

	metadata := map[string]any{
		"expected": expectedSequence,
		"actual":   actualSequence,
	}

	return Score{
		Value:     matches,
		Rationale: rationale,
		Metadata:  metadata,
	}, nil
}

// extractToolCalls extracts tool names from the trace in chronological order.
// TODO: Replace with actual implementation when tracing package is available.
func (s *ToolCallTrajectoryScorer) extractToolCalls(trace *Trace) []string {
	// Placeholder implementation - will be replaced when tracing package exists
	// This should iterate through trace spans, filter by SpanType == "TOOL",
	// sort by start time, and extract tool names.
	return []string{}
}

// compareSequences compares actual and expected tool sequences.
func (s *ToolCallTrajectoryScorer) compareSequences(actual, expected []string) (bool, string) {
	if s.options.IgnoreOrder {
		// Check that all expected tools are present
		expectedSet := make(map[string]int)
		for _, tool := range expected {
			expectedSet[tool]++
		}

		actualSet := make(map[string]int)
		for _, tool := range actual {
			actualSet[tool]++
		}

		for tool, count := range expectedSet {
			actualCount := actualSet[tool]
			if actualCount < count {
				return false, fmt.Sprintf("Missing expected tool %q (expected %d, got %d)", tool, count, actualCount)
			}
		}

		if s.options.Strict {
			// No extra tools allowed
			for tool, count := range actualSet {
				expectedCount := expectedSet[tool]
				if count > expectedCount {
					return false, fmt.Sprintf("Unexpected tool %q (expected %d, got %d)", tool, expectedCount, count)
				}
			}
		}

		return true, fmt.Sprintf("All expected tools called: %v", expected)
	} else {
		// Order matters - find expected sequence in actual
		if len(actual) == 0 && len(expected) == 0 {
			return true, "No tools called (as expected)"
		}

		if len(expected) == 0 {
			if s.options.Strict {
				return false, fmt.Sprintf("Expected no tools but got: %v", actual)
			}
			return true, "No specific tool sequence expected"
		}

		if len(actual) == 0 {
			return false, fmt.Sprintf("No tools called but expected: %v", expected)
		}

		// Find expected sequence in actual
		matchIdx := -1
		for i := 0; i <= len(actual)-len(expected); i++ {
			match := true
			for j := 0; j < len(expected); j++ {
				if actual[i+j] != expected[j] {
					match = false
					break
				}
			}
			if match {
				matchIdx = i
				break
			}
		}

		if matchIdx == -1 {
			return false, fmt.Sprintf("Expected sequence %v not found in actual %v", expected, actual)
		}

		if s.options.Strict {
			// Exact match required
			if len(actual) != len(expected) {
				return false, fmt.Sprintf("Sequence matches but extra tools present. Expected %v, got %v", expected, actual)
			}
		}

		return true, fmt.Sprintf("Tool sequence matches: %v", expected)
	}
}

// StepValidationOptions configures step validation behavior.
type StepValidationOptions struct {
	// MinSteps: minimum number of steps required (default: 0, no minimum)
	MinSteps int

	// MaxSteps: maximum number of steps allowed (default: 0, no maximum)
	MaxSteps int

	// ContentPatterns: regex patterns that must match step outputs
	// Key is step index (0-based), value is regex pattern
	// If step index doesn't exist, validation fails
	ContentPatterns map[int]string
}

// StepValidationScorer validates agent step count and content.
type StepValidationScorer struct {
	name         string
	options      StepValidationOptions
	extractSteps func(trace *Trace) []string
}

// NewStepValidation creates a StepValidationScorer.
func NewStepValidation() *StepValidationScorer {
	scorer := &StepValidationScorer{
		name:    "StepValidation",
		options: StepValidationOptions{},
	}
	// Set default extractSteps implementation
	scorer.extractSteps = scorer.defaultExtractSteps
	return scorer
}

// WithOptions sets step validation options.
func (s *StepValidationScorer) WithOptions(opts StepValidationOptions) *StepValidationScorer {
	s.options = opts
	return s
}

// WithName sets a custom name for the scorer.
func (s *StepValidationScorer) WithName(name string) *StepValidationScorer {
	s.name = name
	return s
}

// Name returns the scorer's identifier.
func (s *StepValidationScorer) Name() string {
	return s.name
}

// Score evaluates whether the agent's steps meet validation criteria.
func (s *StepValidationScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Check for trace
	if input.Trace == nil {
		return Score{
			Error:     fmt.Errorf("trace is required for StepValidation scorer"),
			Rationale: "No trace available - StepValidation requires tracing to be enabled",
		}, nil
	}

	// Extract steps from trace
	// NOTE: This is a placeholder implementation since the tracing package doesn't exist yet.
	// When the tracing package is implemented, this should extract step spans by SpanType == "CHAIN"
	steps := s.extractSteps(input.Trace)
	stepCount := len(steps)

	// Validate step count
	if s.options.MinSteps > 0 && stepCount < s.options.MinSteps {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("Step count %d is below minimum %d", stepCount, s.options.MinSteps),
			Metadata: map[string]any{
				"step_count": stepCount,
				"min_steps":  s.options.MinSteps,
			},
		}, nil
	}

	if s.options.MaxSteps > 0 && stepCount > s.options.MaxSteps {
		return Score{
			Value:     false,
			Rationale: fmt.Sprintf("Step count %d exceeds maximum %d", stepCount, s.options.MaxSteps),
			Metadata: map[string]any{
				"step_count": stepCount,
				"max_steps":  s.options.MaxSteps,
			},
		}, nil
	}

	// Validate content patterns
	if len(s.options.ContentPatterns) > 0 {
		for stepIdx, patternStr := range s.options.ContentPatterns {
			if stepIdx >= len(steps) {
				return Score{
					Value:     false,
					Rationale: fmt.Sprintf("Step %d does not exist (only %d steps)", stepIdx, len(steps)),
					Metadata: map[string]any{
						"step_count": stepCount,
					},
				}, nil
			}

			pattern, err := regexp.Compile(patternStr)
			if err != nil {
				return Score{
					Error:     fmt.Errorf("invalid regex pattern for step %d: %w", stepIdx, err),
					Rationale: fmt.Sprintf("Failed to compile pattern: %v", err),
				}, nil
			}

			stepContent := steps[stepIdx]
			if !pattern.MatchString(stepContent) {
				return Score{
					Value:     false,
					Rationale: fmt.Sprintf("Step %d content does not match pattern %q", stepIdx, patternStr),
					Metadata: map[string]any{
						"step_index":   stepIdx,
						"step_content": stepContent,
						"pattern":      patternStr,
					},
				}, nil
			}
		}
	}

	rationale := fmt.Sprintf("Step validation passed: %d steps", stepCount)
	metadata := map[string]any{
		"step_count": stepCount,
	}

	return Score{
		Value:     true,
		Rationale: rationale,
		Metadata:  metadata,
	}, nil
}

// defaultExtractSteps extracts step content from the trace.
// TODO: Replace with actual implementation when tracing package is available.
func (s *StepValidationScorer) defaultExtractSteps(trace *Trace) []string {
	// Placeholder implementation - will be replaced when tracing package exists
	// This should iterate through trace spans, filter by SpanType == "CHAIN",
	// sort by start time, and extract step outputs.
	return []string{}
}

// toStringSlice converts any value to a string slice.
func toStringSlice(v any) ([]string, error) {
	// Try direct conversion
	if slice, ok := v.([]string); ok {
		return slice, nil
	}

	// Try []interface{} conversion
	if slice, ok := v.([]interface{}); ok {
		result := make([]string, len(slice))
		for i, item := range slice {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("element %d is not a string", i)
			}
			result[i] = str
		}
		return result, nil
	}

	return nil, fmt.Errorf("cannot convert %T to []string", v)
}
