package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"charm.land/fantasy/mlflowclient"
)

// Tracer manages trace lifecycle and span creation.
type Tracer struct {
	client       *mlflowclient.Client
	experimentID string
	flushTimeout time.Duration

	mu      sync.Mutex
	traces  map[string]*Trace
	pending []*Trace

	// Track active spans for orphan cleanup
	activeSpans map[string]*Span
}

// NewTracer creates a new Tracer instance.
func NewTracer(client *mlflowclient.Client, experimentID string, opts ...TracerOption) *Tracer {
	t := &Tracer{
		client:       client,
		experimentID: experimentID,
		flushTimeout: 10 * time.Second,
		traces:       make(map[string]*Trace),
		activeSpans:  make(map[string]*Span),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// TracerOption configures a Tracer.
type TracerOption func(*Tracer)

// WithFlushTimeout sets the timeout for flushing traces.
func WithFlushTimeout(d time.Duration) TracerOption {
	return func(t *Tracer) {
		t.flushTimeout = d
	}
}

// NewTrace creates a new trace and returns a context with the trace embedded.
func (t *Tracer) NewTrace(ctx context.Context, config TracingConfig) (context.Context, *Trace, error) {
	traceID := generateID()

	trace := &Trace{
		TraceID:       traceID,
		ExperimentID:  config.ExperimentID,
		RequestTimeMs: time.Now().UnixMilli(),
		State:         TraceStateInProgress,
		TraceMetadata: make(map[string]string),
		Tags:          make(map[string]string),
		Spans:         make([]*Span, 0),
	}

	// Apply config
	if config.ExperimentID == "" {
		trace.ExperimentID = t.experimentID
	}
	if config.Tags != nil {
		for k, v := range config.Tags {
			trace.Tags[k] = v
		}
	}
	if config.AgentName != "" {
		trace.TraceMetadata["agent_name"] = config.AgentName
	}
	if config.ModelName != "" {
		trace.TraceMetadata["model_name"] = config.ModelName
	}
	if config.SessionID != "" {
		trace.TraceMetadata["session_id"] = config.SessionID
	}

	t.mu.Lock()
	t.traces[traceID] = trace
	t.mu.Unlock()

	ctx = withTrace(ctx, trace)
	return ctx, trace, nil
}

// StartSpan creates a new span within a trace.
func (t *Tracer) StartSpan(ctx context.Context, name string, spanType SpanType) (*Span, context.Context) {
	// Check for context cancellation
	if ctx.Err() != nil {
		return nil, ctx
	}

	trace := TraceFromContext(ctx)
	if trace == nil {
		return nil, ctx
	}

	parentSpan := SpanFromContext(ctx)
	parentID := ""
	if parentSpan != nil {
		parentID = parentSpan.SpanID
	}

	span := &Span{
		TraceID:     trace.TraceID,
		SpanID:      generateID(),
		ParentID:    parentID,
		Name:        name,
		SpanType:    spanType,
		StartTimeNs: time.Now().UnixNano(),
		Attributes:  make(map[string]any),
		Events:      make([]SpanEvent, 0),
	}

	trace.AddSpan(span)

	// If this is the first span, set it as the root
	if trace.RootSpan() == nil {
		trace.SetRootSpan(span)
	}

	// Track active span
	t.mu.Lock()
	t.activeSpans[span.SpanID] = span
	t.mu.Unlock()

	ctx = withSpan(ctx, span)
	return span, ctx
}

// EndSpan marks a span as ended and removes it from active tracking.
func (t *Tracer) EndSpan(span *Span) {
	if span == nil {
		return
	}
	span.End()
	t.mu.Lock()
	delete(t.activeSpans, span.SpanID)
	t.mu.Unlock()
}

// EndTrace marks a trace as complete and prepares it for flushing.
func (t *Tracer) EndTrace(ctx context.Context, state TraceState) error {
	trace := TraceFromContext(ctx)
	if trace == nil {
		return nil
	}

	// Clean up orphan spans for this trace
	t.cleanupOrphanSpans(trace.TraceID)

	trace.mu.Lock()
	trace.State = state
	if trace.rootSpan != nil && trace.rootSpan.EndTimeNs > 0 {
		trace.ExecutionDuration = trace.rootSpan.EndTimeNs - trace.rootSpan.StartTimeNs
	}
	trace.mu.Unlock()

	// Add to pending for flush
	t.mu.Lock()
	t.pending = append(t.pending, trace)
	delete(t.traces, trace.TraceID)
	t.mu.Unlock()

	return nil
}

// cleanupOrphanSpans ends any spans that weren't properly closed for a trace.
func (t *Tracer) cleanupOrphanSpans(traceID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for spanID, span := range t.activeSpans {
		if span.TraceID == traceID && !span.IsEnded() {
			span.EndWithStatus(SpanStatus{
				Code:        SpanStatusError,
				Description: "span orphaned - not properly closed",
			})
			delete(t.activeSpans, spanID)
		}
	}
}

// Flush sends a single trace to MLflow.
func (t *Tracer) Flush(ctx context.Context, trace *Trace) error {
	if t.client == nil {
		return nil
	}

	// Convert to mlflowclient.Trace
	mlTrace := t.convertTrace(trace)

	ctx, cancel := context.WithTimeout(ctx, t.flushTimeout)
	defer cancel()

	_, err := t.client.StartTrace(ctx, mlTrace)
	return err
}

// FlushAll flushes all pending traces.
func (t *Tracer) FlushAll(ctx context.Context) error {
	t.mu.Lock()
	pending := t.pending
	t.pending = nil
	t.mu.Unlock()

	var lastErr error
	for _, trace := range pending {
		if err := t.Flush(ctx, trace); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// convertTrace converts internal Trace to mlflowclient.Trace for wire transport.
func (t *Tracer) convertTrace(trace *Trace) *mlflowclient.Trace {
	mlTrace := &mlflowclient.Trace{
		TraceID:         trace.TraceID,
		ExperimentID:    trace.ExperimentID,
		RequestTimeMs:   trace.RequestTimeMs,
		ExecutionTimeMs: trace.ExecutionDuration / 1_000_000, // ns to ms
		State:           convertTraceState(trace.State),
		RequestPreview:  trace.RequestPreview,
		ResponsePreview: trace.ResponsePreview,
		TraceMetadata:   trace.TraceMetadata,
		Tags:            trace.Tags,
		Spans:           make([]*mlflowclient.Span, 0, len(trace.Spans)),
	}

	for _, span := range trace.Spans {
		mlTrace.Spans = append(mlTrace.Spans, t.convertSpan(span))
	}

	return mlTrace
}

// convertSpan converts internal Span to mlflowclient.Span for wire transport.
func (t *Tracer) convertSpan(span *Span) *mlflowclient.Span {
	mlSpan := &mlflowclient.Span{
		TraceID:      span.TraceID,
		SpanID:       span.SpanID,
		ParentSpanID: span.ParentID,
		Name:         span.Name,
		SpanType:     convertSpanType(span.SpanType),
		StartTimeNs:  span.StartTimeNs,
		EndTimeNs:    span.EndTimeNs,
		Status: &mlflowclient.SpanStatus{
			Code:        convertSpanStatusCode(span.Status.Code),
			Description: span.Status.Description,
		},
		Attributes: span.Attributes,
		Events:     make([]*mlflowclient.SpanEvent, 0, len(span.Events)),
	}

	for _, event := range span.Events {
		mlSpan.Events = append(mlSpan.Events, &mlflowclient.SpanEvent{
			Name:       event.Name,
			TimeNs:     event.Timestamp,
			Attributes: event.Attributes,
		})
	}

	return mlSpan
}

func convertTraceState(state TraceState) mlflowclient.TraceState {
	switch state {
	case TraceStateOK:
		return mlflowclient.TraceStateOK
	case TraceStateError:
		return mlflowclient.TraceStateError
	case TraceStateInProgress:
		return mlflowclient.TraceStateInProgress
	default:
		return mlflowclient.TraceStateUnspecified
	}
}

func convertSpanType(st SpanType) mlflowclient.SpanType {
	switch st {
	case SpanTypeAgent:
		return mlflowclient.SpanTypeAgent
	case SpanTypeLLM:
		return mlflowclient.SpanTypeLLM
	case SpanTypeTool:
		return mlflowclient.SpanTypeTool
	case SpanTypeChain:
		return mlflowclient.SpanTypeChain
	case SpanTypeRetriever:
		return mlflowclient.SpanTypeRetriever
	case SpanTypeEmbedding:
		return mlflowclient.SpanTypeEmbedding
	default:
		return mlflowclient.SpanTypeUnknown
	}
}

func convertSpanStatusCode(code SpanStatusCode) mlflowclient.SpanStatusCode {
	switch code {
	case SpanStatusOK:
		return mlflowclient.SpanStatusOK
	case SpanStatusError:
		return mlflowclient.SpanStatusError
	default:
		return mlflowclient.SpanStatusUnset
	}
}

// generateID generates a random 16-byte hex ID.
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}
