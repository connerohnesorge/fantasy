## ADDED Requirements

### Requirement: Span and Trace ID Generation

The system SHALL generate unique identifiers for traces and spans following OpenTelemetry conventions.

#### Scenario: Trace ID generation
- GIVEN a new trace is being created
- WHEN `tracer.NewTrace()` is called
- THEN a 16-byte (128-bit) trace ID is generated using crypto/rand
- AND the trace ID is formatted as 32 lowercase hexadecimal characters
- AND the MLflow trace request ID format is `tr-<32_hex_chars>` (e.g., "tr-1234567890abcdef1234567890abcdef")
- AND the total trace ID length is 35 characters (3 for "tr-" + 32 hex chars)

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
- AND tool call ID is sourced from the ToolCall.ID field provided by Fantasy callbacks
- AND the ID format follows OpenAI's tool_call ID pattern (e.g., "call_abc123")

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

### Requirement: Core Data Structures

The system SHALL define core data structures for traces and spans.

#### Scenario: Span structure definition
- GIVEN the tracing package
- WHEN the Span type is defined
- THEN it has the following structure:
```go
// Span represents a unit of work within a trace.
type Span struct {
    TraceID     string            // Trace ID this span belongs to (format: tr-<hex>)
    SpanID      string            // Unique span ID within trace (16 hex chars)
    ParentID    string            // Parent span ID (empty for root span)
    Name        string            // Span name (e.g., "agent", "step-1", tool name)
    SpanType    SpanType          // Type: AGENT, LLM, TOOL, CHAIN, etc.
    StartTimeNs int64             // Start time in nanoseconds since epoch
    EndTimeNs   int64             // End time in nanoseconds since epoch
    Status      SpanStatus        // Status: OK, ERROR, or UNSET
    Attributes  map[string]any    // Span attributes (inputs, outputs, etc.)
    Events      []SpanEvent       // Span events (for streaming chunks, errors)
    mu          sync.Mutex        // Protects concurrent access to span state
}

// SpanStatus represents the completion status of a span.
type SpanStatus struct {
    Code        SpanStatusCode // OK, ERROR, or UNSET
    Description string         // Human-readable status message
}

// SpanStatusCode is the status code for a span.
type SpanStatusCode int

const (
    SpanStatusUnset SpanStatusCode = iota
    SpanStatusOK
    SpanStatusError
)

// SpanEvent represents an event that occurred during span execution.
type SpanEvent struct {
    Name       string         // Event name
    Timestamp  int64          // Nanoseconds since epoch
    Attributes map[string]any // Event attributes
}
```

#### Scenario: Trace structure definition
- GIVEN the tracing package
- WHEN the Trace type is defined
- THEN it has the following structure:
```go
// Trace represents a complete trace with metadata and spans.
type Trace struct {
    TraceID          string            // Unique trace ID (format: tr-<hex>)
    ExperimentID     string            // MLflow experiment ID
    RequestTime      int64             // Start time in milliseconds since epoch
    ExecutionDuration int64            // Duration in milliseconds
    State            TraceState        // OK, ERROR, or IN_PROGRESS
    RequestPreview   string            // Truncated user input preview
    ResponsePreview  string            // Truncated response preview
    TraceMetadata    map[string]string // Immutable metadata (model, session, etc.)
    Tags             map[string]string // Mutable tags
    Spans            []*Span           // All spans in the trace
    mu               sync.RWMutex      // Protects concurrent access
}

// TraceState represents the state of a trace.
type TraceState int

const (
    TraceStateInProgress TraceState = iota
    TraceStateOK
    TraceStateError
)
```

#### Scenario: TracingResult structure definition
- GIVEN a trace operation completes
- WHEN results need to be returned
- THEN TracingResult is defined as:
```go
// TracingResult contains the result of a traced agent execution.
type TracingResult struct {
    Trace      *Trace // The completed trace (nil if tracing disabled)
    FlushError error  // Error from flushing to MLflow (nil on success)
}
```

### Requirement: Tracer Type

The system SHALL provide a Tracer for creating MLflow-compatible traces and spans.

#### Scenario: Tracer creation
- GIVEN an MLflow client and experiment ID
- WHEN a Tracer is created
- THEN it is configured to create traces in the specified experiment

#### Scenario: Trace initialization
- GIVEN a Tracer instance
- WHEN `tracer.NewTrace(ctx, config TracingConfig)` is called
- THEN a new Trace struct is created with a unique ID
- AND the context is updated to contain the trace (for SpanFromContext)
- AND request time is set to current timestamp in milliseconds
- AND state is set to IN_PROGRESS
- AND the function returns `(ctx context.Context, trace *Trace, err error)`
- AND the returned context should be used for all subsequent operations in this trace

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

### Requirement: Error Handling

The system SHALL handle error conditions gracefully without panicking.

#### Scenario: Nil MLflow client
- GIVEN TracingConfig.Client is nil
- WHEN tracing is enabled
- THEN a ValidationError is returned immediately
- AND no tracing operations are attempted

#### Scenario: Invalid experiment ID
- GIVEN TracingConfig.ExperimentID is empty
- WHEN a Tracer is created
- THEN a ValidationError is returned with Field="experimentID"
- AND the error message indicates the field is required

#### Scenario: JSON serialization failure
- GIVEN span inputs or outputs contain non-serializable values (channels, functions, circular refs)
- WHEN the value is serialized
- THEN the error is logged at WARN level
- AND a placeholder string "[serialization error]" is used
- AND span recording continues (not aborted)

