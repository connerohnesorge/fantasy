package tracing

import (
	"context"
	"encoding/json"
)

// AgentStreamCallbacksAdapter wraps TracingCallbacks to provide Fantasy-compatible callbacks.
// This allows tracing to be integrated with Fantasy agents via the AgentStreamCall.
//
// Usage:
//
//	callbacks := tracing.NewTracingCallbacks(config)
//	adapter := callbacks.ToAgentCallbacks()
//
//	result, err := agent.Stream(ctx, fantasy.AgentStreamCall{
//	    Prompt: "Hello",
//	    OnAgentStart: adapter.OnAgentStart,
//	    OnAgentFinish: adapter.OnAgentFinish,
//	    OnStepStart: adapter.OnStepStart,
//	    OnStepFinish: adapter.OnStepFinish,
//	    OnLLMStart: adapter.OnLLMStart,
//	    OnLLMFinish: adapter.OnLLMFinish,
//	    OnToolCall: adapter.OnToolCall,
//	    OnToolResult: adapter.OnToolResult,
//	})
//
//	tracingResult := callbacks.GetResult()
type AgentStreamCallbacksAdapter struct {
	tc *Callbacks
}

// ToAgentCallbacks converts TracingCallbacks to a Fantasy-compatible adapter.
func (tc *Callbacks) ToAgentCallbacks() *AgentStreamCallbacksAdapter {
	return &AgentStreamCallbacksAdapter{tc: tc}
}

// OnAgentStartAdapter is the adapter for OnAgentStart.
// Note: Fantasy's OnAgentStart doesn't receive the request, so we use the first
// message from OnStepStart or OnLLMStart to capture it.
func (a *AgentStreamCallbacksAdapter) OnAgentStartAdapter() func() {
	return func() {
		// We don't have the request yet, so start with empty
		a.tc.OnAgentStart("")
	}
}

// OnAgentFinishAdapter is the adapter for OnAgentFinish.
// It extracts the response text from the AgentResult.
func (a *AgentStreamCallbacksAdapter) OnAgentFinishAdapter() func(result interface{}) error {
	return func(result interface{}) error {
		// Extract response text from result
		responseText := ""

		// Try to extract text content
		// This is a simplified extraction - in real usage, you'd need to
		// import the fantasy package and do proper type assertion
		if resultJSON, err := json.Marshal(result); err == nil {
			responseText = string(resultJSON)
		}

		return a.tc.OnAgentFinish(context.Background(), responseText)
	}
}

// OnStepStartAdapter is the adapter for OnStepStart.
func (a *AgentStreamCallbacksAdapter) OnStepStartAdapter() func(stepNumber int) error {
	return func(stepNumber int) error {
		return a.tc.OnStepStart(stepNumber)
	}
}

// OnStepFinishAdapter is the adapter for OnStepFinish.
func (a *AgentStreamCallbacksAdapter) OnStepFinishAdapter() func(stepResult interface{}) error {
	return func(stepResult interface{}) error {
		return a.tc.OnStepFinish()
	}
}

// OnLLMStartAdapter is the adapter for OnLLMStart.
// It converts Fantasy's messages to a generic representation.
func (a *AgentStreamCallbacksAdapter) OnLLMStartAdapter() func(model interface{}, messages interface{}) error {
	return func(model interface{}, messages interface{}) error {
		// Convert messages to []any
		var msgArray []any

		// If we can marshal/unmarshal, do it
		if data, err := json.Marshal(messages); err == nil {
			_ = json.Unmarshal(data, &msgArray)
		}

		// Update request preview if this is the first LLM call
		if a.tc.tracer.GetTrace() != nil {
			trace := a.tc.tracer.GetTrace()
			if trace.RequestPreview == "" {
				if data, err := json.Marshal(messages); err == nil {
					a.tc.tracer.SetRequestPreview(string(data))
				}
			}
		}

		return a.tc.OnLLMStart(msgArray)
	}
}

// OnLLMFinishAdapter is the adapter for OnLLMFinish.
// It extracts token usage from Fantasy's Usage struct.
func (a *AgentStreamCallbacksAdapter) OnLLMFinishAdapter() func(usage interface{}, finishReason interface{}, err error) error {
	return func(usage interface{}, finishReason interface{}, err error) error {
		// Extract token counts from usage
		var inputTokens, outputTokens, totalTokens int

		// Try to extract via JSON marshaling
		if usageJSON, jsonErr := json.Marshal(usage); jsonErr == nil {
			var usageMap map[string]interface{}
			if jsonErr := json.Unmarshal(usageJSON, &usageMap); jsonErr == nil {
				if val, ok := usageMap["input_tokens"].(float64); ok {
					inputTokens = int(val)
				}
				if val, ok := usageMap["output_tokens"].(float64); ok {
					outputTokens = int(val)
				}
				if val, ok := usageMap["total_tokens"].(float64); ok {
					totalTokens = int(val)
				}
			}
		}

		return a.tc.OnLLMFinish(inputTokens, outputTokens, totalTokens, err)
	}
}

// OnToolCallAdapter is the adapter for OnToolCall.
func (a *AgentStreamCallbacksAdapter) OnToolCallAdapter() func(toolCall interface{}) error {
	return func(toolCall interface{}) error {
		// Extract tool name and input
		var toolName string
		var input any

		// Try to extract via JSON
		if toolJSON, err := json.Marshal(toolCall); err == nil {
			var toolMap map[string]interface{}
			if err := json.Unmarshal(toolJSON, &toolMap); err == nil {
				if name, ok := toolMap["tool_name"].(string); ok {
					toolName = name
				}
				if inp, ok := toolMap["input"]; ok {
					input = inp
				}
			}
		}

		return a.tc.OnToolCall(toolName, input)
	}
}

// OnToolResultAdapter is the adapter for OnToolResult.
func (a *AgentStreamCallbacksAdapter) OnToolResultAdapter() func(result interface{}) error {
	return func(result interface{}) error {
		// Extract result content
		var resultContent any

		// Try to extract via JSON
		if resultJSON, err := json.Marshal(result); err == nil {
			var resultMap map[string]interface{}
			if err := json.Unmarshal(resultJSON, &resultMap); err == nil {
				if content, ok := resultMap["result"]; ok {
					resultContent = content
				} else if content, ok := resultMap["content"]; ok {
					resultContent = content
				} else {
					resultContent = resultMap
				}
			}
		}

		// Check for error
		var resultErr error
		if resultJSON, err := json.Marshal(result); err == nil {
			var resultMap map[string]interface{}
			if err := json.Unmarshal(resultJSON, &resultMap); err == nil {
				if errStr, ok := resultMap["error"].(string); ok && errStr != "" {
					resultErr = &genericError{msg: errStr}
				}
			}
		}

		return a.tc.OnToolResult(resultContent, resultErr)
	}
}

// genericError is a simple error type for error strings.
type genericError struct {
	msg string
}

func (e *genericError) Error() string {
	return e.msg
}
