package tracing

import (
	"context"
	"encoding/json"
	"sync"
	"unicode/utf8"

	"charm.land/fantasy"
	"charm.land/fantasy/mlflowclient"
)

// maxPreviewLength is the maximum length for request/response preview strings.
const maxPreviewLength = 1000

// TracingCallbacks provides MLflow tracing integration for Fantasy agents.
// It implements the callback pattern used by AgentStreamCall to capture
// trace data during agent execution.
type TracingCallbacks struct {
	tracer *Tracer
	config TracingConfig

	mu            sync.Mutex
	trace         *Trace
	agentSpan     *Span
	currentStep   *Span
	stepLLMSpan   *Span
	stepToolSpans map[string]*Span // keyed by tool call ID

	// Track token usage per step for LLM span attributes
	stepUsage fantasy.Usage
}

// NewTracingCallbacks creates a new TracingCallbacks instance.
func NewTracingCallbacks(tracer *Tracer, config TracingConfig) *TracingCallbacks {
	return &TracingCallbacks{
		tracer:        tracer,
		config:        config,
		stepToolSpans: make(map[string]*Span),
	}
}

// OnAgentStart is called when the agent starts execution.
// Creates the root agent span.
func (tc *TracingCallbacks) OnAgentStart(ctx context.Context, prompt string) (context.Context, error) {
	defer tc.recoverPanic("OnAgentStart")
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Create a new trace
	newCtx, trace, err := tc.tracer.NewTrace(ctx, tc.config)
	if err != nil {
		return ctx, err
	}
	tc.trace = trace

	// Set request preview
	tc.trace.RequestPreview = truncateUTF8(prompt, maxPreviewLength)

	// Create root agent span
	agentName := tc.config.AgentName
	if agentName == "" {
		agentName = "agent"
	}

	span, spanCtx := tc.tracer.StartSpan(newCtx, agentName, SpanTypeAgent)
	if span != nil {
		span.SetAttribute(AttrAgentName, agentName)
		if tc.config.ModelName != "" {
			span.SetAttribute(AttrModelName, tc.config.ModelName)
		}
		tc.agentSpan = span
	}

	return spanCtx, nil
}

// OnAgentFinish is called when the agent completes execution.
// Ends the agent span and flushes the trace.
func (tc *TracingCallbacks) OnAgentFinish(ctx context.Context, result *fantasy.AgentResult) (*TracingResult, error) {
	defer tc.recoverPanic("OnAgentFinish")
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.trace == nil {
		return nil, nil
	}

	// Set response preview from final response
	if result != nil && result.Response.Content != nil {
		responseText := result.Response.Content.Text()
		tc.trace.ResponsePreview = truncateUTF8(responseText, maxPreviewLength)

		// Set total usage on agent span
		if tc.agentSpan != nil {
			tc.agentSpan.SetAttribute(AttrInputTokens, result.TotalUsage.InputTokens)
			tc.agentSpan.SetAttribute(AttrOutputTokens, result.TotalUsage.OutputTokens)
			tc.agentSpan.SetAttribute(AttrTotalTokens, result.TotalUsage.TotalTokens)
		}
	}

	// End agent span
	if tc.agentSpan != nil {
		tc.agentSpan.End()
	}

	// Determine trace state based on final result
	state := TraceStateOK
	if result == nil {
		state = TraceStateError
	}

	// End the trace
	if err := tc.tracer.EndTrace(ctx, state); err != nil {
		return &TracingResult{
			Trace:      tc.trace,
			FlushError: err,
		}, nil
	}

	// Flush the trace
	flushErr := tc.tracer.Flush(ctx, tc.trace)

	return &TracingResult{
		Trace:      tc.trace,
		FlushError: flushErr,
	}, nil
}

// OnAgentError is called when the agent encounters an error.
func (tc *TracingCallbacks) OnAgentError(ctx context.Context, err error) {
	defer tc.recoverPanic("OnAgentError")
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.agentSpan != nil {
		tc.agentSpan.EndWithError(err)
	}

	// End current step if active
	if tc.currentStep != nil {
		tc.currentStep.EndWithError(err)
		tc.currentStep = nil
	}

	if tc.trace != nil {
		_ = tc.tracer.EndTrace(ctx, TraceStateError)
		_ = tc.tracer.Flush(ctx, tc.trace)
	}
}

