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
| **Assessment** | mlflowclient | MLflow API entity for trace annotations (maps from Score) |
| **Trace ID** | tracing | Full trace identifier with `tr-` prefix (e.g., `tr-abc123...`) |
| **Request ID** | tracing | Deprecated alias for Trace ID (used in MLflow v2 API) |

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
- Non-invasive integration
- Users opt-in with `WithTracing(config)`
- Callbacks already exist for step/tool events
- No changes to core agent logic

**Span Hierarchy**: `Agent -> Step -> (LLM, Tool)`

LLM and Tool spans are siblings—both are children of the Step span. When a step executes,
it may involve an LLM call followed by zero or more tool calls, all at the same hierarchical level.

**Note on LLM Span Inference**: Fantasy's callback system provides OnStepStart/OnStepFinish and
OnToolCall/OnToolResult, but does not expose direct LLM call hooks. LLM spans are therefore
**inferred retroactively** from step timing: the LLM span covers the time between step start
and the first tool call (or step end if no tools). This is an approximation but provides
useful visibility into LLM execution time. Token usage is captured from the step's response
metadata when available.

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

Note: `fantasy.WithTracing()` is in the fantasy package for consistency with other agent options
(e.g., `fantasy.WithTools()`, `fantasy.WithSystemPrompt()`). The TracingConfig struct is defined in
the tracing/ package but the WithTracing() function is exported from fantasy/ to maintain API ergonomics.
The tracing wrapper intercepts Run/Stream calls to capture context before delegating to the agent.

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
     git clone --depth 1 --tag v3.8.0 https://github.com/mlflow/mlflow.git mlflow-ref/mlflow
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

## Migration Plan

No migration needed - this is purely additive functionality.

### Adoption Path
1. Users add `mlflowclient` dependency
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
├── mlflowclient/                   # REST client package
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
│   └── config.go                   # TracingConfig struct
│
└── agent.go                        # WithTracing() option (fantasy package)
```
