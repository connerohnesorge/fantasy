## ADDED Requirements

### Requirement: Package Dependencies

The eval package SHALL import the tracing package for type safety with agent-specific scorers.

#### Scenario: Tracing dependency
- GIVEN the eval package uses *tracing.Trace for agent-specific scorers
- WHEN users import the eval package
- THEN the tracing package is also imported transitively
- AND users who don't use agent-specific scorers can ignore this dependency
- AND no MLflow server connection is required to use the eval package locally

### Requirement: Scorer Interface

The system SHALL define a Scorer interface for evaluation criteria.

#### Scenario: Scorer interface definition
- GIVEN the eval package
- WHEN Scorer interface is defined
- THEN it has the following Go signature:
```go
// Scorer evaluates outputs against expectations.
type Scorer interface {
    // Name returns the scorer's identifier.
    Name() string

    // Score evaluates the given input/output pair.
    // ctx: context for cancellation/timeout
    // input: ScorerInput containing outputs, expectations, and optional trace
    // Returns: Score result or error
    Score(ctx context.Context, input ScorerInput) (Score, error)
}

// ScorerInput contains all data needed for scoring.
type ScorerInput struct {
    Outputs      map[string]any   // Generated outputs to evaluate
    Expectations map[string]any   // Expected values for comparison
    Trace        *tracing.Trace   // Optional trace data (nil for scorers that don't need it)
    Inputs       map[string]any   // Original inputs (for context in scoring)
}
```

#### Scenario: Score types
- GIVEN a scorer returning a result
- WHEN the Score is created
- THEN it has the following structure:
```go
type Score struct {
    Value     any             // bool, float64, int, or string
    Rationale string          // Explanation of the score
    Metadata  map[string]any  // Additional structured information
    Error     error           // Non-nil if scorer failed (Value should be nil/zero)
}
```
- AND Value can be boolean (pass/fail), numeric (0.0-1.0 for normalized, any range for raw), or categorical (string label)

#### Scenario: Score value range handling
- GIVEN a numeric score value
- WHEN the score is validated
- THEN values are accepted as-is (no clamping or rejection)
- AND normalized scorers (LLM judges) return values in 0.0-1.0 range
- AND heuristic scorers may return values in any range appropriate to the metric
- AND boolean scores are treated as 1.0 (true) or 0.0 (false) for aggregation

#### Scenario: Score with error
- GIVEN a scorer that encounters an error
- WHEN the error is captured
- THEN Score.Error is set to the error
- AND Score.Value is set to nil or zero value
- AND Score.Rationale may contain error context
- AND the test case is marked as having an error (not pass or fail)

### Requirement: Dataset Types

The system SHALL provide Dataset and TestCase types for evaluation inputs.

#### Scenario: TestCase structure
- GIVEN a test case definition
- WHEN the test case is created
- THEN it contains inputs (map[string]any), expectations (map[string]any)
- AND optionally pre-generated outputs (map[string]any) and tags (map[string]string)

#### Scenario: TestCase with pre-generated outputs
- GIVEN a test case with Outputs field populated
- WHEN the test case is evaluated
- THEN the pre-generated outputs are used directly
- AND no predict function is called for this test case
- AND trace field is nil (no execution occurred)

#### Scenario: TestCase without pre-generated outputs
- GIVEN a test case with Outputs field nil or empty
- WHEN the test case is evaluated
- THEN a predict function MUST be provided via WithPredict()
- AND the predict function generates outputs and optional trace
- AND error is returned if no predict function is configured

#### Scenario: Dataset structure
- GIVEN a dataset definition
- WHEN the dataset is created
- THEN it contains a name, list of test cases, and optional metadata

#### Scenario: JSON dataset loading
- GIVEN a JSON file with test cases
- WHEN `LoadDataset(path)` is called
- THEN the file is parsed into a Dataset struct
- AND validation errors are returned for malformed files

#### Scenario: Dataset validation rules
- GIVEN a dataset is loaded or created
- WHEN validation is performed
- THEN the following rules are enforced:
  - Dataset must have a non-empty name
  - Dataset must have at least one test case
  - Each test case must have a non-nil Inputs map
  - Inputs and Expectations values must be JSON-serializable (no channels, functions)
  - If Outputs is nil and no PredictFunc is provided, an error is returned during evaluation
- AND validation errors include the specific field and test case index that failed

### Requirement: Evaluation Runner

The system SHALL provide an Evaluator for running scorers against datasets.

