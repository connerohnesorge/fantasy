# Fantasy Evaluation Framework

A flexible evaluation framework for scoring Fantasy agent outputs using heuristic and LLM-based scorers.

## Overview

The eval package provides:

- **Scorer Interface**: Extensible scoring system for custom metrics
- **Built-in Heuristic Scorers**: ExactMatch, Contains, Regex, JSONMatch, NumericRange
- **LLM-as-Judge Scorers**: Correctness, Relevance, Groundedness, Guidelines
- **Dataset Management**: JSON-based test case datasets
- **MLflow Integration**: Export results as metrics and assessments
- **Parallel Execution**: Configurable worker count for concurrent evaluation

## Quick Start

```go
import (
    "context"
    "charm.land/fantasy/eval"
)

// Create test dataset
dataset := eval.NewDataset("my-tests",
    eval.NewTestCase(
        map[string]any{"prompt": "What is 2+2?"},
        map[string]any{"expected": "4"},
    ),
)

// Define scorers
scorers := []eval.Scorer{
    eval.ExactMatchScorer("output"),
    eval.ContainsScorer("output"),
}

// Run evaluation
evaluator := eval.NewEvaluator()
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(func(ctx context.Context, inputs map[string]any) (map[string]any, any, error) {
        // Call your agent here
        return map[string]any{"output": "4"}, nil, nil
    }),
)

// Check results
for name, stats := range results.Summary {
    fmt.Printf("%s: mean=%.2f, pass_rate=%.2f\n", name, stats.Mean, stats.PassRate)
}
```

## Scorers

### Heuristic Scorers

```go
// Exact string match - compares output field to "expected" expectation
scorer := eval.ExactMatchScorer("output")

// Substring contains - checks if output contains expected
scorer := eval.ContainsScorer("output")

// Regex pattern match - matches expected pattern against output
scorer := eval.RegexScorer("output")

// JSON structural match
scorer := eval.JSONMatchScorer("output", eval.JSONMatchOptions{
    IgnoreExtraKeys: true,
})

// Numeric range validation
scorer := eval.NumericRangeScorer("output", 0.0, 1.0)
```

### LLM-as-Judge Scorers

```go
// Correctness evaluation
scorer := eval.NewCorrectnessScorer(model)

// Relevance evaluation
scorer := eval.NewRelevanceScorer(model)

// Groundedness (fact-checking against context)
scorer := eval.NewGroundednessScorer(model)

// Custom guidelines compliance
scorer := eval.NewGuidelinesScorer(model, `
1. Response must be polite
2. Response must not contain PII
3. Response must cite sources
`)
```

### Custom Scorers

```go
type MyScorer struct{}

func (s *MyScorer) Name() string { return "my_scorer" }

func (s *MyScorer) Score(ctx context.Context, input eval.ScorerInput) (eval.Score, error) {
    output := input.Outputs["output"].(string)
    expected := input.Expectations["expected"].(string)

    score := calculateSimilarity(output, expected)

    return eval.Score{
        Value:     score,
        Rationale: fmt.Sprintf("Similarity score: %.2f", score),
    }, nil
}
```

## Datasets

### Creating Datasets

```go
// Programmatically
dataset := eval.NewDataset("qa-tests",
    eval.NewTestCase(
        map[string]any{"prompt": "Question 1"},
        map[string]any{"expected": "Answer 1"},
    ).WithTags(map[string]string{"category": "math"}),
)

// From JSON file
dataset, err := eval.LoadDataset("tests.json")

// Save to JSON
err := eval.SaveDataset("tests.json", dataset)
```

### Dataset JSON Format

```json
{
  "name": "qa-tests",
  "test_cases": [
    {
      "inputs": {"prompt": "What is 2+2?"},
      "expectations": {"expected": "4"},
      "tags": {"category": "math"}
    }
  ]
}
```

## Running Evaluations

```go
evaluator := eval.NewEvaluator()

results, err := evaluator.Run(ctx, dataset, scorers,
    // Parallel execution
    eval.WithParallelism(4),

    // Per-test timeout
    eval.WithTimeout(30 * time.Second),

    // Prediction function (generate outputs)
    eval.WithPredict(predictFunc),
)
```

## Results

```go
// Overall summary
for scorerName, stats := range results.Summary {
    fmt.Printf("%s:\n", scorerName)
    fmt.Printf("  Mean: %.2f\n", stats.Mean)
    fmt.Printf("  Pass Rate: %.2f\n", stats.PassRate)
    fmt.Printf("  Std Dev: %.2f\n", stats.StdDev)
    fmt.Printf("  Error Rate: %.2f\n", stats.ErrorRate)
}

// Per-test-case results
for i, tcResult := range results.TestCases {
    for scorerName, score := range tcResult.Scores {
        fmt.Printf("Test %d - %s: %v (%s)\n",
            i, scorerName, score.Value, score.Rationale)
    }
}
```

## MLflow Integration

Export evaluation results to MLflow for tracking and visualization:

```go
import "charm.land/fantasy/mlflowclient"

client, _ := mlflowclient.New("http://localhost:5000")

// Create exporter
exporter := eval.NewMLflowExporter(client, "experiment-id")

// Run evaluation and export
results, _ := evaluator.Run(ctx, dataset, scorers)

// Export to a run
run, _ := eval.DatasetToRun(ctx, client, "experiment-id", dataset, "eval-run")
exporter.ExportResults(ctx, results, run.Info.RunID)

// Or export as assessments to a trace
exporter.ExportAssessments(ctx, results, traceID)
```

### Metrics Exported

- `eval_duration_seconds`: Total evaluation time
- `eval_total_tests`: Number of test cases
- `eval_<scorer>_mean`: Mean score per scorer
- `eval_<scorer>_pass_rate`: Pass rate per scorer
- `eval_<scorer>_std_dev`: Standard deviation per scorer
- `eval_<scorer>_error_rate`: Error rate per scorer
