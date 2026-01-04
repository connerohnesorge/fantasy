package mlflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Assessment represents feedback or expectation attached to a trace.
type Assessment struct {
	ID          string            `json:"assessment_id,omitempty"`
	Name        string            `json:"name"`
	Source      AssessmentSource  `json:"source"`
	TraceID     string            `json:"trace_id,omitempty"`
	SpanID      string            `json:"span_id,omitempty"`
	Rationale   string            `json:"rationale,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Feedback    *FeedbackValue    `json:"feedback,omitempty"`
	Expectation *ExpectationValue `json:"expectation,omitempty"`
}

// AssessmentSource identifies who/what created the assessment.
type AssessmentSource struct {
	SourceType string `json:"source_type"` // "CODE", "HUMAN", or "LLM_JUDGE"
	SourceID   string `json:"source_id"`   // Identifier (e.g., scorer name, user email, model name)
}

// FeedbackValue contains the actual feedback score.
type FeedbackValue struct {
	Value any              `json:"value"`           // bool, float64, int, string, or structured value
	Error *AssessmentError `json:"error,omitempty"` // Error if scoring failed (optional)
}

// ExpectationValue contains the expected/ground truth value.
type ExpectationValue struct {
	Value any `json:"value"` // Expected value (JSON-serializable)
}

// AssessmentError captures scorer failure information.
type AssessmentError struct {
	ErrorMessage string `json:"error_message"`
	ErrorCode    string `json:"error_code,omitempty"`
	StackTrace   string `json:"stack_trace,omitempty"`
}

// CreateAssessment creates a new assessment for a trace.
//
// API endpoint: POST /api/3.0/mlflow/traces/{trace_id}/assessments
//
// Parameters:
//   - ctx: Context for the request
//   - traceID: The trace ID to attach the assessment to
//   - assessment: The assessment to create (exactly one of Feedback or Expectation must be set)
//
// Returns the created assessment with assessment_id populated.
func (c *Client) CreateAssessment(ctx context.Context, traceID string, assessment *Assessment) (*Assessment, error) {
	// Validate required fields
	if traceID == "" {
		return nil, &ValidationError{
			Field:   "traceID",
			Message: "trace ID is required",
			Value:   traceID,
		}
	}
	if assessment == nil {
		return nil, &ValidationError{
			Field:   "assessment",
			Message: "assessment is required",
			Value:   nil,
		}
	}
	if assessment.Name == "" {
		return nil, &ValidationError{
			Field:   "name",
			Message: "assessment name is required",
			Value:   assessment.Name,
		}
	}
	if assessment.Source.SourceType == "" {
		return nil, &ValidationError{
			Field:   "source.source_type",
			Message: "source type is required",
			Value:   assessment.Source.SourceType,
		}
	}

	// Validate that exactly one of Feedback or Expectation is set
	if (assessment.Feedback == nil && assessment.Expectation == nil) ||
		(assessment.Feedback != nil && assessment.Expectation != nil) {
		return nil, &ValidationError{
			Field:   "feedback/expectation",
			Message: "exactly one of Feedback or Expectation must be set",
			Value:   nil,
		}
	}

	// Build the API path
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/assessments", traceID)

	// The API expects the assessment to be wrapped in an "assessment" key
	requestBody := map[string]any{
		"assessment": assessment,
	}

	// Response will contain the created assessment wrapped in "assessment" key
	var result struct {
		Assessment Assessment `json:"assessment"`
	}
	if err := c.doJSONRequest(ctx, http.MethodPost, path, requestBody, &result); err != nil {
		return nil, err
	}

	return &result.Assessment, nil
}

// UpdateAssessment updates an existing assessment.
//
// API endpoint: PATCH /api/3.0/mlflow/traces/{trace_id}/assessments/{assessment_id}
//
// Parameters:
//   - ctx: Context for the request
//   - traceID: The trace ID
//   - assessmentID: The assessment ID to update
//   - assessment: The updated assessment data
//   - updateMask: List of field names to update (e.g., ["rationale", "feedback.value"])
//
// Returns the updated assessment.
func (c *Client) UpdateAssessment(ctx context.Context, traceID, assessmentID string, assessment *Assessment, updateMask []string) (*Assessment, error) {
	// Validate required fields
	if traceID == "" {
		return nil, &ValidationError{
			Field:   "traceID",
			Message: "trace ID is required",
			Value:   traceID,
		}
	}
	if assessmentID == "" {
		return nil, &ValidationError{
			Field:   "assessmentID",
			Message: "assessment ID is required",
			Value:   assessmentID,
		}
	}
	if assessment == nil {
		return nil, &ValidationError{
			Field:   "assessment",
			Message: "assessment is required",
			Value:   nil,
		}
	}
	if len(updateMask) == 0 {
		return nil, &ValidationError{
			Field:   "updateMask",
			Message: "update mask is required and must not be empty",
			Value:   updateMask,
		}
	}

	// Build the API path
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/assessments/%s", traceID, assessmentID)

	// Create request payload with update_mask
	requestPayload := map[string]any{
		"assessment":  assessment,
		"update_mask": updateMask,
	}

	// Make the request
	var result Assessment
	if err := c.doJSONRequest(ctx, http.MethodPatch, path, requestPayload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteAssessment deletes an assessment.
//
// API endpoint: DELETE /api/3.0/mlflow/traces/{trace_id}/assessments/{assessment_id}
//
// Parameters:
//   - ctx: Context for the request
//   - traceID: The trace ID
//   - assessmentID: The assessment ID to delete
func (c *Client) DeleteAssessment(ctx context.Context, traceID, assessmentID string) error {
	// Validate required fields
	if traceID == "" {
		return &ValidationError{
			Field:   "traceID",
			Message: "trace ID is required",
			Value:   traceID,
		}
	}
	if assessmentID == "" {
		return &ValidationError{
			Field:   "assessmentID",
			Message: "assessment ID is required",
			Value:   assessmentID,
		}
	}

	// Build the API path
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/assessments/%s", traceID, assessmentID)

	// Make the DELETE request (no request or response body)
	return c.doJSONRequest(ctx, http.MethodDelete, path, nil, nil)
}

// doJSONRequest performs an HTTP request with JSON serialization (not protobuf).
// This is used for endpoints that don't have proto definitions.
func (c *Client) doJSONRequest(ctx context.Context, method, path string, req, resp any) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// If this is a retry, wait with exponential backoff
		if attempt > 0 {
			backoff := calculateBackoff(c.retryConfig, attempt-1)
			if err := sleep(ctx, backoff); err != nil {
				return &TimeoutError{
					Operation: fmt.Sprintf("%s %s", method, path),
					Timeout:   int64(c.timeout.Milliseconds()),
					Cause:     err,
				}
			}
		}

		// Perform the actual request
		err := c.doSingleJSONRequest(ctx, method, path, req, resp)
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

// doSingleJSONRequest performs a single HTTP request with JSON serialization.
func (c *Client) doSingleJSONRequest(ctx context.Context, method, path string, req, resp any) error {
	// Build the full URL
	url := c.baseURL + path

	// Serialize request body if provided
	var bodyReader io.Reader
	if req != nil {
		data, err := json.Marshal(req)
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
				Timeout:   int64(c.timeout.Milliseconds()),
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
	if resp != nil && len(body) > 0 {
		if err := json.Unmarshal(body, resp); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
