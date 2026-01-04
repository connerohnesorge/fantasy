# Eval Package

The `eval` package provides a comprehensive evaluation framework for testing and scoring AI model outputs. It supports both heuristic-based and LLM-as-Judge scoring methods, with built-in MLflow integration for tracking and analysis.

## Overview

The evaluation framework follows this flow:

1. **Define a Dataset** with test cases (inputs, expected outputs)
2. **Create Scorers** to evaluate outputs (heuristic or LLM-based)
3. **Run Evaluation** with an Evaluator
4. **Analyze Results** with aggregated statistics and individual scores

## Core Types

### Score

The `Score` type represents the result of a scorer evaluation:

```go
type Score struct {
    Value     any            // bool, float64, int, or string
    Rationale string         // Explanation of the score
    Metadata  map[string]any // Additional structured information
    Error     error          // Non-nil if scorer failed
}
```

### Dataset and TestCase

```go
type TestCase struct {
    ID           string            // Optional unique identifier
    Inputs       map[string]any    // Required: input data
    Expectations map[string]any    // Expected output values
    Outputs      map[string]any    // Optional: pre-generated outputs
    Tags         map[string]string // Optional: metadata tags
}

type Dataset struct {
    Name      string
    TestCases []TestCase
    Metadata  map[string]any
}
```

Load a dataset from JSON:

```go
dataset, err := eval.LoadDataset("path/to/dataset.json")
if err != nil {
    log.Fatal(err)
}
```

### Scorer Interface

All scorers implement the `Scorer` interface:

```go
type Scorer interface {
    Name() string
    Score(ctx context.Context, input ScorerInput) (Score, error)
}

type ScorerInput struct {
    Outputs      map[string]any // Generated outputs to evaluate
    Expectations map[string]any // Expected values for comparison
    Trace        any            // Optional trace data (*tracing.Trace)
    Inputs       map[string]any // Original inputs (for context in scoring)
}
```

## Built-in Scorers

### Heuristic Scorers

#### ExactMatch

Compares output to expected string with exact equality:

```go
scorer := eval.NewExactMatch()
```

#### Contains

Checks if output contains expected substring:

```go
// Case-sensitive (default)
scorer := eval.NewContains()

// Case-insensitive
scorer := eval.NewContains(eval.WithCaseInsensitive(true))
```

#### Regex

Validates output against a regex pattern:

```go
scorer, err := eval.NewRegex(`\d{3}-\d{3}-\d{4}`) // Phone number pattern
if err != nil {
    log.Fatal(err)
}
```

#### JSONMatch

Compares JSON structures with flexible options:

```go
// Default: strict matching
scorer := eval.NewJSONMatch()

// Ignore array order
scorer := eval.NewJSONMatch(eval.WithIgnoreOrder(true))

// Allow extra fields in output
scorer := eval.NewJSONMatch(eval.WithIgnoreExtraFields(true))

// Loose type comparison (e.g., 1 == "1")
scorer := eval.NewJSONMatch(eval.WithIgnoreTypes(true))
```

#### NumericRange

Checks if a numeric value is within a specified range:

```go
// Default: inclusive on both bounds [0.0, 1.0]
scorer := eval.NewNumericRange(0.0, 1.0)

// Exclusive bounds (0.0, 1.0)
scorer := eval.NewNumericRange(0.0, 1.0,
    eval.WithMinInclusive(false),
    eval.WithMaxInclusive(false),
)
```

#### ToolCallTrajectory

Validates tool call sequences (for agentic systems):

```go
// Contains: expected is subsequence of actual
scorer := eval.NewToolCallTrajectory()

// Exact match: sequences must match exactly
scorer := eval.NewToolCallTrajectory(eval.WithExactMatch(true))
```

Expected format:
```json
{
  "expectations": {
    "tool_calls": ["search", "summarize", "respond"]
  }
}
```

#### StepValidation

Validates step count and content:

```go
scorer, err := eval.NewStepValidation(
    eval.WithMinSteps(2),
    eval.WithMaxSteps(5),
    eval.WithStepPatterns([]string{`\bsearch\b`, `\banalyze\b`}),
    eval.WithAllMatch(false), // At least one step must match
)
```

### LLM Judge Scorers

All LLM judges return scores normalized to 0-1 range (from original 1-5 scale).

#### Correctness

Evaluates answer correctness compared to expected answer:

```go
scorer := eval.NewCorrectnessJudge(model)
```

#### Guidelines

Evaluates outputs against custom guidelines:

```go
guidelines := `
1. Response must be concise (under 100 words)
2. Must use professional tone
3. Must include actionable recommendations
`
scorer := eval.NewGuidelinesJudge(model, guidelines)
```

#### Relevance

Evaluates response relevance to input/context:

```go
scorer := eval.NewRelevanceJudge(model)
```

