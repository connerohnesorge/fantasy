package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"charm.land/fantasy"
)

// mockLanguageModel simulates an LLM for testing.
type mockLanguageModel struct {
	response       string
	shouldError    bool
	errorOnAttempt int // Fail on specific attempt (0 = never fail)
	attemptCount   int
}

func (m *mockLanguageModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	m.attemptCount++

	// Simulate transient errors
	if m.shouldError && m.errorOnAttempt > 0 && m.attemptCount == m.errorOnAttempt {
		return nil, &fantasy.ProviderError{
			StatusCode: 503,
			Message:    "Service temporarily unavailable",
		}
	}

	// Parse the prompt to understand what we're judging
	var systemMsg, userMsg string
	for _, msg := range call.Prompt {
		if msg.Role == fantasy.MessageRoleSystem {
			for _, part := range msg.Content {
				if textPart, ok := part.(fantasy.TextPart); ok {
					systemMsg = textPart.Text
				}
			}
		}
		if msg.Role == fantasy.MessageRoleUser {
			for _, part := range msg.Content {
				if textPart, ok := part.(fantasy.TextPart); ok {
					userMsg = textPart.Text
				}
			}
		}
	}

	// Generate appropriate mock response based on context
	var score float64
	var rationale string
	var evidence []string

	// Detect judge type from system message
	if strings.Contains(systemMsg, "correctness") {
		if strings.Contains(userMsg, "Paris") {
			score = 5.0
			rationale = "The output correctly identifies Paris as the capital of France."
			evidence = []string{"Paris is mentioned in the output"}
		} else {
			score = 1.0
			rationale = "The output is incorrect."
		}
	} else if strings.Contains(systemMsg, "relevance") {
		score = 4.0
		rationale = "The response is mostly relevant to the question with minor off-topic elements."
	} else if strings.Contains(systemMsg, "guidelines") {
		score = 3.0
		rationale = "The output partially follows the guidelines but has notable gaps."
	} else if strings.Contains(systemMsg, "grounded") {
		score = 2.0
		rationale = "The output contains some grounded claims but also unsupported content."
		evidence = []string{"Claim X is supported", "Claim Y is not found in context"}
	} else {
		// Default response
		score = 4.0
		rationale = "Good quality output."
	}

	// Use custom response if provided
	if m.response != "" {
		// Create tool call response with the mock JSON
		toolCallID := "call_123"
		toolCall := fantasy.ToolCallContent{
			ToolCallID: toolCallID,
			ToolName:   "json_response",
			Input:      m.response,
		}

		return &fantasy.Response{
			Content: fantasy.ResponseContent{
				toolCall,
			},
			FinishReason: fantasy.FinishReasonToolCalls,
			Usage: fantasy.Usage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		}, nil
	}

	// Build JSON response
	judgeResp := JudgeResponse{
		Score:     score,
		Rationale: rationale,
		Evidence:  evidence,
	}
	responseJSON, _ := json.Marshal(judgeResp)

	// Simulate tool call response (similar to structured output)
	toolCallID := "call_123"
	toolCall := fantasy.ToolCallContent{
		ToolCallID: toolCallID,
		ToolName:   "json_response",
		Input:      string(responseJSON),
	}

	return &fantasy.Response{
		Content: fantasy.ResponseContent{
			toolCall,
		},
		FinishReason: fantasy.FinishReasonToolCalls,
		Usage: fantasy.Usage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
	}, nil
}

func (m *mockLanguageModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, fmt.Errorf("streaming not supported in mock")
}

