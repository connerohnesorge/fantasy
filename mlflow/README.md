# MLflow Go Client

A comprehensive Go client library for the MLflow REST API, providing type-safe access to experiments, runs, traces, and assessments.

## Features

- Full support for MLflow REST API v2.0 and v3.0
- Experiment management (CRUD operations)
- Run tracking with metrics, parameters, and tags
- Distributed tracing with OpenTelemetry integration
- Assessment management for trace evaluation
- Automatic retry with exponential backoff
- Bearer token authentication
- Type-safe protobuf and JSON serialization
- Comprehensive error handling

## Installation

```bash
go get charm.land/fantasy/mlflow
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "charm.land/fantasy/mlflow"
    pb "charm.land/fantasy/proto/gen/mlflow"
)

func main() {
    // Create a client
    client := mlflow.New(
        "http://localhost:5000",
        mlflow.WithToken("your-token-here"),
        mlflow.WithTimeout(30*time.Second),
    )

    ctx := context.Background()

    // Create an experiment
    expID, err := client.CreateExperiment(ctx, "my-experiment")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created experiment: %s\n", expID)

    // Create a run
    run, err := client.CreateRun(ctx, expID,
        mlflow.WithRunName("my-run"),
        mlflow.WithTags(map[string]string{
            "env": "production",
            "model": "gpt-4",
        }),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created run: %s\n", run.Info.RunId)

    // Log metrics
    metrics := []pb.Metric{
        {Key: "accuracy", Value: 0.95, Timestamp: time.Now().UnixMilli()},
        {Key: "loss", Value: 0.05, Timestamp: time.Now().UnixMilli()},
    }
    if err := client.LogBatch(ctx, run.Info.RunId, metrics, nil, nil); err != nil {
        log.Fatal(err)
    }

    // Update run status
    _, err = client.UpdateRun(ctx, run.Info.RunId, "FINISHED", time.Now().UnixMilli())
    if err != nil {
        log.Fatal(err)
    }
}
```

## Client Configuration

### Creating a Client

```go
import (
    "time"
    "net/http"
    "charm.land/fantasy/mlflow"
)

// Basic client
client := mlflow.New("http://localhost:5000")

// With authentication
client := mlflow.New(
    "http://localhost:5000",
    mlflow.WithToken("your-bearer-token"),
)

// With custom timeout (default: 30s)
client := mlflow.New(
    "http://localhost:5000",
    mlflow.WithTimeout(60*time.Second),
)

// With custom retry config (default: 3 retries)
client := mlflow.New(
    "http://localhost:5000",
    mlflow.WithRetries(5), // Set to 0 to disable retries
)

// With custom HTTP client
customClient := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns: 100,
    },
}
client := mlflow.New(
    "http://localhost:5000",
    mlflow.WithHTTPClient(customClient),
)
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithToken(token)` | Bearer token for authentication | None |
| `WithTimeout(duration)` | Request timeout | 30 seconds |
| `WithRetries(n)` | Maximum retry attempts | 3 |
| `WithHTTPClient(client)` | Custom HTTP client | `&http.Client{}` |

## Experiments

### Create an Experiment

```go
expID, err := client.CreateExperiment(ctx, "my-experiment")
if err != nil {
    log.Fatal(err)
}
```

### Get an Experiment

```go
exp, err := client.GetExperiment(ctx, experimentID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Name: %s\n", exp.Name)
fmt.Printf("Lifecycle: %s\n", exp.LifecycleStage)
```

### Search Experiments

```go
results, err := client.SearchExperiments(ctx, mlflow.SearchExperimentsOptions{
    Filter:     "name LIKE 'prod-%'",
    MaxResults: 100,
    OrderBy:    []string{"creation_time DESC"},
    ViewType:   "ACTIVE_ONLY", // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL"
})
if err != nil {
    log.Fatal(err)
}

for _, exp := range results.Experiments {
    fmt.Printf("ID: %s, Name: %s\n", exp.ExperimentId, exp.Name)
}

// Handle pagination
if results.NextPageToken != "" {
    nextPage, err := client.SearchExperiments(ctx, mlflow.SearchExperimentsOptions{
        PageToken: results.NextPageToken,
    })
    // ...
}
```

### Update an Experiment

```go
err := client.UpdateExperiment(ctx, experimentID, "new-experiment-name")
if err != nil {
    log.Fatal(err)
}
```

### Delete an Experiment

```go
err := client.DeleteExperiment(ctx, experimentID)
if err != nil {
    log.Fatal(err)
}
```

## Runs

### Create a Run

