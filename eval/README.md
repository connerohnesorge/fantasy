# Evaluation Framework

A flexible evaluation system for testing AI agent outputs with Go-native types, built-in scorers, and optional MLflow export.

## Features

- **Go-native types**: Works without requiring an MLflow server (results can be local-only)
- **Multiple scorer types**: Heuristic, LLM-as-judge, and agent-specific scorers
- **Configurable parallelism**: Sequential (default) or parallel execution modes
- **Timeout handling**: Per test case timeouts
- **Result aggregation**: Summary statistics and detailed results
- **Optional MLflow export**: Export results to MLflow for tracking and visualization

## Installation

```bash
go get charm.land/fantasy/eval
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "charm.land/fantasy/eval"
)

func main() {
    ctx := context.Background()

    // Define test cases
    dataset := &eval.Dataset{
        Name: "math-qa-test",
        TestCases: []eval.TestCase{
            {
                Inputs:       map[string]any{"question": "What is 2+2?"},
                Expectations: map[string]any{"answer": "4"},
            },
            {
                Inputs:       map[string]any{"question": "What is 10*5?"},
                Expectations: map[string]any{"answer": "50"},
            },
        },
    }

    // Create scorers
    scorers := []eval.Scorer{
        eval.NewExactMatch("answer", "answer"),
    }

    // Define prediction function
    predictFunc := func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
        // Your agent or model inference logic here
        question := inputs["question"].(string)
        answer := generateAnswer(question) // Your implementation
        return map[string]any{"answer": answer}, nil
    }

    // Run evaluation
    evaluator := eval.NewEvaluator()
    results, err := evaluator.Run(ctx, dataset, scorers,
        eval.WithPredict(predictFunc),
        eval.WithTimeout(30*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Print results
    fmt.Printf("Total: %d, Passed: %d, Failed: %d\n",
        results.TotalCases, results.PassedCases, results.FailedCases)
}
```

## Built-in Scorers

### Heuristic Scorers

Heuristic scorers are code-based and don't require an LLM.

#### ExactMatch

Checks for exact string equality:

```go
// Basic usage
scorer := eval.NewExactMatch("answer", "answer")

// With options
scorer := eval.NewExactMatch("answer", "answer").
    WithCaseSensitive(false).
    WithName("case_insensitive_match")
```

#### Contains

Checks if output contains a substring:

```go
// Check if answer contains keyword
scorer := eval.NewContains("answer", "keyword").
    WithCaseSensitive(false)
```

#### Regex

Pattern matching with regular expressions:

```go
// Check if output matches pattern
scorer := eval.NewRegex("answer", `^\d{4}-\d{2}-\d{2}$`).
    WithName("date_format_check")
```

#### JSONMatch

Structural JSON comparison:

```go
// Compare JSON structures
scorer := eval.NewJSONMatch("response", "expected_response").
    WithIgnoreNulls(true). // Ignore null value differences
    WithName("json_structure_match")
```

#### NumericRange

Check if numeric value is within bounds:

```go
// Check if score is between 0 and 1
scorer := eval.NewNumericRange("score", 0.0, 1.0).
    WithInclusive(true).
    WithName("probability_range_check")
```

### LLM-as-Judge Scorers

LLM-as-judge scorers use an LLM to evaluate outputs. These require a Fantasy `LanguageModel`.

#### Correctness

Evaluates answer accuracy with reference:

```go
import (
    "charm.land/fantasy"
    "charm.land/fantasy/providers/openai"
)

// Create a model for judging
provider, _ := openai.New(openai.WithAPIKey("your-key"))
model, _ := provider.LanguageModel(ctx, "gpt-4")

// Create correctness scorer
scorer := eval.NewCorrectness(model).
    WithName("answer_correctness")

// Use with dataset that has reference answers
dataset := &eval.Dataset{
    TestCases: []eval.TestCase{
        {
            Inputs: map[string]any{
                "question": "What is the capital of France?",
            },
            Expectations: map[string]any{
                "reference": "Paris is the capital of France.",
            },
        },
    },
}
```

#### Guidelines

Custom criteria evaluation based on guidelines:

```go
scorer := eval.NewGuidelines(model, []string{
    "Response must be polite and professional",
    "Response must be concise (under 100 words)",
    "Response must address the question directly",
}).WithName("custom_guidelines")
```

