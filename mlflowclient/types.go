package mlflowclient

import (
	"time"
)

// Trace represents a complete trace for API transport.
// This is the wire format for sending traces to MLflow.
type Trace struct {
	TraceID          string            `json:"trace_id,omitempty"`
	ClientRequestID  string            `json:"client_request_id,omitempty"`
	ExperimentID     string            `json:"experiment_id,omitempty"`
	RequestPreview   string            `json:"request_preview,omitempty"`
	ResponsePreview  string            `json:"response_preview,omitempty"`
	RequestTimeMs    int64             `json:"request_time,omitempty"`
	ExecutionTimeMs  int64             `json:"execution_duration,omitempty"`
	State            TraceState        `json:"state,omitempty"`
	TraceMetadata    map[string]string `json:"trace_metadata,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
	Spans            []*Span           `json:"spans,omitempty"`
}

// TraceState represents the execution state of a trace.
type TraceState string

const (
	TraceStateUnspecified TraceState = ""
	TraceStateOK          TraceState = "OK"
	TraceStateError       TraceState = "ERROR"
	TraceStateInProgress  TraceState = "IN_PROGRESS"
)

// Span represents a unit of work within a trace.
type Span struct {
	TraceID      string            `json:"trace_id,omitempty"`
	SpanID       string            `json:"span_id,omitempty"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name,omitempty"`
	SpanType     SpanType          `json:"span_kind,omitempty"`
	StartTimeNs  int64             `json:"start_time_unix_nano,omitempty"`
	EndTimeNs    int64             `json:"end_time_unix_nano,omitempty"`
	Status       *SpanStatus       `json:"status,omitempty"`
	Attributes   map[string]any    `json:"attributes,omitempty"`
	Events       []*SpanEvent      `json:"events,omitempty"`
}

// SpanType represents the type of span.
type SpanType string

const (
	SpanTypeAgent     SpanType = "AGENT"
	SpanTypeLLM       SpanType = "LLM"
	SpanTypeTool      SpanType = "TOOL"
	SpanTypeChain     SpanType = "CHAIN"
	SpanTypeRetriever SpanType = "RETRIEVER"
	SpanTypeEmbedding SpanType = "EMBEDDING"
	SpanTypeUnknown   SpanType = "UNKNOWN"
)

// SpanStatus represents the completion status of a span.
type SpanStatus struct {
	Code        SpanStatusCode `json:"code,omitempty"`
	Description string         `json:"message,omitempty"`
}

// SpanStatusCode is the status code for a span.
type SpanStatusCode int

const (
	SpanStatusUnset SpanStatusCode = 0
	SpanStatusOK    SpanStatusCode = 1
	SpanStatusError SpanStatusCode = 2
)

// SpanEvent represents an event that occurred during span execution.
type SpanEvent struct {
	Name       string         `json:"name,omitempty"`
	TimeNs     int64          `json:"time_unix_nano,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Experiment represents an MLflow experiment.
type Experiment struct {
	ExperimentID     string            `json:"experiment_id,omitempty"`
	Name             string            `json:"name,omitempty"`
	ArtifactLocation string            `json:"artifact_location,omitempty"`
	LifecycleStage   string            `json:"lifecycle_stage,omitempty"`
	LastUpdateTime   int64             `json:"last_update_time,omitempty"`
	CreationTime     int64             `json:"creation_time,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// Run represents a single MLflow run.
type Run struct {
	Info *RunInfo `json:"info,omitempty"`
	Data *RunData `json:"data,omitempty"`
}

// RunInfo contains metadata of a single run.
type RunInfo struct {
	RunID          string    `json:"run_id,omitempty"`
	RunName        string    `json:"run_name,omitempty"`
	ExperimentID   string    `json:"experiment_id,omitempty"`
	Status         RunStatus `json:"status,omitempty"`
	StartTime      int64     `json:"start_time,omitempty"`
	EndTime        int64     `json:"end_time,omitempty"`
	ArtifactURI    string    `json:"artifact_uri,omitempty"`
	LifecycleStage string    `json:"lifecycle_stage,omitempty"`
}

// RunStatus represents the status of a run.
type RunStatus string

const (
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusScheduled RunStatus = "SCHEDULED"
	RunStatusFinished  RunStatus = "FINISHED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusKilled    RunStatus = "KILLED"
)

// RunData contains run metrics, params, and tags.
type RunData struct {
	Metrics []*Metric `json:"metrics,omitempty"`
	Params  []*Param  `json:"params,omitempty"`
	Tags    []*RunTag `json:"tags,omitempty"`
}

// Metric associated with a run.
type Metric struct {
	Key       string  `json:"key,omitempty"`
	Value     float64 `json:"value,omitempty"`
	Timestamp int64   `json:"timestamp,omitempty"`
	Step      int64   `json:"step,omitempty"`
}

// Param associated with a run.
type Param struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// RunTag is a tag for a run.
type RunTag struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// Assessment represents feedback or expectation attached to a trace.
type Assessment struct {
	AssessmentID string            `json:"assessment_id,omitempty"`
	Name         string            `json:"assessment_name,omitempty"`
	Source       *AssessmentSource `json:"source,omitempty"`
	TraceID      string            `json:"trace_id,omitempty"`
	SpanID       string            `json:"span_id,omitempty"`
	Rationale    string            `json:"rationale,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Feedback     *FeedbackValue    `json:"feedback,omitempty"`
	Expectation  *ExpectationValue `json:"expectation,omitempty"`
	CreateTime   time.Time         `json:"create_time,omitempty"`
	UpdateTime   time.Time         `json:"last_update_time,omitempty"`
}

// AssessmentSource identifies who/what created the assessment.
type AssessmentSource struct {
	SourceType string `json:"source_type,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
}

// Assessment source types.
const (
	SourceTypeCode     = "CODE"
	SourceTypeHuman    = "HUMAN"
	SourceTypeLLMJudge = "LLM_JUDGE"
)

// FeedbackValue contains the actual feedback score.
type FeedbackValue struct {
	Value any              `json:"value,omitempty"`
	Error *AssessmentError `json:"error,omitempty"`
}

// ExpectationValue contains the expected/ground truth value.
type ExpectationValue struct {
	Value any `json:"value,omitempty"`
}

// AssessmentError captures scorer failure information.
type AssessmentError struct {
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	StackTrace   string `json:"stack_trace,omitempty"`
}

// SerializedScorer represents a scorer definition for registration.
type SerializedScorer struct {
	Type        string         `json:"scorer_type,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
}

// ScorerVersion represents a specific version of a scorer.
type ScorerVersion struct {
	Version    int64             `json:"version,omitempty"`
	Scorer     *SerializedScorer `json:"scorer,omitempty"`
	CreateTime time.Time         `json:"create_time,omitempty"`
}
