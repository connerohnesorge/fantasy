package eval

import (
	"testing"
	"time"
)

func TestEvaluationResultsToMetrics(t *testing.T) {
	results := &Results{
		StartTime:  time.Now().Add(-10 * time.Second),
		EndTime:    time.Now(),
		TotalTests: 100,
		Summary: map[string]ScorerStats{
			"exact_match": {
				Mean:       0.85,
				PassRate:   0.9,
				StdDev:     0.1,
				ErrorRate:  0.05,
				ErrorCount: 5,
				Count:      100,
			},
			"contains": {
				Mean:       0.95,
				PassRate:   0.95,
				StdDev:     0.05,
				ErrorRate:  0.0,
				ErrorCount: 0,
				Count:      100,
			},
		},
	}

	metrics := EvaluationResultsToMetrics(results)

	// Check duration
	if _, ok := metrics["eval_duration_seconds"]; !ok {
		t.Error("expected eval_duration_seconds metric")
	}

	// Check total tests
	if metrics["eval_total_tests"] != 100.0 {
		t.Errorf("eval_total_tests = %f, want 100.0", metrics["eval_total_tests"])
	}

	// Check exact_match metrics
	if metrics["eval_exact_match_mean"] != 0.85 {
		t.Errorf("eval_exact_match_mean = %f, want 0.85", metrics["eval_exact_match_mean"])
	}
	if metrics["eval_exact_match_pass_rate"] != 0.9 {
		t.Errorf("eval_exact_match_pass_rate = %f, want 0.9", metrics["eval_exact_match_pass_rate"])
	}
	if metrics["eval_exact_match_std_dev"] != 0.1 {
		t.Errorf("eval_exact_match_std_dev = %f, want 0.1", metrics["eval_exact_match_std_dev"])
	}
	if metrics["eval_exact_match_error_rate"] != 0.05 {
		t.Errorf("eval_exact_match_error_rate = %f, want 0.05", metrics["eval_exact_match_error_rate"])
	}
	if metrics["eval_exact_match_count"] != 100.0 {
		t.Errorf("eval_exact_match_count = %f, want 100.0", metrics["eval_exact_match_count"])
	}

	// Check contains metrics
	if metrics["eval_contains_mean"] != 0.95 {
		t.Errorf("eval_contains_mean = %f, want 0.95", metrics["eval_contains_mean"])
	}
}

func TestEvaluationResultsToMetrics_ZeroStdDev(t *testing.T) {
	results := &Results{
		StartTime:  time.Now(),
		EndTime:    time.Now(),
		TotalTests: 10,
		Summary: map[string]ScorerStats{
			"test_scorer": {
				Mean:     1.0,
				PassRate: 1.0,
				StdDev:   0.0, // No variation
				Count:    10,
			},
		},
	}

	metrics := EvaluationResultsToMetrics(results)

	// StdDev of 0 should not be included
	if _, ok := metrics["eval_test_scorer_std_dev"]; ok {
		t.Error("std_dev should not be included when zero")
	}
}

func TestScoreToAssessment(t *testing.T) {
	tests := []struct {
		name        string
		score       Score
		scorerName  string
		traceID     string
		spanID      string
		tcIndex     int
		wantIsError bool
	}{
		{
			name: "successful bool score",
			score: Score{
				Value:     true,
				Rationale: "exact match",
			},
			scorerName:  "exact_match",
			traceID:     "trace-123",
			spanID:      "span-456",
			tcIndex:     0,
			wantIsError: false,
		},
		{
			name: "successful float score",
			score: Score{
				Value:     0.85,
				Rationale: "85% similarity",
			},
			scorerName:  "similarity",
			traceID:     "trace-123",
			spanID:      "",
			tcIndex:     5,
			wantIsError: false,
		},
		{
			name: "error score",
			score: Score{
				Error:     &ValidationError{Field: "test", Message: "test error"},
				Rationale: "scorer failed",
			},
			scorerName:  "failing_scorer",
			traceID:     "trace-123",
			spanID:      "",
			tcIndex:     0,
			wantIsError: true,
		},
		{
			name: "score with metadata",
			score: Score{
				Value:     true,
				Rationale: "passed",
				Metadata: map[string]any{
					"confidence": 0.95,
					"details":    "some details",
				},
			},
			scorerName:  "complex_scorer",
			traceID:     "trace-123",
			spanID:      "",
			tcIndex:     0,
			wantIsError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := ScoreToAssessment(tt.score, tt.scorerName, tt.traceID, tt.spanID, tt.tcIndex)

			if assessment.Name != tt.scorerName {
				t.Errorf("Name = %s, want %s", assessment.Name, tt.scorerName)
			}

			if assessment.TraceID != tt.traceID {
				t.Errorf("TraceID = %s, want %s", assessment.TraceID, tt.traceID)
			}

			if assessment.SpanID != tt.spanID {
				t.Errorf("SpanID = %s, want %s", assessment.SpanID, tt.spanID)
			}

			if assessment.Rationale != tt.score.Rationale {
				t.Errorf("Rationale = %s, want %s", assessment.Rationale, tt.score.Rationale)
			}

			if assessment.Source == nil {
				t.Error("Source should not be nil")
			} else if assessment.Source.SourceID != "eval/"+tt.scorerName {
				t.Errorf("Source.SourceID = %s, want eval/%s", assessment.Source.SourceID, tt.scorerName)
			}

			// Check metadata
			if assessment.Metadata["test_case_index"] != "0" && assessment.Metadata["test_case_index"] != "5" {
				t.Errorf("Metadata[test_case_index] = %s", assessment.Metadata["test_case_index"])
			}

			// Check error vs non-error
			if tt.wantIsError {
				if assessment.Feedback == nil || assessment.Feedback.Error == nil {
					t.Error("expected error in feedback")
				}
			} else {
				if assessment.Feedback == nil {
					t.Error("expected feedback")
				} else if assessment.Feedback.Error != nil {
					t.Errorf("unexpected error in feedback: %v", assessment.Feedback.Error)
				}
			}
		})
	}
}
