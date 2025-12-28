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
	Trace        any            // Optional trace data (will be *tracing.Trace, use any to avoid circular import)
	Inputs       map[string]any // Original inputs (for context in scoring)
}

// Score represents the result of a scorer evaluation.
type Score struct {
	Value     any            // bool, float64, int, or string
	Rationale string         // Explanation of the score
	Metadata  map[string]any // Additional structured information
	Error     error          // Non-nil if scorer failed
}