func (m *mockLanguageModel) GenerateObject(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	m.attemptCount++

	// Simulate transient errors
	if m.shouldError && m.errorOnAttempt > 0 && m.attemptCount == m.errorOnAttempt {
		return nil, &fantasy.ProviderError{
			StatusCode: 503,
			Message:    "Service temporarily unavailable",
		}
	}

	// Parse the system message to determine response
	var systemMsg string
	for _, msg := range call.Prompt {
		if msg.Role == fantasy.MessageRoleSystem {
			for _, part := range msg.Content {
				if textPart, ok := part.(fantasy.TextPart); ok {
					systemMsg = textPart.Text
				}
			}
		}
	}

	// Generate appropriate mock response based on context
	var score float64
	var rationale string
	var evidence []string

	// Use custom response if provided
	if m.response != "" {
		var judgeResp JudgeResponse
		if err := json.Unmarshal([]byte(m.response), &judgeResp); err != nil {
			return nil, fmt.Errorf("failed to parse mock response: %w", err)
		}

		return &fantasy.ObjectResponse{
			Object:       judgeResp,
			RawText:      m.response,
			Usage:        fantasy.Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
			FinishReason: fantasy.FinishReasonStop,
		}, nil
	}

	// Detect judge type from system message
	if strings.Contains(systemMsg, "correctness") {
		score = 5.0
		rationale = "The output is completely correct and matches the expected answer"
		evidence = []string{"Paris is mentioned in the output"}
	} else if strings.Contains(systemMsg, "relevance") {
		score = 4.0
		rationale = "The response is mostly relevant to the question with minor off-topic elements."
	} else if strings.Contains(systemMsg, "guidelines") {
		score = 3.0
		rationale = "The output partially follows the guidelines but has notable gaps."
	} else if strings.Contains(systemMsg, "grounded") {
		score = 2.0
		rationale = "The output contains some grounded claims but also unsupported content."
		evidence = []string{"Claim X is supported", "Claim Y is not found in context"}
	} else {
		// Default response
		score = 4.0
		rationale = "Good quality output."
	}

	judgeResp := JudgeResponse{
		Score:     score,
		Rationale: rationale,
		Evidence:  evidence,
	}

	responseJSON, _ := json.Marshal(judgeResp)

	return &fantasy.ObjectResponse{
		Object:       judgeResp,
		RawText:      string(responseJSON),
		Usage:        fantasy.Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
		FinishReason: fantasy.FinishReasonStop,
	}, nil
}

func (m *mockLanguageModel) StreamObject(ctx context.Context, call fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, fmt.Errorf("object streaming not supported in mock")
}

func (m *mockLanguageModel) Provider() string {
	return "mock"
}

func (m *mockLanguageModel) Model() string {
	return "mock-model"
}

func TestNewLLMJudge(t *testing.T) {
	mock := &mockLanguageModel{}

	judge := NewLLMJudge("test_judge", JudgeConfig{
		Model:          mock,
		SystemPrompt:   "You are a test judge",
		PromptTemplate: "Evaluate: {{.Output}}",
	})

	if judge.Name() != "test_judge" {
		t.Errorf("Expected name 'test_judge', got %q", judge.Name())
	}

	// Check defaults were set
	if judge.config.Temperature == nil {
		t.Error("Expected Temperature to be set with default")
	}
	if *judge.config.Temperature != 0.0 {
		t.Errorf("Expected default temperature 0.0, got %v", *judge.config.Temperature)
	}

	if judge.config.MaxTokens == nil {
		t.Error("Expected MaxTokens to be set with default")
	}
	if *judge.config.MaxTokens != 1024 {
		t.Errorf("Expected default MaxTokens 1024, got %v", *judge.config.MaxTokens)
	}

	if judge.config.MaxRetries != 3 {
		t.Errorf("Expected default MaxRetries 3, got %v", judge.config.MaxRetries)
	}
}

