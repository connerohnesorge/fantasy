# Tracing

MLflow tracing integration for Fantasy agents, providing observability into agent execution with distributed tracing.

## Features

- **Automatic span capture**: Captures Agent -> Step -> LLM/Tool span hierarchies
- **Thread-safe**: Supports concurrent tool calls within a single step
- **Non-invasive**: Integrates via Fantasy's callback system
- **Error resilient**: Tracing errors don't fail agent execution
- **Automatic flush**: Traces are automatically sent to MLflow on completion

## Installation

```bash
go get charm.land/fantasy/tracing
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "charm.land/fantasy"
    "charm.land/fantasy/mlflowclient"
    "charm.land/fantasy/tracing"
    "charm.land/fantasy/providers/openai"
)

func main() {
    ctx := context.Background()

    // Create MLflow client
    mlflowClient := mlflowclient.New("http://localhost:5000",
        mlflowclient.WithToken("your-token"),
    )

    // Create experiment (one-time setup)
    expID, err := mlflowClient.CreateExperiment(ctx, "my-agent-traces", nil)
    if err != nil {
        log.Fatal(err)
    }

    // Create your agent
    provider, _ := openai.New(openai.WithAPIKey("your-key"))
    model, _ := provider.LanguageModel(ctx, "gpt-4")

    baseAgent := fantasy.NewAgent(model,
        fantasy.WithSystemPrompt("You are a helpful assistant."),
    )

    // Wrap with tracing
    agent, err := fantasy.WithTracing(baseAgent, tracing.TracingConfig{
        Client:       mlflowClient,
        ExperimentID: expID,
        AgentName:    "my-agent",
        ModelName:    "gpt-4",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Use agent normally - tracing is automatic!
    result, err := agent.Generate(ctx, fantasy.AgentCall{
        Prompt: "What's the weather like today?",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Response.Content.Text())
    // Trace is automatically flushed to MLflow
}
```

## Configuration

### TracingConfig

Configure tracing behavior with `TracingConfig`:

```go
config := tracing.TracingConfig{
    // Required: MLflow client
    Client: mlflowClient,

    // Required: Target experiment ID
    ExperimentID: "experiment-123",

    // Optional: Agent name (default: "agent")
    AgentName: "my-agent",

    // Optional: Model name for metadata
    ModelName: "gpt-4",

    // Optional: Session ID (auto-generated UUID v4 if empty)
    SessionID: "session-abc",

    // Optional: Custom tags
    Tags: map[string]string{
        "env":     "production",
        "version": "1.0",
        "team":    "ml-research",
    },

    // Optional: Flush timeout (default: 10s)
    FlushTimeout: 30 * time.Second,
}
```

### Auto-generated Session ID

If you don't provide a `SessionID`, one is automatically generated as a UUID v4:

```go
config := tracing.TracingConfig{
    Client:       mlflowClient,
    ExperimentID: expID,
    // SessionID will be auto-generated (e.g., "550e8400-e29b-41d4-a716-446655440000")
}
```

### Custom Tags

Add custom tags to traces for filtering and organization:

```go
config := tracing.TracingConfig{
    Client:       mlflowClient,
    ExperimentID: expID,
    Tags: map[string]string{
        "user_id":    "user-123",
        "request_id": "req-456",
        "feature":    "search",
    },
}
```

## Span Hierarchy

Tracing creates the following span hierarchy:

```
Agent (root span)
├─ Step 1
│  ├─ LLM (inferred from step timing)
│  ├─ Tool Call 1
│  └─ Tool Call 2
├─ Step 2
│  ├─ LLM (inferred)
│  └─ Tool Call 1
└─ Step 3
   └─ LLM (inferred)
```

### Span Types

- **Agent**: Root span representing the entire agent execution
- **LLM**: Inferred LLM call spans (retroactively created from step timing)
- **Tool**: Tool invocation spans
- **Step**: Logical step in agent reasoning (internal, not exported)

### LLM Span Inference

