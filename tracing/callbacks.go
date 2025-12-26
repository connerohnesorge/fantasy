package tracing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"charm.land/fantasy"
)

// TracingWrapper wraps an agent to provide MLflow tracing integration.
type TracingWrapper struct {
	agent  fantasy.Agent
	tracer *Tracer

	// State for tracking spans across callbacks
	mu                sync.RWMutex
	currentTrace      *Trace
	agentSpan         *Span
	stepSpans         map[int]*Span          // step number -> span
	toolSpans         map[string]*Span       // tool call ID -> span
	stepStartTimes    map[int]time.Time      // step number -> start time
	firstToolCallTime map[int]time.Time      // step number -> first tool call time
}

// NewTracingWrapper creates a new tracing wrapper around an agent.
func NewTracingWrapper(agent fantasy.Agent, config TracingConfig) (*TracingWrapper, error) {
	tracer, err := NewTracer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer: %w", err)
	}

	return &TracingWrapper{
		agent:             agent,
		tracer:            tracer,
		stepSpans:         make(map[int]*Span),
		toolSpans:         make(map[string]*Span),
		stepStartTimes:    make(map[int]time.Time),
		firstToolCallTime: make(map[int]time.Time),
	}, nil
}

// Generate implements fantasy.Agent with tracing.
func (w *TracingWrapper) Generate(ctx context.Context, call fantasy.AgentCall) (result *fantasy.AgentResult, err error) {
	// Create trace
	trace, ctx, err := w.tracer.StartTrace(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start trace: %w", err)
	}

	w.mu.Lock()
	w.currentTrace = trace
	w.mu.Unlock()

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			w.handlePanic(ctx, r)
			panic(r) // Re-panic after cleanup
		}
	}()

	// Create agent span
	agentSpan, ctx, err := w.tracer.StartSpan(ctx, w.tracer.agentName, SpanTypeAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to start agent span: %w", err)
	}

	w.mu.Lock()
	w.agentSpan = agentSpan
	w.mu.Unlock()

	// Capture initial prompt
	agentSpan.SetJSONAttribute(AttrSpanInputs, map[string]any{
		"prompt":   call.Prompt,
		"messages": call.Messages,
	})

	// Set request preview
	trace.RequestPreview = truncateUTF8(call.Prompt, MaxPreviewSize)

	// Create callbacks for the call
	streamCall := w.createStreamCall(call, ctx)

	// Execute agent
	result, err = w.agent.Stream(ctx, streamCall)

	// Handle result - check for errors via err or finish reason
	hasError := err != nil

	if result != nil {
		// Set agent span outputs
		agentSpan.SetJSONAttribute(AttrSpanOutputs, map[string]any{
			"response": result.Response,
			"steps":    len(result.Steps),
		})

		// Set response preview
		if len(result.Steps) > 0 {
			lastStep := result.Steps[len(result.Steps)-1]
			trace.ResponsePreview = extractTextPreview(lastStep.Content)
		}
	}

	// End agent span
	if hasError {
		agentSpan.End(SpanStatus{Code: SpanStatusError, Description: "agent execution failed"})
	} else {
		agentSpan.End(SpanStatus{Code: SpanStatusOK})
	}

	// End trace and flush
	if flushErr := w.tracer.EndTrace(ctx, trace, hasError); flushErr != nil {
		log.Printf("[ERROR] tracing: failed to flush trace: %v", flushErr)
	}

	return result, err
}

