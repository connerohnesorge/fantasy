package eval

import (
	"context"
	"testing"
)

func TestExactMatchScorer(t *testing.T) {
	scorer := ExactMatchScorer("output")

	tests := []struct {
		name    string
		outputs map[string]any
		expects map[string]any
		want    bool
	}{
		{
			name:    "exact string match",
			outputs: map[string]any{"output": "hello"},
			expects: map[string]any{"output": "hello"},
			want:    true,
		},
		{
			name:    "string mismatch",
			outputs: map[string]any{"output": "hello"},
			expects: map[string]any{"output": "world"},
			want:    false,
		},
		{
			name:    "numeric match",
			outputs: map[string]any{"output": 42},
			expects: map[string]any{"output": 42},
			want:    true,
		},
		{
			name:    "missing output field",
			outputs: map[string]any{},
			expects: map[string]any{"output": "hello"},
			want:    false,
		},
		{
			name:    "missing expected field",
			outputs: map[string]any{"output": "hello"},
			expects: map[string]any{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := scorer.Score(context.Background(), ScorerInput{
				Outputs:      tt.outputs,
				Expectations: tt.expects,
			})
			if err != nil {
				t.Fatalf("Score() error = %v", err)
			}
			if score.BoolValue() != tt.want {
				t.Errorf("Score() = %v, want %v", score.BoolValue(), tt.want)
			}
		})
	}
}

func TestContainsScorer(t *testing.T) {
	scorer := ContainsScorer("output")

	tests := []struct {
		name    string
		outputs map[string]any
		expects map[string]any
		want    bool
	}{
		{
			name:    "contains substring",
			outputs: map[string]any{"output": "hello world"},
			expects: map[string]any{"output": "world"},
			want:    true,
		},
		{
			name:    "exact match counts as contains",
			outputs: map[string]any{"output": "hello"},
			expects: map[string]any{"output": "hello"},
			want:    true,
		},
		{
			name:    "does not contain",
			outputs: map[string]any{"output": "hello world"},
			expects: map[string]any{"output": "foo"},
			want:    false,
		},
		{
			name:    "case sensitive",
			outputs: map[string]any{"output": "Hello World"},
			expects: map[string]any{"output": "hello"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := scorer.Score(context.Background(), ScorerInput{
				Outputs:      tt.outputs,
				Expectations: tt.expects,
			})
			if err != nil {
				t.Fatalf("Score() error = %v", err)
			}
			if score.BoolValue() != tt.want {
				t.Errorf("Score() = %v, want %v", score.BoolValue(), tt.want)
			}
		})
	}
}

func TestRegexScorer(t *testing.T) {
	scorer := RegexScorer("output")

	tests := []struct {
		name    string
		outputs map[string]any
		expects map[string]any
		want    bool
		wantErr bool
	}{
		{
			name:    "matches pattern",
			outputs: map[string]any{"output": "hello123"},
			expects: map[string]any{"output": `\d+`},
			want:    true,
		},
		{
			name:    "does not match pattern",
			outputs: map[string]any{"output": "hello"},
			expects: map[string]any{"output": `\d+`},
			want:    false,
		},
		{
			name:    "full match pattern",
			outputs: map[string]any{"output": "abc123def"},
			expects: map[string]any{"output": `^abc\d+def$`},
			want:    true,
		},
		{
			name:    "invalid regex",
			outputs: map[string]any{"output": "hello"},
			expects: map[string]any{"output": `[invalid`},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := scorer.Score(context.Background(), ScorerInput{
				Outputs:      tt.outputs,
				Expectations: tt.expects,
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Score() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && score.BoolValue() != tt.want {
				t.Errorf("Score() = %v, want %v", score.BoolValue(), tt.want)
			}
		})
	}
}

func TestJSONMatchScorer(t *testing.T) {
	tests := []struct {
		name    string
		opts    JSONMatchOptions
		outputs map[string]any
		expects map[string]any
		want    bool
	}{
		{
			name:    "identical objects",
			outputs: map[string]any{"output": map[string]any{"a": 1, "b": "hello"}},
			expects: map[string]any{"output": map[string]any{"a": 1, "b": "hello"}},
			want:    true,
		},
		{
			name:    "different values",
			outputs: map[string]any{"output": map[string]any{"a": 1}},
			expects: map[string]any{"output": map[string]any{"a": 2}},
			want:    false,
		},
		{
			name:    "JSON string parsing",
			outputs: map[string]any{"output": `{"a": 1}`},
			expects: map[string]any{"output": map[string]any{"a": float64(1)}},
			want:    true,
		},
		{
			name:    "array order matters by default",
			outputs: map[string]any{"output": []any{1, 2, 3}},
			expects: map[string]any{"output": []any{3, 2, 1}},
			want:    false,
		},
		{
			name:    "array order ignored when configured",
			opts:    JSONMatchOptions{IgnoreArrayOrder: true},
			outputs: map[string]any{"output": []any{1, 2, 3}},
			expects: map[string]any{"output": []any{3, 2, 1}},
			want:    true,
		},
		{
			name:    "float tolerance",
			opts:    JSONMatchOptions{FloatTolerance: 0.01},
			outputs: map[string]any{"output": map[string]any{"value": 1.005}},
			expects: map[string]any{"output": map[string]any{"value": 1.0}},
			want:    true,
		},
		{
			name:    "null equals missing",
			opts:    JSONMatchOptions{NullEqualsMissing: true},
			outputs: map[string]any{"output": map[string]any{"a": 1}},
			expects: map[string]any{"output": map[string]any{"a": 1, "b": nil}},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := JSONMatchScorer("output", tt.opts)
			score, err := scorer.Score(context.Background(), ScorerInput{
				Outputs:      tt.outputs,
				Expectations: tt.expects,
			})
			if err != nil {
				t.Fatalf("Score() error = %v", err)
			}
			if score.BoolValue() != tt.want {
				t.Errorf("Score() = %v, want %v (rationale: %s)", score.BoolValue(), tt.want, score.Rationale)
			}
		})
	}
}

