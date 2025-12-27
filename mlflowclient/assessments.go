package mlflowclient

import (
	"context"
	"fmt"
)

// CreateAssessment creates a new assessment attached to a trace.
func (c *Client) CreateAssessment(ctx context.Context, traceID string, assessment *Assessment) (*Assessment, error) {
	if traceID == "" {
		return nil, &ValidationError{Field: "traceID", Message: "trace ID is required"}
	}
	if assessment == nil {
		return nil, &ValidationError{Field: "assessment", Message: "assessment is required"}
	}
	if assessment.Name == "" {
		return nil, &ValidationError{Field: "assessment.Name", Message: "assessment name is required"}
	}
	if assessment.Feedback == nil && assessment.Expectation == nil {
		return nil, &ValidationError{Field: "assessment", Message: "either Feedback or Expectation must be set"}
	}

	req := map[string]any{
		"assessment": assessment,
	}

	var resp struct {
		Assessment *Assessment `json:"assessment"`
	}

	path := fmt.Sprintf("/traces/%s/assessments", traceID)
	if err := c.doRequest(ctx, "POST", path, apiV3, req, &resp); err != nil {
		return nil, err
	}

	return resp.Assessment, nil
}

// UpdateAssessment updates an existing assessment.
// Only fields specified in updateMask are updated.
func (c *Client) UpdateAssessment(ctx context.Context, traceID, assessmentID string, assessment *Assessment, updateMask []string) (*Assessment, error) {
	if traceID == "" {
		return nil, &ValidationError{Field: "traceID", Message: "trace ID is required"}
	}
	if assessmentID == "" {
		return nil, &ValidationError{Field: "assessmentID", Message: "assessment ID is required"}
	}
	if assessment == nil {
		return nil, &ValidationError{Field: "assessment", Message: "assessment is required"}
	}

	req := map[string]any{
		"assessment": assessment,
	}

	if len(updateMask) > 0 {
		req["update_mask"] = updateMask
	}

	var resp struct {
		Assessment *Assessment `json:"assessment"`
	}

	path := fmt.Sprintf("/traces/%s/assessments/%s", traceID, assessmentID)
	if err := c.doRequest(ctx, "PATCH", path, apiV3, req, &resp); err != nil {
		return nil, err
	}

	return resp.Assessment, nil
}

// DeleteAssessment deletes an assessment.
func (c *Client) DeleteAssessment(ctx context.Context, traceID, assessmentID string) error {
	if traceID == "" {
		return &ValidationError{Field: "traceID", Message: "trace ID is required"}
	}
	if assessmentID == "" {
		return &ValidationError{Field: "assessmentID", Message: "assessment ID is required"}
	}

	path := fmt.Sprintf("/traces/%s/assessments/%s", traceID, assessmentID)
	return c.doRequest(ctx, "DELETE", path, apiV3, nil, nil)
}
