## ADDED Requirements

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

The system SHALL provide a REST client for MLflow API v3 operations.

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

### Requirement: Run Management

The system SHALL support MLflow run CRUD and logging operations.

#### Scenario: Create run
- GIVEN an experiment ID
- WHEN `client.CreateRun(ctx, experimentID)` is called
- THEN a POST request is made to `/api/2.0/mlflow/runs/create`
- AND the run metadata is returned

#### Scenario: Log batch
- GIVEN a run ID, metrics, params, and tags
- WHEN `client.LogBatch(ctx, runID, metrics, params, tags)` is called
- THEN a POST request is made to `/api/2.0/mlflow/runs/log-batch`
- AND metrics, params, and tags are logged atomically

#### Scenario: Update run status
- GIVEN a run ID and new status
- WHEN `client.UpdateRun(ctx, runID, status, endTime)` is called
- THEN a POST request is made to `/api/2.0/mlflow/runs/update`
- AND the run status is updated

### Requirement: Trace Management

The system SHALL support MLflow trace API v3 operations.

#### Scenario: Start trace
- GIVEN trace info with experiment location
- WHEN `client.StartTrace(ctx, trace)` is called
- THEN a POST request is made to `/api/3.0/mlflow/traces`
- AND the trace is created with the provided spans

#### Scenario: Get trace
- GIVEN a trace ID
- WHEN `client.GetTrace(ctx, traceID, allowPartial)` is called
- THEN a GET request is made to `/api/3.0/mlflow/traces/get`
- AND the full trace with spans is returned

#### Scenario: Search traces
- GIVEN locations, filter, and pagination options
- WHEN `client.SearchTraces(ctx, opts)` is called
- THEN a POST request is made to `/api/3.0/mlflow/traces/search`
- AND matching trace infos are returned

#### Scenario: Delete traces
- GIVEN an experiment ID and deletion criteria
- WHEN `client.DeleteTraces(ctx, experimentID, opts)` is called
- THEN a POST request is made to `/api/3.0/mlflow/traces/delete-traces`
- AND the count of deleted traces is returned

#### Scenario: Set trace tag
- GIVEN a trace ID, key, and value
- WHEN `client.SetTraceTag(ctx, traceID, key, value)` is called
- THEN a PATCH request is made to `/api/3.0/mlflow/traces/{trace_id}/tags`
- AND the tag is set on the trace

### Requirement: Assessment Management

The system SHALL support MLflow assessment API operations.

#### Scenario: Create assessment
- GIVEN a trace ID and assessment with feedback
- WHEN `client.CreateAssessment(ctx, traceID, assessment)` is called
- THEN a POST request is made to `/api/3.0/mlflow/traces/{trace_id}/assessments`
- AND the assessment is created with an ID

#### Scenario: Update assessment
- GIVEN a trace ID, assessment ID, and update mask
- WHEN `client.UpdateAssessment(ctx, traceID, assessmentID, assessment, mask)` is called
- THEN a PATCH request is made to the assessment endpoint
- AND only specified fields are updated

#### Scenario: Delete assessment
- GIVEN a trace ID and assessment ID
- WHEN `client.DeleteAssessment(ctx, traceID, assessmentID)` is called
- THEN a DELETE request is made to the assessment endpoint
- AND the assessment is removed

### Requirement: Scorer Registration

The system SHALL support MLflow scorer registration API.

#### Scenario: Register scorer
- GIVEN an experiment ID, scorer name, and serialized scorer
- WHEN `client.RegisterScorer(ctx, experimentID, name, serialized)` is called
- THEN a POST request is made to `/api/3.0/mlflow/scorers/register`
- AND a new version is created with scorer ID

#### Scenario: List scorers
- GIVEN an experiment ID
- WHEN `client.ListScorers(ctx, experimentID)` is called
- THEN a GET request is made to `/api/3.0/mlflow/scorers/list`
- AND latest versions of all scorers are returned

#### Scenario: Get scorer
- GIVEN an experiment ID, scorer name, and optional version
- WHEN `client.GetScorer(ctx, experimentID, name, version)` is called
- THEN a GET request is made to `/api/3.0/mlflow/scorers/get`
- AND the scorer definition is returned