func TestNumericRangeScorer(t *testing.T) {
	scorer := NumericRangeScorer("output", 0.0, 1.0)

	tests := []struct {
		name    string
		outputs map[string]any
		want    bool
	}{
		{
			name:    "within range",
			outputs: map[string]any{"output": 0.5},
			want:    true,
		},
		{
			name:    "at minimum",
			outputs: map[string]any{"output": 0.0},
			want:    true,
		},
		{
			name:    "at maximum",
			outputs: map[string]any{"output": 1.0},
			want:    true,
		},
		{
			name:    "below range",
			outputs: map[string]any{"output": -0.1},
			want:    false,
		},
		{
			name:    "above range",
			outputs: map[string]any{"output": 1.1},
			want:    false,
		},
		{
			name:    "integer value",
			outputs: map[string]any{"output": 1},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := scorer.Score(context.Background(), ScorerInput{
				Outputs:      tt.outputs,
				Expectations: map[string]any{},
			})
			if err != nil {
				t.Fatalf("Score() error = %v", err)
			}
			if score.BoolValue() != tt.want {
				t.Errorf("Score() = %v, want %v", score.BoolValue(), tt.want)
			}
		})
	}
}

func TestScore_FloatValue(t *testing.T) {
	tests := []struct {
		name  string
		score Score
		want  float64
	}{
		{
			name:  "float64 value",
			score: Score{Value: 0.75},
			want:  0.75,
		},
		{
			name:  "int value",
			score: Score{Value: 42},
			want:  42.0,
		},
		{
			name:  "bool true",
			score: Score{Value: true},
			want:  1.0,
		},
		{
			name:  "bool false",
			score: Score{Value: false},
			want:  0.0,
		},
		{
			name:  "nil value",
			score: Score{Value: nil},
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.score.FloatValue(); got != tt.want {
				t.Errorf("FloatValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataset_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dataset *Dataset
		wantErr bool
	}{
		{
			name: "valid dataset",
			dataset: &Dataset{
				Name: "test",
				TestCases: []*TestCase{
					{Inputs: map[string]any{"prompt": "hello"}},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			dataset: &Dataset{
				TestCases: []*TestCase{
					{Inputs: map[string]any{"prompt": "hello"}},
				},
			},
			wantErr: true,
		},
		{
			name: "no test cases",
			dataset: &Dataset{
				Name:      "test",
				TestCases: []*TestCase{},
			},
			wantErr: true,
		},
		{
			name: "test case without inputs",
			dataset: &Dataset{
				Name: "test",
				TestCases: []*TestCase{
					{Inputs: nil},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dataset.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
