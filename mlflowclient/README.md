# MLflow Client

A thin REST client for the MLflow API, using protojson marshaling with generated proto types. Supports both legacy v2 endpoints (experiments, runs, metrics) and modern v3 endpoints (traces, assessments, scorers).

## Installation

```bash
go get charm.land/fantasy/mlflowclient
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "charm.land/fantasy/mlflowclient"
)

func main() {
    // Create a client
    client := mlflowclient.New("http://localhost:5000",
        mlflowclient.WithToken("your-mlflow-token"),
        mlflowclient.WithTimeout(60*time.Second),
    )

    ctx := context.Background()

    // Create an experiment
    expID, err := client.CreateExperiment(ctx, "my-experiment", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created experiment: %s\n", expID)

    // Create a run
    run, err := client.CreateRun(ctx, expID, mlflowclient.WithRunName("my-run"))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created run: %s\n", run.GetRunId())
}
```

## API Versioning

The client uses a dual API version strategy:

- `/api/2.0/mlflow/` - Legacy endpoints for experiments, runs, and metrics (stable)
- `/api/3.0/mlflow/` - Modern endpoints for traces, assessments, and scorers (v3.8+)

## Configuration

### Client Options

```go
client := mlflowclient.New("http://localhost:5000",
    // Set authentication token
    mlflowclient.WithToken("your-token"),

    // Set request timeout (default: 30s)
    mlflowclient.WithTimeout(60*time.Second),

    // Configure retry behavior
    mlflowclient.WithMaxRetries(5),
)
```

### Retry Configuration

The client automatically retries failed requests with exponential backoff:

- **Maximum retries**: 3 (configurable)
- **Initial delay**: 1 second
- **Backoff multiplier**: 2x (exponential)
- **Maximum delay**: 10 seconds
- **Jitter**: ±10% randomization

Retryable errors include:
- 5xx server errors
- 429 rate limit errors
- Connection errors

## Usage Examples

### Experiment Management

```go
// Create an experiment
tags := map[string]string{
    "team": "ml-research",
    "project": "agent-evaluation",
}
expID, err := client.CreateExperiment(ctx, "my-experiment", tags)
if err != nil {
    log.Fatal(err)
}

// Get experiment by ID
exp, err := client.GetExperiment(ctx, expID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Experiment: %s\n", exp.GetName())

// Search experiments
experiments, err := client.SearchExperiments(ctx, "name LIKE 'my-%'", 10)
if err != nil {
    log.Fatal(err)
}
for _, exp := range experiments {
    fmt.Printf("Found: %s (%s)\n", exp.GetName(), exp.GetExperimentId())
}
```

### Run Management

```go
// Create a run with tags
run, err := client.CreateRun(ctx, expID,
    mlflowclient.WithRunName("baseline-v1"),
    mlflowclient.WithRunTags(map[string]string{
        "model": "gpt-4",
        "version": "1.0",
    }),
)
if err != nil {
    log.Fatal(err)
}

// Log metrics in batch
err = client.LogBatch(ctx, run.GetRunId(),
    mlflowclient.WithMetrics([]mlflowclient.Metric{
        {Key: "accuracy", Value: 0.95, Timestamp: time.Now().UnixMilli()},
        {Key: "latency", Value: 1.2, Timestamp: time.Now().UnixMilli()},
    }),
    mlflowclient.WithParams([]mlflowclient.Param{
        {Key: "temperature", Value: "0.7"},
        {Key: "max_tokens", Value: "1000"},
    }),
)
if err != nil {
    log.Fatal(err)
}

// Update run status
err = client.UpdateRun(ctx, run.GetRunId(), "FINISHED", nil)
if err != nil {
    log.Fatal(err)
}

// Search runs
runs, err := client.SearchRuns(ctx, []string{expID},
    "metrics.accuracy > 0.9",
    []string{"metrics.accuracy DESC"},
    10)
if err != nil {
    log.Fatal(err)
}
```

### Trace Management

```go
import (
    mlflow "charm.land/fantasy/proto/gen/mlflow"
)

// Start a trace
trace := &mlflowclient.Trace{
    TraceInfoV3: &mlflow.TraceInfoV3{
        TraceId:      stringPtr("tr-" + generateRandomID()),
        ExperimentId: stringPtr(expID),
        State:        mlflow.TraceInfoV3_IN_PROGRESS.Enum(),
        ExecutionTimeMs: int64Ptr(0),
        Tags: []*mlflow.TraceTag{
            {Key: stringPtr("agent"), Value: stringPtr("my-agent")},
        },
    },
}
result, err := client.StartTrace(ctx, trace)
if err != nil {
    log.Fatal(err)
}

// Get trace
trace, err = client.GetTrace(ctx, result.GetTraceId())
if err != nil {
    log.Fatal(err)
}

// Set trace tag
err = client.SetTraceTag(ctx, result.GetTraceId(), "status", "completed")
if err != nil {
    log.Fatal(err)
}

// Search traces
traces, err := client.SearchTraces(ctx, expID, "tags.agent = 'my-agent'", 10)
if err != nil {
    log.Fatal(err)
}
```

### Assessment Management

Assessments are MLflow's way of attaching evaluation feedback (scores, rationales) to traces.

