//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"charm.land/fantasy/mlflow"
	"charm.land/fantasy/tracing"
)

func main() {
	ctx := context.Background()
	client := mlflow.New("http://localhost:5000")

	fmt.Println("=== MLflow v3.8.1 Trace Upload Test ===")

	// 1. Get or create experiment
	experimentName := "trace-upload-test"
	experimentID, err := client.CreateExperiment(ctx, experimentName)
	if err != nil {
		// Try to find existing experiment
		experiments, err := client.SearchExperiments(ctx, mlflow.SearchExperimentsOptions{})
		if err != nil {
			log.Fatalf("Failed to search experiments: %v", err)
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
	fmt.Printf("Using experiment ID: %s\n", experimentID)

	// 2. Create tracing config
	config := tracing.TracingConfig{
		Client:       client,
		ExperimentID: experimentID,
		AgentName:    "trace-test-agent",
		ModelName:    "test-model",
		FlushTimeout: 30 * time.Second,
		Tags: map[string]string{
			"test": "trace-upload",
		},
	}

	// 3. Create callbacks and simulate agent execution
	callbacks := tracing.NewTracingCallbacks(config)

	// Simulate agent execution
	question := "What is 2+2?"
	fmt.Printf("Simulating agent call: %s\n", question)

	callbacks.OnAgentStart(question)

	// Simulate a step
	callbacks.OnStepStart(1)

	// Simulate LLM call
	messages := []any{
		map[string]any{
			"role":    "user",
			"content": question,
		},
	}
	callbacks.OnLLMStart(messages)
	callbacks.OnLLMFinish(10, 5, 15, nil)

	// Finish step
	callbacks.OnStepFinish()

	// Finish agent with response
	response := "4"
	fmt.Printf("Simulated response: %s\n", response)
	callbacks.OnAgentFinish(ctx, response)

	// 4. Get trace result
	result := callbacks.GetResult()
	if result.FlushError != nil {
		log.Fatalf("Trace upload failed: %v", result.FlushError)
	}

	fmt.Printf("\n=== SUCCESS ===\n")
	fmt.Printf("Trace uploaded successfully!\n")
	fmt.Printf("Trace ID: %s\n", result.Trace.TraceID)

	// 5. Verify by searching for the trace
	fmt.Printf("\n=== Verifying trace exists ===\n")
	searchResult, err := client.SearchTraces(ctx, mlflow.SearchTracesOptions{
		ExperimentIDs: []string{experimentID},
		MaxResults:    10,
	})
	if err != nil {
		log.Fatalf("Failed to search traces: %v", err)
	}

	fmt.Printf("Found %d traces in experiment\n", len(searchResult.Traces))

	// Find our trace
	found := false
	for _, t := range searchResult.Traces {
		if t.TraceId != nil && *t.TraceId == result.Trace.TraceID {
			found = true
			fmt.Printf("Found our trace: %s\n", *t.TraceId)
			if t.State != nil {
				fmt.Printf("  State: %s\n", t.State.String())
			}
			if t.RequestTime != nil {
				fmt.Printf("  Request Time: %s\n", t.RequestTime.AsTime().Format("2006-01-02 15:04:05"))
			}
			break
		}
	}

	if !found {
		log.Fatalf("Trace %s not found in search results!", result.Trace.TraceID)
	}

	fmt.Printf("\n=== TEST PASSED ===\n")
	fmt.Printf("MLflow v3.8.1 trace upload working correctly!\n")
}
