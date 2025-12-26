// Package main demonstrates the full MLflow integration workflow with Fantasy agents.
//
// This example shows:
// 1. Setting up MLflow client and experiment
// 2. Creating an agent with tracing enabled
// 3. Running the agent and capturing traces
// 4. Evaluating agent outputs with multiple scorers
// 5. Exporting evaluation results to MLflow
//
// Prerequisites:
// - MLflow server running (default: http://localhost:5000)
// - OpenAI API key (or compatible provider)
//
// Run with:
//   OPENAI_API_KEY=your-key go run main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/eval"
	"charm.land/fantasy/mlflowclient"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/tracing"
)

func main() {
	ctx := context.Background()

	// Step 1: Setup MLflow client
	fmt.Println("=== Step 1: Setting up MLflow client ===")
	mlflowURL := getEnvOrDefault("MLFLOW_URL", "http://localhost:5000")
	mlflowClient := mlflowclient.New(mlflowURL,
		mlflowclient.WithTimeout(60*time.Second),
	)
	fmt.Printf("MLflow client connected to: %s\n\n", mlflowURL)

	// Step 2: Create or get experiment
	fmt.Println("=== Step 2: Creating MLflow experiment ===")
	expID, err := mlflowClient.CreateExperiment(ctx, "fantasy-mlflow-demo", map[string]string{
		"framework": "fantasy",
		"purpose":   "demo",
	})
	if err != nil {
		// Experiment might already exist, try to get it
		fmt.Printf("Note: %v\n", err)
		// In production, search for existing experiment by name
	} else {
		fmt.Printf("Created experiment: %s\n\n", expID)
	}

	// Step 3: Setup provider and model
	fmt.Println("=== Step 3: Setting up AI provider ===")
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	provider, err := openai.New(openai.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	model, err := provider.LanguageModel(ctx, "gpt-4o-mini")
	if err != nil {
		log.Fatalf("Failed to get model: %v", err)
	}
	fmt.Println("Model ready: gpt-4o-mini\n")

	// Step 4: Create tools for the agent
	fmt.Println("=== Step 4: Creating agent tools ===")
	weatherTool := fantasy.NewAgentTool(
		"get_weather",
		"Get current weather for a city",
		func(ctx context.Context, input map[string]any) (map[string]any, error) {
			city := input["city"].(string)
			// Simulated weather data
			return map[string]any{
				"city":        city,
				"temperature": 72,
				"conditions":  "sunny",
				"humidity":    45,
			}, nil
		},
	)
	fmt.Println("Created tool: get_weather\n")

	// Step 5: Create base agent
	fmt.Println("=== Step 5: Creating base agent ===")
	baseAgent := fantasy.NewAgent(model,
		fantasy.WithSystemPrompt("You are a helpful weather assistant. Use the get_weather tool to provide accurate weather information."),
		fantasy.WithTools(weatherTool),
	)
	fmt.Println("Base agent created\n")

	// Step 6: Wrap agent with tracing
	fmt.Println("=== Step 6: Enabling MLflow tracing ===")
	tracedAgent, err := fantasy.WithTracing(baseAgent, tracing.TracingConfig{
		Client:       mlflowClient,
		ExperimentID: expID,
		AgentName:    "weather-assistant",
		ModelName:    "gpt-4o-mini",
		Tags: map[string]string{
			"example":     "mlflow-demo",
			"environment": "development",
		},
		FlushTimeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to enable tracing: %v", err)
	}
	fmt.Println("Tracing enabled\n")

	// Step 7: Run agent with tracing
	fmt.Println("=== Step 7: Running agent (with automatic tracing) ===")
	result, err := tracedAgent.Generate(ctx, fantasy.AgentCall{
		Prompt: "What's the weather like in San Francisco?",
	})
	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}
	fmt.Printf("Agent response: %s\n", result.Response.Content.Text())
	fmt.Printf("Steps executed: %d\n", len(result.Steps))
	fmt.Println("Trace automatically flushed to MLflow\n")

	// Step 8: Setup evaluation
	fmt.Println("=== Step 8: Setting up evaluation ===")

	// Create judge model for LLM-as-judge scorers
	judgeModel, err := provider.LanguageModel(ctx, "gpt-4o-mini")
	if err != nil {
		log.Fatalf("Failed to create judge model: %v", err)
	}

	// Define test dataset
	dataset := &eval.Dataset{
		Name: "weather-qa-test",
		TestCases: []eval.TestCase{
			{
				Inputs: map[string]any{
					"question": "What's the weather in New York?",
				},
				Expectations: map[string]any{
					"contains": "New York",
					"reference": "The weather information should mention New York and include temperature.",
				},
			},
			{
				Inputs: map[string]any{
					"question": "Tell me about the weather in Tokyo?",
				},
				Expectations: map[string]any{
					"contains": "Tokyo",
					"reference": "The response should mention Tokyo and provide weather details.",
				},
			},
		},
	}
	fmt.Printf("Created dataset with %d test cases\n\n", len(dataset.TestCases))

	// Step 9: Create scorers
	fmt.Println("=== Step 9: Creating evaluation scorers ===")
	scorers := []eval.Scorer{
		// Heuristic scorer: check if response contains city name
		eval.NewContains("answer", "contains").
			WithCaseSensitive(false).
			WithName("city_mention_check"),

		// LLM-as-judge scorer: evaluate correctness
		eval.NewCorrectness(judgeModel).
			WithName("answer_correctness"),

		// LLM-as-judge scorer: evaluate relevance
		eval.NewRelevance(judgeModel).
			WithName("response_relevance"),
	}
	fmt.Printf("Created %d scorers\n\n", len(scorers))

	// Step 10: Define prediction function
	fmt.Println("=== Step 10: Running evaluation ===")
	predictFunc := func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
		question := inputs["question"].(string)

		result, err := tracedAgent.Generate(ctx, fantasy.AgentCall{
			Prompt: question,
		})
		if err != nil {
			return nil, err
		}

		return map[string]any{
			"answer": result.Response.Content.Text(),
		}, nil
	}

	// Step 11: Run evaluation with MLflow export
	evaluator := eval.NewEvaluator()
	results, err := evaluator.Run(ctx, dataset, scorers,
		eval.WithPredict(predictFunc),
		eval.WithTimeout(60*time.Second),
		eval.WithParallelism(2), // Run 2 test cases in parallel
	)
	if err != nil {
		log.Fatalf("Evaluation failed: %v", err)
	}

	// Step 12: Display results
	fmt.Println("\n=== Step 12: Evaluation Results ===")
	fmt.Printf("Total test cases: %d\n", results.TotalCases)
	fmt.Printf("Passed: %d\n", results.PassedCases)
	fmt.Printf("Failed: %d\n", results.FailedCases)
	fmt.Printf("Error rate: %.2f%%\n\n", results.ErrorRate*100)

	// Detailed results
	for i, result := range results.Results {
		fmt.Printf("Test Case %d:\n", i+1)
		fmt.Printf("  Input: %v\n", result.Inputs)
		fmt.Printf("  Status: %s\n", result.Status)
		fmt.Printf("  Scores:\n")
		for scorerName, score := range result.Scores {
			if score.Error != nil {
				fmt.Printf("    %s: ERROR - %v\n", scorerName, score.Error)
			} else {
				fmt.Printf("    %s: %v\n", scorerName, score.Value)
				if score.Rationale != "" {
					fmt.Printf("      Rationale: %s\n", score.Rationale)
				}
			}
		}
		fmt.Println()
	}

	// Step 13: Export results to JSON (optional)
	fmt.Println("=== Step 13: Exporting results ===")
	resultsJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal results: %v", err)
	} else {
		err = os.WriteFile("eval_results.json", resultsJSON, 0644)
		if err != nil {
			log.Printf("Failed to write results file: %v", err)
		} else {
			fmt.Println("Results exported to eval_results.json")
		}
	}

	// Step 14: View traces in MLflow
	fmt.Println("\n=== Step 14: Viewing results in MLflow ===")
	fmt.Printf("Open MLflow UI: %s\n", mlflowURL)
	fmt.Printf("Navigate to experiment: %s\n", expID)
	fmt.Println("View traces in the 'Traces' tab")
	fmt.Println("\nDemo complete!")
}

// getEnvOrDefault returns the environment variable value or a default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
