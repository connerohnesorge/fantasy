package eval

import (
	"context"
	"testing"
	"time"

	"charm.land/fantasy/mlflow"
	"charm.land/fantasy/tracing"
)

// TestScoreToAssessment tests converting a Score to an MLflow Assessment.
func TestScoreToAssessment(t *testing.T) {
	tests := []struct {
		name        string
		score       Score
		traceID     string
		scorerName  string
		wantErr     bool
		checkResult func(*testing.T, *mlflow.Assessment)
	}{
		{
			name: "passing score with rationale",
			score: Score{
				Value:     true,
				Rationale: "Output matches expected format",
				Metadata:  map[string]any{"confidence": 0.95},
			},
			traceID:    "trace-123",
			scorerName: "format_checker",
			wantErr:    false,
			checkResult: func(t *testing.T, a *mlflow.Assessment) {
				if a.Name != "format_checker" {
					t.Errorf("expected name 'format_checker', got %s", a.Name)
				}
				if a.TraceID != "trace-123" {
					t.Errorf("expected traceID 'trace-123', got %s", a.TraceID)
				}
				if a.Feedback == nil {
					t.Fatal("expected Feedback to be set")
				}
				if a.Feedback.Value != "PASS" {
					t.Errorf("expected Value 'PASS', got %v", a.Feedback.Value)
				}
				if a.Rationale != "Output matches expected format" {
					t.Errorf("expected rationale to be preserved")
				}
				if a.Metadata["confidence"] != "0.95" {
					t.Errorf("expected metadata to be converted to strings")
				}
			},
		},
		{
			name: "failing score with numeric value",
			score: Score{
				Value:     0.3,
				Rationale: "Score below threshold",
			},
			traceID:    "trace-456",
			scorerName: "accuracy",
			wantErr:    false,
			checkResult: func(t *testing.T, a *mlflow.Assessment) {
				if a.Feedback == nil {
					t.Fatal("expected Feedback to be set")
				}
				if a.Feedback.Value != "FAIL" {
					t.Errorf("expected Value 'FAIL' for score < 0.5, got %v", a.Feedback.Value)
				}
			},
		},
		{
			name: "passing score with numeric value",
			score: Score{
				Value:     0.8,
				Rationale: "Score above threshold",
			},
			traceID:    "trace-789",
			scorerName: "quality",
			wantErr:    false,
			checkResult: func(t *testing.T, a *mlflow.Assessment) {
				if a.Feedback == nil {
					t.Fatal("expected Feedback to be set")
				}
				if a.Feedback.Value != "PASS" {
					t.Errorf("expected Value 'PASS' for score > 0.5, got %v", a.Feedback.Value)
				}
			},
		},
		{
			name: "score with error",
			score: Score{
				Error:     &testError{msg: "scorer failed"},
				Rationale: "Error occurred during scoring",
			},
			traceID:    "trace-error",
			scorerName: "error_scorer",
			wantErr:    false,
			checkResult: func(t *testing.T, a *mlflow.Assessment) {
				if a.Feedback == nil {
					t.Fatal("expected Feedback to be set")
				}
				if a.Feedback.Value != "FAIL" {
					t.Errorf("expected Value 'FAIL' for errored score")
				}
				if a.Feedback.Error == nil {
					t.Error("expected Error to be set in Feedback")
				}
				if a.Feedback.Error.ErrorMessage != "scorer failed" {
					t.Errorf("expected error message to be preserved")
				}
			},
		},
		{
			name: "LLM judge with long rationale",
			score: Score{
				Value:     true,
				Rationale: "This is a very long rationale that suggests this is from an LLM judge rather than a simple heuristic scorer. It provides detailed reasoning.",
			},
			traceID:    "trace-llm",
			scorerName: "llm_judge",
			wantErr:    false,
			checkResult: func(t *testing.T, a *mlflow.Assessment) {
				if a.Source.SourceType != "LLM_JUDGE" {
					t.Errorf("expected SourceType 'LLM_JUDGE' for long rationale, got %s", a.Source.SourceType)
				}
			},
		},
		{
			name: "missing traceID",
			score: Score{
				Value: true,
			},
			traceID:    "",
			scorerName: "test",
			wantErr:    true,
		},
		{
			name: "missing scorerName",
			score: Score{
				Value: true,
			},
			traceID:    "trace-123",
			scorerName: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment, err := ScoreToAssessment(tt.score, tt.traceID, tt.scorerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScoreToAssessment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, assessment)
			}
		})
	}
}

