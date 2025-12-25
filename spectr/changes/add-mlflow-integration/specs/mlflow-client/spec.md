## ADDED Requirements

### API Version Strategy

The client uses two API versions:
- `/api/2.0/mlflow/` - Legacy endpoints for experiments, runs, and metrics (stable)
- `/api/3.0/mlflow/` - Modern endpoints for traces, assessments, and scorers (v3.8+)

Both versions use the same authentication and base URL. Version selection is automatic based on the operation.

### Authentication

The client SHALL support Bearer token authentication.

#### Scenario: Bearer token authentication
- GIVEN a client configured with `WithToken(token)` option
- WHEN any API request is made
- THEN the `Authorization: Bearer <token>` header is included
- AND if no token is configured, no Authorization header is sent

#### Scenario: Authentication error handling
- GIVEN an API response with status code 401 (Unauthorized)
- WHEN the response is processed
- THEN an APIError is returned with IsUnauthorized() returning true
- AND the error message indicates authentication failure

- GIVEN an API response with status code 403 (Forbidden)
- WHEN the response is processed
- THEN an APIError is returned with IsForbidden() returning true

### Timeout and Retry Configuration

The client SHALL support configurable timeouts and automatic retries.

#### Scenario: Default timeout
- GIVEN a client created without explicit timeout configuration
- WHEN an API request is made
- THEN the request times out after 30 seconds by default
- AND context cancellation is respected

#### Scenario: Custom timeout
- GIVEN a client configured with `WithTimeout(duration)` option
- WHEN an API request is made
- THEN the request uses the specified timeout duration

#### Scenario: Retry configuration
- GIVEN a client configured with retry options
- WHEN a retryable error occurs (5xx status codes, connection errors)
- THEN the client retries with the following default configuration:
  - Maximum retries: 3
  - Initial delay: 1 second
  - Backoff multiplier: 2x (exponential backoff)
  - Maximum delay: 10 seconds
  - Jitter: ±10% randomization to prevent thundering herd
- AND retries are NOT triggered for client errors (4xx except 429)
- AND rate limit errors (429) trigger retry with Retry-After header if present

#### Scenario: Retry disabled
- GIVEN a client configured with `WithRetries(0)` option
- WHEN an error occurs
- THEN no retries are attempted

### Requirement: Proto-based Code Generation

The system SHALL generate Go structs from MLflow proto files using the Buf toolchain.

#### Scenario: Proto generation workflow
- GIVEN the `proto/mlflow/` directory contains curated MLflow protos
- WHEN `task proto:gen` is executed
- THEN Go code is generated to `gen/mlflow/` package
- AND all generated types compile successfully

#### Scenario: OpenTelemetry span types
- GIVEN OpenTelemetry protos are present in `proto/opentelemetry/proto/`
- WHEN proto generation runs
- THEN OTel Span types are available for trace construction

#### Scenario: ScalaPB extension compatibility
- GIVEN MLflow protos contain ScalaPB extension references
- WHEN proto generation runs
- THEN extensions are stubbed and do not cause compilation errors

### Requirement: MLflow REST Client

The system SHALL provide a REST client for MLflow API operations (v2 and v3 endpoints).

#### Scenario: Client initialization
- GIVEN a valid MLflow server URL
- WHEN a new client is created with `mlflowclient.New(baseURL)`
- THEN the client is configured for HTTP requests
- AND custom HTTP clients can be provided via options

#### Scenario: Request serialization
- GIVEN a proto message request
- WHEN the client makes an API call
- THEN the request is serialized using protojson
- AND the Content-Type header is set to `application/json`

#### Scenario: Response deserialization
- GIVEN an API response with JSON body
- WHEN the response is processed
- THEN the body is deserialized into the proto response type
- AND unknown fields are discarded without error

#### Scenario: Error handling
- GIVEN an API response with status code >= 400
- WHEN the response is processed
- THEN an APIError is returned with status code and body

