package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// TestNewClient tests client creation with various options.
func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		opts        []Option
		wantBaseURL string
		wantTimeout time.Duration
		wantRetries int
		checkToken  bool
		wantToken   string
	}{
		{
			name:        "default client",
			baseURL:     "http://localhost:5000",
			opts:        nil,
			wantBaseURL: "http://localhost:5000",
			wantTimeout: defaultTimeout,
			wantRetries: defaultMaxRetries,
			checkToken:  false,
		},
		{
			name:        "client with trailing slash removed",
			baseURL:     "http://localhost:5000/",
			opts:        nil,
			wantBaseURL: "http://localhost:5000",
			wantTimeout: defaultTimeout,
			wantRetries: defaultMaxRetries,
			checkToken:  false,
		},
		{
			name:        "client with token",
			baseURL:     "http://localhost:5000",
			opts:        []Option{WithToken("test-token-123")},
			wantBaseURL: "http://localhost:5000",
			wantTimeout: defaultTimeout,
			wantRetries: defaultMaxRetries,
			checkToken:  true,
			wantToken:   "test-token-123",
		},
		{
			name:        "client with custom timeout",
			baseURL:     "http://localhost:5000",
			opts:        []Option{WithTimeout(5 * time.Second)},
			wantBaseURL: "http://localhost:5000",
			wantTimeout: 5 * time.Second,
			wantRetries: defaultMaxRetries,
			checkToken:  false,
		},
		{
			name:        "client with custom retries",
			baseURL:     "http://localhost:5000",
			opts:        []Option{WithRetries(5)},
			wantBaseURL: "http://localhost:5000",
			wantTimeout: defaultTimeout,
			wantRetries: 5,
			checkToken:  false,
		},
		{
			name:    "client with retries disabled",
			baseURL: "http://localhost:5000",
			opts:    []Option{WithRetries(0)},
			wantBaseURL: "http://localhost:5000",
			wantTimeout: defaultTimeout,
			wantRetries: 0,
			checkToken:  false,
		},
		{
			name:    "client with multiple options",
			baseURL: "http://localhost:5000",
			opts: []Option{
				WithToken("multi-option-token"),
				WithTimeout(10 * time.Second),
				WithRetries(2),
			},
			wantBaseURL: "http://localhost:5000",
			wantTimeout: 10 * time.Second,
			wantRetries: 2,
			checkToken:  true,
			wantToken:   "multi-option-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.baseURL, tt.opts...)

			if client.baseURL != tt.wantBaseURL {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.wantBaseURL)
			}
			if client.timeout != tt.wantTimeout {
				t.Errorf("timeout = %v, want %v", client.timeout, tt.wantTimeout)
			}
			if client.maxRetries != tt.wantRetries {
				t.Errorf("maxRetries = %d, want %d", client.maxRetries, tt.wantRetries)
			}
			if tt.checkToken && client.token != tt.wantToken {
				t.Errorf("token = %q, want %q", client.token, tt.wantToken)
			}
			if client.httpClient == nil {
				t.Error("httpClient is nil")
			}
			if client.httpClient.Timeout != tt.wantTimeout {
				t.Errorf("httpClient.Timeout = %v, want %v", client.httpClient.Timeout, tt.wantTimeout)
			}
		})
	}
}

// TestWithHTTPClient tests the WithHTTPClient option.
func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	client := New("http://localhost:5000", WithHTTPClient(customClient))

	if client.httpClient != customClient {
		t.Error("httpClient was not set to custom client")
	}
	if client.httpClient.Timeout != 15*time.Second {
		t.Errorf("httpClient.Timeout = %v, want 15s", client.httpClient.Timeout)
	}
}