#### Relevance

Evaluates relevance of response to input:

```go
scorer := eval.NewRelevance(model).
    WithName("response_relevance")

// Dataset with question-answer pairs
dataset := &eval.Dataset{
    TestCases: []eval.TestCase{
        {
            Inputs: map[string]any{
                "query": "How do I install Python?",
            },
            Expectations: map[string]any{
                "output": "To install Python, visit python.org...",
            },
        },
    },
}
```

#### Groundedness

Checks if response is grounded in provided context:

```go
scorer := eval.NewGroundedness(model).
    WithName("context_groundedness")

// Dataset with context and response
dataset := &eval.Dataset{
    TestCases: []eval.TestCase{
        {
            Inputs: map[string]any{
                "context": "The company was founded in 2020...",
            },
            Expectations: map[string]any{
                "output": "The company is 4 years old.",
            },
        },
    },
}
```

### Agent-Specific Scorers

Agent-specific scorers analyze agent execution traces. These require `*tracing.Trace` data.

#### ToolCallTrajectory

Validates tool call sequences:

```go
import "charm.land/fantasy/tracing"

// Expect specific tool call sequence
scorer := eval.NewToolCallTrajectory([]string{
    "search_database",
    "format_results",
    "send_email",
}).WithName("workflow_validation")

// Use with trace data
// Note: Trace must be passed via ScorerInput.Trace
```

#### StepValidation

Validates step count and content:

```go
// Expect 2-4 steps, with first step containing "analyze"
scorer := eval.NewStepValidation(2, 4).
    WithStepContentPattern(0, "analyze").
    WithName("step_structure_check")
```

## Evaluation Options

### Sequential vs Parallel Execution

```go
// Sequential (default) - easier to debug
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
)

// Parallel - faster for large datasets
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
    eval.WithParallelism(10), // 10 concurrent workers
)
```

### Timeout Handling

```go
// Set timeout per test case
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
    eval.WithTimeout(30*time.Second), // Each test case times out after 30s
)
```

### Custom Scorers

Implement the `Scorer` interface to create custom scorers:

```go
type CustomScorer struct {
    name string
}

func (s *CustomScorer) Name() string {
    return s.name
}

func (s *CustomScorer) Score(ctx context.Context, input eval.ScorerInput) (eval.Score, error) {
    // Your scoring logic here
    output := input.Outputs["answer"]
    expected := input.Expectations["answer"]

    // Return score
    return eval.Score{
        Value:     output == expected, // bool, float64, int, or string
        Rationale: "Custom scoring logic applied",
        Metadata: map[string]any{
            "custom_field": "custom_value",
        },
    }, nil
}
```

## Working with Datasets

### In-Memory Datasets

```go
dataset := &eval.Dataset{
    Name: "my-test-suite",
    TestCases: []eval.TestCase{
        {
            Inputs:       map[string]any{"question": "What is AI?"},
            Expectations: map[string]any{"answer": "Artificial Intelligence"},
        },
    },
}
```

### Loading from JSON

```go
// Load dataset from JSON file
dataset, err := eval.LoadDatasetFromJSON("testdata/dataset.json")
if err != nil {
    log.Fatal(err)
}
```

Example JSON format:

```json
{
  "name": "qa-evaluation",
  "test_cases": [
    {
      "inputs": {
        "question": "What is 2+2?"
      },
      "expectations": {
        "answer": "4"
      }
    }
  ]
}
```

## MLflow Integration

Export evaluation results to MLflow for tracking and visualization:

```go
import (
    "charm.land/fantasy/eval"
    "charm.land/fantasy/mlflowclient"
)

// Create MLflow client
mlflowClient := mlflowclient.New("http://localhost:5000")

// Run evaluation with MLflow export
evaluator := eval.NewEvaluator()
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
    eval.WithMLflowExport(mlflowClient, "experiment-123"),
)
```

### Converting Scores to Assessments

```go
import (
    "charm.land/fantasy/eval"
)

// Convert score to MLflow assessment
assessment := eval.ScoreToAssessment(score, "tr-123", "correctness_scorer")

// Create assessment in MLflow
_, err := mlflowClient.CreateAssessment(ctx, assessment)
```

## Results and Analysis

### Accessing Results

