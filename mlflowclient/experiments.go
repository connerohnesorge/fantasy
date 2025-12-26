package mlflowclient

import (
	"context"
	"fmt"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// CreateExperiment creates a new experiment with the given name.
// Returns the ID of the newly created experiment.
func (c *Client) CreateExperiment(ctx context.Context, name string, tags map[string]string) (string, error) {
	if name == "" {
		return "", &ValidationError{
			Field:   "name",
			Message: "experiment name cannot be empty",
			Value:   name,
		}
	}

	req := &pb.CreateExperiment{
		Name: &name,
	}

	if len(tags) > 0 {
		req.Tags = make([]*pb.ExperimentTag, 0, len(tags))
		for k, v := range tags {
			key := k
			val := v
			req.Tags = append(req.Tags, &pb.ExperimentTag{
				Key:   &key,
				Value: &val,
			})
		}
	}

	resp := &pb.CreateExperiment_Response{}
	if err := c.doRequest(ctx, "POST", "/api/2.0/mlflow/experiments/create", req, resp); err != nil {
		return "", err
	}

	if resp.ExperimentId == nil {
		return "", fmt.Errorf("server returned empty experiment ID")
	}

	return *resp.ExperimentId, nil
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

	resp := &pb.GetExperiment_Response{}
	if err := c.doRequest(ctx, "GET", "/api/2.0/mlflow/experiments/get?experiment_id="+experimentID, nil, resp); err != nil {
		return nil, err
	}

	return resp.Experiment, nil
}

// SearchExperimentsOptions defines options for searching experiments.
type SearchExperimentsOptions struct {
	Filter     string   // Filter string (e.g., "name LIKE 'my-%'")
	MaxResults int      // Maximum results to return (default: 1000, max: 50000)
	PageToken  string   // Token for pagination continuation
	OrderBy    []string // Order by columns (e.g., ["name ASC", "creation_time DESC"])
	ViewType   string   // "ACTIVE_ONLY", "DELETED_ONLY", or "ALL" (default: "ACTIVE_ONLY")
}

// SearchExperimentsResult contains search results and pagination info.
type SearchExperimentsResult struct {
	Experiments   []*pb.Experiment
	NextPageToken string
}

// SearchExperiments searches for experiments matching the given criteria.
func (c *Client) SearchExperiments(ctx context.Context, opts *SearchExperimentsOptions) (*SearchExperimentsResult, error) {
	req := &pb.SearchExperiments{}

	if opts != nil {
		if opts.Filter != "" {
			req.Filter = &opts.Filter
		}
		if opts.MaxResults > 0 {
			maxResults := int64(opts.MaxResults)
			req.MaxResults = &maxResults
		}
		if opts.PageToken != "" {
			req.PageToken = &opts.PageToken
		}
		if len(opts.OrderBy) > 0 {
			req.OrderBy = opts.OrderBy
		}
		if opts.ViewType != "" {
			viewType := pb.ViewType(pb.ViewType_value[opts.ViewType])
			req.ViewType = &viewType
		}
	}

	resp := &pb.SearchExperiments_Response{}
	if err := c.doRequest(ctx, "POST", "/api/2.0/mlflow/experiments/search", req, resp); err != nil {
		return nil, err
	}

	result := &SearchExperimentsResult{
		Experiments: resp.Experiments,
	}

	if resp.NextPageToken != nil {
		result.NextPageToken = *resp.NextPageToken
	}

	return result, nil
}

// UpdateExperiment updates experiment metadata.
func (c *Client) UpdateExperiment(ctx context.Context, experimentID string, newName string) error {
	if experimentID == "" {
		return &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	req := &pb.UpdateExperiment{
		ExperimentId: &experimentID,
	}

	if newName != "" {
		req.NewName = &newName
	}

	resp := &pb.UpdateExperiment_Response{}
	return c.doRequest(ctx, "POST", "/api/2.0/mlflow/experiments/update", req, resp)
}

// DeleteExperiment marks an experiment for deletion.
func (c *Client) DeleteExperiment(ctx context.Context, experimentID string) error {
	if experimentID == "" {
		return &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	req := &pb.DeleteExperiment{
		ExperimentId: &experimentID,
	}

	resp := &pb.DeleteExperiment_Response{}
	return c.doRequest(ctx, "POST", "/api/2.0/mlflow/experiments/delete", req, resp)
}