func TestLLMJudge_Score(t *testing.T) {
	tests := []struct {
		name          string
		judgeResponse JudgeResponse
		expectedScore float64 // Normalized 0-1
		expectError   bool
	}{
		{
			name: "perfect score",
			judgeResponse: JudgeResponse{
				Score:     5.0,
				Rationale: "Perfect answer",
			},
			expectedScore: 1.0,
			expectError:   false,
		},
		{
			name: "good score",
			judgeResponse: JudgeResponse{
				Score:     4.0,
				Rationale: "Good answer with minor issues",
			},
			expectedScore: 0.75,
			expectError:   false,
		},
		{
			name: "average score",
			judgeResponse: JudgeResponse{
				Score:     3.0,
				Rationale: "Average answer",
			},
			expectedScore: 0.5,
			expectError:   false,
		},
		{
			name: "poor score",
			judgeResponse: JudgeResponse{
				Score:     2.0,
				Rationale: "Poor answer",
			},
			expectedScore: 0.25,
			expectError:   false,
		},
		{
			name: "worst score",
			judgeResponse: JudgeResponse{
				Score:     1.0,
				Rationale: "Completely wrong",
			},
			expectedScore: 0.0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseJSON, _ := json.Marshal(tt.judgeResponse)
			mock := &mockLanguageModel{
				response: string(responseJSON),
			}

			judge := NewLLMJudge("test", JudgeConfig{
				Model:          mock,
				SystemPrompt:   "Test",
				PromptTemplate: "{{.Output}}",
			})

			score, err := judge.Score(context.Background(), ScorerInput{
				Outputs: map[string]any{
					"output": "Test output",
				},
				Expectations: map[string]any{
					"expected": "Test expected",
				},
			})

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if scoreVal, ok := score.Value.(float64); !ok {
					t.Errorf("Expected float64 score, got %T", score.Value)
				} else if scoreVal != tt.expectedScore {
					t.Errorf("Expected score %v, got %v", tt.expectedScore, scoreVal)
				}

				if score.Rationale != tt.judgeResponse.Rationale {
					t.Errorf("Expected rationale %q, got %q", tt.judgeResponse.Rationale, score.Rationale)
				}
			}
		})
	}
}