// OnStepStart is called when a new step starts.
// Creates a chain span for the step.
func (tc *TracingCallbacks) OnStepStart(ctx context.Context, stepNumber int) (context.Context, error) {
	defer tc.recoverPanic("OnStepStart")
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// End previous step if still active
	if tc.currentStep != nil {
		tc.currentStep.End()
	}

	// End previous step's LLM span if active
	if tc.stepLLMSpan != nil {
		tc.stepLLMSpan.End()
		tc.stepLLMSpan = nil
	}

	// Create step span as child of agent span
	stepCtx := ctx
	if tc.agentSpan != nil {
		stepCtx = WithParentSpan(ctx, tc.agentSpan)
	}

	span, newCtx := tc.tracer.StartSpan(stepCtx, "step", SpanTypeChain)
	if span != nil {
		span.SetAttribute(AttrStepNumber, stepNumber)
		tc.currentStep = span
	}

	// Reset step state
	tc.stepToolSpans = make(map[string]*Span)
	tc.stepUsage = fantasy.Usage{}

	// Create inferred LLM span within the step
	// This will capture the LLM call timing
	if span != nil {
		llmSpan, _ := tc.tracer.StartSpan(newCtx, "llm", SpanTypeLLM)
		if llmSpan != nil {
			if tc.config.ModelName != "" {
				llmSpan.SetAttribute(AttrModelName, tc.config.ModelName)
			}
			tc.stepLLMSpan = llmSpan
		}
	}

	return newCtx, nil
}

// OnStepFinish is called when a step completes.
func (tc *TracingCallbacks) OnStepFinish(ctx context.Context, stepResult fantasy.StepResult) error {
	defer tc.recoverPanic("OnStepFinish")
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// End LLM span with usage data
	if tc.stepLLMSpan != nil {
		tc.stepLLMSpan.SetAttribute(AttrInputTokens, stepResult.Usage.InputTokens)
		tc.stepLLMSpan.SetAttribute(AttrOutputTokens, stepResult.Usage.OutputTokens)
		tc.stepLLMSpan.SetAttribute(AttrTotalTokens, stepResult.Usage.TotalTokens)
		if stepResult.Usage.ReasoningTokens > 0 {
			tc.stepLLMSpan.SetAttribute(AttrReasoningTokens, stepResult.Usage.ReasoningTokens)
		}
		tc.stepLLMSpan.End()
		tc.stepLLMSpan = nil
	}

	// End step span
	if tc.currentStep != nil {
		// Add step content as output
		if stepResult.Content != nil {
			text := stepResult.Content.Text()
			if text != "" {
				tc.currentStep.SetAttribute(AttrOutput, truncateUTF8(text, maxPreviewLength))
			}
		}

		// Set finish reason
		tc.currentStep.SetAttribute("finish_reason", string(stepResult.FinishReason))

		tc.currentStep.End()
		tc.currentStep = nil
	}

	return nil
}

// OnToolCall is called when a tool call is initiated.
// Creates a tool span.
func (tc *TracingCallbacks) OnToolCall(ctx context.Context, toolCall fantasy.ToolCallContent) (context.Context, error) {
	defer tc.recoverPanic("OnToolCall")
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Create tool span as child of current step
	toolCtx := ctx
	if tc.currentStep != nil {
		toolCtx = WithParentSpan(ctx, tc.currentStep)
	}

	span, newCtx := tc.tracer.StartSpan(toolCtx, toolCall.ToolName, SpanTypeTool)
	if span != nil {
		span.SetAttribute(AttrToolName, toolCall.ToolName)
		span.SetAttribute(AttrToolCallID, toolCall.ToolCallID)

		// Parse and store input
		var input map[string]any
		if err := json.Unmarshal([]byte(toolCall.Input), &input); err == nil {
			span.SetAttribute(AttrInput, input)
		} else {
			span.SetAttribute(AttrInput, toolCall.Input)
		}

		tc.stepToolSpans[toolCall.ToolCallID] = span
	}

	return newCtx, nil
}

