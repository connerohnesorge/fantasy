package main

// This is a comprehensive example demonstrating MLflow integration with the Fantasy AI framework.
// It shows:
// 1. MLflow client setup and experiment management
// 2. Tracing integration with agent execution
// 3. Evaluation framework with heuristic scorers
// 4. Complete flow from agent execution to MLflow export

import (
	"context"
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
	fmt.Printf("=== MLflow Setup ===\n")
	fmt.Printf("Connecting to MLflow: %s\n", mlflowURL)
	client := mlflow.New(mlflowURL)

	// 3. Create or get experiment
	experimentName := "fantasy-mlflow-demo"
	experimentID, err := client.CreateExperiment(ctx, experimentName)
	if err != nil {
		// Experiment might already exist, try to search for it
		fmt.Printf("Experiment creation note: %v\n", err)
		fmt.Printf("Searching for existing experiment...\n")
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

	// 4. Create provider and model
	fmt.Printf("\n=== Agent Setup ===\n")
	provider, err := openai.New(openai.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	model, err := provider.LanguageModel(ctx, "gpt-4o-mini")
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}
	fmt.Printf("Model: gpt-4o-mini\n")

	// 5. Set up tracing configuration
	config := tracing.TracingConfig{
		Client:       client,
		ExperimentID: experimentID,
		AgentName:    "math-qa-agent",
		ModelName:    "gpt-4o-mini",
		FlushTimeout: 30 * time.Second,
		Tags: map[string]string{
			"example": "mlflow-integration",
			"version": "1.0",
		},
	}

	// 6. Create agent
	agent := fantasy.NewAgent(model,
		fantasy.WithSystemPrompt("You are a helpful assistant that answers math questions. Always provide just the numerical answer without any explanation."),
	)
	fmt.Printf("Agent created with tracing enabled\n")

	// 7. Demonstrate simple tracing with a single question
	fmt.Println("\n=== Single Agent Execution with Tracing ===")
	callbacks := tracing.NewTracingCallbacks(config)

	question := "What is 2+2?"
	fmt.Printf("Question: %s\n", question)

	callbacks.OnAgentStart(question)

	result, err := agent.Generate(ctx, fantasy.AgentCall{
		Prompt: question,
	})

	if err != nil {
		callbacks.OnAgentFinish(ctx, "")
		log.Printf("Agent error: %v", err)
	} else {
		callbacks.OnAgentFinish(ctx, result.Response.Content.Text())
		fmt.Printf("Response: %s\n", result.Response.Content.Text())
	}

	// Get trace result
	traceResult := callbacks.GetResult()
	if traceResult.FlushError != nil {
		log.Printf("Trace flush error: %v", traceResult.FlushError)
	} else {
		fmt.Printf("Trace uploaded successfully! Trace ID: %s\n", traceResult.Trace.TraceID)
	}

	// 8. Demonstrate evaluation framework with MLflow export
	fmt.Println("\n=== Evaluation Framework ===")

	// Create test dataset
	dataset := &eval.Dataset{
		Name: "math-qa-dataset",
		TestCases: []eval.TestCase{
			{
				ID:           "test-1",
				Inputs:       map[string]any{"question": "What is 2+2?"},
				Expectations: map[string]any{"expected": "4"},
			},
			{
				ID:           "test-2",
				Inputs:       map[string]any{"question": "What is 3*3?"},
				Expectations: map[string]any{"expected": "9"},
			},
			{
				ID:           "test-3",
				Inputs:       map[string]any{"question": "What is 10-5?"},
				Expectations: map[string]any{"expected": "5"},
			},
			{
				ID:           "test-4",
				Inputs:       map[string]any{"question": "What is 15/3?"},
				Expectations: map[string]any{"expected": "5"},
			},
			{
				ID:           "test-5",
				Inputs:       map[string]any{"question": "What is 7+8?"},
				Expectations: map[string]any{"expected": "15"},
			},
		},
		Metadata: map[string]any{
			"domain":     "mathematics",
			"difficulty": "easy",
			"operations": []string{"addition", "subtraction", "multiplication", "division"},
		},
	}

	fmt.Printf("Dataset: %s (%d test cases)\n", dataset.Name, len(dataset.TestCases))

	// Create scorers
	scorers := []eval.Scorer{
		eval.NewContains(),
		eval.NewExactMatch(),
	}

	fmt.Printf("Scorers: %d configured\n", len(scorers))
	for _, scorer := range scorers {
		fmt.Printf("  - %s\n", scorer.Name())
	}

	// Create evaluator with MLflow export
	evaluator := eval.NewEvaluator(
		eval.WithMLflowExport(client, experimentID),
	)

	// Run evaluation with predict function
	fmt.Println("\nRunning evaluation...")
	startTime := time.Now()

	results, err := evaluator.Run(ctx, dataset, scorers,
		eval.WithPredict(func(ctx context.Context, inputs map[string]any) (map[string]any, any, error) {
			question, _ := inputs["question"].(string)

			// Create tracing callbacks for this test case
			testCallbacks := tracing.NewTracingCallbacks(config)
			testCallbacks.OnAgentStart(question)

			result, err := agent.Generate(ctx, fantasy.AgentCall{
				Prompt: question,
			})

			if err != nil {
				testCallbacks.OnAgentFinish(ctx, "")
				return nil, nil, err
			}

			response := result.Response.Content.Text()
			testCallbacks.OnAgentFinish(ctx, response)

			// Get the trace
			traceResult := testCallbacks.GetResult()
			if traceResult.FlushError != nil {
				log.Printf("Warning: Trace flush error for test case: %v", traceResult.FlushError)
			}

			return map[string]any{"output": response}, traceResult.Trace, nil
		}),
		eval.WithParallelism(1), // Sequential execution for demo clarity
	)

	if err != nil {
		log.Fatalf("Evaluation error: %v", err)
	}

	duration := time.Since(startTime)

	// 9. Print detailed results
	fmt.Printf("\n=== Evaluation Results ===\n")
	fmt.Printf("Total tests: %d\n", results.TotalTests)
	fmt.Printf("Duration: %v\n", duration)

	if len(results.Errors) > 0 {
		fmt.Printf("\nErrors encountered: %d\n", len(results.Errors))
		for i, evalErr := range results.Errors {
			fmt.Printf("  %d. [%s] %s\n", i+1, evalErr.Phase, evalErr.Message)
		}
	}

	// Print per-scorer statistics
	fmt.Println("\nPer-Scorer Statistics:")
	for scorerName, stats := range results.Summary {
		fmt.Printf("\n%s:\n", scorerName)
		fmt.Printf("  Pass rate:  %.1f%% (%d/%d passed)\n",
			stats.PassRate*100,
			int(stats.PassRate*float64(stats.Count)),
			stats.Count)
		fmt.Printf("  Mean score: %.2f\n", stats.Mean)
		if stats.StdDev > 0 {
			fmt.Printf("  Std dev:    %.2f\n", stats.StdDev)
		}
		if stats.ErrorRate > 0 {
			fmt.Printf("  Error rate: %.1f%%\n", stats.ErrorRate*100)
		}
	}

	// Print individual test case results
	fmt.Println("\nIndividual Test Results:")
	for i, tcResult := range results.TestCases {
		question := tcResult.TestCase.Inputs["question"]
		expected := tcResult.TestCase.Expectations["expected"]
		output := tcResult.Outputs["output"]

		fmt.Printf("\nTest %d: %s\n", i+1, question)
		fmt.Printf("  Expected: %v\n", expected)
		fmt.Printf("  Got:      %v\n", output)

		// Show scorer results
		for scorerName, score := range tcResult.Scores {
			status := "PASS"
			if score.Error != nil {
				status = "ERROR"
			} else if numVal, ok := score.Value.(float64); ok && numVal < 0.5 {
				status = "FAIL"
			}
			fmt.Printf("  %s: %s (%.2f)\n", scorerName, status, score.Value)
		}
	}

	// 10. Completion message
	fmt.Println("\n=== Completed Successfully ===")
	fmt.Printf("\nView results in MLflow UI:\n")
	fmt.Printf("  %s/#/experiments/%s\n", mlflowURL, experimentID)
	fmt.Printf("\nWhat was demonstrated:\n")
	fmt.Printf("  ✓ MLflow client setup and experiment management\n")
	fmt.Printf("  ✓ Single agent execution with tracing\n")
	fmt.Printf("  ✓ Evaluation framework with dataset\n")
	fmt.Printf("  ✓ Heuristic scorers (Contains, ExactMatch)\n")
	fmt.Printf("  ✓ Automatic MLflow export (traces, assessments, metrics)\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  - View traces in the MLflow UI\n")
	fmt.Printf("  - Check assessments linked to each trace\n")
	fmt.Printf("  - Review aggregated metrics in the run\n")
	fmt.Printf("  - Try modifying the dataset or scorers\n")
}