Fantasy's callback system provides `OnStepStart`/`OnStepFinish` and `OnToolCall`/`OnToolResult`, but does not expose direct LLM call hooks. LLM spans are **inferred retroactively** from step timing:

- LLM span covers the time between step start and first tool call (or step end if no tools)
- Token usage is captured from step response metadata when available
- This provides useful visibility into LLM execution time

## Span Attributes

Spans include rich metadata:

### Agent Span

- `mlflow.spanInputs`: Initial prompt and messages
- `mlflow.spanOutputs`: Final response and step count
- `mlflow.spanType`: "AGENT"

### LLM Span (Inferred)

- `mlflow.spanInputs`: Messages sent to LLM
- `mlflow.spanOutputs`: LLM response
- `mlflow.spanType`: "LLM"
- `gen_ai.request.model`: Model name
- `gen_ai.usage.input_tokens`: Input token count
- `gen_ai.usage.output_tokens`: Output token count

### Tool Span

- `mlflow.spanInputs`: Tool input parameters
- `mlflow.spanOutputs`: Tool result
- `mlflow.spanType`: "TOOL"
- `tool.name`: Tool identifier

## Working with Traces

### Viewing Traces in MLflow

After running your agent, traces are available in the MLflow UI:

1. Navigate to your MLflow server (e.g., `http://localhost:5000`)
2. Select your experiment
3. Click on "Traces" tab
4. View span hierarchy, timing, and attributes

### Searching Traces via API

```go
// Search for traces by tag
traces, err := mlflowClient.SearchTraces(ctx, expID,
    "tags.agent = 'my-agent' AND tags.env = 'production'",
    100, // max results
)

// Get specific trace
trace, err := mlflowClient.GetTrace(ctx, "tr-abc123...")

// Add tags to trace
err = mlflowClient.SetTraceTag(ctx, "tr-abc123...", "reviewed", "true")
```

## Advanced Usage

### Multiple Sessions

Track multiple user sessions:

```go
// Session 1
agent1, _ := fantasy.WithTracing(baseAgent, tracing.TracingConfig{
    Client:       mlflowClient,
    ExperimentID: expID,
    SessionID:    "user-123-session-1",
})

// Session 2
agent2, _ := fantasy.WithTracing(baseAgent, tracing.TracingConfig{
    Client:       mlflowClient,
    ExperimentID: expID,
    SessionID:    "user-123-session-2",
})
```

### Dynamic Tags

Add dynamic tags based on runtime context:

```go
config := tracing.TracingConfig{
    Client:       mlflowClient,
    ExperimentID: expID,
    Tags: map[string]string{
        "request_id": requestID,
        "timestamp":  time.Now().Format(time.RFC3339),
    },
}
```

### Flush Timeout

Control how long to wait for trace flush:

```go
config := tracing.TracingConfig{
    Client:       mlflowClient,
    ExperimentID: expID,
    FlushTimeout: 60 * time.Second, // Wait up to 60s for flush
}
```

### Context Cancellation

Traces respect context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := agent.Generate(ctx, fantasy.AgentCall{
    Prompt: "Complex task that might timeout...",
})

// If context is cancelled, trace is marked as ERROR and flushed
```

## Error Handling

### Tracing Errors

Tracing errors are logged but don't fail agent execution:

```go
result, err := agent.Generate(ctx, fantasy.AgentCall{
    Prompt: "Hello",
})

// err is nil even if trace export failed
// Check logs for tracing errors
```

### Agent Errors

Agent errors are captured in spans:

```go
result, err := agent.Generate(ctx, fantasy.AgentCall{
    Prompt: "Invalid request",
})