#### Scenario: Evaluator type definition
- GIVEN the eval package
- WHEN Evaluator is defined
- THEN it has the following structure:
```go
// Evaluator runs scorers against datasets.
type Evaluator struct {
    // internal fields for configuration
}

// NewEvaluator creates an Evaluator with optional configuration.
func NewEvaluator(opts ...EvalOption) *Evaluator

// Run executes evaluation on the given dataset with the specified scorers.
func (e *Evaluator) Run(ctx context.Context, dataset *Dataset, scorers []Scorer, opts ...RunOption) (*Results, error)

// EvalOption configures the Evaluator at creation time.
type EvalOption func(*Evaluator)

// RunOption configures a specific evaluation run.
type RunOption func(*runConfig)

// Available RunOptions:
func WithParallelism(n int) RunOption           // Set number of parallel workers (default: 1 = sequential)
func WithTimeout(d time.Duration) RunOption     // Set per-test-case timeout
func WithPredict(fn PredictFunc) RunOption      // Set prediction function for generating outputs
func WithTracing(config TracingConfig) RunOption // Enable tracing for test cases
```

#### Scenario: Sequential evaluation
- GIVEN a dataset and list of scorers
- WHEN `evaluator.Run(ctx, dataset, scorers)` is called with default options
- THEN test cases are processed sequentially
- AND results are returned in order

#### Scenario: Parallel evaluation
- GIVEN a dataset and list of scorers
- WHEN `evaluator.Run(ctx, dataset, scorers, WithParallelism(10))` is called
- THEN test cases are processed concurrently with up to 10 workers
- AND results are collected and returned

#### Scenario: Parallel evaluation semantics
- GIVEN parallel evaluation is enabled
- WHEN test cases are processed concurrently
- THEN results are returned in original test case order (stable ordering)
- AND if one test case fails, other test cases continue processing
- AND context cancellation stops all pending test cases
- AND each scorer receives its own goroutine per test case
- AND errors are collected per test case, not propagated globally

#### Scenario: Timeout handling
- GIVEN a scorer that takes too long
- WHEN evaluation runs with `WithTimeout(5*time.Second)`
- THEN the scorer is cancelled after timeout
- AND an error score is recorded for that test case

### Requirement: Error Handling

The system SHALL handle errors gracefully during evaluation.

#### Scenario: Scorer panic recovery
- GIVEN a scorer that panics during execution
- WHEN the panic occurs
- THEN the panic is recovered (not propagated)
- AND an error Score is recorded with Rationale containing the panic message
- AND evaluation continues with remaining test cases
- AND the panic is logged at ERROR level

#### Scenario: Dataset file not found
- GIVEN a dataset is loaded from a file path
- WHEN the file does not exist or is not readable
- THEN an error is returned from the dataset loading function
- AND the error wraps the underlying file system error
- AND evaluation does not start

#### Scenario: Context cancellation during evaluation
- GIVEN an evaluation is in progress
- WHEN the context is cancelled (e.g., user interrupt)
- THEN in-progress scorers are cancelled via their context
- AND completed scores are preserved in Results
- AND remaining test cases are skipped
- AND Results.Errors includes a "context cancelled" error

#### Scenario: MLflow export failure
- GIVEN evaluation completes and export is requested
- WHEN the MLflow client fails to connect or send data
- THEN the error is added to Results.Errors with Phase="export"
- AND local results are still returned
- AND the caller can decide whether to retry or ignore

#### Scenario: Trace extraction failure
- GIVEN a test case runs with tracing enabled
- WHEN trace extraction fails (missing spans, malformed data)
- THEN TestCaseResult.Trace is nil
- AND a warning is logged
- AND agent-specific scorers (ToolCallTrajectory, StepValidation) receive nil trace
- AND those scorers return error Scores indicating trace unavailable

#### Scenario: Type coercion error in ScorerInput
- GIVEN a scorer expects a specific input type
- WHEN the input map contains an incompatible type
- THEN the scorer returns an error Score
- AND Rationale includes details about the type mismatch
- AND the error is not propagated (evaluation continues)

#### Scenario: Predict function
- GIVEN a dataset without pre-generated outputs
- WHEN a predict function is provided via `WithPredict(fn PredictFunc)`
- THEN the predict function is called for each test case to generate outputs
- AND generated outputs are passed to scorers
- AND the predict function signature is:
```go
// PredictFunc generates outputs from inputs.
// Returns outputs map and optional trace for agent-specific scorers.
// The trace is *tracing.Trace from the tracing package.
type PredictFunc func(ctx context.Context, inputs map[string]any) (outputs map[string]any, trace *tracing.Trace, err error)
```
- AND if the predict function returns an error, the test case is marked as errored
- AND if trace is nil, agent-specific scorers (ToolCallTrajectory, StepValidation) will skip or error

