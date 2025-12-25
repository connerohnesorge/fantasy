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

- **NEW**: Thin REST client using protojson marshaling
- **NEW**: Trace management (create, get, search, delete, tags)
- **NEW**: Assessment management (create, update, delete) - Assessments are MLflow's way of attaching evaluation feedback (scores, rationales) to traces
- **NEW**: Experiment/Run management
- **NEW**: Scorer registration API

### 3. Tracing Integration

- **NEW**: Full span hierarchy: Agent -> Step -> LLM call -> Tool execution
- **NEW**: SpanTypes: AGENT, LLM, TOOL, CHAIN, RETRIEVER, etc.
- **NEW**: Full message content in span attributes (`mlflow.spanInputs`, `mlflow.spanOutputs`)
- **NEW**: Integration with Fantasy's existing callbacks via tracing wrapper (captures context from Run/Stream calls, hooks OnStepStart, OnStepFinish, OnToolCall, OnToolResult)
- **NEW**: `WithTracing(config)` agent option to enable MLflow tracing

### 4. Evaluation Framework (`eval/`)

- **NEW**: Decoupled evaluation framework with own Go types
- **NEW**: Go function scorers with `Scorer` interface
- **NEW**: LLM-as-judge scorers using Fantasy's own providers
- **NEW**: In-memory datasets with optional file-based loading (JSON only)
- **NEW**: Configurable parallelism (default sequential, opt-in parallel)
- **NEW**: Converters to/from proto types for MLflow export
- **NEW**: Heuristic scorers: ExactMatch, Contains, Regex, JSONMatch, NumericRange
- **NEW**: LLM-as-judge scorers: Correctness, Guidelines, Relevance, Groundedness
- **NEW**: Agent-specific scorers: ToolCallTrajectory, StepValidation

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
