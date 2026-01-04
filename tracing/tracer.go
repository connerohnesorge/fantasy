package tracing

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// Tracer manages trace and span creation with parent-child relationships.
type Tracer struct {
	config    TracingConfig
	trace     *Trace
	spanStack []*Span // Stack for parent-child relationships
	mu        sync.Mutex
}

// NewTracer creates a new tracer with the given configuration.
func NewTracer(config TracingConfig) *Tracer {
	return &Tracer{
		config:    config,
		spanStack: make([]*Span, 0),
	}
}

// TraceIDPrefix is the prefix MLflow uses for trace IDs.
const TraceIDPrefix = "tr-"

// StartTrace creates a new trace with the given request.
func (t *Tracer) StartTrace(request string) *Trace {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Generate the raw OTel trace ID (16 bytes as hex = 32 chars)
	otelTraceID := generateID()
	// MLflow trace ID uses the "tr-" prefix
	traceID := TraceIDPrefix + otelTraceID

	t.trace = &Trace{
		TraceID:        traceID,
		OtelTraceID:    otelTraceID, // Store raw OTel trace ID for spans
		ExperimentID:   t.config.ExperimentID,
		RequestTime:    nowMillis(),
		State:          TraceStateInProgress,
		RequestPreview: TruncatePreview(request),
		TraceMetadata:  make(map[string]string),
		Tags:           make(map[string]string),
		Spans:          make([]*Span, 0),
	}

	// Copy custom tags
	if t.config.Tags != nil {
		for k, v := range t.config.Tags {
			t.trace.Tags[k] = v
		}
	}

	// Add standard metadata
	if t.config.SessionID != "" {
		t.trace.TraceMetadata["session_id"] = t.config.SessionID
	}
	if t.config.ModelName != "" {
		t.trace.TraceMetadata["model_name"] = t.config.ModelName
	}
	if t.config.AgentName != "" {
		t.trace.TraceMetadata["agent_name"] = t.config.AgentName
	}

	return t.trace
}

// StartSpan creates a new span with the given name and type.
// Spans are automatically linked in parent-child relationships based on the stack.
func (t *Tracer) StartSpan(name string, spanType SpanType) *Span {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.trace == nil {
		return nil
	}

	spanID := generateSpanID()
	var parentID string

	// Determine parent based on span stack
	if len(t.spanStack) > 0 {
		parentID = t.spanStack[len(t.spanStack)-1].SpanID
	}

	span := &Span{
		TraceID:     t.trace.OtelTraceID, // Use OTel trace ID for span linkage
		SpanID:      spanID,
		ParentID:    parentID,
		Name:        name,
		SpanType:    spanType,
		StartTimeNs: nowNanos(),
		Attributes:  make(map[string]any),
		Events:      make([]SpanEvent, 0),
	}

	// Set span type attribute
	span.SetAttribute(AttrSpanType, string(spanType))

	// Add to trace
	t.trace.AddSpan(span)

	// Push to stack
	t.spanStack = append(t.spanStack, span)

	return span
}

// EndSpan ends the given span and removes it from the stack.
func (t *Tracer) EndSpan(span *Span) {
	if span == nil {
		return
	}

	span.End()

	t.mu.Lock()
	defer t.mu.Unlock()

	// Remove from stack (find and remove the span)
	for i := len(t.spanStack) - 1; i >= 0; i-- {
		if t.spanStack[i].SpanID == span.SpanID {
			// Remove by slicing
			t.spanStack = append(t.spanStack[:i], t.spanStack[i+1:]...)
			break
		}
	}
}

// GetCurrentSpan returns the current (top of stack) span.
func (t *Tracer) GetCurrentSpan() *Span {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.spanStack) == 0 {
		return nil
	}
	return t.spanStack[len(t.spanStack)-1]
}

// GetTrace returns the current trace.
func (t *Tracer) GetTrace() *Trace {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.trace
}

// SetRequestPreview sets the request preview on the trace.
func (t *Tracer) SetRequestPreview(preview string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.trace != nil {
		t.trace.mu.Lock()
		t.trace.RequestPreview = TruncatePreview(preview)
		t.trace.mu.Unlock()
	}
}

// SetResponsePreview sets the response preview on the trace.
func (t *Tracer) SetResponsePreview(preview string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.trace != nil {
		t.trace.mu.Lock()
		t.trace.ResponsePreview = TruncatePreview(preview)
		t.trace.mu.Unlock()
	}
}

// EndOrphanSpans ends any spans that weren't properly closed.
func (t *Tracer) EndOrphanSpans() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.trace == nil {
		return
	}

	t.trace.mu.RLock()
	spans := t.trace.Spans
	t.trace.mu.RUnlock()

	for _, span := range spans {
		span.mu.Lock()
		if span.EndTimeNs == 0 {
			span.EndTimeNs = nowNanos()
		}
		span.mu.Unlock()
	}

	// Clear the stack
	t.spanStack = make([]*Span, 0)
}

// generateID generates a random 16-byte hex-encoded ID (for trace IDs).
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to pseudo-random if crypto/rand fails
		// In practice, this should never happen
		for i := range b {
			b[i] = byte(i)
		}
	}
	return hex.EncodeToString(b)
}

// generateSpanID generates a random 8-byte hex-encoded ID (for span IDs).
func generateSpanID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(i)
		}
	}
	return hex.EncodeToString(b)
}

// nowMillis returns the current time in milliseconds since epoch.
func nowMillis() int64 {
	return nowNanos() / 1_000_000
}