```go
run, err := client.CreateRun(ctx, experimentID,
    mlflow.WithRunName("my-run"),
    mlflow.WithStartTime(time.Now().UnixMilli()),
    mlflow.WithTags(map[string]string{
        "env": "staging",
        "version": "1.0.0",
    }),
)
if err != nil {
    log.Fatal(err)
}
```

### Get a Run

```go
run, err := client.GetRun(ctx, runID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Status: %s\n", run.Info.Status)
fmt.Printf("Duration: %d ms\n", run.Info.EndTime - run.Info.StartTime)
```

### Log Metrics, Parameters, and Tags

```go
metrics := []pb.Metric{
    {Key: "accuracy", Value: 0.95, Timestamp: time.Now().UnixMilli()},
    {Key: "precision", Value: 0.92, Timestamp: time.Now().UnixMilli()},
}

params := []pb.Param{
    {Key: "learning_rate", Value: "0.001"},
    {Key: "batch_size", Value: "32"},
}

tags := []pb.RunTag{
    {Key: "model_type", Value: "transformer"},
    {Key: "dataset", Value: "v2"},
}

err := client.LogBatch(ctx, runID, metrics, params, tags)
if err != nil {
    log.Fatal(err)
}
```

### Update Run Status

```go
// Valid statuses: "RUNNING", "SCHEDULED", "FINISHED", "FAILED", "KILLED"
runInfo, err := client.UpdateRun(ctx, runID, "FINISHED", time.Now().UnixMilli())
if err != nil {
    log.Fatal(err)
}
```

### Search Runs

```go
results, err := client.SearchRuns(ctx, mlflow.SearchRunsOptions{
    ExperimentIDs: []string{experimentID},
    Filter:        "metrics.accuracy > 0.9 AND params.model = 'gpt-4'",
    MaxResults:    50,
    OrderBy:       []string{"metrics.accuracy DESC"},
    RunViewType:   "ACTIVE_ONLY", // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL"
})
if err != nil {
    log.Fatal(err)
}

for _, run := range results.Runs {
    fmt.Printf("Run ID: %s, Accuracy: %v\n",
        run.Info.RunId,
        run.Data.Metrics[0].Value,
    )
}
```

### Delete a Run

```go
err := client.DeleteRun(ctx, runID)
if err != nil {
    log.Fatal(err)
}
```

## Traces

Traces provide distributed tracing capabilities with OpenTelemetry integration.

### Start a Trace

```go
import (
    pb "charm.land/fantasy/proto/gen/mlflow"
)

// Create a complete trace (all spans must be populated before upload)
trace := &pb.Trace{
    TraceInfo: &pb.TraceInfoV3{
        TraceId:      proto.String("trace-123"),
        ExperimentId: proto.String(experimentID),
        TimestampMs:  proto.Int64(time.Now().UnixMilli()),
        RequestId:    proto.String("req-456"),
    },
    // Note: Spans are logged separately via LogSpans
}

traceID, err := client.StartTrace(ctx, trace)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Started trace: %s\n", traceID)
```

### Log Spans (OpenTelemetry)

```go
import (
    tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

spans := []*tracev1.Span{
    {
        TraceId: []byte("trace-123"),
        SpanId:  []byte("span-456"),
        Name:    "llm-call",
        StartTimeUnixNano: uint64(time.Now().UnixNano()),
        EndTimeUnixNano:   uint64(time.Now().Add(2*time.Second).UnixNano()),
        // ... other span fields
    },
}

err := client.LogSpans(ctx, experimentID, spans)
if err != nil {
    log.Fatal(err)
}
```

### Get a Trace

```go
// Get complete trace (returns error if IN_PROGRESS)
trace, err := client.GetTrace(ctx, traceID, false)
if err != nil {
    log.Fatal(err)
}

// Get partial trace (allows IN_PROGRESS)
trace, err := client.GetTrace(ctx, traceID, true)
if err != nil {
    log.Fatal(err)
}
```

### Search Traces

```go
results, err := client.SearchTraces(ctx, mlflow.SearchTracesOptions{
    ExperimentIDs: []string{experimentID},
    Filter:        "status = 'OK' AND tags.env = 'prod'",
    MaxResults:    100,
    OrderBy:       []string{"timestamp_ms DESC"},
})
if err != nil {
    log.Fatal(err)
}

for _, traceInfo := range results.Traces {
    fmt.Printf("Trace ID: %s, Status: %s\n",
        *traceInfo.TraceId,
        *traceInfo.Status,
    )
}
```

### Delete Traces

