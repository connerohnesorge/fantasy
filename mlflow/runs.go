package mlflow

import (
	"context"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// SearchRunsOptions configures run search parameters.
type SearchRunsOptions struct {
	ExperimentIDs []string // Experiment IDs to search within (required)
	Filter        string   // Filter string (e.g., "metrics.accuracy > 0.9")
	RunViewType   string   // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL" (default: "ACTIVE_ONLY")
	MaxResults    int      // Maximum results to return (default: 1000, max: 50000)
	OrderBy       []string // Order by columns (e.g., ["metrics.accuracy DESC"])
	PageToken     string   // Token for pagination continuation
}

// SearchRunsResult contains the results of a search runs operation.
type SearchRunsResult struct {
	Runs          []*pb.Run // Matching runs
	NextPageToken string    // Token for next page (empty if no more pages)
}

// RunOption is a functional option for configuring run creation.
type RunOption func(*createRunRequest)

// WithRunName sets the run name.
func WithRunName(name string) RunOption {
	return func(req *createRunRequest) {
		req.RunName = name
	}
}

// WithStartTime sets the run start time (milliseconds since epoch).
func WithStartTime(t int64) RunOption {
	return func(req *createRunRequest) {
		req.StartTime = t
	}
}

// WithTags sets the initial tags for the run.
func WithTags(tags map[string]string) RunOption {
	return func(req *createRunRequest) {
		if len(tags) > 0 {
			req.Tags = make([]runTag, 0, len(tags))
			for k, v := range tags {
				req.Tags = append(req.Tags, runTag{
					Key:   k,
					Value: v,
				})
			}
		}
	}
}

// CreateRun creates a new run in the specified experiment.
func (c *Client) CreateRun(ctx context.Context, experimentID string, opts ...RunOption) (*pb.Run, error) {
	if experimentID == "" {
		return nil, &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	req := &createRunRequest{
		ExperimentID: experimentID,
	}

	// Apply options
	for _, opt := range opts {
		opt(req)
	}

	resp := createRunResponse{}

	if err := c.doJSONRequest(ctx, "POST", "/api/2.0/mlflow/runs/create", req, &resp); err != nil {
		return nil, err
	}

	return resp.Run, nil
}

// GetRun retrieves run metadata and data by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*pb.Run, error) {
	if runID == "" {
		return nil, &ValidationError{
			Field:   "runID",
			Message: "run ID cannot be empty",
			Value:   runID,
		}
	}

	req := getRunRequest{
		RunID: runID,
	}

	resp := getRunResponse{}

	if err := c.doJSONRequest(ctx, "GET", "/api/2.0/mlflow/runs/get", req, &resp); err != nil {
		return nil, err
	}

	return resp.Run, nil
}

// UpdateRun updates a run's status and optionally its end time.
// Status must be one of: "RUNNING", "SCHEDULED", "FINISHED", "FAILED", "KILLED".
// Pass endTime=0 to not update the end time.
func (c *Client) UpdateRun(ctx context.Context, runID string, status string, endTime int64) (*pb.RunInfo, error) {
	if runID == "" {
		return nil, &ValidationError{
			Field:   "runID",
			Message: "run ID cannot be empty",
			Value:   runID,
		}
	}
	if status == "" {
		return nil, &ValidationError{
			Field:   "status",
			Message: "status cannot be empty",
			Value:   status,
		}
	}

	// Validate status value
	validStatuses := map[string]bool{
		"RUNNING":   true,
		"SCHEDULED": true,
		"FINISHED":  true,
		"FAILED":    true,
		"KILLED":    true,
	}
	if !validStatuses[status] {
		return nil, &ValidationError{
			Field:   "status",
			Message: "invalid status, must be one of: RUNNING, SCHEDULED, FINISHED, FAILED, KILLED",
			Value:   status,
		}
	}

	req := updateRunRequest{
		RunID:   runID,
		Status:  status,
		EndTime: endTime,
	}

	resp := updateRunResponse{}

	if err := c.doJSONRequest(ctx, "POST", "/api/2.0/mlflow/runs/update", req, &resp); err != nil {
		return nil, err
	}

	return resp.RunInfo, nil
}

// DeleteRun marks a run as deleted.
func (c *Client) DeleteRun(ctx context.Context, runID string) error {
	if runID == "" {
		return &ValidationError{
			Field:   "runID",
			Message: "run ID cannot be empty",
			Value:   runID,
		}
	}

	req := deleteRunRequest{
		RunID: runID,
	}

	if err := c.doJSONRequest(ctx, "POST", "/api/2.0/mlflow/runs/delete", req, nil); err != nil {
		return err
	}

	return nil
}

// LogBatch logs metrics, params, and tags to a run atomically.
// Any of the slices can be nil or empty.
func (c *Client) LogBatch(ctx context.Context, runID string, metrics []pb.Metric, params []pb.Param, tags []pb.RunTag) error {
	if runID == "" {
		return &ValidationError{
			Field:   "runID",
			Message: "run ID cannot be empty",
			Value:   runID,
		}
	}

	req := logBatchRequest{
		RunID: runID,
	}

	// Convert to pointers for the request
	if len(metrics) > 0 {
		req.Metrics = make([]*pb.Metric, len(metrics))
		for i := range metrics {
			req.Metrics[i] = &metrics[i]
		}
	}

	if len(params) > 0 {
		req.Params = make([]*pb.Param, len(params))
		for i := range params {
			req.Params[i] = &params[i]
		}
	}

	if len(tags) > 0 {
		req.Tags = make([]*pb.RunTag, len(tags))
		for i := range tags {
			req.Tags[i] = &tags[i]
		}
	}

	if err := c.doJSONRequest(ctx, "POST", "/api/2.0/mlflow/runs/log-batch", req, nil); err != nil {
		return err
	}

	return nil
}

// SearchRuns searches for runs matching the given criteria.
func (c *Client) SearchRuns(ctx context.Context, opts SearchRunsOptions) (*SearchRunsResult, error) {
	if len(opts.ExperimentIDs) == 0 {
		return nil, &ValidationError{
			Field:   "ExperimentIDs",
			Message: "at least one experiment ID is required",
			Value:   opts.ExperimentIDs,
		}
	}

	req := searchRunsRequest{
		ExperimentIDs: opts.ExperimentIDs,
		Filter:        opts.Filter,
		RunViewType:   opts.RunViewType,
		MaxResults:    opts.MaxResults,
		OrderBy:       opts.OrderBy,
		PageToken:     opts.PageToken,
	}

	resp := searchRunsResponse{}

	if err := c.doJSONRequest(ctx, "POST", "/api/2.0/mlflow/runs/search", req, &resp); err != nil {
		return nil, err
	}

	return &SearchRunsResult{
		Runs:          resp.Runs,
		NextPageToken: resp.NextPageToken,
	}, nil
}

// Internal request/response types for runs API
// These match the MLflow REST API JSON structure

type createRunRequest struct {
	ExperimentID string   `json:"experiment_id"`
	RunName      string   `json:"run_name,omitempty"`
	StartTime    int64    `json:"start_time,omitempty"`
	Tags         []runTag `json:"tags,omitempty"`
}

type createRunResponse struct {
	Run *pb.Run `json:"run"`
}

type getRunRequest struct {
	RunID string `json:"run_id"`
}

type getRunResponse struct {
	Run *pb.Run `json:"run"`
}

type updateRunRequest struct {
	RunID   string `json:"run_id"`
	Status  string `json:"status"`
	EndTime int64  `json:"end_time,omitempty"`
}

type updateRunResponse struct {
	RunInfo *pb.RunInfo `json:"run_info"`
}

type deleteRunRequest struct {
	RunID string `json:"run_id"`
}

type logBatchRequest struct {
	RunID   string       `json:"run_id"`
	Metrics []*pb.Metric `json:"metrics,omitempty"`
	Params  []*pb.Param  `json:"params,omitempty"`
	Tags    []*pb.RunTag `json:"tags,omitempty"`
}

type searchRunsRequest struct {
	ExperimentIDs []string `json:"experiment_ids"`
	Filter        string   `json:"filter,omitempty"`
	RunViewType   string   `json:"run_view_type,omitempty"`
	MaxResults    int      `json:"max_results,omitempty"`
	OrderBy       []string `json:"order_by,omitempty"`
	PageToken     string   `json:"page_token,omitempty"`
}

type searchRunsResponse struct {
	Runs          []*pb.Run `json:"runs"`
	NextPageToken string    `json:"next_page_token,omitempty"`
}

type runTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
