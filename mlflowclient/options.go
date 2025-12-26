package mlflowclient

import (
	"net/http"
	"time"
)

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithToken sets the Bearer token for authentication.
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithTimeout sets the request timeout duration.
// Default is 30 seconds if not specified.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithRetries sets the maximum number of retry attempts.
// Default is 3 retries if not specified.
// Set to 0 to disable retries.
func WithRetries(maxRetries int) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// WithRetryConfig sets detailed retry configuration.
func WithRetryConfig(config RetryConfig) Option {
	return func(c *Client) {
		c.retryConfig = config
	}
}

// RetryConfig defines the retry behavior for failed requests.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	// Default: 3
	MaxRetries int

	// InitialDelay is the initial delay before the first retry.
	// Default: 1 second
	InitialDelay time.Duration

	// BackoffMultiplier is the multiplier for exponential backoff.
	// Default: 2.0 (doubles each retry)
	BackoffMultiplier float64

	// MaxDelay is the maximum delay between retries.
	// Default: 10 seconds
	MaxDelay time.Duration

	// JitterPercent is the percentage of jitter to apply (0-100).
	// Default: 10 (±10% randomization)
	JitterPercent float64
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialDelay:      1 * time.Second,
		BackoffMultiplier: 2.0,
		MaxDelay:          10 * time.Second,
		JitterPercent:     10.0,
	}
}
