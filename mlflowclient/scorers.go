package mlflowclient

import (
	"context"
	"fmt"
)

// RegisterScorer registers a new scorer version.
func (c *Client) RegisterScorer(ctx context.Context, experimentID, name string, scorer SerializedScorer) (*ScorerVersion, error) {
	if experimentID == "" {
		return nil, &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}
	if name == "" {
		return nil, &ValidationError{Field: "name", Message: "scorer name is required"}
	}

	req := map[string]any{
		"experiment_id": experimentID,
		"name":          name,
		"scorer":        scorer,
	}

	var resp struct {
		ScorerVersion *ScorerVersion `json:"scorer_version"`
	}

	if err := c.doRequest(ctx, "POST", "/scorers/register", apiV3, req, &resp); err != nil {
		return nil, err
	}

	return resp.ScorerVersion, nil
}

// ListScorers lists all scorers in an experiment.
func (c *Client) ListScorers(ctx context.Context, experimentID string) ([]*ScorerVersion, error) {
	if experimentID == "" {
		return nil, &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}

	var resp struct {
		Scorers []*ScorerVersion `json:"scorers"`
	}

	path := fmt.Sprintf("/scorers/list?experiment_id=%s", experimentID)
	if err := c.doRequest(ctx, "GET", path, apiV3, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Scorers, nil
}

// GetScorer retrieves a scorer by name.
// If version is 0, the latest version is returned.
func (c *Client) GetScorer(ctx context.Context, experimentID, name string, version int) (*ScorerVersion, error) {
	if experimentID == "" {
		return nil, &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}
	if name == "" {
		return nil, &ValidationError{Field: "name", Message: "scorer name is required"}
	}

	path := fmt.Sprintf("/scorers/get?experiment_id=%s&name=%s", experimentID, name)
	if version > 0 {
		path += fmt.Sprintf("&version=%d", version)
	}

	var resp struct {
		ScorerVersion *ScorerVersion `json:"scorer_version"`
	}

	if err := c.doRequest(ctx, "GET", path, apiV3, nil, &resp); err != nil {
		return nil, err
	}

	return resp.ScorerVersion, nil
}

// DeleteScorer deletes a scorer.
func (c *Client) DeleteScorer(ctx context.Context, experimentID, name string) error {
	if experimentID == "" {
		return &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}
	if name == "" {
		return &ValidationError{Field: "name", Message: "scorer name is required"}
	}

	req := map[string]any{
		"experiment_id": experimentID,
		"name":          name,
	}

	return c.doRequest(ctx, "POST", "/scorers/delete", apiV3, req, nil)
}
