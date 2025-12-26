# MLflow Integration Testing Guide

This document provides steps for manually testing the Fantasy MLflow integration against a local MLflow server.

## Prerequisites

- Docker (for running MLflow server)
- Go 1.25+
- Buf CLI (for proto generation)

## Setup Local MLflow Server

### Option 1: Docker Compose (Recommended)

Create a `docker-compose.yml` file:

```yaml
version: '3'
services:
  mlflow:
    image: ghcr.io/mlflow/mlflow:v3.8.0
    ports:
      - "5000:5000"
    command: mlflow server --host 0.0.0.0 --port 5000
```

Start the server:

```bash
docker-compose up -d
```

### Option 2: Local Installation

```bash
pip install mlflow==3.8.0
mlflow server --host 127.0.0.1 --port 5000
```

## Verify MLflow Server

Check that the server is running:

```bash
curl http://localhost:5000/health
```

Expected response: `OK`

## Manual Test Checklist

### 1. Experiment Management

```go
package main

import (
    "context"
    "fmt"
    "log"

    "charm.land/fantasy/mlflowclient"
)

func main() {
    ctx := context.Background()
    client := mlflowclient.New("http://localhost:5000")

    // Create experiment
    expID, err := client.CreateExperiment(ctx, "test-experiment", map[string]string{
        "env": "testing",
    })
    if err != nil {
        log.Fatalf("Failed to create experiment: %v", err)
    }
    fmt.Printf("✓ Created experiment: %s\n", expID)

    // Get experiment
    exp, err := client.GetExperiment(ctx, expID)
    if err != nil {
        log.Fatalf("Failed to get experiment: %v", err)
    }
    fmt.Printf("✓ Retrieved experiment: %s\n", exp.GetName())

    // Search experiments
    experiments, err := client.SearchExperiments(ctx, "", 10)
    if err != nil {
        log.Fatalf("Failed to search experiments: %v", err)
    }
    fmt.Printf("✓ Found %d experiments\n", len(experiments))
}
```

**Expected Results:**
- Experiment created successfully
- Experiment retrieved with correct name
- Search returns at least 1 experiment

### 2. Run Management

```go
// Create a run
run, err := client.CreateRun(ctx, expID,
    mlflowclient.WithRunName("test-run"),
    mlflowclient.WithRunTags(map[string]string{"model": "test"}),
)
if err != nil {
    log.Fatalf("Failed to create run: %v", err)
}
fmt.Printf("✓ Created run: %s\n", run.Info.GetRunId())

// Log metrics and parameters
err = client.LogBatch(ctx, run.Info.GetRunId(),
    []*pb.Metric{{Key: stringPtr("accuracy"), Value: float64Ptr(0.95), Timestamp: int64Ptr(time.Now().UnixMilli())}},
    []*pb.Param{{Key: stringPtr("lr"), Value: stringPtr("0.001")}},
    nil,
)
if err != nil {
    log.Fatalf("Failed to log batch: %v", err)
}
fmt.Printf("✓ Logged metrics and params\n")

// Get run
runInfo, err := client.GetRun(ctx, run.Info.GetRunId())
if err != nil {
    log.Fatalf("Failed to get run: %v", err)
}
fmt.Printf("✓ Retrieved run with %d metrics\n", len(runInfo.Data.Metrics))
```

**Expected Results:**
- Run created with tags
- Metrics and parameters logged successfully
- Run retrieved with correct data

### 3. Tracing API (V3)

```go
import pb "charm.land/fantasy/proto/gen/mlflow"

// Start a trace
trace := &mlflowclient.Trace{
    TraceInfoV3: &pb.TraceInfoV3{
        TraceId: stringPtr("test-trace-" + uuid.New().String()),
        State:   pb.TraceInfoV3_IN_PROGRESS.Enum(),
    },
}

traceInfo, err := client.StartTrace(ctx, trace)
if err != nil {
    log.Fatalf("Failed to start trace: %v", err)
}
fmt.Printf("✓ Started trace: %s\n", traceInfo.GetTraceId())

// Get trace
retrievedTrace, err := client.GetTrace(ctx, traceInfo.GetTraceId(), false)
if err != nil {
    log.Fatalf("Failed to get trace: %v", err)
}
fmt.Printf("✓ Retrieved trace with state: %v\n", retrievedTrace.TraceInfo.GetState())

// Search traces
traces, err := client.SearchTraces(ctx, "", 10, "", []string{})
if err != nil {
    log.Fatalf("Failed to search traces: %v", err)
}
fmt.Printf("✓ Found %d traces\n", len(traces))
```

**Expected Results:**
- Trace created with IN_PROGRESS state
- Trace retrieved successfully
- Search returns at least 1 trace

### 4. Assessment API

```go
// Create assessment
assessment := &pb.Assessment{
    AssessmentName: stringPtr("correctness"),
    Value:          structpb.NewNumberValue(1.0),
    Rationale:      stringPtr("Correct response"),
}

createdAssessment, err := client.CreateAssessment(ctx, traceInfo.GetTraceId(), assessment)
if err != nil {
    log.Fatalf("Failed to create assessment: %v", err)
}
fmt.Printf("✓ Created assessment: %s\n", createdAssessment.GetAssessmentId())

// Get assessment
retrievedAssessment, err := client.GetAssessment(ctx, traceInfo.GetTraceId(), createdAssessment.GetAssessmentId())
if err != nil {
    log.Fatalf("Failed to get assessment: %v", err)
}
fmt.Printf("✓ Retrieved assessment: %s\n", retrievedAssessment.GetAssessmentName())
```

**Expected Results:**
- Assessment created and linked to trace
- Assessment retrieved with correct data

### 5. Full Integration Test

Run the complete integration test from the examples:

```bash
cd examples/mlflow
go run main.go
```

**Expected Results:**
- Agent executes successfully
- Trace logged to MLflow
- Assessments created from evaluation scores
- All data visible in MLflow UI at http://localhost:5000

## Verification in MLflow UI

1. Open http://localhost:5000 in your browser
2. Navigate to the test experiment
3. Verify:
   - Experiment exists with tags
   - Runs are listed with metrics/params
   - Traces are visible in the Traces tab
   - Assessments are attached to traces

## Cleanup

Stop the MLflow server:

```bash
docker-compose down
```

Or if using local installation:

```bash
# Stop the mlflow server process
```

## Troubleshooting

### Connection Refused
- Ensure MLflow server is running: `curl http://localhost:5000/health`
- Check Docker logs: `docker-compose logs mlflow`

### API Version Errors
- Verify MLflow version is 3.8.0+: `curl http://localhost:5000/version`
- Check that trace/assessment endpoints exist

### Timeout Errors
- Increase client timeout: `mlflowclient.New("http://localhost:5000", mlflowclient.WithTimeout(60*time.Second))`

## Notes

- The MLflow server data is ephemeral in Docker unless you mount a volume
- For persistent testing, add a volume mount to docker-compose.yml:
  ```yaml
  volumes:
    - ./mlflow-data:/mlflow
  ```
- The client uses protojson for marshaling, which should be compatible with MLflow's REST API