### Requirement: Heuristic Scorers

The system SHALL provide built-in heuristic scorers.

#### Scenario: ExactMatch scorer
- GIVEN expected string and actual output
- WHEN ExactMatch scorer is applied
- THEN it returns true if strings are equal, false otherwise

#### Scenario: Contains scorer
- GIVEN expected substring and actual output
- WHEN Contains scorer is applied
- THEN it returns true if output contains substring

#### Scenario: Regex scorer
- GIVEN a regex pattern and actual output
- WHEN Regex scorer is applied
- THEN it returns true if pattern matches output

#### Scenario: JSONMatch scorer
- GIVEN expected JSON structure and actual output
- WHEN JSONMatch scorer is applied
- THEN it returns true if structures are semantically equivalent
- AND comparison ignores whitespace formatting
- AND comparison ignores object key ordering

#### Scenario: JSONMatch null handling configuration
- GIVEN a JSONMatch scorer with configurable options
- WHEN `JSONMatchScorer(JSONMatchOptions{...})` is called
- THEN the following options are available:
```go
type JSONMatchOptions struct {
    // NullEqualsMissing: if true, null values are considered equal to missing keys
    // Default: false (null and missing are different)
    NullEqualsMissing bool

    // IgnoreArrayOrder: if true, array elements are compared as sets (order doesn't matter)
    // Default: false (array order matters)
    IgnoreArrayOrder bool

    // FloatTolerance: tolerance for float comparisons (absolute difference)
    // Default: 0 (exact match required)
    FloatTolerance float64
}
```

#### Scenario: NumericRange scorer
- GIVEN min and max bounds and numeric output
- WHEN NumericRange scorer is applied
- THEN it returns true if value is within range

#### Scenario: ToolCallTrajectory scorer
- GIVEN expected tool call sequence and trace with tool spans
- WHEN ToolCallTrajectory scorer is applied
- THEN it returns true if tool names match expected sequence in order
- AND requires a non-nil *tracing.Trace in ScorerInput
- AND extracts tool spans from the trace by SpanType == "TOOL"
- AND compares tool names in chronological order (by start time)
- AND the expected sequence is specified via `expectations["tool_sequence"]` as []string

#### Scenario: ToolCallTrajectory options
- GIVEN a ToolCallTrajectory scorer with options
- WHEN `ToolCallTrajectoryScorer(ToolCallTrajectoryOptions{...})` is called
- THEN the following options are available:
```go
type ToolCallTrajectoryOptions struct {
    // Strict: if true, requires exact match (no extra tools allowed)
    // Default: false (extra tools are allowed as long as expected sequence is present)
    Strict bool

    // IgnoreOrder: if true, checks that all expected tools were called (in any order)
    // Default: false (order matters)
    IgnoreOrder bool
}
```

#### Scenario: StepValidation scorer
- GIVEN expected step count and validation rules
- WHEN StepValidation scorer is applied
- THEN it validates the agent completed within expected step range
- AND optionally validates step content patterns via regex
- AND returns true if all validations pass
- AND requires a non-nil *tracing.Trace in ScorerInput
- AND extracts step spans from the trace by SpanType == "CHAIN"

#### Scenario: StepValidation options
- GIVEN a StepValidation scorer with options
- WHEN `StepValidationScorer(StepValidationOptions{...})` is called
- THEN the following options are available:
```go
type StepValidationOptions struct {
    // MinSteps: minimum number of steps required (default: 0, no minimum)
    MinSteps int

    // MaxSteps: maximum number of steps allowed (default: 0, no maximum)
    MaxSteps int

    // ContentPatterns: regex patterns that must match step outputs
    // Key is step index (0-based), value is regex pattern
    // If step index doesn't exist, validation fails
    ContentPatterns map[int]string
}
```

### Requirement: LLM-as-Judge Scorers

The system SHALL provide LLM-based evaluation scorers using Fantasy providers.