#### Scenario: APIError type definition
- GIVEN the mlflowclient package
- WHEN APIError is defined
- THEN it has the following structure:
```go
// APIError represents an error response from the MLflow API.
type APIError struct {
    StatusCode int    // HTTP status code (e.g., 404, 500)
    Message    string // Error message from response body
    ErrorCode  string // MLflow error code (e.g., "RESOURCE_DOES_NOT_EXIST")
    Cause      error  // Underlying error (for error wrapping)
}

func (e *APIError) Error() string       // Implements error interface
func (e *APIError) Unwrap() error       // Returns Cause for errors.Is/As support
func (e *APIError) IsNotFound() bool    // Returns true for 404 status codes
func (e *APIError) IsConflict() bool    // Returns true for 409 status codes
func (e *APIError) IsServerError() bool // Returns true for 5xx status codes
func (e *APIError) IsUnauthorized() bool // Returns true for 401 status codes
func (e *APIError) IsForbidden() bool   // Returns true for 403 status codes
func (e *APIError) IsRateLimited() bool // Returns true for 429 status codes
func (e *APIError) IsRetryable() bool   // Returns true for 429, 5xx, or connection errors
```

#### Scenario: ValidationError type definition
- GIVEN the mlflowclient package
- WHEN ValidationError is defined
- THEN it has the following structure:
```go
// ValidationError represents a client-side validation failure.
type ValidationError struct {
    Field   string // Field that failed validation (e.g., "experimentID", "traceID")
    Message string // Description of the validation failure
    Value   any    // The invalid value (for debugging)
}

func (e *ValidationError) Error() string // Implements error interface
```
- AND it is returned for:
  - Empty required fields (experimentID, traceID, runID)
  - Invalid format (malformed trace ID, invalid timestamp)
  - Invalid option combinations (e.g., DeleteTracesOptions with no criteria)

#### Scenario: TimeoutError type definition
- GIVEN the mlflowclient package
- WHEN TimeoutError is defined
- THEN it has the following structure:
```go
// TimeoutError represents a request that exceeded its deadline.
type TimeoutError struct {
    Operation string        // Operation that timed out (e.g., "StartTrace", "GetRun")
    Timeout   time.Duration // The timeout that was exceeded
    Cause     error         // Underlying context.DeadlineExceeded or similar
}

func (e *TimeoutError) Error() string // Implements error interface
func (e *TimeoutError) Unwrap() error // Returns Cause for errors.Is/As support
```

#### Scenario: ConnectionError type definition
- GIVEN the mlflowclient package
- WHEN ConnectionError is defined
- THEN it has the following structure:
```go
// ConnectionError represents a network-level failure.
type ConnectionError struct {
    URL     string // The URL that failed to connect
    Message string // Description of the connection failure
    Cause   error  // Underlying net error
}

func (e *ConnectionError) Error() string   // Implements error interface
func (e *ConnectionError) Unwrap() error   // Returns Cause for errors.Is/As support
func (e *ConnectionError) IsRetryable() bool // Returns true (connection errors are retryable)
```

#### Scenario: Error wrapping and inspection
- GIVEN any mlflowclient error type
- WHEN errors.Is() or errors.As() is used
- THEN the error chain can be inspected via Unwrap()
- AND type assertions work correctly:
```go
var apiErr *APIError
if errors.As(err, &apiErr) {
    if apiErr.IsRetryable() {
        // Handle retryable error
    }
}

var validationErr *ValidationError
if errors.As(err, &validationErr) {
    log.Printf("Invalid %s: %v", validationErr.Field, validationErr.Value)
}
```

### Requirement: Experiment Management

The system SHALL support MLflow experiment CRUD operations.

#### Scenario: Create experiment
- GIVEN an experiment name
- WHEN `client.CreateExperiment(ctx, name)` is called
- THEN a POST request is made to `/api/2.0/mlflow/experiments/create`
- AND the experiment ID is returned

#### Scenario: Get experiment
- GIVEN an experiment ID
- WHEN `client.GetExperiment(ctx, experimentID)` is called
- THEN a GET request is made to `/api/2.0/mlflow/experiments/get`
- AND experiment metadata is returned

#### Scenario: Search experiments
- GIVEN a filter string and max results
- WHEN `client.SearchExperiments(ctx, opts)` is called
- THEN a POST request is made to `/api/2.0/mlflow/experiments/search`
- AND matching experiments are returned with pagination token

#### Scenario: SearchExperimentsOptions structure
- GIVEN the mlflowclient package
- WHEN SearchExperimentsOptions is defined
- THEN it has the following structure:
```go
type SearchExperimentsOptions struct {
    Filter    string // Filter string (e.g., "name LIKE 'my-%'")
    MaxResults int   // Maximum results to return (default: 1000, max: 50000)
    PageToken string // Token for pagination continuation
    OrderBy   []string // Order by columns (e.g., ["name ASC", "creation_time DESC"])
    ViewType  string // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL" (default: "ACTIVE_ONLY")
}
```

