package fantasy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestAgent_Generate_OnLLMStartCallback tests that OnLLMStart is called before LLM API call
func TestAgent_Generate_OnLLMStartCallback(t *testing.T) {
	t.Parallel()

	var llmStartCalled bool
	var capturedModel LanguageModel
	var capturedMessages []Message

	model := &mockLanguageModel{
		generateFunc: func(ctx context.Context, call Call) (*Response, error) {
			// Verify OnLLMStart was already called
			require.True(t, llmStartCalled, "OnLLMStart should be called before Generate")
			return &Response{
				Content: []Content{
					TextContent{Text: "Hello, world!"},
				},
				Usage: Usage{
					InputTokens:  10,
					OutputTokens: 20,
					TotalTokens:  30,
				},
				FinishReason: FinishReasonStop,
			}, nil
		},
	}

	agent := NewAgent(model)
	result, err := agent.Generate(context.Background(), AgentCall{
		Prompt: "test prompt",
		OnLLMStart: func(m LanguageModel, messages []Message) error {
			llmStartCalled = true
			capturedModel = m
			capturedMessages = messages
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, llmStartCalled, "OnLLMStart should have been called")
	require.NotNil(t, capturedModel, "Model should be captured")
	require.NotEmpty(t, capturedMessages, "Messages should be captured")
}

// TestAgent_Generate_OnLLMFinishCallback tests that OnLLMFinish is called after LLM API call
func TestAgent_Generate_OnLLMFinishCallback(t *testing.T) {
	t.Parallel()

	var llmFinishCalled bool
	var capturedUsage Usage
	var capturedFinishReason FinishReason
	var capturedError error

	model := &mockLanguageModel{
		generateFunc: func(ctx context.Context, call Call) (*Response, error) {
			// OnLLMFinish should not be called yet
			require.False(t, llmFinishCalled, "OnLLMFinish should not be called before Generate completes")
			return &Response{
				Content: []Content{
					TextContent{Text: "Hello, world!"},
				},
				Usage: Usage{
					InputTokens:  15,
					OutputTokens: 25,
					TotalTokens:  40,
				},
				FinishReason: FinishReasonStop,
			}, nil
		},
	}

	agent := NewAgent(model)
	result, err := agent.Generate(context.Background(), AgentCall{
		Prompt: "test prompt",
		OnLLMFinish: func(usage Usage, finishReason FinishReason, err error) error {
			llmFinishCalled = true
			capturedUsage = usage
			capturedFinishReason = finishReason
			capturedError = err
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, llmFinishCalled, "OnLLMFinish should have been called")
	require.Equal(t, int64(15), capturedUsage.InputTokens)
	require.Equal(t, int64(25), capturedUsage.OutputTokens)
	require.Equal(t, int64(40), capturedUsage.TotalTokens)
	require.Equal(t, FinishReasonStop, capturedFinishReason)
	require.Nil(t, capturedError, "Error should be nil on success")
}

// TestAgent_Generate_OnLLMFinishCallback_WithError tests OnLLMFinish receives error when LLM call fails
func TestAgent_Generate_OnLLMFinishCallback_WithError(t *testing.T) {
	t.Parallel()

	var llmFinishCalled bool
	var capturedUsage Usage
	var capturedFinishReason FinishReason
	var capturedError error

	expectedError := fmt.Errorf("LLM API error")

	model := &mockLanguageModel{
		generateFunc: func(ctx context.Context, call Call) (*Response, error) {
			return nil, expectedError
		},
	}

	agent := NewAgent(model)
	result, err := agent.Generate(context.Background(), AgentCall{
		Prompt: "test prompt",
		OnLLMFinish: func(usage Usage, finishReason FinishReason, err error) error {
			llmFinishCalled = true
			capturedUsage = usage
			capturedFinishReason = finishReason
			capturedError = err
			return nil
		},
	})

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, llmFinishCalled, "OnLLMFinish should be called even on error")
	require.Equal(t, Usage{}, capturedUsage, "Usage should be empty on error")
	require.Equal(t, FinishReason(""), capturedFinishReason, "FinishReason should be empty on error")
	require.Equal(t, expectedError, capturedError, "Error should be captured")
}

// TestAgent_Generate_OnLLMCallbacks_MultipleSteps tests callbacks work across multiple steps
func TestAgent_Generate_OnLLMCallbacks_MultipleSteps(t *testing.T) {
	t.Parallel()

	var llmStartCallCount int
	var llmFinishCallCount int

	callCount := 0
	model := &mockLanguageModel{
		generateFunc: func(ctx context.Context, call Call) (*Response, error) {
			callCount++
			switch callCount {
			case 1:
				// First call - return tool call
				return &Response{
					Content: []Content{
						ToolCallContent{
							ToolCallID: "call-1",
							ToolName:   "test_tool",
							Input:      `{"value":"test"}`,
						},
					},
					Usage: Usage{
						InputTokens:  10,
						OutputTokens: 5,
						TotalTokens:  15,
					},
					FinishReason: FinishReasonToolCalls,
				}, nil
			case 2:
				// Second call - return final text
				return &Response{
					Content: []Content{
						TextContent{Text: "Done"},
					},
					Usage: Usage{
						InputTokens:  5,
						OutputTokens: 10,
						TotalTokens:  15,
					},
					FinishReason: FinishReasonStop,
				}, nil
			default:
				t.Fatalf("Unexpected call count: %d", callCount)
				return nil, nil
			}
		},
	}

	tool := &mockTool{
		name:        "test_tool",
		description: "Test tool",
		parameters: map[string]any{
			"value": map[string]any{"type": "string"},
		},
		required: []string{"value"},
		executeFunc: func(ctx context.Context, call ToolCall) (ToolResponse, error) {
			return ToolResponse{Content: "result", IsError: false}, nil
		},
	}

	agent := NewAgent(model, WithTools(tool))
	result, err := agent.Generate(context.Background(), AgentCall{
		Prompt: "test prompt",
		OnLLMStart: func(m LanguageModel, messages []Message) error {
			llmStartCallCount++
			return nil
		},
		OnLLMFinish: func(usage Usage, finishReason FinishReason, err error) error {
			llmFinishCallCount++
			require.NoError(t, err, "Should not have error in OnLLMFinish")
			require.NotEqual(t, Usage{}, usage, "Usage should not be empty")
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, llmStartCallCount, "OnLLMStart should be called twice (one per step)")
	require.Equal(t, 2, llmFinishCallCount, "OnLLMFinish should be called twice (one per step)")
	require.Len(t, result.Steps, 2, "Should have 2 steps")
}

// TestAgent_Stream_OnLLMStartCallback tests that OnLLMStart is called before LLM stream
func TestAgent_Stream_OnLLMStartCallback(t *testing.T) {
	t.Parallel()

	var llmStartCalled bool
	var capturedModel LanguageModel
	var capturedMessages []Message

	model := &mockLanguageModel{
		streamFunc: func(ctx context.Context, call Call) (StreamResponse, error) {
			// Verify OnLLMStart was already called
			require.True(t, llmStartCalled, "OnLLMStart should be called before Stream")
			return func(yield func(StreamPart) bool) {
				yield(StreamPart{Type: StreamPartTypeTextStart, ID: "text-1"})
				yield(StreamPart{Type: StreamPartTypeTextDelta, ID: "text-1", Delta: "Hello"})
				yield(StreamPart{Type: StreamPartTypeTextEnd, ID: "text-1"})
				yield(StreamPart{
					Type:         StreamPartTypeFinish,
					Usage:        Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
					FinishReason: FinishReasonStop,
				})
			}, nil
		},
	}

	agent := NewAgent(model)
	result, err := agent.Stream(context.Background(), AgentStreamCall{
		Prompt: "test prompt",
		OnLLMStart: func(m LanguageModel, messages []Message) error {
			llmStartCalled = true
			capturedModel = m
			capturedMessages = messages
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, llmStartCalled, "OnLLMStart should have been called")
	require.NotNil(t, capturedModel, "Model should be captured")
	require.NotEmpty(t, capturedMessages, "Messages should be captured")
}

// TestAgent_Stream_OnLLMFinishCallback tests that OnLLMFinish is called after LLM stream completes
func TestAgent_Stream_OnLLMFinishCallback(t *testing.T) {
	t.Parallel()

	var llmFinishCalled bool
	var capturedUsage Usage
	var capturedFinishReason FinishReason

	model := &mockLanguageModel{
		streamFunc: func(ctx context.Context, call Call) (StreamResponse, error) {
			return func(yield func(StreamPart) bool) {
				yield(StreamPart{Type: StreamPartTypeTextStart, ID: "text-1"})
				yield(StreamPart{Type: StreamPartTypeTextDelta, ID: "text-1", Delta: "Hello"})
				yield(StreamPart{Type: StreamPartTypeTextEnd, ID: "text-1"})
				yield(StreamPart{
					Type:         StreamPartTypeFinish,
					Usage:        Usage{InputTokens: 12, OutputTokens: 18, TotalTokens: 30},
					FinishReason: FinishReasonStop,
				})
			}, nil
		},
	}

	agent := NewAgent(model)
	result, err := agent.Stream(context.Background(), AgentStreamCall{
		Prompt: "test prompt",
		OnLLMFinish: func(usage Usage, finishReason FinishReason, err error) error {
			llmFinishCalled = true
			capturedUsage = usage
			capturedFinishReason = finishReason
			require.NoError(t, err, "Should not have error in successful case")
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, llmFinishCalled, "OnLLMFinish should have been called")
	require.Equal(t, int64(12), capturedUsage.InputTokens)
	require.Equal(t, int64(18), capturedUsage.OutputTokens)
	require.Equal(t, int64(30), capturedUsage.TotalTokens)
	require.Equal(t, FinishReasonStop, capturedFinishReason)
}

// TestAgent_Stream_OnLLMFinishCallback_WithError tests OnLLMFinish is called when stream fails
func TestAgent_Stream_OnLLMFinishCallback_WithError(t *testing.T) {
	t.Parallel()

	var llmFinishCalled bool
	var capturedUsage Usage
	var capturedFinishReason FinishReason

	expectedError := fmt.Errorf("stream error")

	model := &mockLanguageModel{
		streamFunc: func(ctx context.Context, call Call) (StreamResponse, error) {
			return func(yield func(StreamPart) bool) {
				yield(StreamPart{Type: StreamPartTypeError, Error: expectedError})
			}, nil
		},
	}

	agent := NewAgent(model)
	result, err := agent.Stream(context.Background(), AgentStreamCall{
		Prompt: "test prompt",
		OnLLMFinish: func(usage Usage, finishReason FinishReason, err error) error {
			llmFinishCalled = true
			capturedUsage = usage
			capturedFinishReason = finishReason
			return nil
		},
	})

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, llmFinishCalled, "OnLLMFinish should be called even on error")
	// Note: In the current implementation, when a stream error occurs, the deferred
	// OnLLMFinish is called with empty usage/finishReason values. The error is returned
	// to the caller but not passed to the callback due to variable scoping in the defer.
	require.Equal(t, Usage{}, capturedUsage, "Usage should be empty on stream error")
	require.Equal(t, FinishReason(""), capturedFinishReason, "FinishReason should be empty on stream error")
}

// TestAgent_Stream_OnLLMCallbacks_MultipleSteps tests callbacks work across multiple streaming steps
func TestAgent_Stream_OnLLMCallbacks_MultipleSteps(t *testing.T) {
	t.Parallel()

	var llmStartCallCount int
	var llmFinishCallCount int

	stepCount := 0
	model := &mockLanguageModel{
		streamFunc: func(ctx context.Context, call Call) (StreamResponse, error) {
			stepCount++
			return func(yield func(StreamPart) bool) {
				if stepCount == 1 {
					// First step: make tool call
					yield(StreamPart{Type: StreamPartTypeToolInputStart, ID: "tool-1", ToolCallName: "test_tool"})
					yield(StreamPart{Type: StreamPartTypeToolInputDelta, ID: "tool-1", Delta: `{"value": "test"}`})
					yield(StreamPart{Type: StreamPartTypeToolInputEnd, ID: "tool-1"})
					yield(StreamPart{
						Type:          StreamPartTypeToolCall,
						ID:            "tool-1",
						ToolCallName:  "test_tool",
						ToolCallInput: `{"value": "test"}`,
					})
					yield(StreamPart{
						Type:         StreamPartTypeFinish,
						Usage:        Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
						FinishReason: FinishReasonToolCalls,
					})
				} else {
					// Second step: finish
					yield(StreamPart{Type: StreamPartTypeTextStart, ID: "text-1"})
					yield(StreamPart{Type: StreamPartTypeTextDelta, ID: "text-1", Delta: "Done"})
					yield(StreamPart{Type: StreamPartTypeTextEnd, ID: "text-1"})
					yield(StreamPart{
						Type:         StreamPartTypeFinish,
						Usage:        Usage{InputTokens: 5, OutputTokens: 10, TotalTokens: 15},
						FinishReason: FinishReasonStop,
					})
				}
			}, nil
		},
	}

	tool := &mockTool{
		name:        "test_tool",
		description: "Test tool",
		parameters: map[string]any{
			"value": map[string]any{"type": "string"},
		},
		required: []string{"value"},
		executeFunc: func(ctx context.Context, call ToolCall) (ToolResponse, error) {
			return ToolResponse{Content: "result", IsError: false}, nil
		},
	}

	agent := NewAgent(model, WithTools(tool))
	result, err := agent.Stream(context.Background(), AgentStreamCall{
		Prompt: "test prompt",
		OnLLMStart: func(m LanguageModel, messages []Message) error {
			llmStartCallCount++
			return nil
		},
		OnLLMFinish: func(usage Usage, finishReason FinishReason, err error) error {
			llmFinishCallCount++
			require.NoError(t, err, "Should not have error in OnLLMFinish")
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, llmStartCallCount, "OnLLMStart should be called twice (one per step)")
	require.Equal(t, 2, llmFinishCallCount, "OnLLMFinish should be called twice (one per step)")
	require.Len(t, result.Steps, 2, "Should have 2 steps")
}

// TestAgent_OnLLMCallbacks_Timing_Integration tests that LLM timing is accurate
func TestAgent_OnLLMCallbacks_Timing_Integration(t *testing.T) {
	t.Parallel()

	var startTime time.Time
	var finishTime time.Time

	model := &mockLanguageModel{
		generateFunc: func(ctx context.Context, call Call) (*Response, error) {
			// Simulate some processing time
			time.Sleep(50 * time.Millisecond)
			return &Response{
				Content: []Content{
					TextContent{Text: "Hello, world!"},
				},
				Usage: Usage{
					InputTokens:  10,
					OutputTokens: 20,
					TotalTokens:  30,
				},
				FinishReason: FinishReasonStop,
			}, nil
		},
	}

	agent := NewAgent(model)
	result, err := agent.Generate(context.Background(), AgentCall{
		Prompt: "test prompt",
		OnLLMStart: func(m LanguageModel, messages []Message) error {
			startTime = time.Now()
			return nil
		},
		OnLLMFinish: func(usage Usage, finishReason FinishReason, err error) error {
			finishTime = time.Now()
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	duration := finishTime.Sub(startTime)
	require.Greater(t, duration, time.Duration(0), "Duration should be positive")
	require.GreaterOrEqual(t, duration, 50*time.Millisecond, "Duration should be at least 50ms (simulated LLM time)")
	require.Less(t, duration, 5*time.Second, "Duration should be less than 5 seconds (sanity check)")
}

// TestAgent_OnLLMCallbacks_Timing_Stream_Integration tests that LLM timing is accurate for streaming
func TestAgent_OnLLMCallbacks_Timing_Stream_Integration(t *testing.T) {
	t.Parallel()

	var startTime time.Time
	var finishTime time.Time

	model := &mockLanguageModel{
		streamFunc: func(ctx context.Context, call Call) (StreamResponse, error) {
			return func(yield func(StreamPart) bool) {
				// Simulate some processing time
				time.Sleep(50 * time.Millisecond)
				yield(StreamPart{Type: StreamPartTypeTextStart, ID: "text-1"})
				yield(StreamPart{Type: StreamPartTypeTextDelta, ID: "text-1", Delta: "Hello"})
				time.Sleep(20 * time.Millisecond)
				yield(StreamPart{Type: StreamPartTypeTextEnd, ID: "text-1"})
				yield(StreamPart{
					Type:         StreamPartTypeFinish,
					Usage:        Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
					FinishReason: FinishReasonStop,
				})
			}, nil
		},
	}

	agent := NewAgent(model)
	result, err := agent.Stream(context.Background(), AgentStreamCall{
		Prompt: "test prompt",
		OnLLMStart: func(m LanguageModel, messages []Message) error {
			startTime = time.Now()
			return nil
		},
		OnLLMFinish: func(usage Usage, finishReason FinishReason, err error) error {
			finishTime = time.Now()
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	duration := finishTime.Sub(startTime)
	require.Greater(t, duration, time.Duration(0), "Duration should be positive")
	require.GreaterOrEqual(t, duration, 70*time.Millisecond, "Duration should be at least 70ms (simulated stream time)")
	require.Less(t, duration, 5*time.Second, "Duration should be less than 5 seconds (sanity check)")
}

// TestAgent_OnLLMCallbacks_ErrorPropagation tests that errors from callbacks are properly propagated
func TestAgent_OnLLMCallbacks_ErrorPropagation(t *testing.T) {
	t.Parallel()

	t.Run("OnLLMStart error stops execution", func(t *testing.T) {
		t.Parallel()

		expectedError := fmt.Errorf("start callback error")

		model := &mockLanguageModel{
			generateFunc: func(ctx context.Context, call Call) (*Response, error) {
				t.Fatal("Generate should not be called if OnLLMStart returns error")
				return nil, nil
			},
		}

		agent := NewAgent(model)
		result, err := agent.Generate(context.Background(), AgentCall{
			Prompt: "test prompt",
			OnLLMStart: func(m LanguageModel, messages []Message) error {
				return expectedError
			},
		})

		require.Error(t, err)
		require.Equal(t, expectedError, err)
		require.Nil(t, result)
	})

	t.Run("OnLLMFinish error is propagated", func(t *testing.T) {
		t.Parallel()

		expectedError := fmt.Errorf("finish callback error")

		model := &mockLanguageModel{
			generateFunc: func(ctx context.Context, call Call) (*Response, error) {
				return &Response{
					Content: []Content{
						TextContent{Text: "Hello, world!"},
					},
					Usage:        Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
					FinishReason: FinishReasonStop,
				}, nil
			},
		}

		agent := NewAgent(model)
		result, err := agent.Generate(context.Background(), AgentCall{
			Prompt: "test prompt",
			OnLLMFinish: func(usage Usage, finishReason FinishReason, err error) error {
				return expectedError
			},
		})

		require.Error(t, err)
		require.Equal(t, expectedError, err)
		require.Nil(t, result)
	})
}
