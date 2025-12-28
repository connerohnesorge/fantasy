package eval

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// PredictFunc generates outputs from inputs.
// Returns outputs map and optional trace for agent-specific scorers.
// The trace will be *tracing.Trace from the tracing package (using any to avoid circular import).
type PredictFunc func(ctx context.Context, inputs map[string]any) (outputs map[string]any, trace any, err error)

// Option configures the Evaluator at creation time.
type Option func(*Evaluator)

// RunOption configures a specific evaluation run.
type RunOption func(*runConfig)

// Evaluator runs scorers against datasets.
type Evaluator struct {
	// MLflow export configuration (optional)
	mlflowConfig *mlflowExportConfig
}

// runConfig holds configuration for a specific evaluation run.
type runConfig struct {
	parallelism int
	timeout     time.Duration
	predictFunc PredictFunc
}

// NewEvaluator creates an Evaluator with optional configuration.
func NewEvaluator(opts ...Option) *Evaluator {
	e := &Evaluator{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// WithParallelism sets the number of parallel workers (default: 1 = sequential).
func WithParallelism(n int) RunOption {
	return func(rc *runConfig) {
		rc.parallelism = n
	}
}

// WithTimeout sets per-test-case timeout.
func WithTimeout(d time.Duration) RunOption {
	return func(rc *runConfig) {
		rc.timeout = d
	}
}

// WithPredict sets the prediction function for generating outputs.
func WithPredict(fn PredictFunc) RunOption {
	return func(rc *runConfig) {
		rc.predictFunc = fn
	}
}

// Run executes evaluation on the given dataset.
func (e *Evaluator) Run(ctx context.Context, dataset *Dataset, scorers []Scorer, opts ...RunOption) (*Results, error) {
	// Apply run options
	cfg := &runConfig{
		parallelism: 1, // Default to sequential execution
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate dataset
	if err := dataset.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dataset: %w", err)
	}

	// Initialize results
	results := &Results{
		StartTime:  time.Now(),
		TestCases:  make([]TestCaseResult, len(dataset.TestCases)),
		Summary:    make(map[string]ScorerStats),
		Errors:     []EvalError{},
		TotalTests: len(dataset.TestCases),
	}

	// Execute test cases
	if cfg.parallelism <= 1 {
		// Sequential execution
		e.runSequential(ctx, dataset, scorers, cfg, results)
	} else {
		// Parallel execution
		e.runParallel(ctx, dataset, scorers, cfg, results)
	}

	results.EndTime = time.Now()

	// Compute summary statistics
	e.computeSummary(results, scorers)

	// Export to MLflow if configured
	if e.mlflowConfig != nil {
		// Create a run if needed
		if e.mlflowConfig.runID == "" && e.mlflowConfig.client != nil {
			runID, err := DatasetToRun(ctx, e.mlflowConfig.client, dataset, e.mlflowConfig.experimentID)
			if err != nil {
				results.Errors = append(results.Errors, EvalError{
					Phase:   "export",
					Message: fmt.Sprintf("failed to create MLflow run: %v", err),
					Cause:   err,
				})
			} else {
				e.mlflowConfig.runID = runID
			}
		}

		// Export assessments
		e.exportAssessments(ctx, results)

		// Export metrics
		e.exportMetrics(ctx, results)
	}

	return results, nil
}

// runSequential executes test cases sequentially.
func (e *Evaluator) runSequential(ctx context.Context, dataset *Dataset, scorers []Scorer, cfg *runConfig, results *Results) {
	for i, tc := range dataset.TestCases {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			results.Errors = append(results.Errors, EvalError{
				Phase:   "score",
				Message: "context cancelled",
				Cause:   ctx.Err(),
			})
			return
		default:
		}

		// Execute test case
		result := e.executeTestCase(ctx, tc, scorers, cfg)
		results.TestCases[i] = result
	}
}

// runParallel executes test cases in parallel with worker pool.
func (e *Evaluator) runParallel(ctx context.Context, dataset *Dataset, scorers []Scorer, cfg *runConfig, results *Results) {
	var wg sync.WaitGroup
	workChan := make(chan int, len(dataset.TestCases))

	// Start workers
	for w := 0; w < cfg.parallelism; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range workChan {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Execute test case
				result := e.executeTestCase(ctx, dataset.TestCases[idx], scorers, cfg)
				results.TestCases[idx] = result
			}
		}()
	}

	// Send work to workers
	for i := range dataset.TestCases {
		select {
		case <-ctx.Done():
			results.Errors = append(results.Errors, EvalError{
				Phase:   "score",
				Message: "context cancelled",
				Cause:   ctx.Err(),
			})
			close(workChan)
			wg.Wait()
			return
		case workChan <- i:
		}
	}

	close(workChan)
	wg.Wait()
}

