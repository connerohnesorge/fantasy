package mlflowclient

import (
	"context"
	"fmt"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// SetTraceTag sets a tag on a trace using the V3 API.
func (c *Client) SetTraceTag(ctx context.Context, traceID string, key string, value string) error {
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

	req := &pb.SetTraceTagV3{
		TraceId: &traceID,
		Key:     &key,
		Value:   &value,
	}

	resp := &pb.SetTraceTagV3_Response{}
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/tags", traceID)
	return c.doRequest(ctx, "PATCH", path, req, resp)
}

// DeleteTraceTag deletes a tag from a trace using the V3 API.
func (c *Client) DeleteTraceTag(ctx context.Context, traceID string, key string) error {
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

	req := &pb.DeleteTraceTagV3{
		TraceId: &traceID,
		Key:     &key,
	}

	resp := &pb.DeleteTraceTagV3_Response{}
	path := fmt.Sprintf("/api/3.0/mlflow/traces/%s/tags/%s", traceID, key)
	return c.doRequest(ctx, "DELETE", path, req, resp)
}

// SetRunTag sets a tag on a run.
func (c *Client) SetRunTag(ctx context.Context, runID string, key string, value string) error {
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

	req := &pb.SetTag{
		RunId: &runID,
		Key:   &key,
		Value: &value,
	}

	resp := &pb.SetTag_Response{}
	return c.doRequest(ctx, "POST", "/api/2.0/mlflow/runs/set-tag", req, resp)
}
