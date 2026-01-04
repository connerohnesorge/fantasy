package main

// This example demonstrates evaluation datasets with agent tool calls.
// It shows:
// 1. Creating an agent with tools (calculator, weather)
// 2. Building a dataset with expected tool calls and outputs
// 3. Using LLM judges to evaluate correctness, relevance, helpfulness, and tool usage
// 4. Persisting the dataset and results to MLflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/eval"
	"charm.land/fantasy/mlflow"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/tracing"
)

// Tool input types

// CalculatorInput defines the input for the calculator tool.
type CalculatorInput struct {
	Operation string  `json:"operation" description:"The operation to perform: add, subtract, multiply, divide"`
	A         float64 `json:"a" description:"First operand"`
	B         float64 `json:"b" description:"Second operand"`
}

// WeatherInput defines the input for the weather tool.
type WeatherInput struct {
	City string `json:"city" description:"The city to get weather for"`
}

// calculator performs basic arithmetic operations.
func calculator(ctx context.Context, input CalculatorInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	var result float64
	switch input.Operation {
	case "add":
		result = input.A + input.B
	case "subtract":
		result = input.A - input.B
	case "multiply":
		result = input.A * input.B
	case "divide":
		if input.B == 0 {
			return fantasy.NewTextResponse("Error: division by zero"), nil
		}
		result = input.A / input.B
	default:
		return fantasy.NewTextResponse(fmt.Sprintf("Unknown operation: %s", input.Operation)), nil
	}
	return fantasy.NewTextResponse(fmt.Sprintf("%.2f", result)), nil
}

// getWeather returns mock weather data for a city.
func getWeather(ctx context.Context, input WeatherInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	// Mock weather data
	weatherData := map[string]string{
		"san francisco": "Sunny, 68°F (20°C), light breeze",
		"new york":      "Partly cloudy, 72°F (22°C), humid",
		"london":        "Overcast, 58°F (14°C), light rain",
		"tokyo":         "Clear, 75°F (24°C), pleasant",
		"paris":         "Sunny, 70°F (21°C), mild wind",
	}

	// Normalize city name
	cityLower := ""
	for _, c := range input.City {
		if c >= 'A' && c <= 'Z' {
			cityLower += string(c + 32)
		} else {
			cityLower += string(c)
		}
	}

	if weather, ok := weatherData[cityLower]; ok {
		return fantasy.NewTextResponse(fmt.Sprintf("Weather in %s: %s", input.City, weather)), nil
	}
	return fantasy.NewTextResponse(fmt.Sprintf("Weather in %s: Clear, 65°F (18°C), normal conditions", input.City)), nil
}