#### Groundedness

Evaluates factual grounding in provided context (hallucination detection):

```go
scorer := eval.NewGroundednessJudge(model)
```

Expected format:
```json
{
  "inputs": {
    "context": "Source material for grounding..."
  }
}
```

#### Helpfulness

Evaluates how helpful and actionable a response is:

```go
scorer := eval.NewHelpfulnessJudge(model)
```

#### Tool Usage

Evaluates whether an agent used tools correctly:

```go
scorer := eval.NewToolUsageJudge(model)
```

#### Custom LLM Judge

Create a custom LLM judge:

```go
scorer := eval.NewLLMJudge("custom_judge", eval.JudgeConfig{
    Model:          model,
    SystemPrompt:   "You are evaluating...",
    PromptTemplate: "# Output:\n{{.Output}}\n\n# Expected:\n{{.Expected}}",
    Temperature:    fantasy.Float64(0.0),
    MaxTokens:      fantasy.Int64(1024),
    MaxRetries:     3,
})
```

## Evaluator Usage

### Basic Evaluation

```go
// Create evaluator
evaluator := eval.NewEvaluator()

// Define scorers
scorers := []eval.Scorer{
    eval.NewExactMatch(),
    eval.NewCorrectnessJudge(model),
}

// Run evaluation
results, err := evaluator.Run(ctx, dataset, scorers)
if err != nil {
    log.Fatal(err)
}

// Access results
for scorerName, stats := range results.Summary {
    fmt.Printf("%s: Pass Rate=%.2f%%, Mean=%.2f\n",
        scorerName, stats.PassRate*100, stats.Mean)
}
```

### With Predict Function

Generate outputs dynamically during evaluation:

```go
predictFunc := func(ctx context.Context, inputs map[string]any) (
    outputs map[string]any, trace any, err error,
) {
    // Your model/agent logic here
    prompt := inputs["question"].(string)
    response, err := model.Generate(ctx, prompt)
    if err != nil {
        return nil, nil, err
    }

    return map[string]any{
        "output": response,
    }, nil, nil
}

results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
)
```

### Parallel Execution

Speed up evaluation with parallel workers:

```go
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithParallelism(4), // 4 concurrent workers
)
```

### With Timeout

Set per-test-case timeout:

```go
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithTimeout(30 * time.Second),
)
```

### MLflow Export

Automatically export results to MLflow:

```go
// Create MLflow client
mlflowClient := mlflow.NewClient(mlflow.ClientConfig{
    TrackingURI: "http://localhost:5000",
})

// Create evaluator with MLflow export
evaluator := eval.NewEvaluator(
    eval.WithMLflowExport(mlflowClient, experimentID),
)

// Run evaluation - results are automatically exported
results, err := evaluator.Run(ctx, dataset, scorers)
```

This will:
- Create an MLflow run for the evaluation
- Log test case assessments with trace linkage
- Log aggregate metrics (pass rate, mean score, error rate)

## Results Structure

```go
type Results struct {
    TestCases  []TestCaseResult       // Individual results per test case
    Summary    map[string]ScorerStats // Aggregated stats per scorer name
    Errors     []EvalError            // Evaluation-level errors
    StartTime  time.Time              // When evaluation started
    EndTime    time.Time              // When evaluation completed
    TotalTests int                    // Total number of test cases processed
}

type TestCaseResult struct {
    TestCase TestCase         // The original test case
    Outputs  map[string]any   // Generated outputs (if PredictFunc was run)
    Scores   map[string]Score // Scores keyed by scorer name
    Trace    any              // Captured trace (*tracing.Trace)
}

type ScorerStats struct {
    PassRate   float64 // Percentage of passing scores (0.0-1.0)
    Mean       float64 // Mean score value (for numeric scores)
    StdDev     float64 // Standard deviation (for numeric scores)
    ErrorRate  float64 // Percentage of errored scores (0.0-1.0)
    ErrorCount int     // Number of test cases where scorer errored
    Count      int     // Total number of scores computed
}
```

### Accessing Results

```go
// Iterate through test case results
for i, tcResult := range results.TestCases {
    fmt.Printf("Test Case %d:\n", i)
    for scorerName, score := range tcResult.Scores {
        if score.Error != nil {
            fmt.Printf("  %s: ERROR - %v\n", scorerName, score.Error)
            continue
        }
        fmt.Printf("  %s: %v - %s\n", scorerName, score.Value, score.Rationale)
    }
}

// View summary statistics
for scorerName, stats := range results.Summary {
    fmt.Printf("\n%s Statistics:\n", scorerName)
    fmt.Printf("  Pass Rate: %.2f%%\n", stats.PassRate*100)
    fmt.Printf("  Mean Score: %.4f\n", stats.Mean)
    fmt.Printf("  Std Dev: %.4f\n", stats.StdDev)
    fmt.Printf("  Error Rate: %.2f%%\n", stats.ErrorRate*100)
    fmt.Printf("  Total Count: %d\n", stats.Count)
}

// Check for evaluation errors
if len(results.Errors) > 0 {
    fmt.Println("\nEvaluation Errors:")
    for _, evalErr := range results.Errors {
        fmt.Printf("  [%s] %s: %v\n", evalErr.Phase, evalErr.Message, evalErr.Cause)
    }
}
```

