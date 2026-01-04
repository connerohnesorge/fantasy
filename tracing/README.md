# Tracing Package

MLflow tracing integration for the Fantasy agent framework. This package provides automatic trace collection and instrumentation for agent executions, LLM calls, tool invocations, and custom operations.

## Overview

The tracing package captures hierarchical execution traces compatible with MLflow's tracing API. It supports:

- Automatic instrumentation via callbacks
- Manual span creation for custom operations
- Parent-child span relationships
- Rich metadata and attributes
- OpenTelemetry-compatible format
- Async flush to MLflow

## Key Types

### Trace

Represents a complete execution trace with metadata and spans.

```go
type Trace struct {
    TraceID           string            // MLflow trace ID (format: "tr-<hex>")
    OtelTraceID       string            // OpenTelemetry trace ID
    ExperimentID      string            // MLflow experiment ID
    RequestTime       int64             // Milliseconds since epoch
    ExecutionDuration int64             // Milliseconds
    State             TraceState        // IN_PROGRESS, OK, ERROR
    RequestPreview    string            // Truncated request
    ResponsePreview   string            // Truncated response
    TraceMetadata     map[string]string // Custom metadata
    Tags              map[string]string // Custom tags
    Spans             []*Span           // Child spans
}
```

### Span

Represents a unit of work within a trace.

```go
type Span struct {
    TraceID     string                // Parent trace ID
    SpanID      string                // Unique span ID
    ParentID    string                // Parent span ID
    Name        string                // Span name
    SpanType    SpanType              // AGENT, LLM, TOOL, CHAIN, etc.
    StartTimeNs int64                 // Start time in nanoseconds
    EndTimeNs   int64                 // End time in nanoseconds
    Status      SpanStatus            // OK, ERROR, UNSET
    Attributes  map[string]any        // Custom attributes
    Events      []SpanEvent           // Timeline events
}
```

### Span Types

```go
const (
    SpanTypeAgent     SpanType = "AGENT"      // Top-level agent execution
    SpanTypeLLM       SpanType = "LLM"        // LLM API call
    SpanTypeTool      SpanType = "TOOL"       // Tool invocation
    SpanTypeChain     SpanType = "CHAIN"      // Sequential operations
    SpanTypeRetriever SpanType = "RETRIEVER"  // Document retrieval
    SpanTypeEmbedding SpanType = "EMBEDDING"  // Embedding generation
    SpanTypeUnknown   SpanType = "UNKNOWN"    // Unknown/custom
)
```

### TracingConfig

Configuration for tracing setup.

```go
type TracingConfig struct {
    Client       any               // *mlflow.Client instance
    ExperimentID string            // Required: MLflow experiment ID
    AgentName    string            // Optional: agent identifier
    ModelName    string            // Optional: model identifier
    SessionID    string            // Optional: session identifier
    Tags         map[string]string // Optional: custom tags
    FlushTimeout time.Duration     // Optional: default 10s
}
```

## Usage Examples

### 1. Integrating with Fantasy Agents

Use `TracingCallbacks` to automatically instrument agent executions:

```go
package main

import (
    "context"
    "time"

    "charm.land/fantasy/mlflow"
    "charm.land/fantasy/tracing"
)

func main() {
    // Initialize MLflow client
    client := mlflow.NewClient("http://localhost:5000")

    // Create tracing configuration
    config := tracing.TracingConfig{
        Client:       client,
        ExperimentID: "my-experiment-id",
        AgentName:    "customer-support-agent",
        ModelName:    "claude-opus-4-5",
        SessionID:    "session-123",
        Tags: map[string]string{
            "environment": "production",
            "version":     "v1.0",
        },
        FlushTimeout: 10 * time.Second,
    }

    // Create callbacks
    callbacks := tracing.NewTracingCallbacks(config)

    // Use with agent (pseudo-code - depends on agent implementation)
    agent := NewAgent(WithCallbacks(callbacks))
    response, err := agent.Run(context.Background(), "Hello!")

    // Check for flush errors
    result := callbacks.GetResult()
    if result.FlushError != nil {
        log.Printf("Failed to flush trace: %v", result.FlushError)
    }
}
```

### 2. Manual Span Creation

For custom instrumentation:

```go
package main

import (
    "context"
    "time"

    "charm.land/fantasy/mlflow"
    "charm.land/fantasy/tracing"
)

func processData(ctx context.Context, data string) error {
    // Create tracer
    client := mlflow.NewClient("http://localhost:5000")
    config := tracing.TracingConfig{
        Client:       client,
        ExperimentID: "exp-123",
        AgentName:    "data-processor",
    }
    tracer := tracing.NewTracer(config)

    // Start trace
    trace := tracer.StartTrace("Processing data")
    defer func() {
        // End any orphan spans
        tracer.EndOrphanSpans()

        // Set trace state
        trace.SetState(tracing.TraceStateOK)

        // Flush to MLflow
        flushTrace(ctx, tracer, client)
    }()

    // Create parent span
    parentSpan := tracer.StartSpan("validate", tracing.SpanTypeChain)
    parentSpan.SetAttribute("data_size", len(data))

    // Nested child span
    childSpan := tracer.StartSpan("parse", tracing.SpanTypeTool)
    childSpan.SetAttribute(tracing.AttrSpanInputs, data)

    // Do work...
    time.Sleep(100 * time.Millisecond)

    childSpan.SetAttribute(tracing.AttrSpanOutputs, "parsed result")
    childSpan.SetStatus(tracing.SpanStatusOK, "")
    tracer.EndSpan(childSpan)

    tracer.EndSpan(parentSpan)

    return nil
}

func flushTrace(ctx context.Context, tracer *tracing.Tracer, client *mlflow.Client) error {
    trace := tracer.GetTrace()
    if trace == nil {
        return nil
    }

    // Set execution duration
    trace.mu.Lock()
    trace.ExecutionDuration = (tracing.nowNanos() - (trace.RequestTime * 1_000_000)) / 1_000_000
    trace.mu.Unlock()

    // Convert to protobuf
    pbTrace, err := tracing.TraceToPb(trace)
    if err != nil {
        return err
    }

    // Upload trace info
    _, err = client.StartTrace(ctx, pbTrace)
    if err != nil {
        return err
    }

    // Upload spans via OTLP
    if len(pbTrace.Spans) > 0 {
        return client.LogSpans(ctx, trace.ExperimentID, pbTrace.Spans)
    }

    return nil
}
```

### 3. Adding Span Events

Track events during span execution:

```go
span := tracer.StartSpan("llm-call", tracing.SpanTypeLLM)
defer tracer.EndSpan(span)

// Add events
span.AddEvent("request_sent", map[string]any{
    "timestamp": time.Now().Unix(),
    "endpoint":  "/v1/chat/completions",
})

// Do work...

span.AddEvent("response_received", map[string]any{
    "status_code": 200,
    "latency_ms":  123,
})
```

### 4. Error Handling

Properly handle errors in spans:

```go
span := tracer.StartSpan("risky-operation", tracing.SpanTypeTool)
defer tracer.EndSpan(span)

result, err := doRiskyOperation()
if err != nil {
    span.SetStatus(tracing.SpanStatusError, err.Error())
    span.SetAttribute("error.type", "OperationFailed")
    return err
}

span.SetAttribute(tracing.AttrSpanOutputs, result)
span.SetStatus(tracing.SpanStatusOK, "")
```

### 5. Batch Flushing Multiple Traces

Flush multiple traces concurrently:

```go
func flushAll(ctx context.Context, tracers []*tracing.Tracer, client *mlflow.Client) error {
    return tracing.FlushAll(ctx, tracers, client)
}
```

## Attribute Keys Reference

Standard MLflow attribute keys:

```go
const (
    AttrSpanInputs    = "mlflow.spanInputs"      // Span input data
    AttrSpanOutputs   = "mlflow.spanOutputs"     // Span output data
    AttrSpanType      = "mlflow.spanType"        // Span type (auto-set)
    AttrRequestID     = "mlflow.traceRequestId"  // Request identifier
    AttrExperimentID  = "mlflow.experimentId"    // Experiment ID
    AttrTokenUsage    = "mlflow.chat.tokenUsage" // Token usage stats
    AttrChatTools     = "mlflow.chat.tools"      // Available tools
    AttrMessageFormat = "mlflow.message.format"  // Message format
    AttrLLMReasoning  = "mlflow.llm.reasoning"   // LLM reasoning trace
    AttrFunctionName  = "mlflow.spanFunctionName" // Function name
    AttrLinkedPrompts = "mlflow.linkedPrompts"   // Linked prompts
)
```