```go
results, err := evaluator.Run(ctx, dataset, scorers, ...)

// Summary statistics
fmt.Printf("Total: %d\n", results.TotalCases)
fmt.Printf("Passed: %d\n", results.PassedCases)
fmt.Printf("Failed: %d\n", results.FailedCases)
fmt.Printf("Error Rate: %.2f%%\n", results.ErrorRate*100)

// Detailed results
for i, result := range results.Results {
    fmt.Printf("\nTest Case %d:\n", i+1)
    fmt.Printf("  Status: %s\n", result.Status)

    for scorerName, score := range result.Scores {
        fmt.Printf("  %s: %v (%s)\n", scorerName, score.Value, score.Rationale)
    }
}
```

### Filtering Results

```go
// Get failed test cases
for i, result := range results.Results {
    if result.Status == "failed" {
        fmt.Printf("Failed test case %d: %+v\n", i+1, result.Inputs)
    }
}

// Get test cases with specific scorer failures
for i, result := range results.Results {
    if score, ok := result.Scores["correctness"]; ok {
        if score.Error != nil {
            fmt.Printf("Scorer error in test %d: %v\n", i+1, score.Error)
        }
    }
}
```

## Best Practices

### 1. Use Appropriate Scorers

```go
// Good: Use heuristic scorers for deterministic checks
exactMatch := eval.NewExactMatch("status", "status")

// Good: Use LLM judges for nuanced evaluation
relevance := eval.NewRelevance(model)

// Avoid: Using LLM judges for simple exact matches (expensive and slow)
```

### 2. Handle Timeouts

```go
// Set reasonable timeouts based on your model's response time
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
    eval.WithTimeout(60*time.Second), // Adjust based on model speed
)
```

### 3. Use Parallel Execution Wisely

```go
// Good: Parallel execution for large datasets with I/O-bound operations
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
    eval.WithParallelism(10),
)

// Caution: Consider API rate limits when using parallel execution with LLM judges
// May need to reduce parallelism to avoid rate limiting
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
    eval.WithParallelism(2), // Lower parallelism for LLM API calls
)
```

### 4. Combine Multiple Scorers

```go
// Evaluate from multiple perspectives
scorers := []eval.Scorer{
    eval.NewExactMatch("format", "format"),           // Check format
    eval.NewCorrectness(model),                       // Check correctness
    eval.NewRelevance(model),                         // Check relevance
    eval.NewNumericRange("confidence", 0.0, 1.0),    // Check confidence bounds
}
```

### 5. Log and Track Results

```go
// Export to MLflow for historical tracking
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
    eval.WithMLflowExport(mlflowClient, experimentID),
)

// Or save locally
resultsJSON, _ := json.MarshalIndent(results, "", "  ")
os.WriteFile("eval_results.json", resultsJSON, 0644)
```

## Advanced Usage

### Custom Prediction Functions

```go
// Prediction function with agent execution
predictFunc := func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    question := inputs["question"].(string)

    // Run your agent
    result, err := agent.Generate(ctx, fantasy.AgentCall{
        Prompt: question,
    })
    if err != nil {
        return nil, err
    }

    return map[string]any{
        "answer": result.Response.Content.Text(),
        "steps":  result.Steps, // Optional: include execution data
    }, nil
}
```

### Using Traces in Scorers

```go
// Capture trace during prediction
predictFunc := func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    // Your agent with tracing enabled
    result, err := agent.Generate(ctx, fantasy.AgentCall{
        Prompt: inputs["question"].(string),
    })
    if err != nil {
        return nil, err
    }

    // Return outputs with trace
    return map[string]any{
        "answer": result.Response.Content.Text(),
        "trace":  result.Trace, // Include trace for agent-specific scorers
    }, nil
}

// Use agent-specific scorers
scorers := []eval.Scorer{
    eval.NewToolCallTrajectory([]string{"search", "summarize"}),
    eval.NewStepValidation(1, 3),
}
```

## API Reference

For detailed API documentation, see:
- [pkg.go.dev documentation](https://pkg.go.dev/charm.land/fantasy/eval)
- [MLflow export documentation](./mlflow_export.go)
- [Scorer interface](./scorer.go)

## License

Part of the [Fantasy](https://github.com/charmbracelet/fantasy) project.
