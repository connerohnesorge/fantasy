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

**Pattern**:
```go
func (c *Client) doRequest(ctx context.Context, method, path string,
    req, resp proto.Message) error {
    data, _ := protojson.Marshal(req)
    // ... HTTP request ...
    return protojson.Unmarshal(respData, resp)
}
```

### Decision 3: Decoupled Evaluation Framework

**What**: Evaluation framework uses its own Go types with converters to proto.

**Why**:
- Evaluation works standalone without MLflow server
- Go-native types are more ergonomic (`Scorer` interface, `Dataset` struct)
- Proto types are for serialization, not business logic
- Easier testing with Go types

**Architecture**:
```
eval/
├── scorer.go          # Scorer interface + built-in scorers
├── dataset.go         # Dataset and test case types
├── evaluator.go       # Evaluation runner
├── llm_judge.go       # LLM-as-judge scorers
└── mlflow_export.go   # Converters to proto/Assessment
```

### Decision 4: Callback-based Tracing Integration

**What**: Integrate tracing via Fantasy's existing callback system.

**Why**:
- Non-invasive integration
- Users opt-in with `WithTracing(config)`
- Callbacks already exist for step/tool events
- No changes to core agent logic

**Pattern**:
```go
agent := fantasy.NewAgent(model,
    fantasy.WithTracing(tracing.Config{
        Client:       mlflowClient,
        ExperimentID: "my-experiment",
    }),
)
```

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
│   │   └── experiments.proto       # Experiment/run messages
│   ├── opentelemetry/proto/        # OTel protos (downloaded)
│   │   └── trace/v1/trace.proto
│   └── scalapb/                    # Stub for ScalaPB options
│       └── scalapb.proto
│
├── gen/                            # Generated code (git-ignored)
│   └── mlflow/
│       ├── service.pb.go
│       ├── assessments.pb.go
│       └── experiments.pb.go
│
├── mlflowclient/                   # REST client package
│   ├── client.go                   # HTTP client with protojson
│   ├── traces.go                   # Trace API methods
│   ├── experiments.go              # Experiment API methods
│   ├── assessments.go              # Assessment API methods
│   ├── scorers.go                  # Scorer registration API
│   └── errors.go                   # API error types
│
├── eval/                           # Evaluation framework
│   ├── scorer.go                   # Scorer interface + built-in scorers
│   ├── dataset.go                  # Dataset types
│   ├── evaluator.go                # Evaluation runner
│   ├── llm_judge.go                # LLM-as-judge scorers
│   └── mlflow_export.go            # Proto converters
│
└── tracing/                        # Tracing integration
    ├── tracer.go                   # Trace/span builder
    ├── callbacks.go                # Agent callback integration
    └── options.go                  # WithTracing() option
```
