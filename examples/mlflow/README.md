# MLflow Integration Example

This example demonstrates comprehensive MLflow integration with the Fantasy AI framework.

## What This Example Shows

1. **MLflow Client Setup**
   - Connect to MLflow tracking server
   - Create or retrieve experiments
   - Handle experiment lifecycle

2. **Tracing Integration**
   - Set up TracingCallbacks for agent execution
   - Capture agent runs as MLflow traces
   - Upload traces to MLflow automatically

3. **Evaluation Framework**
   - Define test datasets with inputs and expectations
   - Use heuristic scorers (ExactMatch, Contains)
   - Run batch evaluations with parallelism control

4. **MLflow Export**
   - Export traces for each test case
   - Create assessments linked to traces
   - Log aggregate metrics to MLflow runs

## Prerequisites

### 1. MLflow Server

Start a local MLflow server:

```bash
# Install MLflow if you haven't already
pip install mlflow

# Start the MLflow tracking server
mlflow server --host 0.0.0.0 --port 5000
```

The server will be available at `http://localhost:5000`

Alternatively, you can use Docker:

```bash
docker run -p 5000:5000 \
  ghcr.io/mlflow/mlflow:latest \
  mlflow server --host 0.0.0.0 --port 5000
```

### 2. OpenAI API Key

Set your OpenAI API key:

```bash
export OPENAI_API_KEY="your-api-key-here"
```

## Running the Example

```bash
# From the project root
go run examples/mlflow/main.go

# Or build and run
go build -o mlflow-demo examples/mlflow/main.go
./mlflow-demo
```

## Configuration

The example supports the following environment variables:

- `MLFLOW_TRACKING_URI` - MLflow server URL (default: `http://localhost:5000`)
- `OPENAI_API_KEY` - Required for OpenAI API calls

## What Happens

1. **Setup Phase**
   - Connects to MLflow server
   - Creates experiment named "fantasy-mlflow-demo"
   - Initializes OpenAI provider with gpt-4o-mini model

2. **Single Execution Demo**
   - Runs a single math question through the agent
   - Captures execution as an MLflow trace
   - Shows trace ID after successful upload

3. **Batch Evaluation**
   - Runs 5 math questions through the agent
   - Each execution creates a trace in MLflow
   - Scorers evaluate each response:
     - **Contains**: Checks if expected answer appears in response
     - **ExactMatch**: Checks for exact string equality

4. **MLflow Export**
   - Creates a run in the experiment
   - Uploads all traces
   - Creates assessments for each scored test case
   - Logs aggregate metrics (pass rates, mean scores, etc.)

## Expected Output

```
=== MLflow Setup ===
Connecting to MLflow: http://localhost:5000
Using experiment: fantasy-mlflow-demo (ID: 123456)

=== Agent Setup ===
Model: gpt-4o-mini
Agent created with tracing enabled

=== Single Agent Execution with Tracing ===
Question: What is 2+2?
Response: 4
Trace uploaded successfully! Trace ID: abc-123-def

=== Evaluation Framework ===
Dataset: math-qa-dataset (5 test cases)
Scorers: 2 configured
  - contains
  - exact_match

Running evaluation...

=== Evaluation Results ===
Total tests: 5
Duration: 12.3s

Per-Scorer Statistics:

contains:
  Pass rate:  100.0% (5/5 passed)
  Mean score: 1.00

exact_match:
  Pass rate:  80.0% (4/5 passed)
  Mean score: 0.80

[Individual test results follow...]
```

## Viewing Results in MLflow

After running the example, open your browser to:

```
http://localhost:5000/#/experiments/{experiment-id}
```

In the MLflow UI, you can:

- **View Traces**: See the full execution trace for each agent run
- **Check Assessments**: See scorer results linked to each trace
- **Review Metrics**: View aggregate statistics for the evaluation run
- **Compare Runs**: Track improvements across multiple evaluation runs

## Understanding the Code Structure

### Agent with Tracing

```go
// Create tracing config
config := tracing.TracingConfig{
    Client:       client,
    ExperimentID: experimentID,
    AgentName:    "math-qa-agent",
    ModelName:    "gpt-4o-mini",
    FlushTimeout: 30 * time.Second,
}

// Create callbacks
callbacks := tracing.NewTracingCallbacks(config)

// Wrap agent execution
callbacks.OnAgentStart(question)
result, err := agent.Generate(ctx, fantasy.AgentCall{Prompt: question})
callbacks.OnAgentFinish(ctx, result.Response.Content.Text())

// Get trace result
traceResult := callbacks.GetResult()
```

### Evaluation with MLflow Export

```go
// Create evaluator with MLflow export
evaluator := eval.NewEvaluator(
    eval.WithMLflowExport(client, experimentID),
)

// Run evaluation
results, err := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(func(ctx context.Context, inputs map[string]any) (map[string]any, any, error) {
        // Your prediction logic here
        // Return: outputs, trace, error
    }),
    eval.WithParallelism(1),
)
```

## Next Steps

1. **Modify the Dataset**: Add more test cases or change the questions
2. **Add Scorers**: Try other heuristic scorers like RegexScorer or NumericRangeScorer
3. **LLM Judge**: Implement an LLM-based scorer for more nuanced evaluation
4. **Parallel Execution**: Increase parallelism to speed up evaluation
5. **Custom Metrics**: Add custom metrics to the MLflow export

## Troubleshooting

### MLflow Connection Error

If you see connection errors, ensure:
- MLflow server is running
- Port 5000 is not blocked
- `MLFLOW_TRACKING_URI` is set correctly

### OpenAI API Errors

If you see API errors:
- Verify your API key is valid
- Check your OpenAI account has credits
- Ensure you have access to the gpt-4o-mini model

### Build Errors

If the example doesn't build:
- Ensure you're in the project root directory
- Run `go mod tidy` to update dependencies
- Check that all imports are available
