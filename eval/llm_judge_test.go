package eval

import (
	"context"
	"fmt"
	"testing"
	"time"

	"charm.land/fantasy"
)

// mockLanguageModel is a mock implementation of fantasy.LanguageModel for testing.
type mockLanguageModel struct {
	generateObjectFunc func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error)
	provider           string
	model              string
}

func (m *mockLanguageModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	return nil, fmt.Errorf("Generate not implemented in mock")
}

func (m *mockLanguageModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, fmt.Errorf("Stream not implemented in mock")
}

func (m *mockLanguageModel) GenerateObject(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	if m.generateObjectFunc != nil {
		return m.generateObjectFunc(ctx, call)
	}
	return nil, fmt.Errorf("GenerateObject not implemented in mock")
}

func (m *mockLanguageModel) StreamObject(ctx context.Context, call fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, fmt.Errorf("StreamObject not implemented in mock")
}

func (m *mockLanguageModel) Provider() string {
	if m.provider != "" {
		return m.provider
	}
	return "mock"
}

func (m *mockLanguageModel) Model() string {
	if m.model != "" {
		return m.model
	}
	return "mock-model"
}

// TestCorrectnessScorer tests the Correctness scorer with mocked LLM.
func TestCorrectnessScorer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          ScorerInput
		mockResponse   map[string]any
		mockError      error
		expectedValue  bool
		expectedError  bool
		expectRationale string
	}{
		{
			name: "correct answer - yes",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "What is the capital of France?",
				},
				Outputs: map[string]any{
					"output": "Paris",
				},
				Expectations: map[string]any{
					"expected_answer": "Paris",
				},
			},
			mockResponse: map[string]any{
				"rationale": "Let's think step by step. The claim states 'Paris' and the document confirms 'Paris' as the answer to the question about France's capital.",
				"result":    "yes",
			},
			expectedValue:   true,
			expectedError:   false,
			expectRationale: "Let's think step by step",
		},
		{
			name: "incorrect answer - no",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "What is the capital of France?",
				},
				Outputs: map[string]any{
					"output": "London",
				},
				Expectations: map[string]any{
					"expected_answer": "Paris",
				},
			},
			mockResponse: map[string]any{
				"rationale": "Let's think step by step. The claim states 'Paris' but the document shows 'London', which contradicts the expected answer.",
				"result":    "no",
			},
			expectedValue:   false,
			expectedError:   false,
			expectRationale: "Let's think step by step",
		},
		{
			name: "missing input field",
			input: ScorerInput{
				Outputs: map[string]any{
					"output": "Paris",
				},
				Expectations: map[string]any{
					"expected_answer": "Paris",
				},
			},
			expectedError: true,
		},
		{
			name: "missing output field",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "What is the capital of France?",
				},
				Expectations: map[string]any{
					"expected_answer": "Paris",
				},
			},
			expectedError: true,
		},
		{
			name: "missing expected_answer field",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "What is the capital of France?",
				},
				Outputs: map[string]any{
					"output": "Paris",
				},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLanguageModel{
				generateObjectFunc: func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return &fantasy.ObjectResponse{
						Object: tt.mockResponse,
					}, nil
				},
			}

			scorer := NewCorrectness(mock).WithRetryConfig(RetryConfig{
				MaxRetries:   0, // No retries for tests
				InitialDelay: 0,
				Multiplier:   1.0,
				MaxDelay:     0,
			})

			score, err := scorer.Score(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("Score() returned unexpected error: %v", err)
			}

			if tt.expectedError {
				if score.Error == nil {
					t.Errorf("Expected error in Score.Error, got nil")
				}
			} else {
				if score.Error != nil {
					t.Errorf("Unexpected error in Score.Error: %v", score.Error)
				}
				if score.Value != tt.expectedValue {
					t.Errorf("Expected value %v, got %v", tt.expectedValue, score.Value)
				}
				if tt.expectRationale != "" && score.Rationale == "" {
					t.Errorf("Expected non-empty rationale")
				}
			}
		})
	}
}

