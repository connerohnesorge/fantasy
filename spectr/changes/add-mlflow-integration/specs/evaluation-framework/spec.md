## ADDED Requirements

### Requirement: Scorer Interface

The system SHALL define a Scorer interface for evaluation criteria.

#### Scenario: Scorer signature
- GIVEN a Scorer implementation
- WHEN the scorer is invoked
- THEN it receives context, outputs, expectations, and optional trace
- AND returns a Score with value, rationale, and metadata

#### Scenario: Score types
- GIVEN a scorer returning a result
- WHEN the Score is created
- THEN the value can be boolean, numeric (float64), or categorical (string)
- AND rationale explains the scoring decision
- AND metadata contains additional structured information

### Requirement: Dataset Types

The system SHALL provide Dataset and TestCase types for evaluation inputs.

#### Scenario: TestCase structure
- GIVEN a test case definition
- WHEN the test case is created
- THEN it contains inputs (map[string]any), expectations (map[string]any)
- AND optionally pre-generated outputs and tags

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

#### Scenario: Timeout handling
- GIVEN a scorer that takes too long
- WHEN evaluation runs with `WithTimeout(5*time.Second)`
- THEN the scorer is cancelled after timeout
- AND an error score is recorded for that test case

#### Scenario: Predict function
- GIVEN a dataset without pre-generated outputs
- WHEN a predict function is provided
- THEN the predict function is called for each test case to generate outputs
- AND generated outputs are passed to scorers

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
- THEN it returns true if structures are equivalent (ignoring whitespace)

#### Scenario: NumericRange scorer
- GIVEN min and max bounds and numeric output
- WHEN NumericRange scorer is applied
- THEN it returns true if value is within range

#### Scenario: ToolCallTrajectory scorer
- GIVEN expected tool call sequence and trace with tool spans
- WHEN ToolCallTrajectory scorer is applied
- THEN it returns true if tool names match expected sequence in order

### Requirement: LLM-as-Judge Scorers

The system SHALL provide LLM-based evaluation scorers using Fantasy providers.

#### Scenario: LLM judge configuration
- GIVEN a judge scorer definition
- WHEN the scorer is configured
- THEN it specifies the model, prompt template, and output schema
- AND Fantasy's LanguageModel is used for inference

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

#### Scenario: RelevanceToQuery scorer
- GIVEN user query and actual output
- WHEN RelevanceToQuery scorer is applied
- THEN an LLM judges if output is relevant to the query
- AND returns relevance score with rationale

#### Scenario: RetrievalGroundedness scorer
- GIVEN retrieved documents and actual output
- WHEN RetrievalGroundedness scorer is applied
- THEN an LLM judges if output is grounded in retrieved content
- AND returns grounded/not_grounded with rationale

#### Scenario: Judge retry on failure
- GIVEN an LLM API error during scoring
- WHEN the judge scorer encounters the error
- THEN it retries with exponential backoff
- AND returns error score if all retries fail

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
