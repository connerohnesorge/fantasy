package mlflowclient

import (
	"context"
	"fmt"

	pb "charm.land/fantasy/proto/gen/mlflow"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// CreateAssessment creates a new assessment for a trace.
func (c *Client) CreateAssessment(ctx context.Context, traceID string, assessment *pb.Assessment) (*pb.Assessment, error) {
	if traceID == "" {
		return nil, &ValidationError{
			Field:   "traceID",
			Message: "trace ID cannot be empty",
			Value:   traceID,
		}
	}

	if assessment == nil {
		return nil, &ValidationError{
			Field:   "assessment",
			Message: "assessment cannot be nil",
			Value:   assessment,
		}
	}

	req := &pb.CreateAssessment{
		Assessment: assessment,
	}

	resp := &pb.CreateAssessment_Response{}
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/assessments", traceID)
	if err := c.doRequest(ctx, "POST", path, req, resp); err != nil {
		return nil, err
	}

	return resp.Assessment, nil
}

// GetAssessment retrieves an assessment by trace ID and assessment ID.
func (c *Client) GetAssessment(ctx context.Context, traceID string, assessmentID string) (*pb.Assessment, error) {
	if traceID == "" {
		return nil, &ValidationError{
			Field:   "traceID",
			Message: "trace ID cannot be empty",
			Value:   traceID,
		}
	}

	if assessmentID == "" {
		return nil, &ValidationError{
			Field:   "assessmentID",
			Message: "assessment ID cannot be empty",
			Value:   assessmentID,
		}
	}

	resp := &pb.GetAssessmentRequest_Response{}
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/assessments/%s", traceID, assessmentID)
	if err := c.doRequest(ctx, "GET", path, nil, resp); err != nil {
		return nil, err
	}

	return resp.Assessment, nil
}

// UpdateAssessment updates an assessment with the specified fields.
// The mask parameter specifies which fields to update (e.g., ["rationale", "feedback.value"]).
func (c *Client) UpdateAssessment(ctx context.Context, traceID string, assessmentID string, assessment *pb.Assessment, mask []string) (*pb.Assessment, error) {
	if traceID == "" {
		return nil, &ValidationError{
			Field:   "traceID",
			Message: "trace ID cannot be empty",
			Value:   traceID,
		}
	}

	if assessmentID == "" {
		return nil, &ValidationError{
			Field:   "assessmentID",
			Message: "assessment ID cannot be empty",
			Value:   assessmentID,
		}
	}

	if assessment == nil {
		return nil, &ValidationError{
			Field:   "assessment",
			Message: "assessment cannot be nil",
			Value:   assessment,
		}
	}

	req := &pb.UpdateAssessment{
		Assessment: assessment,
	}

	if len(mask) > 0 {
		req.UpdateMask = &fieldmaskpb.FieldMask{
			Paths: mask,
		}
	}

	resp := &pb.UpdateAssessment_Response{}
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/assessments/%s", traceID, assessmentID)
	if err := c.doRequest(ctx, "PATCH", path, req, resp); err != nil {
		return nil, err
	}

	return resp.Assessment, nil
}

// DeleteAssessment deletes an assessment by trace ID and assessment ID.
func (c *Client) DeleteAssessment(ctx context.Context, traceID string, assessmentID string) error {
	if traceID == "" {
		return &ValidationError{
			Field:   "traceID",
			Message: "trace ID cannot be empty",
			Value:   traceID,
		}
	}

	if assessmentID == "" {
		return &ValidationError{
			Field:   "assessmentID",
			Message: "assessment ID cannot be empty",
			Value:   assessmentID,
		}
	}

	req := &pb.DeleteAssessment{
		TraceId:      &traceID,
		AssessmentId: &assessmentID,
	}

	resp := &pb.DeleteAssessment_Response{}
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/assessments/%s", traceID, assessmentID)
	return c.doRequest(ctx, "DELETE", path, req, resp)
}
