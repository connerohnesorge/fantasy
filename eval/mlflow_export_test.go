package eval

import (
	"errors"
	"testing"
	"time"

	pb "charm.land/fantasy/proto/gen/mlflow"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestScoreToAssessment(t *testing.T) {
	tests := []struct {
		name        string
		scorerName  string
		score       Score
		isLLMJudge  bool
		wantSource  pb.AssessmentSource_SourceType
		wantError   bool
		wantValue   bool
		checkFields func(*testing.T, *pb.Assessment)
	}{
		{
			name:       "boolean score with CODE source",
			scorerName: "exact_match",
			score: Score{
				Value:     true,
				Rationale: "Strings match exactly",
				Metadata:  map[string]any{"confidence": 1.0},
			},
			isLLMJudge: false,
			wantSource: pb.AssessmentSource_CODE,
			wantError:  false,
			wantValue:  true,
			checkFields: func(t *testing.T, a *pb.Assessment) {
				if a.GetRationale() != "Strings match exactly" {
					t.Errorf("Expected rationale 'Strings match exactly', got '%s'", a.GetRationale())
				}
				if len(a.Metadata) != 1 {
					t.Errorf("Expected 1 metadata entry, got %d", len(a.Metadata))
				}
				feedback := a.GetFeedback()
				if feedback == nil {
					t.Fatal("Expected feedback to be set")
				}
				if feedback.Value == nil {
					t.Fatal("Expected feedback value to be set")
				}
				if !feedback.Value.GetBoolValue() {
					t.Error("Expected feedback value to be true")
				}
			},
		},
		{
			name:       "numeric score with LLM_JUDGE source",
			scorerName: "correctness",
			score: Score{
				Value:     0.85,
				Rationale: "Answer is mostly correct",
				Metadata:  map[string]any{"model": "gpt-4"},
			},
			isLLMJudge: true,
			wantSource: pb.AssessmentSource_LLM_JUDGE,
			wantError:  false,
			wantValue:  true,
			checkFields: func(t *testing.T, a *pb.Assessment) {
				feedback := a.GetFeedback()
				if feedback == nil {
					t.Fatal("Expected feedback to be set")
				}
				if feedback.Value == nil {
					t.Fatal("Expected feedback value to be set")
				}
				if feedback.Value.GetNumberValue() != 0.85 {
					t.Errorf("Expected feedback value 0.85, got %f", feedback.Value.GetNumberValue())
				}
			},
		},
		{
			name:       "string score",
			scorerName: "category",
			score: Score{
				Value:     "excellent",
				Rationale: "Output quality is excellent",
			},
			isLLMJudge: false,
			wantSource: pb.AssessmentSource_CODE,
			wantError:  false,
			wantValue:  true,
			checkFields: func(t *testing.T, a *pb.Assessment) {
				feedback := a.GetFeedback()
				if feedback == nil {
					t.Fatal("Expected feedback to be set")
				}
				if feedback.Value == nil {
					t.Fatal("Expected feedback value to be set")
				}
				if feedback.Value.GetStringValue() != "excellent" {
					t.Errorf("Expected feedback value 'excellent', got '%s'", feedback.Value.GetStringValue())
				}
			},
		},
		{
			name:       "score with error",
			scorerName: "failing_scorer",
			score: Score{
				Error:     errors.New("scorer failed"),
				Rationale: "Execution error",
			},
			isLLMJudge: false,
			wantSource: pb.AssessmentSource_CODE,
			wantError:  true,
			wantValue:  false,
			checkFields: func(t *testing.T, a *pb.Assessment) {
				feedback := a.GetFeedback()
				if feedback == nil {
					t.Fatal("Expected feedback to be set")
				}
				if feedback.Value != nil {
					t.Error("Expected feedback value to be nil when error is present")
				}
				if feedback.Error == nil {
					t.Fatal("Expected feedback error to be set")
				}
				if feedback.Error.GetErrorCode() != "SCORER_ERROR" {
					t.Errorf("Expected error code 'SCORER_ERROR', got '%s'", feedback.Error.GetErrorCode())
				}
				if feedback.Error.GetErrorMessage() != "scorer failed" {
					t.Errorf("Expected error message 'scorer failed', got '%s'", feedback.Error.GetErrorMessage())
				}
			},
		},
		{
			name:       "integer score",
			scorerName: "step_count",
			score: Score{
				Value:     5,
				Rationale: "Agent took 5 steps",
			},
			isLLMJudge: false,
			wantSource: pb.AssessmentSource_CODE,
			wantError:  false,
			wantValue:  true,
			checkFields: func(t *testing.T, a *pb.Assessment) {
				feedback := a.GetFeedback()
				if feedback == nil {
					t.Fatal("Expected feedback to be set")
				}
				if feedback.Value == nil {
					t.Fatal("Expected feedback value to be set")
				}
				if feedback.Value.GetNumberValue() != 5.0 {
					t.Errorf("Expected feedback value 5.0, got %f", feedback.Value.GetNumberValue())
				}
			},
		},
		{
			name:       "nil score value",
			scorerName: "null_scorer",
			score: Score{
				Value:     nil,
				Rationale: "No value available",
			},
			isLLMJudge: false,
			wantSource: pb.AssessmentSource_CODE,
			wantError:  false,
			wantValue:  true,
			checkFields: func(t *testing.T, a *pb.Assessment) {
				feedback := a.GetFeedback()
				if feedback == nil {
					t.Fatal("Expected feedback to be set")
				}
				if feedback.Value == nil {
					t.Fatal("Expected feedback value to be set (as null)")
				}
				if feedback.Value.GetNullValue() != structpb.NullValue_NULL_VALUE {
					t.Error("Expected feedback value to be null")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := ScoreToAssessment(tt.scorerName, tt.score, tt.isLLMJudge)

			// Check assessment name
			if assessment.GetAssessmentName() != tt.scorerName {
				t.Errorf("Expected assessment name '%s', got '%s'", tt.scorerName, assessment.GetAssessmentName())
			}

			// Check source type
			if assessment.Source.GetSourceType() != tt.wantSource {
				t.Errorf("Expected source type %v, got %v", tt.wantSource, assessment.Source.GetSourceType())
			}

			// Check source ID
			if assessment.Source.GetSourceId() != tt.scorerName {
				t.Errorf("Expected source ID '%s', got '%s'", tt.scorerName, assessment.Source.GetSourceId())
			}

			// Run custom field checks
			if tt.checkFields != nil {
				tt.checkFields(t, assessment)
			}
		})
	}
}

func TestDatasetToParams(t *testing.T) {
	tests := []struct {
		name       string
		dataset    *Dataset
		wantParams int
		checkFunc  func(*testing.T, []*pb.Param)
	}{
		{
			name: "basic dataset",
			dataset: &Dataset{
				Name:      "test-dataset",
				TestCases: []TestCase{{}, {}, {}},
			},
			wantParams: 2, // name + count
			checkFunc: func(t *testing.T, params []*pb.Param) {
				// Check dataset name param
				found := false
				for _, p := range params {
					if p.GetKey() == "eval.dataset_name" {
						if p.GetValue() != "test-dataset" {
							t.Errorf("Expected dataset name 'test-dataset', got '%s'", p.GetValue())
						}
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected eval.dataset_name param not found")
				}

				// Check test case count
				found = false
				for _, p := range params {
					if p.GetKey() == "eval.test_case_count" {
						if p.GetValue() != "3" {
							t.Errorf("Expected test case count '3', got '%s'", p.GetValue())
						}
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected eval.test_case_count param not found")
				}
			},
		},
		{
			name: "dataset with metadata",
			dataset: &Dataset{
				Name:      "metadata-dataset",
				TestCases: []TestCase{{}},
				Metadata: map[string]any{
					"version": "1.0",
					"source":  "production",
				},
			},
			wantParams: 4, // name + count + 2 metadata
			checkFunc: func(t *testing.T, params []*pb.Param) {
				// Check metadata params
				metadataFound := 0
				for _, p := range params {
					key := p.GetKey()
					if key == "eval.dataset.version" {
						if p.GetValue() != "1.0" {
							t.Errorf("Expected version '1.0', got '%s'", p.GetValue())
						}
						metadataFound++
					} else if key == "eval.dataset.source" {
						if p.GetValue() != "production" {
							t.Errorf("Expected source 'production', got '%s'", p.GetValue())
						}
						metadataFound++
					}
				}
				if metadataFound != 2 {
					t.Errorf("Expected 2 metadata params, found %d", metadataFound)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DatasetToParams(tt.dataset)

			if len(params) != tt.wantParams {
				t.Errorf("Expected %d params, got %d", tt.wantParams, len(params))
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, params)
			}
		})
	}
}

func TestEvaluationResultsToMetrics(t *testing.T) {
	now := time.Now()
	results := &Results{
		TestCases: []TestCaseResult{
			{Scores: map[string]Score{"scorer1": {Value: true}}},
			{Scores: map[string]Score{"scorer1": {Value: false}}},
			{Scores: map[string]Score{"scorer1": {Value: true}}},
		},
		Summary: map[string]ScorerStats{
			"scorer1": {
				PassRate:   0.67,
				Mean:       0.67,
				StdDev:     0.47,
				ErrorRate:  0.0,
				ErrorCount: 0,
				Count:      3,
			},
			"scorer2": {
				PassRate:   1.0,
				Mean:       0.95,
				StdDev:     0.05,
				ErrorRate:  0.0,
				ErrorCount: 0,
				Count:      2,
			},
		},
		Errors:     []EvalError{},
		StartTime:  now.Add(-10 * time.Second),
		EndTime:    now,
		TotalTests: 3,
	}

	metrics := EvaluationResultsToMetrics(results)

	// Expected metrics:
	// - Per scorer: pass_rate, mean, std_dev, error_rate, error_count, count (6 each)
	// - Overall: total_tests, duration_seconds (2)
	// Total: 6*2 + 2 = 14
	expectedMetrics := 14
	if len(metrics) != expectedMetrics {
		t.Errorf("Expected %d metrics, got %d", expectedMetrics, len(metrics))
	}

	// Check for specific metrics
	metricMap := make(map[string]float64)
	for _, m := range metrics {
		metricMap[m.GetKey()] = m.GetValue()
	}

	// Check scorer1 metrics
	if val, ok := metricMap["eval.scorer1.pass_rate"]; !ok {
		t.Error("Expected eval.scorer1.pass_rate metric")
	} else if val != 0.67 {
		t.Errorf("Expected pass_rate 0.67, got %f", val)
	}

	if val, ok := metricMap["eval.scorer1.mean"]; !ok {
		t.Error("Expected eval.scorer1.mean metric")
	} else if val != 0.67 {
		t.Errorf("Expected mean 0.67, got %f", val)
	}

	// Check scorer2 metrics
	if val, ok := metricMap["eval.scorer2.pass_rate"]; !ok {
		t.Error("Expected eval.scorer2.pass_rate metric")
	} else if val != 1.0 {
		t.Errorf("Expected pass_rate 1.0, got %f", val)
	}

	// Check overall metrics
	if val, ok := metricMap["eval.total_tests"]; !ok {
		t.Error("Expected eval.total_tests metric")
	} else if val != 3.0 {
		t.Errorf("Expected total_tests 3, got %f", val)
	}

	if val, ok := metricMap["eval.duration_seconds"]; !ok {
		t.Error("Expected eval.duration_seconds metric")
	} else if val < 9.0 || val > 11.0 {
		t.Errorf("Expected duration_seconds around 10, got %f", val)
	}

	// Verify all metrics have timestamps
	for _, m := range metrics {
		if m.Timestamp == nil {
			t.Errorf("Metric %s missing timestamp", m.GetKey())
		}
		if m.Step == nil || m.GetStep() != 0 {
			t.Errorf("Metric %s has invalid step (expected 0)", m.GetKey())
		}
	}
}

func TestAnyToProtoValue(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		wantError bool
		checkFunc func(*testing.T, *structpb.Value)
	}{
		{
			name:      "nil value",
			input:     nil,
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				if v.GetNullValue() != structpb.NullValue_NULL_VALUE {
					t.Error("Expected null value")
				}
			},
		},
		{
			name:      "boolean true",
			input:     true,
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				if !v.GetBoolValue() {
					t.Error("Expected true boolean value")
				}
			},
		},
		{
			name:      "boolean false",
			input:     false,
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				if v.GetBoolValue() {
					t.Error("Expected false boolean value")
				}
			},
		},
		{
			name:      "float64",
			input:     3.14,
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				if v.GetNumberValue() != 3.14 {
					t.Errorf("Expected 3.14, got %f", v.GetNumberValue())
				}
			},
		},
		{
			name:      "int",
			input:     42,
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				if v.GetNumberValue() != 42.0 {
					t.Errorf("Expected 42.0, got %f", v.GetNumberValue())
				}
			},
		},
		{
			name:      "int64",
			input:     int64(100),
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				if v.GetNumberValue() != 100.0 {
					t.Errorf("Expected 100.0, got %f", v.GetNumberValue())
				}
			},
		},
		{
			name:      "string",
			input:     "test string",
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				if v.GetStringValue() != "test string" {
					t.Errorf("Expected 'test string', got '%s'", v.GetStringValue())
				}
			},
		},
		{
			name:      "string slice",
			input:     []string{"a", "b", "c"},
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				list := v.GetListValue()
				if list == nil {
					t.Fatal("Expected list value")
				}
				if len(list.Values) != 3 {
					t.Errorf("Expected 3 values, got %d", len(list.Values))
				}
				if list.Values[0].GetStringValue() != "a" {
					t.Errorf("Expected 'a', got '%s'", list.Values[0].GetStringValue())
				}
			},
		},
		{
			name:      "empty string slice",
			input:     []string{},
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				list := v.GetListValue()
				if list == nil {
					t.Fatal("Expected list value")
				}
				if len(list.Values) != 0 {
					t.Errorf("Expected 0 values, got %d", len(list.Values))
				}
			},
		},
		{
			name:      "unsupported type converts to string",
			input:     struct{ Name string }{"test"},
			wantError: false,
			checkFunc: func(t *testing.T, v *structpb.Value) {
				// Should be converted to string representation
				str := v.GetStringValue()
				if str == "" {
					t.Error("Expected non-empty string for unsupported type")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := anyToProtoValue(tt.input)

			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil && value != nil {
				tt.checkFunc(t, value)
			}
		})
	}
}

func TestMLflowExporter(t *testing.T) {
	// Test NewMLflowExporter
	exporter := NewMLflowExporter(nil)
	if exporter == nil {
		t.Fatal("Expected non-nil exporter")
	}
	if exporter.client != nil {
		t.Error("Expected nil client")
	}
}
