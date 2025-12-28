package mlflow

import (
	"context"
	"fmt"
)

// SetTraceTag sets a tag on a trace.
// Uses PATCH method to /api/3.0/mlflow/traces/{trace_id}/tags.
func (c *Client) SetTraceTag(ctx context.Context, traceID, key, value string) error {
	// Validate inputs
	if traceID == "" {
		return &ValidationError{
			Field:   "traceID",
			Message: "trace ID cannot be empty",
			Value:   traceID,
		}
	}
	if key == "" {
		return &ValidationError{
			Field:   "key",
			Message: "tag key cannot be empty",
			Value:   key,
		}
	}

	// Build the request body
	reqBody := map[string]string{
		"key":   key,
		"value": value,
	}

	// Build the path
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/tags", traceID)

	// Make request using PATCH method
	return c.doJSONRequest(ctx, "PATCH", path, reqBody, nil)
}

// DeleteTraceTag deletes a tag from a trace.
// Uses DELETE method to /api/3.0/mlflow/traces/{trace_id}/tags/{key}.
func (c *Client) DeleteTraceTag(ctx context.Context, traceID, key string) error {
	// Validate inputs
	if traceID == "" {
		return &ValidationError{
			Field:   "traceID",
			Message: "trace ID cannot be empty",
			Value:   traceID,
		}
	}
	if key == "" {
		return &ValidationError{
			Field:   "key",
			Message: "tag key cannot be empty",
			Value:   key,
		}
	}

	// Build the path
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/tags/%s", traceID, key)

	// Make request using DELETE method
	return c.doJSONRequest(ctx, "DELETE", path, nil, nil)
}

// SetRunTag sets a tag on a run.
// Uses POST method to /api/2.0/mlflow/runs/set-tag.
func (c *Client) SetRunTag(ctx context.Context, runID, key, value string) error {
	// Validate inputs
	if runID == "" {
		return &ValidationError{
			Field:   "runID",
			Message: "run ID cannot be empty",
			Value:   runID,
		}
	}
	if key == "" {
		return &ValidationError{
			Field:   "key",
			Message: "tag key cannot be empty",
			Value:   key,
		}
	}

	// Build the request body
	reqBody := map[string]string{
		"run_id": runID,
		"key":    key,
		"value":  value,
	}

	// Build the path
	path := "/api/2.0/mlflow/runs/set-tag"

	// Make request using POST method
	return c.doJSONRequest(ctx, "POST", path, reqBody, nil)
}

// DeleteRunTag deletes a tag from a run.
// Uses POST method to /api/2.0/mlflow/runs/delete-tag.
func (c *Client) DeleteRunTag(ctx context.Context, runID, key string) error {
	// Validate inputs
	if runID == "" {
		return &ValidationError{
			Field:   "runID",
			Message: "run ID cannot be empty",
			Value:   runID,
		}
	}
	if key == "" {
		return &ValidationError{
			Field:   "key",
			Message: "tag key cannot be empty",
			Value:   key,
		}
	}

	// Build the request body
	reqBody := map[string]string{
		"run_id": runID,
		"key":    key,
	}

	// Build the path
	path := "/api/2.0/mlflow/runs/delete-tag"

	// Make request using POST method
	return c.doJSONRequest(ctx, "POST", path, reqBody, nil)
}
