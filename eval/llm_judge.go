package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"text/template"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/schema"
)

// JudgeConfig configures an LLM-as-judge scorer.
type JudgeConfig struct {
	Model          fantasy.LanguageModel // Required: the LLM to use for judging
	PromptTemplate string                // Prompt template with {{.Input}}, {{.Output}}, etc.
	OutputSchema   any                   // Expected structured output schema
	Temperature    float64               // Model temperature (default: 0.0 for determinism)
}

// JudgeResponse represents the structured response from an LLM judge.
type JudgeResponse struct {
	Rationale string `json:"rationale"`
	Result    string `json:"result"` // "yes", "no", or other values depending on scorer
}

// RetryConfig defines retry behavior for LLM API failures.
type RetryConfig struct {
	MaxRetries   int           // Maximum number of retries (default: 3)
	InitialDelay time.Duration // Initial delay before first retry (default: 1s)
	Multiplier   float64       // Backoff multiplier (default: 2.0 for exponential backoff)
	MaxDelay     time.Duration // Maximum delay between retries (default: 10s)
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		Multiplier:   2.0,
		MaxDelay:     10 * time.Second,
	}
}

// callLLMJudge calls the LLM with retry logic and structured output parsing.
func callLLMJudge(ctx context.Context, config JudgeConfig, templateData map[string]any, retryConfig RetryConfig) (JudgeResponse, error) {
	// Parse and execute template
	tmpl, err := template.New("judge").Parse(config.PromptTemplate)
	if err != nil {
		return JudgeResponse{}, fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var promptBuf strings.Builder
	if err := tmpl.Execute(&promptBuf, templateData); err != nil {
		return JudgeResponse{}, fmt.Errorf("failed to execute prompt template: %w", err)
	}
	promptText := promptBuf.String()

	// Create prompt
	prompt := fantasy.Prompt{
		fantasy.NewUserMessage(promptText),
	}

	// Prepare temperature
	temp := config.Temperature
	if temp == 0.0 {
		// Default to 0.0 for deterministic judging
		temp = 0.0
	}

	// Retry logic with exponential backoff
	var lastErr error
	delay := retryConfig.InitialDelay

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying
			select {
			case <-ctx.Done():
				return JudgeResponse{}, ctx.Err()
			case <-time.After(delay):
			}

			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * retryConfig.Multiplier)
			if delay > retryConfig.MaxDelay {
				delay = retryConfig.MaxDelay
			}
		}

		// Generate schema from the output schema type
		var schemaObj fantasy.Schema
		if s, ok := config.OutputSchema.(fantasy.Schema); ok {
			schemaObj = s
		} else {
			// Generate schema from reflect.Type
			schemaObj = schema.Generate(reflect.TypeOf(config.OutputSchema))
		}

		// Call LLM with structured output
		response, err := config.Model.GenerateObject(ctx, fantasy.ObjectCall{
			Prompt:            prompt,
			Schema:            schemaObj,
			SchemaName:        "JudgeResponse",
			SchemaDescription: "LLM judge response with rationale and result",
			Temperature:       &temp,
		})

		if err != nil {
			lastErr = err
			// Retry on specific errors: connection errors, rate limits (429), server errors (5xx)
			// Don't retry on client errors (4xx except 429), validation errors
			if shouldRetry(err) && attempt < retryConfig.MaxRetries {
				continue
			}
			return JudgeResponse{}, fmt.Errorf("LLM judge API call failed: %w", err)
		}

		// Parse the structured output
		var judgeResp JudgeResponse

		// Try to assert the object to JudgeResponse directly
		if respMap, ok := response.Object.(map[string]any); ok {
			// Extract rationale
			if r, ok := respMap["rationale"].(string); ok {
				judgeResp.Rationale = r
			}
			// Extract result
			if res, ok := respMap["result"].(string); ok {
				judgeResp.Result = res
			}

			// Validate that we got the required fields
			if judgeResp.Rationale == "" || judgeResp.Result == "" {
				// Try JSON marshaling/unmarshaling as fallback
				jsonData, err := json.Marshal(response.Object)
				if err != nil {
					return JudgeResponse{}, fmt.Errorf("failed to marshal response object: %w", err)
				}
				if err := json.Unmarshal(jsonData, &judgeResp); err != nil {
					return JudgeResponse{}, fmt.Errorf("failed to parse judge response: %w", err)
				}
			}
		} else {
			// Fallback: try JSON marshaling/unmarshaling
			jsonData, err := json.Marshal(response.Object)
			if err != nil {
				return JudgeResponse{}, fmt.Errorf("failed to marshal response object: %w", err)
			}
			if err := json.Unmarshal(jsonData, &judgeResp); err != nil {
				return JudgeResponse{}, fmt.Errorf("failed to parse judge response: %w", err)
			}
		}

		return judgeResp, nil
	}

	return JudgeResponse{}, fmt.Errorf("LLM judge failed after %d retries: %w", retryConfig.MaxRetries, lastErr)
}