// TestAPIErrorHandling tests HTTP error response handling.
func TestAPIErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		wantStatusCode int
		wantMessage    string
		wantErrorCode  string
		checkMethod    func(*APIError) bool
		checkName      string
	}{
		{
			name:           "404 not found",
			statusCode:     404,
			responseBody:   `{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"Experiment not found"}`,
			wantStatusCode: 404,
			wantMessage:    "Experiment not found",
			wantErrorCode:  "RESOURCE_DOES_NOT_EXIST",
			checkMethod:    (*APIError).IsNotFound,
			checkName:      "IsNotFound",
		},
		{
			name:           "409 conflict",
			statusCode:     409,
			responseBody:   `{"error_code":"RESOURCE_ALREADY_EXISTS","message":"Experiment already exists"}`,
			wantStatusCode: 409,
			wantMessage:    "Experiment already exists",
			wantErrorCode:  "RESOURCE_ALREADY_EXISTS",
			checkMethod:    (*APIError).IsConflict,
			checkName:      "IsConflict",
		},
		{
			name:           "401 unauthorized",
			statusCode:     401,
			responseBody:   `{"error_code":"UNAUTHORIZED","message":"Invalid credentials"}`,
			wantStatusCode: 401,
			wantMessage:    "Invalid credentials",
			wantErrorCode:  "UNAUTHORIZED",
			checkMethod:    (*APIError).IsUnauthorized,
			checkName:      "IsUnauthorized",
		},
		{
			name:           "403 forbidden",
			statusCode:     403,
			responseBody:   `{"error_code":"PERMISSION_DENIED","message":"Access denied"}`,
			wantStatusCode: 403,
			wantMessage:    "Access denied",
			wantErrorCode:  "PERMISSION_DENIED",
			checkMethod:    (*APIError).IsForbidden,
			checkName:      "IsForbidden",
		},
		{
			name:           "429 rate limited",
			statusCode:     429,
			responseBody:   `{"error_code":"RATE_LIMITED","message":"Too many requests"}`,
			wantStatusCode: 429,
			wantMessage:    "Too many requests",
			wantErrorCode:  "RATE_LIMITED",
			checkMethod:    (*APIError).IsRateLimited,
			checkName:      "IsRateLimited",
		},
		{
			name:           "500 server error",
			statusCode:     500,
			responseBody:   `{"error_code":"INTERNAL_ERROR","message":"Server error"}`,
			wantStatusCode: 500,
			wantMessage:    "Server error",
			wantErrorCode:  "INTERNAL_ERROR",
			checkMethod:    (*APIError).IsServerError,
			checkName:      "IsServerError",
		},
		{
			name:           "503 service unavailable",
			statusCode:     503,
			responseBody:   `{"message":"Service temporarily unavailable"}`,
			wantStatusCode: 503,
			wantMessage:    "Service temporarily unavailable",
			wantErrorCode:  "",
			checkMethod:    (*APIError).IsServerError,
			checkName:      "IsServerError",
		},
		{
			name:           "plain text error response",
			statusCode:     500,
			responseBody:   "Internal Server Error",
			wantStatusCode: 500,
			wantMessage:    "Internal Server Error",
			wantErrorCode:  "",
			checkMethod:    (*APIError).IsServerError,
			checkName:      "IsServerError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := New(server.URL, WithRetries(0)) // Disable retries for cleaner tests
			ctx := context.Background()

			_, err := client.CreateExperiment(ctx, "test-experiment")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected *APIError, got %T: %v", err, err)
			}

			if apiErr.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.wantStatusCode)
			}
			if apiErr.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", apiErr.Message, tt.wantMessage)
			}
			if apiErr.ErrorCode != tt.wantErrorCode {
				t.Errorf("ErrorCode = %q, want %q", apiErr.ErrorCode, tt.wantErrorCode)
			}
			if tt.checkMethod != nil && !tt.checkMethod(apiErr) {
				t.Errorf("%s() = false, want true", tt.checkName)
			}
		})
	}
}

