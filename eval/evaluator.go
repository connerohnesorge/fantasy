package eval

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"charm.land/fantasy/mlflowclient"
)

// PredictFunc generates outputs from inputs.
// Returns outputs map and optional trace for agent-specific scorers.
type PredictFunc func(ctx context.Context, inputs map[string]any) (outputs map[string]any, trace *Trace, err error)

// Evaluator runs scorers against datasets.
type Evaluator struct {
	mlflowExporter *MLflowExporter
}

// NewEvaluator creates an Evaluator with optional configuration.
func NewEvaluator(opts ...EvalOption) *Evaluator {
	e := &Evaluator{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// EvalOption configures the Evaluator at creation time.
type EvalOption func(*Evaluator)

// WithMLflowExport enables automatic export of evaluation results to MLflow.
// The client parameter should be a configured MLflow client.
func WithMLflowExport(exporter *MLflowExporter) EvalOption {
	return func(e *Evaluator) {
		e.mlflowExporter = exporter
	}
}

// Run executes evaluation on the given dataset with the specified scorers.
func (e *Evaluator) Run(ctx context.Context, dataset *Dataset, scorers []Scorer, opts ...RunOption) (*Results, error) {
	// Parse run options
	cfg := &runConfig{
		parallelism: 1, // Default: sequential
		timeout:     0, // Default: no timeout
		predictFunc: nil,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Initialize results
	results := &Results{
		TestCases:  make([]TestCaseResult, len(dataset.TestCases)),
		Summary:    make(map[string]ScorerStats),
		Errors:     []EvalError{},
		StartTime:  time.Now(),
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

	// Aggregate results
	e.aggregateResults(results, scorers)

	// Export to MLflow if configured
	if e.mlflowExporter != nil {
		if err := e.exportToMLflow(ctx, dataset, results, scorers, cfg); err != nil {
			results.Errors = append(results.Errors, EvalError{
				Phase:   "export",
				Message: "failed to export results to MLflow",
				Cause:   err,
			})
		}
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
				Message: "evaluation cancelled",
				Cause:   ctx.Err(),
			})
			return
		default:
		}

		// Execute test case
		tcResult := e.executeTestCase(ctx, tc, scorers, cfg)
		results.TestCases[i] = tcResult
	}
}

// runParallel executes test cases in parallel with configured worker count.
func (e *Evaluator) runParallel(ctx context.Context, dataset *Dataset, scorers []Scorer, cfg *runConfig, results *Results) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.parallelism)

	// Process each test case
	for i := range dataset.TestCases {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			// Execute test case
			tc := dataset.TestCases[idx]
			tcResult := e.executeTestCase(ctx, tc, scorers, cfg)
			results.TestCases[idx] = tcResult
		}(i)
	}

	wg.Wait()
}

// executeTestCase runs a single test case through all scorers.
func (e *Evaluator) executeTestCase(ctx context.Context, tc TestCase, scorers []Scorer, cfg *runConfig) TestCaseResult {
	result := TestCaseResult{
		TestCase: tc,
		Scores:   make(map[string]Score),
	}

	// Generate outputs if needed
	if tc.Outputs == nil || len(tc.Outputs) == 0 {
		if cfg.predictFunc == nil {
			// No outputs and no predict function - error
			for _, scorer := range scorers {
				result.Scores[scorer.Name()] = Score{
					Error:     fmt.Errorf("no outputs provided and no predict function configured"),
					Rationale: "Test case requires outputs but none were provided",
				}
			}
			return result
		}

		// Apply timeout for predict if configured
		predictCtx := ctx
		if cfg.timeout > 0 {
			var cancel context.CancelFunc
			predictCtx, cancel = context.WithTimeout(ctx, cfg.timeout)
			defer cancel()
		}

		// Run predict function with panic recovery
		var outputs map[string]any
		var trace *Trace
		var predictErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					predictErr = fmt.Errorf("predict function panicked: %v", r)
					log.Printf("ERROR: Predict function panic: %v", r)
				}
			}()
			outputs, trace, predictErr = cfg.predictFunc(predictCtx, tc.Inputs)
		}()

		if predictErr != nil {
			// Predict failed - mark all scorers as errored
			for _, scorer := range scorers {
				result.Scores[scorer.Name()] = Score{
					Error:     predictErr,
					Rationale: "Predict function failed",
				}
			}
			return result
		}

		result.Outputs = outputs
		result.Trace = trace
	} else {
		result.Outputs = tc.Outputs
		result.Trace = nil // No trace for pre-generated outputs
	}

	// Run each scorer
	for _, scorer := range scorers {
		score := e.executeScorer(ctx, scorer, result.Outputs, tc, result.Trace, cfg)
		result.Scores[scorer.Name()] = score
	}

	return result
}

