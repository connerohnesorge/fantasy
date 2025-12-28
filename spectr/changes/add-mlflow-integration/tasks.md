# Implementation Tasks

## Priority & Context

**Implementation Strategy**: Foundation-first approach
- **Phase 1** (Critical Path): Proto Infrastructure (1.x) → MLflow Proto Curation (2.x) → MLflow REST Client (3.x)
- **Phase 2** (Parallel): Evaluation Framework (4.x, 5.x, 6.x) + Tracing Integration (8.x)
- **Phase 3** (Integration): MLflow Export (7.x), Documentation (9.x), Validation (10.x)

**Documentation Approach**: Incremental - write README and examples alongside each component for better testability

**Evaluation Coverage**: Comprehensive - all scorer types (heuristic, LLM-as-judge, trajectory) are equally important

## 0. MLflow Server Setup (Prerequisite)

- [ ] 0.1 Create `docker-compose.yml` for local MLflow server with PostgreSQL backend
  - Context: Needed for testing during development. Run with `docker compose up -d`
  - Acceptance: MLflow UI accessible at http://localhost:5000
  - Acceptance: Server persists experiments/runs across restarts
- [ ] 0.2 Add Taskfile task `mlflow:start` to start Docker Compose services
- [ ] 0.3 Add Taskfile task `mlflow:stop` to stop services
- [ ] 0.4 Add Taskfile task `mlflow:logs` to view server logs
- [ ] 0.5 Document MLflow setup in root README.md (port 5000, Docker requirement)
  - Context: Quick start guide for local testing

## 0a. Fantasy API Changes (Prerequisite for Tracing)

**Context**: Add OnLLMStart/OnLLMFinish callbacks to Fantasy for accurate LLM span timing in MLflow traces.

- [x] 0a.1 Add OnLLMStartFunc and OnLLMFinishFunc callback types to agent.go
  - Context: New callbacks for accurate LLM span timing
  - Acceptance: Callback types defined with correct signatures (model, call, usage, duration, err)
  - Status: COMPLETED
- [x] 0a.2 Add OnLLMStart and OnLLMFinish fields to AgentStreamCall struct
  - Context: Register callbacks for streaming mode
  - Acceptance: Fields added after OnError in agent-level callbacks section
  - Status: COMPLETED
- [x] 0a.3 Add OnLLMStart and OnLLMFinish fields to AgentCall struct
  - Context: Register callbacks for Generate mode
  - Acceptance: Fields added after RepairToolCall
  - Status: COMPLETED
- [x] 0a.4 Invoke OnLLMStart before LLM call in agent.Stream()
  - Context: Fire callback before stepModel.Stream() call
  - Acceptance: Callback receives stepModel and streamCall parameters
  - Acceptance: Error handling doesn't break agent execution
  - Status: COMPLETED
- [x] 0a.5 Update processStepStream signature to receive llmStartTime, llmFinishCalled, stepModel
  - Context: Need timing data and state tracking for OnLLMFinish
  - Acceptance: Signature updated, all call sites updated
  - Status: COMPLETED
- [x] 0a.6 Invoke OnLLMFinish when StreamPartTypeFinish received in processStepStream
  - Context: Fire callback when LLM completes with accurate usage
  - Acceptance: Callback receives usage, finishReason, duration from llmStartTime
  - Acceptance: Duration calculated accurately
  - Status: COMPLETED
- [x] 0a.7 Add deferred OnLLMFinish call in agent.Stream() for error cases
  - Context: Ensure callback fires even if stream fails early
  - Acceptance: Deferred call only fires if OnLLMFinish not already called
  - Status: COMPLETED
- [x] 0a.8 Invoke OnLLMStart/OnLLMFinish in agent.Generate() method
  - Context: Non-streaming mode needs same callbacks
  - Acceptance: Callbacks fire before/after stepModel.Generate()
  - Acceptance: Duration calculated accurately
  - Status: COMPLETED
- [ ] 0a.9 Add unit tests for OnLLMStart/OnLLMFinish callback invocation
  - Context: Verify callbacks fire at correct times
  - Acceptance: Test streaming mode
  - Acceptance: Test Generate mode
  - Acceptance: Test with nil callbacks (no crash)
  - Acceptance: Test callback errors (logged, not propagated)
- [ ] 0a.10 Add integration test for LLM timing accuracy
  - Context: Verify accurate span timing end-to-end
  - Acceptance: Compare OnLLMStart/Finish timestamps to actual LLM call duration
  - Acceptance: Verify duration includes retry attempts

## 1. Proto Infrastructure Setup

**Context**: Foundation for type-safe MLflow communication. Buf manages proto compilation and linting.

- [ ] 1.1 Create `proto/` directory structure with `buf.yaml` and `buf.gen.yaml`
  - Context: `buf.yaml` defines linting rules, `buf.gen.yaml` configures Go code generation
  - Acceptance: `buf lint` runs successfully (even with no protos yet)
- [ ] 1.2 Add Taskfile tasks: `proto:deps`, `proto:gen`, `proto:lint`
  - Context: `proto:deps` installs buf, `proto:gen` generates Go code, `proto:lint` validates protos
- [ ] 1.3 Create `proto/scalapb/scalapb.proto` stub for extension compatibility
  - Context: MLflow protos reference ScalaPB extensions; stub prevents compilation errors