// TestResultsToMetrics tests converting evaluation results to MLflow metrics.
func TestResultsToMetrics(t *testing.T) {
	now := time.Now()
	results := &Results{
		StartTime: now,
		EndTime:   now.Add(5 * time.Second),
		Summary: map[string]ScorerStats{
			"accuracy": {
				PassRate:   0.8,
				Mean:       0.75,
				StdDev:     0.1,
				ErrorRate:  0.1,
				ErrorCount: 2,
				Count:      20,
			},
			"relevance": {
				PassRate:   0.9,
				Mean:       0.85,
				StdDev:     0.05,
				ErrorRate:  0.05,
				ErrorCount: 1,
				Count:      20,
			},
		},
		TotalTests: 20,
	}

	metrics := ResultsToMetrics(results)

	if len(metrics) == 0 {
		t.Fatal("expected metrics to be generated")
	}

	// Check that we have metrics for both scorers
	metricsMap := make(map[string]float64)
	for _, m := range metrics {
		if m.Key != nil && m.Value != nil {
			metricsMap[*m.Key] = *m.Value
		}
	}

	// Per-scorer metrics
	expectedKeys := []string{
		"accuracy_pass_rate",
		"accuracy_mean_score",
		"accuracy_error_rate",
		"accuracy_std_dev",
		"relevance_pass_rate",
		"relevance_mean_score",
		"relevance_error_rate",
		"relevance_std_dev",
		"overall_pass_rate",
		"overall_mean_score",
		"overall_error_rate",
		"total_tests",
		"evaluation_duration_seconds",
	}

	for _, key := range expectedKeys {
		if _, exists := metricsMap[key]; !exists {
			t.Errorf("expected metric %s to be present", key)
		}
	}

	// Verify specific values
	if metricsMap["accuracy_pass_rate"] != 0.8 {
		t.Errorf("expected accuracy_pass_rate=0.8, got %f", metricsMap["accuracy_pass_rate"])
	}

	if metricsMap["total_tests"] != 20 {
		t.Errorf("expected total_tests=20, got %f", metricsMap["total_tests"])
	}

	// Check overall metrics are averages (with epsilon for floating point comparison)
	expectedOverallPassRate := (0.8 + 0.9) / 2
	epsilon := 0.0001
	if abs(metricsMap["overall_pass_rate"]-expectedOverallPassRate) > epsilon {
		t.Errorf("expected overall_pass_rate=%f, got %f", expectedOverallPassRate, metricsMap["overall_pass_rate"])
	}
}

// TestResultsToMetrics_Empty tests ResultsToMetrics with empty results.
func TestResultsToMetrics_Empty(t *testing.T) {
	metrics := ResultsToMetrics(nil)
	if metrics != nil {
		t.Error("expected nil metrics for nil results")
	}

	emptyResults := &Results{}
	metrics = ResultsToMetrics(emptyResults)
	if len(metrics) == 0 {
		t.Error("expected at least total_tests metric for empty results")
	}
}