// Stream implements fantasy.Agent with tracing.
func (w *TracingWrapper) Stream(ctx context.Context, call fantasy.AgentStreamCall) (result *fantasy.AgentResult, err error) {
	// Create trace
	trace, ctx, err := w.tracer.StartTrace(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start trace: %w", err)
	}

	w.mu.Lock()
	w.currentTrace = trace
	w.mu.Unlock()

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			w.handlePanic(ctx, r)
			panic(r) // Re-panic after cleanup
		}
	}()

	// Create agent span
	agentSpan, ctx, err := w.tracer.StartSpan(ctx, w.tracer.agentName, SpanTypeAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to start agent span: %w", err)
	}

	w.mu.Lock()
	w.agentSpan = agentSpan
	w.mu.Unlock()

	// Capture initial prompt
	agentSpan.SetJSONAttribute(AttrSpanInputs, map[string]any{
		"prompt":   call.Prompt,
		"messages": call.Messages,
	})

	// Set request preview
	trace.RequestPreview = truncateUTF8(call.Prompt, MaxPreviewSize)

	// Wrap callbacks
	wrappedCall := w.wrapStreamCallbacks(call, ctx)

	// Execute agent
	result, err = w.agent.Stream(ctx, wrappedCall)

	// Handle result - check for errors via err or finish reason
	hasError := err != nil

	if result != nil {
		// Set agent span outputs
		agentSpan.SetJSONAttribute(AttrSpanOutputs, map[string]any{
			"response": result.Response,
			"steps":    len(result.Steps),
		})

		// Set response preview
		if len(result.Steps) > 0 {
			lastStep := result.Steps[len(result.Steps)-1]
			trace.ResponsePreview = extractTextPreview(lastStep.Content)
		}
	}

	// End agent span
	if hasError {
		agentSpan.End(SpanStatus{Code: SpanStatusError, Description: "agent execution failed"})
	} else {
		agentSpan.End(SpanStatus{Code: SpanStatusOK})
	}

	// End trace and flush
	flushErr := w.tracer.EndTrace(ctx, trace, hasError)
	if flushErr != nil {
		log.Printf("[ERROR] tracing: failed to flush trace: %v", flushErr)
	}

	return result, err
}

// createStreamCall creates a stream call from a regular call with callbacks.
func (w *TracingWrapper) createStreamCall(call fantasy.AgentCall, ctx context.Context) fantasy.AgentStreamCall {
	streamCall := fantasy.AgentStreamCall{
		Prompt:           call.Prompt,
		Files:            call.Files,
		Messages:         call.Messages,
		MaxOutputTokens:  call.MaxOutputTokens,
		Temperature:      call.Temperature,
		TopP:             call.TopP,
		TopK:             call.TopK,
		PresencePenalty:  call.PresencePenalty,
		FrequencyPenalty: call.FrequencyPenalty,
		ActiveTools:      call.ActiveTools,
		ProviderOptions:  call.ProviderOptions,
		OnRetry:          call.OnRetry,
		MaxRetries:       call.MaxRetries,
		StopWhen:         call.StopWhen,
		PrepareStep:      call.PrepareStep,
		RepairToolCall:   call.RepairToolCall,
	}

	return w.wrapStreamCallbacks(streamCall, ctx)
}

// wrapStreamCallbacks wraps the stream call callbacks with tracing.
func (w *TracingWrapper) wrapStreamCallbacks(call fantasy.AgentStreamCall, ctx context.Context) fantasy.AgentStreamCall {
	// Wrap OnStepStart
	originalOnStepStart := call.OnStepStart
	call.OnStepStart = func(stepNumber int) error {
		if err := w.handleStepStart(ctx, stepNumber); err != nil {
			log.Printf("[ERROR] tracing: OnStepStart failed: %v", err)
		}
		if originalOnStepStart != nil {
			return originalOnStepStart(stepNumber)
		}
		return nil
	}

	// Wrap OnStepFinish
	originalOnStepFinish := call.OnStepFinish
	call.OnStepFinish = func(stepResult fantasy.StepResult) error {
		if err := w.handleStepFinish(ctx, stepResult); err != nil {
			log.Printf("[ERROR] tracing: OnStepFinish failed: %v", err)
		}
		if originalOnStepFinish != nil {
			return originalOnStepFinish(stepResult)
		}
		return nil
	}

	// Wrap OnToolCall
	originalOnToolCall := call.OnToolCall
	call.OnToolCall = func(toolCall fantasy.ToolCallContent) error {
		if err := w.handleToolCall(ctx, toolCall); err != nil {
			log.Printf("[ERROR] tracing: OnToolCall failed: %v", err)
		}
		if originalOnToolCall != nil {
			return originalOnToolCall(toolCall)
		}
		return nil
	}

	// Wrap OnToolResult
	originalOnToolResult := call.OnToolResult
	call.OnToolResult = func(result fantasy.ToolResultContent) error {
		if err := w.handleToolResult(ctx, result); err != nil {
			log.Printf("[ERROR] tracing: OnToolResult failed: %v", err)
		}
		if originalOnToolResult != nil {
			return originalOnToolResult(result)
		}
		return nil
	}

	return call
}