### Requirement: Filter String Syntax

All search operations SHALL use a common filter string syntax.

#### Scenario: Filter string syntax
- GIVEN a search operation that accepts a filter parameter
- WHEN a filter string is provided
- THEN it follows MLflow's filter syntax:
  - Attribute comparison: `attribute_name = 'value'` or `attribute_name != 'value'`
  - Numeric comparison: `metrics.accuracy > 0.9` or `metrics.loss < 0.1`
  - LIKE pattern: `name LIKE 'prefix-%'` (% is wildcard)
  - ILIKE for case-insensitive: `name ILIKE 'test%'`
  - AND/OR combinations: `metrics.accuracy > 0.9 AND status = 'FINISHED'`
  - IN lists: `attribute.key IN ('value1', 'value2')`
- AND supported attribute types vary by entity:
  - Experiments: `name`, `creation_time`, `lifecycle_stage`
  - Runs: `metrics.<key>`, `params.<key>`, `tags.<key>`, `attributes.<key>` (run_name, status, etc.)
  - Traces: `trace_id`, `state`, `request_time`, `tags.<key>`, `trace_metadata.<key>`
- AND invalid filter syntax returns a ValidationError

### Requirement: Pagination

All search operations SHALL support cursor-based pagination.

#### Scenario: Pagination behavior
- GIVEN a search operation that returns paginated results
- WHEN results are returned
- THEN the response includes:
  - `Items` - the current page of results
  - `NextPageToken` - token for the next page (empty string if no more pages)
- AND pagination works as follows:
  - First request: omit PageToken (or empty string)
  - Subsequent requests: pass the NextPageToken from previous response
  - Last page: NextPageToken is empty string
- AND page tokens are opaque strings (not user-parseable)
- AND page tokens expire after 1 hour
- AND using an expired token returns an error (not a partial result)

### Requirement: Run Management

The system SHALL support MLflow run CRUD and logging operations.

#### Scenario: Create run
- GIVEN an experiment ID and optional run configuration
- WHEN `client.CreateRun(ctx, experimentID, opts...)` is called
- THEN a POST request is made to `/api/2.0/mlflow/runs/create`
- AND the run metadata is returned
- AND optional parameters can be passed via functional options:
  - `WithRunName(name string)` - set run name
  - `WithStartTime(t int64)` - set start time (milliseconds since epoch)
  - `WithTags(tags map[string]string)` - set initial tags

#### Scenario: Log batch
- GIVEN a run ID, metrics, params, and tags
- WHEN `client.LogBatch(ctx, runID, metrics, params, tags)` is called
- THEN a POST request is made to `/api/2.0/mlflow/runs/log-batch`
- AND metrics, params, and tags are logged atomically

#### Scenario: Update run status
- GIVEN a run ID and new status
- WHEN `client.UpdateRun(ctx, runID, status, endTime)` is called
- THEN a POST request is made to `/api/2.0/mlflow/runs/update`
- AND the updated RunInfo is returned
- AND status must be one of: "RUNNING", "SCHEDULED", "FINISHED", "FAILED", "KILLED"
- AND endTime is optional (pass 0 to not update)

#### Scenario: Get run
- GIVEN a run ID
- WHEN `client.GetRun(ctx, runID)` is called
- THEN a GET request is made to `/api/2.0/mlflow/runs/get`
- AND the Run object is returned containing RunInfo and RunData
- AND RunData includes metrics, params, and tags

#### Scenario: Search runs
- GIVEN experiment IDs and optional filter criteria
- WHEN `client.SearchRuns(ctx, opts)` is called
- THEN a POST request is made to `/api/2.0/mlflow/runs/search`
- AND matching runs are returned with pagination token

#### Scenario: SearchRunsOptions structure
- GIVEN the mlflowclient package
- WHEN SearchRunsOptions is defined
- THEN it has the following structure:
```go
type SearchRunsOptions struct {
    ExperimentIDs []string   // Experiment IDs to search within (required)
    Filter        string     // Filter string (e.g., "metrics.accuracy > 0.9")
    RunViewType   string     // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL" (default: "ACTIVE_ONLY")
    MaxResults    int        // Maximum results to return (default: 1000, max: 50000)
    OrderBy       []string   // Order by columns (e.g., ["metrics.accuracy DESC"])
    PageToken     string     // Token for pagination continuation
}
```

