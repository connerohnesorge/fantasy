package eval

import (
	"context"
	"testing"
)

func TestToolCallTrajectoryScorer_NoTrace(t *testing.T) {
	scorer := NewToolCallTrajectory("tool_sequence")

	input := ScorerInput{
		Outputs: map[string]any{},
		Expectations: map[string]any{
			"tool_sequence": []string{"tool1", "tool2"},
		},
		Trace: nil,
	}

	score, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if score.Error == nil {
		t.Fatal("expected error for missing trace")
	}
}

func TestToolCallTrajectoryScorer_MissingExpectation(t *testing.T) {
	scorer := NewToolCallTrajectory("tool_sequence")

	input := ScorerInput{
		Outputs:      map[string]any{},
		Expectations: map[string]any{},
		Trace:        &Trace{}, // Non-nil trace
	}

	score, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if score.Error == nil {
		t.Fatal("expected error for missing expectation")
	}
}

func TestToolCallTrajectoryScorer_InvalidExpectationType(t *testing.T) {
	scorer := NewToolCallTrajectory("tool_sequence")

	input := ScorerInput{
		Outputs: map[string]any{},
		Expectations: map[string]any{
			"tool_sequence": "not a slice", // Invalid type
		},
		Trace: &Trace{},
	}

	score, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if score.Error == nil {
		t.Fatal("expected error for invalid expectation type")
	}
}

func TestToolCallTrajectoryScorer_CompareSequences(t *testing.T) {
	tests := []struct {
		name        string
		actual      []string
		expected    []string
		options     ToolCallTrajectoryOptions
		wantMatch   bool
		description string
	}{
		{
			name:        "exact match",
			actual:      []string{"tool1", "tool2", "tool3"},
			expected:    []string{"tool1", "tool2", "tool3"},
			options:     ToolCallTrajectoryOptions{},
			wantMatch:   true,
			description: "sequence matches exactly",
		},
		{
			name:        "sequence with extra tools (non-strict)",
			actual:      []string{"tool1", "tool2", "tool3", "tool4"},
			expected:    []string{"tool1", "tool2", "tool3"},
			options:     ToolCallTrajectoryOptions{Strict: false},
			wantMatch:   true,
			description: "expected sequence is present, extra tools allowed",
		},
		{
			name:        "sequence with extra tools (strict)",
			actual:      []string{"tool1", "tool2", "tool3", "tool4"},
			expected:    []string{"tool1", "tool2", "tool3"},
			options:     ToolCallTrajectoryOptions{Strict: true},
			wantMatch:   false,
			description: "strict mode rejects extra tools",
		},
		{
			name:        "out of order (order matters)",
			actual:      []string{"tool2", "tool1", "tool3"},
			expected:    []string{"tool1", "tool2", "tool3"},
			options:     ToolCallTrajectoryOptions{IgnoreOrder: false},
			wantMatch:   false,
			description: "order matters and sequence is wrong",
		},
		{
			name:        "out of order (order ignored)",
			actual:      []string{"tool2", "tool1", "tool3"},
			expected:    []string{"tool1", "tool2", "tool3"},
			options:     ToolCallTrajectoryOptions{IgnoreOrder: true},
			wantMatch:   true,
			description: "all tools present, order doesn't matter",
		},
		{
			name:        "missing tool",
			actual:      []string{"tool1", "tool3"},
			expected:    []string{"tool1", "tool2", "tool3"},
			options:     ToolCallTrajectoryOptions{},
			wantMatch:   false,
			description: "tool2 is missing",
		},
		{
			name:        "empty sequences",
			actual:      []string{},
			expected:    []string{},
			options:     ToolCallTrajectoryOptions{},
			wantMatch:   true,
			description: "both sequences are empty",
		},
		{
			name:        "duplicate tools (ignore order)",
			actual:      []string{"tool1", "tool1", "tool2"},
			expected:    []string{"tool1", "tool1", "tool2"},
			options:     ToolCallTrajectoryOptions{IgnoreOrder: true},
			wantMatch:   true,
			description: "duplicate tools match when order ignored",
		},
		{
			name:        "duplicate tools count mismatch",
			actual:      []string{"tool1", "tool2"},
			expected:    []string{"tool1", "tool1", "tool2"},
			options:     ToolCallTrajectoryOptions{IgnoreOrder: true},
			wantMatch:   false,
			description: "expected two tool1 but got one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewToolCallTrajectory("tool_sequence").WithOptions(tt.options)
			matches, rationale := scorer.compareSequences(tt.actual, tt.expected)

			if matches != tt.wantMatch {
				t.Errorf("expected match=%v, got %v. Rationale: %s", tt.wantMatch, matches, rationale)
			}
		})
	}
}

func TestStepValidationScorer_NoTrace(t *testing.T) {
	scorer := NewStepValidation()

	input := ScorerInput{
		Outputs:      map[string]any{},
		Expectations: map[string]any{},
		Trace:        nil,
	}

	score, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if score.Error == nil {
		t.Fatal("expected error for missing trace")
	}
}

