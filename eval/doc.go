// Package eval provides an evaluation framework for testing AI agent outputs.
//
// The eval package implements a flexible evaluation system with:
//   - Go-native types that work without requiring an MLflow server
//   - Scorer interface for implementing custom evaluation criteria
//   - Built-in heuristic scorers (ExactMatch, Contains, Regex, JSONMatch, NumericRange)
//   - Agent-specific scorers (ToolCallTrajectory, StepValidation)
//   - LLM-as-judge scorers (Correctness, Guidelines, Relevance, Groundedness)
//   - Configurable parallelism with sequential (default) and parallel execution modes
//   - Timeout handling per test case
//   - Result aggregation with summary statistics
//   - Optional MLflow export via converter functions
//
// Basic usage:
//
//	// Define test cases
//	dataset := &eval.Dataset{
//		Name: "my-test-suite",
//		TestCases: []eval.TestCase{
//			{
//				Inputs:       map[string]any{"question": "What is 2+2?"},
//				Expectations: map[string]any{"answer": "4"},
//			},
//		},
//	}
//
//	// Create scorers
//	scorers := []eval.Scorer{
//		eval.NewExactMatchScorer("answer"),
//	}
//
//	// Run evaluation
//	evaluator := eval.NewEvaluator()
//	results, err := evaluator.Run(ctx, dataset, scorers,
//		eval.WithPredict(myPredictFunc),
//		eval.WithParallelism(10),
//		eval.WithTimeout(30*time.Second),
//	)
package eval
