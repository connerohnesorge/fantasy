package mlflowclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pb "charm.land/fantasy/proto/gen/mlflow"
	"google.golang.org/protobuf/encoding/protojson"
)

// mockServer creates a test HTTP server with configurable responses
type mockServer struct {
	*httptest.Server
	requestCount int
	lastRequest  *http.Request
	lastBody     string
}

func newMockServer(handler http.HandlerFunc) *mockServer {
	ms := &mockServer{}
	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ms.requestCount++
		ms.lastRequest = r

		// Read body
		buf := make([]byte, 1024*1024)
		n, _ := r.Body.Read(buf)
		ms.lastBody = string(buf[:n])
		r.Body.Close()

		handler(w, r)
	}))
	return ms
}

func TestClientCreation(t *testing.T) {
	client := New("http://localhost:5000")
	if client.baseURL != "http://localhost:5000" {
		t.Errorf("expected baseURL to be http://localhost:5000, got %s", client.baseURL)
	}
	if client.timeout != 30*time.Second {
		t.Errorf("expected default timeout to be 30s, got %s", client.timeout)
	}
	if client.maxRetries != 3 {
		t.Errorf("expected default maxRetries to be 3, got %d", client.maxRetries)
	}
}

func TestClientWithOptions(t *testing.T) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	client := New("http://localhost:5000",
		WithHTTPClient(httpClient),
		WithToken("test-token"),
		WithTimeout(60*time.Second),
		WithRetries(5),
	)

	if client.httpClient != httpClient {
		t.Error("expected custom HTTP client to be set")
	}
	if client.token != "test-token" {
		t.Errorf("expected token to be test-token, got %s", client.token)
	}
	if client.timeout != 60*time.Second {
		t.Errorf("expected timeout to be 60s, got %s", client.timeout)
	}
	if client.maxRetries != 5 {
		t.Errorf("expected maxRetries to be 5, got %d", client.maxRetries)
	}
}

func TestCreateExperiment(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/2.0/mlflow/experiments/create") {
			t.Errorf("expected experiments/create path, got %s", r.URL.Path)
		}

		// Check authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token in Authorization header")
		}

		// Return success response
		resp := &pb.CreateExperiment_Response{
			ExperimentId: stringPtr("exp-123"),
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
	defer server.Close()

	client := New(server.URL, WithToken("test-token"))
	ctx := context.Background()

	expID, err := client.CreateExperiment(ctx, "test-experiment", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("CreateExperiment failed: %v", err)
	}

	if expID != "exp-123" {
		t.Errorf("expected experiment ID exp-123, got %s", expID)
	}

	if server.requestCount != 1 {
		t.Errorf("expected 1 request, got %d", server.requestCount)
	}
}

func TestGetExperiment(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/2.0/mlflow/experiments/get") {
			t.Errorf("expected experiments/get path, got %s", r.URL.Path)
		}

		// Return experiment
		exp := &pb.Experiment{
			ExperimentId: stringPtr("exp-123"),
			Name:         stringPtr("test-experiment"),
		}
		resp := &pb.GetExperiment_Response{
			Experiment: exp,
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	exp, err := client.GetExperiment(ctx, "exp-123")
	if err != nil {
		t.Fatalf("GetExperiment failed: %v", err)
	}

	if exp.GetExperimentId() != "exp-123" {
		t.Errorf("expected experiment ID exp-123, got %s", exp.GetExperimentId())
	}
	if exp.GetName() != "test-experiment" {
		t.Errorf("expected experiment name test-experiment, got %s", exp.GetName())
	}
}

func TestCreateRun(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/2.0/mlflow/runs/create") {
			t.Errorf("expected runs/create path, got %s", r.URL.Path)
		}

		// Return run
		run := &pb.Run{
			Info: &pb.RunInfo{
				RunId:        stringPtr("run-123"),
				ExperimentId: stringPtr("exp-123"),
			},
		}
		resp := &pb.CreateRun_Response{
			Run: run,
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	run, err := client.CreateRun(ctx, "exp-123", WithRunName("test-run"))
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	if run.Info.GetRunId() != "run-123" {
		t.Errorf("expected run ID run-123, got %s", run.Info.GetRunId())
	}
}

func TestAPIError(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Experiment not found",
		})
	})
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	_, err := client.GetExperiment(ctx, "non-existent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to return true")
	}
	if apiErr.ErrorCode != "RESOURCE_DOES_NOT_EXIST" {
		t.Errorf("expected error code RESOURCE_DOES_NOT_EXIST, got %s", apiErr.ErrorCode)
	}
}

func TestRetry(t *testing.T) {
	attempts := 0
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Return server error to trigger retry
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Server error",
			})
			return
		}
		// Success on third attempt
		resp := &pb.GetExperiment_Response{
			Experiment: &pb.Experiment{
				ExperimentId: stringPtr("exp-123"),
				Name:         stringPtr("test"),
			},
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
	defer server.Close()

	// Configure fast retries for testing
	client := New(server.URL, WithRetryConfig(RetryConfig{
		MaxRetries:        3,
		InitialDelay:      10 * time.Millisecond,
		BackoffMultiplier: 1.5,
		MaxDelay:          100 * time.Millisecond,
		JitterPercent:     0,
	}))
	ctx := context.Background()

	exp, err := client.GetExperiment(ctx, "exp-123")
	if err != nil {
		t.Fatalf("GetExperiment failed after retries: %v", err)
	}

	if exp.GetExperimentId() != "exp-123" {
		t.Errorf("expected experiment ID exp-123, got %s", exp.GetExperimentId())
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestValidationError(t *testing.T) {
	client := New("http://localhost:5000")
	ctx := context.Background()

	// Test empty experiment name
	_, err := client.CreateExperiment(ctx, "", nil)
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if valErr.Field != "name" {
		t.Errorf("expected field name, got %s", valErr.Field)
	}
}

func TestTimeout(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := New(server.URL, WithTimeout(50*time.Millisecond))
	ctx := context.Background()

	_, err := client.GetExperiment(ctx, "exp-123")
	if err == nil {
		t.Fatal("expected timeout error")
	}

	timeoutErr, ok := err.(*TimeoutError)
	if !ok {
		t.Fatalf("expected TimeoutError, got %T: %v", err, err)
	}

	if timeoutErr.Timeout != 50*time.Millisecond {
		t.Errorf("expected timeout 50ms, got %s", timeoutErr.Timeout)
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
