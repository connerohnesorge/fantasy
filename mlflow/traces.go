package mlflow

import (
	"context"
	"fmt"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// SearchTracesOptions defines options for searching traces.
type SearchTracesOptions struct {
	ExperimentIDs []string // Experiment IDs to search within
	Filter        string   // Filter string (e.g., "status = 'OK'", "tags.env = 'prod'")
	MaxResults    int      // Maximum results to return (default: 100, max: 1000)
	PageToken     string   // Token for pagination continuation
	OrderBy       []string // Order by columns (e.g., ["timestamp_ms DESC"])
}

// DeleteTracesOptions defines options for deleting traces.
// At least one of MaxTraces, MaxTimestampMs, or Filter must be specified.
type DeleteTracesOptions struct {
	// At least one of these must be specified:
	MaxTraces      int    // Maximum number of traces to delete
	MaxTimestampMs int64  // Delete traces older than this timestamp (milliseconds since epoch)
	Filter         string // Filter string to select traces for deletion
}

// SearchTracesResult represents the result of a trace search operation.
type SearchTracesResult struct {
	Traces        []*pb.TraceInfoV3
	NextPageToken string
}

// StartTrace creates a new trace in MLflow.
// Despite the name "Start", this operation uploads a COMPLETE trace to MLflow.
// The trace must have all spans already populated (tracing.Trace → mlflow.Trace conversion happens before this call).
// MLflow stores the trace immutably; no spans can be added after StartTrace.
//
// Returns the trace ID as confirmation.
func (c *Client) StartTrace(ctx context.Context, trace *pb.Trace) (string, error) {
	if trace == nil {
		return "", &ValidationError{
			Field:   "trace",
			Message: "trace cannot be nil",
			Value:   nil,
		}
	}

	if trace.TraceInfo == nil {
		return "", &ValidationError{
			Field:   "trace.TraceInfo",
			Message: "trace info cannot be nil",
			Value:   nil,
		}
	}

	// Create the request using the StartTraceV3 message
	req := &pb.StartTraceV3{
		Trace: trace,
	}

	// Response will contain the created trace
	resp := &pb.StartTraceV3_Response{}

	// Make the request to the v3 API
	if err := c.doRequest(ctx, "POST", "/api/3.0/mlflow/traces", req, resp); err != nil {
		return "", fmt.Errorf("failed to start trace: %w", err)
	}

	// Extract and return the trace ID from the response
	if resp.Trace == nil || resp.Trace.TraceInfo == nil || resp.Trace.TraceInfo.TraceId == nil {
		return "", fmt.Errorf("response missing trace_id")
	}

	return *resp.Trace.TraceInfo.TraceId, nil
}

// GetTrace retrieves a trace by ID.
//
// allowPartial Parameter:
// - allowPartial = false (default): Returns error if trace state is IN_PROGRESS
// - allowPartial = true: Returns whatever data is available even for IN_PROGRESS traces
//
// Note: Since our client uploads complete traces via StartTrace, you will typically only see
// IN_PROGRESS traces if there was a failure during upload or if reading traces uploaded by
// other clients that use incremental span addition.
func (c *Client) GetTrace(ctx context.Context, traceID string, allowPartial bool) (*pb.Trace, error) {
	if traceID == "" {
		return nil, &ValidationError{
			Field:   "traceID",
			Message: "traceID cannot be empty",
			Value:   "",
		}
	}

	// Response will be the GetTrace response containing the full trace
	resp := &pb.GetTrace_Response{}

	// Make the request (using GET with query params)
	// MLflow REST API uses GET with query params for this endpoint
	path := fmt.Sprintf("/api/3.0/mlflow/traces/get?trace_id=%s", traceID)
	if allowPartial {
		path += "&allow_partial=true"
	}

	if err := c.doRequest(ctx, "GET", path, nil, resp); err != nil {
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}

	return resp.Trace, nil
}

// SearchTraces searches for traces matching the given criteria.
// Returns matching trace infos with pagination support.
func (c *Client) SearchTraces(ctx context.Context, opts SearchTracesOptions) (*SearchTracesResult, error) {
	// Set defaults
	maxResults := int32(opts.MaxResults)
	if maxResults == 0 {
		maxResults = 100
	}
	if maxResults > 1000 {
		return nil, &ValidationError{
			Field:   "maxResults",
			Message: "maxResults cannot exceed 1000",
			Value:   maxResults,
		}
	}

	// Build TraceLocation objects from experiment IDs
	var locations []*pb.TraceLocation
	for _, expID := range opts.ExperimentIDs {
		expIDCopy := expID // Create a copy for the pointer
		locType := pb.TraceLocation_MLFLOW_EXPERIMENT
		locations = append(locations, &pb.TraceLocation{
			Type: &locType,
			Identifier: &pb.TraceLocation_MlflowExperiment{
				MlflowExperiment: &pb.TraceLocation_MlflowExperimentLocation{
					ExperimentId: &expIDCopy,
				},
			},
		})
	}

	// Build the request using SearchTracesV3
	req := &pb.SearchTracesV3{
		Locations:  locations,
		MaxResults: &maxResults,
	}

	if opts.Filter != "" {
		req.Filter = &opts.Filter
	}
	if opts.PageToken != "" {
		req.PageToken = &opts.PageToken
	}
	if len(opts.OrderBy) > 0 {
		req.OrderBy = opts.OrderBy
	}

	// Create the response
	resp := &pb.SearchTracesV3_Response{}

	// Make the request
	if err := c.doRequest(ctx, "POST", "/api/3.0/mlflow/traces/search", req, resp); err != nil {
		return nil, fmt.Errorf("failed to search traces: %w", err)
	}

	// Build the result
	result := &SearchTracesResult{
		Traces: resp.Traces,
	}
	if resp.NextPageToken != nil {
		result.NextPageToken = *resp.NextPageToken
	}

	return result, nil
}

// DeleteTraces deletes traces matching the given criteria.
// At least one of MaxTraces, MaxTimestampMs, or Filter must be specified.
//
// Return Value Semantics:
// - Returns (deletedCount int, err error)
// - deletedCount is the number of traces actually deleted
// - If filter matches 100 traces but MaxTraces=50, deletedCount=50 (partial deletion)
// - If no traces match the criteria, deletedCount=0 (not an error)
// - If server fails mid-deletion, returns error with partial deletedCount if available.
func (c *Client) DeleteTraces(ctx context.Context, experimentID string, opts DeleteTracesOptions) (int, error) {
	if experimentID == "" {
		return 0, &ValidationError{
			Field:   "experimentID",
			Message: "experimentID cannot be empty",
			Value:   "",
		}
	}

	// Validate that at least one deletion criteria is specified
	if opts.MaxTraces == 0 && opts.MaxTimestampMs == 0 && opts.Filter == "" {
		return 0, &ValidationError{
			Field:   "opts",
			Message: "at least one of MaxTraces, MaxTimestampMs, or Filter must be specified",
			Value:   opts,
		}
	}

	// Build the request using DeleteTracesV3
	req := &pb.DeleteTracesV3{
		ExperimentId: &experimentID,
	}

	if opts.MaxTraces > 0 {
		maxTraces := int32(opts.MaxTraces)
		req.MaxTraces = &maxTraces
	}
	if opts.MaxTimestampMs > 0 {
		req.MaxTimestampMillis = &opts.MaxTimestampMs
	}
	if opts.Filter != "" {
		req.Filter = &opts.Filter
	}

	// Create the response
	resp := &pb.DeleteTracesV3_Response{}

	// Make the request
	if err := c.doRequest(ctx, "POST", "/api/3.0/mlflow/traces/delete-traces", req, resp); err != nil {
		return 0, fmt.Errorf("failed to delete traces: %w", err)
	}

	// Extract the deleted count
	if resp.TracesDeleted == nil {
		return 0, nil
	}

	return int(*resp.TracesDeleted), nil
}