// TestRetryLogic tests retry behavior with exponential backoff.
func TestRetryLogic(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		failCount      int
		maxRetries     int
		expectRetries  int
		expectSuccess  bool
	}{
		{
			name:          "retry on 503 and succeed",
			statusCode:    503,
			failCount:     2,
			maxRetries:    3,
			expectRetries: 2,
			expectSuccess: true,
		},
		{
			name:          "retry on 500 and succeed",
			statusCode:    500,
			failCount:     1,
			maxRetries:    3,
			expectRetries: 1,
			expectSuccess: true,
		},
		{
			name:          "retry on 429 rate limit",
			statusCode:    429,
			failCount:     2,
			maxRetries:    3,
			expectRetries: 2,
			expectSuccess: true,
		},
		{
			name:          "exhaust all retries",
			statusCode:    500,
			failCount:     10,
			maxRetries:    3,
			expectRetries: 3,
			expectSuccess: false,
		},
		{
			name:          "no retry on 404",
			statusCode:    404,
			failCount:     10,
			maxRetries:    3,
			expectRetries: 0,
			expectSuccess: false,
		},
		{
			name:          "no retry when retries disabled",
			statusCode:    500,
			failCount:     10,
			maxRetries:    0,
			expectRetries: 0,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				if requestCount <= tt.failCount {
					w.WriteHeader(tt.statusCode)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error_code": "TEMPORARY_ERROR",
						"message":    "Temporary failure",
					})
					return
				}
				// Success after failures
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(createExperimentResponse{
					ExperimentID: "test-exp-123",
				})
			}))
			defer server.Close()

			client := New(server.URL, WithRetries(tt.maxRetries), WithTimeout(5*time.Second))
			ctx := context.Background()

			expID, err := client.CreateExperiment(ctx, "test-experiment")

			actualRetries := requestCount - 1

			if actualRetries != tt.expectRetries {
				t.Errorf("retries = %d, want %d", actualRetries, tt.expectRetries)
			}

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("expected success, got error: %v", err)
				}
				if expID != "test-exp-123" {
					t.Errorf("experimentID = %q, want %q", expID, "test-exp-123")
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				}
			}
		})
	}
}

// TestAuthorizationHeader tests that the Bearer token is set correctly.
func TestAuthorizationHeader(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantAuth  string
		checkAuth bool
	}{
		{
			name:      "with token",
			token:     "my-secret-token",
			wantAuth:  "Bearer my-secret-token",
			checkAuth: true,
		},
		{
			name:      "without token",
			token:     "",
			wantAuth:  "",
			checkAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedAuth string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedAuth = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(createExperimentResponse{
					ExperimentID: "test-exp",
				})
			}))
			defer server.Close()

			opts := []Option{WithRetries(0)}
			if tt.token != "" {
				opts = append(opts, WithToken(tt.token))
			}

			client := New(server.URL, opts...)
			ctx := context.Background()

			_, err := client.CreateExperiment(ctx, "test")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkAuth {
				if receivedAuth != tt.wantAuth {
					t.Errorf("Authorization header = %q, want %q", receivedAuth, tt.wantAuth)
				}
			} else {
				if receivedAuth != "" {
					t.Errorf("Authorization header = %q, want empty", receivedAuth)
				}
			}
		})
	}
}

// TestCreateExperiment tests the CreateExperiment operation.
func TestCreateExperiment(t *testing.T) {
	tests := []struct {
		name          string
		experimentName string
		responseBody   string
		statusCode     int
		wantExpID      string
		wantErr        bool
		checkErrType   func(error) bool
	}{
		{
			name:           "successful creation",
			experimentName: "my-experiment",
			responseBody:   `{"experiment_id":"123"}`,
			statusCode:     http.StatusOK,
			wantExpID:      "123",
			wantErr:        false,
		},
		{
			name:           "empty experiment name",
			experimentName: "",
			wantErr:        true,
			checkErrType: func(err error) bool {
				_, ok := err.(*ValidationError)
				return ok
			},
		},
		{
			name:           "server returns empty experiment_id",
			experimentName: "test",
			responseBody:   `{"experiment_id":""}`,
			statusCode:     http.StatusOK,
			wantErr:        true,
			checkErrType: func(err error) bool {
				apiErr, ok := err.(*APIError)
				return ok && apiErr.StatusCode == 500
			},
		},
		{
			name:           "experiment already exists",
			experimentName: "existing",
			responseBody:   `{"error_code":"RESOURCE_ALREADY_EXISTS","message":"Experiment exists"}`,
			statusCode:     http.StatusConflict,
			wantErr:        true,
			checkErrType: func(err error) bool {
				apiErr, ok := err.(*APIError)
				return ok && apiErr.IsConflict()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				}
				if tt.responseBody != "" {
					_, _ = w.Write([]byte(tt.responseBody))
				}
			}))
			defer server.Close()

			client := New(server.URL, WithRetries(0))
			ctx := context.Background()

			expID, err := client.CreateExperiment(ctx, tt.experimentName)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.checkErrType != nil && !tt.checkErrType(err) {
					t.Errorf("unexpected error type: %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if expID != tt.wantExpID {
					t.Errorf("experimentID = %q, want %q", expID, tt.wantExpID)
				}
			}
		})
	}
}