### Token Usage Example

```go
span.SetAttribute(tracing.AttrTokenUsage, tracing.TokenUsage{
    InputTokens:  100,
    OutputTokens: 50,
    TotalTokens:  150,
})
```

## Truncation Limits

To prevent excessive payload sizes:

- **Attributes**: Max 10KB per attribute (configurable via `MaxAttributeBytes`)
- **Previews**: Max 1000 characters (configurable via `MaxPreviewChars`)

Use truncation helpers:

```go
truncated := tracing.TruncateString(longString, tracing.MaxAttributeBytes)
preview := tracing.TruncatePreview(requestBody)
```

## Thread Safety

All types are thread-safe:

- `Trace.AddSpan()`, `Trace.SetState()` - thread-safe
- `Span.SetAttribute()`, `Span.SetStatus()`, `Span.AddEvent()` - thread-safe
- `Tracer.StartSpan()`, `Tracer.EndSpan()` - thread-safe

## Cleanup and Error Handling

### Always Close Spans

```go
span := tracer.StartSpan("operation", tracing.SpanTypeTool)
defer tracer.EndSpan(span) // Ensures span is closed even on panic
```

### Handle Orphan Spans

End any unclosed spans before flushing:

```go
tracer.EndOrphanSpans()
```

### Check Flush Errors

```go
result := callbacks.GetResult()
if result.FlushError != nil {
    log.Printf("Trace uploaded but flush failed: %v", result.FlushError)
}
```

## Integration with MLflow

The package integrates with MLflow's tracing API:

1. **Trace Info** - Uploaded via `POST /api/2.0/mlflow/traces`
2. **Spans** - Uploaded via OTLP endpoint `/api/2.0/mlflow/traces/otlp`

Both operations are required for complete trace visibility in MLflow UI (MLflow >= 3.4).

## Best Practices

1. **Always set span status** - Explicitly mark success or failure
2. **Use descriptive names** - Make spans easily identifiable
3. **Truncate large payloads** - Prevent excessive memory usage
4. **Close spans in defer** - Ensure cleanup on errors/panics
5. **Set appropriate span types** - Helps MLflow UI categorization
6. **Add metadata** - Enrich traces with context (model, session, etc.)
7. **Handle flush errors** - Don't fail the main operation on trace errors

## Example: Complete Agent Tracing

```go
package main

import (
    "context"
    "log"

    "charm.land/fantasy/mlflow"
    "charm.land/fantasy/tracing"
)

func main() {
    ctx := context.Background()

    // Setup
    client := mlflow.NewClient("http://localhost:5000")
    config := tracing.TracingConfig{
        Client:       client,
        ExperimentID: "0",
        AgentName:    "research-assistant",
        ModelName:    "claude-opus-4-5",
        Tags: map[string]string{
            "user_id": "user-123",
        },
    }

    callbacks := tracing.NewTracingCallbacks(config)

    // Simulate agent execution
    callbacks.OnAgentStart("Research quantum computing")

    // Step 1
    callbacks.OnStepStart(1)
    callbacks.OnToolCall("search", map[string]any{"query": "quantum computing"})
    callbacks.OnToolResult(map[string]any{"results": "..."}, nil)
    callbacks.OnStepFinish()

    // Step 2
    callbacks.OnStepStart(2)
    callbacks.OnLLMStart([]any{"messages": "..."})
    callbacks.OnLLMFinish(1000, 500, 1500, nil)
    callbacks.OnStepFinish()

    // Complete
    err := callbacks.OnAgentFinish(ctx, "Quantum computing is...")
    if err != nil {
        log.Printf("Flush error: %v", err)
    }

    // Check result
    result := callbacks.GetResult()
    log.Printf("Trace ID: %s", result.Trace.TraceID)
}
```

## Time Utilities

Internal time helpers (not exported):

- `nowNanos()` - Current time in nanoseconds since epoch
- `nowMillis()` - Current time in milliseconds since epoch
- `millisToTime()` - Convert milliseconds to `time.Time`
- `millisToDuration()` - Convert milliseconds to `time.Duration`

These are used internally for consistent timestamp handling across traces and spans.
