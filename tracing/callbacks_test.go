package tracing

import (
	"context"
	"testing"
	"time"

	"charm.land/fantasy"
)

func TestTracingCallbacks_OnAgentStart(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{
		ExperimentID: "exp-123",
		AgentName:    "test-agent",
		ModelName:    "gpt-4",
	}
	callbacks := NewTracingCallbacks(tracer, config)

	ctx, err := callbacks.OnAgentStart(context.Background(), "Hello, world!")
	if err != nil {
		t.Fatalf("OnAgentStart() error = %v", err)
	}

	// Verify trace was created
	if callbacks.trace == nil {
		t.Fatal("trace should not be nil")
	}

	if callbacks.trace.RequestPreview != "Hello, world!" {
		t.Errorf("RequestPreview = %s, want 'Hello, world!'", callbacks.trace.RequestPreview)
	}

	// Verify agent span was created
	if callbacks.agentSpan == nil {
		t.Fatal("agentSpan should not be nil")
	}

	// Verify context contains trace and span
	trace := TraceFromContext(ctx)
	if trace == nil {
		t.Error("context should contain trace")
	}

	span := SpanFromContext(ctx)
	if span == nil {
		t.Error("context should contain span")
	}
}

func TestTracingCallbacks_OnAgentFinish(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{ExperimentID: "exp-123"}
	callbacks := NewTracingCallbacks(tracer, config)

	ctx, _ := callbacks.OnAgentStart(context.Background(), "test prompt")

	result := &fantasy.AgentResult{
		Response: fantasy.Response{
			Content: fantasy.ResponseContent{fantasy.TextContent{Text: "This is the response"}},
		},
		TotalUsage: fantasy.Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}

	tracingResult, err := callbacks.OnAgentFinish(ctx, result)
	if err != nil {
		t.Fatalf("OnAgentFinish() error = %v", err)
	}

	if tracingResult == nil {
		t.Fatal("tracingResult should not be nil")
	}

	if tracingResult.Trace == nil {
		t.Fatal("tracingResult.Trace should not be nil")
	}

	if tracingResult.Trace.ResponsePreview != "This is the response" {
		t.Errorf("ResponsePreview = %s, want 'This is the response'", tracingResult.Trace.ResponsePreview)
	}

	// Verify agent span has usage attributes
	if callbacks.agentSpan != nil {
		if callbacks.agentSpan.Attributes[AttrInputTokens] != int64(100) {
			t.Errorf("InputTokens = %v, want 100", callbacks.agentSpan.Attributes[AttrInputTokens])
		}
	}
}

func TestTracingCallbacks_OnAgentFinish_NilResult(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{ExperimentID: "exp-123"}
	callbacks := NewTracingCallbacks(tracer, config)

	ctx, _ := callbacks.OnAgentStart(context.Background(), "test prompt")

	tracingResult, err := callbacks.OnAgentFinish(ctx, nil)
	if err != nil {
		t.Fatalf("OnAgentFinish() error = %v", err)
	}

	// Should still return result with trace
	if tracingResult == nil || tracingResult.Trace == nil {
		t.Error("should return tracing result even with nil agent result")
	}

	// Trace state should be error for nil result
	if tracingResult.Trace.State != TraceStateError {
		t.Errorf("State = %v, want TraceStateError", tracingResult.Trace.State)
	}
}

func TestTracingCallbacks_OnStepStart(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{
		ExperimentID: "exp-123",
		ModelName:    "gpt-4",
	}
	callbacks := NewTracingCallbacks(tracer, config)

	ctx, _ := callbacks.OnAgentStart(context.Background(), "test prompt")

	ctx, err := callbacks.OnStepStart(ctx, 1)
	if err != nil {
		t.Fatalf("OnStepStart() error = %v", err)
	}

	// Verify step span was created
	if callbacks.currentStep == nil {
		t.Fatal("currentStep should not be nil")
	}

	if callbacks.currentStep.Attributes[AttrStepNumber] != 1 {
		t.Errorf("StepNumber = %v, want 1", callbacks.currentStep.Attributes[AttrStepNumber])
	}

	// Verify LLM span was created
	if callbacks.stepLLMSpan == nil {
		t.Fatal("stepLLMSpan should not be nil")
	}
}

func TestTracingCallbacks_OnStepFinish(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{ExperimentID: "exp-123"}
	callbacks := NewTracingCallbacks(tracer, config)

	ctx, _ := callbacks.OnAgentStart(context.Background(), "test prompt")
	ctx, _ = callbacks.OnStepStart(ctx, 1)

	stepResult := fantasy.StepResult{
		Response: fantasy.Response{
			Content: fantasy.ResponseContent{fantasy.TextContent{Text: "Step output"}},
			Usage: fantasy.Usage{
				InputTokens:     50,
				OutputTokens:    25,
				TotalTokens:     75,
				ReasoningTokens: 10,
			},
			FinishReason: fantasy.FinishReasonStop,
		},
	}

	err := callbacks.OnStepFinish(ctx, stepResult)
	if err != nil {
		t.Fatalf("OnStepFinish() error = %v", err)
	}

	// LLM span should be ended
	if callbacks.stepLLMSpan != nil {
		t.Error("stepLLMSpan should be nil after finish")
	}

	// Step span should be ended
	if callbacks.currentStep != nil {
		t.Error("currentStep should be nil after finish")
	}
}