```go
// Delete by count
deletedCount, err := client.DeleteTraces(ctx, experimentID, mlflow.DeleteTracesOptions{
    MaxTraces: 100,
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Deleted %d traces\n", deletedCount)

// Delete by timestamp (delete traces older than timestamp)
deletedCount, err := client.DeleteTraces(ctx, experimentID, mlflow.DeleteTracesOptions{
    MaxTimestampMs: time.Now().Add(-7*24*time.Hour).UnixMilli(), // 7 days ago
})

// Delete by filter
deletedCount, err := client.DeleteTraces(ctx, experimentID, mlflow.DeleteTracesOptions{
    Filter: "status = 'ERROR'",
})
```

## Assessments

Assessments attach feedback or expectations to traces for evaluation.

### Create an Assessment

```go
assessment := &mlflow.Assessment{
    Name: "response-quality",
    Source: mlflow.AssessmentSource{
        SourceType: "LLM_JUDGE", // "CODE", "HUMAN", or "LLM_JUDGE"
        SourceID:   "gpt-4-judge",
    },
    Feedback: &mlflow.FeedbackValue{
        Value: 4.5, // Can be bool, float64, int, string, or structured
    },
    Rationale: "Response was accurate and well-formatted",
    Metadata: map[string]string{
        "model_version": "1.0",
        "judge_temp": "0.7",
    },
}

created, err := client.CreateAssessment(ctx, traceID, assessment)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Assessment ID: %s\n", created.ID)
```

### Create an Expectation Assessment

```go
assessment := &mlflow.Assessment{
    Name: "expected-output",
    Source: mlflow.AssessmentSource{
        SourceType: "HUMAN",
        SourceID:   "annotator@example.com",
    },
    Expectation: &mlflow.ExpectationValue{
        Value: "The capital of France is Paris",
    },
}

created, err := client.CreateAssessment(ctx, traceID, assessment)
```

### Create an Assessment with Error

```go
assessment := &mlflow.Assessment{
    Name: "hallucination-check",
    Source: mlflow.AssessmentSource{
        SourceType: "CODE",
        SourceID:   "hallucination-detector-v1",
    },
    Feedback: &mlflow.FeedbackValue{
        Value: nil,
        Error: &mlflow.AssessmentError{
            ErrorMessage: "API rate limit exceeded",
            ErrorCode:    "RATE_LIMIT",
            StackTrace:   "...",
        },
    },
}

created, err := client.CreateAssessment(ctx, traceID, assessment)
```

### Update an Assessment

```go
updatedAssessment := &mlflow.Assessment{
    Feedback: &mlflow.FeedbackValue{
        Value: 5.0,
    },
    Rationale: "Updated rationale after review",
}

updated, err := client.UpdateAssessment(
    ctx,
    traceID,
    assessmentID,
    updatedAssessment,
    []string{"feedback.value", "rationale"}, // Update mask
)
if err != nil {
    log.Fatal(err)
}
```

### Delete an Assessment

```go
err := client.DeleteAssessment(ctx, traceID, assessmentID)
if err != nil {
    log.Fatal(err)
}
```

## Error Handling

The client provides structured error types for different failure scenarios.

### Error Types

```go
import "errors"

result, err := client.GetExperiment(ctx, experimentID)
if err != nil {
    // Check for specific error types
    var apiErr *mlflow.APIError
    var validationErr *mlflow.ValidationError
    var timeoutErr *mlflow.TimeoutError
    var connErr *mlflow.ConnectionError

    switch {
    case errors.As(err, &apiErr):
        // Handle API errors
        if apiErr.IsNotFound() {
            fmt.Println("Experiment not found")
        } else if apiErr.IsUnauthorized() {
            fmt.Println("Invalid or missing authentication token")
        } else if apiErr.IsRateLimited() {
            fmt.Println("Rate limit exceeded, will retry automatically")
        }

    case errors.As(err, &validationErr):
        // Handle validation errors (client-side)
        fmt.Printf("Invalid input: %s (field: %s)\n",
            validationErr.Message,
            validationErr.Field,
        )

    case errors.As(err, &timeoutErr):
        // Handle timeout errors
        fmt.Printf("Request timed out after %dms: %s\n",
            timeoutErr.Timeout,
            timeoutErr.Operation,
        )

    case errors.As(err, &connErr):
        // Handle connection errors
        fmt.Printf("Connection failed to %s: %s\n",
            connErr.URL,
            connErr.Message,
        )
    }
}
```

### APIError Methods

