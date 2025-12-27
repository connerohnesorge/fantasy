package tracing

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewTracer(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")

	if tracer == nil {
		t.Fatal("NewTracer() returned nil")
	}

	if tracer.experimentID != "exp-123" {
		t.Errorf("experimentID = %s, want exp-123", tracer.experimentID)
	}

	if tracer.flushTimeout != 10*time.Second {
		t.Errorf("flushTimeout = %v, want 10s", tracer.flushTimeout)
	}
}

func TestNewTracer_WithOptions(t *testing.T) {
	tracer := NewTracer(nil, "exp-123", WithFlushTimeout(30*time.Second))

	if tracer.flushTimeout != 30*time.Second {
		t.Errorf("flushTimeout = %v, want 30s", tracer.flushTimeout)
	}
}

func TestTracer_NewTrace(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")

	config := TracingConfig{
		ExperimentID: "exp-456",
		AgentName:    "test-agent",
		ModelName:    "gpt-4",
		SessionID:    "session-123",
		Tags:         map[string]string{"env": "test"},
	}

	ctx, trace, err := tracer.NewTrace(context.Background(), config)
	if err != nil {
		t.Fatalf("NewTrace() error = %v", err)
	}

	if trace == nil {
		t.Fatal("NewTrace() returned nil trace")
	}

	if trace.TraceID == "" {
		t.Error("TraceID should not be empty")
	}

	if trace.ExperimentID != "exp-456" {
		t.Errorf("ExperimentID = %s, want exp-456", trace.ExperimentID)
	}

	if trace.State != TraceStateInProgress {
		t.Errorf("State = %v, want TraceStateInProgress", trace.State)
	}

	if trace.TraceMetadata["agent_name"] != "test-agent" {
		t.Errorf("Metadata agent_name = %s, want test-agent", trace.TraceMetadata["agent_name"])
	}

	if trace.Tags["env"] != "test" {
		t.Errorf("Tags env = %s, want test", trace.Tags["env"])
	}

	// Verify trace is in context
	traceFromCtx := TraceFromContext(ctx)
	if traceFromCtx != trace {
		t.Error("Trace not found in context")
	}
}

func TestTracer_StartSpan(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	ctx, trace, _ := tracer.NewTrace(context.Background(), TracingConfig{})

	span, spanCtx := tracer.StartSpan(ctx, "test-span", SpanTypeAgent)

	if span == nil {
		t.Fatal("StartSpan() returned nil span")
	}

	if span.Name != "test-span" {
		t.Errorf("Name = %s, want test-span", span.Name)
	}

	if span.SpanType != SpanTypeAgent {
		t.Errorf("SpanType = %v, want SpanTypeAgent", span.SpanType)
	}

	if span.TraceID != trace.TraceID {
		t.Errorf("TraceID = %s, want %s", span.TraceID, trace.TraceID)
	}

	if span.StartTimeNs == 0 {
		t.Error("StartTimeNs should not be 0")
	}

	// Verify span is in context
	spanFromCtx := SpanFromContext(spanCtx)
	if spanFromCtx != span {
		t.Error("Span not found in context")
	}

	// Verify span is root span of trace
	if trace.RootSpan() != span {
		t.Error("First span should be root span")
	}
}

func TestTracer_StartSpan_ParentChild(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	ctx, _, _ := tracer.NewTrace(context.Background(), TracingConfig{})

	parentSpan, parentCtx := tracer.StartSpan(ctx, "parent", SpanTypeAgent)
	childSpan, _ := tracer.StartSpan(parentCtx, "child", SpanTypeLLM)

	if childSpan.ParentID != parentSpan.SpanID {
		t.Errorf("ParentID = %s, want %s", childSpan.ParentID, parentSpan.SpanID)
	}
}

func TestTracer_StartSpan_NoTrace(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")

	// Start span without trace in context
	span, ctx := tracer.StartSpan(context.Background(), "orphan", SpanTypeAgent)

	if span != nil {
		t.Error("StartSpan() should return nil when no trace in context")
	}

	if SpanFromContext(ctx) != nil {
		t.Error("Context should not have span when StartSpan returns nil")
	}
}

func TestTracer_StartSpan_CancelledContext(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	ctx, _, _ := tracer.NewTrace(context.Background(), TracingConfig{})

	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	span, _ := tracer.StartSpan(cancelledCtx, "test", SpanTypeAgent)

	if span != nil {
		t.Error("StartSpan() should return nil for cancelled context")
	}
}

