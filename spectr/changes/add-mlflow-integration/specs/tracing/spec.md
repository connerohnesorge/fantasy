## ADDED Requirements

### Requirement: Span and Trace ID Generation

The system SHALL generate unique identifiers for traces and spans following OpenTelemetry conventions.

#### Scenario: Trace ID generation
- GIVEN a new trace is being created
- WHEN `tracer.StartTrace(ctx)` is called
- THEN a 16-byte (128-bit) trace ID is generated using crypto/rand
- AND the trace ID is formatted as 32 lowercase hexadecimal characters
- AND the MLflow trace request ID format is `tr-<hex_trace_id>` (e.g., "tr-1234567890abcdef...")

#### Scenario: Span ID generation
- GIVEN a new span is being created
- WHEN a span is started within a trace
- THEN an 8-byte (64-bit) span ID is generated using crypto/rand
- AND the span ID is formatted as 16 lowercase hexadecimal characters
- AND span IDs are unique within a trace (collision probability ~1/2^64)

#### Scenario: ID generation failure
- GIVEN crypto/rand fails to provide entropy
- WHEN ID generation is attempted
- THEN the error is propagated up (do not fall back to pseudo-random)
- AND the trace/span creation fails with a clear error message

### Requirement: Concurrency and Thread Safety

The system SHALL support concurrent agent executions and tool calls safely.

#### Scenario: Concurrent tool calls within a step
- GIVEN multiple tools are called concurrently within a single step
- WHEN tool spans are created simultaneously from different goroutines
- THEN each tool gets its own span with correct parent (step span)
- AND spans are thread-safe (use sync.Mutex for span state)
- AND no data races occur

#### Scenario: Multiple concurrent traces
- GIVEN multiple agent executions occur concurrently
- WHEN each has tracing enabled
- THEN each agent gets an independent trace
- AND trace context is stored in context.Context (not global state)
- AND traces do not interfere with each other

#### Scenario: Callback execution order
- GIVEN callbacks fire in unpredictable order due to concurrency
- WHEN OnToolCall and OnToolResult fire out of order
- THEN the tracer handles this gracefully
- AND spans are matched by tool call ID (not by order)

### Requirement: Context Propagation

The system SHALL propagate trace context through Go context.Context.

#### Scenario: Context storage
- GIVEN an active span exists
- WHEN context is passed to child operations
- THEN the current span is retrievable via `tracing.SpanFromContext(ctx)`
- AND the context key is a package-private type to avoid collisions

#### Scenario: Parent span discovery
- GIVEN a new span needs to be created
- WHEN the parent is determined
- THEN the parent is found by checking context for active span
- AND if no span in context, the span becomes a root span
- AND parent_span_id is set to the parent's span ID (or nil for root)

#### Scenario: Context cancellation
- GIVEN a trace is in progress
- WHEN the context is cancelled
- THEN any in-progress spans are ended with ERROR status
- AND the trace is flushed with partial data
- AND status message includes "context cancelled"

### Requirement: Orphan Span Cleanup

The system SHALL handle spans that never receive explicit end calls.

#### Scenario: Span timeout
- GIVEN a span has been active for longer than a configurable timeout
- WHEN the tracer detects the timeout (checked during trace flush or periodic cleanup)
- THEN the span is ended with ERROR status
- AND status message is "span timed out (no EndSpan called)"
- AND default timeout is 5 minutes

#### Scenario: Trace flush with orphan spans
- GIVEN EndTrace is called but some spans were never ended
- WHEN the trace is finalized
- THEN orphan spans are automatically ended with ERROR status
- AND a warning is logged identifying the orphan spans
- AND the trace is still sent to MLflow

#### Scenario: Panic recovery
- GIVEN a panic occurs during agent execution
- WHEN the panic propagates up
- THEN defer handlers end active spans with ERROR status
- AND status message includes panic information
- AND the trace is flushed before the panic continues

### Requirement: Tracer Type

The system SHALL provide a Tracer for creating MLflow-compatible traces and spans.

