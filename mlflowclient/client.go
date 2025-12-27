package mlflowclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is the MLflow REST API client.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	token       string
	timeout     time.Duration
	maxRetries  int
	retryConfig RetryConfig
}

// New creates a new MLflow client.
func New(baseURL string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, &ValidationError{Field: "baseURL", Message: "base URL is required"}
	}

	// Normalize base URL
	baseURL = strings.TrimSuffix(baseURL, "/")

	c := &Client{
		baseURL:     baseURL,
		httpClient:  http.DefaultClient,
		timeout:     30 * time.Second,
		maxRetries:  3,
		retryConfig: DefaultRetryConfig(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// BaseURL returns the base URL of the client.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// apiVersion represents the API version to use.
type apiVersion string

const (
	apiV2 apiVersion = "/api/2.0/mlflow"
	apiV3 apiVersion = "/api/3.0/mlflow"
)

// doRequest performs an HTTP request with retry logic.
func (c *Client) doRequest(ctx context.Context, method, path string, version apiVersion, body any, result any) error {
	fullURL := c.baseURL + string(version) + path

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.doSingleRequest(ctx, method, fullURL, body, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !c.isRetryable(err) {
			return err
		}
	}

	return lastErr
}

// doSingleRequest performs a single HTTP request.
func (c *Client) doSingleRequest(ctx context.Context, method, fullURL string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.wrapError(fullURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return c.parseAPIError(resp.StatusCode, respBody)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// wrapError wraps network errors in appropriate error types.
func (c *Client) wrapError(urlStr string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &TimeoutError{
			Operation: urlStr,
			Timeout:   c.timeout,
			Cause:     err,
		}
	}

	if errors.Is(err, context.Canceled) {
		return err
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return &ConnectionError{
			URL:     urlStr,
			Message: netErr.Error(),
			Cause:   err,
		}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &ConnectionError{
			URL:     urlStr,
			Message: urlErr.Error(),
			Cause:   err,
		}
	}

	return err
}

// parseAPIError parses an error response from the API.
func (c *Client) parseAPIError(statusCode int, body []byte) error {
	apiErr := &APIError{
		StatusCode: statusCode,
	}

	var errResp struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		apiErr.ErrorCode = errResp.ErrorCode
		apiErr.Message = errResp.Message
	} else {
		apiErr.Message = string(body)
	}

	return apiErr
}

// isRetryable checks if an error is retryable.
func (c *Client) isRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRetryable()
	}

	var connErr *ConnectionError
	if errors.As(err, &connErr) {
		return connErr.IsRetryable()
	}

	return false
}

// calculateBackoff calculates the backoff duration for a retry attempt.
func (c *Client) calculateBackoff(attempt int) time.Duration {
	delay := float64(c.retryConfig.InitialDelay)
	for i := 1; i < attempt; i++ {
		delay *= c.retryConfig.BackoffFactor
	}

	if delay > float64(c.retryConfig.MaxDelay) {
		delay = float64(c.retryConfig.MaxDelay)
	}

	// Add jitter
	jitter := delay * c.retryConfig.JitterFraction * (rand.Float64()*2 - 1)
	delay += jitter

	return time.Duration(delay)
}
