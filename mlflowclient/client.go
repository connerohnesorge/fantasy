package mlflowclient

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Client is a REST client for the MLflow API.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	token       string
	timeout     time.Duration
	maxRetries  int
	retryConfig RetryConfig
}

// New creates a new MLflow client with the given base URL and options.
// The base URL should be the root of the MLflow server (e.g., "http://localhost:5000").
func New(baseURL string, opts ...Option) *Client {
	// Ensure baseURL doesn't end with a slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	c := &Client{
		baseURL:     baseURL,
		httpClient:  &http.Client{},
		timeout:     30 * time.Second,
		maxRetries:  3,
		retryConfig: DefaultRetryConfig(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// doRequest performs an HTTP request with retry logic and protojson marshaling.
func (c *Client) doRequest(ctx context.Context, method, path string, reqBody, respBody proto.Message) error {
	var reqData []byte
	var err error

	if reqBody != nil {
		reqData, err = protojson.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	// Construct full URL
	fullURL := c.baseURL + path

	// Apply timeout to context
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	// Retry loop
	var lastErr error
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay with exponential backoff and jitter
			delay := c.calculateBackoff(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return &TimeoutError{
					Operation: method + " " + path,
					Timeout:   c.timeout,
					Cause:     ctx.Err(),
				}
			}
		}

		// Create request
		var body io.Reader
		if reqData != nil {
			body = bytes.NewReader(reqData)
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		// Perform request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Check if context was cancelled
			if ctx.Err() != nil {
				return &TimeoutError{
					Operation: method + " " + path,
					Timeout:   c.timeout,
					Cause:     ctx.Err(),
				}
			}

			// Connection error - retryable
			lastErr = &ConnectionError{
				URL:     fullURL,
				Message: err.Error(),
				Cause:   err,
			}

			// Retry on connection errors
			if attempt < c.retryConfig.MaxRetries {
				continue
			}
			return lastErr
		}

		defer resp.Body.Close()

		// Read response body
		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check for HTTP errors
		if resp.StatusCode >= 400 {
			apiErr := c.parseAPIError(resp.StatusCode, respData)

			// Retry on retryable errors (5xx, 429)
			if apiErr.IsRetryable() && attempt < c.retryConfig.MaxRetries {
				lastErr = apiErr
				continue
			}

			return apiErr
		}

		// Successful response - unmarshal if response body is expected
		if respBody != nil && len(respData) > 0 {
			if err := protojson.Unmarshal(respData, respBody); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return nil
	}

	return lastErr
}

// calculateBackoff calculates the backoff delay for a given retry attempt.
func (c *Client) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: initialDelay * (multiplier ^ (attempt - 1))
	delay := float64(c.retryConfig.InitialDelay) * math.Pow(c.retryConfig.BackoffMultiplier, float64(attempt-1))

	// Cap at max delay
	if delay > float64(c.retryConfig.MaxDelay) {
		delay = float64(c.retryConfig.MaxDelay)
	}

	// Apply jitter (±jitterPercent)
	if c.retryConfig.JitterPercent > 0 {
		jitter := delay * c.retryConfig.JitterPercent / 100.0
		// Random value between -jitter and +jitter
		jitterBytes := make([]byte, 8)
		rand.Read(jitterBytes)
		jitterVal := float64(int64(jitterBytes[0])%int64(jitter*2)) - jitter
		delay += jitterVal
	}

	return time.Duration(delay)
}

// parseAPIError parses an API error response.
func (c *Client) parseAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    string(body),
	}

	// Try to parse error details from JSON response
	var errResp struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"message"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.ErrorCode != "" {
			apiErr.ErrorCode = errResp.ErrorCode
		}
		if errResp.Message != "" {
			apiErr.Message = errResp.Message
		}
	}

	return apiErr
}

// doRequestJSON performs an HTTP request with standard JSON marshaling (for non-proto requests).
func (c *Client) doRequestJSON(ctx context.Context, method, path string, reqBody interface{}, respBody interface{}) error {
	var reqData []byte
	var err error

	if reqBody != nil {
		reqData, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	// Construct full URL
	fullURL := c.baseURL + path

	// Apply timeout to context
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	// Retry loop
	var lastErr error
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
			delay := c.calculateBackoff(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return &TimeoutError{
					Operation: method + " " + path,
					Timeout:   c.timeout,
					Cause:     ctx.Err(),
				}
			}
		}

		// Create request
		var body io.Reader
		if reqData != nil {
			body = bytes.NewReader(reqData)
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		// Perform request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Check if context was cancelled
			if ctx.Err() != nil {
				return &TimeoutError{
					Operation: method + " " + path,
					Timeout:   c.timeout,
					Cause:     ctx.Err(),
				}
			}

			// Connection error - retryable
			lastErr = &ConnectionError{
				URL:     fullURL,
				Message: err.Error(),
				Cause:   err,
			}

			// Retry on connection errors
			if attempt < c.retryConfig.MaxRetries {
				continue
			}
			return lastErr
		}

		defer resp.Body.Close()

		// Read response body
		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check for HTTP errors
		if resp.StatusCode >= 400 {
			apiErr := c.parseAPIError(resp.StatusCode, respData)

			// Retry on retryable errors
			if apiErr.IsRetryable() && attempt < c.retryConfig.MaxRetries {
				lastErr = apiErr
				continue
			}

			return apiErr
		}

		// Successful response - unmarshal if response body is expected
		if respBody != nil && len(respData) > 0 {
			if err := json.Unmarshal(respData, respBody); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return nil
	}

	return lastErr
}

// generateID generates a random hex ID of the specified byte length.
func generateID(byteLength int) string {
	b := make([]byte, byteLength)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// buildQueryParams builds a query string from a map of parameters.
func buildQueryParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	values := url.Values{}
	for k, v := range params {
		if v != "" {
			values.Set(k, v)
		}
	}

	query := values.Encode()
	if query != "" {
		return "?" + query
	}
	return ""
}