// TestGetExperiment tests the GetExperiment operation.
func TestGetExperiment(t *testing.T) {
	tests := []struct {
		name         string
		experimentID string
		responseBody string
		statusCode   int
		wantErr      bool
		checkErrType func(error) bool
		checkResult  func(*testing.T, *pb.Experiment)
	}{
		{
			name:         "successful get",
			experimentID: "123",
			responseBody: `{"experiment":{"experiment_id":"123","name":"my-exp","lifecycle_stage":"active"}}`,
			statusCode:   http.StatusOK,
			wantErr:      false,
			checkResult: func(t *testing.T, exp *pb.Experiment) {
				if exp.ExperimentId == nil || *exp.ExperimentId != "123" {
					t.Errorf("ExperimentId = %v, want %q", exp.ExperimentId, "123")
				}
				if exp.Name == nil || *exp.Name != "my-exp" {
					t.Errorf("Name = %v, want %q", exp.Name, "my-exp")
				}
			},
		},
		{
			name:         "empty experiment ID",
			experimentID: "",
			wantErr:      true,
			checkErrType: func(err error) bool {
				_, ok := err.(*ValidationError)
				return ok
			},
		},
		{
			name:         "experiment not found",
			experimentID: "999",
			responseBody: `{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"Not found"}`,
			statusCode:   http.StatusNotFound,
			wantErr:      true,
			checkErrType: func(err error) bool {
				apiErr, ok := err.(*APIError)
				return ok && apiErr.IsNotFound()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				}
				if tt.responseBody != "" {
					_, _ = w.Write([]byte(tt.responseBody))
				}
			}))
			defer server.Close()

			client := New(server.URL, WithRetries(0))
			ctx := context.Background()

			exp, err := client.GetExperiment(ctx, tt.experimentID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.checkErrType != nil && !tt.checkErrType(err) {
					t.Errorf("unexpected error type: %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.checkResult != nil {
					tt.checkResult(t, exp)
				}
			}
		})
	}
}

