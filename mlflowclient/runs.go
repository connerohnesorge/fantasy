package mlflowclient

import (
	"context"
	"time"
)

// CreateRun creates a new run in the specified experiment.
func (c *Client) CreateRun(ctx context.Context, experimentID string, opts ...CreateRunOption) (*Run, error) {
	if experimentID == "" {
		return nil, &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}

	cfg := &createRunConfig{
		startTime: time.Now().UnixMilli(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	req := map[string]any{
		"experiment_id": experimentID,
		"start_time":    cfg.startTime,
	}

	if cfg.runName != "" {
		req["run_name"] = cfg.runName
	}

	if len(cfg.tags) > 0 {
		tagList := make([]map[string]string, 0, len(cfg.tags))
		for k, v := range cfg.tags {
			tagList = append(tagList, map[string]string{"key": k, "value": v})
		}
		req["tags"] = tagList
	}

	var resp struct {
		Run *Run `json:"run"`
	}

	if err := c.doRequest(ctx, "POST", "/runs/create", apiV2, req, &resp); err != nil {
		return nil, err
	}

	return resp.Run, nil
}

// GetRun retrieves a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*Run, error) {
	if runID == "" {
		return nil, &ValidationError{Field: "runID", Message: "run ID is required"}
	}

	var resp struct {
		Run *Run `json:"run"`
	}

	path := "/runs/get?run_id=" + runID
	if err := c.doRequest(ctx, "GET", path, apiV2, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Run, nil
}

// UpdateRun updates a run's status and/or end time.
func (c *Client) UpdateRun(ctx context.Context, runID string, status RunStatus, endTime int64) (*RunInfo, error) {
	if runID == "" {
		return nil, &ValidationError{Field: "runID", Message: "run ID is required"}
	}

	req := map[string]any{
		"run_id": runID,
	}

	if status != "" {
		req["status"] = string(status)
	}

	if endTime > 0 {
		req["end_time"] = endTime
	}

	var resp struct {
		RunInfo *RunInfo `json:"run_info"`
	}

	if err := c.doRequest(ctx, "POST", "/runs/update", apiV2, req, &resp); err != nil {
		return nil, err
	}

	return resp.RunInfo, nil
}

// DeleteRun marks a run for deletion.
func (c *Client) DeleteRun(ctx context.Context, runID string) error {
	if runID == "" {
		return &ValidationError{Field: "runID", Message: "run ID is required"}
	}

	req := map[string]any{
		"run_id": runID,
	}

	return c.doRequest(ctx, "POST", "/runs/delete", apiV2, req, nil)
}

// RestoreRun restores a deleted run.
func (c *Client) RestoreRun(ctx context.Context, runID string) error {
	if runID == "" {
		return &ValidationError{Field: "runID", Message: "run ID is required"}
	}

	req := map[string]any{
		"run_id": runID,
	}

	return c.doRequest(ctx, "POST", "/runs/restore", apiV2, req, nil)
}

// SearchRuns searches for runs matching the criteria.
func (c *Client) SearchRuns(ctx context.Context, opts SearchRunsOptions) ([]*Run, string, error) {
	if len(opts.ExperimentIDs) == 0 {
		return nil, "", &ValidationError{Field: "experimentIDs", Message: "at least one experiment ID is required"}
	}

	req := map[string]any{
		"experiment_ids": opts.ExperimentIDs,
	}

	if opts.Filter != "" {
		req["filter"] = opts.Filter
	}
	if opts.RunViewType != "" {
		req["run_view_type"] = opts.RunViewType
	}
	if opts.MaxResults > 0 {
		req["max_results"] = opts.MaxResults
	}
	if len(opts.OrderBy) > 0 {
		req["order_by"] = opts.OrderBy
	}
	if opts.PageToken != "" {
		req["page_token"] = opts.PageToken
	}

	var resp struct {
		Runs          []*Run `json:"runs"`
		NextPageToken string `json:"next_page_token"`
	}

	if err := c.doRequest(ctx, "POST", "/runs/search", apiV2, req, &resp); err != nil {
		return nil, "", err
	}

	return resp.Runs, resp.NextPageToken, nil
}

// LogBatch logs metrics, params, and tags to a run atomically.
func (c *Client) LogBatch(ctx context.Context, runID string, metrics []*Metric, params []*Param, tags []*RunTag) error {
	if runID == "" {
		return &ValidationError{Field: "runID", Message: "run ID is required"}
	}

	req := map[string]any{
		"run_id": runID,
	}

	if metrics != nil {
		req["metrics"] = metrics
	} else {
		req["metrics"] = []any{}
	}

	if params != nil {
		req["params"] = params
	} else {
		req["params"] = []any{}
	}

	if tags != nil {
		req["tags"] = tags
	} else {
		req["tags"] = []any{}
	}

	return c.doRequest(ctx, "POST", "/runs/log-batch", apiV2, req, nil)
}

// LogMetric logs a single metric to a run.
func (c *Client) LogMetric(ctx context.Context, runID string, key string, value float64, timestamp int64, step int64) error {
	if runID == "" {
		return &ValidationError{Field: "runID", Message: "run ID is required"}
	}
	if key == "" {
		return &ValidationError{Field: "key", Message: "metric key is required"}
	}

	req := map[string]any{
		"run_id":    runID,
		"key":       key,
		"value":     value,
		"timestamp": timestamp,
		"step":      step,
	}

	return c.doRequest(ctx, "POST", "/runs/log-metric", apiV2, req, nil)
}

// LogParam logs a single parameter to a run.
func (c *Client) LogParam(ctx context.Context, runID string, key, value string) error {
	if runID == "" {
		return &ValidationError{Field: "runID", Message: "run ID is required"}
	}
	if key == "" {
		return &ValidationError{Field: "key", Message: "param key is required"}
	}

	req := map[string]any{
		"run_id": runID,
		"key":    key,
		"value":  value,
	}

	return c.doRequest(ctx, "POST", "/runs/log-param", apiV2, req, nil)
}

// SetRunTag sets a tag on a run.
func (c *Client) SetRunTag(ctx context.Context, runID, key, value string) error {
	if runID == "" {
		return &ValidationError{Field: "runID", Message: "run ID is required"}
	}
	if key == "" {
		return &ValidationError{Field: "key", Message: "tag key is required"}
	}

	req := map[string]any{
		"run_id": runID,
		"key":    key,
		"value":  value,
	}

	return c.doRequest(ctx, "POST", "/runs/set-tag", apiV2, req, nil)
}

// DeleteRunTag deletes a tag from a run.
func (c *Client) DeleteRunTag(ctx context.Context, runID, key string) error {
	if runID == "" {
		return &ValidationError{Field: "runID", Message: "run ID is required"}
	}
	if key == "" {
		return &ValidationError{Field: "key", Message: "tag key is required"}
	}

	req := map[string]any{
		"run_id": runID,
		"key":    key,
	}

	return c.doRequest(ctx, "POST", "/runs/delete-tag", apiV2, req, nil)
}
