package eval

import (
	"context"
	"fmt"
	"time"

	"charm.land/fantasy/mlflow"
	pb "charm.land/fantasy/proto/gen/mlflow"
	"charm.land/fantasy/tracing"
)

// ScoreToAssessment converts a Score to an MLflow Assessment.
// traceID: The MLflow trace ID that was created for this test case.
// scorerName: Name of the scorer.
func ScoreToAssessment(score Score, traceID string, scorerName string) (*mlflow.Assessment, error) {
	if traceID == "" {
		return nil, fmt.Errorf("traceID is required")
	}
	if scorerName == "" {
		return nil, fmt.Errorf("scorerName is required")
	}

	assessment := &mlflow.Assessment{
		Name:    scorerName,
		TraceID: traceID,
	}

	// Determine source type based on scorer characteristics
	// Heuristic scorers are marked as HUMAN, LLM judges as LLM_JUDGE
	// For now, we'll use a simple heuristic: if the score has a rationale that looks
	// LLM-generated (longer text), treat it as LLM_JUDGE, otherwise HUMAN
	sourceType := "HUMAN"
	if len(score.Rationale) > 50 { // Simple heuristic
		sourceType = "LLM_JUDGE"
	}

	assessment.Source = mlflow.AssessmentSource{
		SourceType: sourceType,
		SourceID:   scorerName,
	}

	// Set rationale
	assessment.Rationale = score.Rationale

	// Convert metadata to string map
	if score.Metadata != nil {
		assessment.Metadata = make(map[string]string)
		for k, v := range score.Metadata {
			assessment.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	// Handle error case
	if score.Error != nil {
		assessment.Feedback = &mlflow.FeedbackValue{
			Value: "FAIL",
			Error: &mlflow.AssessmentError{
				ErrorMessage: score.Error.Error(),
			},
		}
		return assessment, nil
	}

	// Convert score value to PASS/FAIL based on threshold
	var feedbackValue any
	if numVal, ok := convertToFloat64(score.Value); ok {
		if numVal > 0.5 {
			feedbackValue = "PASS"
		} else {
			feedbackValue = "FAIL"
		}
	} else if boolVal, ok := score.Value.(bool); ok {
		if boolVal {
			feedbackValue = "PASS"
		} else {
			feedbackValue = "FAIL"
		}
	} else {
		// For other types (string), use the value directly
		feedbackValue = score.Value
	}

	assessment.Feedback = &mlflow.FeedbackValue{
		Value: feedbackValue,
	}

	return assessment, nil
}

// DatasetToRun converts a Dataset to an MLflow Run for logging.
// experimentID: Target experiment.
func DatasetToRun(ctx context.Context, client *mlflow.Client, dataset *Dataset, experimentID string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("client is required")
	}
	if dataset == nil {
		return "", fmt.Errorf("dataset is required")
	}
	if experimentID == "" {
		return "", fmt.Errorf("experimentID is required")
	}

	// Validate dataset
	if err := dataset.Validate(); err != nil {
		return "", fmt.Errorf("invalid dataset: %w", err)
	}

	// Create run with tags
	tags := map[string]string{
		"dataset_name":      dataset.Name,
		"test_case_count":   fmt.Sprintf("%d", len(dataset.TestCases)),
		"created_timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Add dataset metadata as tags if available
	if dataset.Metadata != nil {
		for k, v := range dataset.Metadata {
			tags[fmt.Sprintf("dataset.%s", k)] = fmt.Sprintf("%v", v)
		}
	}

	run, err := client.CreateRun(ctx, experimentID,
		mlflow.WithRunName(fmt.Sprintf("eval-%s", dataset.Name)),
		mlflow.WithTags(tags),
		mlflow.WithStartTime(time.Now().UnixMilli()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create run: %w", err)
	}

	if run.Info == nil || run.Info.RunId == nil {
		return "", fmt.Errorf("run created but RunId is nil")
	}

	return *run.Info.RunId, nil
}

// ResultsToMetrics converts evaluation results to MLflow metrics.
func ResultsToMetrics(results *Results) []pb.Metric {
	if results == nil {
		return nil
	}

	metrics := make([]pb.Metric, 0)
	timestamp := time.Now().UnixMilli()

	// Per-scorer metrics
	for scorerName, stats := range results.Summary {
		// Pass rate metric
		passRateKey := fmt.Sprintf("%s_pass_rate", scorerName)
		passRate := stats.PassRate
		metrics = append(metrics, pb.Metric{
			Key:       &passRateKey,
			Value:     &passRate,
			Timestamp: &timestamp,
		})

		// Mean score metric
		meanKey := fmt.Sprintf("%s_mean_score", scorerName)
		mean := stats.Mean
		metrics = append(metrics, pb.Metric{
			Key:       &meanKey,
			Value:     &mean,
			Timestamp: &timestamp,
		})

		// Error rate metric
		errorRateKey := fmt.Sprintf("%s_error_rate", scorerName)
		errorRate := stats.ErrorRate
		metrics = append(metrics, pb.Metric{
			Key:       &errorRateKey,
			Value:     &errorRate,
			Timestamp: &timestamp,
		})

		// Standard deviation (if available)
		if stats.StdDev > 0 {
			stdDevKey := fmt.Sprintf("%s_std_dev", scorerName)
			stdDev := stats.StdDev
			metrics = append(metrics, pb.Metric{
				Key:       &stdDevKey,
				Value:     &stdDev,
				Timestamp: &timestamp,
			})
		}
	}

	// Overall metrics (averaged across all scorers)
	if len(results.Summary) > 0 {
		var totalPassRate, totalMeanScore, totalErrorRate float64
		count := 0

		for _, stats := range results.Summary {
			totalPassRate += stats.PassRate
			totalMeanScore += stats.Mean
			totalErrorRate += stats.ErrorRate
			count++
		}

		if count > 0 {
			overallPassRateKey := "overall_pass_rate"
			overallPassRate := totalPassRate / float64(count)
			metrics = append(metrics, pb.Metric{
				Key:       &overallPassRateKey,
				Value:     &overallPassRate,
				Timestamp: &timestamp,
			})

			overallMeanKey := "overall_mean_score"
			overallMean := totalMeanScore / float64(count)
			metrics = append(metrics, pb.Metric{
				Key:       &overallMeanKey,
				Value:     &overallMean,
				Timestamp: &timestamp,
			})

			overallErrorRateKey := "overall_error_rate"
			overallErrorRate := totalErrorRate / float64(count)
			metrics = append(metrics, pb.Metric{
				Key:       &overallErrorRateKey,
				Value:     &overallErrorRate,
				Timestamp: &timestamp,
			})
		}
	}

	// Total test count
	totalTestsKey := "total_tests"
	totalTests := float64(results.TotalTests)
	metrics = append(metrics, pb.Metric{
		Key:       &totalTestsKey,
		Value:     &totalTests,
		Timestamp: &timestamp,
	})

	// Evaluation duration (in seconds)
	if !results.EndTime.IsZero() && !results.StartTime.IsZero() {
		durationKey := "evaluation_duration_seconds"
		duration := results.EndTime.Sub(results.StartTime).Seconds()
		metrics = append(metrics, pb.Metric{
			Key:       &durationKey,
			Value:     &duration,
			Timestamp: &timestamp,
		})
	}

	return metrics
}

// extractTraceID extracts the trace ID from a trace object (which is *tracing.Trace).
func extractTraceID(trace any) string {
	if trace == nil {
		return ""
	}
	if t, ok := trace.(*tracing.Trace); ok {
		return t.TraceID
	}
	return ""
}

// mlflowExportConfig holds the configuration for MLflow export.
type mlflowExportConfig struct {
	client       *mlflow.Client
	experimentID string
	runID        string
}

// WithMLflowExport enables automatic export of evaluation results to MLflow.
// This option will:
// - Create assessments for each scored test case.
// - Link assessments to traces via trace IDs.
// - Log aggregate metrics to the run.
func WithMLflowExport(client *mlflow.Client, experimentID string) Option {
	return func(e *Evaluator) {
		if e.mlflowConfig == nil {
			e.mlflowConfig = &mlflowExportConfig{
				client:       client,
				experimentID: experimentID,
			}
		}
	}
}

// exportAssessments creates MLflow assessments for all scored test cases.
func (e *Evaluator) exportAssessments(ctx context.Context, results *Results) {
	if e.mlflowConfig == nil || e.mlflowConfig.client == nil {
		return
	}

	for _, tcResult := range results.TestCases {
		// Extract trace ID from the trace
		traceID := extractTraceID(tcResult.Trace)
		if traceID == "" {
			// Skip if no trace ID available
			continue
		}

		// Create assessments for each scorer
		for scorerName, score := range tcResult.Scores {
			assessment, err := ScoreToAssessment(score, traceID, scorerName)
			if err != nil {
				// Log error but don't fail evaluation
				results.Errors = append(results.Errors, EvalError{
					Phase:   "export",
					Message: fmt.Sprintf("failed to convert score to assessment for scorer %s: %v", scorerName, err),
					Cause:   err,
				})
				continue
			}

			// Create the assessment in MLflow
			_, err = e.mlflowConfig.client.CreateAssessment(ctx, traceID, assessment)
			if err != nil {
				// Log error but don't fail evaluation
				results.Errors = append(results.Errors, EvalError{
					Phase:   "export",
					Message: fmt.Sprintf("failed to create assessment in MLflow for scorer %s: %v", scorerName, err),
					Cause:   err,
				})
			}
		}
	}
}

// exportMetrics logs aggregate metrics to the MLflow run.
func (e *Evaluator) exportMetrics(ctx context.Context, results *Results) {
	if e.mlflowConfig == nil || e.mlflowConfig.client == nil || e.mlflowConfig.runID == "" {
		return
	}

	metrics := ResultsToMetrics(results)
	if len(metrics) == 0 {
		return
	}

	err := e.mlflowConfig.client.LogBatch(ctx, e.mlflowConfig.runID, metrics, nil, nil)
	if err != nil {
		// Log error but don't fail evaluation
		results.Errors = append(results.Errors, EvalError{
			Phase:   "export",
			Message: fmt.Sprintf("failed to log metrics to MLflow: %v", err),
			Cause:   err,
		})
	}
}
