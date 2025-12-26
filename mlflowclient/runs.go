package mlflowclient

import (
	"context"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// CreateRunOptions defines optional parameters for creating a run.
type CreateRunOptions struct {
	RunName   string
	StartTime int64
	Tags      map[string]string
}

// CreateRun creates a new run within an experiment.
func (c *Client) CreateRun(ctx context.Context, experimentID string, opts ...func(*CreateRunOptions)) (*pb.Run, error) {
	if experimentID == "" {
		return nil, &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	// Apply options
	options := &CreateRunOptions{}
	for _, opt := range opts {
		opt(options)
	}

	req := &pb.CreateRun{
		ExperimentId: &experimentID,
	}

	if options.RunName != "" {
		req.RunName = &options.RunName
	}

	if options.StartTime > 0 {
		req.StartTime = &options.StartTime
	}

	if len(options.Tags) > 0 {
		req.Tags = make([]*pb.RunTag, 0, len(options.Tags))
		for k, v := range options.Tags {
			key := k
			val := v
			req.Tags = append(req.Tags, &pb.RunTag{
				Key:   &key,
				Value: &val,
			})
		}
	}

	resp := &pb.CreateRun_Response{}
	if err := c.doRequest(ctx, "POST", "/api/2.0/mlflow/runs/create", req, resp); err != nil {
		return nil, err
	}

	return resp.Run, nil
}

// WithRunName sets the run name option.
func WithRunName(name string) func(*CreateRunOptions) {
	return func(opts *CreateRunOptions) {
		opts.RunName = name
	}
}

// WithStartTime sets the start time option (milliseconds since epoch).
func WithStartTime(t int64) func(*CreateRunOptions) {
	return func(opts *CreateRunOptions) {
		opts.StartTime = t
	}
}

// WithTags sets the tags option.
func WithTags(tags map[string]string) func(*CreateRunOptions) {
	return func(opts *CreateRunOptions) {
		opts.Tags = tags
	}
}

// GetRun retrieves a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*pb.Run, error) {
	if runID == "" {
		return nil, &ValidationError{
			Field:   "runID",
			Message: "run ID cannot be empty",
			Value:   runID,
		}
	}

	resp := &pb.GetRun_Response{}
	if err := c.doRequest(ctx, "GET", "/api/2.0/mlflow/runs/get?run_id="+runID, nil, resp); err != nil {
		return nil, err
	}

	return resp.Run, nil
}

// UpdateRun updates run metadata including status and end time.
func (c *Client) UpdateRun(ctx context.Context, runID string, status string, endTime int64) (*pb.RunInfo, error) {
	if runID == "" {
		return nil, &ValidationError{
			Field:   "runID",
			Message: "run ID cannot be empty",
			Value:   runID,
		}
	}

	req := &pb.UpdateRun{
		RunId: &runID,
	}

	if status != "" {
		runStatus := pb.RunStatus(pb.RunStatus_value[status])
		req.Status = &runStatus
	}

	if endTime > 0 {
		req.EndTime = &endTime
	}

	resp := &pb.UpdateRun_Response{}
	if err := c.doRequest(ctx, "POST", "/api/2.0/mlflow/runs/update", req, resp); err != nil {
		return nil, err
	}

	return resp.RunInfo, nil
}

// LogBatch logs metrics, params, and tags for a run in a single batch operation.
func (c *Client) LogBatch(ctx context.Context, runID string, metrics []*pb.Metric, params []*pb.Param, tags []*pb.RunTag) error {
	if runID == "" {
		return &ValidationError{
			Field:   "runID",
			Message: "run ID cannot be empty",
			Value:   runID,
		}
	}

	req := &pb.LogBatch{
		RunId: &runID,
	}

	if len(metrics) > 0 {
		req.Metrics = metrics
	}

	if len(params) > 0 {
		req.Params = params
	}

	if len(tags) > 0 {
		req.Tags = tags
	}

	resp := &pb.LogBatch_Response{}
	return c.doRequest(ctx, "POST", "/api/2.0/mlflow/runs/log-batch", req, resp)
}

// SearchRunsOptions defines options for searching runs.
type SearchRunsOptions struct {
	ExperimentIDs []string // Experiment IDs to search within (required)
	Filter        string   // Filter string (e.g., "metrics.accuracy > 0.9")
	RunViewType   string   // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL" (default: "ACTIVE_ONLY")
	MaxResults    int      // Maximum results to return (default: 1000, max: 50000)
	OrderBy       []string // Order by columns (e.g., ["metrics.accuracy DESC"])
	PageToken     string   // Token for pagination continuation
}

// SearchRunsResult contains search results and pagination info.
type SearchRunsResult struct {
	Runs          []*pb.Run
	NextPageToken string
}

// SearchRuns searches for runs matching the given criteria.
func (c *Client) SearchRuns(ctx context.Context, opts *SearchRunsOptions) (*SearchRunsResult, error) {
	if opts == nil || len(opts.ExperimentIDs) == 0 {
		return nil, &ValidationError{
			Field:   "experimentIDs",
			Message: "at least one experiment ID is required",
			Value:   opts,
		}
	}

	req := &pb.SearchRuns{
		ExperimentIds: opts.ExperimentIDs,
	}

	if opts.Filter != "" {
		req.Filter = &opts.Filter
	}

	if opts.RunViewType != "" {
		viewType := pb.ViewType(pb.ViewType_value[opts.RunViewType])
		req.RunViewType = &viewType
	}

	if opts.MaxResults > 0 {
		maxResults := int32(opts.MaxResults)
		req.MaxResults = &maxResults
	}

	if len(opts.OrderBy) > 0 {
		req.OrderBy = opts.OrderBy
	}

	if opts.PageToken != "" {
		req.PageToken = &opts.PageToken
	}

	resp := &pb.SearchRuns_Response{}
	if err := c.doRequest(ctx, "POST", "/api/2.0/mlflow/runs/search", req, resp); err != nil {
		return nil, err
	}

	result := &SearchRunsResult{
		Runs: resp.Runs,
	}

	if resp.NextPageToken != nil {
		result.NextPageToken = *resp.NextPageToken
	}

	return result, nil
}