// TestGuidelinesScorer tests the Guidelines scorer.
func TestGuidelinesScorer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          ScorerInput
		mockResponse   map[string]any
		expectedValue  bool
		expectedError  bool
	}{
		{
			name: "follows all guidelines - yes",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "Write a polite email",
				},
				Outputs: map[string]any{
					"output": "Dear Sir/Madam, Thank you for your time. Best regards.",
				},
				Expectations: map[string]any{
					"guidelines": []string{
						"Must be polite",
						"Must include greeting",
						"Must include closing",
					},
				},
			},
			mockResponse: map[string]any{
				"rationale": "Let's think step by step. The output includes a polite greeting 'Dear Sir/Madam', shows politeness throughout, and has a proper closing 'Best regards'. All guidelines are satisfied.",
				"result":    "yes",
			},
			expectedValue: true,
			expectedError: false,
		},
		{
			name: "violates guidelines - no",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "Write a polite email",
				},
				Outputs: map[string]any{
					"output": "Hey, thanks.",
				},
				Expectations: map[string]any{
					"guidelines": []string{
						"Must be polite",
						"Must include greeting",
						"Must include closing",
					},
				},
			},
			mockResponse: map[string]any{
				"rationale": "Let's think step by step. The output uses casual language 'Hey' which violates the politeness guideline. The closing is too brief.",
				"result":    "no",
			},
			expectedValue: false,
			expectedError: false,
		},
		{
			name: "guidelines as string",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "Test",
				},
				Outputs: map[string]any{
					"output": "Test output",
				},
				Expectations: map[string]any{
					"guidelines": "Must be brief",
				},
			},
			mockResponse: map[string]any{
				"rationale": "Output is brief",
				"result":    "yes",
			},
			expectedValue: true,
			expectedError: false,
		},
		{
			name: "missing guidelines field",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "Test",
				},
				Outputs: map[string]any{
					"output": "Test output",
				},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLanguageModel{
				generateObjectFunc: func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
					return &fantasy.ObjectResponse{
						Object: tt.mockResponse,
					}, nil
				},
			}

			scorer := NewGuidelines(mock).WithRetryConfig(RetryConfig{
				MaxRetries:   0,
				InitialDelay: 0,
				Multiplier:   1.0,
				MaxDelay:     0,
			})

			score, err := scorer.Score(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("Score() returned unexpected error: %v", err)
			}

			if tt.expectedError {
				if score.Error == nil {
					t.Errorf("Expected error in Score.Error, got nil")
				}
			} else {
				if score.Error != nil {
					t.Errorf("Unexpected error in Score.Error: %v", score.Error)
				}
				if score.Value != tt.expectedValue {
					t.Errorf("Expected value %v, got %v", tt.expectedValue, score.Value)
				}
			}
		})
	}
}

// TestRelevanceScorer tests the Relevance scorer.
func TestRelevanceScorer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         ScorerInput
		mockResponse  map[string]any
		expectedValue bool
		expectedError bool
	}{
		{
			name: "relevant answer - yes",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "What is machine learning?",
				},
				Outputs: map[string]any{
					"output": "Machine learning is a subset of AI that enables systems to learn from data.",
				},
			},
			mockResponse: map[string]any{
				"rationale": "Let's think step by step. The answer directly addresses the question about machine learning.",
				"result":    "yes",
			},
			expectedValue: true,
			expectedError: false,
		},
		{
			name: "irrelevant answer - no",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "What is machine learning?",
				},
				Outputs: map[string]any{
					"output": "The weather is nice today.",
				},
			},
			mockResponse: map[string]any{
				"rationale": "Let's think step by step. The answer talks about weather, which is completely unrelated to machine learning.",
				"result":    "no",
			},
			expectedValue: false,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLanguageModel{
				generateObjectFunc: func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
					return &fantasy.ObjectResponse{
						Object: tt.mockResponse,
					}, nil
				},
			}

			scorer := NewRelevance(mock).WithRetryConfig(RetryConfig{
				MaxRetries:   0,
				InitialDelay: 0,
				Multiplier:   1.0,
				MaxDelay:     0,
			})

			score, err := scorer.Score(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("Score() returned unexpected error: %v", err)
			}

			if tt.expectedError {
				if score.Error == nil {
					t.Errorf("Expected error in Score.Error, got nil")
				}
			} else {
				if score.Error != nil {
					t.Errorf("Unexpected error in Score.Error: %v", score.Error)
				}
				if score.Value != tt.expectedValue {
					t.Errorf("Expected value %v, got %v", tt.expectedValue, score.Value)
				}
			}
		})
	}
}