```go
import (
    mlflow "charm.land/fantasy/proto/gen/mlflow"
)

// Create an assessment
assessment := &mlflow.CreateAssessmentRequest{
    TraceId: stringPtr("tr-123"),
    Name:    stringPtr("correctness"),
    Source: &mlflow.AssessmentSource{
        SourceType: mlflow.AssessmentSource_CODE.Enum(),
        SourceId:   stringPtr("eval-v1"),
    },
    BooleanValue: boolPtr(true),
    Rationale:    stringPtr("Output matches expected answer exactly"),
    Metadata: map[string]string{
        "scorer": "exact_match",
        "version": "1.0",
    },
}
assessmentID, err := client.CreateAssessment(ctx, assessment)
if err != nil {
    log.Fatal(err)
}

// Update assessment
err = client.UpdateAssessment(ctx, assessmentID,
    mlflowclient.WithNumericValue(0.95),
    mlflowclient.WithRationale("Revised score based on fuzzy matching"),
)
if err != nil {
    log.Fatal(err)
}
```

### Scorer Registration

```go
import (
    mlflow "charm.land/fantasy/proto/gen/mlflow"
)

// Register a custom scorer
scorer := &mlflow.RegisterScorerRequest{
    Name: stringPtr("custom_relevance"),
    ScorerType: mlflow.ScorerType_LLM_AS_JUDGE.Enum(),
    Definition: &mlflow.ScorerDefinition{
        ModelName: stringPtr("gpt-4"),
        Prompt:    stringPtr("Rate the relevance of the answer to the question..."),
    },
}
scorerID, err := client.RegisterScorer(ctx, scorer)
if err != nil {
    log.Fatal(err)
}

// List scorers
scorers, err := client.ListScorers(ctx, 100)
if err != nil {
    log.Fatal(err)
}
for _, s := range scorers {
    fmt.Printf("Scorer: %s (type: %s)\n", s.GetName(), s.GetScorerType())
}
```

## Error Handling

The client provides structured error types:

```go
_, err := client.GetExperiment(ctx, "non-existent")
if apiErr, ok := err.(*mlflowclient.APIError); ok {
    // Check error type
    if apiErr.IsNotFound() {
        fmt.Println("Experiment not found")
    } else if apiErr.IsRetryable() {
        fmt.Println("Temporary server error, retry later")
    }

    // Access error details
    fmt.Printf("Status: %d\n", apiErr.StatusCode)
    fmt.Printf("Code: %s\n", apiErr.ErrorCode)
    fmt.Printf("Message: %s\n", apiErr.Message)
}

// Handle timeout errors
if timeoutErr, ok := err.(*mlflowclient.TimeoutError); ok {
    fmt.Printf("Request timed out after %v\n", timeoutErr.Timeout)
}

// Handle connection errors
if connErr, ok := err.(*mlflowclient.ConnectionError); ok {
    fmt.Printf("Failed to connect to %s: %s\n", connErr.URL, connErr.Message)
}
```

## Best Practices

### 1. Use Context for Cancellation

Always pass a context with timeout or cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

exp, err := client.GetExperiment(ctx, expID)
```

### 2. Handle Errors Gracefully

Check for specific error types and handle them appropriately:

```go
exp, err := client.GetExperiment(ctx, expID)
if err != nil {
    if apiErr, ok := err.(*mlflowclient.APIError); ok && apiErr.IsNotFound() {
        // Create the experiment if it doesn't exist
        expID, err = client.CreateExperiment(ctx, "my-experiment", nil)
    } else {
        return err
    }
}
```

### 3. Batch Operations

Use `LogBatch` for efficient metric/parameter logging:

```go
// Better: batch multiple metrics
err := client.LogBatch(ctx, runID,
    mlflowclient.WithMetrics([]mlflowclient.Metric{
        {Key: "train_loss", Value: 0.1},
        {Key: "val_loss", Value: 0.15},
        {Key: "accuracy", Value: 0.95},
    }),
)

// Avoid: multiple individual calls
// client.LogMetric(ctx, runID, "train_loss", 0.1)
// client.LogMetric(ctx, runID, "val_loss", 0.15)
// client.LogMetric(ctx, runID, "accuracy", 0.95)
```

### 4. Clean Up Resources

Always close or finalize runs when done:

```go
defer func() {
    if err := client.UpdateRun(ctx, runID, "FINISHED", nil); err != nil {
        log.Printf("Failed to finalize run: %v", err)
    }
}()
```

## Integration with Fantasy Agents

The mlflowclient is designed to integrate seamlessly with Fantasy's tracing and evaluation frameworks:

```go
import (
    "charm.land/fantasy"
    "charm.land/fantasy/mlflowclient"
    "charm.land/fantasy/tracing"
)

// Create MLflow client
mlflowClient := mlflowclient.New("http://localhost:5000")

// Enable tracing on your agent
agent := fantasy.NewAgent(model,
    fantasy.WithTracing(tracing.TracingConfig{
        Client:       mlflowClient,
        ExperimentID: "my-experiment",
        AgentName:    "my-agent",
        ModelName:    "gpt-4",
    }),
)
```

See the [tracing package documentation](../tracing/README.md) for more details on agent tracing.

## Helper Functions

```go
// Pointer helpers for proto fields
func stringPtr(s string) *string {
    return &s
}

func int64Ptr(i int64) *int64 {
    return &i
}

func boolPtr(b bool) *bool {
    return &b
}

// Generate random trace ID
func generateRandomID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

## API Reference

For detailed API documentation, see:
- [pkg.go.dev documentation](https://pkg.go.dev/charm.land/fantasy/mlflowclient)
- [MLflow REST API docs](https://mlflow.org/docs/latest/rest-api.html)
- [Proto definitions](../proto/mlflow/)

## License

Part of the [Fantasy](https://github.com/charmbracelet/fantasy) project.
