package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/object"
	"charm.land/fantasy/schema"
)

// JudgeConfig configures an LLM-as-Judge scorer.
type JudgeConfig struct {
	Model          fantasy.LanguageModel // LLM to use for judging
	SystemPrompt   string                // System prompt for the judge
	PromptTemplate string                // Template with {{.Output}}, {{.Expected}}, {{.Context}} placeholders
	Temperature    *float64              // Typically 0.0-0.3 for consistent judging (defaults to 0.0)
	MaxTokens      *int64                // Max response tokens (defaults to 1024)
	MaxRetries     int                   // Maximum number of retries (defaults to 3)
}

// JudgeResponse represents the structured response from an LLM judge.
type JudgeResponse struct {
	Score     float64  `json:"score" description:"Score from 1 to 5, where 1 is worst and 5 is best"`
	Rationale string   `json:"rationale" description:"Detailed explanation for the score"`
	Evidence  []string `json:"evidence,omitempty" description:"Supporting quotes or evidence from the text"`
}

// LLMJudge is the base implementation for LLM-based scoring.
type LLMJudge struct {
	name   string
	config JudgeConfig
}

// NewLLMJudge creates a new LLM judge scorer with the given configuration.
func NewLLMJudge(name string, config JudgeConfig) *LLMJudge {
	// Set defaults
	if config.Temperature == nil {
		temp := 0.0
		config.Temperature = &temp
	}
	if config.MaxTokens == nil {
		tokens := int64(1024)
		config.MaxTokens = &tokens
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &LLMJudge{
		name:   name,
		config: config,
	}
}

// Name returns the scorer's identifier.
func (j *LLMJudge) Name() string {
	return j.name
}

// Score evaluates the input using the LLM judge.
func (j *LLMJudge) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Build the prompt from template
	prompt, err := j.buildPrompt(input)
	if err != nil {
		return Score{Error: err}, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Call the LLM with retry logic
	response, err := j.callLLMWithRetry(ctx, prompt)
	if err != nil {
		return Score{Error: err}, fmt.Errorf("LLM judge call failed: %w", err)
	}

	// Normalize score to 0-1 range (LLM returns 1-5)
	normalizedScore := (response.Score - 1.0) / 4.0

	return Score{
		Value:     normalizedScore,
		Rationale: response.Rationale,
		Metadata: map[string]any{
			"raw_score": response.Score,
			"evidence":  response.Evidence,
		},
	}, nil
}

// buildPrompt constructs the judge prompt from the template and input data.
func (j *LLMJudge) buildPrompt(input ScorerInput) (string, error) {
	// Extract output
	output := extractStringOutput(input.Outputs)

	// Extract expected (if available)
	expected := ""
	if input.Expectations != nil {
		expected = extractStringExpectation(input.Expectations)
	}

	// Extract context (if available)
	contextStr := ""
	if input.Inputs != nil {
		if ctxVal, ok := input.Inputs["context"]; ok {
			if str, ok := ctxVal.(string); ok {
				contextStr = str
			} else {
				// Convert to JSON string
				if bytes, err := json.Marshal(ctxVal); err == nil {
					contextStr = string(bytes)
				}
			}
		}
	}

	// Simple template replacement (Go doesn't have template literals like JS)
	prompt := j.config.PromptTemplate
	prompt = replaceTemplate(prompt, "{{.Output}}", output)
	prompt = replaceTemplate(prompt, "{{.Expected}}", expected)
	prompt = replaceTemplate(prompt, "{{.Context}}", contextStr)

	return prompt, nil
}

// callLLMWithRetry calls the LLM with exponential backoff retry logic.
func (j *LLMJudge) callLLMWithRetry(ctx context.Context, prompt string) (*JudgeResponse, error) {
	var lastErr error
	initialDelay := 1 * time.Second
	maxDelay := 10 * time.Second

	for attempt := 0; attempt < j.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := min(initialDelay*time.Duration(1<<attempt-1), maxDelay)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		response, err := j.callLLM(ctx, prompt)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("max retries exceeded (%d attempts): %w", j.config.MaxRetries, lastErr)
}

// callLLM makes a single LLM call to judge the input.
func (j *LLMJudge) callLLM(ctx context.Context, prompt string) (*JudgeResponse, error) {
	// Use Fantasy's structured output to get a typed response
	result, err := object.Generate[JudgeResponse](ctx, j.config.Model, fantasy.ObjectCall{
		Prompt: fantasy.Prompt{
			fantasy.NewSystemMessage(j.config.SystemPrompt),
			fantasy.NewUserMessage(prompt),
		},
		Temperature:     j.config.Temperature,
		MaxOutputTokens: j.config.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("structured output generation failed: %w", err)
	}

	// Validate score is in valid range
	if result.Object.Score < 1.0 || result.Object.Score > 5.0 {
		return nil, fmt.Errorf("invalid score from LLM: %v (must be 1-5)", result.Object.Score)
	}

	return &result.Object, nil
}

// isRetryableError determines if an error should trigger a retry.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for Fantasy provider errors using errors.As
	var providerErr *fantasy.ProviderError
	if errors.As(err, &providerErr) {
		// Retry on rate limits and server errors
		if providerErr.StatusCode == 429 || providerErr.StatusCode == 503 || providerErr.StatusCode >= 500 {
			return true
		}
	}

	// Retry on schema/parsing errors (might be transient)
	var parseErr *schema.ParseError
	return errors.As(err, &parseErr)
}

// replaceTemplate is a simple template replacement helper.
func replaceTemplate(template, placeholder, value string) string {
	// Use strings.ReplaceAll equivalent
	result := template
	for {
		newResult := ""
		found := false
		for i := 0; i < len(result); i++ {
			if i+len(placeholder) <= len(result) && result[i:i+len(placeholder)] == placeholder {
				newResult += value
				i += len(placeholder) - 1
				found = true
			} else {
				newResult += string(result[i])
			}
		}
		if !found {
			break
		}
		result = newResult
	}
	return result
}

// NewCorrectnessJudge creates a judge that evaluates answer correctness.
func NewCorrectnessJudge(model fantasy.LanguageModel) *LLMJudge {
	systemPrompt := `You are an expert evaluator assessing the correctness of AI-generated answers.
Your task is to compare the generated output with the expected correct answer and evaluate factual accuracy.

Scoring Guidelines:
- Score 5: The output is completely correct and matches the expected answer
- Score 4: The output is mostly correct with minor inaccuracies
- Score 3: The output is partially correct but has significant gaps or errors
- Score 2: The output is mostly incorrect with only minor correct elements
- Score 1: The output is completely incorrect or irrelevant

Be objective and focus on factual accuracy, not style or formatting.`

	promptTemplate := `# Expected Answer:
{{.Expected}}

# Generated Output:
{{.Output}}

# Instructions:
Evaluate how correct the generated output is compared to the expected answer. Focus on factual accuracy and completeness.`

	return NewLLMJudge("correctness", JudgeConfig{
		Model:          model,
		SystemPrompt:   systemPrompt,
		PromptTemplate: promptTemplate,
	})
}

// NewGuidelinesJudge creates a judge that evaluates outputs against custom guidelines.
func NewGuidelinesJudge(model fantasy.LanguageModel, guidelines string) *LLMJudge {
	systemPrompt := `You are an expert evaluator assessing AI-generated outputs against specific guidelines.
Your task is to evaluate how well the output adheres to the provided criteria.

Scoring Guidelines:
- Score 5: Perfectly follows all guidelines
- Score 4: Follows most guidelines with minor deviations
- Score 3: Partially follows guidelines with notable gaps
- Score 2: Minimally follows guidelines with major issues
- Score 1: Does not follow guidelines at all

Be objective and systematic in your evaluation.`

	promptTemplate := fmt.Sprintf(`# Guidelines to Evaluate Against:
%s

# Generated Output:
{{.Output}}

# Instructions:
Evaluate how well the generated output follows the provided guidelines.`, guidelines)

	return NewLLMJudge("guidelines", JudgeConfig{
		Model:          model,
		SystemPrompt:   systemPrompt,
		PromptTemplate: promptTemplate,
	})
}

// NewRelevanceJudge creates a judge that evaluates response relevance.
func NewRelevanceJudge(model fantasy.LanguageModel) *LLMJudge {
	systemPrompt := `You are an expert evaluator assessing the relevance of AI-generated responses.
Your task is to evaluate whether the response is relevant to the input and context.

Scoring Guidelines:
- Score 5: Highly relevant, directly addresses the question/context
- Score 4: Mostly relevant with minor off-topic elements
- Score 3: Partially relevant but includes significant off-topic content
- Score 2: Minimally relevant, mostly off-topic
- Score 1: Completely irrelevant or unrelated

Focus on relevance, not correctness or quality.`

	promptTemplate := `# Context/Input:
{{.Expected}}

# Generated Output:
{{.Output}}

# Instructions:
Evaluate how relevant the generated output is to the given context or question.`

	return NewLLMJudge("relevance", JudgeConfig{
		Model:          model,
		SystemPrompt:   systemPrompt,
		PromptTemplate: promptTemplate,
	})
}

// NewGroundednessJudge creates a judge that evaluates factual grounding in provided context.
func NewGroundednessJudge(model fantasy.LanguageModel) *LLMJudge {
	systemPrompt := `You are an expert evaluator assessing whether AI-generated responses are grounded in the provided context.
Your task is to detect hallucinations and verify that claims are supported by the source material.

Scoring Guidelines:
- Score 5: All claims are directly supported by the context
- Score 4: Most claims are supported, minor unsupported details
- Score 3: Some claims are supported, but significant unsupported content
- Score 2: Minimal grounding, mostly unsupported or fabricated claims
- Score 1: No grounding, completely fabricated or contradicts context

Identify specific evidence (quotes) from the context that support or contradict the output.`

	promptTemplate := `# Source Context:
{{.Context}}

# Generated Output:
{{.Output}}

# Instructions:
Evaluate whether the generated output is grounded in the provided context. Identify any hallucinations or unsupported claims. Provide specific quotes as evidence.`

	return NewLLMJudge("groundedness", JudgeConfig{
		Model:          model,
		SystemPrompt:   systemPrompt,
		PromptTemplate: promptTemplate,
	})
}
