package tracing

import (
	"time"
)

// TracingConfig configures tracing for an agent.
type TracingConfig struct {
	Client       any               // *mlflow.Client (use any to avoid circular import)
	ExperimentID string            // Required: MLflow experiment ID
	AgentName    string            // Optional: defaults to "agent"
	ModelName    string            // Optional: model identifier
	SessionID    string            // Optional: auto-generated UUID v4 if empty
	Tags         map[string]string // Optional: custom tags
	FlushTimeout time.Duration     // Optional: default 10s
}