// TestExtractTraceID tests extracting trace ID from various trace types.
func TestExtractTraceID(t *testing.T) {
	tests := []struct {
		name  string
		trace any
		want  string
	}{
		{
			name:  "nil trace",
			trace: nil,
			want:  "",
		},
		{
			name: "valid trace",
			trace: &tracing.Trace{
				TraceID: "trace-abc-123",
			},
			want: "trace-abc-123",
		},
		{
			name:  "wrong type",
			trace: "not a trace",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTraceID(tt.trace)
			if got != tt.want {
				t.Errorf("extractTraceID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWithMLflowExport tests the MLflow export option.
func TestWithMLflowExport(t *testing.T) {
	// Create a mock client (we don't need to actually connect)
	client := mlflow.New("http://localhost:5000")
	experimentID := "exp-123"

	// Create evaluator with MLflow export
	evaluator := NewEvaluator(WithMLflowExport(client, experimentID))

	if evaluator.mlflowConfig == nil {
		t.Fatal("expected mlflowConfig to be set")
	}

	if evaluator.mlflowConfig.client != client {
		t.Error("expected client to be set correctly")
	}

	if evaluator.mlflowConfig.experimentID != experimentID {
		t.Error("expected experimentID to be set correctly")
	}
}

// TestEvaluatorIntegration_MLflowExport tests the full integration with MLflow export.
// Note: This test requires a running MLflow server, so it's skipped by default.
func TestEvaluatorIntegration_MLflowExport(t *testing.T) {
	t.Skip("Integration test - requires running MLflow server")

	// Create a test dataset
	dataset := &Dataset{
		Name: "test-dataset",
		TestCases: []TestCase{
			{
				ID:           "tc-1",
				Inputs:       map[string]any{"question": "What is 2+2?"},
				Outputs:      map[string]any{"answer": "4"},
				Expectations: map[string]any{"answer": "4"},
			},
		},
	}

	// Create a simple scorer
	scorer := &mockScorer{
		name: "exact_match",
		scoreFunc: func(ctx context.Context, input ScorerInput) (Score, error) {
			expected := input.Expectations["answer"]
			actual := input.Outputs["answer"]
			return Score{
				Value:     expected == actual,
				Rationale: "Exact match comparison",
			}, nil
		},
	}

	// Create MLflow client
	client := mlflow.New("http://localhost:5000")

	// Create evaluator with MLflow export
	evaluator := NewEvaluator(WithMLflowExport(client, "0"))

	// Run evaluation
	ctx := context.Background()
	results, err := evaluator.Run(ctx, dataset, []Scorer{scorer})
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Check that results were exported
	if len(results.Errors) > 0 {
		for _, e := range results.Errors {
			if e.Phase == "export" {
				t.Errorf("export error: %s - %v", e.Message, e.Cause)
			}
		}
	}
}

// Mock types for testing

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

type mockScorer struct {
	name      string
	scoreFunc func(context.Context, ScorerInput) (Score, error)
}

func (m *mockScorer) Name() string {
	return m.name
}

func (m *mockScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	return m.scoreFunc(ctx, input)
}

// TestDatasetToRun tests the DatasetToRun converter (unit test without actual MLflow calls).
func TestDatasetToRun_Validation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		client       *mlflow.Client
		dataset      *Dataset
		experimentID string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "nil client",
			client:       nil,
			dataset:      &Dataset{Name: "test", TestCases: []TestCase{{Inputs: map[string]any{}}}},
			experimentID: "exp-1",
			wantErr:      true,
			errContains:  "client is required",
		},
		{
			name:         "nil dataset",
			client:       mlflow.New("http://localhost:5000"),
			dataset:      nil,
			experimentID: "exp-1",
			wantErr:      true,
			errContains:  "dataset is required",
		},
		{
			name:         "empty experimentID",
			client:       mlflow.New("http://localhost:5000"),
			dataset:      &Dataset{Name: "test", TestCases: []TestCase{{Inputs: map[string]any{}}}},
			experimentID: "",
			wantErr:      true,
			errContains:  "experimentID is required",
		},
		{
			name:         "invalid dataset",
			client:       mlflow.New("http://localhost:5000"),
			dataset:      &Dataset{Name: "", TestCases: []TestCase{}},
			experimentID: "exp-1",
			wantErr:      true,
			errContains:  "invalid dataset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DatasetToRun(ctx, tt.client, tt.dataset, tt.experimentID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DatasetToRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function for absolute value of float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestMetricsHaveTimestamps tests that all metrics have timestamps set.
func TestMetricsHaveTimestamps(t *testing.T) {
	results := &Results{
		StartTime: time.Now(),
		EndTime:   time.Now().Add(1 * time.Second),
		Summary: map[string]ScorerStats{
			"test_scorer": {
				PassRate:  0.5,
				Mean:      0.5,
				ErrorRate: 0.1,
			},
		},
		TotalTests: 10,
	}

	metrics := ResultsToMetrics(results)

	for _, m := range metrics {
		if m.Timestamp == nil {
			t.Errorf("metric %s missing timestamp", *m.Key)
		}
		if m.Key == nil {
			t.Error("metric missing key")
		}
		if m.Value == nil {
			t.Errorf("metric %s missing value", *m.Key)
		}
	}
}

// TestExportAssessments_NoTrace tests that assessments are skipped when trace is nil.
func TestExportAssessments_NoTrace(t *testing.T) {
	client := mlflow.New("http://localhost:5000")
	evaluator := NewEvaluator(WithMLflowExport(client, "exp-1"))
	evaluator.mlflowConfig.runID = "run-1"

	results := &Results{
		TestCases: []TestCaseResult{
			{
				Trace: nil, // No trace
				Scores: map[string]Score{
					"scorer1": {Value: true, Rationale: "test"},
				},
			},
		},
	}

	ctx := context.Background()
	evaluator.exportAssessments(ctx, results)

	// Should not error, just skip the test case
	// We can't easily verify this without a real MLflow server,
	// but at least we check it doesn't panic
}

// TestExportMetrics_NoRunID tests that metrics export is skipped when runID is not set.
func TestExportMetrics_NoRunID(t *testing.T) {
	client := mlflow.New("http://localhost:5000")
	evaluator := NewEvaluator(WithMLflowExport(client, "exp-1"))
	// Don't set runID

	results := &Results{
		Summary: map[string]ScorerStats{
			"test": {PassRate: 0.5},
		},
	}

	ctx := context.Background()
	evaluator.exportMetrics(ctx, results)

	// Should not error or panic
}
