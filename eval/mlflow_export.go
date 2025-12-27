package eval

import (
	"context"
	"fmt"
	"time"

	"charm.land/fantasy/mlflowclient"
)

// MLflowExporter exports evaluation results to MLflow.
type MLflowExporter struct {
	client       *mlflowclient.Client
	experimentID string
}

// NewMLflowExporter creates a new MLflow exporter.
func NewMLflowExporter(client *mlflowclient.Client, experimentID string) *MLflowExporter {
	return &MLflowExporter{
		client:       client,
		experimentID: experimentID,
	}
}

// ExportResults exports evaluation results to MLflow as metrics and assessments.
func (e *MLflowExporter) ExportResults(ctx context.Context, results *Results, runID string) error {
	if e.client == nil {
		return nil
	}

	// Convert results to metrics
	metrics := EvaluationResultsToMetrics(results)

	// Log metrics to the run
	if len(metrics) > 0 {
		mlMetrics := make([]*mlflowclient.Metric, 0, len(metrics))
		for key, value := range metrics {
			mlMetrics = append(mlMetrics, &mlflowclient.Metric{
				Key:       key,
				Value:     value,
				Timestamp: time.Now().UnixMilli(),
			})
		}
		if err := e.client.LogBatch(ctx, runID, mlMetrics, nil, nil); err != nil {
			return fmt.Errorf("failed to log metrics: %w", err)
		}
	}

	return nil
}

// ExportAssessments exports individual test case scores as MLflow assessments.
func (e *MLflowExporter) ExportAssessments(ctx context.Context, results *Results, traceID string) error {
	if e.client == nil {
		return nil
	}

	for i, tcResult := range results.TestCases {
		for scorerName, score := range tcResult.Scores {
			assessment := ScoreToAssessment(score, scorerName, traceID, "", i)
			if _, err := e.client.CreateAssessment(ctx, traceID, assessment); err != nil {
				return fmt.Errorf("failed to create assessment for test case %d: %w", i, err)
			}
		}
	}

	return nil
}

// ScoreToAssessment converts a Score to an MLflow Assessment.
func ScoreToAssessment(score Score, scorerName, traceID, spanID string, testCaseIndex int) *mlflowclient.Assessment {
	assessment := &mlflowclient.Assessment{
		Name:    scorerName,
		TraceID: traceID,
		SpanID:  spanID,
		Source: &mlflowclient.AssessmentSource{
			SourceType: mlflowclient.SourceTypeCode,
			SourceID:   fmt.Sprintf("eval/%s", scorerName),
		},
		Rationale: score.Rationale,
		Metadata:  make(map[string]string),
	}

	// Add test case index to metadata
	assessment.Metadata["test_case_index"] = fmt.Sprintf("%d", testCaseIndex)

	// Handle error case
	if score.Error != nil {
		assessment.Feedback = &mlflowclient.FeedbackValue{
			Error: &mlflowclient.AssessmentError{
				ErrorMessage: score.Error.Error(),
			},
		}
		return assessment
	}

	// Convert score value
	assessment.Feedback = &mlflowclient.FeedbackValue{
		Value: score.Value,
	}

	// Add metadata
	for key, val := range score.Metadata {
		if strVal, ok := val.(string); ok {
			assessment.Metadata[key] = strVal
		} else {
			assessment.Metadata[key] = fmt.Sprintf("%v", val)
		}
	}

	return assessment
}

// DatasetToRun creates an MLflow run from a dataset evaluation.
func DatasetToRun(ctx context.Context, client *mlflowclient.Client, experimentID string, dataset *Dataset, runName string) (*mlflowclient.Run, error) {
	if client == nil {
		return nil, nil
	}

	if runName == "" {
		runName = fmt.Sprintf("eval_%s_%d", dataset.Name, time.Now().Unix())
	}

	run, err := client.CreateRun(ctx, experimentID, mlflowclient.WithRunName(runName))
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}

	// Log dataset info as parameters
	params := []*mlflowclient.Param{
		{Key: "dataset_name", Value: dataset.Name},
		{Key: "test_case_count", Value: fmt.Sprintf("%d", len(dataset.TestCases))},
	}

	if err := client.LogBatch(ctx, run.Info.RunID, nil, params, nil); err != nil {
		return nil, fmt.Errorf("failed to log parameters: %w", err)
	}

	return run, nil
}

// EvaluationResultsToMetrics converts evaluation results to MLflow metrics.
func EvaluationResultsToMetrics(results *Results) map[string]float64 {
	metrics := make(map[string]float64)

	// Add duration
	if !results.EndTime.IsZero() && !results.StartTime.IsZero() {
		metrics["eval_duration_seconds"] = results.EndTime.Sub(results.StartTime).Seconds()
	}

	// Add test counts
	metrics["eval_total_tests"] = float64(results.TotalTests)

	// Add per-scorer summary statistics
	for scorerName, stats := range results.Summary {
		prefix := fmt.Sprintf("eval_%s", scorerName)
		metrics[prefix+"_mean"] = stats.Mean
		metrics[prefix+"_pass_rate"] = stats.PassRate
		metrics[prefix+"_error_rate"] = stats.ErrorRate
		if stats.StdDev > 0 {
			metrics[prefix+"_std_dev"] = stats.StdDev
		}
		metrics[prefix+"_count"] = float64(stats.Count)
	}

	return metrics
}

// WithMLflowExport returns a RunOption that exports results to MLflow.
func WithMLflowExport(client *mlflowclient.Client, experimentID string) RunOption {
	return func(c *runConfig) {
		c.mlflowClient = client
		c.mlflowExperimentID = experimentID
	}
}
