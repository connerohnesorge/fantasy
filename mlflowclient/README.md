# MLflow Client

Go client for the MLflow REST API (v2 and v3).

## Overview

This package provides a type-safe Go client for interacting with MLflow's REST API:

- **Experiments**: Create, search, update, delete experiments
- **Runs**: Create runs, log metrics/params/tags, search runs
- **Traces**: Upload traces, search traces, manage trace tags
- **Assessments**: Create and manage assessments on traces
- **Scorers**: Register and manage scorers

## Installation

```go
import "charm.land/fantasy/mlflowclient"
```

## Quick Start

```go
// Create client
client, err := mlflowclient.New("http://localhost:5000")
if err != nil {
    log.Fatal(err)
}

// Create experiment
exp, err := client.CreateExperiment(ctx, "my-experiment", nil)

// Create run
run, err := client.CreateRun(ctx, exp.ExperimentID,
    mlflowclient.WithRunName("training-run"),
)

// Log metrics
err = client.LogMetric(ctx, run.Info.RunID, "accuracy", 0.95, time.Now().UnixMilli(), 1)

// Search runs
runs, _, err := client.SearchRuns(ctx, mlflowclient.SearchRunsOptions{
    ExperimentIDs: []string{exp.ExperimentID},
    Filter:        "metrics.accuracy > 0.9",
})
```

## Client Configuration

```go
client, err := mlflowclient.New("http://localhost:5000",
    // Custom HTTP client
    mlflowclient.WithHTTPClient(customClient),

    // Bearer token authentication
    mlflowclient.WithToken("your-token"),

    // Request timeout
    mlflowclient.WithTimeout(30 * time.Second),

    // Retry configuration
    mlflowclient.WithRetries(5),
    mlflowclient.WithRetryConfig(mlflowclient.RetryConfig{
        MaxRetries:     5,
        InitialDelay:   1 * time.Second,
        MaxDelay:       30 * time.Second,
        BackoffFactor:  2.0,
        JitterFraction: 0.1,
    }),
)
```

## Experiments

```go
// Create
exp, err := client.CreateExperiment(ctx, "experiment-name", nil)

// Get by ID
exp, err := client.GetExperiment(ctx, experimentID)

// Get by name
exp, err := client.GetExperimentByName(ctx, "experiment-name")

// Search
exps, nextToken, err := client.SearchExperiments(ctx, mlflowclient.SearchExperimentsOptions{
    Filter:     "name LIKE 'train-%'",
    MaxResults: 100,
})

// Update
err := client.UpdateExperiment(ctx, experimentID, "new-name")

// Delete
err := client.DeleteExperiment(ctx, experimentID)
```

## Runs

```go
// Create with options
run, err := client.CreateRun(ctx, experimentID,
    mlflowclient.WithRunName("my-run"),
    mlflowclient.WithStartTime(time.Now().UnixMilli()),
    mlflowclient.WithTags(map[string]string{
        "model": "gpt-4",
        "version": "1.0",
    }),
)

// Log batch
err := client.LogBatch(ctx, runID,
    []*mlflowclient.Metric{
        {Key: "loss", Value: 0.5, Timestamp: time.Now().UnixMilli()},
    },
    []*mlflowclient.Param{
        {Key: "learning_rate", Value: "0.001"},
    },
    []*mlflowclient.RunTag{
        {Key: "framework", Value: "pytorch"},
    },
)

// Update status
info, err := client.UpdateRun(ctx, runID, mlflowclient.RunStatusFinished, time.Now().UnixMilli())
```

## Traces (API v3)

```go
// Upload trace
trace := &mlflowclient.Trace{
    ExperimentID:    experimentID,
    RequestTimeMs:   time.Now().UnixMilli(),
    ExecutionTimeMs: 1500,
    State:           mlflowclient.TraceStateOK,
    RequestPreview:  "What is 2+2?",
    ResponsePreview: "4",
    Spans: []*mlflowclient.Span{
        {
            SpanID:      "span-1",
            Name:        "agent",
            SpanType:    mlflowclient.SpanTypeAgent,
            StartTimeNs: time.Now().UnixNano(),
            EndTimeNs:   time.Now().Add(time.Second).UnixNano(),
            Status:      &mlflowclient.SpanStatus{Code: mlflowclient.SpanStatusOK},
        },
    },
}
traceID, err := client.StartTrace(ctx, trace)

// Get trace
trace, err := client.GetTrace(ctx, traceID, false)

// Search traces
traces, nextToken, err := client.SearchTraces(ctx, mlflowclient.SearchTracesOptions{
    ExperimentIDs: []string{experimentID},
    Filter:        "state = 'OK'",
    MaxResults:    100,
})

// Manage tags
err := client.SetTraceTag(ctx, traceID, "env", "production")
err := client.DeleteTraceTag(ctx, traceID, "env")
```

## Assessments (API v3)

```go
// Create assessment
assessment, err := client.CreateAssessment(ctx, traceID, &mlflowclient.Assessment{
    Name:      "correctness",
    Rationale: "Response matches expected answer",
    Source: &mlflowclient.AssessmentSource{
        SourceType: mlflowclient.SourceTypeCode,
        SourceID:   "eval/correctness",
    },
    Feedback: &mlflowclient.FeedbackValue{
        Value: true,
    },
})

// Update assessment
assessment, err := client.UpdateAssessment(ctx, traceID, assessmentID, &mlflowclient.Assessment{
    Rationale: "Updated rationale",
}, []string{"rationale"})

// Delete assessment
err := client.DeleteAssessment(ctx, traceID, assessmentID)
```

## Error Handling

```go
result, err := client.GetExperiment(ctx, experimentID)
if err != nil {
    var apiErr *mlflowclient.APIError
    if errors.As(err, &apiErr) {
        if apiErr.IsNotFound() {
            // Handle 404
        } else if apiErr.IsRateLimited() {
            // Handle rate limiting
        } else if apiErr.IsRetryable() {
            // Server error, can retry
        }
    }
}
```

## Error Types

- `APIError`: HTTP API errors with status code and message
- `ValidationError`: Client-side validation errors
- `TimeoutError`: Request timeout
- `ConnectionError`: Network connectivity issues