func TestLLMJudge_RetryLogic(t *testing.T) {
	t.Run("succeeds after retry", func(t *testing.T) {
		mock := &mockLanguageModel{
			shouldError:    true,
			errorOnAttempt: 1, // Fail on first attempt, succeed on second
		}

		responseJSON, _ := json.Marshal(JudgeResponse{
			Score:     4.0,
			Rationale: "Success after retry",
		})
		mock.response = string(responseJSON)

		judge := NewLLMJudge("test", JudgeConfig{
			Model:          mock,
			SystemPrompt:   "Test",
			PromptTemplate: "{{.Output}}",
			MaxRetries:     3,
		})

		score, err := judge.Score(context.Background(), ScorerInput{
			Outputs: map[string]any{
				"output": "Test",
			},
		})
		if err != nil {
			t.Errorf("Expected success after retry, got error: %v", err)
		}

		if mock.attemptCount < 2 {
			t.Errorf("Expected at least 2 attempts, got %d", mock.attemptCount)
		}

		if score.Value.(float64) != 0.75 { // (4-1)/4 = 0.75
			t.Errorf("Expected normalized score 0.75, got %v", score.Value)
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		mock := &mockLanguageModel{
			shouldError:    true,
			errorOnAttempt: 1,
		}

		judge := NewLLMJudge("test", JudgeConfig{
			Model:          mock,
			SystemPrompt:   "Test",
			PromptTemplate: "{{.Output}}",
			MaxRetries:     1, // Only 1 retry allowed
		})

		_, err := judge.Score(context.Background(), ScorerInput{
			Outputs: map[string]any{
				"output": "Test",
			},
		})

		if err == nil {
			t.Error("Expected error after max retries, got nil")
		}

		if !strings.Contains(err.Error(), "max retries exceeded") {
			t.Errorf("Expected 'max retries exceeded' error, got: %v", err)
		}
	})
}

func TestNewCorrectnessJudge(t *testing.T) {
	mock := &mockLanguageModel{}
	judge := NewCorrectnessJudge(mock)

	if judge.Name() != "correctness" {
		t.Errorf("Expected name 'correctness', got %q", judge.Name())
	}

	if !strings.Contains(judge.config.SystemPrompt, "correctness") {
		t.Error("System prompt should mention correctness")
	}

	// Test scoring
	score, err := judge.Score(context.Background(), ScorerInput{
		Outputs: map[string]any{
			"output": "Paris",
		},
		Expectations: map[string]any{
			"expected": "Paris",
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Mock returns score 5 for Paris, normalized to 1.0
	if score.Value.(float64) != 1.0 {
		t.Errorf("Expected normalized score 1.0, got %v", score.Value)
	}
}

func TestNewGuidelinesJudge(t *testing.T) {
	mock := &mockLanguageModel{}
	guidelines := "Be concise and professional"
	judge := NewGuidelinesJudge(mock, guidelines)

	if judge.Name() != "guidelines" {
		t.Errorf("Expected name 'guidelines', got %q", judge.Name())
	}

	if !strings.Contains(judge.config.PromptTemplate, guidelines) {
		t.Error("Prompt template should include the guidelines")
	}

	// Test scoring
	score, err := judge.Score(context.Background(), ScorerInput{
		Outputs: map[string]any{
			"output": "Brief professional response",
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if score.Value == nil {
		t.Error("Expected non-nil score value")
	}
}

func TestNewRelevanceJudge(t *testing.T) {
	mock := &mockLanguageModel{}
	judge := NewRelevanceJudge(mock)

	if judge.Name() != "relevance" {
		t.Errorf("Expected name 'relevance', got %q", judge.Name())
	}

	if !strings.Contains(judge.config.SystemPrompt, "relevance") {
		t.Error("System prompt should mention relevance")
	}

	// Test scoring
	score, err := judge.Score(context.Background(), ScorerInput{
		Outputs: map[string]any{
			"output": "Relevant response",
		},
		Expectations: map[string]any{
			"expected": "Question context",
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if score.Value == nil {
		t.Error("Expected non-nil score value")
	}
}

func TestNewGroundednessJudge(t *testing.T) {
	mock := &mockLanguageModel{}
	judge := NewGroundednessJudge(mock)

	if judge.Name() != "groundedness" {
		t.Errorf("Expected name 'groundedness', got %q", judge.Name())
	}

	if !strings.Contains(judge.config.SystemPrompt, "grounded") {
		t.Error("System prompt should mention grounding")
	}

	// Test scoring with context
	score, err := judge.Score(context.Background(), ScorerInput{
		Outputs: map[string]any{
			"output": "Based on the context, Paris is the capital.",
		},
		Inputs: map[string]any{
			"context": "France's capital is Paris, known for the Eiffel Tower.",
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if score.Value == nil {
		t.Error("Expected non-nil score value")
	}

	// Check for evidence in metadata
	if score.Metadata != nil {
		if evidence, ok := score.Metadata["evidence"]; ok {
			if evidenceSlice, ok := evidence.([]string); ok && len(evidenceSlice) > 0 {
				t.Logf("Evidence found: %v", evidenceSlice)
			}
		}
	}
}

func TestLLMJudge_PromptTemplate(t *testing.T) {
	mock := &mockLanguageModel{}

	judge := NewLLMJudge("test", JudgeConfig{
		Model:          mock,
		SystemPrompt:   "Test",
		PromptTemplate: "Output: {{.Output}}, Expected: {{.Expected}}, Context: {{.Context}}",
	})

	prompt, err := judge.buildPrompt(ScorerInput{
		Outputs: map[string]any{
			"output": "test output",
		},
		Expectations: map[string]any{
			"expected": "test expected",
		},
		Inputs: map[string]any{
			"context": "test context",
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !strings.Contains(prompt, "test output") {
		t.Error("Prompt should contain output")
	}
	if !strings.Contains(prompt, "test expected") {
		t.Error("Prompt should contain expected")
	}
	if !strings.Contains(prompt, "test context") {
		t.Error("Prompt should contain context")
	}
}

func TestLLMJudge_InvalidScore(t *testing.T) {
	// Test score outside valid range
	responseJSON, _ := json.Marshal(JudgeResponse{
		Score:     10.0, // Invalid: must be 1-5
		Rationale: "Invalid score",
	})

	mock := &mockLanguageModel{
		response: string(responseJSON),
	}

	judge := NewLLMJudge("test", JudgeConfig{
		Model:          mock,
		SystemPrompt:   "Test",
		PromptTemplate: "{{.Output}}",
	})

	_, err := judge.Score(context.Background(), ScorerInput{
		Outputs: map[string]any{
			"output": "Test",
		},
	})

	if err == nil {
		t.Error("Expected error for invalid score, got nil")
	}

	if !strings.Contains(err.Error(), "invalid score") {
		t.Errorf("Expected 'invalid score' error, got: %v", err)
	}
}
