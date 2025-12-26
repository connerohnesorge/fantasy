package eval

import (
	"context"
)

// Scorer evaluates outputs against expectations.
type Scorer interface {
	// Name returns the scorer's identifier.
	Name() string

	// Score evaluates the given input/output pair.
	// ctx: context for cancellation/timeout
	// input: ScorerInput containing outputs, expectations, and optional trace
	// Returns: Score result or error
	Score(ctx context.Context, input ScorerInput) (Score, error)
}

// ScorerInput contains all data needed for scoring.
type ScorerInput struct {
	Outputs      map[string]any // Generated outputs to evaluate
	Expectations map[string]any // Expected values for comparison
	Trace        *Trace         // Optional trace data (nil for scorers that don't need it)
	Inputs       map[string]any // Original inputs (for context in scoring)
}

// Score represents the result of scoring an output.
type Score struct {
	Value     any            // bool, float64, int, or string
	Rationale string         // Explanation of the score
	Metadata  map[string]any // Additional structured information
	Error     error          // Non-nil if scorer failed (Value should be nil/zero)
}

// Trace is a placeholder type that will be replaced with the actual tracing.Trace type
// when the tracing package is implemented. This allows the eval package to compile
// without the tracing package being present yet.
//
// TODO: Replace with charm.land/fantasy/tracing.Trace when tracing package is implemented.
type Trace struct {
	// Placeholder fields - actual structure defined in tracing package
}
