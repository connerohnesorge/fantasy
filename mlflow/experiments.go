package mlflow

import (
	"context"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// SearchExperimentsOptions configures experiment search parameters.
type SearchExperimentsOptions struct {
	Filter     string   // Filter string (e.g., "name LIKE 'my-%'")
	MaxResults int      // Maximum results to return (default: 1000, max: 50000)
	PageToken  string   // Token for pagination continuation
	OrderBy    []string // Order by columns (e.g., ["name ASC", "creation_time DESC"])
	ViewType   string   // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL" (default: "ACTIVE_ONLY")
}

// SearchExperimentsResult contains the results of a search experiments operation.
type SearchExperimentsResult struct {
	Experiments   []*pb.Experiment // Matching experiments
	NextPageToken string           // Token for next page (empty if no more pages)
}

// CreateExperiment creates a new experiment with the given name.
// Returns the experiment ID on success.
func (c *Client) CreateExperiment(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", &ValidationError{
			Field:   "name",
			Message: "experiment name cannot be empty",
			Value:   name,
		}
	}

	req := createExperimentRequest{
		Name: name,
	}

	resp := createExperimentResponse{}

	if err := c.doJSONRequest(ctx, "POST", "/api/2.0/mlflow/experiments/create", req, &resp); err != nil {
		return "", err
	}

	if resp.ExperimentID == "" {
		return "", &APIError{
			StatusCode: 500,
			Message:    "server returned empty experiment_id",
		}
	}

	return resp.ExperimentID, nil
}

// GetExperiment retrieves experiment metadata by ID.
func (c *Client) GetExperiment(ctx context.Context, experimentID string) (*pb.Experiment, error) {
	if experimentID == "" {
		return nil, &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	req := getExperimentRequest{
		ExperimentID: experimentID,
	}

	resp := getExperimentResponse{}

	if err := c.doJSONRequest(ctx, "GET", "/api/2.0/mlflow/experiments/get", req, &resp); err != nil {
		return nil, err
	}

	return resp.Experiment, nil
}

// SearchExperiments searches for experiments matching the given criteria.
func (c *Client) SearchExperiments(ctx context.Context, opts SearchExperimentsOptions) (*SearchExperimentsResult, error) {
	// Apply default MaxResults if not set (API requires positive integer)
	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = 1000 // Default as per MLflow API documentation
	}

	req := searchExperimentsRequest{
		Filter:     opts.Filter,
		MaxResults: maxResults,
		PageToken:  opts.PageToken,
		OrderBy:    opts.OrderBy,
		ViewType:   opts.ViewType,
	}

	resp := searchExperimentsResponse{}

	if err := c.doJSONRequest(ctx, "POST", "/api/2.0/mlflow/experiments/search", req, &resp); err != nil {
		return nil, err
	}

	return &SearchExperimentsResult{
		Experiments:   resp.Experiments,
		NextPageToken: resp.NextPageToken,
	}, nil
}

// UpdateExperiment updates an experiment's name.
func (c *Client) UpdateExperiment(ctx context.Context, experimentID, newName string) error {
	if experimentID == "" {
		return &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}
	if newName == "" {
		return &ValidationError{
			Field:   "newName",
			Message: "new experiment name cannot be empty",
			Value:   newName,
		}
	}

	req := updateExperimentRequest{
		ExperimentID: experimentID,
		NewName:      newName,
	}

	if err := c.doJSONRequest(ctx, "POST", "/api/2.0/mlflow/experiments/update", req, nil); err != nil {
		return err
	}

	return nil
}

// DeleteExperiment marks an experiment as deleted.
func (c *Client) DeleteExperiment(ctx context.Context, experimentID string) error {
	if experimentID == "" {
		return &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	req := deleteExperimentRequest{
		ExperimentID: experimentID,
	}

	if err := c.doJSONRequest(ctx, "POST", "/api/2.0/mlflow/experiments/delete", req, nil); err != nil {
		return err
	}

	return nil
}

// Internal request/response types for experiments API
// These match the MLflow REST API JSON structure

type createExperimentRequest struct {
	Name             string          `json:"name"`
	ArtifactLocation string          `json:"artifact_location,omitempty"`
	Tags             []experimentTag `json:"tags,omitempty"`
}

type createExperimentResponse struct {
	ExperimentID string `json:"experiment_id"`
}

type getExperimentRequest struct {
	ExperimentID string `json:"experiment_id"`
}

type getExperimentResponse struct {
	Experiment *pb.Experiment `json:"experiment"`
}

type searchExperimentsRequest struct {
	Filter     string   `json:"filter,omitempty"`
	MaxResults int      `json:"max_results,omitempty"`
	PageToken  string   `json:"page_token,omitempty"`
	OrderBy    []string `json:"order_by,omitempty"`
	ViewType   string   `json:"view_type,omitempty"`
}

type searchExperimentsResponse struct {
	Experiments   []*pb.Experiment `json:"experiments"`
	NextPageToken string           `json:"next_page_token,omitempty"`
}

type updateExperimentRequest struct {
	ExperimentID string `json:"experiment_id"`
	NewName      string `json:"new_name"`
}

type deleteExperimentRequest struct {
	ExperimentID string `json:"experiment_id"`
}

type experimentTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
