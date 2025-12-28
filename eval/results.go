package eval

import (
	"time"
)

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
	TestCase TestCase         // The original test case
	Outputs  map[string]any   // Generated outputs (if PredictFunc was run)
	Scores   map[string]Score // Scores keyed by scorer name
	Trace    any              // Captured trace (will be *tracing.Trace)
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