#### Scenario: Tracer creation
- GIVEN an MLflow client and experiment ID
- WHEN a Tracer is created
- THEN it is configured to create traces in the specified experiment

#### Scenario: Trace initialization
- GIVEN a Tracer instance
- WHEN `tracer.StartTrace(ctx)` is called
- THEN a new trace is created with a unique ID
- AND request time is set to current timestamp
- AND state is set to IN_PROGRESS

#### Scenario: Trace completion
- GIVEN an active trace
- WHEN `tracer.EndTrace(ctx)` is called
- THEN execution duration is calculated
- AND state is set based on AgentResult error status (see State Determination below)
- AND the trace is sent to MLflow server

#### Scenario: Trace state determination
- GIVEN an agent execution completes
- WHEN the final AgentResult is available
- THEN state is set to ERROR if AgentResult.Error is non-nil
- AND state is set to OK if AgentResult.Error is nil
- AND state is set to ERROR if context was cancelled or timed out

### Requirement: Span Hierarchy

The system SHALL create hierarchical spans matching agent execution structure.

The span hierarchy follows: Agent -> Step -> (LLM | Tool)

Note: LLM spans are inferred from step timing since Fantasy callbacks do not provide
direct LLM call/result hooks. This approximation captures the primary LLM interaction
per step but may not reflect retry attempts or multi-model scenarios.

#### Scenario: Agent span
- GIVEN an agent execution starts via the tracing wrapper
- WHEN the root span is created
- THEN span type is set to AGENT
- AND span name is set to "agent" (or custom name from TracingConfig.AgentName)
- AND span becomes parent for all child spans

#### Scenario: Step span
- GIVEN an agent step executes
- WHEN a step span is created under the agent span
- THEN span type is set to CHAIN
- AND span name includes step number (e.g., "step-1")
- AND parent_span_id references the agent span

#### Scenario: LLM span (inferred)
- GIVEN a step span is active
- WHEN the step completes (OnStepFinish fires)
- THEN an LLM span is created retroactively as child of step span
- AND span type is set to LLM
- AND span name includes the model name from TracingConfig.ModelName
- AND span timing is derived from step start to step finish (approximation)
- AND token usage is captured from step result if available

#### Scenario: Tool span
- GIVEN a tool is executed during a step
- WHEN a tool span is created
- THEN span type is set to TOOL
- AND span name is the tool name
- AND parent_span_id references the step span

### Requirement: Span Attributes

The system SHALL capture comprehensive span attributes.

#### Scenario: Input attributes
- GIVEN a span with inputs
- WHEN the span is created
- THEN `mlflow.spanInputs` attribute contains JSON-serialized inputs
- AND inputs are truncated if they exceed 10,240 bytes

#### Scenario: Output attributes
- GIVEN a span completing with outputs
- WHEN the span is ended
- THEN `mlflow.spanOutputs` attribute contains JSON-serialized outputs
- AND outputs are truncated if they exceed 10,240 bytes

#### Scenario: Token usage attributes
- GIVEN an LLM span with usage information
- WHEN the span is ended
- THEN `mlflow.chat.tokenUsage` attribute contains token counts as JSON:
```go
type TokenUsage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
    TotalTokens  int `json:"total_tokens"`
}
```
- AND token usage is extracted from the LLM response's Usage field
- AND if the LLM provider does not return usage, the attribute is omitted

#### Scenario: Error attributes
- GIVEN a span that errors
- WHEN the span is ended with error
- THEN span status is set to STATUS_CODE_ERROR
- AND status message contains error description

### Requirement: Agent Callback Integration

The system SHALL integrate with Fantasy's existing agent callbacks via a tracing wrapper.

#### Scenario: WithTracing option
- GIVEN an agent configuration
- WHEN `fantasy.WithTracing(config)` option is applied
- THEN tracing is enabled via a call wrapper pattern
- AND the wrapper intercepts Run/Stream calls to capture context
- AND no changes to Fantasy's core API are required