// OnToolResult is called when a tool execution completes.
func (tc *TracingCallbacks) OnToolResult(ctx context.Context, result fantasy.ToolResultContent) error {
	defer tc.recoverPanic("OnToolResult")
	tc.mu.Lock()
	defer tc.mu.Unlock()

	span, exists := tc.stepToolSpans[result.ToolCallID]
	if !exists || span == nil {
		return nil
	}

	// Set output based on result type
	switch r := result.Result.(type) {
	case fantasy.ToolResultOutputContentText:
		span.SetAttribute(AttrOutput, truncateUTF8(r.Text, maxPreviewLength))
	case fantasy.ToolResultOutputContentError:
		if r.Error != nil {
			span.EndWithError(r.Error)
			delete(tc.stepToolSpans, result.ToolCallID)
			return nil
		}
	case fantasy.ToolResultOutputContentMedia:
		span.SetAttribute(AttrOutput, "[media: "+r.MediaType+"]")
	}

	span.End()
	delete(tc.stepToolSpans, result.ToolCallID)
	return nil
}

// StreamCallbacks returns callback functions suitable for use with AgentStreamCall.
// This allows easy integration with the streaming agent API.
func (tc *TracingCallbacks) StreamCallbacks() StreamCallbackFuncs {
	return StreamCallbackFuncs{
		OnAgentStart: func() {
			// Create a background context since OnAgentStart has no context param
			_, _ = tc.OnAgentStart(context.Background(), "")
		},
		OnAgentFinish: func(result *fantasy.AgentResult) error {
			_, err := tc.OnAgentFinish(context.Background(), result)
			return err
		},
		OnStepStart: func(stepNumber int) error {
			_, err := tc.OnStepStart(context.Background(), stepNumber)
			return err
		},
		OnStepFinish: func(stepResult fantasy.StepResult) error {
			return tc.OnStepFinish(context.Background(), stepResult)
		},
		OnToolCall: func(toolCall fantasy.ToolCallContent) error {
			_, err := tc.OnToolCall(context.Background(), toolCall)
			return err
		},
		OnToolResult: func(result fantasy.ToolResultContent) error {
			return tc.OnToolResult(context.Background(), result)
		},
		OnError: func(err error) {
			tc.OnAgentError(context.Background(), err)
		},
	}
}

// StreamCallbackFuncs holds callback functions for AgentStreamCall.
type StreamCallbackFuncs struct {
	OnAgentStart  func()
	OnAgentFinish func(result *fantasy.AgentResult) error
	OnStepStart   func(stepNumber int) error
	OnStepFinish  func(stepResult fantasy.StepResult) error
	OnToolCall    func(toolCall fantasy.ToolCallContent) error
	OnToolResult  func(result fantasy.ToolResultContent) error
	OnError       func(err error)
}

// WrapAgentOptions creates a traced agent using the provided tracing configuration.
// Returns the agent with tracing callbacks installed.
func WrapAgentOptions(client *mlflowclient.Client, config TracingConfig) (*TracingCallbacks, *Tracer) {
	tracer := NewTracer(client, config.ExperimentID)
	if config.FlushTimeout > 0 {
		tracer.flushTimeout = config.FlushTimeout
	}
	callbacks := NewTracingCallbacks(tracer, config)
	return callbacks, tracer
}

// recoverPanic recovers from panics in callback methods and logs them.
// This ensures that tracing errors don't crash the agent.
func (tc *TracingCallbacks) recoverPanic(methodName string) {
	if r := recover(); r != nil {
		// Log the panic but don't propagate it
		// In a production system, this would log to stderr or a logging system
		// For now, we just swallow it to not crash the agent
		_ = r
		_ = methodName
	}
}

// truncateUTF8 safely truncates a string to maxLen bytes, ensuring valid UTF-8.
func truncateUTF8(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Find the last valid UTF-8 boundary before maxLen
	truncated := s[:maxLen]
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}

	// If we lost significant content, add ellipsis
	if len(truncated) > 3 {
		return truncated[:len(truncated)-3] + "..."
	}
	return truncated
}