// handleStepStart handles the OnStepStart callback.
func (w *TracingWrapper) handleStepStart(ctx context.Context, stepNumber int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Record step start time for LLM span inference
	w.stepStartTimes[stepNumber] = time.Now()

	// Create step span
	stepName := fmt.Sprintf("step-%d", stepNumber+1)

	// Get agent span as parent
	parentCtx := contextWithSpan(ctx, w.agentSpan)

	stepSpan, _, err := w.tracer.StartSpan(parentCtx, stepName, SpanTypeChain)
	if err != nil {
		return fmt.Errorf("failed to start step span: %w", err)
	}

	w.stepSpans[stepNumber] = stepSpan

	return nil
}

// handleStepFinish handles the OnStepFinish callback.
func (w *TracingWrapper) handleStepFinish(ctx context.Context, stepResult fantasy.StepResult) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	stepNumber := len(w.stepSpans) - 1
	stepSpan := w.stepSpans[stepNumber]
	if stepSpan == nil {
		return fmt.Errorf("no step span found for step %d", stepNumber)
	}

	// Capture step outputs
	stepSpan.SetJSONAttribute(AttrSpanOutputs, map[string]any{
		"content":       stepResult.Content,
		"finish_reason": stepResult.FinishReason,
		"usage":         stepResult.Usage,
	})

	// Create inferred LLM span
	if err := w.createInferredLLMSpan(ctx, stepNumber, stepResult); err != nil {
		log.Printf("[ERROR] tracing: failed to create inferred LLM span: %v", err)
	}

	// End step span - check for errors via finish reason
	stepSpan.End(SpanStatus{Code: SpanStatusOK})

	return nil
}

// createInferredLLMSpan creates an LLM span inferred from step timing.
func (w *TracingWrapper) createInferredLLMSpan(ctx context.Context, stepNumber int, stepResult fantasy.StepResult) error {
	stepStartTime := w.stepStartTimes[stepNumber]
	if stepStartTime.IsZero() {
		return fmt.Errorf("no start time recorded for step %d", stepNumber)
	}

	stepSpan := w.stepSpans[stepNumber]
	if stepSpan == nil {
		return fmt.Errorf("no step span found for step %d", stepNumber)
	}

	// Calculate LLM span timing
	llmStartNs := stepStartTime.UnixNano()

	// End time is either first tool call or step end
	llmEndNs := time.Now().UnixNano()
	if firstToolTime, ok := w.firstToolCallTime[stepNumber]; ok {
		llmEndNs = firstToolTime.UnixNano()
	}

	// Create LLM span
	llmSpanID, err := generateSpanID()
	if err != nil {
		return fmt.Errorf("failed to generate LLM span ID: %w", err)
	}

	modelName := w.tracer.modelName
	if modelName == "" {
		modelName = "llm"
	}

	llmSpan := &Span{
		TraceID:     stepSpan.TraceID,
		SpanID:      llmSpanID,
		ParentID:    stepSpan.SpanID, // LLM is child of step
		Name:        modelName,
		SpanType:    SpanTypeLLM,
		StartTimeNs: llmStartNs,
		EndTimeNs:   llmEndNs,
		Attributes:  make(map[string]any),
		Events:      make([]SpanEvent, 0),
		Status:      SpanStatus{Code: SpanStatusOK},
	}

	// Set attributes
	llmSpan.SetAttribute(AttrSpanType, SpanTypeLLM)
	llmSpan.SetAttribute(AttrRequestID, stepSpan.TraceID)

	// Set token usage if available
	if stepResult.Usage.TotalTokens > 0 {
		llmSpan.SetJSONAttribute(AttrTokenUsage, map[string]int64{
			"input_tokens":  stepResult.Usage.InputTokens,
			"output_tokens": stepResult.Usage.OutputTokens,
			"total_tokens":  stepResult.Usage.TotalTokens,
		})
	}

	// Set reasoning content if available
	reasoning := extractReasoning(stepResult.Content)
	if reasoning != "" {
		llmSpan.SetAttribute(AttrLLMReasoning, reasoning)
	}

	// Add to trace
	w.currentTrace.AddSpan(llmSpan)

	return nil
}

