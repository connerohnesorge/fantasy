# Implementation Tasks

## 1. Proto Infrastructure Setup

- [ ] 1.1 Create `proto/` directory structure with `buf.yaml` and `buf.gen.yaml`
- [ ] 1.2 Add Taskfile tasks: `proto:deps`, `proto:gen`, `proto:lint`
- [ ] 1.3 Create `proto/scalapb/scalapb.proto` stub for extension compatibility
- [ ] 1.4 Download OpenTelemetry protos to `proto/opentelemetry/proto/`
- [ ] 1.5 Download Google well-known types (timestamp, duration, field_mask)
- [ ] 1.6 Add `gen/` to `.gitignore`

## 2. MLflow Proto Curation

- [ ] 2.0 Clone MLflow repository (v3.8.0): `git clone --depth 1 --tag v3.8.0 https://github.com/mlflow/mlflow.git mlflow-ref/mlflow`
- [ ] 2.1 Copy core protos from `mlflow-ref/mlflow/protos/` to `proto/mlflow/`
- [ ] 2.2 Strip ScalaPB extensions and Databricks-specific options
- [ ] 2.3 Create `service.proto` with trace messages (TraceInfoV3, Span, etc.)
- [ ] 2.4 Create `assessments.proto` with Assessment, Feedback, Expectation
- [ ] 2.5 Create `experiments.proto` with Experiment, Run, Metric, Param
- [ ] 2.6 Create `scorers.proto` with Scorer registration messages
- [ ] 2.7 Verify `buf lint` passes on curated protos
- [ ] 2.8 Generate Go code with `task proto:gen` and verify compilation

## 3. MLflow REST Client

- [ ] 3.1 Create `mlflowclient/` package scaffold
- [ ] 3.2 Implement `client.go` with base HTTP client and protojson marshaling
- [ ] 3.3 Implement `errors.go` with APIError type and error code handling
- [ ] 3.4 Implement `options.go` with functional options pattern
- [ ] 3.5 Implement `experiments.go`: Create, Get, Search, Update, Delete
- [ ] 3.6 Implement `runs.go`: Create, Get, Update, LogBatch, Search
- [ ] 3.7 Implement `traces.go`: StartTrace, GetTrace, SearchTraces, DeleteTraces
- [ ] 3.8 Implement `assessments.go`: Create, Get, Update, Delete
- [ ] 3.9 Implement `scorers.go`: Register, List, Get, Delete
- [ ] 3.10 Implement `tags.go`: SetTraceTag, DeleteTraceTag, SetRunTag
- [ ] 3.11 Add unit tests with mock HTTP server
- [ ] 3.12 Add integration test against local MLflow server (optional, manual)

## 4. Evaluation Framework Core

- [ ] 4.1 Create `eval/` package scaffold
- [ ] 4.2 Define `Scorer` interface in `scorer.go`
- [ ] 4.3 Define `Score` result type with value, rationale, metadata
- [ ] 4.4 Define `Dataset` and `TestCase` types in `dataset.go`
- [ ] 4.5 Implement JSON dataset loader in `dataset.go`
- [ ] 4.6 Implement `Evaluator` runner in `evaluator.go`
- [ ] 4.7 Add sequential execution mode (default)
- [ ] 4.8 Add parallel execution with configurable worker count
- [ ] 4.9 Add timeout handling per test case
- [ ] 4.10 Implement result aggregation and summary statistics

## 5. Built-in Heuristic Scorers

- [ ] 5.1 Implement heuristic scorers in `eval/scorer.go`
- [ ] 5.2 Implement `ExactMatch` scorer (string equality)
- [ ] 5.3 Implement `Contains` scorer (substring match)
- [ ] 5.4 Implement `Regex` scorer (pattern matching)
- [ ] 5.5 Implement `JSONMatch` scorer (structural comparison)
- [ ] 5.6 Implement `NumericRange` scorer (value in range)
- [ ] 5.7 Implement `ToolCallTrajectory` scorer (tool sequence validation)
- [ ] 5.8 Implement `StepValidation` scorer (step count and content validation)
- [ ] 5.9 Add unit tests for all heuristic scorers

