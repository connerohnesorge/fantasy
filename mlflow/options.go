package mlflow

import (
	"net/http"
	"time"
)

// Option is a functional option for configuring a Client.
type Option func(*Client)

// WithToken sets the Bearer token for authentication.
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithTimeout sets the timeout for HTTP requests.
// Default is 30 seconds if not specified.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithHTTPClient sets a custom HTTP client.
// This is useful for testing or when you need custom transport configuration.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithRetries sets the maximum number of retries for failed requests.
// Default is 3 if not specified. Set to 0 to disable retries.
func WithRetries(maxRetries int) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}
