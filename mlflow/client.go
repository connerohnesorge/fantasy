package mlflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
)

// Client is the MLflow REST API client.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	token       string
	timeout     time.Duration
	maxRetries  int
	retryConfig retryConfig
}

// New creates a new MLflow client with the given base URL and options.
// The baseURL should be the root URL of the MLflow server (e.g., "http://localhost:5000").
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		httpClient:  &http.Client{},
		timeout:     defaultTimeout,
		maxRetries:  defaultMaxRetries,
		retryConfig: defaultRetryConfig(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Set timeout on the HTTP client if not already set
	if c.httpClient.Timeout == 0 {
		c.httpClient.Timeout = c.timeout
	}

	return c
}

// doRequest performs an HTTP request with retry logic and protobuf serialization.
// This is the core method used by all API operations.
func (c *Client) doRequest(ctx context.Context, method, path string, req, resp proto.Message) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// If this is a retry, wait with exponential backoff
		if attempt > 0 {
			backoff := calculateBackoff(c.retryConfig, attempt-1)
			if err := sleep(ctx, backoff); err != nil {
				return &TimeoutError{
					Operation: fmt.Sprintf("%s %s", method, path),
					Timeout:   int64(c.timeout / time.Millisecond),
					Cause:     err,
				}
			}
		}

		// Perform the actual request
		err := c.doSingleRequest(ctx, method, path, req, resp)
		if err == nil {
			return nil // Success!
		}

		lastErr = err

		// Check if we should retry
		if !shouldRetry(err, attempt, c.maxRetries) {
			break
		}
	}

	return lastErr
}

// doSingleRequest performs a single HTTP request without retry logic.
func (c *Client) doSingleRequest(ctx context.Context, method, path string, req, resp proto.Message) error {
	// Build the full URL
	url := c.baseURL + path

	// Serialize request body if provided
	var bodyReader io.Reader
	if req != nil {
		marshaler := protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
		}
		data, err := marshaler.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return &ConnectionError{
			URL:     url,
			Message: "failed to create request",
			Cause:   err,
		}
	}

	// Set headers
	if req != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Check if it's a network error
		if isNetworkError(err) {
			return &ConnectionError{
				URL:     url,
				Message: err.Error(),
				Cause:   err,
			}
		}

		// Check if it's a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return &TimeoutError{
				Operation: fmt.Sprintf("%s %s", method, path),
				Timeout:   int64(c.timeout / time.Millisecond),
				Cause:     err,
			}
		}

		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if httpResp.StatusCode >= 400 {
		return c.parseAPIError(httpResp.StatusCode, body)
	}

	// Deserialize response if provided
	if resp != nil {
		unmarshaler := protojson.UnmarshalOptions{
			DiscardUnknown: true,
		}
		if err := unmarshaler.Unmarshal(body, resp); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// parseAPIError parses an error response from the MLflow API.
func (c *Client) parseAPIError(statusCode int, body []byte) error {
	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    string(body),
	}

	// Try to parse structured error response
	var errorResp struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(body, &errorResp); err == nil {
		if errorResp.Message != "" {
			apiErr.Message = errorResp.Message
		}
		if errorResp.ErrorCode != "" {
			apiErr.ErrorCode = errorResp.ErrorCode
		}
	}

	return apiErr
}