// shouldRetry determines if an error should trigger a retry.
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Retry on rate limit errors
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		return true
	}

	// Retry on server errors (5xx)
	if strings.Contains(errStr, "500") || strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") || strings.Contains(errStr, "504") {
		return true
	}

	// Retry on connection timeouts
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return true
	}

	// Don't retry on client errors (4xx except 429)
	return false
}

// CorrectnessScorer evaluates if the output correctly answers the question.
type CorrectnessScorer struct {
	name        string
	model       fantasy.LanguageModel
	temperature float64
	retryConfig RetryConfig
}

// NewCorrectness creates a new Correctness scorer.
func NewCorrectness(model fantasy.LanguageModel) *CorrectnessScorer {
	return &CorrectnessScorer{
		name:        "Correctness",
		model:       model,
		temperature: 0.0,
		retryConfig: DefaultRetryConfig(),
	}
}

// WithName sets a custom name for the scorer.
func (s *CorrectnessScorer) WithName(name string) *CorrectnessScorer {
	s.name = name
	return s
}

// WithTemperature sets the model temperature.
func (s *CorrectnessScorer) WithTemperature(temp float64) *CorrectnessScorer {
	s.temperature = temp
	return s
}

// WithRetryConfig sets the retry configuration.
func (s *CorrectnessScorer) WithRetryConfig(config RetryConfig) *CorrectnessScorer {
	s.retryConfig = config
	return s
}

// Name returns the scorer's identifier.
func (s *CorrectnessScorer) Name() string {
	return s.name
}

// Score evaluates whether the output correctly answers the question.
func (s *CorrectnessScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract required fields
	userInput, ok := input.Inputs["input"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'input' in Inputs"),
			Rationale: "Missing required field 'input' in Inputs",
		}, nil
	}

	output, ok := input.Outputs["output"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'output' in Outputs"),
			Rationale: "Missing required field 'output' in Outputs",
		}, nil
	}

	expectedAnswer, ok := input.Expectations["expected_answer"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'expected_answer' in Expectations"),
			Rationale: "Missing required field 'expected_answer' in Expectations",
		}, nil
	}

	// Prepare template data
	templateData := map[string]any{
		"Input":          fmt.Sprintf("%v", userInput),
		"Output":         fmt.Sprintf("%v", output),
		"ExpectedAnswer": fmt.Sprintf("%v", expectedAnswer),
	}

	// Define prompt template
	promptTemplate := `Consider the following question, claim and document. You must determine whether the claim is
supported by the document in the context of the question. Do not focus on the correctness or
completeness of the claim. Do not make assumptions, approximations, or bring in external knowledge.

<question>{{.Input}}</question>
<claim>{{.ExpectedAnswer}}</claim>
<document>{{.Input}} - {{.Output}}</document>

Please indicate whether each statement in the claim is supported by the document in the context
of the question using only the following json format. Do not use any markdown formatting.
{
  "rationale": "Reason for the assessment. Start with 'Let's think step by step'",
  "result": "yes|no"
}`

	// Define schema (pointer for proper type assertion)
	schema := &JudgeResponse{}

	// Call LLM judge
	config := JudgeConfig{
		Model:          s.model,
		PromptTemplate: promptTemplate,
		OutputSchema:   schema,
		Temperature:    s.temperature,
	}

	resp, err := callLLMJudge(ctx, config, templateData, s.retryConfig)
	if err != nil {
		return Score{
			Error:     err,
			Rationale: fmt.Sprintf("LLM judge failed: %v", err),
		}, nil
	}

	// Convert result to boolean score
	isCorrect := strings.ToLower(strings.TrimSpace(resp.Result)) == "yes"

	return Score{
		Value:     isCorrect,
		Rationale: resp.Rationale,
		Metadata: map[string]any{
			"result":          resp.Result,
			"input":           userInput,
			"output":          output,
			"expected_answer": expectedAnswer,
		},
	}, nil
}

