# Change: Add MLflow Integration with Proto-based Client and Evaluation Framework

## Why

Many agentic libraries (LangChain, DSPy, CrewAI) support pushing session contents to MLflow for debugging and evaluation. Fantasy users need:

1. **Observability**: Visibility into agent thinking processes, tool invocations, and LLM interactions
2. **Systematic Evaluation**: Run evaluations with datasets and scorers to measure agent quality
3. **Production Monitoring**: Track agent behavior, detect regressions, collect feedback

MLflow's GenAI platform (v3.8+) provides distributed tracing, evaluation scorers, and assessment APIs that enable all three use cases.

## What Changes

### 1. Proto-based Go Client Generation (`proto/`, `gen/`)

- **NEW**: Clone MLflow repository (v3.8.0) to `mlflow-ref/` for proto source files
  ```bash
  git clone --depth 1 --tag v3.8.0 https://github.com/mlflow/mlflow.git mlflow-ref/mlflow
  ```
- **NEW**: Copy and curate MLflow protos from `mlflow-ref/mlflow/protos/`
- **NEW**: Strip ScalaPB extensions for pure Go struct generation
- **NEW**: Use Buf toolchain for proto compilation
- **NEW**: Include OpenTelemetry protos for span types
- **NEW**: Generate Go code to `gen/mlflow/` package
- **NEW**: Add Taskfile tasks for proto generation workflow

### 2. MLflow REST Client (`mlflowclient/`)

- **NEW**: Thin REST client using protojson marshaling with Bearer token authentication
- **NEW**: Dual API version strategy:
  - `/api/2.0/mlflow/` - Legacy endpoints for experiments, runs, and metrics (stable)
  - `/api/3.0/mlflow/` - Modern endpoints for traces, assessments, and scorers (v3.8+)
- **NEW**: Trace management (StartTrace, GetTrace, SearchTraces, DeleteTraces, SetTraceTag, DeleteTraceTag)
- **NEW**: Assessment management (CreateAssessment, UpdateAssessment, DeleteAssessment) - Assessments are MLflow's way of attaching evaluation feedback (scores, rationales) to traces
- **NEW**: Experiment management (CreateExperiment, GetExperiment, SearchExperiments)
- **NEW**: Run management (CreateRun, GetRun, UpdateRun, LogBatch, SearchRuns, SetRunTag)
- **NEW**: Scorer registration API (RegisterScorer, ListScorers, GetScorer)
- **NEW**: Configurable timeout (default 30s) and automatic retry (3 retries with exponential backoff)

### 3. Tracing Integration

- **NEW**: Full span hierarchy: Agent -> Step -> (LLM, Tool)
  - LLM and Tool spans are siblings (both children of Step span)
  - LLM spans are inferred retroactively from step timing (Fantasy doesn't expose direct LLM hooks)
- **NEW**: SpanTypes: AGENT, LLM, TOOL, CHAIN, RETRIEVER, etc.
- **NEW**: Full message content in span attributes (`mlflow.spanInputs`, `mlflow.spanOutputs`) with truncation at 10,240 bytes
- **NEW**: Integration with Fantasy's existing callbacks via tracing wrapper (hooks OnStepStart, OnStepFinish, OnToolCall, OnToolResult)
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

- **NEW**: Evaluation framework with Go-native types (works without MLflow server, optional export)
- **NEW**: Go function scorers with `Scorer` interface
- **NEW**: LLM-as-judge scorers using Fantasy's own providers
- **NEW**: In-memory datasets with optional file-based loading (JSON only)
- **NEW**: Configurable parallelism (default sequential, opt-in parallel via WithParallelism)
- **NEW**: Converters to/from proto types for optional MLflow export
- **NEW**: Heuristic scorers: ExactMatch, Contains, Regex, JSONMatch, NumericRange
- **NEW**: LLM-as-judge scorers: Correctness, Guidelines, Relevance, Groundedness
- **NEW**: Agent-specific scorers: ToolCallTrajectory, StepValidation (require *tracing.Trace)

## Impact

- **Affected specs**: NEW capabilities (no modifications to existing specs)
  - `mlflow-client`: Proto generation and REST client
  - `tracing`: MLflow tracing integration
  - `evaluation-framework`: Decoupled evaluation with scorers

- **Affected code**:
  - New `proto/` directory with Buf configuration
  - New `gen/mlflow/` generated package
  - New `mlflowclient/` package
  - New `eval/` package
  - Agent callbacks integration in `agent.go`

- **Dependencies**:
  - `google.golang.org/protobuf` for protobuf runtime
  - Buf toolchain for code generation (dev dependency)

- **Breaking changes**: None - purely additive

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
