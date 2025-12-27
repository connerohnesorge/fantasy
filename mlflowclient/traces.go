package mlflowclient

import (
	"context"
	"fmt"
)

// StartTrace uploads a trace to MLflow.
// The trace must have ExperimentID and RequestTimeMs set at minimum.
// Returns the trace ID (request_id) assigned by MLflow.
func (c *Client) StartTrace(ctx context.Context, trace *Trace) (string, error) {
	if trace == nil {
		return "", &ValidationError{Field: "trace", Message: "trace is required"}
	}
	if trace.ExperimentID == "" {
		return "", &ValidationError{Field: "experimentID", Message: "experiment ID is required in trace"}
	}

	// Build request in MLflow's expected format
	req := map[string]any{
		"experiment_id": trace.ExperimentID,
		"timestamp_ms":  trace.RequestTimeMs,
	}

	if trace.ExecutionTimeMs > 0 {
		req["execution_time_ms"] = trace.ExecutionTimeMs
	}

	// Map TraceState to MLflow status
	switch trace.State {
	case TraceStateOK:
		req["status"] = "OK"
	case TraceStateError:
		req["status"] = "ERROR"
	case TraceStateInProgress:
		req["status"] = "IN_PROGRESS"
	}

	// Convert tags map to array format
	if len(trace.Tags) > 0 {
		tags := make([]map[string]string, 0, len(trace.Tags))
		for k, v := range trace.Tags {
			tags = append(tags, map[string]string{"key": k, "value": v})
		}
		req["tags"] = tags
	}

	// Add request/response metadata if present
	if trace.RequestPreview != "" {
		req["request_preview"] = trace.RequestPreview
	}
	if trace.ResponsePreview != "" {
		req["response_preview"] = trace.ResponsePreview
	}

	// Parse response
	var resp struct {
		TraceInfo struct {
			RequestID string `json:"request_id"`
		} `json:"trace_info"`
	}

	if err := c.doRequest(ctx, "POST", "/traces", apiV2, req, &resp); err != nil {
		return "", err
	}

	return resp.TraceInfo.RequestID, nil
}

// GetTrace retrieves a trace by ID.
// If allowPartial is true, returns partial data for IN_PROGRESS traces.
func (c *Client) GetTrace(ctx context.Context, traceID string, allowPartial bool) (*Trace, error) {
	if traceID == "" {
		return nil, &ValidationError{Field: "traceID", Message: "trace ID is required"}
	}

	path := fmt.Sprintf("/traces/get?trace_id=%s", traceID)
	if allowPartial {
		path += "&allow_partial=true"
	}

	var resp struct {
		Trace *Trace `json:"trace"`
	}

	if err := c.doRequest(ctx, "GET", path, apiV2, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Trace, nil
}

// SearchTraces searches for traces matching the criteria.
func (c *Client) SearchTraces(ctx context.Context, opts SearchTracesOptions) ([]*Trace, string, error) {
	req := make(map[string]any)

	if len(opts.ExperimentIDs) > 0 {
		req["experiment_ids"] = opts.ExperimentIDs
	}
	if opts.Filter != "" {
		req["filter"] = opts.Filter
	}
	if opts.MaxResults > 0 {
		req["max_results"] = opts.MaxResults
	}
	if opts.PageToken != "" {
		req["page_token"] = opts.PageToken
	}
	if len(opts.OrderBy) > 0 {
		req["order_by"] = opts.OrderBy
	}

	var resp struct {
		Traces        []*Trace `json:"traces"`
		NextPageToken string   `json:"next_page_token"`
	}

	if err := c.doRequest(ctx, "POST", "/traces/search", apiV2, req, &resp); err != nil {
		return nil, "", err
	}

	return resp.Traces, resp.NextPageToken, nil
}

// DeleteTraces deletes traces matching the criteria.
// Returns the number of traces deleted.
func (c *Client) DeleteTraces(ctx context.Context, experimentID string, opts DeleteTracesOptions) (int, error) {
	if experimentID == "" {
		return 0, &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}

	// At least one criteria must be specified
	if opts.MaxTraces == 0 && opts.MaxTimestampMs == 0 && opts.Filter == "" {
		return 0, &ValidationError{
			Field:   "opts",
			Message: "at least one of MaxTraces, MaxTimestampMs, or Filter must be specified",
		}
	}

	req := map[string]any{
		"experiment_id": experimentID,
	}

	if opts.MaxTraces > 0 {
		req["max_traces"] = opts.MaxTraces
	}
	if opts.MaxTimestampMs > 0 {
		req["max_timestamp_ms"] = opts.MaxTimestampMs
	}
	if opts.Filter != "" {
		req["filter"] = opts.Filter
	}

	var resp struct {
		DeletedCount int `json:"deleted_count"`
	}

	if err := c.doRequest(ctx, "POST", "/traces/delete-traces", apiV2, req, &resp); err != nil {
		return 0, err
	}

	return resp.DeletedCount, nil
}

// SetTraceTag sets a tag on a trace.
func (c *Client) SetTraceTag(ctx context.Context, traceID, key, value string) error {
	if traceID == "" {
		return &ValidationError{Field: "traceID", Message: "trace ID is required"}
	}
	if key == "" {
		return &ValidationError{Field: "key", Message: "tag key is required"}
	}

	req := map[string]any{
		"key":   key,
		"value": value,
	}

	path := fmt.Sprintf("/traces/%s/tags", traceID)
	return c.doRequest(ctx, "PATCH", path, apiV2, req, nil)
}

// DeleteTraceTag deletes a tag from a trace.
func (c *Client) DeleteTraceTag(ctx context.Context, traceID, key string) error {
	if traceID == "" {
		return &ValidationError{Field: "traceID", Message: "trace ID is required"}
	}
	if key == "" {
		return &ValidationError{Field: "key", Message: "tag key is required"}
	}

	path := fmt.Sprintf("/traces/%s/tags/%s", traceID, key)
	return c.doRequest(ctx, "DELETE", path, apiV2, nil, nil)
}
