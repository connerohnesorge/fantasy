package eval

import (
	"context"
	"testing"
)

func TestExactMatchScorer(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		expected      string
		caseSensitive bool
		wantMatch     bool
		wantError     bool
	}{
		{
			name:          "exact match",
			output:        "hello world",
			expected:      "hello world",
			caseSensitive: true,
			wantMatch:     true,
		},
		{
			name:          "no match",
			output:        "hello world",
			expected:      "goodbye world",
			caseSensitive: true,
			wantMatch:     false,
		},
		{
			name:          "case sensitive match fails",
			output:        "Hello World",
			expected:      "hello world",
			caseSensitive: true,
			wantMatch:     false,
		},
		{
			name:          "case insensitive match succeeds",
			output:        "Hello World",
			expected:      "hello world",
			caseSensitive: false,
			wantMatch:     true,
		},
		{
			name:          "numeric match",
			output:        "42",
			expected:      "42",
			caseSensitive: true,
			wantMatch:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewExactMatch("output", "expected").WithCaseSensitive(tt.caseSensitive)

			input := ScorerInput{
				Outputs: map[string]any{
					"output": tt.output,
				},
				Expectations: map[string]any{
					"expected": tt.expected,
				},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError {
				if score.Error == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if score.Error != nil {
				t.Fatalf("unexpected score error: %v", score.Error)
			}

			match, ok := score.Value.(bool)
			if !ok {
				t.Fatalf("expected bool value, got %T", score.Value)
			}

			if match != tt.wantMatch {
				t.Errorf("expected match=%v, got %v", tt.wantMatch, match)
			}
		})
	}
}

