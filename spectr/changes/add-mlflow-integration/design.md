## Context

Fantasy is a Go library for building AI agents. Users need observability and evaluation capabilities similar to Python agentic libraries (LangChain, DSPy) that integrate with MLflow. This design covers proto-based client generation, tracing integration, and a decoupled evaluation framework.

### Stakeholders
- Fantasy library users building AI agents
- Crush (Charm's AI coding agent) which uses Fantasy
- Developers debugging agent behavior
- Teams evaluating agent quality systematically

### Constraints
- Must work with OSS MLflow v3.8+ (no Databricks-specific features)
- Go-native implementation (no CGo, no Python dependencies)
- Evaluation framework must work standalone (MLflow is optional backend)
- Must not break existing Fantasy API

### Terminology Glossary

This section clarifies terminology used across specifications:

| Term | Package | Description |
|------|---------|-------------|
| **State** | tracing | Trace-level state: `IN_PROGRESS`, `OK`, `ERROR` (TraceState) |
| **Status** | tracing | Span-level status: `UNSET`, `OK`, `ERROR` (SpanStatus) |
| **Score** | eval | Local evaluation result with Value, Rationale, Error |
| **Assessment** | mlflow | MLflow API entity for trace annotations (maps from Score) |
| **Trace ID** | tracing | Full trace identifier with `tr-` prefix (e.g., `tr-abc123...`) |
| **Request ID** | tracing | Deprecated alias for Trace ID (used in MLflow v2 API) |

### Type Hierarchy Clarification

The tracing system uses distinct types for construction vs. transport:

| Type | Package | Purpose |
|------|---------|---------|
| `Trace` | tracing | In-memory trace builder; holds spans, state, metadata during execution |
| `Span` | tracing | In-memory span data; attached to a Trace |
| `Tracer` | tracing | Factory/manager that creates Traces; not a data type |
| `TracingCallbacks` | tracing | Callback handler that manages trace lifecycle |
| `mlflow.Trace` | mlflow | Wire format wrapper around proto type for API transport |

**Data Flow**:
```
TracingCallbacks creates → tracing.Trace (with Spans)
                            ↓ ConvertTrace()
                        mlflow.Trace
                            ↓ client.StartTrace()
                        HTTP/JSON to MLflow server
```

**Key Points**:
- Users interact with `TracingConfig` to enable tracing
- `Trace` and `Span` are internal data structures managed by `TracingCallbacks`
- `Tracer` is a helper that generates IDs and creates empty Trace instances
- Conversion to `mlflow.Trace` happens automatically during flush

## Goals / Non-Goals

### Goals
- Type-safe Go client generated from MLflow protos
- Full agent execution tracing with span hierarchy
- Decoupled evaluation framework with Go and LLM-as-judge scorers
- Configurable parallelism for evaluation runs
- Clean integration with Fantasy's existing callback system

### Non-Goals
- MLflow UI compatibility (results don't need perfect UI rendering)
- gRPC transport (REST only, MLflow doesn't expose gRPC)
- Databricks-specific features (UC Schema, V4 APIs)
- Model registry integration
- Real-time streaming of traces

## Decisions

### Decision 1: Proto-based Code Generation with Buf

**What**: Use Buf toolchain to generate Go structs from MLflow proto files.

**Why**:
- Type safety from proto definitions
- Automatic JSON marshaling via `protojson`
- Easy regeneration when MLflow API changes
- OTel span types included for free
- ~300 lines of client code vs ~2000+ manual struct definitions

**Alternatives considered**:
- Hand-written structs: More work, error-prone, hard to maintain
- OpenAPI codegen: MLflow doesn't publish OpenAPI specs
- Raw protoc: Works but Buf provides better dependency management

### Decision 2: REST Client with protojson Marshaling

**What**: Thin REST wrapper that uses protojson for request/response serialization.

**Why**:
- MLflow exposes REST API, not gRPC
- protojson handles proto3 JSON mapping correctly
- Minimal code surface area
- Generated types ensure API compatibility

**Authentication**: Bearer token authentication via `WithToken(token)` option.

**Timeout and Retry Defaults**:
- Default request timeout: 30 seconds
- Retry configuration (for 5xx and connection errors):
  - Maximum retries: 3
  - Initial delay: 1 second
  - Backoff multiplier: 2x (exponential)
  - Maximum delay: 10 seconds
  - Jitter: ±10% randomization

**Pattern**:
```go
func (c *Client) doRequest(ctx context.Context, method, path string,
    req, resp proto.Message) error {
    data, _ := protojson.Marshal(req)
    // ... HTTP request with timeout and retry ...
    return protojson.Unmarshal(respData, resp)
}
```

### Decision 3: Evaluation Framework with Optional MLflow Export

**What**: Evaluation framework uses Go-native types with optional MLflow export via converters.

**Why**:
- Evaluation runs without requiring an MLflow server (results can be local-only)
- Go-native types are more ergonomic (`Scorer` interface, `Dataset` struct)
- Proto types are for serialization to MLflow, not business logic
- Easier testing with Go types

**Note on Dependencies**: The eval package imports the tracing package for agent-specific scorers
(ToolCallTrajectory, StepValidation) that need to inspect `*tracing.Trace` data. This coupling is
intentional for type safety. Users who don't need agent-specific scorers can still use the eval
package without ever calling MLflow APIs—the dependency is compile-time only, not runtime.

**Architecture**:
```
eval/
├── scorer.go          # Scorer interface + built-in heuristic scorers
├── heuristic.go       # Heuristic scorers: ExactMatch, Contains, Regex, JSONMatch, NumericRange
├── agent_scorers.go   # Agent-specific scorers: ToolCallTrajectory, StepValidation
├── dataset.go         # Dataset and test case types
├── evaluator.go       # Evaluation runner
├── llm_judge.go       # LLM-as-judge scorers (Correctness, Guidelines, Relevance, Groundedness)
└── mlflow_export.go   # Converters to proto/Assessment for MLflow export
```

**Heuristic Scorers** (code-based, no LLM required):
- `ExactMatch`: String equality comparison
- `Contains`: Substring matching
- `Regex`: Pattern matching
- `JSONMatch`: Structural JSON comparison (configurable null handling)
- `NumericRange`: Value within bounds check

**Agent-specific Scorers** (code-based, require `*tracing.Trace`):
- `ToolCallTrajectory`: Tool call sequence validation
- `StepValidation`: Step count and content validation

### Decision 4: Callback-based Tracing Integration

**What**: Integrate tracing via Fantasy's existing callback system.

**Why**:
- Non-invasive integration using existing Fantasy callback infrastructure
- Users opt-in with `WithTracing(config)`
- Callbacks already exist for step/tool events (OnStepStart, OnStepFinish, OnToolCall, OnToolResult)
- No changes to core agent logic or Run/Stream method signatures

**Implementation**: Tracing is implemented entirely via Fantasy's callback system:
1. `WithTracing(config)` registers a `TracingCallbacks` implementation with the agent
2. `TracingCallbacks` implements Fantasy's `Callbacks` interface
3. Each callback method creates/updates spans and manages trace state
4. No wrapper intercepts Run/Stream calls; the agent runs normally with callbacks registered

**Span Hierarchy**: `Agent → Step → [LLM | Tool]`

LLM and Tool spans are siblings—both are children of the Step span. When a step executes,
it may involve an LLM call followed by zero or more tool calls, all at the same hierarchical level.

**LLM Span Tracking with OnLLMStart/OnLLMFinish**: Fantasy's callback system has been extended with
dedicated LLM-level callbacks for accurate span timing:
- **OnLLMStart** fires immediately before the LLM API call (Generate or Stream)
- **OnLLMFinish** fires immediately after the LLM API call completes
- LLM span timing is **precise** - wall-clock time from API call start to completion
- Duration includes retry attempts (wraps the entire retry sequence)
- Token usage is captured at the exact moment it's available (from StreamPartTypeFinish or Generate response)

**Unified Callbacks**: The same OnLLMStart/OnLLMFinish callbacks work for both streaming (Stream) and
non-streaming (Generate) modes, providing consistent timing across execution modes.

**Callback Flow**:
```
Agent.Run() or Agent.Stream() starts
  → OnAgentStart callback → creates Agent span (root)
    → OnStepStart callback → creates Step span (child of Agent)
      → OnLLMStart callback → creates LLM span (child of Step)
        [LLM API call happens - Generate() or Stream()]
      → OnLLMFinish callback → ends LLM span with usage and timing
      → [Optional] OnToolCall callback → creates Tool span (child of Step)
      → [Optional] OnToolResult callback → ends Tool span
    → OnStepFinish callback → ends Step span
  → OnAgentFinish callback → ends Agent span, flushes trace to MLflow
Agent execution returns
```

**SessionID Generation**:
- If `SessionID` is empty in `TracingConfig`, a new UUID v4 is generated once per agent instance
- The UUID is generated during `NewAgent()` when processing the `WithTracing()` option
- All traces from that agent instance share the same SessionID (useful for multi-turn conversations)
- To use different SessionIDs per call, create new agent instances or explicitly set SessionID

**Pattern**:
```go
agent := fantasy.NewAgent(model,
    fantasy.WithTracing(tracing.TracingConfig{
        Client:       mlflowClient,
        ExperimentID: "my-experiment",
        AgentName:    "my-agent",      // optional, defaults to "agent"
        ModelName:    "gpt-4",         // optional
        SessionID:    "session-123",   // optional, auto-generated UUID v4 if not set
        Tags:         map[string]string{"env": "prod"},  // optional custom tags
    }),
)
```

**Package Structure Note**: `fantasy.WithTracing()` is in the fantasy package for consistency with
other agent options (e.g., `fantasy.WithTools()`, `fantasy.WithSystemPrompt()`). The `TracingConfig`
struct and `TracingCallbacks` implementation are defined in the `tracing/` package. This avoids
circular dependencies: `fantasy` imports `tracing` for the config type, but `tracing` does not
import `fantasy`.

### Decision 5: Configurable Parallelism for Evaluation

**What**: Default sequential evaluation with opt-in parallelism.

**Why**:
- Sequential is easier to debug
- Parallel is faster for large datasets
- LLM API rate limits may require throttling
- Users control concurrency based on their needs

**Pattern**:
```go
results := evaluator.Run(ctx, dataset, scorers,
    eval.WithParallelism(10),
    eval.WithTimeout(30*time.Second),
)
```

## Risks / Trade-offs

### Risk: Proto Files Drift
- **Risk**: MLflow updates protos, our copies become stale
- **Mitigation**: Pin to MLflow version, document update process, add Taskfile target for proto sync
- **Proto Update Workflow**:
  1. Clone the MLflow repository at the desired version:
     ```bash
     git clone --depth 1 --branch v3.8.0 https://github.com/mlflow/mlflow.git mlflow-ref/mlflow
     ```
  2. Copy and curate protos from `mlflow-ref/mlflow/protos/` to `proto/mlflow/`
  3. Strip ScalaPB extensions and regenerate Go code
  4. Update version comment in `proto/buf.yaml` (e.g., `# MLflow v3.8.0`)

### Risk: protojson Edge Cases
- **Risk**: protojson may not handle all MLflow JSON quirks
- **Mitigation**: Add test cases from MLflow examples, custom unmarshal for edge cases if needed

### Trade-off: Separate Types for Eval vs Proto
- **Trade-off**: Converters add code, but eval framework is more usable standalone
- **Decision**: Accept converter overhead for better ergonomics

### Trade-off: REST vs OTLP for Traces
- **Trade-off**: OTLP is more standard, but REST is simpler and documented
- **Decision**: Use REST API with proto types; can add OTLP later if needed

### Constraint: Span Attribute Size Limits
- **Constraint**: MLflow limits span inputs/outputs to 10,240 bytes
- **Decision**: Truncate span attributes that exceed limit with suffix "..."

### Constraint: Timestamp Precision
- **Span timestamps**: Use nanoseconds (`time.Now().UnixNano()`) for `StartTimeNs`, `EndTimeNs`
- **Trace timestamps**: Use milliseconds for `RequestTime` (MLflow API requirement)
- **Duration**: Calculated in the appropriate unit for each type
- **Conversion**: `milliseconds = nanoseconds / 1_000_000`

### Edge Case: Concurrent Agent Calls
- **Scenario**: Multiple goroutines call agent.Run() simultaneously
- **Handling**: Each call gets an independent trace via context-local storage
- **Key**: Trace state is stored in `context.Context`, not global variables
- **Risk**: Memory usage scales with concurrent executions; no automatic limits

### Edge Case: Streaming Responses
- **Scenario**: Agent uses Stream() instead of Run()
- **Handling**: Span outputs capture final accumulated content, not individual chunks
- **Timing**: LLM span ends when streaming completes (not when first token arrives)
- **Token usage**: May not be available until stream completes (provider-dependent)

### Edge Case: Panic During Execution
- **Scenario**: Agent code panics during execution
- **Handling**: defer/recover in callbacks captures panic, ends all open spans with ERROR
- **Trace state**: Set to ERROR with panic message in status description
- **Re-throw**: Panic is re-raised after trace flush to preserve expected behavior

## Migration Plan

No migration needed - this is purely additive functionality.

### Adoption Path
1. Users add `mlflow` dependency
2. Configure client with MLflow server URL
3. Enable tracing with `WithTracing()` option
4. Run evaluations and optionally export to MLflow

### Rollback
- Remove `WithTracing()` option to disable tracing
- Evaluation framework works without MLflow export

## Resolved Questions

1. **Proto Versioning**: Should we track MLflow version in proto/buf.yaml comments?
   - **Resolved**: Yes, add `# MLflow v3.8.0` comment to track source version

2. **Scorer Naming**: Should built-in scorers match MLflow Python names exactly?
   - **Resolved**: No, use Go idiomatic names (`Correctness`, `Guidelines`, `Relevance`, `Groundedness`)

3. **Dataset Format**: JSON vs YAML vs both for file-based datasets?
   - **Resolved**: JSON only (matches MLflow format)

## Package Structure

```
fantasy/
├── proto/
│   ├── buf.yaml                    # Buf configuration
│   ├── buf.gen.yaml                # Code generation config
│   ├── mlflow/                     # Curated MLflow protos
│   │   ├── service.proto           # Core trace/assessment messages
│   │   ├── assessments.proto       # Assessment types
│   │   ├── experiments.proto       # Experiment/run messages
│   │   └── scorers.proto           # Scorer registration messages
│   ├── opentelemetry/proto/        # OTel protos (downloaded)
│   │   └── trace/v1/trace.proto
│   ├── scalapb/                    # Stub for ScalaPB options
│   │   └── scalapb.proto
│   └── gen/                        # Generated code (git-ignored)
│       └── mlflow/
│           ├── service.pb.go
│           ├── assessments.pb.go
│           ├── experiments.pb.go
│           └── scorers.pb.go
│
├── mlflow/                   # REST client package
│   ├── client.go                   # HTTP client with protojson
│   ├── options.go                  # Functional options pattern
│   ├── traces.go                   # Trace API methods
│   ├── runs.go                     # Run API methods (Create, Get, Update, LogBatch, Search)
│   ├── experiments.go              # Experiment API methods
│   ├── assessments.go              # Assessment API methods
│   ├── scorers.go                  # Scorer registration API
│   ├── tags.go                     # Tag operations (SetTraceTag, DeleteTraceTag, SetRunTag)
│   └── errors.go                   # API error types (APIError)
│
├── eval/                           # Evaluation framework
│   ├── scorer.go                   # Scorer interface + built-in heuristic scorers
│   ├── heuristic.go                # Heuristic scorers: ExactMatch, Contains, Regex, JSONMatch, NumericRange
│   ├── agent_scorers.go            # Agent-specific scorers: ToolCallTrajectory, StepValidation
│   ├── dataset.go                  # Dataset and test case types
│   ├── evaluator.go                # Evaluation runner
│   ├── llm_judge.go                # LLM-as-judge scorers (Correctness, Guidelines, Relevance, Groundedness)
│   └── mlflow_export.go            # Converters to proto/Assessment for MLflow export
│
├── tracing/                        # Tracing integration
│   ├── tracer.go                   # Trace/span builder
│   ├── callbacks.go                # Agent callback integration
│   ├── types.go                    # Trace, Span, TokenUsage structs
│   └── config.go                   # TracingConfig struct
│
└── agent.go                        # WithTracing() option (fantasy package)
```

## Cross-Document Consistency

This section documents decisions that ensure consistency across all spec files.

### Trace Type Field Mapping

| `tracing.Trace` Field | `mlflow.Trace` Field | Proto Field | Notes |
|----------------------|---------------------------|-------------|-------|
| TraceID | TraceID | request_id | Format: `tr-<32hex>` |
| ExperimentID | ExperimentID | experiment_id | String |
| RequestTime | RequestTime | timestamp_ms | Milliseconds |
| State | State | state | IN_PROGRESS, OK, ERROR |
| Spans | Spans | spans | Array of Span |
| Tags | Tags | tags | map[string]string |
| (internal mutex) | - | - | Not serialized |

### Span Type Field Mapping

| `tracing.Span` Field | Proto Span Field | Unit | Notes |
|---------------------|-----------------|------|-------|
| SpanID | span_id | - | 16 hex chars |
| ParentID | parent_id | - | Empty for root |
| Name | name | - | String |
| SpanType | span_type | - | AGENT, LLM, TOOL, etc. |
| StartTimeNs | start_time_ns | nanoseconds | int64 |
| EndTimeNs | end_time_ns | nanoseconds | int64 |
| Attributes | attributes | - | JSON-serialized map |
| Status | status | - | SpanStatus struct |
| Events | events | - | Array of SpanEvent |

### Error Type Hierarchy

All packages use a consistent error handling strategy:

| Package | Error Types | When Used |
|---------|------------|-----------|
| mlflow | APIError, ValidationError, TimeoutError, ConnectionError | HTTP/network operations |
| tracing | (wraps mlflow errors) | Flush failures |
| eval | EvalError | Evaluation-level errors |

**Error Wrapping Pattern**:
- All errors implement `error` interface
- All errors with `Cause` implement `Unwrap()` for `errors.Is/As`
- Higher-level packages wrap lower-level errors (eval wraps tracing wraps mlflow)

### Timestamp Units

| Context | Unit | Type | Example |
|---------|------|------|---------|
| Span.StartTimeNs | nanoseconds | int64 | `time.Now().UnixNano()` |
| Span.EndTimeNs | nanoseconds | int64 | `time.Now().UnixNano()` |
| Trace.RequestTime | milliseconds | int64 | `time.Now().UnixMilli()` |
| SpanEvent.Timestamp | nanoseconds | int64 | `time.Now().UnixNano()` |
| API response times | milliseconds | int64 | From server |

**Conversion**: `milliseconds = nanoseconds / 1_000_000`

### Token Usage Extraction

Token usage is extracted from Fantasy's step response:

```go
// Token usage is provider-dependent. Common structure:
type Usage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
    TotalTokens  int `json:"total_tokens,omitempty"`
}

// Extraction in OnStepFinish:
if stepResult.Response != nil && stepResult.Response.Usage != nil {
    usage := stepResult.Response.Usage
    span.SetAttribute("mlflow.tokenUsage", map[string]int{
        "input_tokens":  usage.InputTokens,
        "output_tokens": usage.OutputTokens,
        "total_tokens":  usage.TotalTokens,
    })
}
```

If `Usage` is not available (provider doesn't return it), the attribute is omitted (not set to zero).

### Assessment Source Types

| Source Type | When Used | Example SourceID |
|-------------|-----------|------------------|
| CODE | Heuristic scorers (ExactMatch, JSONMatch) | "ExactMatch" |
| LLM_JUDGE | LLM-as-judge scorers | "Correctness" |
| HUMAN | Not generated by this library | (for UI/manual feedback) |

The eval framework only generates CODE and LLM_JUDGE assessments. HUMAN is reserved for
assessments created through MLflow UI or other tools.

### Span Types Usage

| Span Type | When Created | Parent |
|-----------|-------------|--------|
| AGENT | OnAgentStart | None (root) |
| CHAIN | OnStepStart | AGENT |
| LLM | OnStepFinish (inferred) | CHAIN |
| TOOL | OnToolCall | CHAIN |
| RETRIEVER | Reserved for future RAG support | - |
| EMBEDDING | Reserved for future embedding support | - |
| UNKNOWN | Fallback for unrecognized types | - |

RETRIEVER and EMBEDDING are defined for forward compatibility but not currently
created by the Fantasy integration.

### Terminology Consistency

These terms are used consistently across all documents:

| Term | Meaning | NOT |
|------|---------|-----|
| Trace | Complete execution record | (not "request") |
| Span | Single operation within trace | (not "event") |
| State | Trace-level: IN_PROGRESS, OK, ERROR | Status |
| Status | Span-level: UNSET, OK, ERROR | State |
| Score | eval package result | Assessment |
| Assessment | mlflow API entity | Score |
| Flush | Send trace to MLflow | Export |
| Export | Send eval results to MLflow | Flush |
