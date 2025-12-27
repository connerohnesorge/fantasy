package mlflowclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "valid URL",
			baseURL: "http://localhost:5000",
			wantErr: false,
		},
		{
			name:    "empty URL",
			baseURL: "",
			wantErr: true,
		},
		{
			name:    "URL with trailing slash",
			baseURL: "http://localhost:5000/",
			wantErr: false,
		},
		{
			name:    "with options",
			baseURL: "http://localhost:5000",
			opts:    []Option{WithToken("test-token"), WithTimeout(60 * time.Second)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.baseURL, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("New() returned nil client without error")
			}
		})
	}
}

func TestClient_BaseURL(t *testing.T) {
	client, _ := New("http://localhost:5000/")
	if client.BaseURL() != "http://localhost:5000" {
		t.Errorf("BaseURL() = %v, want %v", client.BaseURL(), "http://localhost:5000")
	}
}

func TestClient_CreateExperiment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/2.0/mlflow/experiments/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if req.Name != "test-experiment" {
			t.Errorf("expected name 'test-experiment', got %s", req.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"experiment_id": "123",
		})
	}))
	defer server.Close()

	client, _ := New(server.URL)
	expID, err := client.CreateExperiment(context.Background(), "test-experiment")
	if err != nil {
		t.Fatalf("CreateExperiment() error = %v", err)
	}
	if expID != "123" {
		t.Errorf("ExperimentID = %v, want %v", expID, "123")
	}
}

func TestClient_GetExperiment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"experiment": map[string]any{
				"experiment_id": "123",
				"name":          "test-experiment",
			},
		})
	}))
	defer server.Close()

	client, _ := New(server.URL)
	exp, err := client.GetExperiment(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetExperiment() error = %v", err)
	}
	if exp.ExperimentID != "123" {
		t.Errorf("ExperimentID = %v, want %v", exp.ExperimentID, "123")
	}
	if exp.Name != "test-experiment" {
		t.Errorf("Name = %v, want %v", exp.Name, "test-experiment")
	}
}

func TestClient_CreateRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"run": map[string]any{
				"info": map[string]any{
					"run_id":        "run-123",
					"experiment_id": "exp-123",
					"status":        "RUNNING",
				},
			},
		})
	}))
	defer server.Close()

	client, _ := New(server.URL)
	run, err := client.CreateRun(context.Background(), "exp-123", WithRunName("test-run"))
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	if run.Info.RunID != "run-123" {
		t.Errorf("RunID = %v, want %v", run.Info.RunID, "run-123")
	}
}

func TestClient_LogBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/2.0/mlflow/runs/log-batch" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req struct {
			RunID   string    `json:"run_id"`
			Metrics []*Metric `json:"metrics"`
			Params  []*Param  `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if req.RunID != "run-123" {
			t.Errorf("expected run_id 'run-123', got %s", req.RunID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	client, _ := New(server.URL)
	metrics := []*Metric{{Key: "accuracy", Value: 0.95, Timestamp: time.Now().UnixMilli()}}
	params := []*Param{{Key: "learning_rate", Value: "0.001"}}

	err := client.LogBatch(context.Background(), "run-123", metrics, params, nil)
	if err != nil {
		t.Fatalf("LogBatch() error = %v", err)
	}
}

func TestClient_StartTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/2.0/mlflow/traces" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"trace_info": map[string]any{
				"request_id": "trace-123",
			},
		})
	}))
	defer server.Close()

	client, _ := New(server.URL)
	trace := &Trace{
		ExperimentID:    "exp-123",
		RequestTimeMs:   time.Now().UnixMilli(),
		ExecutionTimeMs: 1000,
		State:           TraceStateOK,
		RequestPreview:  "test request",
		ResponsePreview: "test response",
	}

	traceID, err := client.StartTrace(context.Background(), trace)
	if err != nil {
		t.Fatalf("StartTrace() error = %v", err)
	}
	if traceID != "trace-123" {
		t.Errorf("traceID = %v, want %v", traceID, "trace-123")
	}
}

func TestClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error_code": "RESOURCE_NOT_FOUND",
			"message":    "Experiment not found",
		})
	}))
	defer server.Close()

	client, _ := New(server.URL)
	_, err := client.GetExperiment(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %v, want %v", apiErr.StatusCode, http.StatusNotFound)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to return true")
	}
}

func TestClient_RetryOnServerError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"experiment": map[string]any{
				"experiment_id": "123",
				"name":          "test",
			},
		})
	}))
	defer server.Close()

	client, _ := New(server.URL,
		WithRetries(3),
		WithRetryConfig(RetryConfig{
			MaxRetries:     3,
			InitialDelay:   1 * time.Millisecond,
			MaxDelay:       10 * time.Millisecond,
			BackoffFactor:  2.0,
			JitterFraction: 0.1,
		}),
	)

	_, err := client.GetExperiment(context.Background(), "123")
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, _ := New(server.URL, WithTimeout(10*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetExperiment(ctx, "123")
	if err == nil {
		t.Fatal("expected error due to cancelled context")
	}
}

func TestClient_WithToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected 'Bearer test-token', got '%s'", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"experiment": map[string]any{
				"experiment_id": "123",
			},
		})
	}))
	defer server.Close()

	client, _ := New(server.URL, WithToken("test-token"))
	_, err := client.GetExperiment(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetExperiment() error = %v", err)
	}
}
