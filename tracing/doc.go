// Package tracing provides MLflow tracing integration for Fantasy agents.
//
// This package implements distributed tracing for agent executions, capturing
// span hierarchies (Agent -> Step -> LLM/Tool) and sending them to MLflow for
// observability and evaluation.
//
// # Usage
//
// Enable tracing on an agent using fantasy.WithTracing():
//
//	client := mlflowclient.NewClient("http://localhost:5000")
//	agent := fantasy.NewAgent(model,
//	    fantasy.WithTracing(tracing.TracingConfig{
//	        Client:       client,
//	        ExperimentID: "my-experiment",
//	        AgentName:    "my-agent",
//	        ModelName:    "gpt-4",
//	    }),
//	)
//
// # Span Hierarchy
//
// The tracing system creates the following span hierarchy:
//
//	Agent (root)
//	├─ Step 1
//	│  ├─ LLM (inferred from step timing)
//	│  ├─ Tool Call 1
//	│  └─ Tool Call 2
//	└─ Step 2
//	   └─ LLM (inferred from step timing)
//
// LLM spans are inferred retroactively from step timing since Fantasy's
// callback system does not expose direct LLM call hooks.
//
// # Thread Safety
//
// All tracing operations are thread-safe and support concurrent tool calls
// within a single step. Spans are matched by tool call ID rather than order.
//
// # Error Handling
//
// Tracing errors are logged but do not fail agent execution. Flush errors
// are returned in TracingResult for optional inspection.
package tracing
