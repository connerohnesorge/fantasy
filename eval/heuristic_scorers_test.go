package eval

import (
	"context"
	"testing"
)

func TestExactMatchScorer(t *testing.T) {
	scorer := NewExactMatch()

	tests := []struct {
		name     string
		output   string
		expected string
		wantVal  float64
	}{
		{
			name:     "exact match",
			output:   "hello world",
			expected: "hello world",
			wantVal:  1.0,
		},
		{
			name:     "no match",
			output:   "hello world",
			expected: "goodbye world",
			wantVal:  0.0,
		},
		{
			name:     "case sensitive",
			output:   "Hello World",
			expected: "hello world",
			wantVal:  0.0,
		},
		{
			name:     "empty strings match",
			output:   "",
			expected: "",
			wantVal:  1.0,
		},
		{
			name:     "whitespace matters",
			output:   "hello world",
			expected: "hello  world",
			wantVal:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := ScorerInput{
				Outputs:      map[string]any{"output": tt.output},
				Expectations: map[string]any{"expected": tt.expected},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Value != tt.wantVal {
				t.Errorf("got score %v, want %v", score.Value, tt.wantVal)
			}
		})
	}
}

func TestContainsScorer(t *testing.T) {
	tests := []struct {
		name            string
		output          string
		expected        string
		caseInsensitive bool
		wantVal         float64
	}{
		{
			name:     "contains substring",
			output:   "hello world",
			expected: "world",
			wantVal:  1.0,
		},
		{
			name:     "does not contain",
			output:   "hello world",
			expected: "goodbye",
			wantVal:  0.0,
		},
		{
			name:            "case insensitive match",
			output:          "Hello World",
			expected:        "world",
			caseInsensitive: true,
			wantVal:         1.0,
		},
		{
			name:            "case sensitive no match",
			output:          "Hello World",
			expected:        "world",
			caseInsensitive: false,
			wantVal:         0.0,
		},
		{
			name:     "empty substring always matches",
			output:   "hello",
			expected: "",
			wantVal:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewContains(WithCaseInsensitive(tt.caseInsensitive))

			input := ScorerInput{
				Outputs:      map[string]any{"output": tt.output},
				Expectations: map[string]any{"expected": tt.expected},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Value != tt.wantVal {
				t.Errorf("got score %v, want %v", score.Value, tt.wantVal)
			}
		})
	}
}

