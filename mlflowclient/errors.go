package mlflowclient

import (
	"fmt"
	"time"
)

// APIError represents an error response from the MLflow API.
type APIError struct {
	StatusCode int    // HTTP status code (e.g., 404, 500)
	Message    string // Error message from response body
	ErrorCode  string // MLflow error code (e.g., "RESOURCE_DOES_NOT_EXIST")
	Cause      error  // Underlying error (for error wrapping)
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.ErrorCode != "" {
		return fmt.Sprintf("mlflow api error %d (%s): %s", e.StatusCode, e.ErrorCode, e.Message)
	}
	return fmt.Sprintf("mlflow api error %d: %s", e.StatusCode, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *APIError) Unwrap() error {
	return e.Cause
}

// IsNotFound returns true for 404 status codes.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsConflict returns true for 409 status codes.
func (e *APIError) IsConflict() bool {
	return e.StatusCode == 409
}

// IsServerError returns true for 5xx status codes.
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsUnauthorized returns true for 401 status codes.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsForbidden returns true for 403 status codes.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}

// IsRateLimited returns true for 429 status codes.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}

// IsRetryable returns true for 429, 5xx, or connection errors.
func (e *APIError) IsRetryable() bool {
	return e.IsRateLimited() || e.IsServerError()
}

// ValidationError represents a client-side validation failure.
type ValidationError struct {
	Field   string // Field that failed validation (e.g., "experimentID", "traceID")
	Message string // Description of the validation failure
	Value   any    // The invalid value (for debugging)
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation error for %s: %s (value: %v)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// TimeoutError represents a request that exceeded its deadline.
type TimeoutError struct {
	Operation string        // Operation that timed out (e.g., "StartTrace", "GetRun")
	Timeout   time.Duration // The timeout that was exceeded
	Cause     error         // Underlying context.DeadlineExceeded or similar
}

// Error implements the error interface.
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("operation %s timed out after %s", e.Operation, e.Timeout)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *TimeoutError) Unwrap() error {
	return e.Cause
}

// ConnectionError represents a network-level failure.
type ConnectionError struct {
	URL     string // The URL that failed to connect
	Message string // Description of the connection failure
	Cause   error  // Underlying net error
}

// Error implements the error interface.
func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection error to %s: %s", e.URL, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns true (connection errors are retryable).
func (e *ConnectionError) IsRetryable() bool {
	return true
}