func TestStepValidationScorer_StepCount(t *testing.T) {
	tests := []struct {
		name      string
		stepCount int
		minSteps  int
		maxSteps  int
		wantMatch bool
	}{
		{
			name:      "within range",
			stepCount: 5,
			minSteps:  3,
			maxSteps:  7,
			wantMatch: true,
		},
		{
			name:      "below minimum",
			stepCount: 2,
			minSteps:  3,
			maxSteps:  7,
			wantMatch: false,
		},
		{
			name:      "above maximum",
			stepCount: 8,
			minSteps:  3,
			maxSteps:  7,
			wantMatch: false,
		},
		{
			name:      "no minimum",
			stepCount: 1,
			minSteps:  0,
			maxSteps:  7,
			wantMatch: true,
		},
		{
			name:      "no maximum",
			stepCount: 100,
			minSteps:  3,
			maxSteps:  0,
			wantMatch: true,
		},
		{
			name:      "no constraints",
			stepCount: 42,
			minSteps:  0,
			maxSteps:  0,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewStepValidation().WithOptions(StepValidationOptions{
				MinSteps: tt.minSteps,
				MaxSteps: tt.maxSteps,
			})

			// Mock extractSteps to return the desired count
			originalExtractSteps := scorer.extractSteps
			scorer.extractSteps = func(trace *Trace) []string {
				steps := make([]string, tt.stepCount)
				for i := range steps {
					steps[i] = "step content"
				}
				return steps
			}
			defer func() { scorer.extractSteps = originalExtractSteps }()

			input := ScorerInput{
				Outputs:      map[string]any{},
				Expectations: map[string]any{},
				Trace:        &Trace{},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Error != nil {
				t.Fatalf("unexpected score error: %v", score.Error)
			}

			match, ok := score.Value.(bool)
			if !ok {
				t.Fatalf("expected bool value, got %T", score.Value)
			}

			if match != tt.wantMatch {
				t.Errorf("expected match=%v, got %v. Rationale: %s", tt.wantMatch, match, score.Rationale)
			}
		})
	}
}

func TestStepValidationScorer_ContentPatterns(t *testing.T) {
	tests := []struct {
		name        string
		steps       []string
		patterns    map[int]string
		wantMatch   bool
		description string
	}{
		{
			name:  "pattern matches",
			steps: []string{"step 1: hello", "step 2: world"},
			patterns: map[int]string{
				0: "hello",
				1: "world",
			},
			wantMatch:   true,
			description: "all patterns match",
		},
		{
			name:  "pattern does not match",
			steps: []string{"step 1: hello", "step 2: world"},
			patterns: map[int]string{
				0: "goodbye",
			},
			wantMatch:   false,
			description: "pattern does not match step content",
		},
		{
			name:  "step index out of bounds",
			steps: []string{"step 1: hello"},
			patterns: map[int]string{
				1: "world",
			},
			wantMatch:   false,
			description: "pattern for non-existent step",
		},
		{
			name:  "regex pattern",
			steps: []string{"error: code 404", "success"},
			patterns: map[int]string{
				0: `error: code \d+`,
			},
			wantMatch:   true,
			description: "regex pattern matches",
		},
		{
			name:        "no patterns",
			steps:       []string{"step 1", "step 2"},
			patterns:    map[int]string{},
			wantMatch:   true,
			description: "no patterns to validate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewStepValidation().WithOptions(StepValidationOptions{
				ContentPatterns: tt.patterns,
			})

			// Mock extractSteps to return the desired steps
			originalExtractSteps := scorer.extractSteps
			scorer.extractSteps = func(trace *Trace) []string {
				return tt.steps
			}
			defer func() { scorer.extractSteps = originalExtractSteps }()

			input := ScorerInput{
				Outputs:      map[string]any{},
				Expectations: map[string]any{},
				Trace:        &Trace{},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Some tests expect an error (like invalid regex)
			if score.Error != nil && tt.wantMatch {
				t.Fatalf("unexpected score error: %v", score.Error)
			}

			if score.Error == nil {
				match, ok := score.Value.(bool)
				if !ok {
					t.Fatalf("expected bool value, got %T", score.Value)
				}

				if match != tt.wantMatch {
					t.Errorf("expected match=%v, got %v. Rationale: %s", tt.wantMatch, match, score.Rationale)
				}
			}
		})
	}
}

func TestStepValidationScorer_InvalidRegex(t *testing.T) {
	scorer := NewStepValidation().WithOptions(StepValidationOptions{
		ContentPatterns: map[int]string{
			0: "[invalid",
		},
	})

	// Mock extractSteps to return a step
	originalExtractSteps := scorer.extractSteps
	scorer.extractSteps = func(trace *Trace) []string {
		return []string{"step content"}
	}
	defer func() { scorer.extractSteps = originalExtractSteps }()

	input := ScorerInput{
		Outputs:      map[string]any{},
		Expectations: map[string]any{},
		Trace:        &Trace{},
	}

	score, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if score.Error == nil {
		t.Fatal("expected error for invalid regex pattern")
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    []string
		wantErr bool
	}{
		{
			name:    "string slice",
			input:   []string{"a", "b", "c"},
			want:    []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "interface slice",
			input:   []interface{}{"a", "b", "c"},
			want:    []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "empty slice",
			input:   []string{},
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "mixed types in interface slice",
			input:   []interface{}{"a", 123, "c"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "not a slice",
			input:   "not a slice",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "int slice",
			input:   []int{1, 2, 3},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toStringSlice(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("expected length %d, got %d", len(tt.want), len(got))
			}

			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: expected %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}
