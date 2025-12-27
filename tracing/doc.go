// Package tracing provides MLflow-compatible distributed tracing for Fantasy agents.
//
// The package creates hierarchical traces of agent executions, capturing:
// - Agent invocations (root span)
// - Step executions (chain spans)
// - LLM calls (inferred from step timing)
// - Tool calls (tool spans)
//
// Basic usage with Fantasy agent:
//
//	import "charm.land/fantasy/tracing"
//	import "charm.land/fantasy/mlflowclient"
//
//	client, _ := mlflowclient.New("http://localhost:5000")
//
//	config := tracing.TracingConfig{
//	    Client:       client,
//	    ExperimentID: "my-experiment",
//	    AgentName:    "my-agent",
//	}
//
//	agent := fantasy.NewAgent(model,
//	    fantasy.WithTracing(config),
//	)
//
//	result, err := agent.Run(ctx, prompt)
//	// Trace is automatically sent to MLflow
//
// Manual trace creation:
//
//	tracer := tracing.NewTracer(client, "experiment-id")
//	ctx, trace, _ := tracer.NewTrace(ctx, config)
//
//	span := trace.StartSpan(ctx, "my-operation", tracing.SpanTypeTool)
//	defer span.End()
//
//	// ... do work ...
//
//	tracer.EndTrace(ctx)  // Sends trace to MLflow
package tracing