- [ ] 1.4 Download OpenTelemetry protos to `proto/opentelemetry/proto/`
  - Context: MLflow traces use OTel span format; needed for TraceInfoV3 messages
- [ ] 1.5 Download Google well-known types to `proto/google/protobuf/` (timestamp.proto, duration.proto, struct.proto, any.proto, field_mask.proto)
  - Context: MLflow messages use these standard types extensively
- [ ] 1.6 Add `proto/gen/` to `.gitignore`
  - Context: Generated code shouldn't be committed; rebuilt via `task proto:gen`
- [ ] 1.7 Add Taskfile task `proto:sync` to update protos from mlflow-ref (documented in design)
  - Context: Enables easy proto updates when MLflow releases new versions
- [ ] 1.8 Write `proto/README.md` with proto generation workflow and troubleshooting
  - Context: Document buf setup, how to add new protos, common errors

## 2. MLflow Proto Curation

**Context**: Extract and clean MLflow proto definitions. We use v3.8.0 for stability; traces API is relatively new.

- [ ] 2.0 Clone MLflow repository (v3.8.0): `git clone --depth 1 --branch v3.8.0 https://github.com/mlflow/mlflow.git mlflow-ref/mlflow`
  - Context: Shallow clone saves disk space; v3.8.0 has stable traces API
  - Acceptance: `mlflow-ref/mlflow/protos/` directory exists
- [ ] 2.1 Copy core protos from `mlflow-ref/mlflow/protos/` to `proto/mlflow/`
  - Context: Start with original protos, then modify for our use case
  - Files to copy: `service.proto`, `assessments.proto`, `datasets.proto`, `internal.proto`
  - Skip: Databricks-specific files (`databricks_*.proto`), Unity Catalog files (`unity_catalog_*.proto`)
  - Acceptance: Selected .proto files copied with correct directory structure
- [ ] 2.2 Strip ScalaPB extensions and Databricks-specific options
  - Context: Remove `[(scalapb.field).type = ...]` and Databricks auth options we don't need
  - Method: Use sed/awk or manual edit to remove lines matching `scalapb.` and `databricks_`
  - Pattern to remove: `[(scalapb.*)]`, `option (scalapb.*) = ...`, imports of `scalapb/scalapb.proto`
  - Acceptance: `buf lint` passes without unknown extension errors
  - Acceptance: Proto files compile successfully with `buf build`
- [ ] 2.3 Create `service.proto` with trace messages (TraceInfoV3, Span, etc.)
  - Context: Consolidate trace-related messages; TraceInfoV3 is the current format
- [ ] 2.4 Create `assessments.proto` with Assessment, Feedback, Expectation
  - Context: Evaluation results storage; maps to our scorer output
- [ ] 2.5 Create `experiments.proto` with Experiment, Run, Metric, Param
  - Context: Core MLflow tracking types; needed for logging eval results
- [ ] 2.6 Create `scorers.proto` with Scorer registration messages
  - Context: Custom scorer metadata for MLflow UI display
- [ ] 2.7 Verify `buf lint` passes on curated protos
  - Context: Ensures protos follow best practices before codegen
  - Acceptance: Zero lint errors, zero warnings
- [ ] 2.8 Generate Go code with `task proto:gen` and verify compilation
  - Context: Smoke test that generated code is valid Go
  - Acceptance: `proto/gen/mlflow/*.pb.go` files exist and `go build ./proto/gen/...` succeeds

## 3. MLflow REST Client

**Context**: Type-safe HTTP client for MLflow REST API. Uses protojson for serialization, functional options for configuration.
**Priority**: Critical path - enables testing with real MLflow server early.

- [ ] 3.1 Create `mlflow/` package scaffold
  - Context: Foundation for all MLflow communication
  - Acceptance: Package compiles with `go build ./mlflow`
- [ ] 3.2 Implement `client.go` with base HTTP client and protojson marshaling
  - Context: Central HTTP client with `BaseURL`, auth headers, protojson codec
  - Acceptance: `NewClient(baseURL string, opts...)` creates client
  - Acceptance: `doRequest()` helper marshals proto → JSON → HTTP request
- [ ] 3.3 Implement `errors.go` with APIError type and error code handling
  - Context: Structured error handling for debugging and retry logic
  - Acceptance: APIError, ValidationError, TimeoutError, ConnectionError types defined
  - Acceptance: All error types implement `error` interface and `Unwrap()` for wrapping
  - Acceptance: `IsRetryable()` returns true for 429, 5xx, and connection errors
- [ ] 3.4 Implement `options.go` with functional options pattern
  - Context: Idiomatic Go configuration: `NewClient(url, WithTimeout(30*time.Second), WithRetries(5))`
  - Acceptance: Options for timeout, retries, auth token, custom headers, HTTP client
- [ ] 3.5 Implement `experiments.go`: Create, Get, Search, Update, Delete
  - Context: Experiment management; needed before creating runs/traces
  - Acceptance: All methods accept proto request types, return proto response types
- [ ] 3.6 Implement `runs.go`: Create, Get, GetRun, Update, LogBatch, Search, SearchRuns
  - Context: Run lifecycle and metrics logging for eval results
  - Acceptance: `GetRun` and `SearchRuns` included (required by proposal)
  - Acceptance: `LogBatch` supports metrics, params, tags in single call