#### Scenario: Duplicate StartTrace call
- GIVEN a trace is already active in the current context
- WHEN `tracer.StartTrace(ctx)` is called again
- THEN the existing trace is used (no new trace created)
- AND a warning is logged about the duplicate call

#### Scenario: Duplicate EndTrace/EndSpan call
- GIVEN a trace or span has already been ended
- WHEN `EndTrace()` or `EndSpan()` is called again
- THEN the call is a no-op (no error)
- AND a debug log is emitted noting the duplicate call

#### Scenario: Network error during flush
- GIVEN trace data is being sent to MLflow server
- WHEN a network error occurs
- THEN TracingResult.FlushError contains the error
- AND the trace is still returned in TracingResult.Trace
- AND the error is logged at ERROR level
- AND no retry is attempted (retry is the caller's responsibility)

#### Scenario: Invalid token usage data
- GIVEN token usage values are negative or overflow int64
- WHEN token usage is recorded
- THEN the invalid values are logged at WARN level
- AND the span continues with zero token usage

### Requirement: Span Hierarchy

The system SHALL create hierarchical spans matching agent execution structure.

The span hierarchy follows: Agent -> Step -> (LLM, Tool)

LLM and Tool spans are siblings—both are children of the Step span. When a step executes,
it may involve an LLM call followed by zero or more tool calls, all at the same hierarchical level.

Note: LLM spans are created using OnLLMStart/OnLLMFinish callbacks that fire directly before and
after LLM API calls. This provides accurate wall-clock timing including retry attempts.

#### Scenario: Agent span
- GIVEN an agent execution starts (OnAgentStart fires)
- WHEN the root span is created
- THEN span type is set to AGENT
- AND span name is set to TracingConfig.AgentName (default: "agent")
- AND span becomes parent for all child spans (Steps)

#### Scenario: Step span
- GIVEN an agent step executes
- WHEN a step span is created under the agent span
- THEN span type is set to CHAIN
- AND span name includes step number (e.g., "step-1")
- AND parent_span_id references the agent span

#### Scenario: LLM span (with OnLLMStart/OnLLMFinish)
- GIVEN a step is executing
- WHEN OnLLMStart callback fires (immediately before LLM API call)
- THEN an LLM span is created as child of step span
- AND span type is set to LLM
- AND span name is formatted as `llm-<model_name>` (e.g., "llm-gpt-4")
- AND span start time is set to current timestamp (nanoseconds since epoch)
- AND span inputs are captured from the Call parameter (messages, tools, temperature, etc.)
- WHEN OnLLMFinish callback fires (immediately after LLM API call completes)
- THEN LLM span end time is set to current timestamp
- AND LLM span timing is **accurate**: `duration = end_time - start_time` (wall-clock time)
- AND duration **includes retry attempts** (wraps the entire retry sequence)
- AND token usage from the Usage parameter is attached to the LLM span
- AND finish reason from the FinishReason parameter is recorded
- AND if an error occurred, span status is set to ERROR with error description
- AND if no token usage is available (Usage.TotalTokens == 0), token usage attributes are omitted
- NOTE: Only ONE LLM span is created per step (one LLM call per step in Fantasy's execution model)

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

The system SHALL integrate with Fantasy's existing agent callbacks via TracingCallbacks.

**TracingCallbacks** is a callback implementation that:
1. Implements Fantasy's `Callbacks` interface (`OnAgentStart`, `OnAgentFinish`, `OnStepStart`, `OnStepFinish`, `OnToolCall`, `OnToolResult`)
2. Creates and manages MLflow spans in response to agent events
3. Is registered via `fantasy.WithTracing(config)` agent option
4. Maintains internal state mapping callback events to their corresponding spans

#### Scenario: WithTracing option
- GIVEN an agent configuration
- WHEN `fantasy.WithTracing(config)` option is applied
- THEN a TracingCallbacks instance is created and registered with the agent
- AND the callbacks fire during agent execution to create/manage spans
- AND no changes to Fantasy's core API are required

#### Scenario: TracingConfig structure
- GIVEN a tracing configuration
- WHEN TracingConfig is created
- THEN it contains: Client (*mlflow.Client), ExperimentID (string)
- AND optional: AgentName (string), ModelName (string), SessionID (string), Tags (map[string]string)
- AND optional: FlushTimeout (time.Duration, default 10s) for trace upload deadline
- AND SessionID defaults to a new UUID v4 if not provided
- AND FlushTimeout controls how long to wait when sending trace data to MLflow server

#### Scenario: TracingCallbacks initialization (OnAgentStart)
- GIVEN tracing is enabled via WithTracing(config)
- WHEN agent.Run() or agent.Stream() is called
- THEN OnAgentStart callback fires before agent execution begins
- AND creates a new trace with context from TracingConfig
- AND captures the initial prompt from the agent input
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

#### Scenario: TracingCallbacks completion (OnAgentFinish)
- GIVEN tracing is enabled and agent completes
- WHEN OnAgentFinish callback fires
- THEN the callback captures the AgentResult
- AND agent span is ended with appropriate status
- AND trace is finalized and sent to MLflow (blocking with FlushTimeout)

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
- WHEN the content exceeds 1,000 characters
- THEN the preview is truncated with "..." suffix
- AND truncation happens at UTF-8 character boundaries

### Requirement: Timestamp Precision

The system SHALL use nanosecond precision for span timestamps.

#### Scenario: Timestamp format
- GIVEN span start and end times are recorded
- WHEN timestamps are captured
- THEN they use `time.Now().UnixNano()` for nanosecond precision
- AND they are stored as int64 nanoseconds since Unix epoch
- AND duration is calculated as `end_time_ns - start_time_ns`
