# Fantasy Tracing

MLflow-compatible distributed tracing for Fantasy agents.

## Overview

The tracing package captures hierarchical execution traces from Fantasy agent runs and exports them to MLflow for visualization and analysis. It provides:

- Automatic span hierarchy: Agent → Step → [LLM | Tool]
- Thread-safe trace collection
- Automatic trace flush on agent completion
- UTF-8 safe preview truncation

## Usage

### Basic Integration with Agent Option

```go
import (
    "charm.land/fantasy"
    "charm.land/fantasy/mlflowclient"
    "charm.land/fantasy/tracing"
)

// Create MLflow client
client, _ := mlflowclient.New("http://localhost:5000")

// Configure tracing
config := tracing.TracingConfig{
    ExperimentID: "my-experiment",
    AgentName:    "my-agent",
    ModelName:    "gpt-4",
}

// Create tracing callbacks
callbacks, tracer := tracing.WrapAgentOptions(client, config)
cbs := callbacks.StreamCallbacks()

// Create agent with built-in tracing callbacks
agent := fantasy.NewAgent(model, fantasy.WithTracingCallbacks(fantasy.TracingCallbackConfig{
    OnAgentStart:  cbs.OnAgentStart,
    OnAgentFinish: cbs.OnAgentFinish,
    OnStepStart:   cbs.OnStepStart,
    OnStepFinish:  cbs.OnStepFinish,
    OnToolCall:    cbs.OnToolCall,
    OnToolResult:  cbs.OnToolResult,
    OnError:       cbs.OnError,
}))

// All Stream calls automatically use the tracing callbacks
result, err := agent.Stream(ctx, fantasy.AgentStreamCall{
    Prompt: "Hello, world!",
})
```

### Per-Call Integration

You can also set callbacks per-call if you need more control:

```go
agent := fantasy.NewAgent(model)
result, err := agent.Stream(ctx, fantasy.AgentStreamCall{
    Prompt:        "Hello, world!",
    OnAgentStart:  cbs.OnAgentStart,
    OnAgentFinish: cbs.OnAgentFinish,
    OnStepStart:   cbs.OnStepStart,
    OnStepFinish:  cbs.OnStepFinish,
    OnToolCall:    cbs.OnToolCall,
    OnToolResult:  cbs.OnToolResult,
    OnError:       cbs.OnError,
})
```

### Manual Trace Management

```go
tracer := tracing.NewTracer(client, "experiment-id")

// Start a new trace
ctx, trace, _ := tracer.NewTrace(ctx, tracing.TracingConfig{
    AgentName: "my-agent",
})

// Create spans
span, ctx := tracer.StartSpan(ctx, "operation", tracing.SpanTypeTool)
span.SetAttribute("key", "value")
span.End()

// End and flush trace
tracer.EndTrace(ctx, tracing.TraceStateOK)
tracer.Flush(ctx, trace)
```

## Span Types

- `SpanTypeAgent` - Root agent execution span
- `SpanTypeLLM` - Language model inference
- `SpanTypeTool` - Tool execution
- `SpanTypeChain` - Step/chain execution
- `SpanTypeRetriever` - Document retrieval
- `SpanTypeEmbedding` - Embedding generation

## Span Hierarchy

```
Agent (root)
├── Step 1 (chain)
│   ├── LLM (inferred)
│   └── Tool: search (tool)
├── Step 2 (chain)
│   ├── LLM (inferred)
│   ├── Tool: read_file (tool)
│   └── Tool: write_file (tool)
└── Step 3 (chain)
    └── LLM (inferred)
```

## Configuration

### TracingConfig

| Field | Type | Description |
|-------|------|-------------|
| ExperimentID | string | MLflow experiment ID (required) |
| AgentName | string | Name for the agent span |
| ModelName | string | Model identifier for LLM spans |
| SessionID | string | Session ID (auto-generated if empty) |
| Tags | map[string]string | Custom trace tags |
| FlushTimeout | time.Duration | Trace upload timeout (default: 10s) |

### TracerOption

- `WithFlushTimeout(d time.Duration)` - Set flush timeout

## Viewing Traces

After running an agent with tracing enabled, view traces in the MLflow UI:

```bash
# Start MLflow server
task mlflow:start

# Open browser
open http://localhost:5000
```

Navigate to your experiment and click on a trace to see the span hierarchy and timing.

## Error Handling

The package includes panic recovery to ensure tracing errors don't crash your agent:

- All callback methods recover from panics
- Orphan spans are automatically cleaned up on trace end
- Context cancellation is respected

## Thread Safety

All operations are thread-safe:
- Trace and span state is protected by mutexes
- Multiple spans can be created concurrently
- Flush operations are synchronized
