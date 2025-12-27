package mlflowclient

import (
	"context"
)

// CreateExperiment creates a new experiment with the given name.
func (c *Client) CreateExperiment(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", &ValidationError{Field: "name", Message: "experiment name is required"}
	}

	req := map[string]any{
		"name": name,
	}

	var resp struct {
		ExperimentID string `json:"experiment_id"`
	}

	if err := c.doRequest(ctx, "POST", "/experiments/create", apiV2, req, &resp); err != nil {
		return "", err
	}

	return resp.ExperimentID, nil
}

// CreateExperimentWithOptions creates a new experiment with optional configuration.
func (c *Client) CreateExperimentWithOptions(ctx context.Context, name string, artifactLocation string, tags map[string]string) (string, error) {
	if name == "" {
		return "", &ValidationError{Field: "name", Message: "experiment name is required"}
	}

	req := map[string]any{
		"name": name,
	}

	if artifactLocation != "" {
		req["artifact_location"] = artifactLocation
	}

	if len(tags) > 0 {
		tagList := make([]map[string]string, 0, len(tags))
		for k, v := range tags {
			tagList = append(tagList, map[string]string{"key": k, "value": v})
		}
		req["tags"] = tagList
	}

	var resp struct {
		ExperimentID string `json:"experiment_id"`
	}

	if err := c.doRequest(ctx, "POST", "/experiments/create", apiV2, req, &resp); err != nil {
		return "", err
	}

	return resp.ExperimentID, nil
}

// GetExperiment retrieves an experiment by ID.
func (c *Client) GetExperiment(ctx context.Context, experimentID string) (*Experiment, error) {
	if experimentID == "" {
		return nil, &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}

	var resp struct {
		Experiment *Experiment `json:"experiment"`
	}

	path := "/experiments/get?experiment_id=" + experimentID
	if err := c.doRequest(ctx, "GET", path, apiV2, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Experiment, nil
}

// GetExperimentByName retrieves an experiment by name.
func (c *Client) GetExperimentByName(ctx context.Context, name string) (*Experiment, error) {
	if name == "" {
		return nil, &ValidationError{Field: "name", Message: "experiment name is required"}
	}

	var resp struct {
		Experiment *Experiment `json:"experiment"`
	}

	path := "/experiments/get-by-name?experiment_name=" + name
	if err := c.doRequest(ctx, "GET", path, apiV2, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Experiment, nil
}

// SearchExperiments searches for experiments matching the criteria.
func (c *Client) SearchExperiments(ctx context.Context, opts SearchExperimentsOptions) ([]*Experiment, string, error) {
	req := make(map[string]any)

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
	if opts.ViewType != "" {
		req["view_type"] = opts.ViewType
	}

	var resp struct {
		Experiments   []*Experiment `json:"experiments"`
		NextPageToken string        `json:"next_page_token"`
	}

	if err := c.doRequest(ctx, "POST", "/experiments/search", apiV2, req, &resp); err != nil {
		return nil, "", err
	}

	return resp.Experiments, resp.NextPageToken, nil
}

// UpdateExperiment updates an experiment's name.
func (c *Client) UpdateExperiment(ctx context.Context, experimentID, newName string) error {
	if experimentID == "" {
		return &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}
	if newName == "" {
		return &ValidationError{Field: "newName", Message: "new name is required"}
	}

	req := map[string]any{
		"experiment_id": experimentID,
		"new_name":      newName,
	}

	return c.doRequest(ctx, "POST", "/experiments/update", apiV2, req, nil)
}

// DeleteExperiment marks an experiment for deletion.
func (c *Client) DeleteExperiment(ctx context.Context, experimentID string) error {
	if experimentID == "" {
		return &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}

	req := map[string]any{
		"experiment_id": experimentID,
	}

	return c.doRequest(ctx, "POST", "/experiments/delete", apiV2, req, nil)
}

// RestoreExperiment restores a deleted experiment.
func (c *Client) RestoreExperiment(ctx context.Context, experimentID string) error {
	if experimentID == "" {
		return &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}

	req := map[string]any{
		"experiment_id": experimentID,
	}

	return c.doRequest(ctx, "POST", "/experiments/restore", apiV2, req, nil)
}

// SetExperimentTag sets a tag on an experiment.
func (c *Client) SetExperimentTag(ctx context.Context, experimentID, key, value string) error {
	if experimentID == "" {
		return &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}
	if key == "" {
		return &ValidationError{Field: "key", Message: "tag key is required"}
	}

	req := map[string]any{
		"experiment_id": experimentID,
		"key":           key,
		"value":         value,
	}

	return c.doRequest(ctx, "POST", "/experiments/set-experiment-tag", apiV2, req, nil)
}