## 6. LLM-as-Judge Scorers

- [ ] 6.1 Create `eval/llm_judge.go` with base LLM judge implementation
- [ ] 6.2 Define `JudgeConfig` with model, prompt template, output schema
- [ ] 6.3 Implement `Correctness` scorer (answer accuracy)
- [ ] 6.4 Implement `Guidelines` scorer (custom criteria)
- [ ] 6.5 Implement `Relevance` scorer
- [ ] 6.6 Implement `Groundedness` scorer
- [ ] 6.7 Add structured output parsing for judge responses
- [ ] 6.8 Add retry logic for LLM API failures (3 retries, 1s initial, 2x backoff, 10s max)
- [ ] 6.9 Add unit tests with mocked LLM responses

## 7. MLflow Export for Evaluation

- [ ] 7.1 Create `eval/mlflow_export.go` with converter functions
- [ ] 7.2 Implement `ScoreToAssessment()` converter
- [ ] 7.3 Implement `DatasetToRun()` converter for logging
- [ ] 7.4 Implement `EvaluationResultsToMetrics()` converter
- [ ] 7.5 Add `WithMLflowExport(client)` evaluator option
- [ ] 7.6 Implement automatic assessment creation on evaluation completion
- [ ] 7.7 Add unit tests for converter functions

## 8. Tracing Integration

- [ ] 8.1 Create `tracing/` package scaffold
- [ ] 8.2 Implement `Tracer` type with span creation methods
- [ ] 8.3 Implement span hierarchy: Agent -> Step -> (LLM inferred) -> Tool
- [ ] 8.4 Define span attribute keys (mlflow.spanInputs, mlflow.spanOutputs, etc.)
- [ ] 8.5 Implement `TracingConfig` struct with Client, ExperimentID, AgentName, ModelName, SessionID, Tags
- [ ] 8.6 Implement tracing wrapper that intercepts Run/Stream calls
- [ ] 8.7 Create `fantasy.WithTracing(config)` agent option
- [ ] 8.8 Hook OnStepStart/OnStepFinish for step spans and inferred LLM spans
- [ ] 8.9 Hook OnToolCall/OnToolResult for tool spans
- [ ] 8.10 Implement automatic trace flush on wrapper completion
- [ ] 8.11 Add unit tests for span generation
- [ ] 8.12 Add integration test with real agent execution

## 9. Documentation

- [ ] 9.1 Add README.md to `mlflowclient/` with usage examples
- [ ] 9.2 Add README.md to `eval/` with scorer examples
- [ ] 9.3 Add README.md to `tracing/` with setup instructions
- [ ] 9.4 Update main Fantasy README with MLflow integration section
- [ ] 9.5 Add example in `examples/mlflow/` demonstrating full workflow

## 10. Testing and Validation

- [ ] 10.1 Run `task lint` and fix any issues
- [ ] 10.2 Run `task test` and ensure all tests pass
- [ ] 10.3 Add VCR cassettes for MLflow API tests (if applicable)
- [ ] 10.4 Verify proto generation is reproducible
- [ ] 10.5 Test against local MLflow server (manual verification)

## Dependencies

- Tasks 2.x depend on 1.x (proto infrastructure)
- Tasks 3.x depend on 2.x (client needs generated types)
- Tasks 4.x-6.x can be parallelized (eval framework is independent)
- Tasks 7.x depend on 3.x and 4.x (export needs client and eval types)
- Tasks 8.x depend on 2.x (tracing needs proto types for Trace/Span) and 3.x (tracing needs client)
- Tasks 9.x and 10.x are final validation

## Parallelizable Work

After proto infrastructure (1.x, 2.x), the following can be parallelized:
- **Stream A**: MLflow REST client (3.x)
- **Stream B**: Evaluation framework core (4.x, 5.x, 6.x)

Merge points:
- 7.x requires both A and B
- 8.x requires A