func TestRegexScorer(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		output  string
		wantVal float64
		wantErr bool
	}{
		{
			name:    "simple match",
			pattern: `^hello`,
			output:  "hello world",
			wantVal: 1.0,
		},
		{
			name:    "no match",
			pattern: `^goodbye`,
			output:  "hello world",
			wantVal: 0.0,
		},
		{
			name:    "digit pattern",
			pattern: `\d+`,
			output:  "test123",
			wantVal: 1.0,
		},
		{
			name:    "complex pattern",
			pattern: `^[A-Z][a-z]+\s\d+$`,
			output:  "Hello 123",
			wantVal: 1.0,
		},
		{
			name:    "invalid regex",
			pattern: `[invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer, err := NewRegex(tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			input := ScorerInput{
				Outputs: map[string]any{"output": tt.output},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Value != tt.wantVal {
				t.Errorf("got score %v, want %v", score.Value, tt.wantVal)
			}
		})
	}
}

func TestJSONMatchScorer(t *testing.T) {
	tests := []struct {
		name              string
		output            any
		expected          any
		ignoreOrder       bool
		ignoreExtraFields bool
		ignoreTypes       bool
		wantVal           float64
	}{
		{
			name:     "exact object match",
			output:   map[string]any{"name": "Alice", "age": 30},
			expected: map[string]any{"name": "Alice", "age": 30},
			wantVal:  1.0,
		},
		{
			name:     "object mismatch",
			output:   map[string]any{"name": "Alice", "age": 30},
			expected: map[string]any{"name": "Bob", "age": 30},
			wantVal:  0.0,
		},
		{
			name:              "ignore extra fields",
			output:            map[string]any{"name": "Alice", "age": 30, "city": "NYC"},
			expected:          map[string]any{"name": "Alice", "age": 30},
			ignoreExtraFields: true,
			wantVal:           1.0,
		},
		{
			name:              "fail on extra fields",
			output:            map[string]any{"name": "Alice", "age": 30, "city": "NYC"},
			expected:          map[string]any{"name": "Alice", "age": 30},
			ignoreExtraFields: false,
			wantVal:           0.0,
		},
		{
			name:     "array exact match",
			output:   []any{"a", "b", "c"},
			expected: []any{"a", "b", "c"},
			wantVal:  1.0,
		},
		{
			name:        "array order matters by default",
			output:      []any{"a", "c", "b"},
			expected:    []any{"a", "b", "c"},
			ignoreOrder: false,
			wantVal:     0.0,
		},
		{
			name:        "array ignore order",
			output:      []any{"a", "c", "b"},
			expected:    []any{"a", "b", "c"},
			ignoreOrder: true,
			wantVal:     1.0,
		},
		{
			name:        "ignore types",
			output:      map[string]any{"value": "123"},
			expected:    map[string]any{"value": 123},
			ignoreTypes: true,
			wantVal:     1.0,
		},
		{
			name:     "nested object match",
			output:   map[string]any{"user": map[string]any{"name": "Alice"}},
			expected: map[string]any{"user": map[string]any{"name": "Alice"}},
			wantVal:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewJSONMatch(
				WithIgnoreOrder(tt.ignoreOrder),
				WithIgnoreExtraFields(tt.ignoreExtraFields),
				WithIgnoreTypes(tt.ignoreTypes),
			)

			input := ScorerInput{
				Outputs:      map[string]any{"output": tt.output},
				Expectations: map[string]any{"expected": tt.expected},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Value != tt.wantVal {
				t.Errorf("got score %v, want %v", score.Value, tt.wantVal)
			}
		})
	}
}

func TestJSONMatchScorer_JSONStrings(t *testing.T) {
	scorer := NewJSONMatch()

	input := ScorerInput{
		Outputs:      map[string]any{"output": `{"name": "Alice", "age": 30}`},
		Expectations: map[string]any{"expected": `{"name": "Alice", "age": 30}`},
	}

	score, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if score.Value != 1.0 {
		t.Errorf("got score %v, want 1.0", score.Value)
	}
}

func TestNumericRangeScorer(t *testing.T) {
	tests := []struct {
		name         string
		min          float64
		max          float64
		minInclusive bool
		maxInclusive bool
		value        any
		wantVal      float64
		wantErr      bool
	}{
		{
			name:         "within range inclusive",
			min:          0.0,
			max:          100.0,
			minInclusive: true,
			maxInclusive: true,
			value:        50.0,
			wantVal:      1.0,
		},
		{
			name:         "below range",
			min:          0.0,
			max:          100.0,
			minInclusive: true,
			maxInclusive: true,
			value:        -1.0,
			wantVal:      0.0,
		},
		{
			name:         "above range",
			min:          0.0,
			max:          100.0,
			minInclusive: true,
			maxInclusive: true,
			value:        101.0,
			wantVal:      0.0,
		},
		{
			name:         "at min inclusive",
			min:          0.0,
			max:          100.0,
			minInclusive: true,
			maxInclusive: true,
			value:        0.0,
			wantVal:      1.0,
		},
		{
			name:         "at min exclusive",
			min:          0.0,
			max:          100.0,
			minInclusive: false,
			maxInclusive: true,
			value:        0.0,
			wantVal:      0.0,
		},
		{
			name:         "at max inclusive",
			min:          0.0,
			max:          100.0,
			minInclusive: true,
			maxInclusive: true,
			value:        100.0,
			wantVal:      1.0,
		},
		{
			name:         "at max exclusive",
			min:          0.0,
			max:          100.0,
			minInclusive: true,
			maxInclusive: false,
			value:        100.0,
			wantVal:      0.0,
		},
		{
			name:         "integer value",
			min:          0.0,
			max:          100.0,
			minInclusive: true,
			maxInclusive: true,
			value:        42,
			wantVal:      1.0,
		},
		{
			name:         "string number",
			min:          0.0,
			max:          100.0,
			minInclusive: true,
			maxInclusive: true,
			value:        "50",
			wantVal:      1.0,
		},
		{
			name:         "invalid string",
			min:          0.0,
			max:          100.0,
			minInclusive: true,
			maxInclusive: true,
			value:        "not a number",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewNumericRange(tt.min, tt.max,
				WithMinInclusive(tt.minInclusive),
				WithMaxInclusive(tt.maxInclusive),
			)

			input := ScorerInput{
				Outputs: map[string]any{"output": tt.value},
			}

			score, err := scorer.Score(context.Background(), input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Value != tt.wantVal {
				t.Errorf("got score %v, want %v", score.Value, tt.wantVal)
			}
		})
	}
}

func TestToolCallTrajectoryScorer(t *testing.T) {
	tests := []struct {
		name       string
		actual     any
		expected   any
		exactMatch bool
		wantVal    float64
	}{
		{
			name:       "exact match",
			actual:     []string{"search", "calculate", "respond"},
			expected:   []string{"search", "calculate", "respond"},
			exactMatch: true,
			wantVal:    1.0,
		},
		{
			name:       "exact mismatch",
			actual:     []string{"search", "respond"},
			expected:   []string{"search", "calculate", "respond"},
			exactMatch: true,
			wantVal:    0.0,
		},
		{
			name:       "contains match",
			actual:     []string{"search", "filter", "calculate", "format", "respond"},
			expected:   []string{"search", "calculate", "respond"},
			exactMatch: false,
			wantVal:    1.0,
		},
		{
			name:       "contains mismatch order",
			actual:     []string{"search", "respond", "calculate"},
			expected:   []string{"search", "calculate", "respond"},
			exactMatch: false,
			wantVal:    0.0,
		},
		{
			name:       "tool call objects",
			actual:     []any{map[string]any{"name": "search"}, map[string]any{"name": "respond"}},
			expected:   []string{"search", "respond"},
			exactMatch: true,
			wantVal:    1.0,
		},
		{
			name:       "empty expected matches empty actual",
			actual:     []string{},
			expected:   []string{},
			exactMatch: true,
			wantVal:    1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewToolCallTrajectory(WithExactMatch(tt.exactMatch))

			input := ScorerInput{
				Outputs:      map[string]any{"tool_calls": tt.actual},
				Expectations: map[string]any{"tool_calls": tt.expected},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Value != tt.wantVal {
				t.Errorf("got score %v, want %v", score.Value, tt.wantVal)
			}
		})
	}
}

func TestStepValidationScorer(t *testing.T) {
	tests := []struct {
		name         string
		steps        []string
		minSteps     int
		maxSteps     int
		stepPatterns []string
		allMatch     bool
		wantVal      float64
		wantErr      bool
	}{
		{
			name:     "count validation pass",
			steps:    []string{"step1", "step2", "step3"},
			minSteps: 2,
			maxSteps: 5,
			wantVal:  1.0,
		},
		{
			name:     "too few steps",
			steps:    []string{"step1"},
			minSteps: 2,
			maxSteps: 5,
			wantVal:  0.0,
		},
		{
			name:     "too many steps",
			steps:    []string{"step1", "step2", "step3", "step4", "step5", "step6"},
			minSteps: 2,
			maxSteps: 5,
			wantVal:  0.0,
		},
		{
			name:         "pattern match - at least one",
			steps:        []string{"analyze data", "process", "generate report"},
			stepPatterns: []string{`analyze`, `report`},
			allMatch:     false,
			wantVal:      1.0,
		},
		{
			name:         "pattern match - all must match",
			steps:        []string{"analyze data", "process data", "generate report"},
			stepPatterns: []string{`data`, `report`},
			allMatch:     true,
			wantVal:      1.0, // All steps match at least one pattern
		},
		{
			name:         "pattern match - all must match fail",
			steps:        []string{"analyze data", "process data", "something else"},
			stepPatterns: []string{`data`, `report`},
			allMatch:     true,
			wantVal:      0.0, // "something else" doesn't match any pattern
		},
		{
			name:         "pattern no match",
			steps:        []string{"step1", "step2"},
			stepPatterns: []string{`analyze`},
			allMatch:     false,
			wantVal:      0.0,
		},
		{
			name:         "count and pattern both pass",
			steps:        []string{"analyze", "process", "report"},
			minSteps:     2,
			maxSteps:     5,
			stepPatterns: []string{`analyze`},
			allMatch:     false,
			wantVal:      1.0,
		},
		{
			name:         "invalid pattern",
			stepPatterns: []string{`[invalid`},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer, err := NewStepValidation(
				WithMinSteps(tt.minSteps),
				WithMaxSteps(tt.maxSteps),
				WithStepPatterns(tt.stepPatterns),
				WithAllMatch(tt.allMatch),
			)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error during creation: %v", err)
			}

			input := ScorerInput{
				Outputs: map[string]any{"steps": tt.steps},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Value != tt.wantVal {
				t.Errorf("got score %v, want %v (rationale: %s)", score.Value, tt.wantVal, score.Rationale)
			}
		})
	}
}

func TestExtractNumeric(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    float64
		wantErr bool
	}{
		{name: "float64", input: 42.5, want: 42.5},
		{name: "float32", input: float32(42.5), want: 42.5},
		{name: "int", input: 42, want: 42.0},
		{name: "int64", input: int64(42), want: 42.0},
		{name: "int32", input: int32(42), want: 42.0},
		{name: "string number", input: "42.5", want: 42.5},
		{name: "string invalid", input: "not a number", wantErr: true},
		{name: "nil", input: nil, wantErr: true},
		{name: "unsupported type", input: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractNumeric(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractToolNames(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  []string
	}{
		{
			name:  "string slice",
			input: []string{"tool1", "tool2"},
			want:  []string{"tool1", "tool2"},
		},
		{
			name:  "any slice with strings",
			input: []any{"tool1", "tool2"},
			want:  []string{"tool1", "tool2"},
		},
		{
			name:  "any slice with maps",
			input: []any{map[string]any{"name": "tool1"}, map[string]any{"name": "tool2"}},
			want:  []string{"tool1", "tool2"},
		},
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty slice",
			input: []string{},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolNames(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("got length %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("at index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsSubsequence(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
		actual   []string
		want     bool
	}{
		{
			name:     "exact match",
			expected: []string{"a", "b", "c"},
			actual:   []string{"a", "b", "c"},
			want:     true,
		},
		{
			name:     "is subsequence",
			expected: []string{"a", "c"},
			actual:   []string{"a", "b", "c"},
			want:     true,
		},
		{
			name:     "not subsequence - wrong order",
			expected: []string{"c", "a"},
			actual:   []string{"a", "b", "c"},
			want:     false,
		},
		{
			name:     "empty expected",
			expected: []string{},
			actual:   []string{"a", "b"},
			want:     true,
		},
		{
			name:     "empty actual",
			expected: []string{"a"},
			actual:   []string{},
			want:     false,
		},
		{
			name:     "both empty",
			expected: []string{},
			actual:   []string{},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubsequence(tt.expected, tt.actual)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractSteps(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  []string
	}{
		{
			name:  "string slice",
			input: []string{"step1", "step2"},
			want:  []string{"step1", "step2"},
		},
		{
			name:  "any slice",
			input: []any{"step1", "step2"},
			want:  []string{"step1", "step2"},
		},
		{
			name:  "mixed types",
			input: []any{"step1", 42, map[string]any{"key": "value"}},
			want:  []string{"step1", "42", "map[key:value]"},
		},
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSteps(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("got length %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("at index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
