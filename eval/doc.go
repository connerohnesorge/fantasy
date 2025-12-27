// Package eval provides evaluation framework for testing AI agents and models.
//
// The package supports running scorers against datasets to evaluate model performance.
// Both heuristic scorers (exact match, regex, JSON matching) and LLM-as-judge scorers
// are supported.
//
// Basic usage:
//
//	dataset := &eval.Dataset{
//	    Name: "my-test-suite",
//	    TestCases: []*eval.TestCase{
//	        {
//	            Inputs: map[string]any{"question": "What is 2+2?"},
//	            Expectations: map[string]any{"answer": "4"},
//	        },
//	    },
//	}
//
//	evaluator := eval.NewEvaluator()
//	results, err := evaluator.Run(ctx, dataset, []eval.Scorer{
//	    eval.ExactMatchScorer("answer"),
//	})
//
// With prediction function:
//
//	results, err := evaluator.Run(ctx, dataset, scorers,
//	    eval.WithPredict(func(ctx context.Context, inputs map[string]any) (map[string]any, *tracing.Trace, error) {
//	        // Call your model here
//	        return outputs, trace, nil
//	    }),
//	)
//
// Parallel evaluation:
//
//	results, err := evaluator.Run(ctx, dataset, scorers,
//	    eval.WithParallelism(10),
//	    eval.WithTimeout(30 * time.Second),
//	)
package eval
