# Change: Add MLflow Integration with Proto-based Client and Evaluation Framework

## Why

Popular agentic libraries (including LangChain, DSPy, and CrewAI) support pushing session contents to MLflow for debugging and evaluation. Fantasy users need:

1. **Observability**: Visibility into agent thinking processes, tool invocations, and LLM interactions
2. **Systematic Evaluation**: Run evaluations with datasets and scorers to measure agent quality
3. **Production Monitoring**: Track agent behavior, detect regressions, collect feedback

MLflow's GenAI platform (v3.8.0 and later) provides distributed tracing, evaluation scorers, and assessment APIs that enable all three use cases. This integration targets v3.8.0 as the minimum supported version; newer MLflow releases are expected to be compatible.

## What Changes

### 1. Proto-based Go Client Generation (`proto/`)

- **NEW**: Clone MLflow repository (v3.8.0) to `mlflow-ref/` for proto source files
  ```bash
  git clone --depth 1 --branch v3.8.0 https://github.com/mlflow/mlflow.git mlflow-ref/mlflow
  ```
- **NEW**: Copy and curate MLflow protos from `mlflow-ref/mlflow/protos/`
- **NEW**: Strip ScalaPB extensions for pure Go struct generation
- **NEW**: Use Buf toolchain for proto compilation
- **NEW**: Include OpenTelemetry protos for span types
- **NEW**: Generate Go code to `proto/gen/mlflow/` package
- **NEW**: Add Taskfile tasks for proto generation workflow

### 2. MLflow REST Client (`mlflowclient/`)

- **NEW**: Thin REST client using protojson marshaling with Bearer token authentication
- **NEW**: API version strategy (version selection is automatic based on the operation):
  - `/api/2.0/mlflow/` - Stable endpoints for experiments, runs, and metrics (available since MLflow 1.x)
  - `/api/3.0/mlflow/` - GenAI endpoints for traces, assessments, and scorers (introduced in MLflow v3.8.0)
  - Note: There is no `/api/1.0/` in use; MLflow uses v2.0 for tracking APIs and v3.0 for GenAI features
- **NEW**: Trace management (StartTrace, GetTrace, SearchTraces, DeleteTraces, SetTraceTag, DeleteTraceTag)
- **NEW**: Assessment management (CreateAssessment, UpdateAssessment, DeleteAssessment) - Assessments are MLflow's way of attaching evaluation feedback (scores, rationales) to traces
- **NEW**: Experiment management (CreateExperiment, GetExperiment, SearchExperiments, UpdateExperiment, DeleteExperiment)
- **NEW**: Run management (CreateRun, GetRun, UpdateRun, DeleteRun, LogBatch, SearchRuns, SetRunTag, DeleteRunTag)
- **NEW**: Scorer registration API (RegisterScorer, ListScorers, GetScorer)
- **NEW**: Configurable timeout (default 30s) and automatic retry with:
  - Maximum retries: 3
  - Initial delay: 1 second
  - Backoff multiplier: 2x (exponential)
  - Maximum delay: 10 seconds
  - Jitter: ±10% randomization to prevent thundering herd
  - Retryable errors: 5xx status codes, 429 rate limit, connection errors

### 3. Tracing Integration