#### Scenario: TracingConfig structure
- GIVEN a tracing configuration
- WHEN TracingConfig is created
- THEN it contains: Client (*mlflowclient.Client), ExperimentID (string)
- AND optional: AgentName (string), ModelName (string), SessionID (string), Tags (map[string]string)
- AND SessionID defaults to a new UUID if not provided

#### Scenario: Tracing wrapper initialization
- GIVEN tracing is enabled via WithTracing(config)
- WHEN agent.Run() or agent.Stream() is called
- THEN the wrapper intercepts the call before delegating to the agent
- AND creates a new trace with context from TracingConfig
- AND captures the initial prompt from the call parameters
- AND creates the root agent span

#### Scenario: OnStepStart callback
- GIVEN tracing is enabled and agent is running
- WHEN a step starts (OnStepStart fires)
- THEN a step span is created as child of agent span
- AND step number is captured
- AND step start timestamp is recorded for LLM span inference

#### Scenario: OnStepFinish callback
- GIVEN a step span is active
- WHEN step completes (OnStepFinish fires)
- THEN an inferred LLM span is created with step timing
- AND step span is ended
- AND step result content is captured in outputs

#### Scenario: OnToolCall callback
- GIVEN a step span is active
- WHEN a tool is called (OnToolCall fires)
- THEN a tool span is created as child of step span
- AND tool name and input are captured

#### Scenario: OnToolResult callback
- GIVEN a tool span is active
- WHEN tool execution completes (OnToolResult fires)
- THEN tool span is ended
- AND tool result is captured in outputs

#### Scenario: Tracing wrapper completion
- GIVEN tracing is enabled and agent completes
- WHEN the wrapped Run/Stream call returns
- THEN the wrapper captures the AgentResult
- AND agent span is ended with appropriate status
- AND trace is finalized and sent to MLflow

### Requirement: Content Capture

The system SHALL capture full message content in traces.

#### Scenario: User message capture
- GIVEN a user prompt in the conversation
- WHEN the trace is created
- THEN request_preview contains the user prompt
- AND full prompt is in the agent span inputs

#### Scenario: Assistant message capture
- GIVEN assistant response content
- WHEN the step span is ended
- THEN response_preview contains the final response
- AND full response is in the step span outputs