- [ ] 3.7 Implement `traces.go`: StartTrace, GetTrace, SearchTraces, DeleteTraces
  - Context: Core tracing API; StartTrace creates trace + root span
  - Acceptance: Methods map to `/api/3.0/mlflow/traces/*` endpoints (v3 API)
- [ ] 3.8 Implement `assessments.go`: Create, Get, Update, Delete
  - Context: Store evaluation results (scorer outputs)
  - Acceptance: Assessment CRUD operations work end-to-end
- [ ] 3.9 Implement `scorers.go`: Register, List, Get, Delete
  - Context: Custom scorer registration for MLflow UI
  - Acceptance: Scorer metadata persists across server restarts
- [ ] 3.10 Implement `tags.go`: SetTraceTag, DeleteTraceTag, SetRunTag, DeleteRunTag
  - Context: Add metadata to traces/runs for filtering and organization
  - Acceptance: Supports string key-value tags
- [ ] 3.10a Define `SearchExperimentsOptions` struct in `experiments.go`
  - Context: Options for SearchExperiments (Filter, MaxResults, PageToken, OrderBy, ViewType)
  - Acceptance: All fields documented with JSON tags and defaults
- [ ] 3.10b Define `SearchRunsOptions` struct in `runs.go`
  - Context: Options for SearchRuns (ExperimentIDs, Filter, RunViewType, MaxResults, OrderBy, PageToken)
  - Acceptance: ExperimentIDs is required; validation returns error if empty
- [ ] 3.10c Define `SearchTracesOptions` struct in `traces.go`
  - Context: Options for SearchTraces (ExperimentIDs, Filter, MaxResults, PageToken, OrderBy)
  - Acceptance: All fields documented; sensible defaults (MaxResults=100)
- [ ] 3.10d Define `DeleteTracesOptions` struct in `traces.go`
  - Context: Options for DeleteTraces (MaxTraces, MaxTimestampMs, Filter)
  - Acceptance: Validation ensures at least one criterion is specified
- [ ] 3.10e Define `SerializedScorer` struct in `scorers.go`
  - Context: Scorer registration format (Type, Name, Description, Config)
  - Acceptance: Config is map[string]any for flexibility
- [ ] 3.10f Define `Assessment` and related types in `assessments.go`
  - Context: Assessment, AssessmentSource, FeedbackValue, ExpectationValue, AssessmentError
  - Acceptance: All types match spec definitions; proper JSON marshaling
- [ ] 3.11 Implement retry logic with exponential backoff and jitter
  - Context: Production-ready resilience for flaky networks and rate limits
  - Acceptance: 3 retries max, 1s initial delay, 2x backoff, +/-10% jitter, 10s max delay
  - Acceptance: Handle 429 rate limiting with Retry-After header
  - Acceptance: `WithRetries(0)` disables retries for testing
- [ ] 3.12 Add unit tests with mock HTTP server (httptest)
  - Context: Fast tests without Docker; verify request/response handling
  - Acceptance: Test error scenarios (4xx, 5xx, timeouts, malformed JSON)
  - Acceptance: Test retry logic with injected failures
- [ ] 3.13 Add integration test against local MLflow server (manual)
  - Context: Verify real MLflow compatibility; run `task mlflow:start` first
  - Acceptance: Test can create experiment, trace, assessment end-to-end
- [ ] 3.14 Write `mlflow/README.md` with usage examples
  - Context: Quick start guide with code snippets for common operations
  - Acceptance: Examples for creating client, experiment, trace, assessment

## 4. Evaluation Framework Core

**Context**: Extensible eval framework for agent/LLM testing. Scorer interface enables heuristic + LLM-based evaluation.
**Design**: Evaluator orchestrates test case execution, scorers implement evaluation logic, results aggregate scores.

- [ ] 4.1 Create `eval/` package scaffold
  - Context: Core evaluation types and interfaces
  - Acceptance: Package structure: `scorer.go`, `dataset.go`, `evaluator.go`, `results.go`
- [ ] 4.2 Define `Scorer` interface in `scorer.go`
  - Context: `Score(ctx context.Context, input ScorerInput) (Score, error)` - universal interface
  - Acceptance: Interface has `Name() string` and `Score()` methods
  - Acceptance: Context enables timeout/cancellation
