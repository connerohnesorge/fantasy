package tracing

import (
	"sync"
	"time"
)

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

// SpanStatusCode is the status code for a span.
type SpanStatusCode int

const (
	SpanStatusUnset SpanStatusCode = iota
	SpanStatusOK
	SpanStatusError
)

// SpanStatus represents the completion status of a span.
type SpanStatus struct {
	Code        SpanStatusCode
	Description string
}

// SpanEvent represents an event that occurred during span execution.
type SpanEvent struct {
	Name       string
	Timestamp  int64
	Attributes map[string]any
}

// Span represents a unit of work within a trace.
type Span struct {
	TraceID     string
	SpanID      string
	ParentID    string
	Name        string
	SpanType    SpanType
	StartTimeNs int64
	EndTimeNs   int64
	Status      SpanStatus
	Attributes  map[string]any
	Events      []SpanEvent
	mu          sync.Mutex
	ended       bool
}

// SetAttribute sets an attribute on the span.
func (s *Span) SetAttribute(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Attributes == nil {
		s.Attributes = make(map[string]any)
	}
	s.Attributes[key] = value
}

// AddEvent adds an event to the span.
func (s *Span) AddEvent(name string, attrs map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now().UnixNano(),
		Attributes: attrs,
	})
}

// End ends the span with OK status.
func (s *Span) End() {
	s.EndWithStatus(SpanStatus{Code: SpanStatusOK})
}

// EndWithError ends the span with ERROR status.
func (s *Span) EndWithError(err error) {
	s.EndWithStatus(SpanStatus{
		Code:        SpanStatusError,
		Description: err.Error(),
	})
}

// EndWithStatus ends the span with the given status.
func (s *Span) EndWithStatus(status SpanStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.ended = true
	s.EndTimeNs = time.Now().UnixNano()
	s.Status = status
}

// IsEnded returns true if the span has been ended.
func (s *Span) IsEnded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ended
}

// Duration returns the span duration.
func (s *Span) Duration() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.EndTimeNs == 0 {
		return time.Duration(time.Now().UnixNano() - s.StartTimeNs)
	}
	return time.Duration(s.EndTimeNs - s.StartTimeNs)
}

// TraceState represents the state of a trace.
type TraceState int

const (
	TraceStateInProgress TraceState = iota
	TraceStateOK
	TraceStateError
)

func (s TraceState) String() string {
	switch s {
	case TraceStateOK:
		return "OK"
	case TraceStateError:
		return "ERROR"
	default:
		return "IN_PROGRESS"
	}
}

// Trace represents a complete trace with metadata and spans.
type Trace struct {
	TraceID           string
	ExperimentID      string
	RequestTimeMs     int64
	ExecutionDuration int64
	State             TraceState
	RequestPreview    string
	ResponsePreview   string
	TraceMetadata     map[string]string
	Tags              map[string]string
	Spans             []*Span
	mu                sync.RWMutex
	rootSpan          *Span
}

// AddSpan adds a span to the trace.
func (t *Trace) AddSpan(span *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Spans = append(t.Spans, span)
}

// RootSpan returns the root span of the trace.
func (t *Trace) RootSpan() *Span {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.rootSpan
}

// SetRootSpan sets the root span of the trace.
func (t *Trace) SetRootSpan(span *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rootSpan = span
}

// SetMetadata sets a metadata key-value pair.
func (t *Trace) SetMetadata(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.TraceMetadata == nil {
		t.TraceMetadata = make(map[string]string)
	}
	t.TraceMetadata[key] = value
}

// SetTag sets a tag on the trace.
func (t *Trace) SetTag(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Tags == nil {
		t.Tags = make(map[string]string)
	}
	t.Tags[key] = value
}

// TracingResult contains the result of a traced agent execution.
type TracingResult struct {
	Trace      *Trace
	FlushError error
}

// TracingConfig configures tracing behavior.
type TracingConfig struct {
	Client       any               // *mlflowclient.Client - uses any to avoid import cycle
	ExperimentID string            // Required: MLflow experiment ID
	AgentName    string            // Optional: Name for the agent span
	ModelName    string            // Optional: Model identifier
	SessionID    string            // Optional: Session ID (auto-generated if empty)
	Tags         map[string]string // Optional: Custom tags
	FlushTimeout time.Duration     // Optional: Timeout for trace upload (default: 10s)
}

// Span attribute key constants.
const (
	// Agent attributes
	AttrAgentName = "agent.name"
	AttrModelName = "model.name"
	AttrSessionID = "session.id"

	// Step attributes
	AttrStepNumber = "step.number"

	// LLM attributes
	AttrInputTokens     = "llm.input_tokens"
	AttrOutputTokens    = "llm.output_tokens"
	AttrTotalTokens     = "llm.total_tokens"
	AttrReasoningTokens = "llm.reasoning_tokens"

	// Tool attributes
	AttrToolName   = "tool.name"
	AttrToolCallID = "tool.call_id"

	// Generic attributes
	AttrInput  = "input"
	AttrOutput = "output"
	AttrError  = "error"
)