// GuidelinesScorer evaluates if the output follows custom guidelines.
type GuidelinesScorer struct {
	name        string
	model       fantasy.LanguageModel
	temperature float64
	retryConfig RetryConfig
}

// NewGuidelines creates a new Guidelines scorer.
func NewGuidelines(model fantasy.LanguageModel) *GuidelinesScorer {
	return &GuidelinesScorer{
		name:        "Guidelines",
		model:       model,
		temperature: 0.0,
		retryConfig: DefaultRetryConfig(),
	}
}

// WithName sets a custom name for the scorer.
func (s *GuidelinesScorer) WithName(name string) *GuidelinesScorer {
	s.name = name
	return s
}

// WithTemperature sets the model temperature.
func (s *GuidelinesScorer) WithTemperature(temp float64) *GuidelinesScorer {
	s.temperature = temp
	return s
}

// WithRetryConfig sets the retry configuration.
func (s *GuidelinesScorer) WithRetryConfig(config RetryConfig) *GuidelinesScorer {
	s.retryConfig = config
	return s
}

// Name returns the scorer's identifier.
func (s *GuidelinesScorer) Name() string {
	return s.name
}

// Score evaluates whether the output follows the guidelines.
func (s *GuidelinesScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract required fields
	userInput, ok := input.Inputs["input"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'input' in Inputs"),
			Rationale: "Missing required field 'input' in Inputs",
		}, nil
	}

	output, ok := input.Outputs["output"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'output' in Outputs"),
			Rationale: "Missing required field 'output' in Outputs",
		}, nil
	}

	guidelines, ok := input.Expectations["guidelines"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'guidelines' in Expectations"),
			Rationale: "Missing required field 'guidelines' in Expectations",
		}, nil
	}

	// Convert guidelines to slice
	var guidelinesSlice []string
	switch g := guidelines.(type) {
	case []string:
		guidelinesSlice = g
	case []any:
		for _, item := range g {
			guidelinesSlice = append(guidelinesSlice, fmt.Sprintf("%v", item))
		}
	case string:
		guidelinesSlice = []string{g}
	default:
		return Score{
			Error:     fmt.Errorf("guidelines must be a string or array of strings"),
			Rationale: "Invalid guidelines format",
		}, nil
	}

	// Prepare template data
	templateData := map[string]any{
		"Input":      fmt.Sprintf("%v", userInput),
		"Output":     fmt.Sprintf("%v", output),
		"Guidelines": guidelinesSlice,
	}

	// Define prompt template
	promptTemplate := `Given the following set of guidelines and some inputs, please assess whether the inputs fully
comply with all the provided guidelines. Only focus on the provided guidelines and not the
correctness, relevance, or effectiveness of the inputs.

<guidelines>
{{range .Guidelines}}<guideline>{{.}}</guideline>
{{end}}
</guidelines>
<input>{{.Input}}</input>
<output>{{.Output}}</output>

Please provide your assessment using only the following json format. Do not use any markdown formatting.
If any of the guidelines are not satisfied, the result must be "no".
{
  "rationale": "Detailed reasoning for your assessment. Start with 'Let's think step by step.'",
  "result": "yes|no"
}`

	// Define schema (pointer for proper type assertion)
	schema := &JudgeResponse{}

	// Call LLM judge
	config := JudgeConfig{
		Model:          s.model,
		PromptTemplate: promptTemplate,
		OutputSchema:   schema,
		Temperature:    s.temperature,
	}

	resp, err := callLLMJudge(ctx, config, templateData, s.retryConfig)
	if err != nil {
		return Score{
			Error:     err,
			Rationale: fmt.Sprintf("LLM judge failed: %v", err),
		}, nil
	}

	// Convert result to boolean score
	followsGuidelines := strings.ToLower(strings.TrimSpace(resp.Result)) == "yes"

	return Score{
		Value:     followsGuidelines,
		Rationale: resp.Rationale,
		Metadata: map[string]any{
			"result":     resp.Result,
			"input":      userInput,
			"output":     output,
			"guidelines": guidelinesSlice,
		},
	}, nil
}