- [ ] 4.2a Define `ScorerInput` struct in `scorer.go`
  - Context: Input container for scorers (Outputs, Expectations, Trace, Inputs)
  - Acceptance: Trace field is `*tracing.Trace` (nil for scorers that don't need it)
- [ ] 4.3 Define `Score` result type with value, rationale, metadata
  - Context: Score value can be bool, float64, int, or string
  - Acceptance: Fields: `Value any`, `Rationale string`, `Metadata map[string]any`, `Error error`
- [ ] 4.3a Define `PredictFunc` type in `evaluator.go`
  - Context: Function signature for generating outputs from inputs
  - Acceptance: `func(ctx, inputs map[string]any) (outputs map[string]any, trace *tracing.Trace, err error)`
- [ ] 4.3b Define `RunOption` functions in `evaluator.go`
  - Context: Functional options for Run(): WithParallelism, WithTimeout, WithPredict, WithTracing
  - Acceptance: Each option function documented with its effect
- [ ] 4.4 Define `Dataset` and `TestCase` types in `dataset.go`
  - Context: Dataset = []TestCase; TestCase = {Input, Expected, Metadata}
  - Acceptance: Supports arbitrary JSON for input/expected (flexible schema)
- [ ] 4.5 Implement JSON dataset loader in `dataset.go`
  - Context: Load test cases from JSON file: `LoadDataset(path string) (*Dataset, error)`
  - Acceptance: Handles malformed JSON gracefully with descriptive errors
- [ ] 4.6 Implement `Evaluator` runner in `evaluator.go`
  - Context: Orchestrates scorer execution across dataset
  - Acceptance: `NewEvaluator(scorers []Scorer, opts...)` creates evaluator
  - Acceptance: `Run(ctx, dataset) (*Results, error)` processes all test cases
  - Acceptance: Errors in individual test cases don't stop other test cases (collect errors)
- [ ] 4.7 Add sequential execution mode (default)
  - Context: Run scorers one-by-one for predictable behavior and debugging
  - Acceptance: Test cases processed in order
- [ ] 4.8 Add parallel execution with configurable worker count
  - Context: `WithWorkers(n)` option for faster eval on large datasets
  - Acceptance: Worker pool processes test cases concurrently
  - Acceptance: Results maintain original test case order
- [ ] 4.9 Add timeout handling per test case
  - Context: `WithTimeout(duration)` prevents hung scorers from blocking eval
  - Acceptance: Timeout cancels scorer context and records error in results
- [ ] 4.10 Implement result aggregation and summary statistics
  - Context: Aggregate scores across dataset (mean, median, pass rate, etc.)
  - Acceptance: `Results` type has `Summary()` method returning stats
  - Acceptance: Per-scorer and overall statistics available
- [ ] 4.10a Define `Results` struct in `results.go`
  - Context: Complete evaluation output (TestCases, Summary, Errors, StartTime, EndTime, TotalTests)
  - Acceptance: All fields match spec; JSON serialization works correctly
- [ ] 4.10b Define `TestCaseResult` struct in `results.go`
  - Context: Per-test-case results (TestCase, Outputs, Scores, Trace)
  - Acceptance: Scores is `map[string]Score` keyed by scorer name
- [ ] 4.10c Define `ScorerStats` struct in `results.go`
  - Context: Aggregated stats per scorer (PassRate, Mean, StdDev, ErrorRate, ErrorCount, Count)
  - Acceptance: Formulas match spec (sample std dev with n-1)
- [ ] 4.10d Define `EvalError` struct in `results.go`
  - Context: Evaluation-level error with Phase, Message, Cause
  - Acceptance: Phase is one of: "setup", "predict", "score", "export"

## 5. Built-in Heuristic Scorers

**Context**: Fast, deterministic scorers without LLM costs. Critical for regression testing and CI/CD.
**Design**: Each scorer implements `Scorer` interface; factory functions create configured instances.
**Priority**: All scorer types equally important - implement comprehensive suite.

- [ ] 5.1 Create `eval/heuristic_scorers.go` file for heuristic implementations
  - Context: Separate file for clarity; keeps `scorer.go` for interfaces only
- [ ] 5.2 Implement `ExactMatch` scorer (string equality)
  - Context: `NewExactMatch(caseSensitive bool)` - binary pass/fail
  - Acceptance: Returns 1.0 if equal, 0.0 if not; case sensitivity configurable
- [ ] 5.3 Implement `Contains` scorer (substring match)
  - Context: `NewContains(substring, caseSensitive)` - checks if output contains substring
  - Acceptance: Useful for "output must mention X" requirements
- [ ] 5.4 Implement `Regex` scorer (pattern matching)
  - Context: `NewRegex(pattern)` - flexible text validation
  - Acceptance: Compiles regex once at construction; returns 1.0 on match
- [ ] 5.5 Implement `JSONMatch` scorer (structural comparison)
  - Context: Deep equality for JSON outputs; essential for API testing
  - Acceptance: Handles nested objects and arrays correctly
- [ ] 5.5a Define `JSONMatchOptions` struct
  - Context: Configuration for JSONMatch (NullEqualsMissing, IgnoreArrayOrder, FloatTolerance)
  - Acceptance: Default: NullEqualsMissing=false, IgnoreArrayOrder=false, FloatTolerance=0
- [ ] 5.6 Implement `NumericRange` scorer (value in range)
  - Context: `NewNumericRange(min, max)` - for metrics, percentages, counts
  - Acceptance: Extracts number from output, checks bounds, scores 1.0 if in range
- [ ] 5.7 Implement `ToolCallTrajectory` scorer (tool sequence validation)
  - Context: Validates agent used tools in expected order (e.g., "must call search before summarize")
  - **Depends on 8.x**: Requires `*tracing.Trace` type from tracing package
  - Acceptance: Compares actual tool call sequence to expected sequence
  - Acceptance: Expected sequence from `expectations["tool_sequence"]` as []string
- [ ] 5.7a Define `ToolCallTrajectoryOptions` struct
  - Context: Configuration for ToolCallTrajectory (Strict, IgnoreOrder)
  - Acceptance: Strict=false (extra tools allowed), IgnoreOrder=false (order matters)
- [ ] 5.8 Implement `StepValidation` scorer (step count and content validation)
  - Context: Checks agent took expected number of steps, each with valid actions
  - **Depends on 8.x**: Requires `*tracing.Trace` type from tracing package
  - Acceptance: Validates step count range (min/max)
- [ ] 5.8a Define `StepValidationOptions` struct
  - Context: Configuration for StepValidation (MinSteps, MaxSteps, ContentPatterns)
  - Acceptance: ContentPatterns is map[int]string for per-step regex validation
- [ ] 5.9 Add unit tests for all heuristic scorers
  - Context: Fast, deterministic tests; no MLflow dependency
  - Acceptance: Test edge cases: empty strings, malformed JSON, regex errors
  - Acceptance: Each scorer has >=3 test cases (pass, fail, edge case)

## 6. LLM-as-Judge Scorers

**Context**: Flexible semantic evaluation using LLMs. More nuanced than heuristics but costs money and slower.
**Design**: Base `LLMJudge` type handles API calls, specific judges customize prompts/schemas.
**Priority**: All judge types equally important - comprehensive suite for production use.

- [ ] 6.1 Create `eval/llm_judge.go` with base LLM judge implementation
  - Context: Shared logic for calling LLM APIs, parsing structured outputs
  - Acceptance: `BaseLLMJudge` type with `CallJudge(ctx, prompt) (*JudgeResponse, error)`
- [ ] 6.2 Define `JudgeConfig` with model, prompt template, output schema, Temperature
  - Context: Configuration for judge behavior: model name, system prompt, response format
  - Acceptance: Fields: `Model string`, `PromptTemplate string`, `Temperature float64`, `Schema any`
  - Acceptance: Template supports placeholders: `{{.Input}}`, `{{.Output}}`, `{{.Expected}}`
- [ ] 6.3 Implement `Correctness` scorer (answer accuracy)
  - Context: "Is the output factually correct given the expected answer?"
  - Acceptance: Returns score 0-1 + rationale explaining correctness assessment
  - Acceptance: Prompt asks judge to compare output to ground truth
- [ ] 6.4 Implement `Guidelines` scorer (custom criteria)
  - Context: Flexible judge for user-defined criteria (e.g., "follows company style guide")
  - Acceptance: `NewGuidelines(criteria string)` - user provides custom evaluation prompt
  - Acceptance: Useful for domain-specific requirements
- [ ] 6.5 Implement `Relevance` scorer
  - Context: "Is the output relevant to the input question/task?"
  - Acceptance: Scores how well output addresses input without requiring ground truth
  - Acceptance: Useful when expected answer is unavailable
- [ ] 6.6 Implement `Groundedness` scorer
  - Context: "Is the output grounded in provided context/sources?" (anti-hallucination)
  - Acceptance: Requires context in test case metadata
  - Acceptance: Checks for unsupported claims
- [ ] 6.7 Add structured output parsing for judge responses
  - Context: Parse JSON from LLM response: `{score: 0.8, rationale: "..."}`
  - Acceptance: Handle malformed JSON gracefully with retries
  - Acceptance: Validate score is in [0, 1] range
- [ ] 6.8 Add retry logic for LLM API failures
  - Context: Production-ready resilience for API timeouts, rate limits
  - Acceptance: 3 retries max, 1s initial delay, 2x backoff, 10s max delay
  - Acceptance: Retry on 429, 5xx, connection errors
- [ ] 6.9 Add unit tests with mocked LLM responses
  - Context: Fast tests without real API calls; inject mock responses
  - Acceptance: Test successful scoring, malformed JSON, API errors, retries
  - Acceptance: Each judge has >=3 test cases

## 7. MLflow Export for Evaluation

**Context**: Bridge between eval framework and MLflow. Converts eval results to MLflow artifacts for tracking/visualization.
**Design**: Converters translate eval types → MLflow protos; evaluator option auto-exports after Run().
**Depends on**: 3.x (MLflow client), 4.x (eval framework core)

- [ ] 7.1 Create `eval/mlflow_export.go` with converter functions
  - Context: Pure converters (no side effects); testable without MLflow server
  - Acceptance: File contains only type conversion logic
- [ ] 7.2 Implement `ScoreToAssessment()` converter
  - Context: `ScoreToAssessment(score *Score, traceID, spanID string) *mlflow.Assessment`
  - Acceptance: Maps score value to assessment, rationale to feedback, metadata to custom fields
- [ ] 7.3 Implement `DatasetToRun()` converter for logging
  - Context: Creates MLflow run with dataset test cases as params/tags
  - Acceptance: Each test case becomes run param (input/expected)
  - Acceptance: Dataset metadata becomes run tags
- [ ] 7.4 Implement `EvaluationResultsToMetrics()` converter
  - Context: Aggregate scores → MLflow metrics for plotting/comparison
  - Acceptance: Creates metrics: `scorer_name.mean`, `scorer_name.median`, `pass_rate`
- [ ] 7.5 Add `WithMLflowExport(client *mlflow.Client, experimentID string)` evaluator option
  - Context: Enable automatic MLflow export during evaluation
  - Acceptance: Option stores client in evaluator config
- [ ] 7.6 Implement automatic assessment creation on evaluation completion
  - Context: After `Run()`, export all scores to MLflow as assessments
  - Acceptance: Calls `client.CreateAssessment()` for each score
  - Acceptance: Links assessments to trace/span if available
- [ ] 7.7 Add unit tests for converter functions
  - Context: Test type conversions without MLflow dependency
  - Acceptance: Verify field mappings, edge cases (nil fields, empty arrays)
- [ ] 7.8 Write `eval/README.md` section on MLflow integration
  - Context: Document how to enable MLflow export, view results in UI
  - Acceptance: Example code showing evaluator with MLflow export enabled

## 8. Tracing Integration

**Context**: Capture agent execution traces and send to MLflow for observability. Critical for debugging and evaluation.
**Design**: Fantasy callbacks → span events → MLflow TraceInfoV3; TracingCallbacks implements Fantasy's Callbacks interface.
**Priority**: Enables agent-specific scorers (5.7-5.8) and production monitoring.
**Depends on**: 2.x (proto types), 3.x (MLflow client)

- [ ] 8.1 Create `tracing/` package scaffold with config.go, tracer.go, callbacks.go, types.go
  - Context: Separate package for tracing logic; fantasy package imports this
  - Acceptance: Package structure: config (TracingConfig), tracer (Tracer), callbacks (TracingCallbacks), types (Trace, Span)
- [ ] 8.1a Define `Trace` struct in `types.go`
  - Context: In-memory trace with TraceID, ExperimentID, RequestTime, Spans, State, etc.
  - Acceptance: Fields match spec; sync.RWMutex for concurrent access
- [ ] 8.1b Define `Span` struct in `types.go`
  - Context: Span with SpanID, ParentID, Name, SpanType, StartTimeNs, EndTimeNs, Attributes
  - Acceptance: Fields match spec; sync.Mutex for concurrent access
- [ ] 8.1c Define `SpanEvent` struct in `types.go`
  - Context: Event with Name, Timestamp, Attributes (for streaming chunks, errors)
  - Acceptance: Timestamp is int64 nanoseconds since epoch
- [ ] 8.1d Define `SpanStatus` and `SpanStatusCode` types in `types.go`
  - Context: Status codes: UNSET, OK, ERROR with description
  - Acceptance: Constants SpanStatusUnset, SpanStatusOK, SpanStatusError
- [ ] 8.1e Define `TraceState` type in `types.go`
  - Context: Trace states: IN_PROGRESS, OK, ERROR
  - Acceptance: Constants TraceStateInProgress, TraceStateOK, TraceStateError
- [ ] 8.1f Define `TracingResult` struct in `types.go`
  - Context: Result of traced execution (Trace, FlushError)
  - Acceptance: Trace is the completed trace; FlushError is nil on success
- [ ] 8.1g Define `TokenUsage` struct in `types.go`
  - Context: Token counts (InputTokens, OutputTokens, TotalTokens)
  - Acceptance: JSON tags match MLflow attribute format
- [ ] 8.2 Implement `Tracer` type with span creation methods (crypto/rand for IDs)
  - Context: Factory that creates Traces and generates IDs
  - Acceptance: Trace IDs use `tr-` prefix with 32 hex chars (crypto/rand for security)
  - Acceptance: Span IDs are 16 hex chars (crypto/rand)
  - Acceptance: Methods: `NewTrace()`, `StartSpan(trace, name, parent)`, `EndSpan(span)`
  - Acceptance: Thread-safe (sync.Mutex protects internal state)
- [ ] 8.2a Define `SpanType` constants in `types.go`
  - Context: AGENT, LLM, TOOL, CHAIN, RETRIEVER, EMBEDDING, UNKNOWN
  - Acceptance: Type is string for JSON serialization compatibility
- [ ] 8.3 Implement span hierarchy: Agent → Step → [LLM | Tool]
  - Context: MLflow span model; Agent is root, Steps are children, LLM/Tool are grandchildren
  - Note: LLM and Tool spans are **siblings** (both children of Step span, not Tool child of LLM)
  - Note: LLM spans are created via **OnLLMStart/OnLLMFinish callbacks** (accurate timing)
  - Acceptance: Parent-child relationships stored in span metadata
  - Acceptance: LLM span created on OnLLMStart, ended on OnLLMFinish
  - REMOVED: Retroactive inference logic (no longer needed)
- [ ] 8.4 Define span attribute key constants (see tracing spec for full list)
  - Context: Standard keys for span attributes (e.g., `mlflow.spanType`, `mlflow.spanInputs`)
  - Acceptance: Constants for all required MLflow attributes
  - Acceptance: Type-safe attribute setters prevent typos
- [ ] 8.5 Implement `TracingConfig` struct in tracing package
  - Context: Configuration for tracing behavior; passed to `fantasy.WithTracing()`
  - Acceptance: Fields match spec (Client, ExperimentID, AgentName, ModelName, SessionID, Tags, FlushTimeout)
  - Acceptance: Defaults: AgentName="agent", SessionID=UUID v4, FlushTimeout=10s
- [ ] 8.6 Implement `TracingCallbacks` that implements Fantasy's Callbacks interface
  - Context: Callback handler that creates/manages spans in response to agent events
  - Acceptance: Implements OnAgentStart, OnAgentFinish, OnStepStart, OnStepFinish, OnToolCall, OnToolResult
  - Acceptance: Creates trace on OnAgentStart, flushes on OnAgentFinish
- [ ] 8.7 Create `fantasy.WithTracing(config)` agent option in fantasy package (agent.go)
  - Context: Fantasy package API for enabling tracing
  - Note: `WithTracing` lives in **fantasy package** for API consistency
  - Note: `TracingConfig` lives in **tracing package** for separation
  - Acceptance: `NewAgent(..., fantasy.WithTracing(tracingConfig))` works
- [ ] 8.8 Hook OnStepStart/OnStepFinish for step spans
  - Context: Fantasy callbacks → step span lifecycle
  - Acceptance: OnStepStart creates step span with step number
  - Acceptance: OnStepFinish ends step span with step outputs
  - Acceptance: Step span captures full step execution
- [ ] 8.8a Hook OnLLMStart/OnLLMFinish for LLM spans
  - Context: Fantasy callbacks → accurate LLM span lifecycle
  - Acceptance: OnLLMStart creates LLM span with model name and call inputs
  - Acceptance: OnLLMFinish ends LLM span with usage and finish reason
  - Acceptance: LLM span timestamps are accurate (not inferred)
  - Acceptance: Duration includes retry attempts
- [ ] 8.9 Hook OnToolCall/OnToolResult for tool spans
  - Context: Tool execution tracking; handles concurrent tool calls
  - Acceptance: OnToolCall creates tool span with tool name and input
  - Acceptance: OnToolResult ends tool span with output/error
  - Acceptance: Tool spans are siblings of LLM span (not children)
  - Acceptance: Match spans by tool call ID (enables concurrent tool calls)
- [ ] 8.10 Implement automatic trace flush on OnAgentFinish (blocking with timeout)
  - Context: Ensure traces reach MLflow when agent completes
  - Acceptance: Flush happens in OnAgentFinish callback before returning
  - Acceptance: FlushTimeout (default 10s) prevents indefinite blocking
  - Acceptance: Flush errors captured in TracingResult, don't fail agent call
- [ ] 8.11 Implement thread safety
  - Context: Multiple concurrent agent calls must not corrupt traces
  - Acceptance: sync.Mutex protects span map modifications
  - Acceptance: Context-based trace storage (one trace per agent call)
- [ ] 8.12 Implement `SpanFromContext(ctx)` helper to retrieve current span
  - Context: Enable custom span attributes in user code
  - Acceptance: `SpanFromContext(ctx)` returns current span or nil
  - Acceptance: User can call `span.SetAttribute(key, value)` for custom metadata
- [ ] 8.13 Implement context cancellation handling
  - Context: Graceful shutdown when agent call is cancelled
  - Acceptance: Context cancellation propagates to flush (respect cancellation)
  - Acceptance: Cancelled traces marked with ERROR status
- [ ] 8.14 Implement orphan span cleanup
  - Context: Prevent memory leaks from spans never ended (bugs, panics)
  - Acceptance: Background goroutine closes spans open >5 minutes
  - Acceptance: Orphan spans marked with INTERNAL_ERROR status
- [ ] 8.15 Implement panic recovery in TracingCallbacks
  - Context: Ensure traces flushed even if agent panics
  - Acceptance: defer/recover in callbacks ends all open spans with ERROR
  - Acceptance: Flush called before re-panicking
  - Acceptance: Original panic preserved (re-throw after flush)
- [ ] 8.16 Implement `FlushAll(ctx)` for flushing all pending traces
  - Context: Application shutdown; ensure no traces lost
  - Acceptance: Flushes all active traces across all agent calls
  - Acceptance: Respects context deadline
- [ ] 8.17 Implement UTF-8 safe truncation for preview strings
  - Context: Span input/output previews limited to 1000 chars; must not break UTF-8
  - Acceptance: Truncate at rune boundary (use `utf8.DecodeLastRuneInString`)
  - Acceptance: Add "..." suffix if truncated
- [ ] 8.18 Add unit tests for span generation
  - Context: Test span hierarchy, attributes, timing without MLflow
  - Acceptance: Test agent → step → LLM/tool hierarchy creation
  - Acceptance: Test concurrent tool calls don't mix spans
  - Acceptance: Test orphan cleanup, panic recovery
- [ ] 8.19 Add integration test with real agent execution
  - Context: End-to-end test with local MLflow server
  - Acceptance: Create agent with tracing enabled, run simple task, verify trace in MLflow
  - Acceptance: Verify span hierarchy and attributes in MLflow UI
- [ ] 8.20 Write `tracing/README.md` with setup and usage guide
  - Context: Document how to enable tracing, configure options, view traces
  - Acceptance: Example code with Fantasy agent + tracing configuration
  - Acceptance: Screenshots or instructions for viewing traces in MLflow UI

## 9. Documentation

**Context**: Incremental documentation approach - READMEs written alongside components (already done in 1.8, 3.14, 7.8, 8.20)
**Priority**: Final polish; comprehensive example demonstrating all features together.

- [ ] 9.1 Verify `proto/README.md` completeness (from 1.8)
  - Context: Should cover buf setup, proto generation, troubleshooting
  - Acceptance: User can generate protos from scratch following README
- [ ] 9.2 Verify `mlflow/README.md` completeness (from 3.14)
  - Context: Should have quick start, common operations, error handling
  - Acceptance: User can create client and make basic API calls
- [ ] 9.3 Verify `eval/README.md` completeness (from 7.8)
  - Context: Should cover scorer usage, MLflow export, result interpretation
  - Acceptance: User can run eval and view results in MLflow
- [ ] 9.4 Verify `tracing/README.md` completeness (from 8.20)
  - Context: Should show Fantasy integration, configuration, trace viewing
  - Acceptance: User can enable tracing on agent and view traces in MLflow UI
- [ ] 9.5 Update main Fantasy README with MLflow integration section
  - Context: High-level overview linking to component READMEs
  - Acceptance: Section explains what MLflow integration provides (tracing, eval, observability)
  - Acceptance: Links to all component READMEs
- [ ] 9.6 Create comprehensive example in `examples/mlflow/`
  - Context: Demonstrates full workflow: agent with tracing + evaluation + MLflow export
  - Acceptance: Example shows agent execution, trace creation, eval run, results in MLflow
  - Acceptance: README in examples/mlflow/ with step-by-step setup (start MLflow, run example, view UI)
  - Acceptance: Example is runnable with `go run examples/mlflow/main.go`

## 10. Testing and Validation

**Context**: Final validation before completion. Ensure all components work together and follow project standards.
**Priority**: Quality gate - no task is complete until tests pass.

- [ ] 10.1 Run `task lint` and fix any issues
  - Context: Ensure code follows Go style, no vet warnings
  - Acceptance: `task lint` exits with status 0 (no errors)
- [ ] 10.2 Run `task test` and ensure all tests pass
  - Context: All unit and integration tests must pass
  - Acceptance: `task test` shows 100% test pass rate
  - Acceptance: No skipped tests without documented reason
- [ ] 10.3 Verify proto generation is reproducible
  - Context: Ensure `task proto:gen` produces identical output every time
  - Acceptance: Run `task proto:gen` twice, verify no git diffs in `proto/gen/`
  - Acceptance: CI can regenerate protos without changing files
- [ ] 10.4 Test against local MLflow server (manual end-to-end verification)
  - Context: Verify all APIs work with real MLflow server
  - Acceptance: Start MLflow with `task mlflow:start`
  - Acceptance: Run integration tests with real server (3.13, 8.19)
  - Acceptance: Manually verify traces visible in MLflow UI at http://localhost:5000
  - Acceptance: Manually verify assessments linked to traces in UI
- [ ] 10.5 Test comprehensive example from 9.6
  - Context: Ensure example workflow works end-to-end
  - Acceptance: `go run examples/mlflow/main.go` completes without errors
  - Acceptance: Trace appears in MLflow UI
  - Acceptance: Evaluation results appear as assessments
- [ ] 10.6 Review test coverage for critical paths
  - Context: Ensure adequate coverage for core functionality
  - Acceptance: Run `go test -cover ./...` - aim for >80% coverage on critical packages
  - Acceptance: Core packages: mlflow, eval, tracing should have high coverage

## Dependencies

**Strict ordering:**
- 0.x (MLflow setup) → enables manual testing throughout development
- 1.x (proto infrastructure) → 2.x (proto curation) → 3.x (REST client)
- 3.x + 4.x → 7.x (MLflow export needs both client and eval types)
- 2.x + 3.x → 8.x (tracing needs proto types + client)
- 8.x → 5.7-5.8 (agent scorers need `*tracing.Trace` type)
- All implementation (1.x-8.x) → 9.x (documentation review) → 10.x (final validation)

**Independent work (can parallelize):**
- 4.x (eval core), 5.1-5.6 (basic heuristic scorers), 6.x (LLM judges) - no cross-dependencies
- Documentation tasks (1.8, 3.14, 7.8, 8.20) - can write alongside implementation

## Parallelizable Work

**Phase 1** (Foundation - sequential):
1. 0.x - MLflow server setup (enables testing)
2. 1.x - Proto infrastructure (buf setup)
3. 2.x - MLflow proto curation (depends on 1.x)

**Phase 2** (Critical path - must complete before Phase 3):
- 3.x - MLflow REST client (depends on 2.x)
  - **Checkpoint**: After 3.x completes, can start Phase 3 streams

**Phase 3** (Parallel streams after 3.x completes):
- **Stream A** (Eval Core): 4.x → 5.1-5.6 → 6.x
  - Context: Build evaluation framework and basic scorers
  - No external dependencies once 4.x starts
  - Documentation: 7.8 (eval README) - can write after 6.x

- **Stream B** (Tracing): 8.1-8.17 → 8.18-8.19 (tests)
  - Context: Build tracing integration
  - Depends on: 2.x (proto types), 3.x (client)
  - Documentation: 8.20 (tracing README) - can write alongside

- **Stream C** (MLflow Export): 7.1-7.7
  - Context: Connect eval to MLflow
  - Depends on: 3.x (client) + 4.x (eval types from Stream A)
  - Wait for: 4.x from Stream A before starting

**Phase 4** (After Stream B completes):
- 5.7-5.8 - Agent-specific scorers (requires `*tracing.Trace` from Stream B)

**Phase 5** (Final integration):
- 9.x - Documentation review (all READMEs from 1.8, 3.14, 7.8, 8.20)
- 9.6 - Comprehensive example (requires all features working)
- 10.x - Testing and validation (final quality gate)

**Optimal parallelization strategy:**
1. Complete 0.x, 1.x, 2.x, 3.x sequentially (foundation)
2. Start Streams A, B, C in parallel (eval + tracing + export)
3. When Stream B done → complete 5.7-5.8
4. When all streams done → Phase 5 (docs + validation)

**Estimated work distribution:**
- Foundation (0.x-3.x): ~40% of implementation effort
- Parallel streams (4.x-8.x): ~50% of implementation effort
- Final validation (9.x-10.x): ~10% of effort
