package mlflow

import (
	"fmt"
	"net"
)

// APIError represents an error response from the MLflow API.
type APIError struct {
	StatusCode int    // HTTP status code (e.g., 404, 500)
	Message    string // Error message from response body
	ErrorCode  string // MLflow error code (e.g., "RESOURCE_DOES_NOT_EXIST")
	Cause      error  // Underlying error (for error wrapping)
}

func (e *APIError) Error() string {
	if e.ErrorCode != "" {
		return fmt.Sprintf("mlflow api error (status %d, code %s): %s", e.StatusCode, e.ErrorCode, e.Message)
	}
	return fmt.Sprintf("mlflow api error (status %d): %s", e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Cause
}

func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

func (e *APIError) IsConflict() bool {
	return e.StatusCode == 409
}

func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}

func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}

func (e *APIError) IsRetryable() bool {
	return e.IsRateLimited() || e.IsServerError()
}

// ValidationError represents a client-side validation failure.
type ValidationError struct {
	Field   string // Field that failed validation (e.g., "experimentID", "traceID")
	Message string // Description of the validation failure
	Value   any    // The invalid value (for debugging)
}

func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation error: %s (field: %s, value: %v)", e.Message, e.Field, e.Value)
	}
	return fmt.Sprintf("validation error: %s (field: %s)", e.Message, e.Field)
}

// TimeoutError represents a request that exceeded its deadline.
type TimeoutError struct {
	Operation string // Operation that timed out (e.g., "StartTrace", "GetRun")
	Timeout   int64  // The timeout that was exceeded (in milliseconds)
	Cause     error  // Underlying context.DeadlineExceeded or similar
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout after %dms during %s: %v", e.Timeout, e.Operation, e.Cause)
}

func (e *TimeoutError) Unwrap() error {
	return e.Cause
}

// ConnectionError represents a network-level failure.
type ConnectionError struct {
	URL     string // The URL that failed to connect
	Message string // Description of the connection failure
	Cause   error  // Underlying net error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection error to %s: %s", e.URL, e.Message)
}

func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

func (e *ConnectionError) IsRetryable() bool {
	return true
}

// isNetworkError checks if an error is a network error that should be retried.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error interface
	if _, ok := err.(net.Error); ok {
		return true
	}

	// Check for connection-related errors by type
	switch err.(type) {
	case *net.OpError, *net.DNSError, *net.AddrError:
		return true
	}

	return false
}
