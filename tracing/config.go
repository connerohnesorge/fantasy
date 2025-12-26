package tracing

import (
	"time"

	"charm.land/fantasy/mlflowclient"
)

// TracingConfig contains configuration for MLflow tracing integration.
type TracingConfig struct {
	// Client is the MLflow client for trace export (required).
	Client *mlflowclient.Client

	// ExperimentID is the target MLflow experiment (required).
	ExperimentID string

	// AgentName is an optional agent name (default: "agent").
	AgentName string

	// ModelName is an optional model name.
	ModelName string

	// SessionID is an optional session ID (auto-generated UUID v4 if empty).
	SessionID string

	// Tags are optional custom tags to attach to traces.
	Tags map[string]string

	// FlushTimeout is the timeout for flushing traces to MLflow (default: 10s).
	FlushTimeout time.Duration
}

// Validate checks if the tracing configuration is valid.
func (c *TracingConfig) Validate() error {
	if c.Client == nil {
		return &ValidationError{Field: "client", Message: "MLflow client is required"}
	}
	if c.ExperimentID == "" {
		return &ValidationError{Field: "experimentID", Message: "experiment ID is required"}
	}
	return nil
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return "tracing config validation error: " + e.Field + ": " + e.Message
}
