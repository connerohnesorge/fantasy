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
- AND state is set based on AgentResult error status (see State Determination below)
- AND the trace is sent to MLflow server

#### Scenario: Trace state determination
- GIVEN an agent execution completes
- WHEN the final AgentResult is available
- THEN state is set to ERROR if AgentResult.Error is non-nil
- AND state is set to OK if AgentResult.Error is nil
- AND state is set to ERROR if context was cancelled or timed out

### Requirement: Span Hierarchy

The system SHALL create hierarchical spans matching agent execution structure.

The span hierarchy follows: Agent -> Step -> (LLM | Tool)

Note: LLM spans are inferred from step timing since Fantasy callbacks do not provide
direct LLM call/result hooks. This approximation captures the primary LLM interaction
per step but may not reflect retry attempts or multi-model scenarios.

#### Scenario: Agent span
- GIVEN an agent execution starts via the tracing wrapper
- WHEN the root span is created
- THEN span type is set to AGENT
- AND span name is set to "agent" (or custom name from TracingConfig.AgentName)
- AND span becomes parent for all child spans

#### Scenario: Step span
- GIVEN an agent step executes
- WHEN a step span is created under the agent span
- THEN span type is set to CHAIN
- AND span name includes step number (e.g., "step-1")
- AND parent_span_id references the agent span

#### Scenario: LLM span (inferred)
- GIVEN a step span is active
- WHEN the step completes (OnStepFinish fires)
- THEN an LLM span is created retroactively as child of step span
- AND span type is set to LLM
- AND span name includes the model name from TracingConfig.ModelName
- AND span timing is derived from step start to step finish (approximation)
- AND token usage is captured from step result if available

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

The system SHALL integrate with Fantasy's existing agent callbacks via a tracing wrapper.

#### Scenario: WithTracing option
- GIVEN an agent configuration
- WHEN `fantasy.WithTracing(config)` option is applied
- THEN tracing is enabled via a call wrapper pattern
- AND the wrapper intercepts Run/Stream calls to capture context
- AND no changes to Fantasy's core API are required

#### Scenario: TracingConfig structure
- GIVEN a tracing configuration
- WHEN TracingConfig is created
- THEN it contains: Client (*mlflowclient.Client), ExperimentID (string)
- AND optional: AgentName (string), ModelName (string), SessionID (string), Tags (map[string]string)
- AND SessionID defaults to a new UUID if not provided

#### Scenario: Tracing wrapper initialization
- GIVEN tracing is enabled via WithTracing(config)
- WHEN agent.Run() or agent.Stream() is called
- THEN the wrapper intercepts the call before delegating to the agent
- AND creates a new trace with context from TracingConfig
- AND captures the initial prompt from the call parameters
- AND creates the root agent span

#### Scenario: OnStepStart callback
- GIVEN tracing is enabled and agent is running
- WHEN a step starts (OnStepStart fires)
- THEN a step span is created as child of agent span
- AND step number is captured
- AND step start timestamp is recorded for LLM span inference

#### Scenario: OnStepFinish callback
- GIVEN a step span is active
- WHEN step completes (OnStepFinish fires)
- THEN an inferred LLM span is created with step timing
- AND step span is ended
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

#### Scenario: Tracing wrapper completion
- GIVEN tracing is enabled and agent completes
- WHEN the wrapped Run/Stream call returns
- THEN the wrapper captures the AgentResult
- AND agent span is ended with appropriate status
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
- GIVEN a TracingConfig with optional SessionID
- WHEN the trace is created
- THEN if SessionID was provided in config, trace_metadata includes `mlflow.trace.session` with that value
- AND if SessionID was not provided, a new UUID v4 is generated and used

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