func TestTracingCallbacks_OnToolCall(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{ExperimentID: "exp-123"}
	callbacks := NewTracingCallbacks(tracer, config)

	ctx, _ := callbacks.OnAgentStart(context.Background(), "test prompt")
	ctx, _ = callbacks.OnStepStart(ctx, 1)

	toolCall := fantasy.ToolCallContent{
		ToolName:   "search",
		ToolCallID: "call-123",
		Input:      `{"query": "test"}`,
	}

	ctx, err := callbacks.OnToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("OnToolCall() error = %v", err)
	}

	// Verify tool span was created
	toolSpan, ok := callbacks.stepToolSpans["call-123"]
	if !ok {
		t.Fatal("tool span should be tracked")
	}

	if toolSpan.Name != "search" {
		t.Errorf("tool span Name = %s, want search", toolSpan.Name)
	}

	if toolSpan.Attributes[AttrToolCallID] != "call-123" {
		t.Errorf("ToolCallID = %v, want call-123", toolSpan.Attributes[AttrToolCallID])
	}
}

func TestTracingCallbacks_OnToolResult(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{ExperimentID: "exp-123"}
	callbacks := NewTracingCallbacks(tracer, config)

	ctx, _ := callbacks.OnAgentStart(context.Background(), "test prompt")
	ctx, _ = callbacks.OnStepStart(ctx, 1)

	toolCall := fantasy.ToolCallContent{
		ToolName:   "search",
		ToolCallID: "call-123",
		Input:      `{"query": "test"}`,
	}
	ctx, _ = callbacks.OnToolCall(ctx, toolCall)

	toolResult := fantasy.ToolResultContent{
		ToolCallID: "call-123",
		Result:     fantasy.ToolResultOutputContentText{Text: "search results"},
	}

	err := callbacks.OnToolResult(ctx, toolResult)
	if err != nil {
		t.Fatalf("OnToolResult() error = %v", err)
	}

	// Tool span should be removed from tracking
	if _, ok := callbacks.stepToolSpans["call-123"]; ok {
		t.Error("tool span should be removed after result")
	}
}

func TestTracingCallbacks_OnAgentError(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{ExperimentID: "exp-123"}
	callbacks := NewTracingCallbacks(tracer, config)

	ctx, _ := callbacks.OnAgentStart(context.Background(), "test prompt")

	testErr := &testError{msg: "agent failed"}
	callbacks.OnAgentError(ctx, testErr)

	// Agent span should be ended with error
	if callbacks.agentSpan == nil {
		t.Fatal("agentSpan should not be nil")
	}

	if callbacks.agentSpan.Status.Code != SpanStatusError {
		t.Errorf("agentSpan Status = %v, want SpanStatusError", callbacks.agentSpan.Status.Code)
	}
}

func TestTracingCallbacks_StreamCallbacks(t *testing.T) {
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{ExperimentID: "exp-123"}
	callbacks := NewTracingCallbacks(tracer, config)

	streamCbs := callbacks.StreamCallbacks()

	// Verify all callbacks are set
	if streamCbs.OnAgentStart == nil {
		t.Error("OnAgentStart should not be nil")
	}
	if streamCbs.OnAgentFinish == nil {
		t.Error("OnAgentFinish should not be nil")
	}
	if streamCbs.OnStepStart == nil {
		t.Error("OnStepStart should not be nil")
	}
	if streamCbs.OnStepFinish == nil {
		t.Error("OnStepFinish should not be nil")
	}
	if streamCbs.OnToolCall == nil {
		t.Error("OnToolCall should not be nil")
	}
	if streamCbs.OnToolResult == nil {
		t.Error("OnToolResult should not be nil")
	}
	if streamCbs.OnError == nil {
		t.Error("OnError should not be nil")
	}

	// Test calling the callbacks
	streamCbs.OnAgentStart()
	if callbacks.trace == nil {
		t.Error("OnAgentStart callback should create trace")
	}
}

func TestWrapAgentOptions(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "exp-123",
		AgentName:    "test-agent",
		FlushTimeout: 5 * time.Second,
	}

	callbacks, tracer := WrapAgentOptions(nil, config)

	if callbacks == nil {
		t.Fatal("callbacks should not be nil")
	}

	if tracer == nil {
		t.Fatal("tracer should not be nil")
	}

	if tracer.flushTimeout != 5*time.Second {
		t.Errorf("flushTimeout = %v, want 5s", tracer.flushTimeout)
	}
}

func TestTruncateUTF8(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "truncate with ellipsis",
			input:  "hello world this is a long string",
			maxLen: 15,
			want:   "hello world ...",
		},
		{
			name:   "unicode safe",
			input:  "hello 世界",
			maxLen: 10,
			want:   "hello 世...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateUTF8(tt.input, tt.maxLen)
			if len(got) > tt.maxLen {
				t.Errorf("truncateUTF8() length = %d, want <= %d", len(got), tt.maxLen)
			}
		})
	}
}

func TestTracingCallbacks_PanicRecovery(t *testing.T) {
	// This test verifies that panic recovery doesn't crash the test
	// The actual panic recovery is in recoverPanic which swallows panics
	tracer := NewTracer(nil, "exp-123")
	config := TracingConfig{ExperimentID: "exp-123"}
	callbacks := NewTracingCallbacks(tracer, config)

	// Should not panic even if internal state is inconsistent
	callbacks.OnAgentError(context.Background(), &testError{msg: "test"})
	callbacks.OnStepFinish(context.Background(), fantasy.StepResult{})
	callbacks.OnToolResult(context.Background(), fantasy.ToolResultContent{ToolCallID: "nonexistent"})
}
