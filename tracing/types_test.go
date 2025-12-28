package tracing

import (
	"testing"
	"time"
)

func TestSpanSetAttribute(t *testing.T) {
	span := &Span{
		SpanID:      "test-span",
		TraceID:     "test-trace",
		Name:        "test",
		SpanType:    SpanTypeTool,
		StartTimeNs: time.Now().UnixNano(),
	}

	span.SetAttribute("key1", "value1")
	span.SetAttribute("key2", 42)

	if span.Attributes["key1"] != "value1" {
		t.Errorf("Expected key1 to be 'value1', got %v", span.Attributes["key1"])
	}
	if span.Attributes["key2"] != 42 {
		t.Errorf("Expected key2 to be 42, got %v", span.Attributes["key2"])
	}
}

func TestSpanSetStatus(t *testing.T) {
	span := &Span{
		SpanID:      "test-span",
		TraceID:     "test-trace",
		Name:        "test",
		SpanType:    SpanTypeTool,
		StartTimeNs: time.Now().UnixNano(),
	}

	span.SetStatus(SpanStatusOK, "success")

	if span.Status.Code != SpanStatusOK {
		t.Errorf("Expected status code OK, got %v", span.Status.Code)
	}
	if span.Status.Description != "success" {
		t.Errorf("Expected status description 'success', got %s", span.Status.Description)
	}
}

func TestSpanEnd(t *testing.T) {
	span := &Span{
		SpanID:      "test-span",
		TraceID:     "test-trace",
		Name:        "test",
		SpanType:    SpanTypeTool,
		StartTimeNs: time.Now().UnixNano(),
	}

	span.End()

	if span.EndTimeNs == 0 {
		t.Error("Expected EndTimeNs to be set")
	}
	if span.EndTimeNs < span.StartTimeNs {
		t.Error("End time should be after start time")
	}

	// Test that calling End() again doesn't change the time
	firstEnd := span.EndTimeNs
	time.Sleep(1 * time.Millisecond)
	span.End()
	if span.EndTimeNs != firstEnd {
		t.Error("End() should not change EndTimeNs if already set")
	}
}

func TestSpanAddEvent(t *testing.T) {
	span := &Span{
		SpanID:      "test-span",
		TraceID:     "test-trace",
		Name:        "test",
		SpanType:    SpanTypeTool,
		StartTimeNs: time.Now().UnixNano(),
	}

	span.AddEvent("event1", map[string]any{"key": "value"})
	span.AddEvent("event2", nil)

	if len(span.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(span.Events))
	}
	if span.Events[0].Name != "event1" {
		t.Errorf("Expected event name 'event1', got %s", span.Events[0].Name)
	}
	if span.Events[0].Attributes["key"] != "value" {
		t.Errorf("Expected event attribute 'key' to be 'value', got %v", span.Events[0].Attributes["key"])
	}
}

func TestTraceStateString(t *testing.T) {
	tests := []struct {
		state    TraceState
		expected string
	}{
		{TraceStateInProgress, "IN_PROGRESS"},
		{TraceStateOK, "OK"},
		{TraceStateError, "ERROR"},
		{TraceState(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("TraceState(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}

func TestTraceAddSpan(t *testing.T) {
	trace := &Trace{
		TraceID:      "test-trace",
		ExperimentID: "exp123",
		State:        TraceStateInProgress,
	}

	span1 := &Span{SpanID: "span1"}
	span2 := &Span{SpanID: "span2"}

	trace.AddSpan(span1)
	trace.AddSpan(span2)

	if len(trace.Spans) != 2 {
		t.Errorf("Expected 2 spans, got %d", len(trace.Spans))
	}
}

func TestTraceGetSpan(t *testing.T) {
	trace := &Trace{
		TraceID:      "test-trace",
		ExperimentID: "exp123",
		State:        TraceStateInProgress,
	}

	span1 := &Span{SpanID: "span1", Name: "first"}
	span2 := &Span{SpanID: "span2", Name: "second"}

	trace.AddSpan(span1)
	trace.AddSpan(span2)

	found := trace.GetSpan("span2")
	if found == nil {
		t.Error("Expected to find span2")
	} else if found.Name != "second" {
		t.Errorf("Expected span name 'second', got %s", found.Name)
	}

	notFound := trace.GetSpan("span3")
	if notFound != nil {
		t.Error("Expected nil for non-existent span")
	}
}

func TestTraceSetState(t *testing.T) {
	trace := &Trace{
		TraceID:      "test-trace",
		ExperimentID: "exp123",
		State:        TraceStateInProgress,
	}

	trace.SetState(TraceStateOK)

	if trace.State != TraceStateOK {
		t.Errorf("Expected state OK, got %v", trace.State)
	}
}