#### Scenario: LogBatch with empty inputs
- GIVEN a LogBatch call with nil or empty slices for metrics, params, or tags
- WHEN `client.LogBatch(ctx, runID, nil, nil, nil)` is called
- THEN the request is still sent with empty arrays
- AND no error is returned (no-op is valid)

### Requirement: Trace Management

The system SHALL support MLflow trace API v3 operations.

#### Scenario: Trace type for API operations
- GIVEN the mlflowclient package needs to send traces to MLflow
- WHEN the client API trace types are defined
- THEN `mlflowclient.Trace` wraps the generated `gen/mlflow.TraceInfoV3` proto type for API transport
- AND conversion functions exist to convert `*tracing.Trace` to `*mlflowclient.Trace`:
```go
// ConvertTrace converts a tracing.Trace to the API wire format.
// This is used when sending traces to the MLflow server.
func ConvertTrace(t *tracing.Trace) *Trace
```
- Note: The `tracing.Trace` type (defined in tracing/spec.md) is used for trace construction,
  while `mlflowclient.Trace` is used for API transport. They serve different purposes.

#### Scenario: Start trace
- GIVEN a Trace with experiment location
- WHEN `client.StartTrace(ctx, trace *Trace)` is called
- THEN a POST request is made to `/api/3.0/mlflow/traces`
- AND the trace is created with the provided spans

#### Scenario: Get trace
- GIVEN a trace ID
- WHEN `client.GetTrace(ctx, traceID, allowPartial)` is called
- THEN a GET request is made to `/api/3.0/mlflow/traces/get`
- AND the full trace with spans is returned
- AND `allowPartial` (bool) determines behavior for incomplete traces:
  - If true: return partial trace data if trace is still in progress
  - If false: return error if trace is incomplete

#### Scenario: Search traces
- GIVEN locations, filter, and pagination options
- WHEN `client.SearchTraces(ctx, opts)` is called
- THEN a POST request is made to `/api/3.0/mlflow/traces/search`
- AND matching trace infos are returned

#### Scenario: SearchTracesOptions structure
- GIVEN the mlflowclient package
- WHEN SearchTracesOptions is defined
- THEN it has the following structure:
```go
type SearchTracesOptions struct {
    ExperimentIDs []string // Experiment IDs to search within
    Filter        string   // Filter string (e.g., "status = 'OK'", "tags.env = 'prod'")
    MaxResults    int      // Maximum results to return (default: 100, max: 1000)
    PageToken     string   // Token for pagination continuation
    OrderBy       []string // Order by columns (e.g., ["timestamp_ms DESC"])
}
```

#### Scenario: Delete traces
- GIVEN an experiment ID and deletion criteria
- WHEN `client.DeleteTraces(ctx, experimentID, opts)` is called
- THEN a POST request is made to `/api/3.0/mlflow/traces/delete-traces`
- AND the count of deleted traces is returned

#### Scenario: DeleteTracesOptions structure
- GIVEN the mlflowclient package
- WHEN DeleteTracesOptions is defined
- THEN it has the following structure:
```go
type DeleteTracesOptions struct {
    // At least one of these must be specified:
    MaxTraces       int    // Maximum number of traces to delete
    MaxTimestampMs  int64  // Delete traces older than this timestamp (milliseconds since epoch)
    Filter          string // Filter string to select traces for deletion
}
```
- AND if none of MaxTraces, MaxTimestampMs, or Filter is specified, an error is returned

#### Scenario: Set trace tag
- GIVEN a trace ID, key, and value
- WHEN `client.SetTraceTag(ctx, traceID, key, value)` is called
- THEN a PATCH request is made to `/api/3.0/mlflow/traces/{trace_id}/tags`
- AND the tag is set on the trace

#### Scenario: Delete trace tag
- GIVEN a trace ID and tag key
- WHEN `client.DeleteTraceTag(ctx, traceID, key)` is called
- THEN a DELETE request is made to `/api/3.0/mlflow/traces/{trace_id}/tags/{key}`
- AND the tag is removed from the trace

#### Scenario: Set run tag
- GIVEN a run ID, key, and value
- WHEN `client.SetRunTag(ctx, runID, key, value)` is called
- THEN a POST request is made to `/api/2.0/mlflow/runs/set-tag`
- AND the tag is set on the run

### Requirement: Assessment Management

The system SHALL support MLflow assessment API operations.

