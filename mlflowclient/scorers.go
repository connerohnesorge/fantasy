package mlflowclient

import (
	"context"
	"encoding/json"

	pb "charm.land/fantasy/proto/gen/mlflow"
)

// SerializedScorer represents a scorer definition for registration.
type SerializedScorer struct {
	Type        string         // Scorer type: "CODE", "LLM_JUDGE", or "HEURISTIC"
	Name        string         // Scorer name (e.g., "correctness")
	Description string         // Human-readable description
	Config      map[string]any // Scorer-specific configuration (JSON-serializable)
}

// RegisterScorer registers a new scorer or creates a new version of an existing scorer.
func (c *Client) RegisterScorer(ctx context.Context, experimentID string, name string, scorer SerializedScorer) (string, error) {
	if experimentID == "" {
		return "", &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	if name == "" {
		return "", &ValidationError{
			Field:   "name",
			Message: "scorer name cannot be empty",
			Value:   name,
		}
	}

	// Serialize the scorer to JSON
	scorerJSON, err := json.Marshal(scorer)
	if err != nil {
		return "", &ValidationError{
			Field:   "scorer",
			Message: "failed to serialize scorer",
			Value:   scorer,
		}
	}

	serialized := string(scorerJSON)
	req := &pb.RegisterScorer{
		ExperimentId:     &experimentID,
		Name:             &name,
		SerializedScorer: &serialized,
	}

	resp := &pb.RegisterScorer_Response{}
	if err := c.doRequest(ctx, "POST", "/api/3.0/mlflow/scorers/register", req, resp); err != nil {
		return "", err
	}

	if resp.ScorerId == nil {
		return "", &APIError{
			StatusCode: 500,
			Message:    "server returned empty scorer ID",
		}
	}

	return *resp.ScorerId, nil
}

// ListScorers returns the latest versions of all scorers for an experiment.
func (c *Client) ListScorers(ctx context.Context, experimentID string) ([]*pb.Scorer, error) {
	if experimentID == "" {
		return nil, &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	resp := &pb.ListScorers_Response{}
	path := "/api/3.0/mlflow/scorers/list?experiment_id=" + experimentID
	if err := c.doRequest(ctx, "GET", path, nil, resp); err != nil {
		return nil, err
	}

	return resp.Scorers, nil
}

// GetScorer retrieves a scorer by name and optional version.
// If version is 0, returns the latest version.
func (c *Client) GetScorer(ctx context.Context, experimentID string, name string, version int) (*pb.Scorer, error) {
	if experimentID == "" {
		return nil, &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	if name == "" {
		return nil, &ValidationError{
			Field:   "name",
			Message: "scorer name cannot be empty",
			Value:   name,
		}
	}

	req := &pb.GetScorer{
		ExperimentId: &experimentID,
		Name:         &name,
	}

	if version > 0 {
		v := int32(version)
		req.Version = &v
	}

	resp := &pb.GetScorer_Response{}
	path := "/api/3.0/mlflow/scorers/get?experiment_id=" + experimentID + "&name=" + name
	if version > 0 {
		path += "&version=" + string(rune(version))
	}

	if err := c.doRequest(ctx, "GET", path, nil, resp); err != nil {
		return nil, err
	}

	return resp.Scorer, nil
}

// DeleteScorer deletes a scorer by name and version.
func (c *Client) DeleteScorer(ctx context.Context, experimentID string, name string, version int) error {
	if experimentID == "" {
		return &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	if name == "" {
		return &ValidationError{
			Field:   "name",
			Message: "scorer name cannot be empty",
			Value:   name,
		}
	}

	req := &pb.DeleteScorer{
		ExperimentId: &experimentID,
		Name:         &name,
	}

	if version > 0 {
		v := int32(version)
		req.Version = &v
	}

	resp := &pb.DeleteScorer_Response{}
	return c.doRequest(ctx, "DELETE", "/api/3.0/mlflow/scorers/delete", req, resp)
}