func TestExactMatchScorer_MissingKeys(t *testing.T) {
	scorer := NewExactMatch("output", "expected")

	tests := []struct {
		name  string
		input ScorerInput
	}{
		{
			name: "missing output",
			input: ScorerInput{
				Outputs: map[string]any{},
				Expectations: map[string]any{
					"expected": "value",
				},
			},
		},
		{
			name: "missing expectation",
			input: ScorerInput{
				Outputs: map[string]any{
					"output": "value",
				},
				Expectations: map[string]any{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := scorer.Score(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Error == nil {
				t.Fatal("expected score error but got none")
			}
		})
	}
}

func TestContainsScorer(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		expected      string
		caseSensitive bool
		wantContains  bool
	}{
		{
			name:          "contains substring",
			output:        "hello world",
			expected:      "world",
			caseSensitive: true,
			wantContains:  true,
		},
		{
			name:          "does not contain",
			output:        "hello world",
			expected:      "goodbye",
			caseSensitive: true,
			wantContains:  false,
		},
		{
			name:          "case sensitive fails",
			output:        "Hello World",
			expected:      "world",
			caseSensitive: true,
			wantContains:  false,
		},
		{
			name:          "case insensitive succeeds",
			output:        "Hello World",
			expected:      "world",
			caseSensitive: false,
			wantContains:  true,
		},
		{
			name:          "empty substring",
			output:        "hello",
			expected:      "",
			caseSensitive: true,
			wantContains:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewContains("output", "expected").WithCaseSensitive(tt.caseSensitive)

			input := ScorerInput{
				Outputs: map[string]any{
					"output": tt.output,
				},
				Expectations: map[string]any{
					"expected": tt.expected,
				},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if score.Error != nil {
				t.Fatalf("unexpected score error: %v", score.Error)
			}

			contains, ok := score.Value.(bool)
			if !ok {
				t.Fatalf("expected bool value, got %T", score.Value)
			}

			if contains != tt.wantContains {
				t.Errorf("expected contains=%v, got %v", tt.wantContains, contains)
			}
		})
	}
}

func TestRegexScorer(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		pattern     string
		wantMatch   bool
		wantError   bool
	}{
		{
			name:      "simple match",
			output:    "hello123",
			pattern:   `hello\d+`,
			wantMatch: true,
		},
		{
			name:      "no match",
			output:    "hello",
			pattern:   `\d+`,
			wantMatch: false,
		},
		{
			name:      "email pattern",
			output:    "user@example.com",
			pattern:   `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			wantMatch: true,
		},
		{
			name:      "invalid pattern",
			output:    "test",
			pattern:   `[`,
			wantMatch: false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewRegex("output", "pattern")

			input := ScorerInput{
				Outputs: map[string]any{
					"output": tt.output,
				},
				Expectations: map[string]any{
					"pattern": tt.pattern,
				},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError {
				if score.Error == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if score.Error != nil {
				t.Fatalf("unexpected score error: %v", score.Error)
			}

			match, ok := score.Value.(bool)
			if !ok {
				t.Fatalf("expected bool value, got %T", score.Value)
			}

			if match != tt.wantMatch {
				t.Errorf("expected match=%v, got %v", tt.wantMatch, match)
			}
		})
	}
}

func TestJSONMatchScorer(t *testing.T) {
	tests := []struct {
		name      string
		output    any
		expected  any
		options   JSONMatchOptions
		wantMatch bool
	}{
		{
			name:      "simple object match",
			output:    map[string]any{"name": "Alice", "age": 30},
			expected:  map[string]any{"name": "Alice", "age": 30},
			wantMatch: true,
		},
		{
			name:      "object with extra keys",
			output:    map[string]any{"name": "Alice", "age": 30, "city": "NYC"},
			expected:  map[string]any{"name": "Alice", "age": 30},
			wantMatch: true,
		},
		{
			name:      "object value mismatch",
			output:    map[string]any{"name": "Alice", "age": 30},
			expected:  map[string]any{"name": "Alice", "age": 25},
			wantMatch: false,
		},
		{
			name:      "nested object match",
			output:    map[string]any{"user": map[string]any{"name": "Alice"}},
			expected:  map[string]any{"user": map[string]any{"name": "Alice"}},
			wantMatch: true,
		},
		{
			name:      "array order matters",
			output:    []any{1, 2, 3},
			expected:  []any{1, 2, 3},
			wantMatch: true,
		},
		{
			name:      "array order mismatch",
			output:    []any{1, 2, 3},
			expected:  []any{3, 2, 1},
			wantMatch: false,
		},
		{
			name:      "array order ignored",
			output:    []any{1, 2, 3},
			expected:  []any{3, 2, 1},
			options:   JSONMatchOptions{IgnoreArrayOrder: true},
			wantMatch: true,
		},
		{
			name:      "null vs missing (strict)",
			output:    map[string]any{"name": "Alice"},
			expected:  map[string]any{"name": "Alice", "age": nil},
			options:   JSONMatchOptions{NullEqualsMissing: false},
			wantMatch: false,
		},
		{
			name:      "null vs missing (permissive)",
			output:    map[string]any{"name": "Alice"},
			expected:  map[string]any{"name": "Alice", "age": nil},
			options:   JSONMatchOptions{NullEqualsMissing: true},
			wantMatch: true,
		},
		{
			name:      "float tolerance exact",
			output:    map[string]any{"value": 1.0},
			expected:  map[string]any{"value": 1.0001},
			options:   JSONMatchOptions{FloatTolerance: 0.0},
			wantMatch: false,
		},
		{
			name:      "float tolerance permissive",
			output:    map[string]any{"value": 1.0},
			expected:  map[string]any{"value": 1.0001},
			options:   JSONMatchOptions{FloatTolerance: 0.001},
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewJSONMatch("output", "expected").WithOptions(tt.options)

			input := ScorerInput{
				Outputs: map[string]any{
					"output": tt.output,
				},
				Expectations: map[string]any{
					"expected": tt.expected,
				},
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

func TestJSONMatchScorer_JSONStrings(t *testing.T) {
	scorer := NewJSONMatch("output", "expected")

	input := ScorerInput{
		Outputs: map[string]any{
			"output": `{"name": "Alice", "age": 30}`,
		},
		Expectations: map[string]any{
			"expected": `{"name":"Alice","age":30}`,
		},
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

	if !match {
		t.Errorf("expected JSON strings to match. Rationale: %s", score.Rationale)
	}
}

func TestNumericRangeScorer(t *testing.T) {
	tests := []struct {
		name      string
		output    any
		min       float64
		max       float64
		minInc    bool
		maxInc    bool
		wantMatch bool
		wantError bool
	}{
		{
			name:      "within range inclusive",
			output:    5.0,
			min:       0.0,
			max:       10.0,
			minInc:    true,
			maxInc:    true,
			wantMatch: true,
		},
		{
			name:      "below range",
			output:    -1.0,
			min:       0.0,
			max:       10.0,
			minInc:    true,
			maxInc:    true,
			wantMatch: false,
		},
		{
			name:      "above range",
			output:    11.0,
			min:       0.0,
			max:       10.0,
			minInc:    true,
			maxInc:    true,
			wantMatch: false,
		},
		{
			name:      "on min boundary inclusive",
			output:    0.0,
			min:       0.0,
			max:       10.0,
			minInc:    true,
			maxInc:    true,
			wantMatch: true,
		},
		{
			name:      "on min boundary exclusive",
			output:    0.0,
			min:       0.0,
			max:       10.0,
			minInc:    false,
			maxInc:    true,
			wantMatch: false,
		},
		{
			name:      "on max boundary inclusive",
			output:    10.0,
			min:       0.0,
			max:       10.0,
			minInc:    true,
			maxInc:    true,
			wantMatch: true,
		},
		{
			name:      "on max boundary exclusive",
			output:    10.0,
			min:       0.0,
			max:       10.0,
			minInc:    true,
			maxInc:    false,
			wantMatch: false,
		},
		{
			name:      "integer value",
			output:    5,
			min:       0.0,
			max:       10.0,
			minInc:    true,
			maxInc:    true,
			wantMatch: true,
		},
		{
			name:      "non-numeric value",
			output:    "not a number",
			min:       0.0,
			max:       10.0,
			minInc:    true,
			maxInc:    true,
			wantMatch: false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewNumericRange("output", tt.min, tt.max).
				WithMinInclusive(tt.minInc).
				WithMaxInclusive(tt.maxInc)

			input := ScorerInput{
				Outputs: map[string]any{
					"output": tt.output,
				},
			}

			score, err := scorer.Score(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError {
				if score.Error == nil {
					t.Fatal("expected error but got none")
				}
				return
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

func TestNumericRangeScorer_Metadata(t *testing.T) {
	scorer := NewNumericRange("output", 0.0, 10.0)

	input := ScorerInput{
		Outputs: map[string]any{
			"output": 5.0,
		},
	}

	score, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if score.Error != nil {
		t.Fatalf("unexpected score error: %v", score.Error)
	}

	// Check metadata
	if score.Metadata == nil {
		t.Fatal("expected metadata but got nil")
	}

	if _, ok := score.Metadata["value"]; !ok {
		t.Error("expected 'value' in metadata")
	}

	if _, ok := score.Metadata["min"]; !ok {
		t.Error("expected 'min' in metadata")
	}

	if _, ok := score.Metadata["max"]; !ok {
		t.Error("expected 'max' in metadata")
	}

	if _, ok := score.Metadata["range"]; !ok {
		t.Error("expected 'range' in metadata")
	}
}