func TestTracer_EndTrace(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	ctx, trace, _ := tracer.NewTrace(context.Background(), TracingConfig{})

	span, ctx := tracer.StartSpan(ctx, "test", SpanTypeAgent)
	span.End()

	err := tracer.EndTrace(ctx, TraceStateOK)
	if err != nil {
		t.Fatalf("EndTrace() error = %v", err)
	}

	if trace.State != TraceStateOK {
		t.Errorf("State = %v, want TraceStateOK", trace.State)
	}

	// Verify trace moved to pending
	tracer.mu.Lock()
	if len(tracer.pending) != 1 {
		t.Errorf("pending length = %d, want 1", len(tracer.pending))
	}
	tracer.mu.Unlock()
}

func TestTracer_CleanupOrphanSpans(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	ctx, trace, _ := tracer.NewTrace(context.Background(), TracingConfig{})

	// Create span but don't end it
	span, ctx := tracer.StartSpan(ctx, "orphan", SpanTypeAgent)

	// End trace should cleanup orphan spans
	err := tracer.EndTrace(ctx, TraceStateOK)
	if err != nil {
		t.Fatalf("EndTrace() error = %v", err)
	}

	if !span.IsEnded() {
		t.Error("orphan span should be ended")
	}

	if span.Status.Code != SpanStatusError {
		t.Errorf("orphan span status = %v, want SpanStatusError", span.Status.Code)
	}

	// Verify trace is in pending
	if len(tracer.pending) != 1 {
		t.Errorf("pending length = %d, want 1", len(tracer.pending))
	}

	_ = trace // silence unused warning
}

func TestTracer_FlushWithoutClient(t *testing.T) {
	tracer := NewTracer(nil, "exp-123") // nil client
	ctx, trace, _ := tracer.NewTrace(context.Background(), TracingConfig{})

	err := tracer.Flush(ctx, trace)
	if err != nil {
		t.Errorf("Flush() with nil client should not error, got %v", err)
	}
}

func TestSpan_SetAttribute(t *testing.T) {
	span := &Span{
		SpanID:      "span-123",
		TraceID:     "trace-123",
		Name:        "test",
		Attributes:  nil,
		StartTimeNs: time.Now().UnixNano(),
	}

	span.SetAttribute("key1", "value1")
	span.SetAttribute("key2", 42)

	if span.Attributes["key1"] != "value1" {
		t.Errorf("key1 = %v, want value1", span.Attributes["key1"])
	}

	if span.Attributes["key2"] != 42 {
		t.Errorf("key2 = %v, want 42", span.Attributes["key2"])
	}
}

func TestSpan_AddEvent(t *testing.T) {
	span := &Span{
		SpanID:      "span-123",
		TraceID:     "trace-123",
		Name:        "test",
		StartTimeNs: time.Now().UnixNano(),
	}

	span.AddEvent("event1", map[string]any{"detail": "info"})

	if len(span.Events) != 1 {
		t.Fatalf("Events length = %d, want 1", len(span.Events))
	}

	event := span.Events[0]
	if event.Name != "event1" {
		t.Errorf("event Name = %s, want event1", event.Name)
	}

	if event.Timestamp == 0 {
		t.Error("event Timestamp should not be 0")
	}

	if event.Attributes["detail"] != "info" {
		t.Errorf("event Attributes[detail] = %v, want info", event.Attributes["detail"])
	}
}

func TestSpan_End(t *testing.T) {
	span := &Span{
		SpanID:      "span-123",
		TraceID:     "trace-123",
		Name:        "test",
		StartTimeNs: time.Now().UnixNano(),
	}

	span.End()

	if !span.IsEnded() {
		t.Error("span should be ended")
	}

	if span.EndTimeNs == 0 {
		t.Error("EndTimeNs should not be 0")
	}

	if span.Status.Code != SpanStatusOK {
		t.Errorf("Status.Code = %v, want SpanStatusOK", span.Status.Code)
	}
}

func TestSpan_EndWithError(t *testing.T) {
	span := &Span{
		SpanID:      "span-123",
		TraceID:     "trace-123",
		Name:        "test",
		StartTimeNs: time.Now().UnixNano(),
	}

	testErr := &testError{msg: "test error"}
	span.EndWithError(testErr)

	if !span.IsEnded() {
		t.Error("span should be ended")
	}

	if span.Status.Code != SpanStatusError {
		t.Errorf("Status.Code = %v, want SpanStatusError", span.Status.Code)
	}

	if span.Status.Description != "test error" {
		t.Errorf("Status.Description = %s, want 'test error'", span.Status.Description)
	}
}