// executeScorer runs a single scorer with timeout and panic recovery.
func (e *Evaluator) executeScorer(ctx context.Context, scorer Scorer, outputs map[string]any, tc TestCase, trace *Trace, cfg *runConfig) Score {
	// Apply timeout if configured
	scorerCtx := ctx
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		scorerCtx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	// Create scorer input
	input := ScorerInput{
		Outputs:      outputs,
		Expectations: tc.Expectations,
		Trace:        trace,
		Inputs:       tc.Inputs,
	}

	// Execute scorer with panic recovery
	var score Score
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("scorer panicked: %v", r)
				log.Printf("ERROR: Scorer %s panic: %v", scorer.Name(), r)
			}
		}()
		score, err = scorer.Score(scorerCtx, input)
	}()

	if err != nil {
		return Score{
			Error:     err,
			Rationale: fmt.Sprintf("Scorer failed: %v", err),
		}
	}

	return score
}

// aggregateResults computes summary statistics for all scorers.
func (e *Evaluator) aggregateResults(results *Results, scorers []Scorer) {
	for _, scorer := range scorers {
		stats := e.computeScorerStats(results.TestCases, scorer.Name())
		results.Summary[scorer.Name()] = stats
	}
}

// computeScorerStats calculates statistics for a single scorer.
func (e *Evaluator) computeScorerStats(testCases []TestCaseResult, scorerName string) ScorerStats {
	stats := ScorerStats{
		Count: len(testCases),
	}

	var numericValues []float64
	var passCount int
	var errorCount int

	for _, tc := range testCases {
		score, ok := tc.Scores[scorerName]
		if !ok {
			continue
		}

		if score.Error != nil {
			errorCount++
			continue
		}

		// Handle different value types
		switch v := score.Value.(type) {
		case bool:
			if v {
				passCount++
				numericValues = append(numericValues, 1.0)
			} else {
				numericValues = append(numericValues, 0.0)
			}
		case float64:
			numericValues = append(numericValues, v)
			if v > 0.5 { // Consider > 0.5 as pass for numeric scores
				passCount++
			}
		case int:
			numericValues = append(numericValues, float64(v))
			if v > 0 {
				passCount++
			}
		case int64:
			numericValues = append(numericValues, float64(v))
			if v > 0 {
				passCount++
			}
		case string:
			// String scores don't contribute to numeric stats
			// Could be enhanced to parse or handle categorical values
		default:
			log.Printf("WARNING: Scorer %s returned unsupported value type: %T", scorerName, v)
		}
	}

	// Compute statistics
	stats.ErrorCount = errorCount
	if stats.Count > 0 {
		stats.ErrorRate = float64(errorCount) / float64(stats.Count)
		stats.PassRate = float64(passCount) / float64(stats.Count)
	}

	if len(numericValues) > 0 {
		// Mean
		var sum float64
		for _, v := range numericValues {
			sum += v
		}
		stats.Mean = sum / float64(len(numericValues))

		// Standard deviation (sample std)
		if len(numericValues) > 1 {
			var sumSquaredDiff float64
			for _, v := range numericValues {
				diff := v - stats.Mean
				sumSquaredDiff += diff * diff
			}
			stats.StdDev = math.Sqrt(sumSquaredDiff / float64(len(numericValues)-1))
		}
	}

	return stats
}

// RunOption configures a specific evaluation run.
type RunOption func(*runConfig)

type runConfig struct {
	parallelism    int
	timeout        time.Duration
	predictFunc    PredictFunc
	experimentID   string
	runID          string
	llmJudgeScorers map[string]bool // Set of scorer names that are LLM judges
}

