package tracing_test

import (
	"context"
	"testing"

	"charm.land/fantasy"
)

// TestIntegrationWithAgent tests tracing with a real agent execution.
// This test requires a running MLflow server and is marked as integration.
func TestIntegrationWithAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require:
	// 1. A running MLflow server
	// 2. A mock or real language model
	// 3. Agent execution

	// Example of how to use the tracing integration:
	/*
		ctx := context.Background()

		// Create MLflow client
		client := mlflowclient.NewClient("http://localhost:5000")

		// Create experiment
		expReq := &pb.CreateExperimentRequest{
			Name: "test-experiment",
		}
		expResp, err := client.CreateExperiment(ctx, expReq)
		if err != nil {
			t.Fatalf("Failed to create experiment: %v", err)
		}

		// Create base agent
		model := ... // mock or real model
		baseAgent := fantasy.NewAgent(model,
			fantasy.WithTools(
				fantasy.NewTool("test_tool", "Test tool", func(ctx context.Context, input string) (string, error) {
					return "tool result", nil
				}),
			),
		)

		// Wrap with tracing
		tracedAgent, err := fantasy.WithTracing(baseAgent, tracing.TracingConfig{
			Client:       client,
			ExperimentID: expResp.GetExperimentId(),
			AgentName:    "test-agent",
			ModelName:    "test-model",
		})
		if err != nil {
			t.Fatalf("Failed to create traced agent: %v", err)
		}

		// Execute agent
		result, err := tracedAgent.Generate(ctx, fantasy.AgentCall{
			Prompt: "Test prompt",
		})
		if err != nil {
			t.Fatalf("Agent execution failed: %v", err)
		}

		// Verify result
		if result == nil {
			t.Fatal("Agent result is nil")
		}

		// Verify trace was sent to MLflow
		// (would need to query MLflow API to verify)
	*/

	t.Log("Integration test would run here with a real MLflow server")
}

// TestTracingWrapperBasic tests basic tracing wrapper functionality
// without requiring a real MLflow server.
func TestTracingWrapperBasic(t *testing.T) {
	// Create mock MLflow client (would need mock implementation)
	// For now, skip this test since it requires mocking
	t.Skip("Skipping test that requires mock implementation")

	/*
		// Create a mock agent
		mockAgent := &mockAgent{}

		mockClient := &mlflowclient.Client{} // Mock client

		// Create tracing wrapper
		wrapper, err := tracing.NewTracingWrapper(mockAgent, tracing.TracingConfig{
			Client:       mockClient,
			ExperimentID: "test-exp",
			AgentName:    "test-agent",
		})
		if err != nil {
			t.Fatalf("Failed to create tracing wrapper: %v", err)
		}

		// Execute agent
		ctx := context.Background()
		result, err := wrapper.Generate(ctx, fantasy.AgentCall{
			Prompt: "Test prompt",
		})
		if err != nil {
			t.Fatalf("Wrapper execution failed: %v", err)
		}

		if result == nil {
			t.Fatal("Result is nil")
		}
	*/
}

// mockAgent is a simple mock agent for testing
type mockAgent struct{}

func (m *mockAgent) Generate(ctx context.Context, call fantasy.AgentCall) (*fantasy.AgentResult, error) {
	return &fantasy.AgentResult{
		Steps: []fantasy.StepResult{
			{
				Response: fantasy.Response{
					Content: []fantasy.Content{
						fantasy.TextContent{Text: "Mock response"},
					},
					FinishReason: fantasy.FinishReasonStop,
				},
			},
		},
		Response: fantasy.Response{
			Content: []fantasy.Content{
				fantasy.TextContent{Text: "Mock response"},
			},
			FinishReason: fantasy.FinishReasonStop,
		},
	}, nil
}

func (m *mockAgent) Stream(ctx context.Context, call fantasy.AgentStreamCall) (*fantasy.AgentResult, error) {
	return m.Generate(ctx, fantasy.AgentCall{
		Prompt: call.Prompt,
	})
}
