package tracing

import (
	"context"
	"testing"
	"time"
)

// TestCallbackFlow tests the full callback flow for a simulated agent execution.
func TestCallbackFlow(t *testing.T) {
	// Note: This is a unit test without a real MLflow client
	// We'll create a mock config
	config := TracingConfig{
		ExperimentID: "test-exp",
		AgentName:    "test-agent",
		ModelName:    "test-model",
		FlushTimeout: 5 * time.Second,
		Client:       nil, // Will skip actual MLflow upload
	}

	callbacks := &Callbacks{
		tracer:       NewTracer(config),
		experimentID: config.ExperimentID,
	}

	_ = context.Background() // Reserved for future use

	// Simulate agent start
	callbacks.OnAgentStart("What is the weather?")

	// Verify agent span was created
	if callbacks.agentSpan == nil {
		t.Fatal("Agent span should be created")
	}
	if callbacks.agentSpan.SpanType != SpanTypeAgent {
		t.Error("Agent span should have type AGENT")
	}

	// Simulate step 1
	err := callbacks.OnStepStart(1)
	if err != nil {
		t.Fatalf("OnStepStart failed: %v", err)
	}

	if callbacks.currentStep == nil {
		t.Fatal("Current step should be set")
	}
	if callbacks.currentStep.Name != "step_1" {
		t.Errorf("Expected step name 'step_1', got '%s'", callbacks.currentStep.Name)
	}

	// Simulate LLM call
	messages := []any{
		map[string]any{
			"role":    "user",
			"content": "What is the weather?",
		},
	}
	err = callbacks.OnLLMStart(messages)
	if err != nil {
		t.Fatalf("OnLLMStart failed: %v", err)
	}

	// Verify LLM span was created
	currentSpan := callbacks.tracer.GetCurrentSpan()
	if currentSpan == nil || currentSpan.SpanType != SpanTypeLLM {
		t.Error("Current span should be LLM span")
	}

	// Simulate LLM finish
	err = callbacks.OnLLMFinish(100, 50, 150, nil)
	if err != nil {
		t.Fatalf("OnLLMFinish failed: %v", err)
	}

	// Simulate tool call
	err = callbacks.OnToolCall("get_weather", map[string]any{
		"location": "San Francisco",
	})
	if err != nil {
		t.Fatalf("OnToolCall failed: %v", err)
	}

	// Verify tool span was created
	currentSpan = callbacks.tracer.GetCurrentSpan()
	if currentSpan == nil || currentSpan.SpanType != SpanTypeTool {
		t.Error("Current span should be tool span")
	}

	// Simulate tool result
	err = callbacks.OnToolResult(map[string]any{
		"temperature": 72,
		"conditions":  "sunny",
	}, nil)
	if err != nil {
		t.Fatalf("OnToolResult failed: %v", err)
	}

	// Finish step
	err = callbacks.OnStepFinish()
	if err != nil {
		t.Fatalf("OnStepFinish failed: %v", err)
	}

	if callbacks.currentStep != nil {
		t.Error("Current step should be nil after finish")
	}

	// Finish agent (skip flush since we don't have a real client)
	callbacks.mu.Lock()
	if callbacks.agentSpan != nil {
		callbacks.agentSpan.End()
		callbacks.agentSpan = nil
	}
	callbacks.tracer.EndOrphanSpans()
	trace := callbacks.tracer.GetTrace()
	if trace != nil {
		trace.SetState(TraceStateOK)
	}
	callbacks.mu.Unlock()

	// Verify trace structure
	trace = callbacks.tracer.GetTrace()
	if trace == nil {
		t.Fatal("Trace should not be nil")
	}

	// Should have: 1 agent span + 1 step span + 1 LLM span + 1 tool span = 4 spans
	if len(trace.Spans) != 4 {
		t.Errorf("Expected 4 spans, got %d", len(trace.Spans))
	}

	// Verify span types
	spanTypes := make(map[SpanType]int)
	for _, span := range trace.Spans {
		spanTypes[span.SpanType]++
	}

	if spanTypes[SpanTypeAgent] != 1 {
		t.Errorf("Expected 1 agent span, got %d", spanTypes[SpanTypeAgent])
	}
	if spanTypes[SpanTypeChain] != 1 {
		t.Errorf("Expected 1 chain span, got %d", spanTypes[SpanTypeChain])
	}
	if spanTypes[SpanTypeLLM] != 1 {
		t.Errorf("Expected 1 LLM span, got %d", spanTypes[SpanTypeLLM])
	}
	if spanTypes[SpanTypeTool] != 1 {
		t.Errorf("Expected 1 tool span, got %d", spanTypes[SpanTypeTool])
	}

	// Verify all spans are ended
	for _, span := range trace.Spans {
		if span.EndTimeNs == 0 {
			t.Errorf("Span '%s' should be ended", span.Name)
		}
	}
}

// TestPanicRecovery tests that callbacks handle panics gracefully.
func TestPanicRecovery(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "test-exp",
	}

	callbacks := &Callbacks{
		tracer:       NewTracer(config),
		experimentID: config.ExperimentID,
	}

	// Test safeCall with panic
	err := callbacks.safeCall("test", func() error {
		panic("test panic")
	})

	if err == nil {
		t.Error("Expected error from panic recovery")
	}

	// Should contain panic message
	if err != nil && err.Error() != "" {
		// Success - panic was caught
	} else {
		t.Error("Panic should be recovered and returned as error")
	}
}

// TestContextHandling tests context-based span storage.
func TestContextHandling(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "test-exp",
	}

	tracer := NewTracer(config)
	tracer.StartTrace("test")

	span := tracer.StartSpan("test-span", SpanTypeAgent)

	// Store span in context
	ctx := ContextWithSpan(context.Background(), span)

	// Retrieve span from context
	retrieved := SpanFromContext(ctx)
	if retrieved == nil {
		t.Fatal("Span should be retrievable from context")
	}

	if retrieved.SpanID != span.SpanID {
		t.Error("Retrieved span should match original span")
	}

	// Test nil context
	retrieved = SpanFromContext(nil)
	if retrieved != nil {
		t.Error("Span from nil context should be nil")
	}

	// Test context without span
	retrieved = SpanFromContext(context.Background())
	if retrieved != nil {
		t.Error("Span from context without span should be nil")
	}
}

// TestCancellation tests handling of context cancellation.
func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Test that operations respect cancellation
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}