// executeTestCase runs a single test case through all scorers.
func (e *Evaluator) executeTestCase(ctx context.Context, tc TestCase, scorers []Scorer, cfg *runConfig) TestCaseResult {
	result := TestCaseResult{
		TestCase: tc,
		Scores:   make(map[string]Score),
	}

	// Apply timeout if configured
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	// Generate outputs if needed
	if tc.Outputs == nil || len(tc.Outputs) == 0 {
		if cfg.predictFunc == nil {
			// No outputs and no predict function - record error in scores
			for _, scorer := range scorers {
				result.Scores[scorer.Name()] = Score{
					Error:     fmt.Errorf("no outputs provided and no predict function configured"),
					Rationale: "Test case has no pre-generated outputs and no predict function was provided",
				}
			}
			return result
		}

		// Call predict function
		outputs, trace, err := cfg.predictFunc(ctx, tc.Inputs)
		if err != nil {
			// Predict failed - record error in all scores
			for _, scorer := range scorers {
				result.Scores[scorer.Name()] = Score{
					Error:     err,
					Rationale: fmt.Sprintf("Predict function failed: %v", err),
				}
			}
			return result
		}

		result.Outputs = outputs
		result.Trace = trace
	} else {
		// Use pre-generated outputs
		result.Outputs = tc.Outputs
		result.Trace = nil
	}

	// Run scorers
	for _, scorer := range scorers {
		score := e.runScorer(ctx, scorer, ScorerInput{
			Outputs:      result.Outputs,
			Expectations: tc.Expectations,
			Trace:        result.Trace,
			Inputs:       tc.Inputs,
		})
		result.Scores[scorer.Name()] = score
	}

	return result
}

// runScorer executes a scorer with panic recovery.
func (e *Evaluator) runScorer(ctx context.Context, scorer Scorer, input ScorerInput) Score {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			// This will be overwritten if we don't return from the panic path
			// But this is for safety
		}
	}()

	// Check context before running
	select {
	case <-ctx.Done():
		return Score{
			Error:     ctx.Err(),
			Rationale: "Context cancelled before scorer could run",
		}
	default:
	}

	// Run scorer with panic recovery
	var score Score
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				score = Score{
					Error:     fmt.Errorf("scorer panicked: %v", r),
					Rationale: fmt.Sprintf("Scorer %s panicked during execution: %v", scorer.Name(), r),
				}
			}
		}()

		score, err = scorer.Score(ctx, input)
		if err != nil {
			score = Score{
				Error:     err,
				Rationale: fmt.Sprintf("Scorer returned error: %v", err),
			}
		}
	}()

	return score
}

// computeSummary calculates aggregated statistics per scorer.
func (e *Evaluator) computeSummary(results *Results, scorers []Scorer) {
	// Initialize stats for each scorer
	for _, scorer := range scorers {
		results.Summary[scorer.Name()] = e.computeScorerStats(scorer.Name(), results.TestCases)
	}
}

// computeScorerStats calculates statistics for a single scorer.
func (e *Evaluator) computeScorerStats(scorerName string, testCases []TestCaseResult) ScorerStats {
	stats := ScorerStats{
		Count: len(testCases),
	}

	var values []float64
	var passCount int
	var errorCount int

	for _, tc := range testCases {
		score, exists := tc.Scores[scorerName]
		if !exists {
			continue
		}

		// Count errors
		if score.Error != nil {
			errorCount++
			continue
		}

		// Convert score value to float64 for aggregation
		if numVal, ok := convertToFloat64(score.Value); ok {
			values = append(values, numVal)
			if numVal > 0.5 { // For boolean scores, this counts true (1.0) as pass
				passCount++
			}
		}
	}

	stats.ErrorCount = errorCount
	if stats.Count > 0 {
		stats.ErrorRate = float64(errorCount) / float64(stats.Count)
	}

	nonErrorCount := len(values)
	if nonErrorCount > 0 {
		// Compute mean
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		stats.Mean = sum / float64(nonErrorCount)
		stats.PassRate = float64(passCount) / float64(nonErrorCount)

		// Compute standard deviation (sample std with Bessel's correction)
		if nonErrorCount > 1 {
			variance := 0.0
			for _, v := range values {
				diff := v - stats.Mean
				variance += diff * diff
			}
			variance /= float64(nonErrorCount - 1)
			stats.StdDev = math.Sqrt(variance)
		}
	}

	return stats
}

// convertToFloat64 converts a score value to float64 for aggregation.
func convertToFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case bool:
		if val {
			return 1.0, true
		}
		return 0.0, true
	default:
		// String and other types are skipped
		return 0.0, false
	}
}