#### Scenario: Assessment type definition
- GIVEN the mlflowclient package
- WHEN Assessment is defined
- THEN it has the following structure:
```go
// Assessment represents feedback or expectation attached to a trace.
type Assessment struct {
    Name       string            // Assessment name (e.g., "correctness", "relevance")
    Source     AssessmentSource  // Source of the assessment (code, human, LLM judge)
    TraceID    string            // Associated trace ID (optional at creation)
    SpanID     string            // Associated span ID (optional, for span-level assessments)
    Rationale  string            // Explanation for the assessment
    Metadata   map[string]string // Additional metadata

    // Exactly one of Feedback or Expectation must be set:
    Feedback    *FeedbackValue    // Feedback value (score result)
    Expectation *ExpectationValue // Expected value (ground truth)
}

// AssessmentSource identifies who/what created the assessment.
type AssessmentSource struct {
    SourceType string // "CODE", "HUMAN", or "LLM_JUDGE"
    SourceID   string // Identifier (e.g., scorer name, user email, model name)
}

// FeedbackValue contains the actual feedback score.
type FeedbackValue struct {
    Value any                // bool, float64, int, string, or structured value
    Error *AssessmentError   // Error if scoring failed (optional)
}

// ExpectationValue contains the expected/ground truth value.
type ExpectationValue struct {
    Value any // Expected value (JSON-serializable)
}

// AssessmentError captures scorer failure information.
type AssessmentError struct {
    ErrorMessage string // Human-readable error message
    ErrorCode    string // Error code (e.g., "SCORER_TIMEOUT", "LLM_FAILURE")
    StackTrace   string // Optional stack trace
}
```

#### Scenario: Create assessment
- GIVEN a trace ID and assessment with feedback
- WHEN `client.CreateAssessment(ctx, traceID, assessment)` is called
- THEN a POST request is made to `/api/3.0/mlflow/traces/{trace_id}/assessments`
- AND the assessment is created with an ID
- AND the created Assessment with assessment_id is returned

#### Scenario: Update assessment
- GIVEN a trace ID, assessment ID, and update mask
- WHEN `client.UpdateAssessment(ctx, traceID, assessmentID, assessment, mask)` is called
- THEN a PATCH request is made to `/api/3.0/mlflow/traces/{trace_id}/assessments/{assessment_id}`
- AND only fields specified in mask are updated
- AND mask is a []string of field names (e.g., ["rationale", "feedback.value"])

#### Scenario: Delete assessment
- GIVEN a trace ID and assessment ID
- WHEN `client.DeleteAssessment(ctx, traceID, assessmentID)` is called
- THEN a DELETE request is made to `/api/3.0/mlflow/traces/{trace_id}/assessments/{assessment_id}`
- AND the assessment is removed

### Requirement: Scorer Registration

The system SHALL support MLflow scorer registration API.

#### Scenario: SerializedScorer structure
- GIVEN the mlflowclient package
- WHEN SerializedScorer is defined
- THEN it has the following structure:
```go
// SerializedScorer represents a scorer definition for registration.
type SerializedScorer struct {
    Type        string            // Scorer type: "CODE", "LLM_JUDGE", or "HEURISTIC"
    Name        string            // Scorer name (e.g., "correctness")
    Description string            // Human-readable description
    Config      map[string]any    // Scorer-specific configuration (JSON-serializable)
}
```
- AND Config contents depend on Type:
  - For "CODE": `{"function_name": "...", "package": "..."}`
  - For "LLM_JUDGE": `{"model": "...", "prompt_template": "...", "temperature": 0.0}`
  - For "HEURISTIC": `{"heuristic_type": "exact_match|contains|regex|json_match|numeric_range"}`

#### Scenario: Register scorer
- GIVEN an experiment ID, scorer name, and serialized scorer
- WHEN `client.RegisterScorer(ctx, experimentID, name, scorer SerializedScorer)` is called
- THEN a POST request is made to `/api/3.0/mlflow/scorers/register`
- AND a new version is created with scorer ID

#### Scenario: List scorers
- GIVEN an experiment ID
- WHEN `client.ListScorers(ctx, experimentID)` is called
- THEN a GET request is made to `/api/3.0/mlflow/scorers/list`
- AND latest versions of all scorers are returned

#### Scenario: Get scorer
- GIVEN an experiment ID, scorer name, and optional version
- WHEN `client.GetScorer(ctx, experimentID, name, version int)` is called
- THEN a GET request is made to `/api/3.0/mlflow/scorers/get`
- AND if version is 0, the latest version is returned
- AND the scorer definition is returned
- AND version is an int (0 for latest, positive integer for specific version)