// RelevanceScorer evaluates if the output is relevant to the query.
type RelevanceScorer struct {
	name        string
	model       fantasy.LanguageModel
	temperature float64
	retryConfig RetryConfig
}

// NewRelevance creates a new Relevance scorer.
func NewRelevance(model fantasy.LanguageModel) *RelevanceScorer {
	return &RelevanceScorer{
		name:        "Relevance",
		model:       model,
		temperature: 0.0,
		retryConfig: DefaultRetryConfig(),
	}
}

// WithName sets a custom name for the scorer.
func (s *RelevanceScorer) WithName(name string) *RelevanceScorer {
	s.name = name
	return s
}

// WithTemperature sets the model temperature.
func (s *RelevanceScorer) WithTemperature(temp float64) *RelevanceScorer {
	s.temperature = temp
	return s
}

// WithRetryConfig sets the retry configuration.
func (s *RelevanceScorer) WithRetryConfig(config RetryConfig) *RelevanceScorer {
	s.retryConfig = config
	return s
}

// Name returns the scorer's identifier.
func (s *RelevanceScorer) Name() string {
	return s.name
}

// Score evaluates whether the output is relevant to the query.
func (s *RelevanceScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract required fields
	userInput, ok := input.Inputs["input"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'input' in Inputs"),
			Rationale: "Missing required field 'input' in Inputs",
		}, nil
	}

	output, ok := input.Outputs["output"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'output' in Outputs"),
			Rationale: "Missing required field 'output' in Outputs",
		}, nil
	}

	// Prepare template data
	templateData := map[string]any{
		"Input":  fmt.Sprintf("%v", userInput),
		"Output": fmt.Sprintf("%v", output),
	}

	// Define prompt template
	promptTemplate := `Consider the following question and answer. You must determine whether the answer provides
information that is (fully or partially) relevant to the question. Do not focus on the correctness
or completeness of the answer. Do not make assumptions, approximations, or bring in external knowledge.

<question>{{.Input}}</question>
<answer>{{.Output}}</answer>

Please indicate whether the answer contains information that is relevant to the question using only
the following json format. Do not use any markdown formatting.
{
  "rationale": "Reason for the assessment. Start with 'Let's think step by step'",
  "result": "yes|no"
}`

	// Define schema (pointer for proper type assertion)
	schema := &JudgeResponse{}

	// Call LLM judge
	config := JudgeConfig{
		Model:          s.model,
		PromptTemplate: promptTemplate,
		OutputSchema:   schema,
		Temperature:    s.temperature,
	}

	resp, err := callLLMJudge(ctx, config, templateData, s.retryConfig)
	if err != nil {
		return Score{
			Error:     err,
			Rationale: fmt.Sprintf("LLM judge failed: %v", err),
		}, nil
	}

	// Convert result to boolean score
	isRelevant := strings.ToLower(strings.TrimSpace(resp.Result)) == "yes"

	return Score{
		Value:     isRelevant,
		Rationale: resp.Rationale,
		Metadata: map[string]any{
			"result": resp.Result,
			"input":  userInput,
			"output": output,
		},
	}, nil
}

// GroundednessScorer evaluates if the output is grounded in retrieved content.
type GroundednessScorer struct {
	name        string
	model       fantasy.LanguageModel
	temperature float64
	retryConfig RetryConfig
}

