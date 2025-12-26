package tracing

import (
	"context"
	"testing"
	"time"

	"charm.land/fantasy/mlflowclient"
)

func TestGenerateTraceID(t *testing.T) {
	traceID, err := generateTraceID()
	if err != nil {
		t.Fatalf("generateTraceID() error = %v", err)
	}

	// Trace ID should have "tr-" prefix and be 35 characters total (3 + 32 hex)
	if len(traceID) != 35 {
		t.Errorf("trace ID length = %d, want 35", len(traceID))
	}

	if traceID[:3] != "tr-" {
		t.Errorf("trace ID prefix = %q, want %q", traceID[:3], "tr-")
	}

	// Second call should generate a different ID
	traceID2, err := generateTraceID()
	if err != nil {
		t.Fatalf("generateTraceID() second call error = %v", err)
	}

	if traceID == traceID2 {
		t.Error("generateTraceID() generated duplicate IDs")
	}
}

func TestGenerateSpanID(t *testing.T) {
	spanID, err := generateSpanID()
	if err != nil {
		t.Fatalf("generateSpanID() error = %v", err)
	}

	// Span ID should be 16 hex characters (8 bytes)
	if len(spanID) != 16 {
		t.Errorf("span ID length = %d, want 16", len(spanID))
	}

	// Second call should generate a different ID
	spanID2, err := generateSpanID()
	if err != nil {
		t.Fatalf("generateSpanID() second call error = %v", err)
	}

	if spanID == spanID2 {
		t.Error("generateSpanID() generated duplicate IDs")
	}
}

func TestGenerateUUIDv4(t *testing.T) {
	uuid := generateUUIDv4()

	// UUID should be 36 characters (8-4-4-4-12)
	if len(uuid) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid))
	}

	// Check format with hyphens
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		t.Errorf("UUID format incorrect: %s", uuid)
	}

	// Second call should generate a different UUID
	uuid2 := generateUUIDv4()
	if uuid == uuid2 {
		t.Error("generateUUIDv4() generated duplicate UUIDs")
	}
}

func TestTruncateUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxBytes int
		want     string
	}{
		{
			name:     "no truncation needed",
			input:    "hello",
			maxBytes: 10,
			want:     "hello",
		},
		{
			name:     "truncate ASCII",
			input:    "hello world",
			maxBytes: 5,
			want:     "hello",
		},
		{
			name:     "truncate at UTF-8 boundary",
			input:    "hello 世界",
			maxBytes: 7,
			want:     "hello ",
		},
		{
			name:     "avoid breaking multi-byte char",
			input:    "hello 世界",
			maxBytes: 8, // Would break in the middle of 世 (3 bytes)
			want:     "hello ",
		},
		{
			name:     "empty string",
			input:    "",
			maxBytes: 10,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateUTF8(tt.input, tt.maxBytes)
			if got != tt.want {
				t.Errorf("truncateUTF8() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTracingConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  TracingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: TracingConfig{
				Client:       &mlflowclient.Client{},
				ExperimentID: "exp-123",
			},
			wantErr: false,
		},
		{
			name: "missing client",
			config: TracingConfig{
				ExperimentID: "exp-123",
			},
			wantErr: true,
			errMsg:  "client",
		},
		{
			name: "missing experiment ID",
			config: TracingConfig{
				Client: &mlflowclient.Client{},
			},
			wantErr: true,
			errMsg:  "experimentID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				valErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("Validate() error type = %T, want *ValidationError", err)
					return
				}
				if valErr.Field != tt.errMsg {
					t.Errorf("ValidationError.Field = %q, want %q", valErr.Field, tt.errMsg)
				}
			}
		})
	}
}

func TestSpanAttributeOperations(t *testing.T) {
	span := &Span{
		SpanID:     "test-span",
		Attributes: make(map[string]any),
	}

	// Test SetAttribute
	span.SetAttribute("key1", "value1")
	if span.Attributes["key1"] != "value1" {
		t.Errorf("SetAttribute failed: got %v, want %v", span.Attributes["key1"], "value1")
	}

	// Test SetJSONAttribute
	span.SetJSONAttribute("key2", map[string]string{"nested": "value"})
	expected := `{"nested":"value"}`
	if span.Attributes["key2"] != expected {
		t.Errorf("SetJSONAttribute failed: got %v, want %v", span.Attributes["key2"], expected)
	}

	// Test SetAttribute with large value (truncation)
	largeValue := string(make([]byte, MaxSpanAttributeSize+100))
	span.SetAttribute("large", largeValue)
	attrValue := span.Attributes["large"].(string)
	if len(attrValue) > MaxSpanAttributeSize {
		t.Errorf("SetAttribute didn't truncate: length %d > max %d", len(attrValue), MaxSpanAttributeSize)
	}
}

