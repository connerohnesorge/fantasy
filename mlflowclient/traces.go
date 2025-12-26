package mlflowclient

import (
	"context"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// Trace wraps the generated TraceInfoV3 proto type for API transport.
type Trace struct {
	*pb.TraceInfoV3
}

// StartTrace creates a new trace within an experiment using the V3 API.
func (c *Client) StartTrace(ctx context.Context, trace *Trace) (*Trace, error) {
	if trace == nil || trace.TraceInfoV3 == nil {
		return nil, &ValidationError{
			Field:   "trace",
			Message: "trace cannot be nil",
			Value:   trace,
		}
	}

	if trace.TraceId == nil || *trace.TraceId == "" {
		return nil, &ValidationError{
			Field:   "traceID",
			Message: "trace ID cannot be empty",
			Value:   trace.TraceId,
		}
	}

	req := &pb.StartTraceV3{
		Trace: &pb.Trace{
			TraceInfo: trace.TraceInfoV3,
		},
	}

	resp := &pb.StartTraceV3_Response{}
	if err := c.doRequest(ctx, "POST", "/api/3.0/mlflow/traces", req, resp); err != nil {
		return nil, err
	}

	if resp.Trace == nil || resp.Trace.TraceInfo == nil {
		return nil, &APIError{
			StatusCode: 500,
			Message:    "server returned empty trace",
		}
	}

	return &Trace{TraceInfoV3: resp.Trace.TraceInfo}, nil
}

// GetTrace retrieves a trace by ID using the V3 API.
func (c *Client) GetTrace(ctx context.Context, traceID string, allowPartial bool) (*pb.Trace, error) {
	if traceID == "" {
		return nil, &ValidationError{
			Field:   "traceID",
			Message: "trace ID cannot be empty",
			Value:   traceID,
		}
	}

	resp := &pb.GetTrace_Response{}
	path := "/api/3.0/mlflow/traces/get?trace_id=" + traceID
	if allowPartial {
		path += "&allow_partial=true"
	}

	if err := c.doRequest(ctx, "GET", path, nil, resp); err != nil {
		return nil, err
	}

	return resp.Trace, nil
}

// SearchTracesOptions defines options for searching traces.
type SearchTracesOptions struct {
	ExperimentIDs []string // Experiment IDs to search within
	Filter        string   // Filter string (e.g., "status = 'OK'", "tags.env = 'prod'")
	MaxResults    int      // Maximum results to return (default: 100, max: 1000)
	PageToken     string   // Token for pagination continuation
	OrderBy       []string // Order by columns (e.g., ["timestamp_ms DESC"])
}

// SearchTracesResult contains search results and pagination info.
type SearchTracesResult struct {
	Traces        []*pb.TraceInfoV3
	NextPageToken string
}

// SearchTraces searches for traces matching the given criteria using the V3 API.
func (c *Client) SearchTraces(ctx context.Context, opts *SearchTracesOptions) (*SearchTracesResult, error) {
	req := &pb.SearchTracesV3{}

	if opts != nil {
		if len(opts.ExperimentIDs) > 0 {
			locations := make([]*pb.TraceLocation, len(opts.ExperimentIDs))
			for i, expID := range opts.ExperimentIDs {
				locType := pb.TraceLocation_MLFLOW_EXPERIMENT
				locations[i] = &pb.TraceLocation{
					Type: &locType,
					Identifier: &pb.TraceLocation_MlflowExperiment{
						MlflowExperiment: &pb.TraceLocation_MlflowExperimentLocation{
							ExperimentId: &expID,
						},
					},
				}
			}
			req.Locations = locations
		}

		if opts.Filter != "" {
			req.Filter = &opts.Filter
		}

		if opts.MaxResults > 0 {
			maxResults := int32(opts.MaxResults)
			req.MaxResults = &maxResults
		}

		if opts.PageToken != "" {
			req.PageToken = &opts.PageToken
		}

		if len(opts.OrderBy) > 0 {
			req.OrderBy = opts.OrderBy
		}
	}

	resp := &pb.SearchTracesV3_Response{}
	if err := c.doRequest(ctx, "POST", "/api/3.0/mlflow/traces/search", req, resp); err != nil {
		return nil, err
	}

	result := &SearchTracesResult{
		Traces: resp.Traces,
	}

	if resp.NextPageToken != nil {
		result.NextPageToken = *resp.NextPageToken
	}

	return result, nil
}

// DeleteTracesOptions defines options for deleting traces.
type DeleteTracesOptions struct {
	// At least one of these must be specified:
	MaxTraces      int    // Maximum number of traces to delete
	MaxTimestampMs int64  // Delete traces older than this timestamp (milliseconds since epoch)
	Filter         string // Filter string to select traces for deletion
}

// DeleteTraces deletes traces based on the specified criteria using the V3 API.
func (c *Client) DeleteTraces(ctx context.Context, experimentID string, opts *DeleteTracesOptions) (int, error) {
	if experimentID == "" {
		return 0, &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	if opts == nil || (opts.MaxTraces == 0 && opts.MaxTimestampMs == 0 && opts.Filter == "") {
		return 0, &ValidationError{
			Field:   "options",
			Message: "at least one of MaxTraces, MaxTimestampMs, or Filter must be specified",
			Value:   opts,
		}
	}

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

	// Note: Filter is not supported in V3 API. Use RequestIds for ID-based deletion or
	// time-based deletion with MaxTraces and MaxTimestampMs

	resp := &pb.DeleteTracesV3_Response{}
	if err := c.doRequest(ctx, "POST", "/api/3.0/mlflow/traces/delete-traces", req, resp); err != nil {
		return 0, err
	}

	if resp.TracesDeleted == nil {
		return 0, nil
	}

	return int(*resp.TracesDeleted), nil
}
