package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"charm.land/fantasy"
)

// JudgeConfig configures an LLM-as-judge scorer.
type JudgeConfig struct {
	// Model is the language model to use for judging.
	Model fantasy.LanguageModel

	// PromptTemplate is the template for the judge prompt.
	// Use {{.Input}}, {{.Output}}, {{.Expected}} placeholders.
	PromptTemplate string

	// SystemPrompt is the system prompt for the judge.
	SystemPrompt string

	// Temperature for the judge model (default: 0.0).
	Temperature *float64

	// MaxRetries for API failures (default: 3).
	MaxRetries int

	// OutputSchema defines the expected JSON output structure.
	// If nil, expects a simple pass/fail response.
	OutputSchema map[string]any
}

// LLMJudgeScorer is a base scorer that uses an LLM as a judge.
type LLMJudgeScorer struct {
	name   string
	config JudgeConfig
}

// NewLLMJudgeScorer creates a new LLM judge scorer.
func NewLLMJudgeScorer(name string, config JudgeConfig) *LLMJudgeScorer {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	return &LLMJudgeScorer{
		name:   name,
		config: config,
	}
}

// Name returns the scorer name.
func (s *LLMJudgeScorer) Name() string {
	return s.name
}

// Score runs the LLM judge on the input.
func (s *LLMJudgeScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	if s.config.Model == nil {
		return Score{}, errors.New("LLM model is required for judge scorer")
	}

	// Build the prompt
	prompt := s.buildPrompt(input)

	// Create the call options
	var temp float64
	if s.config.Temperature != nil {
		temp = *s.config.Temperature
	}

	// Run with retry
	var result *fantasy.Response
	var lastErr error
	for i := 0; i <= s.config.MaxRetries; i++ {
		result, lastErr = s.config.Model.Generate(ctx, fantasy.Call{
			Prompt:      fantasy.Prompt{fantasy.NewUserMessage(prompt)},
			Temperature: &temp,
		})
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return Score{Error: lastErr}, nil
	}

	// Parse the response
	return s.parseResponse(result.Content.Text())
}

// buildPrompt constructs the judge prompt from the template.
func (s *LLMJudgeScorer) buildPrompt(input ScorerInput) string {
	prompt := s.config.PromptTemplate

	// Get output text
	outputStr := ""
	if output, ok := input.Outputs["response"]; ok {
		if str, ok := output.(string); ok {
			outputStr = str
		} else {
			data, _ := json.Marshal(output)
			outputStr = string(data)
		}
	} else if output, ok := input.Outputs["output"]; ok {
		if str, ok := output.(string); ok {
			outputStr = str
		} else {
			data, _ := json.Marshal(output)
			outputStr = string(data)
		}
	}

	// Get input text
	inputStr := ""
	if inp, ok := input.Inputs["prompt"]; ok {
		if str, ok := inp.(string); ok {
			inputStr = str
		}
	} else if inp, ok := input.Inputs["input"]; ok {
		if str, ok := inp.(string); ok {
			inputStr = str
		}
	}

	// Get expected text
	expectedStr := ""
	if exp, ok := input.Expectations["expected"]; ok {
		if str, ok := exp.(string); ok {
			expectedStr = str
		} else {
			data, _ := json.Marshal(exp)
			expectedStr = string(data)
		}
	}

	// Replace placeholders
	prompt = strings.ReplaceAll(prompt, "{{.Input}}", inputStr)
	prompt = strings.ReplaceAll(prompt, "{{.Output}}", outputStr)
	prompt = strings.ReplaceAll(prompt, "{{.Expected}}", expectedStr)

	return prompt
}