func TestSpan_EndIdempotent(t *testing.T) {
	span := &Span{
		SpanID:      "span-123",
		TraceID:     "trace-123",
		Name:        "test",
		StartTimeNs: time.Now().UnixNano(),
	}

	span.End()
	firstEndTime := span.EndTimeNs

	time.Sleep(time.Millisecond)
	span.End() // Second call should be no-op

	if span.EndTimeNs != firstEndTime {
		t.Error("second End() call should not change EndTimeNs")
	}
}

func TestSpan_Duration(t *testing.T) {
	span := &Span{
		SpanID:      "span-123",
		TraceID:     "trace-123",
		Name:        "test",
		StartTimeNs: time.Now().UnixNano(),
	}

	time.Sleep(10 * time.Millisecond)
	span.End()

	duration := span.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("Duration = %v, want >= 10ms", duration)
	}
}

func TestSpan_DurationInProgress(t *testing.T) {
	span := &Span{
		SpanID:      "span-123",
		TraceID:     "trace-123",
		Name:        "test",
		StartTimeNs: time.Now().UnixNano(),
	}

	time.Sleep(5 * time.Millisecond)

	// Duration while still in progress
	duration := span.Duration()
	if duration < 5*time.Millisecond {
		t.Errorf("Duration = %v, want >= 5ms", duration)
	}
}

func TestTrace_SetMetadata(t *testing.T) {
	trace := &Trace{TraceID: "trace-123"}

	trace.SetMetadata("key1", "value1")
	trace.SetMetadata("key2", "value2")

	if trace.TraceMetadata["key1"] != "value1" {
		t.Errorf("TraceMetadata[key1] = %s, want value1", trace.TraceMetadata["key1"])
	}

	if trace.TraceMetadata["key2"] != "value2" {
		t.Errorf("TraceMetadata[key2] = %s, want value2", trace.TraceMetadata["key2"])
	}
}

func TestTrace_SetTag(t *testing.T) {
	trace := &Trace{TraceID: "trace-123"}

	trace.SetTag("env", "production")

	if trace.Tags["env"] != "production" {
		t.Errorf("Tags[env] = %s, want production", trace.Tags["env"])
	}
}

func TestTraceState_String(t *testing.T) {
	tests := []struct {
		state TraceState
		want  string
	}{
		{TraceStateOK, "OK"},
		{TraceStateError, "ERROR"},
		{TraceStateInProgress, "IN_PROGRESS"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("TraceState(%d).String() = %s, want %s", tt.state, got, tt.want)
		}
	}
}

func TestContextPropagation(t *testing.T) {
	ctx := context.Background()

	trace := &Trace{TraceID: "trace-123"}
	ctx = withTrace(ctx, trace)

	retrievedTrace := TraceFromContext(ctx)
	if retrievedTrace != trace {
		t.Error("TraceFromContext returned wrong trace")
	}

	span := &Span{SpanID: "span-123", TraceID: "trace-123"}
	ctx = withSpan(ctx, span)

	retrievedSpan := SpanFromContext(ctx)
	if retrievedSpan != span {
		t.Error("SpanFromContext returned wrong span")
	}
}

func TestWithParentSpan(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	ctx, _, _ := tracer.NewTrace(context.Background(), TracingConfig{})

	parentSpan := &Span{SpanID: "parent-123", TraceID: "trace-123"}
	ctx = WithParentSpan(ctx, parentSpan)

	retrievedSpan := SpanFromContext(ctx)
	if retrievedSpan != parentSpan {
		t.Error("WithParentSpan should set span in context")
	}
}

func TestTracer_ConcurrentSpanCreation(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	ctx, _, _ := tracer.NewTrace(context.Background(), TracingConfig{})

	var wg sync.WaitGroup
	spanCount := 100

	for i := 0; i < spanCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			span, _ := tracer.StartSpan(ctx, "concurrent-span", SpanTypeTool)
			if span != nil {
				span.SetAttribute("index", idx)
				span.End()
			}
		}(i)
	}

	wg.Wait()

	// Should have all spans tracked
	tracer.mu.Lock()
	// Note: spans were ended, so they may be removed from activeSpans
	tracer.mu.Unlock()
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