// ToolCall represents an expected or actual tool call.
type ToolCallRecord struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func main() {
	// 1. Get configuration from environment
	mlflowURL := os.Getenv("MLFLOW_TRACKING_URI")
	if mlflowURL == "" {
		mlflowURL = "http://localhost:5000"
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	ctx := context.Background()

	// 2. Create MLflow client
	fmt.Println("=== MLflow Setup ===")
	fmt.Printf("Connecting to MLflow: %s\n", mlflowURL)
	client := mlflow.New(mlflowURL)

	// 3. Create or get experiment
	experimentName := "agent-tool-evaluation"
	experimentID, err := client.CreateExperiment(ctx, experimentName)
	if err != nil {
		fmt.Printf("Experiment note: %v\n", err)
		experiments, searchErr := client.SearchExperiments(ctx, mlflow.SearchExperimentsOptions{})
		if searchErr != nil {
			log.Fatalf("Failed to search experiments: %v", searchErr)
		}
		for _, exp := range experiments.Experiments {
			if exp.Name != nil && *exp.Name == experimentName {
				experimentID = *exp.ExperimentId
				break
			}
		}
		if experimentID == "" {
			log.Fatal("Failed to create or find experiment")
		}
	}
	fmt.Printf("Using experiment: %s (ID: %s)\n", experimentName, experimentID)

	// 4. Create provider and models
	fmt.Println("\n=== Model Setup ===")
	provider, err := openai.New(openai.WithAPIKey(apiKey), openai.WithBaseURL("https://api.x.ai/v1"))
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Main model for the agent
	agentModelName := "grok-4-1-fast-reasoning"
	agentModel, err := provider.LanguageModel(ctx, agentModelName)
	if err != nil {
		log.Fatalf("Failed to create agent model: %v", err)
	}
	fmt.Printf("Agent model: %s\n", agentModelName)

	// Judge model for evaluation
	judgeModelName := "grok-4-1-fast-reasoning"
	judgeModel, err := provider.LanguageModel(ctx, judgeModelName)
	if err != nil {
		log.Fatalf("Failed to create judge model: %v", err)
	}
	fmt.Printf("Judge model: %s\n", judgeModelName)

	// 5. Create tools using NewAgentTool
	calculatorTool := fantasy.NewAgentTool("calculator", "Perform basic arithmetic operations", calculator)
	weatherTool := fantasy.NewAgentTool("get_weather", "Get current weather for a city", getWeather)

	// 6. Create agent with tools
	agent := fantasy.NewAgent(agentModel,
		fantasy.WithSystemPrompt(`You are a helpful assistant with access to tools.
When asked to perform calculations, use the calculator tool.
When asked about weather, use the get_weather tool.
Always provide clear, helpful responses based on the tool results.`),
		fantasy.WithTools(calculatorTool, weatherTool),
	)
	fmt.Println("Agent created with tools: calculator, get_weather")

	// 7. Set up tracing
	tracingConfig := tracing.TracingConfig{
		Client:       client,
		ExperimentID: experimentID,
		AgentName:    "tool-agent",
		ModelName:    agentModelName,
		FlushTimeout: 30 * time.Second,
		Tags: map[string]string{
			"example": "eval-dataset",
			"version": "1.0",
		},
	}

	// 8. Create evaluation dataset with expected tool calls
	fmt.Println("\n=== Dataset Creation ===")
	dataset := createEvalDataset()
	fmt.Printf("Dataset: %s (%d test cases)\n", dataset.Name, len(dataset.TestCases))

	// Log dataset info
	for i, tc := range dataset.TestCases {
		fmt.Printf("  %d. %s\n", i+1, tc.Inputs["question"])
		if expectedTools, ok := tc.Expectations["expected_tool_calls"]; ok {
			toolsJSON, _ := json.Marshal(expectedTools)
			fmt.Printf("     Expected tools: %s\n", string(toolsJSON))
		}
	}

	// 9. Create LLM judge scorers
	fmt.Println("\n=== LLM Judge Scorers ===")
	scorers := []eval.Scorer{
		eval.NewCorrectnessJudge(judgeModel),
		eval.NewRelevanceJudge(judgeModel),
		eval.NewHelpfulnessJudge(judgeModel),
		eval.NewToolUsageJudge(judgeModel),
	}

	for _, scorer := range scorers {
		fmt.Printf("  - %s\n", scorer.Name())
	}

	// 10. Create evaluator with MLflow export
	evaluator := eval.NewEvaluator(
		eval.WithMLflowExport(client, experimentID),
	)

	// 11. Run evaluation
	fmt.Println("\n=== Running Evaluation ===")
	startTime := time.Now()

	results, err := evaluator.Run(ctx, dataset, scorers,
		eval.WithPredict(createPredictFunc(agent, tracingConfig)),
		eval.WithParallelism(1), // Sequential for clarity
		eval.WithTimeout(60*time.Second),
	)
	if err != nil {
		log.Fatalf("Evaluation error: %v", err)
	}

	duration := time.Since(startTime)

	// 12. Print results
	printResults(results, duration)

	// 13. Log dataset to MLflow as artifact
	fmt.Println("\n=== Dataset Persistence ===")
	if err := logDatasetToMLflow(ctx, client, experimentID, dataset); err != nil {
		log.Printf("Warning: Failed to log dataset: %v", err)
	} else {
		fmt.Println("Dataset logged to MLflow as artifact")
	}

	// 14. Completion
	fmt.Println("\n=== Completed Successfully ===")
	fmt.Printf("\nView results in MLflow UI:\n")
	fmt.Printf("  %s/#/experiments/%s\n", mlflowURL, experimentID)
	fmt.Printf("\nWhat was demonstrated:\n")
	fmt.Printf("  - Agent with tools (calculator, get_weather)\n")
	fmt.Printf("  - Evaluation dataset with expected tool calls\n")
	fmt.Printf("  - LLM judges: correctness, relevance, helpfulness, tool_usage\n")
	fmt.Printf("  - Automatic trace and assessment export to MLflow\n")
	fmt.Printf("  - Dataset persistence for reproducibility\n")
}

// createEvalDataset creates a dataset with expected tool calls and outputs.
func createEvalDataset() *eval.Dataset {
	return &eval.Dataset{
		Name: "agent-tool-calls",
		TestCases: []eval.TestCase{
			{
				ID: "calc-add",
				Inputs: map[string]any{
					"question": "What is 25 + 17?",
					"context":  "User wants to add two numbers",
				},
				Expectations: map[string]any{
					"expected":            "42",
					"expected_tool_calls": []ToolCallRecord{{Name: "calculator", Arguments: map[string]any{"operation": "add", "a": 25, "b": 17}}},
				},
			},
			{
				ID: "calc-multiply",
				Inputs: map[string]any{
					"question": "Calculate 8 times 7",
					"context":  "User wants multiplication",
				},
				Expectations: map[string]any{
					"expected":            "56",
					"expected_tool_calls": []ToolCallRecord{{Name: "calculator", Arguments: map[string]any{"operation": "multiply", "a": 8, "b": 7}}},
				},
			},
			{
				ID: "weather-sf",
				Inputs: map[string]any{
					"question": "What's the weather in San Francisco?",
					"context":  "User wants current weather",
				},
				Expectations: map[string]any{
					"expected":            "Sunny, 68°F",
					"expected_tool_calls": []ToolCallRecord{{Name: "get_weather", Arguments: map[string]any{"city": "San Francisco"}}},
				},
			},
			{
				ID: "weather-tokyo",
				Inputs: map[string]any{
					"question": "Tell me about the weather in Tokyo",
					"context":  "User wants Tokyo weather",
				},
				Expectations: map[string]any{
					"expected":            "Clear, 75°F",
					"expected_tool_calls": []ToolCallRecord{{Name: "get_weather", Arguments: map[string]any{"city": "Tokyo"}}},
				},
			},
			{
				ID: "calc-divide",
				Inputs: map[string]any{
					"question": "What is 100 divided by 4?",
					"context":  "User wants division",
				},
				Expectations: map[string]any{
					"expected":            "25",
					"expected_tool_calls": []ToolCallRecord{{Name: "calculator", Arguments: map[string]any{"operation": "divide", "a": 100, "b": 4}}},
				},
			},
		},
		Metadata: map[string]any{
			"description": "Dataset for testing agent tool usage",
			"tools":       []string{"calculator", "get_weather"},
			"created_at":  time.Now().Format(time.RFC3339),
		},
	}
}

// createPredictFunc creates the prediction function for the evaluator.
func createPredictFunc(agent fantasy.Agent, config tracing.TracingConfig) eval.PredictFunc {
	return func(ctx context.Context, inputs map[string]any) (map[string]any, any, error) {
		question, _ := inputs["question"].(string)

		// Create tracing callbacks
		callbacks := tracing.NewTracingCallbacks(config)
		callbacks.OnAgentStart(question)

		// Track tool calls
		var toolCalls []ToolCallRecord

		// Run agent
		result, err := agent.Generate(ctx, fantasy.AgentCall{
			Prompt: question,
		})

		if err != nil {
			callbacks.OnAgentFinish(ctx, "")
			return nil, nil, err
		}

		// Extract tool calls from result
		for _, step := range result.Steps {
			for _, content := range step.Content {
				if content.GetType() == fantasy.ContentTypeToolCall {
					if tc, ok := content.(fantasy.ToolCallContent); ok {
						var args map[string]any
						if err := json.Unmarshal([]byte(tc.Input), &args); err == nil {
							toolCalls = append(toolCalls, ToolCallRecord{
								Name:      tc.ToolName,
								Arguments: args,
							})
						}
					}
				}
			}
		}

		response := result.Response.Content.Text()
		callbacks.OnAgentFinish(ctx, response)

		// Get trace
		traceResult := callbacks.GetResult()
		if traceResult.FlushError != nil {
			log.Printf("Warning: Trace flush error: %v", traceResult.FlushError)
		}

		// Build outputs
		outputs := map[string]any{
			"output":     response,
			"tool_calls": toolCalls,
		}

		return outputs, traceResult.Trace, nil
	}
}

// printResults prints the evaluation results.
func printResults(results *eval.Results, duration time.Duration) {
	fmt.Println("\n=== Evaluation Results ===")
	fmt.Printf("Total tests: %d\n", results.TotalTests)
	fmt.Printf("Duration: %v\n", duration)

	if len(results.Errors) > 0 {
		fmt.Printf("\nErrors: %d\n", len(results.Errors))
		for i, evalErr := range results.Errors {
			fmt.Printf("  %d. [%s] %s\n", i+1, evalErr.Phase, evalErr.Message)
		}
	}

	// Per-scorer statistics
	fmt.Println("\n--- Scorer Statistics ---")
	for scorerName, stats := range results.Summary {
		fmt.Printf("\n%s:\n", scorerName)
		fmt.Printf("  Mean score: %.2f (normalized 0-1)\n", stats.Mean)
		fmt.Printf("  Pass rate:  %.1f%%\n", stats.PassRate*100)
		if stats.StdDev > 0 {
			fmt.Printf("  Std dev:    %.2f\n", stats.StdDev)
		}
		if stats.ErrorRate > 0 {
			fmt.Printf("  Error rate: %.1f%%\n", stats.ErrorRate*100)
		}
	}

	// Individual results
	fmt.Println("\n--- Individual Test Results ---")
	for i, tcResult := range results.TestCases {
		question := tcResult.TestCase.Inputs["question"]
		fmt.Printf("\n[%d] %s\n", i+1, question)
		fmt.Printf("    Output: %v\n", tcResult.Outputs["output"])

		if toolCalls, ok := tcResult.Outputs["tool_calls"]; ok {
			tcJSON, _ := json.Marshal(toolCalls)
			fmt.Printf("    Tools:  %s\n", string(tcJSON))
		}

		fmt.Printf("    Scores:\n")
		for scorerName, score := range tcResult.Scores {
			if score.Error != nil {
				fmt.Printf("      - %s: ERROR (%v)\n", scorerName, score.Error)
			} else if numVal, ok := score.Value.(float64); ok {
				status := "PASS"
				if numVal < 0.5 {
					status = "FAIL"
				}
				fmt.Printf("      - %s: %s (%.2f)\n", scorerName, status, numVal)
			}
		}
	}
}

// logDatasetToMLflow logs the dataset to MLflow using the Evaluation Dataset API.
func logDatasetToMLflow(ctx context.Context, client *mlflow.Client, experimentID string, dataset *eval.Dataset) error {
	// Convert metadata to tags
	tags := make(map[string]string)
	for k, v := range dataset.Metadata {
		tags[k] = fmt.Sprintf("%v", v)
	}

	// Create the evaluation dataset
	mlDataset, err := client.CreateDataset(ctx, dataset.Name, &mlflow.CreateDatasetOptions{
		ExperimentIDs: []string{experimentID},
		Tags:          tags,
	})
	if err != nil {
		return fmt.Errorf("failed to create dataset: %w", err)
	}

	fmt.Printf("Created dataset: %s (ID: %s)\n", mlDataset.Name, mlDataset.DatasetID)

	// Convert test cases to records
	var records []map[string]any
	for _, tc := range dataset.TestCases {
		record := map[string]any{
			"dataset_record_id": tc.ID,
			"inputs":            tc.Inputs,       // Pass as object, not JSON string
			"expectations":      tc.Expectations, // Pass as object, not JSON string
			"source_type":       "CODE",
		}

		// Add tags if present
		if len(tc.Tags) > 0 {
			record["tags"] = tc.Tags
		}

		records = append(records, record)
	}

	// Upsert records to the dataset
	result, err := client.UpsertDatasetRecords(ctx, mlDataset.DatasetID, records)
	if err != nil {
		return fmt.Errorf("failed to upsert records: %w", err)
	}

	fmt.Printf("Upserted %d records (inserted: %d, updated: %d)\n",
		len(records), result.InsertedCount, result.UpdatedCount)

	return nil
}