// TestGroundednessScorer tests the Groundedness scorer.
func TestGroundednessScorer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         ScorerInput
		mockResponse  map[string]any
		expectedValue bool
		expectedError bool
	}{
		{
			name: "grounded in context - yes",
			input: ScorerInput{
				Inputs: map[string]any{
					"input":             "Who founded Microsoft?",
					"retrieval_context": "Microsoft was founded by Bill Gates and Paul Allen in 1975.",
				},
				Outputs: map[string]any{
					"output": "Bill Gates and Paul Allen founded Microsoft.",
				},
			},
			mockResponse: map[string]any{
				"rationale": "Let's think step by step. The claim that 'Bill Gates and Paul Allen founded Microsoft' is directly supported by the document.",
				"result":    "yes",
			},
			expectedValue: true,
			expectedError: false,
		},
		{
			name: "not grounded in context - no",
			input: ScorerInput{
				Inputs: map[string]any{
					"input":             "Who founded Microsoft?",
					"retrieval_context": "Microsoft was founded by Bill Gates and Paul Allen in 1975.",
				},
				Outputs: map[string]any{
					"output": "Steve Jobs founded Microsoft.",
				},
			},
			mockResponse: map[string]any{
				"rationale": "Let's think step by step. The claim says 'Steve Jobs founded Microsoft', but the document states it was Bill Gates and Paul Allen. Not grounded.",
				"result":    "no",
			},
			expectedValue: false,
			expectedError: false,
		},
		{
			name: "missing retrieval_context",
			input: ScorerInput{
				Inputs: map[string]any{
					"input": "Who founded Microsoft?",
				},
				Outputs: map[string]any{
					"output": "Bill Gates",
				},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLanguageModel{
				generateObjectFunc: func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
					return &fantasy.ObjectResponse{
						Object: tt.mockResponse,
					}, nil
				},
			}

			scorer := NewGroundedness(mock).WithRetryConfig(RetryConfig{
				MaxRetries:   0,
				InitialDelay: 0,
				Multiplier:   1.0,
				MaxDelay:     0,
			})

			score, err := scorer.Score(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("Score() returned unexpected error: %v", err)
			}

			if tt.expectedError {
				if score.Error == nil {
					t.Errorf("Expected error in Score.Error, got nil")
				}
			} else {
				if score.Error != nil {
					t.Errorf("Unexpected error in Score.Error: %v", score.Error)
				}
				if score.Value != tt.expectedValue {
					t.Errorf("Expected value %v, got %v", tt.expectedValue, score.Value)
				}
			}
		})
	}
}

// TestRetryLogic tests the retry mechanism with exponential backoff.
func TestRetryLogic(t *testing.T) {
	t.Parallel()

	t.Run("retries on rate limit error", func(t *testing.T) {
		attempts := 0
		mock := &mockLanguageModel{
			generateObjectFunc: func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
				attempts++
				if attempts < 3 {
					return nil, fmt.Errorf("rate limit exceeded: 429")
				}
				return &fantasy.ObjectResponse{
					Object: map[string]any{
						"rationale": "Success after retries",
						"result":    "yes",
					},
				}, nil
			},
		}

		scorer := NewCorrectness(mock).WithRetryConfig(RetryConfig{
			MaxRetries:   3,
			InitialDelay: 10 * time.Millisecond,
			Multiplier:   2.0,
			MaxDelay:     100 * time.Millisecond,
		})

		input := ScorerInput{
			Inputs: map[string]any{
				"input": "test",
			},
			Outputs: map[string]any{
				"output": "test",
			},
			Expectations: map[string]any{
				"expected_answer": "test",
			},
		}

		score, err := scorer.Score(context.Background(), input)
		if err != nil {
			t.Fatalf("Score() returned error: %v", err)
		}

		if score.Error != nil {
			t.Errorf("Expected success after retries, got error: %v", score.Error)
		}

		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("fails after max retries", func(t *testing.T) {
		mock := &mockLanguageModel{
			generateObjectFunc: func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
				return nil, fmt.Errorf("server error: 500")
			},
		}

		scorer := NewCorrectness(mock).WithRetryConfig(RetryConfig{
			MaxRetries:   2,
			InitialDelay: 10 * time.Millisecond,
			Multiplier:   2.0,
			MaxDelay:     100 * time.Millisecond,
		})

		input := ScorerInput{
			Inputs: map[string]any{
				"input": "test",
			},
			Outputs: map[string]any{
				"output": "test",
			},
			Expectations: map[string]any{
				"expected_answer": "test",
			},
		}

		score, err := scorer.Score(context.Background(), input)
		if err != nil {
			t.Fatalf("Score() returned error: %v", err)
		}

		if score.Error == nil {
			t.Errorf("Expected error after max retries, got nil")
		}
	})

	t.Run("does not retry on client error", func(t *testing.T) {
		attempts := 0
		mock := &mockLanguageModel{
			generateObjectFunc: func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
				attempts++
				return nil, fmt.Errorf("bad request: 400")
			},
		}

		scorer := NewCorrectness(mock).WithRetryConfig(RetryConfig{
			MaxRetries:   3,
			InitialDelay: 10 * time.Millisecond,
			Multiplier:   2.0,
			MaxDelay:     100 * time.Millisecond,
		})

		input := ScorerInput{
			Inputs: map[string]any{
				"input": "test",
			},
			Outputs: map[string]any{
				"output": "test",
			},
			Expectations: map[string]any{
				"expected_answer": "test",
			},
		}

		score, err := scorer.Score(context.Background(), input)
		if err != nil {
			t.Fatalf("Score() returned error: %v", err)
		}

		if score.Error == nil {
			t.Errorf("Expected error for client error, got nil")
		}

		// Should only attempt once (no retries for 4xx errors)
		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})
}