// If agent fails:
// - Agent span marked with ERROR status
// - Trace state set to ERROR
// - Error message included in span attributes
// - Trace is flushed to MLflow
```

### Panic Recovery

Panics are caught and logged:

```go
defer func() {
    if r := recover(); r != nil {
        // Tracing wrapper catches panic:
        // - Marks all open spans as ERROR
        // - Flushes trace to MLflow
        // - Re-panics to maintain normal panic behavior
    }
}()
```

## Thread Safety

All tracing operations are thread-safe:

- Concurrent tool calls are tracked by tool call ID
- Span state protected by mutexes
- Safe for parallel agent execution

## Best Practices

### 1. Create Experiment Once

Create experiments during setup, not per-request:

```go
// Good: One-time setup
expID := getOrCreateExperiment(ctx, mlflowClient, "my-agent")

// Then reuse expID for all traced agents
```

### 2. Use Meaningful Names

```go
config := tracing.TracingConfig{
    Client:       mlflowClient,
    ExperimentID: expID,
    AgentName:    "customer-support-agent", // Descriptive name
    ModelName:    "gpt-4",                  // Include model version
}
```

### 3. Add Contextual Tags

```go
config := tracing.TracingConfig{
    Client:       mlflowClient,
    ExperimentID: expID,
    Tags: map[string]string{
        "user_id":     userID,
        "session_id":  sessionID,
        "environment": env,
        "version":     appVersion,
    },
}
```

### 4. Monitor Flush Timeouts

Set appropriate flush timeouts based on network conditions:

```go
// Local MLflow: shorter timeout
config.FlushTimeout = 5 * time.Second

// Remote MLflow: longer timeout
config.FlushTimeout = 30 * time.Second
```

### 5. Handle Context Properly

Always use context with timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

result, err := agent.Generate(ctx, fantasy.AgentCall{
    Prompt: "...",
})
```

## Integration with Evaluation

Traces can be used with the evaluation framework:

```go
import (
    "charm.land/fantasy/eval"
    "charm.land/fantasy/tracing"
)

// Agent with tracing
agent, _ := fantasy.WithTracing(baseAgent, tracingConfig)

// Prediction function captures trace
predictFunc := func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    result, err := agent.Generate(ctx, fantasy.AgentCall{
        Prompt: inputs["question"].(string),
    })
    if err != nil {
        return nil, err
    }

    // Return with trace for agent-specific scorers
    return map[string]any{
        "answer": result.Response.Content.Text(),
        "trace":  result.Trace, // Include trace
    }, nil
}

// Use agent-specific scorers
scorers := []eval.Scorer{
    eval.NewToolCallTrajectory([]string{"search", "summarize"}),
    eval.NewStepValidation(1, 3),
}

results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(predictFunc),
)
```

## Troubleshooting

### Traces Not Appearing in MLflow

1. **Check MLflow server**: Ensure MLflow is running and accessible
2. **Verify experiment ID**: Confirm experiment exists
3. **Check authentication**: Verify token is valid
4. **Check logs**: Look for trace export errors
5. **Increase flush timeout**: Network issues may require longer timeout

```go
config.FlushTimeout = 60 * time.Second
```

### Incomplete Traces

1. **Context cancelled**: Ensure context isn't cancelled prematurely
2. **Panic without recovery**: Catch panics and allow trace flush
3. **Process termination**: Ensure process doesn't exit before flush completes

### High Overhead

1. **Reduce parallelism**: If running many agents concurrently
2. **Increase flush timeout**: Reduce retry overhead
3. **Use local MLflow**: Network latency can add overhead

## Performance Considerations

- **Span creation**: Minimal overhead (~1-2% CPU)
- **Attribute serialization**: JSON marshaling for span inputs/outputs
- **Network I/O**: Trace export on flush (async, blocking with timeout)
- **Attribute size limit**: 10,240 bytes per attribute (truncated with "..." suffix)

## API Reference

For detailed API documentation, see:
- [pkg.go.dev documentation](https://pkg.go.dev/charm.land/fantasy/tracing)
- [TracingConfig](./config.go)
- [Tracer](./tracer.go)
- [Callbacks](./callbacks.go)

## License

Part of the [Fantasy](https://github.com/charmbracelet/fantasy) project.
