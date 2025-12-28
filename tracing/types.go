package tracing

import (
	"sync"
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
	Timestamp  int64 // Nanoseconds since epoch
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
}

// SetAttribute sets a span attribute in a thread-safe manner.
func (s *Span) SetAttribute(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Attributes == nil {
		s.Attributes = make(map[string]any)
	}
	s.Attributes[key] = value
}

// SetStatus sets the span status in a thread-safe manner.
func (s *Span) SetStatus(code SpanStatusCode, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = SpanStatus{
		Code:        code,
		Description: description,
	}
}

// End sets the span's end time to the current time in nanoseconds.
func (s *Span) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.EndTimeNs == 0 {
		s.EndTimeNs = nowNanos()
	}
}

// AddEvent adds an event to the span in a thread-safe manner.
func (s *Span) AddEvent(name string, attrs map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event := SpanEvent{
		Name:       name,
		Timestamp:  nowNanos(),
		Attributes: attrs,
	}
	s.Events = append(s.Events, event)
}

// TraceState represents the state of a trace.
type TraceState int

const (
	TraceStateInProgress TraceState = iota
	TraceStateOK
	TraceStateError
)

// String returns the string representation of TraceState.
func (s TraceState) String() string {
	switch s {
	case TraceStateInProgress:
		return "IN_PROGRESS"
	case TraceStateOK:
		return "OK"
	case TraceStateError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Trace represents a complete trace with metadata and spans.
type Trace struct {
	TraceID           string
	ExperimentID      string
	RequestTime       int64 // Milliseconds since epoch
	ExecutionDuration int64 // Milliseconds
	State             TraceState
	RequestPreview    string
	ResponsePreview   string
	TraceMetadata     map[string]string
	Tags              map[string]string
	Spans             []*Span
	mu                sync.RWMutex
}

// AddSpan adds a span to the trace in a thread-safe manner.
func (t *Trace) AddSpan(span *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Spans = append(t.Spans, span)
}

// GetSpan retrieves a span by ID in a thread-safe manner.
func (t *Trace) GetSpan(spanID string) *Span {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, span := range t.Spans {
		if span.SpanID == spanID {
			return span
		}
	}
	return nil
}

// SetState sets the trace state in a thread-safe manner.
func (t *Trace) SetState(state TraceState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.State = state
}

// TracingResult contains the result of a traced agent execution.
type TracingResult struct {
	Trace      *Trace
	FlushError error
}

// TokenUsage contains token usage information.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