// WithParallelism sets the number of parallel workers.
// Default is 1 (sequential execution).
func WithParallelism(n int) RunOption {
	return func(cfg *runConfig) {
		if n < 1 {
			n = 1
		}
		cfg.parallelism = n
	}
}

// WithTimeout sets the per-test-case timeout.
// Default is no timeout.
func WithTimeout(d time.Duration) RunOption {
	return func(cfg *runConfig) {
		cfg.timeout = d
	}
}

// WithPredict sets the prediction function for generating outputs.
// Required if test cases don't have pre-generated outputs.
func WithPredict(fn PredictFunc) RunOption {
	return func(cfg *runConfig) {
		cfg.predictFunc = fn
	}
}

// WithExperimentID sets the MLflow experiment ID for result export.
// Only used if the evaluator was created with WithMLflowExport.
func WithExperimentID(experimentID string) RunOption {
	return func(cfg *runConfig) {
		cfg.experimentID = experimentID
	}
}

// WithRunID sets the MLflow run ID for result export.
// If not specified, a new run will be created.
// Only used if the evaluator was created with WithMLflowExport.
func WithRunID(runID string) RunOption {
	return func(cfg *runConfig) {
		cfg.runID = runID
	}
}

// WithLLMJudgeScorers marks which scorers are LLM judges for proper assessment source type.
// Only used if the evaluator was created with WithMLflowExport.
func WithLLMJudgeScorers(scorerNames ...string) RunOption {
	return func(cfg *runConfig) {
		if cfg.llmJudgeScorers == nil {
			cfg.llmJudgeScorers = make(map[string]bool)
		}
		for _, name := range scorerNames {
			cfg.llmJudgeScorers[name] = true
		}
	}
}

// exportToMLflow exports evaluation results to MLflow.
// This includes creating/updating a run with metrics, params, and assessments.
func (e *Evaluator) exportToMLflow(ctx context.Context, dataset *Dataset, results *Results, scorers []Scorer, cfg *runConfig) error {
	if e.mlflowExporter == nil || e.mlflowExporter.client == nil {
		return fmt.Errorf("MLflow exporter not configured")
	}

	client := e.mlflowExporter.client

	// Create or use existing run
	runID := cfg.runID
	if runID == "" && cfg.experimentID != "" {
		// Create a new run
		run, err := client.CreateRun(ctx, cfg.experimentID, func(opts *mlflowclient.CreateRunOptions) {
			opts.RunName = "evaluation-" + dataset.Name
		})
		if err != nil {
			return fmt.Errorf("failed to create MLflow run: %w", err)
		}
		runID = run.GetInfo().GetRunId()
	}

	if runID == "" {
		return fmt.Errorf("no run ID provided and no experiment ID to create a run")
	}

	// Log dataset params
	params := DatasetToParams(dataset)
	if len(params) > 0 {
		if err := client.LogBatch(ctx, runID, nil, params, nil); err != nil {
			log.Printf("WARNING: Failed to log dataset params to MLflow: %v", err)
		}
	}

	// Log metrics
	metrics := EvaluationResultsToMetrics(results)
	if len(metrics) > 0 {
		if err := client.LogBatch(ctx, runID, metrics, nil, nil); err != nil {
			log.Printf("WARNING: Failed to log metrics to MLflow: %v", err)
		}
	}

	// Create assessments for each test case with a trace
	for _, tcResult := range results.TestCases {
		// Skip test cases without traces
		if tcResult.Trace == nil {
			continue
		}

		// For now, we don't have a way to get the trace ID from the Trace placeholder
		// This will be implemented when the tracing package is complete
		// TODO: Extract trace ID from tcResult.Trace when tracing package is implemented
		log.Printf("WARNING: Assessment creation skipped - tracing package not yet implemented")
	}

	// Update run status to finished
	endTime := time.Now().UnixMilli()
	if _, err := client.UpdateRun(ctx, runID, "FINISHED", endTime); err != nil {
		log.Printf("WARNING: Failed to update run status: %v", err)
	}

	return nil
}