// TestNormalizeScore tests the score normalization function.
func TestNormalizeScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     any
		expected  float64
		expectErr bool
	}{
		{"bool true", true, 1.0, false},
		{"bool false", false, 0.0, false},
		{"float64 in range", 0.5, 0.5, false},
		{"float64 above range", 1.5, 1.0, false},
		{"float64 below range", -0.5, 0.0, false},
		{"int", 1, 1.0, false},
		{"int64", int64(0), 0.0, false},
		{"string", "test", 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeScore(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

// TestScorerCustomization tests scorer customization options.
func TestScorerCustomization(t *testing.T) {
	t.Parallel()

	mock := &mockLanguageModel{
		generateObjectFunc: func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
			// Verify temperature is set
			if call.Temperature != nil && *call.Temperature != 0.5 {
				return nil, fmt.Errorf("expected temperature 0.5, got %v", *call.Temperature)
			}
			return &fantasy.ObjectResponse{
				Object: map[string]any{
					"rationale": "test",
					"result":    "yes",
				},
			}, nil
		},
	}

	scorer := NewCorrectness(mock).
		WithName("CustomCorrectness").
		WithTemperature(0.5).
		WithRetryConfig(RetryConfig{
			MaxRetries:   1,
			InitialDelay: 5 * time.Millisecond,
			Multiplier:   1.5,
			MaxDelay:     50 * time.Millisecond,
		})

	if scorer.Name() != "CustomCorrectness" {
		t.Errorf("Expected name 'CustomCorrectness', got %s", scorer.Name())
	}

	input := ScorerInput{
		Inputs: map[string]any{
			"input": "test",
		},
		Outputs: map[string]any{
			"output": "test",
		},
		Expectations: map[string]any{
			"expected_answer": "test",
		},
	}

	_, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Errorf("Score() returned error: %v", err)
	}
}

// TestContextCancellation tests that context cancellation is respected.
func TestContextCancellation(t *testing.T) {
	t.Parallel()

	mock := &mockLanguageModel{
		generateObjectFunc: func(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
			// Simulate slow operation
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return &fantasy.ObjectResponse{
					Object: map[string]any{
						"rationale": "test",
						"result":    "yes",
					},
				}, nil
			}
		},
	}

	scorer := NewCorrectness(mock).WithRetryConfig(RetryConfig{
		MaxRetries:   0,
		InitialDelay: 0,
		Multiplier:   1.0,
		MaxDelay:     0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	input := ScorerInput{
		Inputs: map[string]any{
			"input": "test",
		},
		Outputs: map[string]any{
			"output": "test",
		},
		Expectations: map[string]any{
			"expected_answer": "test",
		},
	}

	score, err := scorer.Score(ctx, input)
	if err != nil {
		t.Fatalf("Score() returned error: %v", err)
	}

	if score.Error == nil {
		t.Errorf("Expected context cancellation error, got nil")
	}
}
