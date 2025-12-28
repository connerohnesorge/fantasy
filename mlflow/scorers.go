package mlflow

import (
	"context"
	"fmt"
	"net/http"
)

// SerializedScorer represents a scorer definition for registration.
// Scorers can be CODE-based (custom Go functions), LLM_JUDGE (LLM-as-judge),
// or HEURISTIC (rule-based scorers like exact match, regex, etc.).
type SerializedScorer struct {
	Type        string         // Scorer type: "CODE", "LLM_JUDGE", or "HEURISTIC"
	Name        string         // Scorer name (e.g., "correctness")
	Description string         // Human-readable description
	Config      map[string]any // Scorer-specific configuration (JSON-serializable)
}

// ScorerInfo contains metadata about a registered scorer.
type ScorerInfo struct {
	ScorerID    string         // Unique scorer ID
	Name        string         // Scorer name
	Version     int            // Scorer version number
	Type        string         // Scorer type: "CODE", "LLM_JUDGE", or "HEURISTIC"
	Description string         // Human-readable description
	Config      map[string]any // Scorer-specific configuration
	CreatedAt   int64          // Creation timestamp (milliseconds since epoch)
}

// RegisterScorer registers a new scorer version with the MLflow server.
// Each registration creates a new version of the scorer with the given name.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - experimentID: The experiment ID to register the scorer under
//   - name: The scorer name (e.g., "correctness")
//   - scorer: The serialized scorer definition
//
// Returns:
//   - scorerID: The unique ID of the registered scorer version
//   - error: Any error that occurred during registration
//
// Example Config by Type:
//   - CODE: {"function_name": "...", "package": "..."}
//   - LLM_JUDGE: {"model": "...", "prompt_template": "...", "temperature": 0.0}
//   - HEURISTIC: {"heuristic_type": "exact_match|contains|regex|json_match|numeric_range"}
func (c *Client) RegisterScorer(ctx context.Context, experimentID, name string, scorer SerializedScorer) (string, error) {
	// Validate inputs
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
	if scorer.Type == "" {
		return "", &ValidationError{
			Field:   "scorer.Type",
			Message: "scorer type cannot be empty",
			Value:   scorer.Type,
		}
	}

	// Build request
	req := map[string]any{
		"experiment_id": experimentID,
		"name":          name,
		"type":          scorer.Type,
		"description":   scorer.Description,
		"config":        scorer.Config,
	}

	// Make request
	var resp map[string]any
	if err := c.doJSONRequest(ctx, http.MethodPost, "/api/3.0/mlflow/scorers/register", req, &resp); err != nil {
		return "", fmt.Errorf("failed to register scorer: %w", err)
	}

	// Extract scorer ID from response
	scorerID, ok := resp["scorer_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response: missing scorer_id")
	}

	return scorerID, nil
}

// ListScorers retrieves all scorers (latest versions) for an experiment.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - experimentID: The experiment ID to list scorers for
//
// Returns:
//   - []ScorerInfo: List of scorers (latest versions only)
//   - error: Any error that occurred during retrieval
func (c *Client) ListScorers(ctx context.Context, experimentID string) ([]ScorerInfo, error) {
	// Validate inputs
	if experimentID == "" {
		return nil, &ValidationError{
			Field:   "experimentID",
			Message: "experiment ID cannot be empty",
			Value:   experimentID,
		}
	}

	// Build query parameters
	path := fmt.Sprintf("/api/3.0/mlflow/scorers/list?experiment_id=%s", experimentID)

	// Make request
	var resp struct {
		Scorers []map[string]any `json:"scorers"`
	}
	if err := c.doJSONRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to list scorers: %w", err)
	}

	// Convert to ScorerInfo structs
	scorers := make([]ScorerInfo, 0, len(resp.Scorers))
	for _, s := range resp.Scorers {
		scorer := ScorerInfo{
			ScorerID:    getStringField(s, "scorer_id"),
			Name:        getStringField(s, "name"),
			Version:     getIntField(s, "version"),
			Type:        getStringField(s, "type"),
			Description: getStringField(s, "description"),
			Config:      getMapField(s, "config"),
			CreatedAt:   getInt64Field(s, "created_at"),
		}
		scorers = append(scorers, scorer)
	}

	return scorers, nil
}

// GetScorer retrieves a specific scorer by name and version.
// If version is 0, the latest version is returned.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - experimentID: The experiment ID
//   - name: The scorer name
//   - version: The scorer version (0 for latest, positive integer for specific version)
//
// Returns:
//   - *ScorerInfo: The scorer information
//   - error: Any error that occurred during retrieval
func (c *Client) GetScorer(ctx context.Context, experimentID, name string, version int) (*ScorerInfo, error) {
	// Validate inputs
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
	if version < 0 {
		return nil, &ValidationError{
			Field:   "version",
			Message: "version must be >= 0 (0 means latest)",
			Value:   version,
		}
	}

	// Build query parameters
	path := fmt.Sprintf("/api/3.0/mlflow/scorers/get?experiment_id=%s&name=%s", experimentID, name)
	if version > 0 {
		path = fmt.Sprintf("%s&version=%d", path, version)
	}

	// Make request
	var resp map[string]any
	if err := c.doJSONRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to get scorer: %w", err)
	}

	// Convert to ScorerInfo
	scorer := &ScorerInfo{
		ScorerID:    getStringField(resp, "scorer_id"),
		Name:        getStringField(resp, "name"),
		Version:     getIntField(resp, "version"),
		Type:        getStringField(resp, "type"),
		Description: getStringField(resp, "description"),
		Config:      getMapField(resp, "config"),
		CreatedAt:   getInt64Field(resp, "created_at"),
	}

	return scorer, nil
}

// DeleteScorer deletes a specific scorer version.
// If version is 0, all versions of the scorer are deleted.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - experimentID: The experiment ID
//   - name: The scorer name
//   - version: The scorer version (0 to delete all versions, positive integer for specific version)
//
// Returns:
//   - error: Any error that occurred during deletion
func (c *Client) DeleteScorer(ctx context.Context, experimentID, name string, version int) error {
	// Validate inputs
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
	if version < 0 {
		return &ValidationError{
			Field:   "version",
			Message: "version must be >= 0 (0 deletes all versions)",
			Value:   version,
		}
	}

	// Build request
	req := map[string]any{
		"experiment_id": experimentID,
		"name":          name,
	}
	if version > 0 {
		req["version"] = version
	}

	// Make request
	if err := c.doJSONRequest(ctx, http.MethodDelete, "/api/3.0/mlflow/scorers/delete", req, nil); err != nil {
		return fmt.Errorf("failed to delete scorer: %w", err)
	}

	return nil
}

// Helper functions to safely extract fields from map[string]any

func getStringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getIntField(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

func getInt64Field(m map[string]any, key string) int64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int64:
			return n
		case int:
			return int64(n)
		case float64:
			return int64(n)
		}
	}
	return 0
}

func getMapField(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mp, ok := v.(map[string]any); ok {
			return mp
		}
	}
	return nil
}
