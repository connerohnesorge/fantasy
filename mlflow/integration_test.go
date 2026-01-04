//go:build integration

package mlflow

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// TestIntegrationMLflowClient tests the MLflow client against a local server.
// Run with: go test -tags=integration -v ./mlflow/
func TestIntegrationMLflowClient(t *testing.T) {
	mlflowURL := os.Getenv("MLFLOW_TRACKING_URI")
	if mlflowURL == "" {
		mlflowURL = "http://localhost:5000"
	}

	client := New(mlflowURL)
	ctx := context.Background()

	t.Run("CreateAndGetExperiment", func(t *testing.T) {
		// Create experiment
		expName := fmt.Sprintf("test-experiment-%d", time.Now().UnixNano())
		expID, err := client.CreateExperiment(ctx, expName)
		if err != nil {
			t.Fatalf("Failed to create experiment: %v", err)
		}
		t.Logf("Created experiment: %s (ID: %s)", expName, expID)

		// Get experiment
		exp, err := client.GetExperiment(ctx, expID)
		if err != nil {
			t.Fatalf("Failed to get experiment: %v", err)
		}
		if exp.Name == nil || *exp.Name != expName {
			t.Errorf("Experiment name mismatch: got %v, want %s", exp.Name, expName)
		}
		t.Logf("Got experiment: %s", *exp.Name)

		// Delete experiment
		err = client.DeleteExperiment(ctx, expID)
		if err != nil {
			t.Fatalf("Failed to delete experiment: %v", err)
		}
		t.Logf("Deleted experiment: %s", expID)
	})

	t.Run("SearchExperiments", func(t *testing.T) {
		result, err := client.SearchExperiments(ctx, SearchExperimentsOptions{})
		if err != nil {
			t.Fatalf("Failed to search experiments: %v", err)
		}
		t.Logf("Found %d experiments", len(result.Experiments))
	})

	t.Run("CreateRunAndLogMetrics", func(t *testing.T) {
		// Use default experiment
		expID := "0"

		// Create run
		run, err := client.CreateRun(ctx, expID,
			WithRunName("integration-test-run"),
			WithTags(map[string]string{
				"test": "integration",
			}),
		)
		if err != nil {
			t.Fatalf("Failed to create run: %v", err)
		}
		runID := *run.Info.RunId
		t.Logf("Created run: %s", runID)

		// Log metrics
		timestamp := time.Now().UnixMilli()
		accuracy := 0.95
		loss := 0.05
		accKey := "accuracy"
		lossKey := "loss"

		err = client.LogBatch(ctx, runID, []pb.Metric{
			{Key: &accKey, Value: &accuracy, Timestamp: &timestamp},
			{Key: &lossKey, Value: &loss, Timestamp: &timestamp},
		}, nil, nil)
		if err != nil {
			t.Fatalf("Failed to log metrics: %v", err)
		}
		t.Logf("Logged metrics: accuracy=%.2f, loss=%.2f", accuracy, loss)

		// Update run status
		_, err = client.UpdateRun(ctx, runID, "FINISHED", time.Now().UnixMilli())
		if err != nil {
			t.Fatalf("Failed to update run: %v", err)
		}
		t.Logf("Updated run status to FINISHED")

		// Get run
		gotRun, err := client.GetRun(ctx, runID)
		if err != nil {
			t.Fatalf("Failed to get run: %v", err)
		}
		t.Logf("Got run: %s, status: %s", *gotRun.Info.RunId, *gotRun.Info.Status)

		// Delete run
		err = client.DeleteRun(ctx, runID)
		if err != nil {
			t.Fatalf("Failed to delete run: %v", err)
		}
		t.Logf("Deleted run: %s", runID)
	})
}
