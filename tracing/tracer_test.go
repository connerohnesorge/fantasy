package tracing

import (
	"sync"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	// Test that IDs are generated
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("Generated ID should not be empty")
	}

	if id2 == "" {
		t.Error("Generated ID should not be empty")
	}

	// IDs should be 32 characters (16 bytes hex encoded)
	if len(id1) != 32 {
		t.Errorf("Expected ID length 32, got %d", len(id1))
	}

	// IDs should be unique
	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}
}

func TestTracer_StartTrace(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "test-exp",
		AgentName:    "test-agent",
		ModelName:    "test-model",
		SessionID:    "test-session",
		Tags: map[string]string{
			"env": "test",
		},
	}

	tracer := NewTracer(config)
	trace := tracer.StartTrace("test request")

	if trace == nil {
		t.Fatal("Trace should not be nil")
	}

	if trace.TraceID == "" {
		t.Error("TraceID should not be empty")
	}

	if trace.ExperimentID != "test-exp" {
		t.Errorf("Expected experiment ID 'test-exp', got '%s'", trace.ExperimentID)
	}

	if trace.State != TraceStateInProgress {
		t.Errorf("Expected state IN_PROGRESS, got %v", trace.State)
	}

	if trace.RequestPreview != "test request" {
		t.Errorf("Expected request preview 'test request', got '%s'", trace.RequestPreview)
	}

	// Check metadata
	if trace.TraceMetadata["agent_name"] != "test-agent" {
		t.Error("agent_name metadata not set")
	}
	if trace.TraceMetadata["model_name"] != "test-model" {
		t.Error("model_name metadata not set")
	}
	if trace.TraceMetadata["session_id"] != "test-session" {
		t.Error("session_id metadata not set")
	}

	// Check tags
	if trace.Tags["env"] != "test" {
		t.Error("Custom tag not copied")
	}
}

func TestTracer_SpanHierarchy(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "test-exp",
	}

	tracer := NewTracer(config)
	tracer.StartTrace("test")

	// Create agent span (root)
	agentSpan := tracer.StartSpan("agent", SpanTypeAgent)
	if agentSpan.ParentID != "" {
		t.Error("Agent span should have no parent")
	}

	// Create step span (child of agent)
	stepSpan := tracer.StartSpan("step_1", SpanTypeChain)
	if stepSpan.ParentID != agentSpan.SpanID {
		t.Errorf("Step span parent should be agent span, got %s", stepSpan.ParentID)
	}

	// Create LLM span (child of step)
	llmSpan := tracer.StartSpan("llm", SpanTypeLLM)
	if llmSpan.ParentID != stepSpan.SpanID {
		t.Errorf("LLM span parent should be step span, got %s", llmSpan.ParentID)
	}

	// End LLM span
	tracer.EndSpan(llmSpan)

	// Current span should now be step span
	current := tracer.GetCurrentSpan()
	if current.SpanID != stepSpan.SpanID {
		t.Error("Current span should be step span after ending LLM span")
	}

	// Create tool span (also child of step)
	toolSpan := tracer.StartSpan("tool", SpanTypeTool)
	if toolSpan.ParentID != stepSpan.SpanID {
		t.Errorf("Tool span parent should be step span, got %s", toolSpan.ParentID)
	}

	// Verify trace has all spans
	trace := tracer.GetTrace()
	if len(trace.Spans) != 4 {
		t.Errorf("Expected 4 spans, got %d", len(trace.Spans))
	}
}

func TestTracer_EndSpan(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "test-exp",
	}

	tracer := NewTracer(config)
	tracer.StartTrace("test")

	span := tracer.StartSpan("test-span", SpanTypeAgent)

	// Span should not be ended yet
	if span.EndTimeNs != 0 {
		t.Error("Span should not be ended yet")
	}

	// End the span
	tracer.EndSpan(span)

	// Span should now be ended
	if span.EndTimeNs == 0 {
		t.Error("Span should be ended")
	}

	// Stack should be empty
	if len(tracer.spanStack) != 0 {
		t.Errorf("Span stack should be empty, got %d items", len(tracer.spanStack))
	}
}

func TestTracer_ThreadSafety(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "test-exp",
	}

	tracer := NewTracer(config)
	tracer.StartTrace("test")

	var wg sync.WaitGroup
	numGoroutines := 10

	// Create spans concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			span := tracer.StartSpan("concurrent-span", SpanTypeAgent)
			time.Sleep(1 * time.Millisecond)
			tracer.EndSpan(span)
		}()
	}

	wg.Wait()

	trace := tracer.GetTrace()
	if len(trace.Spans) != numGoroutines {
		t.Errorf("Expected %d spans, got %d", numGoroutines, len(trace.Spans))
	}

	// Stack should be empty
	if len(tracer.spanStack) != 0 {
		t.Errorf("Span stack should be empty after all goroutines complete, got %d", len(tracer.spanStack))
	}
}

func TestTracer_EndOrphanSpans(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "test-exp",
	}

	tracer := NewTracer(config)
	tracer.StartTrace("test")

	// Create some spans but don't end them
	span1 := tracer.StartSpan("span1", SpanTypeAgent)
	span2 := tracer.StartSpan("span2", SpanTypeChain)
	_ = span1
	_ = span2

	// End orphan spans
	tracer.EndOrphanSpans()

	// All spans should be ended
	trace := tracer.GetTrace()
	for _, span := range trace.Spans {
		if span.EndTimeNs == 0 {
			t.Error("Orphan span should be ended")
		}
	}

	// Stack should be cleared
	if len(tracer.spanStack) != 0 {
		t.Error("Span stack should be cleared")
	}
}

func TestTracer_SetPreviews(t *testing.T) {
	config := TracingConfig{
		ExperimentID: "test-exp",
	}

	tracer := NewTracer(config)
	tracer.StartTrace("initial request")

	tracer.SetRequestPreview("updated request")
	tracer.SetResponsePreview("response text")

	trace := tracer.GetTrace()
	if trace.RequestPreview != "updated request" {
		t.Errorf("Expected request preview 'updated request', got '%s'", trace.RequestPreview)
	}
	if trace.ResponsePreview != "response text" {
		t.Errorf("Expected response preview 'response text', got '%s'", trace.ResponsePreview)
	}
}