#### Scenario: Reasoning content capture
- GIVEN reasoning tokens in the response (e.g., Claude's extended thinking)
- WHEN reasoning content is available in the LLM response
- THEN reasoning text is captured in `mlflow.llm.reasoning` span attribute
- AND the attribute contains the full reasoning text (subject to truncation limits)
- AND if no reasoning content is present, the attribute is omitted

### Requirement: Span Attribute Key Reference

The system SHALL use consistent attribute keys matching MLflow conventions.

#### Scenario: Attribute key constants
- GIVEN the tracing package
- WHEN span attributes are set
- THEN the following attribute keys are used:
```go
const (
    // Core span attributes
    AttrSpanInputs     = "mlflow.spanInputs"      // JSON-serialized inputs
    AttrSpanOutputs    = "mlflow.spanOutputs"     // JSON-serialized outputs
    AttrSpanType       = "mlflow.spanType"        // Span type (AGENT, LLM, TOOL, etc.)
    AttrRequestID      = "mlflow.traceRequestId"  // MLflow trace request ID
    AttrExperimentID   = "mlflow.experimentId"    // Experiment ID

    // LLM-specific attributes
    AttrTokenUsage     = "mlflow.chat.tokenUsage" // Token counts
    AttrChatTools      = "mlflow.chat.tools"      // Available tools
    AttrMessageFormat  = "mlflow.message.format"  // Message format (openai, anthropic, etc.)
    AttrLLMReasoning   = "mlflow.llm.reasoning"   // Extended thinking/reasoning content

    // Tracing metadata
    AttrFunctionName   = "mlflow.spanFunctionName" // Function/tool name
    AttrLinkedPrompts  = "mlflow.linkedPrompts"    // Linked prompt versions
)
```

#### Scenario: Tool call content capture
- GIVEN a tool call in the response
- WHEN the tool span is created
- THEN tool name and JSON input are in span inputs
- AND tool result is in span outputs

### Requirement: Trace Metadata

The system SHALL attach relevant metadata to traces.

#### Scenario: Model metadata
- GIVEN the agent uses a specific model
- WHEN the trace is created
- THEN trace_metadata includes the model identifier

#### Scenario: Session metadata
- GIVEN a TracingConfig with optional SessionID
- WHEN the trace is created
- THEN if SessionID was provided in config, trace_metadata includes `mlflow.trace.session` with that value
- AND if SessionID was not provided, a new UUID v4 is generated and used

#### Scenario: Custom tags
- GIVEN custom tags in tracing config
- WHEN the trace is created
- THEN tags are attached to the trace

### Requirement: Trace Flushing

The system SHALL ensure traces are reliably sent to MLflow.

#### Scenario: Automatic flush on completion
- GIVEN an agent execution completes
- WHEN the trace is finalized
- THEN the trace is sent to MLflow server
- AND any pending spans are included

#### Scenario: Error handling on flush
- GIVEN an MLflow API error during flush
- WHEN the trace cannot be sent
- THEN the error is logged (using Go standard log package or configured logger)
- AND agent execution is not blocked (flush errors are non-fatal)
- AND the error is available via `TracingResult.FlushError` if caller wants to check

#### Scenario: Partial trace on error
- GIVEN an agent execution errors mid-way
- WHEN the error is handled
- THEN completed spans are still sent
- AND trace state is set to ERROR

#### Scenario: Flush blocking behavior
- GIVEN EndTrace is called
- WHEN the trace is finalized and sent to MLflow
- THEN EndTrace blocks until the HTTP request completes or times out
- AND the timeout is configurable via TracingConfig.FlushTimeout (default: 10 seconds)
- AND if flush times out, an error is logged but no panic occurs

#### Scenario: Program exit handling
- GIVEN the program is about to exit
- WHEN active traces exist
- THEN users should call `tracing.FlushAll(ctx)` before exit
- AND FlushAll waits for all pending traces to be sent
- AND FlushAll has a configurable timeout (default: 30 seconds)

### Requirement: Span Type Enumeration

The system SHALL define span types matching MLflow/OpenTelemetry conventions.

#### Scenario: SpanType constants
- GIVEN the tracing package
- WHEN SpanType constants are defined
- THEN the following types are available:
```go
const (
    SpanTypeAgent     = "AGENT"     // Root span for agent execution
    SpanTypeLLM       = "LLM"       // Language model invocation
    SpanTypeTool      = "TOOL"      // Tool/function execution
    SpanTypeChain     = "CHAIN"     // Step in agent execution chain
    SpanTypeRetriever = "RETRIEVER" // Document retrieval operation
    SpanTypeEmbedding = "EMBEDDING" // Embedding generation
    SpanTypeUnknown   = "UNKNOWN"   // Default/fallback type
)
```

### Requirement: Truncation Behavior

The system SHALL truncate span attributes that exceed MLflow limits.

#### Scenario: Attribute size limit
- GIVEN span inputs or outputs exceed 10,240 bytes when JSON-serialized
- WHEN the attribute is set
- THEN the content is truncated to fit within the limit
- AND a "..." suffix is appended to indicate truncation
- AND truncation happens at UTF-8 character boundaries (never mid-character)
- AND the final size including suffix is <= 10,240 bytes

#### Scenario: Preview length
- GIVEN request_preview or response_preview is generated
- WHEN the content exceeds 1,000 characters (OSS) or 10,000 characters (Databricks)
- THEN the preview is truncated with "..." suffix
- AND the limit is determined by tracking URI (OSS vs Databricks)

### Requirement: Timestamp Precision

The system SHALL use nanosecond precision for span timestamps.

#### Scenario: Timestamp format
- GIVEN span start and end times are recorded
- WHEN timestamps are captured
- THEN they use `time.Now().UnixNano()` for nanosecond precision
- AND they are stored as int64 nanoseconds since Unix epoch
- AND duration is calculated as `end_time_ns - start_time_ns`
