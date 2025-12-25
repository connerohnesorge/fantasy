## ADDED Requirements

### Requirement: Tracer Type

The system SHALL provide a Tracer for creating MLflow-compatible traces and spans.

#### Scenario: Tracer creation
- GIVEN an MLflow client and experiment ID
- WHEN a Tracer is created
- THEN it is configured to create traces in the specified experiment

#### Scenario: Trace initialization
- GIVEN a Tracer instance
- WHEN `tracer.StartTrace(ctx)` is called
- THEN a new trace is created with a unique ID
- AND request time is set to current timestamp
- AND state is set to IN_PROGRESS

#### Scenario: Trace completion
- GIVEN an active trace
- WHEN `tracer.EndTrace(ctx)` is called
- THEN execution duration is calculated
- AND state is set to OK or ERROR based on context
- AND the trace is sent to MLflow server

### Requirement: Span Hierarchy

The system SHALL create hierarchical spans matching agent execution structure.

#### Scenario: Agent span
- GIVEN an agent execution starts
- WHEN the root span is created
- THEN span type is set to AGENT
- AND span name reflects the agent identity
- AND span becomes parent for all child spans

#### Scenario: Step span
- GIVEN an agent step executes
- WHEN a step span is created under the agent span
- THEN span type is set to CHAIN
- AND span name includes step number
- AND parent_span_id references the agent span

#### Scenario: LLM span
- GIVEN an LLM API call is made during a step
- WHEN an LLM span is created
- THEN span type is set to LLM
- AND span name includes the model name
- AND parent_span_id references the step span

#### Scenario: Tool span
- GIVEN a tool is executed during a step
- WHEN a tool span is created
- THEN span type is set to TOOL
- AND span name is the tool name
- AND parent_span_id references the step span

### Requirement: Span Attributes

The system SHALL capture comprehensive span attributes.

#### Scenario: Input attributes
- GIVEN a span with inputs
- WHEN the span is created
- THEN `mlflow.spanInputs` attribute contains JSON-serialized inputs
- AND inputs are truncated if they exceed 10,240 bytes

#### Scenario: Output attributes
- GIVEN a span completing with outputs
- WHEN the span is ended
- THEN `mlflow.spanOutputs` attribute contains JSON-serialized outputs
- AND outputs are truncated if they exceed 10,240 bytes

#### Scenario: Token usage attributes
- GIVEN an LLM span with usage information
- WHEN the span is ended
- THEN `mlflow.tokenUsage` attribute contains token counts
- AND includes input_tokens, output_tokens, and total_tokens

#### Scenario: Error attributes
- GIVEN a span that errors
- WHEN the span is ended with error
- THEN span status is set to STATUS_CODE_ERROR
- AND status message contains error description

### Requirement: Agent Callback Integration

The system SHALL integrate with Fantasy's existing agent callbacks.

#### Scenario: WithTracing option
- GIVEN an agent configuration
- WHEN `WithTracing(config)` option is applied
- THEN tracing callbacks are registered automatically
- AND no changes to agent code are required

#### Scenario: OnAgentStart callback
- GIVEN tracing is enabled
- WHEN agent execution starts (OnAgentStart fires)
- THEN a new trace and agent span are created

#### Scenario: OnStepStart callback
- GIVEN tracing is enabled and agent is running
- WHEN a step starts (OnStepStart fires)
- THEN a step span is created as child of agent span
- AND step number is captured

#### Scenario: OnStepFinish callback
- GIVEN a step span is active
- WHEN step completes (OnStepFinish fires)
- THEN step span is ended
- AND step result content is captured in outputs

#### Scenario: OnToolCall callback
- GIVEN a step span is active
- WHEN a tool is called (OnToolCall fires)
- THEN a tool span is created as child of step span
- AND tool name and input are captured

#### Scenario: OnToolResult callback
- GIVEN a tool span is active
- WHEN tool execution completes (OnToolResult fires)
- THEN tool span is ended
- AND tool result is captured in outputs

#### Scenario: OnAgentFinish callback
- GIVEN tracing is enabled and agent completes
- WHEN agent finishes (OnAgentFinish fires)
- THEN agent span is ended
- AND trace is finalized and sent to MLflow

### Requirement: Content Capture

The system SHALL capture full message content in traces.

#### Scenario: User message capture
- GIVEN a user prompt in the conversation
- WHEN the trace is created
- THEN request_preview contains the user prompt
- AND full prompt is in the agent span inputs

#### Scenario: Assistant message capture
- GIVEN assistant response content
- WHEN the step span is ended
- THEN response_preview contains the final response
- AND full response is in the step span outputs

#### Scenario: Reasoning content capture
- GIVEN reasoning tokens in the response
- WHEN reasoning content is streamed
- THEN reasoning text is captured in span attributes
- AND includes reasoning metadata

#### Scenario: Tool call content capture
- GIVEN a tool call in the response
- WHEN the tool span is created
- THEN tool name and JSON input are in span inputs
- AND tool result is in span outputs

### Requirement: Trace Metadata

The system SHALL attach relevant metadata to traces.

#### Scenario: Model metadata
- GIVEN the agent uses a specific model
- WHEN the trace is created
- THEN trace_metadata includes the model identifier

#### Scenario: Session metadata
- GIVEN a session ID is provided
- WHEN the trace is created
- THEN trace_metadata includes `mlflow.sessionId`

#### Scenario: Custom tags
- GIVEN custom tags in tracing config
- WHEN the trace is created
- THEN tags are attached to the trace

### Requirement: Trace Flushing

The system SHALL ensure traces are reliably sent to MLflow.

#### Scenario: Automatic flush on completion
- GIVEN an agent execution completes
- WHEN the trace is finalized
- THEN the trace is sent to MLflow server
- AND any pending spans are included

#### Scenario: Error handling on flush
- GIVEN an MLflow API error during flush
- WHEN the trace cannot be sent
- THEN the error is logged
- AND agent execution is not blocked

#### Scenario: Partial trace on error
- GIVEN an agent execution errors mid-way
- WHEN the error is handled
- THEN completed spans are still sent
- AND trace state is set to ERROR
