package eval

import (
	"context"
	"time"
)

// Scorer evaluates outputs against expectations.
type Scorer interface {
	// Name returns the scorer's identifier.
	Name() string

	// Score evaluates the given input/output pair.
	Score(ctx context.Context, input ScorerInput) (Score, error)
}

// ScorerInput contains all data needed for scoring.
type ScorerInput struct {
	Outputs      map[string]any // Generated outputs to evaluate
	Expectations map[string]any // Expected values for comparison
	Trace        any            // Optional trace data (*tracing.Trace)
	Inputs       map[string]any // Original inputs (for context in scoring)
}

// Score contains the result of a scorer evaluation.
type Score struct {
	Value     any            // bool, float64, int, or string
	Rationale string         // Explanation of the score
	Metadata  map[string]any // Additional structured information
	Error     error          // Non-nil if scorer failed
}

// BoolValue returns the score value as a boolean.
// Returns false if the value is not a boolean.
func (s Score) BoolValue() bool {
	if v, ok := s.Value.(bool); ok {
		return v
	}
	return false
}

// FloatValue returns the score value as a float64.
// Returns 0 if the value is not numeric.
func (s Score) FloatValue() float64 {
	switch v := s.Value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case bool:
		if v {
			return 1.0
		}
		return 0.0
	default:
		return 0
	}
}

// TestCase represents a single test case in a dataset.
type TestCase struct {
	Inputs       map[string]any    // Input values for the test
	Expectations map[string]any    // Expected output values
	Outputs      map[string]any    // Pre-generated outputs (optional)
	Tags         map[string]string // Tags for filtering/grouping
}

// Dataset contains a collection of test cases.
type Dataset struct {
	Name      string            // Dataset name
	TestCases []*TestCase       // Test cases to evaluate
	Metadata  map[string]string // Optional metadata
}

// Validate checks that the dataset is valid.
func (d *Dataset) Validate() error {
	if d.Name == "" {
		return &ValidationError{Field: "Name", Message: "dataset name is required"}
	}
	if len(d.TestCases) == 0 {
		return &ValidationError{Field: "TestCases", Message: "dataset must have at least one test case"}
	}
	for i, tc := range d.TestCases {
		if tc.Inputs == nil {
			return &ValidationError{
				Field:   "TestCases",
				Message: "test case must have inputs",
				Index:   i,
			}
		}
	}
	return nil
}

// ValidationError represents a dataset validation error.
type ValidationError struct {
	Field   string
	Message string
	Index   int // Test case index (-1 if not applicable)
}

func (e *ValidationError) Error() string {
	if e.Index >= 0 {
		return e.Field + "[" + string(rune('0'+e.Index)) + "]: " + e.Message
	}
	return e.Field + ": " + e.Message
}

// Results contains the complete evaluation output.
type Results struct {
	TestCases  []TestCaseResult       // Individual results per test case
	Summary    map[string]ScorerStats // Aggregated stats per scorer name
	Errors     []EvalError            // Evaluation-level errors
	StartTime  time.Time              // When evaluation started
	EndTime    time.Time              // When evaluation completed
	TotalTests int                    // Total number of test cases processed
}

// TestCaseResult contains results for a single test case.
type TestCaseResult struct {
	TestCase *TestCase          // The original test case
	Outputs  map[string]any     // Generated outputs (if PredictFunc was run)
	Scores   map[string]Score   // Scores keyed by scorer name
	Trace    any                // Captured trace (*tracing.Trace)
}

// ScorerStats contains aggregated statistics for a scorer.
type ScorerStats struct {
	PassRate   float64 // Percentage of passing scores (0.0-1.0)
	Mean       float64 // Mean score value (for numeric scores)
	StdDev     float64 // Standard deviation (for numeric scores)
	ErrorRate  float64 // Percentage of errored scores (0.0-1.0)
	ErrorCount int     // Number of test cases where scorer errored
	Count      int     // Total number of scores computed
}

// EvalError represents an evaluation-level error.
type EvalError struct {
	Phase   string // "setup", "predict", "score", "export"
	Message string
	Cause   error
}

func (e EvalError) Error() string {
	if e.Cause != nil {
		return e.Phase + ": " + e.Message + ": " + e.Cause.Error()
	}
	return e.Phase + ": " + e.Message
}

// PredictFunc generates outputs from inputs.
// Returns outputs map and optional trace for agent-specific scorers.
type PredictFunc func(ctx context.Context, inputs map[string]any) (outputs map[string]any, trace any, err error)
