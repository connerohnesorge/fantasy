package tracing

import (
	"testing"
	"time"
)

func TestConvertTrace(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "test-exp",
		AgentName:    "test-agent",
		ModelName:    "test-model",
	}

	tracer := NewTracer(config)
	trace := tracer.StartTrace("What is the weather?")

	// Create agent span
	agentSpan := tracer.StartSpan("agent", SpanTypeAgent)
	agentSpan.SetAttribute("model", "claude-3-opus")
	agentSpan.SetStatus(SpanStatusOK, "")
	agentSpan.End()

	// Create step span
	stepSpan := tracer.StartSpan("step_1", SpanTypeChain)
	stepSpan.End()

	// Create LLM span
	llmSpan := tracer.StartSpan("llm", SpanTypeLLM)
	llmSpan.SetAttribute(AttrTokenUsage, TokenUsage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	})
	llmSpan.SetStatus(SpanStatusOK, "")
	llmSpan.End()

	// End all spans
	tracer.EndSpan(llmSpan)
	tracer.EndSpan(stepSpan)
	tracer.EndSpan(agentSpan)

	// Set trace state
	trace.ExecutionDuration = 1000 // 1 second
	trace.SetState(TraceStateOK)
	tracer.SetResponsePreview("The weather is sunny.")

	// Convert to protobuf
	pbTrace, err := ConvertTrace(trace)
	if err != nil {
		t.Fatalf("ConvertTrace failed: %v", err)
	}

	if pbTrace == nil {
		t.Fatal("Converted trace should not be nil")
	}

	// Verify trace info
	if pbTrace.TraceInfo == nil {
		t.Fatal("TraceInfo should not be nil")
	}

	if pbTrace.TraceInfo.TraceId == nil || *pbTrace.TraceInfo.TraceId != trace.TraceID {
		t.Errorf("TraceID mismatch")
	}

	if pbTrace.TraceInfo.TraceLocation == nil {
		t.Error("TraceLocation should be set")
	}

	if pbTrace.TraceInfo.RequestPreview == nil || *pbTrace.TraceInfo.RequestPreview != trace.RequestPreview {
		t.Errorf("RequestPreview mismatch")
	}

	if pbTrace.TraceInfo.ResponsePreview == nil || *pbTrace.TraceInfo.ResponsePreview != trace.ResponsePreview {
		t.Errorf("ResponsePreview mismatch")
	}

	// Verify spans
	if len(pbTrace.Spans) != 3 {
		t.Errorf("Expected 3 spans, got %d", len(pbTrace.Spans))
	}

	// Verify first span (agent)
	agentPbSpan := pbTrace.Spans[0]
	if agentPbSpan.Name != "agent" {
		t.Errorf("Expected agent span name, got %s", agentPbSpan.Name)
	}

	if len(agentPbSpan.TraceId) != 16 {
		t.Errorf("TraceID should be 16 bytes, got %d", len(agentPbSpan.TraceId))
	}

	if len(agentPbSpan.SpanId) != 8 {
		t.Errorf("SpanID should be 8 bytes, got %d", len(agentPbSpan.SpanId))
	}

	if len(agentPbSpan.ParentSpanId) != 0 {
		t.Error("Agent span should have no parent")
	}

	// Verify span has attributes
	if len(agentPbSpan.Attributes) == 0 {
		t.Error("Agent span should have attributes")
	}

	// Verify status
	if agentPbSpan.Status == nil {
		t.Error("Span should have status")
	}
}

