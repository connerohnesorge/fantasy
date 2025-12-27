package mlflowclient

import (
	"net/http"
	"time"
)

// Option configures the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithToken sets the Bearer token for authentication.
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithRetries sets the maximum number of retries for retryable errors.
// Set to 0 to disable retries.
func WithRetries(maxRetries int) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// WithRetryConfig sets custom retry configuration.
func WithRetryConfig(config RetryConfig) Option {
	return func(c *Client) {
		c.retryConfig = config
	}
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retries (default: 3)
	InitialDelay   time.Duration // Initial delay before first retry (default: 1s)
	MaxDelay       time.Duration // Maximum delay between retries (default: 10s)
	BackoffFactor  float64       // Backoff multiplier (default: 2.0)
	JitterFraction float64       // Jitter as fraction of delay (default: 0.1)
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialDelay:   1 * time.Second,
		MaxDelay:       10 * time.Second,
		BackoffFactor:  2.0,
		JitterFraction: 0.1,
	}
}

// SearchExperimentsOptions configures experiment search.
type SearchExperimentsOptions struct {
	Filter     string   // Filter string (e.g., "name LIKE 'my-%'")
	MaxResults int      // Maximum results to return (default: 1000)
	PageToken  string   // Token for pagination continuation
	OrderBy    []string // Order by columns (e.g., ["name ASC"])
	ViewType   string   // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL"
}

// SearchRunsOptions configures run search.
type SearchRunsOptions struct {
	ExperimentIDs []string // Experiment IDs to search within (required)
	Filter        string   // Filter string (e.g., "metrics.accuracy > 0.9")
	RunViewType   string   // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL"
	MaxResults    int      // Maximum results to return (default: 1000)
	OrderBy       []string // Order by columns (e.g., ["metrics.accuracy DESC"])
	PageToken     string   // Token for pagination continuation
}

// SearchTracesOptions configures trace search.
type SearchTracesOptions struct {
	ExperimentIDs []string // Experiment IDs to search within
	Filter        string   // Filter string (e.g., "state = 'OK'")
	MaxResults    int      // Maximum results to return (default: 100)
	PageToken     string   // Token for pagination continuation
	OrderBy       []string // Order by columns (e.g., ["timestamp_ms DESC"])
}

// DeleteTracesOptions configures trace deletion.
type DeleteTracesOptions struct {
	MaxTraces      int    // Maximum number of traces to delete
	MaxTimestampMs int64  // Delete traces older than this timestamp
	Filter         string // Filter string to select traces for deletion
}

// CreateRunOption configures run creation.
type CreateRunOption func(*createRunConfig)

type createRunConfig struct {
	runName   string
	startTime int64
	tags      map[string]string
}

// WithRunName sets the run name.
func WithRunName(name string) CreateRunOption {
	return func(c *createRunConfig) {
		c.runName = name
	}
}

// WithStartTime sets the run start time in milliseconds since epoch.
func WithStartTime(t int64) CreateRunOption {
	return func(c *createRunConfig) {
		c.startTime = t
	}
}

// WithTags sets initial tags for the run.
func WithTags(tags map[string]string) CreateRunOption {
	return func(c *createRunConfig) {
		c.tags = tags
	}
}
