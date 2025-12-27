package main

// This example demonstrates MLflow integration with Fantasy agents,
// including distributed tracing and evaluation with scorers.

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy"
	"charm.land/fantasy/eval"
	"charm.land/fantasy/mlflowclient"
	"charm.land/fantasy/providers/openrouter"
	"charm.land/fantasy/tracing"
)

func main() {
	ctx := context.Background()

	// Set up MLflow client
	mlflowURL := os.Getenv("MLFLOW_TRACKING_URI")
	if mlflowURL == "" {
		mlflowURL = "http://localhost:5000"
	}

	mlflowClient, err := mlflowclient.New(mlflowURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create MLflow client:", err)
		os.Exit(1)
	}

	// Create or get experiment
	var experimentID string
	exp, err := mlflowClient.GetExperimentByName(ctx, "fantasy-example")
	if err != nil {
		// Create experiment if it doesn't exist
		experimentID, err = mlflowClient.CreateExperiment(ctx, "fantasy-example")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to create experiment:", err)
			os.Exit(1)
		}
		fmt.Printf("Created experiment: fantasy-example (ID: %s)\n", experimentID)
	} else {
		experimentID = exp.ExperimentID
		fmt.Printf("Using experiment: %s (ID: %s)\n", exp.Name, experimentID)
	}

	// Set up provider and model
	provider, err := openrouter.New(openrouter.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create provider:", err)
		os.Exit(1)
	}

	model, err := provider.LanguageModel(ctx, "moonshotai/kimi-k2")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get model:", err)
		os.Exit(1)
	}

	// Run the tracing example
	runTracingExample(ctx, model, mlflowClient, experimentID)

	// Run the evaluation example
	runEvaluationExample(ctx, model, mlflowClient, experimentID)
}

// runTracingExample demonstrates how to trace agent execution to MLflow.
func runTracingExample(ctx context.Context, model fantasy.LanguageModel, client *mlflowclient.Client, experimentID string) {
	fmt.Println("\n=== Tracing Example ===")

	// Create a simple tool for the agent
	type calcInput struct {
		A float64 `json:"a" description:"First number"`
		B float64 `json:"b" description:"Second number"`
	}

	addTool := fantasy.NewAgentTool("add", "Add two numbers together", func(ctx context.Context, input calcInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
		result := input.A + input.B
		return fantasy.NewTextResponse(fmt.Sprintf("%.2f", result)), nil
	})

	// Set up tracing callbacks
	tracingConfig := tracing.TracingConfig{
		ExperimentID: experimentID,
		AgentName:    "math-agent",
		ModelName:    "moonshotai/kimi-k2",
	}

	callbacks, _ := tracing.WrapAgentOptions(client, tracingConfig)
	cbs := callbacks.StreamCallbacks()

	// Create agent with tracing enabled using the new WithTracingCallbacks option
	agent := fantasy.NewAgent(model,
		fantasy.WithSystemPrompt("You are a helpful math assistant. Use the add tool for calculations."),
		fantasy.WithTools(addTool),
		fantasy.WithTracingCallbacks(fantasy.TracingCallbackConfig{
			OnAgentStart:  cbs.OnAgentStart,
			OnAgentFinish: cbs.OnAgentFinish,
			OnStepStart:   cbs.OnStepStart,
			OnStepFinish:  cbs.OnStepFinish,
			OnToolCall:    cbs.OnToolCall,
			OnToolResult:  cbs.OnToolResult,
			OnError:       cbs.OnError,
		}),
	)

	// Run the agent - tracing is automatic!
	result, err := agent.Stream(ctx, fantasy.AgentStreamCall{
		Prompt: "What is 42 + 58?",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Agent error:", err)
		return
	}

	fmt.Println("Result:", result.Response.Content.Text())
	fmt.Println("Trace uploaded to MLflow successfully!")
}

// runEvaluationExample demonstrates how to evaluate agent outputs with scorers.
func runEvaluationExample(ctx context.Context, model fantasy.LanguageModel, client *mlflowclient.Client, experimentID string) {
	fmt.Println("\n=== Evaluation Example ===")

	// Create a simple agent for evaluation
	agent := fantasy.NewAgent(model,
		fantasy.WithSystemPrompt("You are a helpful assistant. Answer questions concisely."),
	)

	// Create test dataset
	dataset := eval.NewDataset("qa-tests",
		eval.NewTestCase(
			map[string]any{"prompt": "What is 2+2?"},
			map[string]any{"expected": "4"},
		),
		eval.NewTestCase(
			map[string]any{"prompt": "What color is the sky?"},
			map[string]any{"expected": "blue"},
		),
		eval.NewTestCase(
			map[string]any{"prompt": "What is the capital of France?"},
			map[string]any{"expected": "Paris"},
		),
	)

	// Define scorers
	scorers := []eval.Scorer{
		eval.ContainsScorer("output"),
	}

	// Create predict function that uses the agent
	predictFunc := func(ctx context.Context, inputs map[string]any) (map[string]any, any, error) {
		prompt, ok := inputs["prompt"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("prompt must be a string")
		}

		result, err := agent.Generate(ctx, fantasy.AgentCall{Prompt: prompt})
		if err != nil {
			return nil, nil, err
		}

		return map[string]any{
			"output": result.Response.Content.Text(),
		}, nil, nil
	}

	// Run evaluation with MLflow export
	evaluator := eval.NewEvaluator()
	results, err := evaluator.Run(ctx, dataset, scorers,
		eval.WithPredict(predictFunc),
		eval.WithMLflowExport(client, experimentID),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Evaluation error:", err)
		return
	}

	// Print results
	fmt.Println("\nEvaluation Results:")
	for scorerName, stats := range results.Summary {
		fmt.Printf("  %s:\n", scorerName)
		fmt.Printf("    Mean: %.2f\n", stats.Mean)
		fmt.Printf("    Pass Rate: %.2f\n", stats.PassRate)
		fmt.Printf("    Count: %d\n", stats.Count)
	}

	fmt.Println("\nResults exported to MLflow successfully!")
}