func TestConvertSpan(t *testing.T) {
	span := &Span{
		TraceID:     "0123456789abcdef0123456789abcdef",
		SpanID:      "0123456789abcdef",
		ParentID:    "fedcba9876543210",
		Name:        "test-span",
		SpanType:    SpanTypeLLM,
		StartTimeNs: time.Now().UnixNano(),
		EndTimeNs:   time.Now().UnixNano() + 1_000_000_000, // 1 second later
		Status: SpanStatus{
			Code:        SpanStatusOK,
			Description: "Success",
		},
		Attributes: map[string]any{
			"key1": "value1",
			"key2": int64(42),
			"key3": 3.14,
			"key4": true,
		},
		Events: []SpanEvent{
			{
				Name:      "test-event",
				Timestamp: time.Now().UnixNano(),
				Attributes: map[string]any{
					"event_key": "event_value",
				},
			},
		},
	}

	pbSpan, err := convertSpan(span)
	if err != nil {
		t.Fatalf("convertSpan failed: %v", err)
	}

	if pbSpan.Name != "test-span" {
		t.Errorf("Expected name 'test-span', got '%s'", pbSpan.Name)
	}

	if len(pbSpan.TraceId) != 16 {
		t.Errorf("TraceID should be 16 bytes, got %d", len(pbSpan.TraceId))
	}

	if len(pbSpan.SpanId) != 8 {
		t.Errorf("SpanID should be 8 bytes, got %d", len(pbSpan.SpanId))
	}

	if len(pbSpan.ParentSpanId) != 8 {
		t.Errorf("ParentSpanID should be 8 bytes, got %d", len(pbSpan.ParentSpanId))
	}

	if pbSpan.StartTimeUnixNano == 0 {
		t.Error("StartTimeUnixNano should be set")
	}

	if pbSpan.EndTimeUnixNano == 0 {
		t.Error("EndTimeUnixNano should be set")
	}

	// Verify attributes (4 custom attributes)
	if len(pbSpan.Attributes) != 4 {
		t.Errorf("Expected 4 attributes, got %d", len(pbSpan.Attributes))
	}

	// Verify events
	if len(pbSpan.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(pbSpan.Events))
	}

	if pbSpan.Events[0].Name != "test-event" {
		t.Errorf("Expected event name 'test-event', got '%s'", pbSpan.Events[0].Name)
	}

	// Verify status
	if pbSpan.Status == nil {
		t.Fatal("Status should not be nil")
	}

	// Check status code mapping
	if pbSpan.Status.Code.String() != "STATUS_CODE_OK" {
		t.Errorf("Expected status OK, got %v", pbSpan.Status.Code)
	}
}

func TestConvertAttribute(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value any
		want  bool // whether conversion should succeed
	}{
		{"string", "key", "value", true},
		{"int", "key", 42, true},
		{"int64", "key", int64(42), true},
		{"float64", "key", 3.14, true},
		{"bool", "key", true, true},
		{"bytes", "key", []byte("data"), true},
		{"TokenUsage", "key", TokenUsage{InputTokens: 100}, true},
		{"map", "key", map[string]any{"nested": "value"}, true},
		{"array", "key", []any{1, 2, 3}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kv, err := convertAttribute(tt.key, tt.value)
			if tt.want && err != nil {
				t.Errorf("convertAttribute failed: %v", err)
			}
			if !tt.want && err == nil {
				t.Error("convertAttribute should have failed")
			}
			if tt.want && kv == nil {
				t.Error("KeyValue should not be nil")
			}
			if tt.want && kv.Key != tt.key {
				t.Errorf("Expected key '%s', got '%s'", tt.key, kv.Key)
			}
		})
	}
}

func TestHexToBytes(t *testing.T) {
	tests := []struct {
		name        string
		hexStr      string
		expectedLen int
		wantErr     bool
	}{
		{"16 bytes", "0123456789abcdef0123456789abcdef", 16, false},
		{"8 bytes", "0123456789abcdef", 8, false},
		{"short padded", "abcdef", 8, false},
		{"long truncated", "0123456789abcdef0123456789abcdef", 8, false},
		{"invalid hex", "xyz", 8, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := hexToBytes(tt.hexStr, tt.expectedLen)
			if tt.wantErr && err == nil {
				t.Error("Expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.wantErr && len(bytes) != tt.expectedLen {
				t.Errorf("Expected %d bytes, got %d", tt.expectedLen, len(bytes))
			}
		})
	}
}
