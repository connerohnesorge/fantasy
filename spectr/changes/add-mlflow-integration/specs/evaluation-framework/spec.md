## ADDED Requirements

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
    Outputs      map[string]any  // Generated outputs to evaluate
    Expectations map[string]any  // Expected values for comparison
    Trace        *Trace          // Optional trace data (nil for scorers that don't need it)
}
```

#### Scenario: Score types
- GIVEN a scorer returning a result
- WHEN the Score is created
- THEN it has the following structure:
```go
type Score struct {
    Value     any             // bool, float64, or string
    Rationale string          // Explanation of the score
    Metadata  map[string]any  // Additional structured information
}
```
- AND Value can be boolean (pass/fail), numeric (0.0-1.0), or categorical (string label)

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

### Requirement: Evaluation Runner

The system SHALL provide an Evaluator for running scorers against datasets.

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

#### Scenario: Predict function
- GIVEN a dataset without pre-generated outputs
- WHEN a predict function is provided via `WithPredict(fn PredictFunc)`
- THEN the predict function is called for each test case to generate outputs
- AND generated outputs are passed to scorers
- AND the predict function signature is:
```go
// PredictFunc generates outputs from inputs.
// Returns outputs map and optional trace for agent-specific scorers.
type PredictFunc func(ctx context.Context, inputs map[string]any) (outputs map[string]any, trace *Trace, err error)
```

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
- AND null values are considered equal to missing keys (optional: configurable)

#### Scenario: NumericRange scorer
- GIVEN min and max bounds and numeric output
- WHEN NumericRange scorer is applied
- THEN it returns true if value is within range

#### Scenario: ToolCallTrajectory scorer
- GIVEN expected tool call sequence and trace with tool spans
- WHEN ToolCallTrajectory scorer is applied
- THEN it returns true if tool names match expected sequence in order

#### Scenario: StepValidation scorer
- GIVEN expected step count and validation rules
- WHEN StepValidation scorer is applied
- THEN it validates the agent completed within expected step range
- AND optionally validates step content patterns via regex
- AND returns true if all validations pass

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

#### Scenario: Guidelines scorer
- GIVEN custom guidelines and actual output
- WHEN Guidelines scorer is applied
- THEN an LLM judges if output follows the guidelines
- AND returns pass/fail with rationale

#### Scenario: Relevance scorer
- GIVEN user query and actual output
- WHEN Relevance scorer is applied
- THEN an LLM judges if output is relevant to the query
- AND returns relevance score with rationale

#### Scenario: Groundedness scorer
- GIVEN retrieved documents and actual output
- WHEN Groundedness scorer is applied
- THEN an LLM judges if output is grounded in retrieved content
- AND returns grounded/not_grounded with rationale

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

#### Scenario: Result aggregation
- GIVEN completed evaluation of all test cases
- WHEN results are aggregated
- THEN summary statistics are computed per scorer (pass rate, mean, std)
- AND individual test case results are available

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
- THEN source type is set to CODE or LLM_JUDGE
- AND value and rationale are mapped correctly

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
