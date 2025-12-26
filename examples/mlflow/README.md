# MLflow Integration Example

This example demonstrates the full MLflow integration workflow with Fantasy agents, including tracing and evaluation.

## What This Example Shows

1. **MLflow Client Setup**: Connecting to MLflow server and creating experiments
2. **Agent Tracing**: Automatic capture of distributed traces with span hierarchies
3. **Tool Integration**: Using tools with traced agent execution
4. **Evaluation Framework**: Systematic evaluation with multiple scorer types
5. **MLflow Export**: Exporting traces and evaluation results to MLflow

## Prerequisites

### 1. MLflow Server

Start a local MLflow server:

```bash
pip install mlflow
mlflow server --host 0.0.0.0 --port 5000
```

The server will be available at http://localhost:5000

### 2. OpenAI API Key

This example uses OpenAI's GPT-4o-mini model. Set your API key:

```bash
export OPENAI_API_KEY="your-api-key-here"
```

Alternatively, you can modify the code to use a different provider.

## Running the Example

```bash
cd examples/mlflow
go run main.go
```

## What Happens

### Step 1-2: MLflow Setup

The example creates an MLflow client and sets up an experiment named "fantasy-mlflow-demo".

### Step 3-5: Agent Creation

Creates a simple weather assistant agent with:
- GPT-4o-mini model
- System prompt for weather assistance
- `get_weather` tool for simulated weather data

### Step 6-7: Tracing

Wraps the agent with MLflow tracing and executes a query. The trace includes:
- Agent root span
- Step spans for reasoning
- LLM spans (inferred from step timing)
- Tool spans for weather tool calls

### Step 8-11: Evaluation

Runs an evaluation with:
- Test dataset with 2 weather queries
- 3 scorers (heuristic + LLM-as-judge)
- Parallel execution (2 workers)

### Step 12-13: Results

Displays evaluation results and exports to JSON file.

### Step 14: Viewing in MLflow

Instructions for viewing traces in the MLflow UI.

## Expected Output

```
=== Step 1: Setting up MLflow client ===
MLflow client connected to: http://localhost:5000

=== Step 2: Creating MLflow experiment ===
Created experiment: 1234567890

=== Step 3: Setting up AI provider ===
Model ready: gpt-4o-mini

=== Step 4: Creating agent tools ===
Created tool: get_weather

=== Step 5: Creating base agent ===
Base agent created

=== Step 6: Enabling MLflow tracing ===
Tracing enabled

=== Step 7: Running agent (with automatic tracing) ===
Agent response: The weather in San Francisco is currently sunny with a temperature of 72°F and 45% humidity.
Steps executed: 2
Trace automatically flushed to MLflow

=== Step 8: Setting up evaluation ===
Created dataset with 2 test cases

=== Step 9: Creating evaluation scorers ===
Created 3 scorers

=== Step 10: Running evaluation ===

=== Step 12: Evaluation Results ===
Total test cases: 2
Passed: 2
Failed: 0
Error rate: 0.00%

Test Case 1:
  Input: map[question:What's the weather in New York?]
  Status: passed
  Scores:
    city_mention_check: true
      Rationale: Output contains the expected substring
    answer_correctness: true
      Rationale: The answer correctly mentions New York and provides weather details
    response_relevance: true
      Rationale: The response is highly relevant to the weather query

Test Case 2:
  Input: map[question:Tell me about the weather in Tokyo?]
  Status: passed
  Scores:
    city_mention_check: true
      Rationale: Output contains the expected substring
    answer_correctness: true
      Rationale: The answer provides weather information for Tokyo
    response_relevance: true
      Rationale: The response directly addresses the weather question

=== Step 13: Exporting results ===
Results exported to eval_results.json

=== Step 14: Viewing results in MLflow ===
Open MLflow UI: http://localhost:5000
Navigate to experiment: 1234567890
View traces in the 'Traces' tab

Demo complete!
```

## Viewing in MLflow UI

1. Open http://localhost:5000 in your browser
2. Navigate to the "fantasy-mlflow-demo" experiment
3. Click on the "Traces" tab
4. You'll see traces for:
   - Initial agent run (Step 7)
   - Evaluation test cases (Step 10)
5. Click on a trace to view:
   - Span hierarchy
   - Timing information
   - Input/output data
   - Token usage (for LLM spans)
   - Tags and metadata

## Customizing the Example

### Use Different Model

```go
model, err := provider.LanguageModel(ctx, "gpt-4") // Use GPT-4 instead
```

### Add More Scorers

```go
scorers := []eval.Scorer{
    eval.NewExactMatch("format", "format"),
    eval.NewRegex("answer", `\d+°F`), // Check for temperature format
    eval.NewGuidelines(judgeModel, []string{
        "Response must be friendly",
        "Response must be concise",
    }),
}
```

### Change Parallelism

```go
eval.WithParallelism(5), // Run 5 test cases in parallel
```

### Add Custom Tags

```go
tracedAgent, _ := fantasy.WithTracing(baseAgent, tracing.TracingConfig{
    // ... other config ...
    Tags: map[string]string{
        "user_id":    "user-123",
        "session_id": "session-456",
        "version":    "v1.0",
    },
})
```

## Troubleshooting

### MLflow Connection Error

Ensure MLflow server is running:

```bash
mlflow server --host 0.0.0.0 --port 5000
```

### Experiment Already Exists

If the experiment already exists, you can either:
1. Delete it from the MLflow UI
2. Modify the code to use `SearchExperiments` to find it
3. Use a different experiment name

### API Rate Limiting

If you hit rate limits with the LLM-as-judge scorers:
- Reduce parallelism: `eval.WithParallelism(1)`
- Increase timeout: `eval.WithTimeout(120*time.Second)`
- Use fewer test cases

## Files Generated

- `eval_results.json`: Evaluation results in JSON format

## Further Reading

- [Tracing Documentation](../../tracing/README.md)
- [Eval Documentation](../../eval/README.md)
- [MLflow Client Documentation](../../mlflowclient/README.md)
- [MLflow Documentation](https://mlflow.org/docs/latest/index.html)

## License

Part of the [Fantasy](https://github.com/charmbracelet/fantasy) project.
