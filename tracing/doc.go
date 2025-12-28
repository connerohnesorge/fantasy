// Package tracing provides MLflow-compatible distributed tracing for Fantasy agents.
//
// This package implements a tracing system that integrates with Fantasy's agent execution
// callbacks to create hierarchical spans representing agent operations. Traces are sent
// to MLflow for visualization and analysis.
//
// Core Types:
//
//   - Trace: Represents a complete agent execution with metadata and spans
//   - Span: Represents a unit of work within a trace (agent, step, LLM call, tool call)
//   - SpanEvent: Represents an event that occurred during span execution
//   - SpanStatus: Represents the completion status of a span (OK, ERROR, UNSET)
//   - TraceState: Represents the overall state of a trace (IN_PROGRESS, OK, ERROR)
//
// Configuration:
//
//   - TracingConfig: Configures tracing for an agent (MLflow client, experiment ID, etc.)
//
// Span Attributes:
//
// The package defines constants for MLflow-compatible span attributes:
//   - AttrSpanInputs: JSON-serialized span inputs
//   - AttrSpanOutputs: JSON-serialized span outputs
//   - AttrTokenUsage: Token usage information for LLM calls
//   - AttrLLMReasoning: Extended thinking/reasoning content
//   - And more...
//
// Thread Safety:
//
// All Trace and Span methods are thread-safe and can be called concurrently from
// multiple goroutines.
package tracing
