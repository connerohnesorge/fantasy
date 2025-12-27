//go:build integration

package mlflowclient

import (
	"context"
	"os"
	"testing"
	"time"
)

// These tests require a running MLflow server.
// Run with: go test -tags=integration ./mlflowclient/...
//
// To start the MLflow server: docker-compose up -d

func getTestClient(t *testing.T) *Client {
	url := os.Getenv("MLFLOW_TRACKING_URI")
	if url == "" {
		url = "http://localhost:5000"
	}

	client, err := New(url)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return client
}

func TestIntegration_CreateAndGetExperiment(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	// Create experiment with unique name
	expName := "integration-test-" + time.Now().Format("20060102150405")
	expID, err := client.CreateExperiment(ctx, expName)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}

	if expID == "" {
		t.Fatal("CreateExperiment() returned empty experiment ID")
	}

	t.Logf("Created experiment: %s (ID: %s)", expName, expID)

	// Get experiment
	exp, err := client.GetExperiment(ctx, expID)
	if err != nil {
		t.Fatalf("GetExperiment() error = %v", err)
	}

	if exp.Name != expName {
		t.Errorf("Experiment name = %s, want %s", exp.Name, expName)
	}

	// Cleanup: delete experiment
	if err := client.DeleteExperiment(ctx, expID); err != nil {
		t.Logf("Warning: failed to delete experiment: %v", err)
	}
}

func TestIntegration_CreateRunAndLogMetrics(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	// Create experiment
	expName := "integration-test-run-" + time.Now().Format("20060102150405")
	expID, err := client.CreateExperiment(ctx, expName)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}
	defer client.DeleteExperiment(ctx, expID)

	// Create run
	run, err := client.CreateRun(ctx, expID, WithRunName("test-run"))
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	if run.Info.RunID == "" {
		t.Fatal("CreateRun() returned empty run ID")
	}

	t.Logf("Created run: %s", run.Info.RunID)

	// Log metrics
	metrics := []*Metric{
		{Key: "accuracy", Value: 0.95, Timestamp: time.Now().UnixMilli()},
		{Key: "loss", Value: 0.05, Timestamp: time.Now().UnixMilli()},
	}

	params := []*Param{
		{Key: "learning_rate", Value: "0.001"},
		{Key: "batch_size", Value: "32"},
	}

	if err := client.LogBatch(ctx, run.Info.RunID, metrics, params, nil); err != nil {
		t.Fatalf("LogBatch() error = %v", err)
	}

	t.Log("Logged metrics and params successfully")

	// Update run to finished
	_, err = client.UpdateRun(ctx, run.Info.RunID, RunStatusFinished, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("UpdateRun() error = %v", err)
	}

	t.Log("Run finished successfully")
}

func TestIntegration_CreateTrace(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	// Create experiment
	expName := "integration-test-trace-" + time.Now().Format("20060102150405")
	expID, err := client.CreateExperiment(ctx, expName)
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}
	defer client.DeleteExperiment(ctx, expID)

	// Create trace
	now := time.Now()
	trace := &Trace{
		ExperimentID:    expID,
		RequestTimeMs:   now.UnixMilli(),
		ExecutionTimeMs: 1000,
		State:           TraceStateOK,
		RequestPreview:  "What is 2+2?",
		ResponsePreview: "4",
		Tags: map[string]string{
			"environment": "test",
		},
		Spans: []*Span{
			{
				SpanID:       "span-1",
				Name:         "agent",
				SpanType:     SpanTypeAgent,
				StartTimeNs:  now.UnixNano(),
				EndTimeNs:    now.Add(time.Second).UnixNano(),
				Status:       &SpanStatus{Code: SpanStatusOK},
				Attributes:   map[string]any{"model": "gpt-4"},
			},
			{
				SpanID:       "span-2",
				ParentSpanID: "span-1",
				Name:         "llm",
				SpanType:     SpanTypeLLM,
				StartTimeNs:  now.Add(100 * time.Millisecond).UnixNano(),
				EndTimeNs:    now.Add(900 * time.Millisecond).UnixNano(),
				Status:       &SpanStatus{Code: SpanStatusOK},
				Attributes: map[string]any{
					"input_tokens":  100,
					"output_tokens": 50,
				},
			},
		},
	}

	traceID, err := client.StartTrace(ctx, trace)
	if err != nil {
		t.Fatalf("StartTrace() error = %v", err)
	}

	if traceID == "" {
		t.Fatal("StartTrace() returned empty trace ID")
	}

	t.Logf("Created trace: %s", traceID)

	// Note: GetTrace API (/traces/get) has issues in MLflow 3.x standalone mode.
	// The trace info can be retrieved via /traces/{request_id}/info endpoint.
	// For this integration test, we just verify trace creation succeeds.
	t.Log("Trace created successfully")
}

func TestIntegration_SearchExperiments(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	// Create a few experiments
	expNames := []string{
		"search-test-a-" + time.Now().Format("20060102150405"),
		"search-test-b-" + time.Now().Format("20060102150405"),
	}

	expIDs := make([]string, len(expNames))
	for i, name := range expNames {
		expID, err := client.CreateExperiment(ctx, name)
		if err != nil {
			t.Fatalf("CreateExperiment() error = %v", err)
		}
		expIDs[i] = expID
	}

	defer func() {
		for _, id := range expIDs {
			client.DeleteExperiment(ctx, id)
		}
	}()

	// Search experiments
	exps, _, err := client.SearchExperiments(ctx, SearchExperimentsOptions{
		Filter:     "name LIKE 'search-test-%'",
		MaxResults: 100,
	})
	if err != nil {
		t.Fatalf("SearchExperiments() error = %v", err)
	}

	if len(exps) < 2 {
		t.Errorf("SearchExperiments() returned %d experiments, want at least 2", len(exps))
	}

	t.Logf("Found %d experiments", len(exps))
}
