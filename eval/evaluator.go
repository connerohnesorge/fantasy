package eval

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// Evaluator runs scorers against datasets.
type Evaluator struct {
	// No configuration needed at creation time
}

// NewEvaluator creates an Evaluator.
func NewEvaluator(opts ...EvalOption) *Evaluator {
	e := &Evaluator{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// EvalOption configures the Evaluator at creation time.
type EvalOption func(*Evaluator)

// RunOption configures a specific evaluation run.
type RunOption func(*runConfig)

type runConfig struct {
	parallelism         int
	timeout             time.Duration
	predict             PredictFunc
	mlflowClient        any    // *mlflowclient.Client - uses any to avoid import cycle
	mlflowExperimentID  string // MLflow experiment ID for export
}

// WithParallelism sets the number of parallel workers.
func WithParallelism(n int) RunOption {
	return func(c *runConfig) {
		c.parallelism = n
	}
}

// WithTimeout sets the per-test-case timeout.
func WithTimeout(d time.Duration) RunOption {
	return func(c *runConfig) {
		c.timeout = d
	}
}

// WithPredict sets the prediction function for generating outputs.
func WithPredict(fn PredictFunc) RunOption {
	return func(c *runConfig) {
		c.predict = fn
	}
}

// Run executes evaluation on the given dataset with the specified scorers.
func (e *Evaluator) Run(ctx context.Context, dataset *Dataset, scorers []Scorer, opts ...RunOption) (*Results, error) {
	cfg := &runConfig{
		parallelism: 1,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate dataset
	if err := dataset.Validate(); err != nil {
		return nil, err
	}

	// Check that we have outputs or a predict function
	for i, tc := range dataset.TestCases {
		if len(tc.Outputs) == 0 && cfg.predict == nil {
			return nil, &ValidationError{
				Field:   "TestCases",
				Message: "test case has no outputs and no predict function provided",
				Index:   i,
			}
		}
	}

	results := &Results{
		TestCases:  make([]TestCaseResult, len(dataset.TestCases)),
		Summary:    make(map[string]ScorerStats),
		StartTime:  time.Now(),
		TotalTests: len(dataset.TestCases),
	}

	// Run evaluation
	if cfg.parallelism <= 1 {
		e.runSequential(ctx, dataset, scorers, cfg, results)
	} else {
		e.runParallel(ctx, dataset, scorers, cfg, results)
	}

	results.EndTime = time.Now()

	// Compute summary statistics
	e.computeSummary(results, scorers)

	return results, nil
}

func (e *Evaluator) runSequential(ctx context.Context, dataset *Dataset, scorers []Scorer, cfg *runConfig, results *Results) {
	for i, tc := range dataset.TestCases {
		result := e.evaluateTestCase(ctx, tc, scorers, cfg)
		results.TestCases[i] = result
	}
}

func (e *Evaluator) runParallel(ctx context.Context, dataset *Dataset, scorers []Scorer, cfg *runConfig, results *Results) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.parallelism)

	for i, tc := range dataset.TestCases {
		wg.Add(1)
		go func(idx int, testCase *TestCase) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := e.evaluateTestCase(ctx, testCase, scorers, cfg)
			results.TestCases[idx] = result
		}(i, tc)
	}

	wg.Wait()
}

func (e *Evaluator) evaluateTestCase(ctx context.Context, tc *TestCase, scorers []Scorer, cfg *runConfig) TestCaseResult {
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

	// Get outputs
	outputs := tc.Outputs
	var trace any

	if cfg.predict != nil && len(outputs) == 0 {
		var err error
		outputs, trace, err = cfg.predict(ctx, tc.Inputs)
		if err != nil {
			// Record error for all scorers
			for _, scorer := range scorers {
				result.Scores[scorer.Name()] = Score{
					Error:     err,
					Rationale: fmt.Sprintf("predict function failed: %v", err),
				}
			}
			return result
		}
	}

	result.Outputs = outputs
	result.Trace = trace

	// Run each scorer
	input := ScorerInput{
		Outputs:      outputs,
		Expectations: tc.Expectations,
		Trace:        trace,
		Inputs:       tc.Inputs,
	}

	for _, scorer := range scorers {
		score := e.runScorer(ctx, scorer, input)
		result.Scores[scorer.Name()] = score
	}

	return result
}

func (e *Evaluator) runScorer(ctx context.Context, scorer Scorer, input ScorerInput) (score Score) {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			score = Score{
				Error:     fmt.Errorf("scorer panicked: %v", r),
				Rationale: fmt.Sprintf("scorer %s panicked: %v", scorer.Name(), r),
			}
		}
	}()

	score, err := scorer.Score(ctx, input)
	if err != nil {
		score.Error = err
		if score.Rationale == "" {
			score.Rationale = err.Error()
		}
	}

	return score
}

func (e *Evaluator) computeSummary(results *Results, scorers []Scorer) {
	for _, scorer := range scorers {
		name := scorer.Name()
		var stats ScorerStats

		var values []float64
		for _, tcResult := range results.TestCases {
			score, ok := tcResult.Scores[name]
			if !ok {
				continue
			}

			stats.Count++

			if score.Error != nil {
				stats.ErrorCount++
				continue
			}

			values = append(values, score.FloatValue())
		}

		if stats.Count > 0 {
			stats.ErrorRate = float64(stats.ErrorCount) / float64(stats.Count)
		}

		if len(values) > 0 {
			// Calculate mean
			var sum float64
			for _, v := range values {
				sum += v
			}
			stats.Mean = sum / float64(len(values))

			// Calculate pass rate (values == 1.0 or true)
			var passCount int
			for _, v := range values {
				if v >= 1.0 {
					passCount++
				}
			}
			stats.PassRate = float64(passCount) / float64(len(values))

			// Calculate standard deviation
			if len(values) > 1 {
				var sumSq float64
				for _, v := range values {
					diff := v - stats.Mean
					sumSq += diff * diff
				}
				stats.StdDev = math.Sqrt(sumSq / float64(len(values)-1))
			}
		}

		results.Summary[name] = stats
	}
}