// TestSearchExperiments tests the SearchExperiments operation.
func TestSearchExperiments(t *testing.T) {
	tests := []struct {
		name         string
		opts         SearchExperimentsOptions
		responseBody string
		statusCode   int
		wantErr      bool
		checkResult  func(*testing.T, *SearchExperimentsResult)
	}{
		{
			name: "successful search with results",
			opts: SearchExperimentsOptions{
				MaxResults: 10,
			},
			responseBody: `{
				"experiments":[
					{"experiment_id":"1","name":"exp1"},
					{"experiment_id":"2","name":"exp2"}
				],
				"next_page_token":"token123"
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			checkResult: func(t *testing.T, result *SearchExperimentsResult) {
				if len(result.Experiments) != 2 {
					t.Errorf("got %d experiments, want 2", len(result.Experiments))
				}
				if result.NextPageToken != "token123" {
					t.Errorf("NextPageToken = %q, want %q", result.NextPageToken, "token123")
				}
			},
		},
		{
			name: "empty results",
			opts: SearchExperimentsOptions{},
			responseBody: `{
				"experiments":[]
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			checkResult: func(t *testing.T, result *SearchExperimentsResult) {
				if len(result.Experiments) != 0 {
					t.Errorf("got %d experiments, want 0", len(result.Experiments))
				}
				if result.NextPageToken != "" {
					t.Errorf("NextPageToken = %q, want empty", result.NextPageToken)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				}
				if tt.responseBody != "" {
					_, _ = w.Write([]byte(tt.responseBody))
				}
			}))
			defer server.Close()

			client := New(server.URL, WithRetries(0))
			ctx := context.Background()

			result, err := client.SearchExperiments(ctx, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

// TestUpdateExperiment tests the UpdateExperiment operation.
func TestUpdateExperiment(t *testing.T) {
	tests := []struct {
		name         string
		experimentID string
		newName      string
		statusCode   int
		wantErr      bool
		checkErrType func(error) bool
	}{
		{
			name:         "successful update",
			experimentID: "123",
			newName:      "updated-name",
			statusCode:   http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "empty experiment ID",
			experimentID: "",
			newName:      "new-name",
			wantErr:      true,
			checkErrType: func(err error) bool {
				_, ok := err.(*ValidationError)
				return ok
			},
		},
		{
			name:         "empty new name",
			experimentID: "123",
			newName:      "",
			wantErr:      true,
			checkErrType: func(err error) bool {
				_, ok := err.(*ValidationError)
				return ok
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				}
			}))
			defer server.Close()

			client := New(server.URL, WithRetries(0))
			ctx := context.Background()

			err := client.UpdateExperiment(ctx, tt.experimentID, tt.newName)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.checkErrType != nil && !tt.checkErrType(err) {
					t.Errorf("unexpected error type: %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestDeleteExperiment tests the DeleteExperiment operation.
func TestDeleteExperiment(t *testing.T) {
	tests := []struct {
		name         string
		experimentID string
		statusCode   int
		wantErr      bool
		checkErrType func(error) bool
	}{
		{
			name:         "successful delete",
			experimentID: "123",
			statusCode:   http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "empty experiment ID",
			experimentID: "",
			wantErr:      true,
			checkErrType: func(err error) bool {
				_, ok := err.(*ValidationError)
				return ok
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				}
			}))
			defer server.Close()

			client := New(server.URL, WithRetries(0))
			ctx := context.Background()

			err := client.DeleteExperiment(ctx, tt.experimentID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.checkErrType != nil && !tt.checkErrType(err) {
					t.Errorf("unexpected error type: %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestContentTypeHeader tests that Content-Type is set correctly.
func TestContentTypeHeader(t *testing.T) {
	var receivedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(createExperimentResponse{
			ExperimentID: "test",
		})
	}))
	defer server.Close()

	client := New(server.URL, WithRetries(0))
	ctx := context.Background()

	_, err := client.CreateExperiment(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", receivedContentType, "application/json")
	}
}

// TestRequestMethod tests that HTTP methods are correct.
func TestRequestMethod(t *testing.T) {
	tests := []struct {
		name       string
		operation  func(context.Context, *Client) error
		wantMethod string
	}{
		{
			name: "CreateExperiment uses POST",
			operation: func(ctx context.Context, c *Client) error {
				_, err := c.CreateExperiment(ctx, "test")
				return err
			},
			wantMethod: "POST",
		},
		{
			name: "GetExperiment uses GET",
			operation: func(ctx context.Context, c *Client) error {
				_, err := c.GetExperiment(ctx, "123")
				return err
			},
			wantMethod: "GET",
		},
		{
			name: "SearchExperiments uses POST",
			operation: func(ctx context.Context, c *Client) error {
				_, err := c.SearchExperiments(ctx, SearchExperimentsOptions{})
				return err
			},
			wantMethod: "POST",
		},
		{
			name: "UpdateExperiment uses POST",
			operation: func(ctx context.Context, c *Client) error {
				return c.UpdateExperiment(ctx, "123", "new-name")
			},
			wantMethod: "POST",
		},
		{
			name: "DeleteExperiment uses POST",
			operation: func(ctx context.Context, c *Client) error {
				return c.DeleteExperiment(ctx, "123")
			},
			wantMethod: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedMethod string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
				// Send minimal valid responses
				if strings.Contains(r.URL.Path, "/create") {
					_ = json.NewEncoder(w).Encode(createExperimentResponse{ExperimentID: "123"})
				} else if strings.Contains(r.URL.Path, "/get") {
					expID := "123"
					expName := "test"
					_ = json.NewEncoder(w).Encode(getExperimentResponse{
						Experiment: &pb.Experiment{ExperimentId: &expID, Name: &expName},
					})
				} else if strings.Contains(r.URL.Path, "/search") {
					_ = json.NewEncoder(w).Encode(searchExperimentsResponse{Experiments: []*pb.Experiment{}})
				}
			}))
			defer server.Close()

			client := New(server.URL, WithRetries(0))
			ctx := context.Background()

			_ = tt.operation(ctx, client)

			if receivedMethod != tt.wantMethod {
				t.Errorf("HTTP method = %q, want %q", receivedMethod, tt.wantMethod)
			}
		})
	}
}

// TestBackoffCalculation tests exponential backoff timing.
func TestBackoffCalculation(t *testing.T) {
	cfg := defaultRetryConfig()

	tests := []struct {
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			attempt:     0,
			minExpected: 900 * time.Millisecond,  // 1s - 10% jitter
			maxExpected: 1100 * time.Millisecond, // 1s + 10% jitter
		},
		{
			attempt:     1,
			minExpected: 1800 * time.Millisecond, // 2s - 10% jitter
			maxExpected: 2200 * time.Millisecond, // 2s + 10% jitter
		},
		{
			attempt:     2,
			minExpected: 3600 * time.Millisecond, // 4s - 10% jitter
			maxExpected: 4400 * time.Millisecond, // 4s + 10% jitter
		},
		{
			attempt:     3,
			minExpected: 7200 * time.Millisecond, // 8s - 10% jitter
			maxExpected: 8800 * time.Millisecond, // 8s + 10% jitter
		},
		{
			attempt:     4,
			minExpected: 9000 * time.Millisecond,  // Capped at 10s - 10% jitter
			maxExpected: 11000 * time.Millisecond, // Capped at 10s + 10% jitter
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			// Test multiple times to account for jitter randomness
			for i := 0; i < 10; i++ {
				backoff := calculateBackoff(cfg, tt.attempt)
				if backoff < tt.minExpected || backoff > tt.maxExpected {
					t.Errorf("backoff = %v, want between %v and %v", backoff, tt.minExpected, tt.maxExpected)
				}
			}
		})
	}
}