```go
type APIError struct {
    StatusCode int
    Message    string
    ErrorCode  string
    Cause      error
}

// Helper methods
err.IsNotFound()      // 404
err.IsUnauthorized()  // 401
err.IsForbidden()     // 403
err.IsConflict()      // 409
err.IsRateLimited()   // 429
err.IsServerError()   // 5xx
err.IsRetryable()     // Rate limited or server error
```

### Automatic Retry Logic

The client automatically retries failed requests using exponential backoff:

- **Default**: 3 retry attempts
- **Retryable errors**: Rate limits (429), server errors (5xx), network errors
- **Backoff**: Exponential with jitter (1s → 2s → 4s, max 10s)
- **Non-retryable**: Validation errors, 4xx client errors (except 429)

```go
// Disable retries
client := mlflow.New(baseURL, mlflow.WithRetries(0))

// Increase retry attempts
client := mlflow.New(baseURL, mlflow.WithRetries(5))
```

## API Reference Summary

### Client Methods

#### Experiments
- `CreateExperiment(ctx, name) (experimentID, error)`
- `GetExperiment(ctx, experimentID) (*Experiment, error)`
- `SearchExperiments(ctx, opts) (*SearchExperimentsResult, error)`
- `UpdateExperiment(ctx, experimentID, newName) error`
- `DeleteExperiment(ctx, experimentID) error`

#### Runs
- `CreateRun(ctx, experimentID, ...opts) (*Run, error)`
- `GetRun(ctx, runID) (*Run, error)`
- `UpdateRun(ctx, runID, status, endTime) (*RunInfo, error)`
- `DeleteRun(ctx, runID) error`
- `LogBatch(ctx, runID, metrics, params, tags) error`
- `SearchRuns(ctx, opts) (*SearchRunsResult, error)`

#### Traces
- `StartTrace(ctx, trace) (traceID, error)`
- `LogSpans(ctx, experimentID, spans) error`
- `GetTrace(ctx, traceID, allowPartial) (*Trace, error)`
- `SearchTraces(ctx, opts) (*SearchTracesResult, error)`
- `DeleteTraces(ctx, experimentID, opts) (deletedCount, error)`

#### Assessments
- `CreateAssessment(ctx, traceID, assessment) (*Assessment, error)`
- `UpdateAssessment(ctx, traceID, assessmentID, assessment, updateMask) (*Assessment, error)`
- `DeleteAssessment(ctx, traceID, assessmentID) error`

### Protobuf Types

The client uses protobuf types from `charm.land/fantasy/proto/gen/mlflow`:

- `pb.Experiment` - Experiment metadata
- `pb.Run` - Run with info and data
- `pb.RunInfo` - Run metadata
- `pb.Metric` - Metric with key, value, timestamp
- `pb.Param` - Parameter with key, value
- `pb.RunTag` - Tag with key, value
- `pb.Trace` - Trace with info and spans
- `pb.TraceInfoV3` - Trace metadata
- `tracev1.Span` - OpenTelemetry span (from `go.opentelemetry.io/proto/otlp/trace/v1`)

## Best Practices

### 1. Use Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

result, err := client.GetExperiment(ctx, experimentID)
```

### 2. Handle Errors Properly

```go
if err != nil {
    var apiErr *mlflow.APIError
    if errors.As(err, &apiErr) {
        if apiErr.IsNotFound() {
            // Create the resource if it doesn't exist
        } else if apiErr.IsRetryable() {
            // The client already retried, log and alert
        }
    }
    return err
}
```

### 3. Use Batch Operations

```go
// Good: Single batch call
metrics := []pb.Metric{
    {Key: "accuracy", Value: 0.95, Timestamp: ts},
    {Key: "loss", Value: 0.05, Timestamp: ts},
}
client.LogBatch(ctx, runID, metrics, nil, nil)

// Avoid: Multiple individual calls
// client.LogMetric(ctx, runID, "accuracy", 0.95)
// client.LogMetric(ctx, runID, "loss", 0.05)
```

### 4. Reuse Client Instances

```go
// Good: Single client for multiple operations
client := mlflow.New(baseURL, mlflow.WithToken(token))
// Use client for multiple requests

// Avoid: Creating new client for each request
// client1 := mlflow.New(baseURL)
// client2 := mlflow.New(baseURL)
```

### 5. Close Run When Done

```go
run, err := client.CreateRun(ctx, experimentID)
if err != nil {
    return err
}

defer func() {
    // Always mark run as finished/failed
    status := "FINISHED"
    if err != nil {
        status = "FAILED"
    }
    _, _ = client.UpdateRun(ctx, run.Info.RunId, status, time.Now().UnixMilli())
}()

// ... perform operations ...
```

## License

See the project root for license information.