#### Scenario: LLM judge configuration
- GIVEN a judge scorer definition
- WHEN the scorer is configured
- THEN it specifies the model, prompt template, and output schema
- AND Fantasy's LanguageModel is used for inference
- AND the configuration structure is:
```go
// JudgeConfig configures an LLM-as-judge scorer.
type JudgeConfig struct {
    Model          fantasy.LanguageModel  // Required: the LLM to use for judging
    PromptTemplate string                 // Prompt template with {{.Input}}, {{.Output}}, etc.
    OutputSchema   any                    // Expected structured output schema
    Temperature    float64                // Model temperature (default: 0.0 for determinism)
}
```

#### Scenario: Correctness scorer
- GIVEN expected answer and actual output
- WHEN Correctness scorer is applied
- THEN an LLM judges if the output correctly answers the question
- AND returns yes/no with rationale
- AND uses the following prompt template:
```
Consider the following question, claim and document. You must determine whether the claim is
supported by the document in the context of the question. Do not focus on the correctness or
completeness of the claim. Do not make assumptions, approximations, or bring in external knowledge.

<question>{{.Input}}</question>
<claim>{{.ExpectedAnswer}}</claim>
<document>{{.Input}} - {{.Output}}</document>

Please indicate whether each statement in the claim is supported by the document in the context
of the question using only the following json format. Do not use any markdown formatting.
{
  "rationale": "Reason for the assessment. Start with 'Let's think step by step'",
  "result": "yes|no"
}
```

#### Scenario: Guidelines scorer
- GIVEN custom guidelines and actual output
- WHEN Guidelines scorer is applied
- THEN an LLM judges if output follows the guidelines
- AND returns pass/fail with rationale
- AND uses the following prompt template:
```
Given the following set of guidelines and some inputs, please assess whether the inputs fully
comply with all the provided guidelines. Only focus on the provided guidelines and not the
correctness, relevance, or effectiveness of the inputs.

<guidelines>
{{range .Guidelines}}<guideline>{{.}}</guideline>
{{end}}
</guidelines>
<input>{{.Input}}</input>
<output>{{.Output}}</output>

Please provide your assessment using only the following json format. Do not use any markdown formatting.
If any of the guidelines are not satisfied, the result must be "no".
{
  "rationale": "Detailed reasoning for your assessment. Start with 'Let's think step by step.'",
  "result": "yes|no"
}
```

#### Scenario: Relevance scorer
- GIVEN user query and actual output
- WHEN Relevance scorer is applied
- THEN an LLM judges if output is relevant to the query
- AND returns relevance score with rationale
- AND uses the following prompt template:
```
Consider the following question and answer. You must determine whether the answer provides
information that is (fully or partially) relevant to the question. Do not focus on the correctness
or completeness of the answer. Do not make assumptions, approximations, or bring in external knowledge.

<question>{{.Input}}</question>
<answer>{{.Output}}</answer>

Please indicate whether the answer contains information that is relevant to the question using only
the following json format. Do not use any markdown formatting.
{
  "rationale": "Reason for the assessment. Start with 'Let's think step by step'",
  "result": "yes|no"
}
```

#### Scenario: Groundedness scorer
- GIVEN retrieved documents and actual output
- WHEN Groundedness scorer is applied
- THEN an LLM judges if output is grounded in retrieved content
- AND returns grounded/not_grounded with rationale
- AND uses the following prompt template:
```
Consider the following claim and document. You must determine whether claim is supported by the
document. Do not focus on the correctness or completeness of the claim. Do not make assumptions,
approximations, or bring in external knowledge.

<claim>
  <question>{{.Input}}</question>
  <answer>{{.Output}}</answer>
</claim>
<document>{{.RetrievalContext}}</document>

Please indicate whether each statement in the claim is supported by the document using only the
following json format. Do not use any markdown formatting.
{
  "rationale": "Reason for the assessment. Start with 'Let's think step by step'",
  "result": "yes|no"
}
```

#### Scenario: Judge retry on failure
- GIVEN an LLM API error during scoring
- WHEN the judge scorer encounters the error
- THEN it retries with exponential backoff using default configuration:
  - Maximum retries: 3
  - Initial delay: 1 second
  - Backoff multiplier: 2x
  - Maximum delay: 10 seconds
- AND retries are triggered for: rate limit errors (429), server errors (5xx), connection timeouts
- AND retries are NOT triggered for: client errors (4xx except 429), validation errors
- AND returns error score with rationale "LLM judge failed after 3 retries" if all retries fail

### Requirement: Evaluation Results

The system SHALL provide structured evaluation results.