// TestContextCancellation tests that context cancellation is respected.
func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(createExperimentResponse{ExperimentID: "123"})
	}))
	defer server.Close()

	client := New(server.URL, WithRetries(0))

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.CreateExperiment(ctx, "test")
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}

	// The error should be context-related
	if !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "connection error") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRequestURL tests that URLs are constructed correctly.
func TestRequestURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		operation   func(context.Context, *Client) error
		wantPath    string
	}{
		{
			name:    "create experiment URL",
			baseURL: "http://localhost:5000",
			operation: func(ctx context.Context, c *Client) error {
				_, err := c.CreateExperiment(ctx, "test")
				return err
			},
			wantPath: "/api/2.0/mlflow/experiments/create",
		},
		{
			name:    "get experiment URL",
			baseURL: "http://mlflow.example.com",
			operation: func(ctx context.Context, c *Client) error {
				_, err := c.GetExperiment(ctx, "123")
				return err
			},
			wantPath: "/api/2.0/mlflow/experiments/get",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedURL = r.URL.Path
				w.WriteHeader(http.StatusOK)
				if strings.Contains(r.URL.Path, "/create") {
					_ = json.NewEncoder(w).Encode(createExperimentResponse{ExperimentID: "123"})
				} else {
					expID := "123"
					_ = json.NewEncoder(w).Encode(getExperimentResponse{
						Experiment: &pb.Experiment{ExperimentId: &expID},
					})
				}
			}))
			defer server.Close()

			client := New(server.URL, WithRetries(0))
			ctx := context.Background()

			_ = tt.operation(ctx, client)

			if receivedURL != tt.wantPath {
				t.Errorf("request path = %q, want %q", receivedURL, tt.wantPath)
			}
		})
	}
}
