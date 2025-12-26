package eval

import (
	"fmt"
	"time"

	"charm.land/fantasy/mlflowclient"
	pb "charm.land/fantasy/proto/gen/mlflow"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MLflowExporter handles exporting evaluation results to MLflow.
type MLflowExporter struct {
	client *mlflowclient.Client
}

// NewMLflowExporter creates a new exporter with the given MLflow client.
func NewMLflowExporter(client *mlflowclient.Client) *MLflowExporter {
	return &MLflowExporter{
		client: client,
	}
}

// ScoreToAssessment converts an eval.Score to a proto Assessment message.
// The scorerName is used as the assessment name.
// The isLLMJudge flag determines the source type (LLM_JUDGE vs CODE).
func ScoreToAssessment(scorerName string, score Score, isLLMJudge bool) *pb.Assessment {
	// Determine source type
	sourceType := pb.AssessmentSource_CODE
	if isLLMJudge {
		sourceType = pb.AssessmentSource_LLM_JUDGE
	}

	assessment := &pb.Assessment{
		AssessmentName: &scorerName,
		Source: &pb.AssessmentSource{
			SourceType: &sourceType,
			SourceId:   &scorerName,
		},
	}

	// Convert the score value to Feedback
	feedback := &pb.Feedback{}

	// Handle error case
	if score.Error != nil {
		// Convert error to AssessmentError
		errMsg := score.Error.Error()
		feedback.Error = &pb.AssessmentError{
			ErrorCode:    stringPtr("SCORER_ERROR"),
			ErrorMessage: &errMsg,
		}
		// Value should be null when error is present
		feedback.Value = nil
	} else {
		// Convert score value to structpb.Value
		value, err := anyToProtoValue(score.Value)
		if err != nil {
			// If conversion fails, treat as error
			errMsg := fmt.Sprintf("failed to convert score value: %v", err)
			feedback.Error = &pb.AssessmentError{
				ErrorCode:    stringPtr("VALUE_CONVERSION_ERROR"),
				ErrorMessage: &errMsg,
			}
		} else {
			feedback.Value = value
		}
	}

	// Set feedback as the assessment value
	assessment.Value = &pb.Assessment_Feedback{
		Feedback: feedback,
	}

	// Add rationale if present
	if score.Rationale != "" {
		assessment.Rationale = &score.Rationale
	}

	// Convert metadata to map[string]string
	if len(score.Metadata) > 0 {
		metadata := make(map[string]string)
		for k, v := range score.Metadata {
			metadata[k] = fmt.Sprintf("%v", v)
		}
		assessment.Metadata = metadata
	}

	return assessment
}

// DatasetToParams converts a Dataset to MLflow Params for logging.
// Returns a slice of Param messages containing dataset metadata.
func DatasetToParams(dataset *Dataset) []*pb.Param {
	params := []*pb.Param{}

	// Add dataset name
	params = append(params, &pb.Param{
		Key:   stringPtr("eval.dataset_name"),
		Value: &dataset.Name,
	})

	// Add test case count
	testCaseCount := fmt.Sprintf("%d", len(dataset.TestCases))
	params = append(params, &pb.Param{
		Key:   stringPtr("eval.test_case_count"),
		Value: &testCaseCount,
	})

	// Add dataset metadata if present
	if len(dataset.Metadata) > 0 {
		for k, v := range dataset.Metadata {
			key := fmt.Sprintf("eval.dataset.%s", k)
			value := fmt.Sprintf("%v", v)
			params = append(params, &pb.Param{
				Key:   &key,
				Value: &value,
			})
		}
	}

	return params
}

// EvaluationResultsToMetrics converts evaluation Results to MLflow Metric messages.
// These metrics capture aggregate statistics for each scorer.
func EvaluationResultsToMetrics(results *Results) []*pb.Metric {
	metrics := []*pb.Metric{}
	timestamp := results.EndTime.UnixMilli()

	for scorerName, stats := range results.Summary {
		// Pass rate metric
		metrics = append(metrics, &pb.Metric{
			Key:       stringPtr(fmt.Sprintf("eval.%s.pass_rate", scorerName)),
			Value:     &stats.PassRate,
			Timestamp: &timestamp,
			Step:      int64Ptr(0),
		})

		// Mean metric
		metrics = append(metrics, &pb.Metric{
			Key:       stringPtr(fmt.Sprintf("eval.%s.mean", scorerName)),
			Value:     &stats.Mean,
			Timestamp: &timestamp,
			Step:      int64Ptr(0),
		})

		// Standard deviation metric
		metrics = append(metrics, &pb.Metric{
			Key:       stringPtr(fmt.Sprintf("eval.%s.std_dev", scorerName)),
			Value:     &stats.StdDev,
			Timestamp: &timestamp,
			Step:      int64Ptr(0),
		})

		// Error rate metric
		metrics = append(metrics, &pb.Metric{
			Key:       stringPtr(fmt.Sprintf("eval.%s.error_rate", scorerName)),
			Value:     &stats.ErrorRate,
			Timestamp: &timestamp,
			Step:      int64Ptr(0),
		})

		// Error count metric
		errorCount := float64(stats.ErrorCount)
		metrics = append(metrics, &pb.Metric{
			Key:       stringPtr(fmt.Sprintf("eval.%s.error_count", scorerName)),
			Value:     &errorCount,
			Timestamp: &timestamp,
			Step:      int64Ptr(0),
		})

		// Total count metric
		totalCount := float64(stats.Count)
		metrics = append(metrics, &pb.Metric{
			Key:       stringPtr(fmt.Sprintf("eval.%s.count", scorerName)),
			Value:     &totalCount,
			Timestamp: &timestamp,
			Step:      int64Ptr(0),
		})
	}

	// Overall metrics
	metrics = append(metrics, &pb.Metric{
		Key:       stringPtr("eval.total_tests"),
		Value:     float64Ptr(float64(results.TotalTests)),
		Timestamp: &timestamp,
		Step:      int64Ptr(0),
	})

	duration := results.EndTime.Sub(results.StartTime).Seconds()
	metrics = append(metrics, &pb.Metric{
		Key:       stringPtr("eval.duration_seconds"),
		Value:     &duration,
		Timestamp: &timestamp,
		Step:      int64Ptr(0),
	})

	return metrics
}

// anyToProtoValue converts a Go value to a protobuf Value.
// Supports bool, float64, int, int64, string, and []string.
func anyToProtoValue(v any) (*structpb.Value, error) {
	if v == nil {
		return structpb.NewNullValue(), nil
	}

	switch val := v.(type) {
	case bool:
		return structpb.NewBoolValue(val), nil
	case float64:
		return structpb.NewNumberValue(val), nil
	case int:
		return structpb.NewNumberValue(float64(val)), nil
	case int64:
		return structpb.NewNumberValue(float64(val)), nil
	case string:
		return structpb.NewStringValue(val), nil
	case []string:
		// Convert to list of string values
		values := make([]any, len(val))
		for i, s := range val {
			values[i] = s
		}
		return structpb.NewListValue(&structpb.ListValue{
			Values: func() []*structpb.Value {
				result := make([]*structpb.Value, len(values))
				for i, v := range values {
					result[i] = structpb.NewStringValue(v.(string))
				}
				return result
			}(),
		}), nil
	default:
		// For unsupported types, convert to string
		return structpb.NewStringValue(fmt.Sprintf("%v", val)), nil
	}
}

// Helper functions to create pointers

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func timestampPtr(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
