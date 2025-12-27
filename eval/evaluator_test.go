package eval

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEvaluator_Run(t *testing.T) {
	dataset := &Dataset{
		Name: "test-dataset",
		TestCases: []*TestCase{
			{
				Inputs:       map[string]any{"prompt": "What is 2+2?"},
				Expectations: map[string]any{"output": "4"},
				Outputs:      map[string]any{"output": "4"},
			},
			{
				Inputs:       map[string]any{"prompt": "What is 3+3?"},
				Expectations: map[string]any{"output": "6"},
				Outputs:      map[string]any{"output": "6"},
			},
		},
	}

	scorers := []Scorer{
		ExactMatchScorer("output"),
	}

	evaluator := NewEvaluator()
	results, err := evaluator.Run(context.Background(), dataset, scorers)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if results.TotalTests != 2 {
		t.Errorf("TotalTests = %d, want 2", results.TotalTests)
	}

	stats, ok := results.Summary["exact_match"]
	if !ok {
		t.Fatal("expected exact_match scorer in summary")
	}

	if stats.Count != 2 {
		t.Errorf("Count = %d, want 2", stats.Count)
	}

	if stats.PassRate != 1.0 {
		t.Errorf("PassRate = %f, want 1.0", stats.PassRate)
	}
}

func TestEvaluator_RunWithPredict(t *testing.T) {
	dataset := &Dataset{
		Name: "test-dataset",
		TestCases: []*TestCase{
			{
				Inputs:       map[string]any{"a": 2, "b": 3},
				Expectations: map[string]any{"output": 5},
			},
			{
				Inputs:       map[string]any{"a": 10, "b": 20},
				Expectations: map[string]any{"output": 30},
			},
		},
	}

	predictFunc := func(ctx context.Context, inputs map[string]any) (map[string]any, any, error) {
		a, _ := inputs["a"].(int)
		b, _ := inputs["b"].(int)
		return map[string]any{"output": a + b}, nil, nil
	}

	scorers := []Scorer{
		ExactMatchScorer("output"),
	}

	evaluator := NewEvaluator()
	results, err := evaluator.Run(context.Background(), dataset, scorers,
		WithPredict(predictFunc),
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if results.Summary["exact_match"].PassRate != 1.0 {
		t.Errorf("PassRate = %f, want 1.0", results.Summary["exact_match"].PassRate)
	}
}

func TestEvaluator_RunWithPredictError(t *testing.T) {
	dataset := &Dataset{
		Name: "test-dataset",
		TestCases: []*TestCase{
			{
				Inputs:       map[string]any{"prompt": "test"},
				Expectations: map[string]any{"output": "result"},
			},
		},
	}

	predictFunc := func(ctx context.Context, inputs map[string]any) (map[string]any, any, error) {
		return nil, nil, errors.New("prediction failed")
	}

	scorers := []Scorer{
		ExactMatchScorer("output"),
	}

	evaluator := NewEvaluator()
	results, err := evaluator.Run(context.Background(), dataset, scorers,
		WithPredict(predictFunc),
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Predict error should result in error scores for all scorers
	if results.Summary["exact_match"].ErrorRate != 1.0 {
		t.Errorf("ErrorRate = %f, want 1.0", results.Summary["exact_match"].ErrorRate)
	}
}

func TestEvaluator_RunParallel(t *testing.T) {
	dataset := &Dataset{
		Name: "test-dataset",
		TestCases: []*TestCase{
			{
				Inputs:       map[string]any{"prompt": "1"},
				Expectations: map[string]any{"output": "1"},
				Outputs:      map[string]any{"output": "1"},
			},
			{
				Inputs:       map[string]any{"prompt": "2"},
				Expectations: map[string]any{"output": "2"},
				Outputs:      map[string]any{"output": "2"},
			},
			{
				Inputs:       map[string]any{"prompt": "3"},
				Expectations: map[string]any{"output": "3"},
				Outputs:      map[string]any{"output": "3"},
			},
			{
				Inputs:       map[string]any{"prompt": "4"},
				Expectations: map[string]any{"output": "4"},
				Outputs:      map[string]any{"output": "4"},
			},
		},
	}

	scorers := []Scorer{
		ExactMatchScorer("output"),
	}

	evaluator := NewEvaluator()
	results, err := evaluator.Run(context.Background(), dataset, scorers,
		WithParallelism(2),
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if results.TotalTests != 4 {
		t.Errorf("TotalTests = %d, want 4", results.TotalTests)
	}

	if results.Summary["exact_match"].PassRate != 1.0 {
		t.Errorf("PassRate = %f, want 1.0", results.Summary["exact_match"].PassRate)
	}
}

func TestEvaluator_RunWithTimeout(t *testing.T) {
	dataset := &Dataset{
		Name: "test-dataset",
		TestCases: []*TestCase{
			{
				Inputs:       map[string]any{"prompt": "test"},
				Expectations: map[string]any{"output": "result"},
			},
		},
	}

	predictFunc := func(ctx context.Context, inputs map[string]any) (map[string]any, any, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return map[string]any{"output": "result"}, nil, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

	scorers := []Scorer{
		ExactMatchScorer("output"),
	}

	evaluator := NewEvaluator()
	results, err := evaluator.Run(context.Background(), dataset, scorers,
		WithPredict(predictFunc),
		WithTimeout(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should have error due to timeout
	if results.Summary["exact_match"].ErrorRate != 1.0 {
		t.Errorf("ErrorRate = %f, want 1.0 (should timeout)", results.Summary["exact_match"].ErrorRate)
	}
}

func TestEvaluator_ScorerPanic(t *testing.T) {
	dataset := &Dataset{
		Name: "test-dataset",
		TestCases: []*TestCase{
			{
				Inputs:       map[string]any{"prompt": "test"},
				Expectations: map[string]any{"output": "result"},
				Outputs:      map[string]any{"output": "result"},
			},
		},
	}

	panicScorer := &panicTestScorer{}
	scorers := []Scorer{panicScorer}

	evaluator := NewEvaluator()
	results, err := evaluator.Run(context.Background(), dataset, scorers)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should recover from panic and record error
	if results.Summary["panic_scorer"].ErrorRate != 1.0 {
		t.Errorf("ErrorRate = %f, want 1.0 (should have panic error)", results.Summary["panic_scorer"].ErrorRate)
	}
}

type panicTestScorer struct{}

func (s *panicTestScorer) Name() string { return "panic_scorer" }
func (s *panicTestScorer) Score(ctx context.Context, input ScorerInput) (Score, error) {
	panic("intentional panic for testing")
}

func TestEvaluator_MultipleScorers(t *testing.T) {
	dataset := &Dataset{
		Name: "test-dataset",
		TestCases: []*TestCase{
			{
				Inputs:       map[string]any{"prompt": "test"},
				Expectations: map[string]any{"output": "hello world", "pattern": `\w+`},
				Outputs:      map[string]any{"output": "hello world"},
			},
		},
	}

	scorers := []Scorer{
		ExactMatchScorer("output"),
		ContainsScorer("output"),
		RegexScorer("output"),
	}

	evaluator := NewEvaluator()
	results, err := evaluator.Run(context.Background(), dataset, scorers)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(results.Summary) != 3 {
		t.Errorf("expected 3 scorers in summary, got %d", len(results.Summary))
	}
}

func TestEvaluator_SummaryStats(t *testing.T) {
	dataset := &Dataset{
		Name: "test-dataset",
		TestCases: []*TestCase{
			{
				Inputs:       map[string]any{},
				Expectations: map[string]any{"output": "match"},
				Outputs:      map[string]any{"output": "match"},
			},
			{
				Inputs:       map[string]any{},
				Expectations: map[string]any{"output": "match"},
				Outputs:      map[string]any{"output": "no match"},
			},
			{
				Inputs:       map[string]any{},
				Expectations: map[string]any{"output": "match"},
				Outputs:      map[string]any{"output": "match"},
			},
		},
	}

	scorers := []Scorer{
		ExactMatchScorer("output"),
	}

	evaluator := NewEvaluator()
	results, err := evaluator.Run(context.Background(), dataset, scorers)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	stats := results.Summary["exact_match"]

	// 2 out of 3 should pass
	expectedPassRate := 2.0 / 3.0
	if stats.PassRate < expectedPassRate-0.01 || stats.PassRate > expectedPassRate+0.01 {
		t.Errorf("PassRate = %f, want approximately %f", stats.PassRate, expectedPassRate)
	}

	// Mean should be 2/3 (bool true = 1.0, false = 0.0)
	if stats.Mean < expectedPassRate-0.01 || stats.Mean > expectedPassRate+0.01 {
		t.Errorf("Mean = %f, want approximately %f", stats.Mean, expectedPassRate)
	}
}