- **NEW**: Full span hierarchy: Agent → Step → [LLM | Tool]
  - LLM and Tool spans are siblings, both children of the Step span (not nested within each other)
  - A Step may have one LLM span and zero or more Tool spans as children
  - LLM spans are inferred retroactively from step timing (Fantasy doesn't expose direct LLM hooks)
- **NEW**: SpanTypes: AGENT, LLM, TOOL, CHAIN, RETRIEVER, EMBEDDING, UNKNOWN
- **NEW**: Full message content in span attributes (`mlflow.spanInputs`, `mlflow.spanOutputs`) with truncation at 10,240 bytes (10 KB, MLflow's documented attribute size limit; truncation applies after JSON serialization)
- **NEW**: Integration with Fantasy's existing callbacks via a tracing wrapper—an internal callback handler that intercepts Run/Stream calls to create/manage spans (hooks OnStepStart, OnStepFinish, OnToolCall, OnToolResult)
- **NEW**: `fantasy.WithTracing(config)` agent option to enable MLflow tracing
- **NEW**: TracingConfig structure with:
  - `Client` - MLflow client for trace export
  - `ExperimentID` - Target experiment
  - `AgentName` - Optional agent name (default: "agent")
  - `ModelName` - Optional model name
  - `SessionID` - Optional session ID (auto-generated UUID v4 if empty)
  - `Tags` - Optional custom tags map
  - `FlushTimeout` - Optional flush timeout (default: 10s)

### 4. Evaluation Framework (`eval/`)

- **NEW**: Evaluation framework with Go-native types (works entirely locally without MLflow server; export to MLflow is optional and only requires a server connection when explicitly enabled via `WithMLflowExport`)
- **NEW**: Go function scorers with `Scorer` interface
- **NEW**: LLM-as-judge scorers using Fantasy's own providers
- **NEW**: In-memory datasets with optional file-based loading (JSON only)
- **NEW**: Configurable parallelism (default sequential, opt-in parallel via WithParallelism)
- **NEW**: Converters to/from proto types for optional MLflow export
- **NEW**: Heuristic scorers: ExactMatch, Contains, Regex, JSONMatch, NumericRange
- **NEW**: LLM-as-judge scorers: Correctness, Guidelines, Relevance, Groundedness
- **NEW**: Agent-specific scorers: ToolCallTrajectory, StepValidation (require `*tracing.Trace` type for trace data, but do not require an MLflow server connection—the Trace is passed in-memory from the predict function)

## Impact

- **Affected specs**: NEW capabilities (no modifications to existing specs)
  - `mlflow-client`: Proto generation and REST client
  - `tracing`: MLflow tracing integration
  - `evaluation-framework`: Decoupled evaluation with scorers

- **Affected code**:
  - New `proto/` directory with Buf configuration
  - New `proto/gen/mlflow/` generated package
  - New `mlflowclient/` package
  - New `eval/` package
  - Agent callbacks integration in `agent.go`

- **Dependencies**:
  - `google.golang.org/protobuf` for protobuf runtime
  - Buf toolchain for code generation (dev dependency)

- **Breaking changes**: None for existing Fantasy API - purely additive new packages. Note: New dependencies (`google.golang.org/protobuf`) will increase binary size and build time for users who import the new packages.

## Scope Boundaries

### In Scope
- OSS MLflow v3.8+ REST API support
- Trace API v3 (`/api/3.0/mlflow/traces`)
- Assessment API for scorer feedback
- Experiment/Run API for organization
- All scorer types (heuristic + LLM-as-judge)

### Out of Scope (Explicitly Excluded)
- Databricks-specific APIs (V4, UC Schema locations)
- MLflow UI compatibility requirements
- gRPC transport (REST only via protojson)
- Model registry APIs
- Gateway/endpoint management APIs

## Terminology

| Term | Definition |
|------|------------|
| **MLflow** | Open-source platform for managing ML lifecycles, including experiment tracking, model registry, and GenAI tracing |
| **Trace** | A complete record of an agent execution, containing spans for each operation |
| **Span** | A single unit of work within a trace (e.g., an LLM call, tool execution) |
| **Assessment** | Feedback or expectations attached to a trace (maps from scorer outputs) |
| **Scorer** | A function that evaluates outputs against expectations, returning a score and rationale |
| **Protobuf** | Protocol Buffers - Google's data serialization format used by MLflow for API definitions |
| **Buf** | Modern toolchain for Protocol Buffers that handles linting, breaking change detection, and code generation |
| **Bearer Token** | An HTTP authentication scheme where a token is sent in the `Authorization: Bearer <token>` header |
| **OpenTelemetry** | Vendor-neutral observability framework; MLflow uses OTel conventions for span types |

## Error Handling Strategy

- **Network errors**: Automatic retry with exponential backoff (configured per-client)
- **4xx errors**: Not retried (except 429 rate limit); returned immediately as `APIError`
- **5xx errors**: Retried up to max retries; final error returned as `APIError`
- **Timeout errors**: Returned as `TimeoutError` with operation context
- **Validation errors**: Returned immediately as `ValidationError` (no retry)
- **Trace flush errors**: Logged but non-fatal; available via `TracingResult.FlushError`

## Security Considerations

- **Token handling**: Tokens are passed via `WithToken()` option and stored only in client instance (not logged)
- **PII in traces**: Span inputs/outputs may contain user data; truncation limits exposure but users should be aware
- **Network security**: All communication uses HTTPS (HTTP allowed only for local development)
- **No credential storage**: Library does not persist credentials; users manage token lifecycle

## Testing Strategy

- **Unit tests**: Mock HTTP responses using `httptest` package; no external dependencies
- **Integration tests**: Against local MLflow server via Docker Compose (opt-in, not run in CI by default)
- **Evaluation tests**: Dataset fixtures with known-good outputs for scorer validation