// NewGroundedness creates a new Groundedness scorer.
func NewGroundedness(model fantasy.LanguageModel) *GroundednessScorer {
	return &GroundednessScorer{
		name:        "Groundedness",
		model:       model,
		temperature: 0.0,
		retryConfig: DefaultRetryConfig(),
	}
}

// WithName sets a custom name for the scorer.
func (s *GroundednessScorer) WithName(name string) *GroundednessScorer {
	s.name = name
	return s
}

// WithTemperature sets the model temperature.
func (s *GroundednessScorer) WithTemperature(temp float64) *GroundednessScorer {
	s.temperature = temp
	return s
}

// WithRetryConfig sets the retry configuration.
func (s *GroundednessScorer) WithRetryConfig(config RetryConfig) *GroundednessScorer {
	s.retryConfig = config
	return s
}

// Name returns the scorer's identifier.
func (s *GroundednessScorer) Name() string {
	return s.name
}

// Score evaluates whether the output is grounded in retrieved content.
func (s *GroundednessScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	// Extract required fields
	userInput, ok := input.Inputs["input"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'input' in Inputs"),
			Rationale: "Missing required field 'input' in Inputs",
		}, nil
	}

	output, ok := input.Outputs["output"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'output' in Outputs"),
			Rationale: "Missing required field 'output' in Outputs",
		}, nil
	}

	retrievalContext, ok := input.Inputs["retrieval_context"]
	if !ok {
		return Score{
			Error:     fmt.Errorf("missing 'retrieval_context' in Inputs"),
			Rationale: "Missing required field 'retrieval_context' in Inputs",
		}, nil
	}

	// Prepare template data
	templateData := map[string]any{
		"Input":            fmt.Sprintf("%v", userInput),
		"Output":           fmt.Sprintf("%v", output),
		"RetrievalContext": fmt.Sprintf("%v", retrievalContext),
	}

	// Define prompt template
	promptTemplate := `Consider the following claim and document. You must determine whether claim is supported by the
document. Do not focus on the correctness or completeness of the claim. Do not make assumptions,
approximations, or bring in external knowledge.

<claim>
  <question>{{.Input}}</question>
  <answer>{{.Output}}</answer>
</claim>
<document>{{.RetrievalContext}}</document>

Please indicate whether each statement in the claim is supported by the document using only the
following json format. Do not use any markdown formatting.
{
  "rationale": "Reason for the assessment. Start with 'Let's think step by step'",
  "result": "yes|no"
}`

	// Define schema (pointer for proper type assertion)
	schema := &JudgeResponse{}

	// Call LLM judge
	config := JudgeConfig{
		Model:          s.model,
		PromptTemplate: promptTemplate,
		OutputSchema:   schema,
		Temperature:    s.temperature,
	}

	resp, err := callLLMJudge(ctx, config, templateData, s.retryConfig)
	if err != nil {
		return Score{
			Error:     err,
			Rationale: fmt.Sprintf("LLM judge failed: %v", err),
		}, nil
	}

	// Convert result to boolean score
	isGrounded := strings.ToLower(strings.TrimSpace(resp.Result)) == "yes"

	return Score{
		Value:     isGrounded,
		Rationale: resp.Rationale,
		Metadata: map[string]any{
			"result":            resp.Result,
			"input":             userInput,
			"output":            output,
			"retrieval_context": retrievalContext,
		},
	}, nil
}

// normalizeScore converts various score types to a float64 between 0.0 and 1.0.
// Used internally for score normalization when needed.
func normalizeScore(value any) (float64, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	case float64:
		// Clamp to [0.0, 1.0]
		return math.Max(0.0, math.Min(1.0, v)), nil
	case float32:
		return math.Max(0.0, math.Min(1.0, float64(v))), nil
	case int:
		return math.Max(0.0, math.Min(1.0, float64(v))), nil
	case int64:
		return math.Max(0.0, math.Min(1.0, float64(v))), nil
	default:
		return 0.0, fmt.Errorf("cannot normalize value of type %T", value)
	}
}
