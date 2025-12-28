// Package eval provides an evaluation framework for scoring AI agent outputs.
//
// The eval package allows you to:
//   - Define scorers that evaluate outputs against expectations
//   - Load datasets from JSON files
//   - Run evaluations sequentially or in parallel
//   - Aggregate results with statistical summaries
//
// Basic usage:
//
//	// Create a scorer
//	scorer := &MyScorer{}
//
//	// Load a dataset
//	dataset, err := eval.LoadDataset("testcases.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create an evaluator
//	evaluator := eval.NewEvaluator()
//
//	// Run evaluation
//	results, err := evaluator.Run(ctx, dataset, []eval.Scorer{scorer})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access results
//	for _, tc := range results.TestCases {
//	    fmt.Printf("Test case: %v\n", tc.TestCase.ID)
//	    for scorerName, score := range tc.Scores {
//	        fmt.Printf("  %s: %v\n", scorerName, score.Value)
//	    }
//	}
//
// For more advanced usage with parallel execution and timeouts:
//
//	results, err := evaluator.Run(ctx, dataset, scorers,
//	    eval.WithParallelism(10),
//	    eval.WithTimeout(30*time.Second),
//	    eval.WithPredict(myPredictFunc),
//	)
package eval