// handleToolCall handles the OnToolCall callback.
func (w *TracingWrapper) handleToolCall(ctx context.Context, toolCall fantasy.ToolCallContent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	stepNumber := len(w.stepSpans) - 1

	// Record first tool call time for LLM span inference
	if _, ok := w.firstToolCallTime[stepNumber]; !ok {
		w.firstToolCallTime[stepNumber] = time.Now()
	}

	// Get step span as parent
	stepSpan := w.stepSpans[stepNumber]
	if stepSpan == nil {
		return fmt.Errorf("no step span found for step %d", stepNumber)
	}

	// Create tool span
	parentCtx := contextWithSpan(ctx, stepSpan)
	toolSpan, _, err := w.tracer.StartSpan(parentCtx, toolCall.ToolName, SpanTypeTool)
	if err != nil {
		return fmt.Errorf("failed to start tool span: %w", err)
	}

	// Set function name attribute
	toolSpan.SetAttribute(AttrFunctionName, toolCall.ToolName)

	// Capture tool inputs
	toolSpan.SetJSONAttribute(AttrSpanInputs, map[string]any{
		"tool_name": toolCall.ToolName,
		"input":     toolCall.Input,
	})

	// Store by tool call ID for matching with result
	w.toolSpans[toolCall.ToolCallID] = toolSpan

	return nil
}

// handleToolResult handles the OnToolResult callback.
func (w *TracingWrapper) handleToolResult(ctx context.Context, result fantasy.ToolResultContent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Find tool span by tool call ID
	toolSpan := w.toolSpans[result.ToolCallID]
	if toolSpan == nil {
		return fmt.Errorf("no tool span found for call ID %s", result.ToolCallID)
	}

	// Check if result is an error
	isError := result.Result.GetType() == fantasy.ToolResultContentTypeError

	// Capture tool outputs
	toolSpan.SetJSONAttribute(AttrSpanOutputs, map[string]any{
		"result":   result.Result,
		"is_error": isError,
	})

	// End tool span
	if isError {
		toolSpan.End(SpanStatus{Code: SpanStatusError, Description: "tool execution failed"})
	} else {
		toolSpan.End(SpanStatus{Code: SpanStatusOK})
	}

	return nil
}

// handlePanic handles panic recovery.
func (w *TracingWrapper) handlePanic(ctx context.Context, r any) {
	w.mu.Lock()
	defer w.mu.Unlock()

	log.Printf("[ERROR] tracing: panic during agent execution: %v", r)

	// End all active spans with error
	for _, span := range w.stepSpans {
		if span.EndTimeNs == 0 {
			span.End(SpanStatus{
				Code:        SpanStatusError,
				Description: fmt.Sprintf("panic: %v", r),
			})
		}
	}

	for _, span := range w.toolSpans {
		if span.EndTimeNs == 0 {
			span.End(SpanStatus{
				Code:        SpanStatusError,
				Description: fmt.Sprintf("panic: %v", r),
			})
		}
	}

	if w.agentSpan != nil && w.agentSpan.EndTimeNs == 0 {
		w.agentSpan.End(SpanStatus{
			Code:        SpanStatusError,
			Description: fmt.Sprintf("panic: %v", r),
		})
	}

	// Flush trace
	if w.currentTrace != nil {
		if err := w.tracer.EndTrace(ctx, w.currentTrace, true); err != nil {
			log.Printf("[ERROR] tracing: failed to flush trace after panic: %v", err)
		}
	}
}

// Helper functions

func extractTextPreview(content fantasy.ResponseContent) string {
	for _, c := range content {
		if textContent, ok := c.(fantasy.TextContent); ok {
			return truncateUTF8(textContent.Text, MaxPreviewSize)
		}
	}
	return ""
}

func extractReasoning(content fantasy.ResponseContent) string {
	for _, c := range content {
		if reasoningContent, ok := c.(fantasy.ReasoningContent); ok {
			return reasoningContent.Text
		}
	}
	return ""
}