#### Scenario: Results structure definition
- GIVEN an evaluation completes
- WHEN results are returned
- THEN the Results type is defined as:
```go
// Results contains the complete evaluation output.
type Results struct {
    TestCases  []TestCaseResult       // Individual results per test case
    Summary    map[string]ScorerStats // Aggregated stats per scorer name
    Errors     []EvalError            // Evaluation-level errors (not per-test-case)
    StartTime  time.Time              // When evaluation started
    EndTime    time.Time              // When evaluation completed
    TotalTests int                    // Total number of test cases processed
}

// TestCaseResult contains results for a single test case.
type TestCaseResult struct {
    TestCase   TestCase             // The original test case
    Outputs    map[string]any       // Generated outputs (if PredictFunc was run)
    Scores     map[string]Score     // Scores keyed by scorer name
    Trace      *tracing.Trace       // Captured trace (if tracing was enabled)
}

// ScorerStats contains aggregated statistics for a scorer.
type ScorerStats struct {
    PassRate   float64 // Percentage of passing scores (0.0-1.0)
    Mean       float64 // Mean score value (for numeric scores)
    StdDev     float64 // Standard deviation (for numeric scores)
    ErrorRate  float64 // Percentage of errored scores (0.0-1.0)
    ErrorCount int     // Number of test cases where scorer errored
    Count      int     // Total number of scores computed
}

// EvalError represents an evaluation-level error.
type EvalError struct {
    Phase   string // "setup", "predict", "score", "export"
    Message string
    Cause   error
}
```

#### Scenario: Result aggregation
- GIVEN completed evaluation of all test cases
- WHEN results are aggregated
- THEN summary statistics are computed per scorer using these formulas:

**Pass Rate** (for boolean scores):
```
pass_rate = count(score.Value == true) / count(all_scores)
```
- Only includes test cases where Score.Error is nil
- Expressed as a float64 between 0.0 and 1.0

**Mean** (for numeric scores):
```
mean = sum(score.Value) / count(all_scores)
```
- Only includes test cases where Score.Error is nil and Score.Value is numeric
- Type coercion for aggregation:
  - float64: used directly
  - int/int64: converted to float64
  - bool: true = 1.0, false = 0.0
  - string: skipped (logged as warning)

**Standard Deviation** (sample std, for numeric scores):
```
std = sqrt(sum((score.Value - mean)^2) / (count - 1))
```
- Uses sample standard deviation (Bessel's correction with n-1)
- Returns 0.0 if count <= 1
- Only includes test cases where Score.Error is nil

**Error Rate**:
```
error_rate = count(Score.Error != nil) / count(all_test_cases)
```

- AND individual test case results are available via `Results.TestCases`

#### Scenario: Result export
- GIVEN evaluation results
- WHEN export is requested
- THEN results can be serialized to JSON
- AND include all test cases, scores, and summary

### Requirement: MLflow Export

The system SHALL support exporting evaluation results to MLflow.

#### Scenario: Score to Assessment conversion
- GIVEN a Score from a scorer
- WHEN converted to MLflow Assessment
- THEN the conversion follows these rules:
```go
// ScoreToAssessment converts an eval.Score to an mlflowclient.Assessment
func ScoreToAssessment(scorerName string, score Score, isLLMJudge bool) *mlflowclient.Assessment {
    sourceType := "CODE"
    if isLLMJudge {
        sourceType = "LLM_JUDGE"
    }

    return &mlflowclient.Assessment{
        Name: scorerName,
        Source: mlflowclient.AssessmentSource{
            SourceType: sourceType,
            SourceID:   scorerName,
        },
        Feedback: &mlflowclient.FeedbackValue{
            Value: score.Value,
            Error: convertError(score.Error),
        },
        Rationale: score.Rationale,
        Metadata:  convertMetadata(score.Metadata),
    }
}
```
- AND source type is "CODE" for heuristic scorers
- AND source type is "LLM_JUDGE" for LLM-based scorers
- AND source type is "HUMAN" for human-provided expectations (if any)
- AND Score.Error is converted to AssessmentError if present
- AND Score.Metadata keys are converted to string values

#### Scenario: Automatic assessment logging
- GIVEN evaluation run with `WithMLflowExport(client)` option
- WHEN evaluation completes
- THEN assessments are created for each score
- AND linked to the appropriate trace

#### Scenario: Metrics logging
- GIVEN evaluation results summary
- WHEN exported to MLflow
- THEN aggregate metrics are logged to the experiment run
- AND include pass rates and score statistics
