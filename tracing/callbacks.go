package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"charm.land/fantasy/mlflow"
	pb "charm.land/fantasy/proto/gen/mlflow"
)

// Callbacks implements Fantasy's callback interface for MLflow tracing.
type Callbacks struct {
	tracer       *Tracer
	client       *mlflow.Client
	experimentID string
	agentSpan    *Span
	currentStep  *Span
	mu           sync.Mutex
	result       *TracingResult
}

// NewTracingCallbacks creates a new tracing callbacks instance.
func NewTracingCallbacks(config TracingConfig) *Callbacks {
	if config.FlushTimeout == 0 {
		config.FlushTimeout = 10 * time.Second
	}

	// Extract the mlflow client from any type
	client, ok := config.Client.(*mlflow.Client)
	if !ok {
		panic("config.Client must be *mlflow.Client")
	}

	return &Callbacks{
		tracer:       NewTracer(config),
		client:       client,
		experimentID: config.ExperimentID,
	}
}

// OnAgentStart is called when the agent starts execution.
func (tc *Callbacks) OnAgentStart(request string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Start the trace
	tc.tracer.StartTrace(request)

	// Create the root agent span
	tc.agentSpan = tc.tracer.StartSpan("agent", SpanTypeAgent)
}

// OnAgentFinish is called when the agent finishes execution.
func (tc *Callbacks) OnAgentFinish(ctx context.Context, response string) error {
	tc.mu.Lock()

	// End the agent span
	if tc.agentSpan != nil {
		tc.agentSpan.End()
		tc.agentSpan = nil
	}

	// Clean up any orphan spans
	tc.tracer.EndOrphanSpans()

	// Set response preview
	tc.tracer.SetResponsePreview(response)

	// Calculate execution duration
	trace := tc.tracer.GetTrace()
	if trace != nil {
		trace.mu.Lock()
		trace.ExecutionDuration = (nowNanos() - (trace.RequestTime * 1_000_000)) / 1_000_000 // Convert to milliseconds
		trace.SetState(TraceStateOK)
		trace.mu.Unlock()
	}

	tc.mu.Unlock()

	// Flush to MLflow
	return tc.flush(ctx)
}

// OnStepStart is called when a step starts.
func (tc *Callbacks) OnStepStart(stepNumber int) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Create a step span as a child of the agent span
	stepName := fmt.Sprintf("step_%d", stepNumber)
	tc.currentStep = tc.tracer.StartSpan(stepName, SpanTypeChain)

	return nil
}

// OnStepFinish is called when a step finishes.
func (tc *Callbacks) OnStepFinish() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.currentStep != nil {
		tc.tracer.EndSpan(tc.currentStep)
		tc.currentStep = nil
	}

	return nil
}

// OnLLMStart is called before an LLM API call.
func (tc *Callbacks) OnLLMStart(messages []any) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Create an LLM span as a child of the current step
	span := tc.tracer.StartSpan("llm", SpanTypeLLM)

	// Marshal messages to JSON
	messagesJSON, err := json.Marshal(messages)
	if err == nil {
		span.SetAttribute(AttrSpanInputs, string(messagesJSON))
	}

	return nil
}

// OnLLMFinish is called after an LLM API call completes.
func (tc *Callbacks) OnLLMFinish(inputTokens, outputTokens, totalTokens int, err error) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	span := tc.tracer.GetCurrentSpan()
	if span != nil && span.SpanType == SpanTypeLLM {
		// Set token usage
		span.SetAttribute(AttrTokenUsage, TokenUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
		})

		// Set status
		if err != nil {
			span.SetStatus(SpanStatusError, err.Error())
		} else {
			span.SetStatus(SpanStatusOK, "")
		}

		tc.tracer.EndSpan(span)
	}

	return nil
}

// OnToolCall is called when a tool is invoked.
func (tc *Callbacks) OnToolCall(toolName string, input any) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Create a tool span as a child of the current step
	span := tc.tracer.StartSpan(toolName, SpanTypeTool)

	// Marshal input to JSON
	inputJSON, err := json.Marshal(input)
	if err == nil {
		span.SetAttribute(AttrSpanInputs, string(inputJSON))
	}

	return nil
}

// OnToolResult is called when a tool execution completes.
func (tc *Callbacks) OnToolResult(result any, err error) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	span := tc.tracer.GetCurrentSpan()
	if span != nil && span.SpanType == SpanTypeTool {
		// Marshal result to JSON
		resultJSON, jsonErr := json.Marshal(result)
		if jsonErr == nil {
			span.SetAttribute(AttrSpanOutputs, string(resultJSON))
		}

		// Set status
		if err != nil {
			span.SetStatus(SpanStatusError, err.Error())
		} else {
			span.SetStatus(SpanStatusOK, "")
		}

		tc.tracer.EndSpan(span)
	}

	return nil
}

// GetResult returns the tracing result after the agent completes.
func (tc *Callbacks) GetResult() *TracingResult {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.result
}

// flush uploads the trace to MLflow.
func (tc *Callbacks) flush(ctx context.Context) error {
	trace := tc.tracer.GetTrace()
	if trace == nil {
		return nil
	}

	// Create a timeout context for the flush
	flushCtx, cancel := context.WithTimeout(ctx, tc.tracer.config.FlushTimeout)
	defer cancel()

	// Convert to protobuf
	pbTrace, err := TraceToPb(trace)
	if err != nil {
		tc.result = &TracingResult{
			Trace:      trace,
			FlushError: fmt.Errorf("failed to convert trace to protobuf: %w", err),
		}
		return tc.result.FlushError
	}

	// Upload to MLflow
	traceID, err := tc.client.StartTrace(flushCtx, pbTrace)
	if err != nil {
		tc.result = &TracingResult{
			Trace:      trace,
			FlushError: fmt.Errorf("failed to upload trace to MLflow: %w", err),
		}
		return tc.result.FlushError
	}

	// Success
	tc.result = &TracingResult{
		Trace:      trace,
		FlushError: nil,
	}

	// Log trace ID for debugging
	_ = traceID // Trace ID is returned for reference

	return nil
}

// safeCall wraps a callback function with panic recovery.
func (tc *Callbacks) safeCall(name string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in callback %s: %v", name, r)
		}
	}()
	return fn()
}

// FlushAll flushes all pending traces from multiple tracers.
func FlushAll(ctx context.Context, tracers []*Tracer, client *mlflow.Client) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(tracers))

	for _, tracer := range tracers {
		wg.Add(1)
		go func(t *Tracer) {
			defer wg.Done()

			trace := t.GetTrace()
			if trace == nil {
				return
			}

			// Convert to protobuf
			pbTrace, err := TraceToPb(trace)
			if err != nil {
				errChan <- fmt.Errorf("failed to convert trace: %w", err)
				return
			}

			// Upload to MLflow
			_, err = client.StartTrace(ctx, pbTrace)
			if err != nil {
				errChan <- fmt.Errorf("failed to upload trace: %w", err)
			}
		}(tracer)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("flush errors: %v", errors)
	}

	return nil
}

// TraceToPb converts a tracing.Trace to protobuf format for MLflow.
// This is a placeholder - the actual conversion will be in converter.go.
func TraceToPb(trace *Trace) (*pb.Trace, error) {
	// This will be implemented in converter.go
	return ConvertTrace(trace)
}