## Error Handling

### Score Errors

Individual scorer failures are captured in the `Score.Error` field and don't stop evaluation:

```go
for scorerName, score := range tcResult.Scores {
    if score.Error != nil {
        // Handle scorer failure
        log.Printf("Scorer %s failed: %v", scorerName, score.Error)
        continue
    }
    // Process successful score
}
```

### Evaluation Errors

System-level errors are collected in `Results.Errors`:

```go
if len(results.Errors) > 0 {
    for _, evalErr := range results.Errors {
        switch evalErr.Phase {
        case "setup":
            // Dataset validation or initialization errors
        case "predict":
            // PredictFunc failures
        case "score":
            // Scorer execution errors (rare, usually caught in Score.Error)
        case "export":
            // MLflow export errors
        }
    }
}
```

### Context Cancellation

The evaluator respects context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

results, err := evaluator.Run(ctx, dataset, scorers)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Evaluation timed out")
    }
}
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "charm.land/fantasy/eval"
    "charm.land/fantasy"
)

func main() {
    ctx := context.Background()

    // Load dataset
    dataset, err := eval.LoadDataset("testdata/qa_dataset.json")
    if err != nil {
        log.Fatal(err)
    }

    // Initialize model
    model := fantasy.NewClaude(fantasy.ClaudeConfig{
        APIKey: os.Getenv("ANTHROPIC_API_KEY"),
        Model:  fantasy.ModelSonnet35,
    })

    // Create scorers
    scorers := []eval.Scorer{
        eval.NewExactMatch(),
        eval.NewContains(eval.WithCaseInsensitive(true)),
        eval.NewCorrectnessJudge(model),
        eval.NewRelevanceJudge(model),
    }

    // Define predict function
    predictFunc := func(ctx context.Context, inputs map[string]any) (
        map[string]any, any, error,
    ) {
        question := inputs["question"].(string)
        response, err := model.Generate(ctx, question)
        if err != nil {
            return nil, nil, err
        }
        return map[string]any{"output": response}, nil, nil
    }

    // Create evaluator
    evaluator := eval.NewEvaluator()

    // Run evaluation
    results, err := evaluator.Run(ctx, dataset, scorers,
        eval.WithPredict(predictFunc),
        eval.WithParallelism(4),
        eval.WithTimeout(30*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Print summary
    fmt.Printf("Evaluation completed in %v\n",
        results.EndTime.Sub(results.StartTime))
    fmt.Printf("Total tests: %d\n\n", results.TotalTests)

    for scorerName, stats := range results.Summary {
        fmt.Printf("%s:\n", scorerName)
        fmt.Printf("  Pass Rate: %.2f%%\n", stats.PassRate*100)
        fmt.Printf("  Mean: %.4f\n", stats.Mean)
        fmt.Printf("  Error Rate: %.2f%%\n\n", stats.ErrorRate*100)
    }
}
```

## Best Practices

1. **Use appropriate scorers**: Heuristic scorers for objective metrics, LLM judges for subjective quality
2. **Set timeouts**: Prevent hung evaluations with `WithTimeout`
3. **Parallelize wisely**: Balance speed with API rate limits using `WithParallelism`
4. **Handle errors**: Check both `Score.Error` and `Results.Errors`
5. **Track with MLflow**: Use `WithMLflowExport` for experiment tracking and analysis
6. **Validate datasets**: Ensure datasets are valid before running long evaluations
7. **Use context**: Respect cancellation signals for clean shutdowns

## MLflow Integration

The eval package integrates seamlessly with MLflow for experiment tracking:

```go
// Setup MLflow
mlflowClient := mlflow.NewClient(mlflow.ClientConfig{
    TrackingURI: "http://localhost:5000",
})

// Create experiment
experimentID, _ := mlflowClient.CreateExperiment(ctx, "my-eval-experiment")

// Create evaluator with automatic export
evaluator := eval.NewEvaluator(
    eval.WithMLflowExport(mlflowClient, experimentID),
)

// Run evaluation - results automatically logged to MLflow
results, _ := evaluator.Run(ctx, dataset, scorers)
```

This automatically logs:
- Individual test case assessments linked to traces
- Aggregate metrics (pass rates, mean scores, error rates)
- Dataset metadata and evaluation duration