// parseResponse parses the LLM response into a Score.
func (s *LLMJudgeScorer) parseResponse(response string) (Score, error) {
	response = strings.TrimSpace(response)

	// Try to parse as JSON first
	var jsonResp struct {
		Score     any    `json:"score"`
		Pass      *bool  `json:"pass"`
		Rationale string `json:"rationale"`
		Reasoning string `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(response), &jsonResp); err == nil {
		score := Score{
			Rationale: jsonResp.Rationale,
		}
		if score.Rationale == "" {
			score.Rationale = jsonResp.Reasoning
		}

		// Handle score value
		if jsonResp.Pass != nil {
			score.Value = *jsonResp.Pass
		} else if jsonResp.Score != nil {
			switch v := jsonResp.Score.(type) {
			case bool:
				score.Value = v
			case float64:
				score.Value = v
			case string:
				lv := strings.ToLower(v)
				if lv == "pass" || lv == "yes" || lv == "true" || lv == "correct" {
					score.Value = true
				} else if lv == "fail" || lv == "no" || lv == "false" || lv == "incorrect" {
					score.Value = false
				} else {
					score.Value = v
				}
			}
		}

		return score, nil
	}

	// Fall back to simple text parsing
	lower := strings.ToLower(response)
	if strings.Contains(lower, "pass") || strings.Contains(lower, "correct") ||
		strings.Contains(lower, "yes") {
		return Score{Value: true, Rationale: response}, nil
	}
	if strings.Contains(lower, "fail") || strings.Contains(lower, "incorrect") ||
		strings.Contains(lower, "no") {
		return Score{Value: false, Rationale: response}, nil
	}

	// Can't determine score
	return Score{
		Value:     false,
		Rationale: response,
		Error:     fmt.Errorf("could not parse judge response"),
	}, nil
}

// CorrectnessScorer evaluates answer accuracy using an LLM.
type CorrectnessScorer struct {
	*LLMJudgeScorer
}

// NewCorrectnessScorer creates a correctness scorer.
func NewCorrectnessScorer(model fantasy.LanguageModel) *CorrectnessScorer {
	temp := 0.0
	return &CorrectnessScorer{
		LLMJudgeScorer: NewLLMJudgeScorer("correctness", JudgeConfig{
			Model:       model,
			Temperature: &temp,
			PromptTemplate: `Evaluate if the following response correctly answers the question.

Question/Input:
{{.Input}}

Expected Answer:
{{.Expected}}

Actual Response:
{{.Output}}

Evaluate if the actual response is correct. Consider:
1. Does it provide the same essential information as the expected answer?
2. Are there any factual errors?
3. Is any critical information missing?

Respond with a JSON object:
{
  "pass": true or false,
  "rationale": "Brief explanation of your evaluation"
}`,
		}),
	}
}

// RelevanceScorer evaluates response relevance using an LLM.
type RelevanceScorer struct {
	*LLMJudgeScorer
}

// NewRelevanceScorer creates a relevance scorer.
func NewRelevanceScorer(model fantasy.LanguageModel) *RelevanceScorer {
	temp := 0.0
	return &RelevanceScorer{
		LLMJudgeScorer: NewLLMJudgeScorer("relevance", JudgeConfig{
			Model:       model,
			Temperature: &temp,
			PromptTemplate: `Evaluate if the following response is relevant to the question.

Question/Input:
{{.Input}}

Response:
{{.Output}}

Evaluate the relevance of the response:
1. Does it directly address the question?
2. Is the content on-topic?
3. Does it provide useful information for the query?

Respond with a JSON object:
{
  "score": 0.0 to 1.0 (1.0 = highly relevant, 0.0 = not relevant),
  "rationale": "Brief explanation of your evaluation"
}`,
		}),
	}
}

// GroundednessScorer evaluates if the response is grounded in provided context.
type GroundednessScorer struct {
	*LLMJudgeScorer
}

// NewGroundednessScorer creates a groundedness scorer.
func NewGroundednessScorer(model fantasy.LanguageModel) *GroundednessScorer {
	temp := 0.0
	return &GroundednessScorer{
		LLMJudgeScorer: NewLLMJudgeScorer("groundedness", JudgeConfig{
			Model:       model,
			Temperature: &temp,
			PromptTemplate: `Evaluate if the response is grounded in the provided context.

Context:
{{.Expected}}

Response:
{{.Output}}

Evaluate groundedness:
1. Are all claims in the response supported by the context?
2. Does the response avoid making unsupported claims?
3. Is the response consistent with the context?

Respond with a JSON object:
{
  "score": 0.0 to 1.0 (1.0 = fully grounded, 0.0 = not grounded),
  "rationale": "Brief explanation of your evaluation"
}`,
		}),
	}
}

// GuidelinesScorer evaluates if the response follows custom guidelines.
type GuidelinesScorer struct {
	*LLMJudgeScorer
}

// NewGuidelinesScorer creates a scorer that checks custom guidelines.
func NewGuidelinesScorer(model fantasy.LanguageModel, guidelines string) *GuidelinesScorer {
	temp := 0.0
	return &GuidelinesScorer{
		LLMJudgeScorer: NewLLMJudgeScorer("guidelines", JudgeConfig{
			Model:       model,
			Temperature: &temp,
			PromptTemplate: fmt.Sprintf(`Evaluate if the response follows these guidelines:

Guidelines:
%s

Input:
{{.Input}}

Response:
{{.Output}}

Evaluate compliance with each guideline. Respond with a JSON object:
{
  "pass": true or false,
  "rationale": "Brief explanation of which guidelines were followed or violated"
}`, guidelines),
		}),
	}
}