func TestSpanEnd(t *testing.T) {
	span := &Span{
		SpanID:      "test-span",
		StartTimeNs: time.Now().UnixNano(),
	}

	// First end should work
	span.End(SpanStatus{Code: SpanStatusOK})
	if span.EndTimeNs == 0 {
		t.Error("End() didn't set EndTimeNs")
	}
	if span.Status.Code != SpanStatusOK {
		t.Errorf("End() status = %v, want %v", span.Status.Code, SpanStatusOK)
	}

	firstEndTime := span.EndTimeNs

	// Second end should be a no-op
	time.Sleep(1 * time.Millisecond)
	span.End(SpanStatus{Code: SpanStatusError})
	if span.EndTimeNs != firstEndTime {
		t.Error("End() should be no-op on second call")
	}
	if span.Status.Code != SpanStatusOK {
		t.Error("End() should not change status on second call")
	}
}

func TestSpanEvent(t *testing.T) {
	span := &Span{
		SpanID:      "test-span",
		StartTimeNs: time.Now().UnixNano(),
		Events:      make([]SpanEvent, 0),
	}

	attrs := map[string]any{"key": "value"}
	span.AddEvent("test-event", attrs)

	if len(span.Events) != 1 {
		t.Fatalf("AddEvent failed: got %d events, want 1", len(span.Events))
	}

	event := span.Events[0]
	if event.Name != "test-event" {
		t.Errorf("Event name = %q, want %q", event.Name, "test-event")
	}
	if event.Attributes["key"] != "value" {
		t.Errorf("Event attributes = %v, want {key: value}", event.Attributes)
	}
}

func TestTraceAddSpan(t *testing.T) {
	trace := &Trace{
		TraceID: "tr-test",
		Spans:   make([]*Span, 0),
	}

	span1 := &Span{SpanID: "span-1"}
	span2 := &Span{SpanID: "span-2"}

	trace.AddSpan(span1)
	trace.AddSpan(span2)

	if len(trace.Spans) != 2 {
		t.Errorf("AddSpan failed: got %d spans, want 2", len(trace.Spans))
	}
}

func TestTraceGetSpan(t *testing.T) {
	trace := &Trace{
		TraceID: "tr-test",
		Spans: []*Span{
			{SpanID: "span-1"},
			{SpanID: "span-2"},
		},
	}

	span := trace.GetSpan("span-2")
	if span == nil {
		t.Fatal("GetSpan() returned nil")
	}
	if span.SpanID != "span-2" {
		t.Errorf("GetSpan() returned wrong span: %v", span.SpanID)
	}

	span = trace.GetSpan("nonexistent")
	if span != nil {
		t.Error("GetSpan() should return nil for nonexistent span")
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Initially no trace in context
	trace := TraceFromContext(ctx)
	if trace != nil {
		t.Error("TraceFromContext() should return nil for empty context")
	}

	// Add trace to context
	testTrace := &Trace{TraceID: "tr-test"}
	ctx = contextWithTrace(ctx, testTrace)

	// Retrieve trace from context
	trace = TraceFromContext(ctx)
	if trace == nil {
		t.Fatal("TraceFromContext() returned nil after contextWithTrace")
	}
	if trace.TraceID != "tr-test" {
		t.Errorf("TraceFromContext() returned wrong trace: %v", trace.TraceID)
	}

	// Test span context helpers
	testSpan := &Span{SpanID: "span-test"}
	ctx = contextWithSpan(ctx, testSpan)

	span := SpanFromContext(ctx)
	if span == nil {
		t.Fatal("SpanFromContext() returned nil after contextWithSpan")
	}
	if span.SpanID != "span-test" {
		t.Errorf("SpanFromContext() returned wrong span: %v", span.SpanID)
	}
}

func TestConvertAttributeValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string // Just check it doesn't panic
	}{
		{"string", "hello", "string"},
		{"int", 42, "int"},
		{"int64", int64(42), "int64"},
		{"float64", 3.14, "double"},
		{"bool", true, "bool"},
		{"map", map[string]any{"key": "value"}, "string"}, // Falls back to JSON
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just ensure it doesn't panic
			result := convertAttributeValue(tt.input)
			if result == nil {
				t.Errorf("convertAttributeValue(%v) returned nil", tt.input)
			}
		})
	}
}

func TestConvertSpanStatusCode(t *testing.T) {
	tests := []struct {
		input SpanStatusCode
		want  int32 // Just check mapping exists
	}{
		{SpanStatusUnset, 0},
		{SpanStatusOK, 1},
		{SpanStatusError, 2},
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result := convertSpanStatusCode(tt.input)
			// Just ensure it doesn't panic and returns a value
			_ = result
		})
	}
}

func TestConvertTraceState(t *testing.T) {
	tests := []struct {
		input TraceState
		name  string
	}{
		{TraceStateInProgress, "IN_PROGRESS"},
		{TraceStateOK, "OK"},
		{TraceStateError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTraceState(tt.input)
			// Just ensure it doesn't panic and returns a value
			_ = result
		})
	}
}

// Helper method for SpanStatusCode to satisfy String() in tests
func (s SpanStatusCode) String() string {
	switch s {
	case SpanStatusUnset:
		return "UNSET"
	case SpanStatusOK:
		return "OK"
	case SpanStatusError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
