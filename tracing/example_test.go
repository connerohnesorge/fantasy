package tracing_test

import (
	"context"
	"time"

	"charm.land/fantasy/mlflow"
	"charm.land/fantasy/tracing"
)

// ExampleTracingCallbacks demonstrates how to use tracing with a Fantasy agent.
func ExampleTracingCallbacks() {
	// Create MLflow client
	client := mlflow.New("http://localhost:5000")

	// Configure tracing
	config := tracing.TracingConfig{
		Client:       client,
		ExperimentID: "my-experiment-id",
		AgentName:    "my-agent",
		ModelName:    "claude-3-opus",
		SessionID:    "session-123",
		FlushTimeout: 10 * time.Second,
		Tags: map[string]string{
			"environment": "production",
			"version":     "1.0.0",
		},
	}

	// Create tracing callbacks
	callbacks := tracing.NewTracingCallbacks(config)

	// In real usage, you would integrate with Fantasy agent like this:
	//
	// agent := fantasy.NewAgent(model)
	// result, err := agent.Stream(ctx, fantasy.AgentStreamCall{
	//     Prompt: "What is the weather in San Francisco?",
	//     OnAgentStart: func() {
	//         callbacks.OnAgentStart("What is the weather in San Francisco?")
	//     },
	//     OnAgentFinish: func(result *fantasy.AgentResult) error {
	//         // Extract response text
	//         responseText := extractResponseText(result)
	//         return callbacks.OnAgentFinish(ctx, responseText)
	//     },
	//     OnStepStart: func(stepNumber int) error {
	//         return callbacks.OnStepStart(stepNumber)
	//     },
	//     OnStepFinish: func(stepResult fantasy.StepResult) error {
	//         return callbacks.OnStepFinish()
	//     },
	//     OnLLMStart: func(model fantasy.LanguageModel, messages []fantasy.Message) error {
	//         return callbacks.OnLLMStart(convertMessages(messages))
	//     },
	//     OnLLMFinish: func(usage fantasy.Usage, finishReason fantasy.FinishReason, err error) error {
	//         return callbacks.OnLLMFinish(
	//             int(usage.InputTokens),
	//             int(usage.OutputTokens),
	//             int(usage.TotalTokens),
	//             err,
	//         )
	//     },
	//     OnToolCall: func(toolCall fantasy.ToolCallContent) error {
	//         return callbacks.OnToolCall(toolCall.ToolName, toolCall.Input)
	//     },
	//     OnToolResult: func(result fantasy.ToolResultContent) error {
	//         return callbacks.OnToolResult(result.Result, nil)
	//     },
	// })

	// After execution, get the tracing result
	tracingResult := callbacks.GetResult()
	if tracingResult != nil && tracingResult.FlushError != nil {
		// Handle flush error
		_ = tracingResult.FlushError
	}

	// The trace has been uploaded to MLflow
	_ = tracingResult
}

// ExampleTracer demonstrates low-level tracer usage.
func ExampleTracer() {
	config := tracing.TracingConfig{
		ExperimentID: "my-experiment",
		AgentName:    "my-agent",
	}

	tracer := tracing.NewTracer(config)

	// Start a trace
	trace := tracer.StartTrace("What is the capital of France?")

	// Create agent span (root)
	agentSpan := tracer.StartSpan("agent", tracing.SpanTypeAgent)
	agentSpan.SetAttribute("agent.name", "my-agent")

	// Create step span
	stepSpan := tracer.StartSpan("step_1", tracing.SpanTypeChain)

	// Create LLM span
	llmSpan := tracer.StartSpan("llm", tracing.SpanTypeLLM)
	llmSpan.SetAttribute(tracing.AttrTokenUsage, tracing.TokenUsage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	})
	llmSpan.SetStatus(tracing.SpanStatusOK, "")
	tracer.EndSpan(llmSpan)

	// End step span
	tracer.EndSpan(stepSpan)

	// Set response preview
	tracer.SetResponsePreview("The capital of France is Paris.")

	// End agent span
	tracer.EndSpan(agentSpan)

	// Finalize trace
	trace.SetState(tracing.TraceStateOK)

	// Convert to protobuf and upload
	ctx := context.Background()
	client := mlflow.New("http://localhost:5000")

	pbTrace, err := tracing.ConvertTrace(trace)
	if err != nil {
		panic(err)
	}

	_, err = client.StartTrace(ctx, pbTrace)
	if err != nil {
		panic(err)
	}
}

// ExampleContextWithSpan demonstrates context-based span propagation.
func ExampleContextWithSpan() {
	config := tracing.TracingConfig{
		ExperimentID: "my-experiment",
	}

	tracer := tracing.NewTracer(config)
	tracer.StartTrace("example")

	// Create a span
	span := tracer.StartSpan("my-operation", tracing.SpanTypeAgent)

	// Store in context
	ctx := tracing.ContextWithSpan(context.Background(), span)

	// Pass context to other functions
	doWork(ctx)

	// End span
	tracer.EndSpan(span)
}

func doWork(ctx context.Context) {
	// Retrieve span from context
	span := tracing.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent("work_started", map[string]any{
			"timestamp": time.Now().Unix(),
		})
	}

	// Do actual work...

	if span != nil {
		span.AddEvent("work_completed", nil)
	}
}
