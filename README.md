# Fantasy

<p>
  <img width="475" alt="The Charm Fantasy logo" src="https://github.com/user-attachments/assets/b22c5862-792a-44c1-bc98-55a2e46c8fb9" /><br>
  <a href="https://github.com/charmbracelet/fantasy/releases"><img src="https://img.shields.io/github/release/charmbracelet/fantasy.svg" alt="Latest Release"></a>
  <a href="https://pkg.go.dev/charm.land/fantasy?tab=doc"><img src="https://godoc.org/charm.land/fantasy?status.svg" alt="GoDoc"></a>
  <a href="https://github.com/charmbracelet/fantasy/actions"><img src="https://github.com/charmbracelet/fantasy/actions/workflows/build.yml/badge.svg?branch=main" alt="Build Status"></a>
</p>

Build AI agents with Go. Multi-provider, multi-model, one API.

1. Choose a model and provider
2. Add some tools
3. Compile to native machine code and let it rip

> [!NOTE]
> Fantasy is currently a preview. Expect API changes.

```go
import "charm.land/fantasy"
import "charm.land/fantasy/providers/openrouter"

// Choose your fave provider.
provider, err := openrouter.New(openrouter.WithAPIKey(myHotKey))
if err != nil {
	fmt.Fprintln(os.Stderr, "Whoops:", err)
	os.Exit(1)
}

ctx := context.Background()

// Pick your fave model.
model, err := provider.LanguageModel(ctx, "moonshotai/kimi-k2")
if err != nil {
	fmt.Fprintln(os.Stderr, "Dang:", err)
	os.Exit(1)
}

// Make your own tools.
cuteDogTool := fantasy.NewAgentTool(
  "cute_dog_tool",
  "Provide up-to-date info on cute dogs.",
  fetchCuteDogInfoFunc,
)

// Equip your agent.
agent := fantasy.NewAgent(
  model,
  fantasy.WithSystemPrompt("You are a moderately helpful, dog-centric assistant."),
  fantasy.WithTools(cuteDogTool),
)

// Put that agent to work!
const prompt = "Find all the cute dogs in Silver Lake, Los Angeles."
result, err := agent.Generate(ctx, fantasy.AgentCall{Prompt: prompt})
if err != nil {
    fmt.Fprintln(os.Stderr, "Oof:", err)
    os.Exit(1)
}
fmt.Println(result.Response.Content.Text())
```

🍔 For the full implementation and more [see the examples directory](https://github.com/charmbracelet/fantasy/tree/main/examples).

## Multi-model? Multi-provider?

Yeah! Fantasy is designed to support a wide variety of providers and models under a single API. While many providers such as Microsoft Azure, Amazon Bedrock, and OpenRouter have dedicated packages in Fantasy, many others work just fine with `openaicompat`, the generic OpenAI-compatible layer. That said, if you find a provider that's not compatible and needs special treatment, please let us know in an issue (or open a PR).

## MLflow Integration

Fantasy includes built-in support for MLflow, enabling observability and evaluation of your AI agents.

### Tracing

Capture distributed traces of agent execution with automatic span hierarchies (Agent → Step → LLM/Tool):

```go
import (
    "charm.land/fantasy"
    "charm.land/fantasy/mlflowclient"
    "charm.land/fantasy/tracing"
)

// Create MLflow client
mlflowClient := mlflowclient.New("http://localhost:5000")

// Create experiment
expID, _ := mlflowClient.CreateExperiment(ctx, "my-agent-traces", nil)

// Wrap agent with tracing
baseAgent := fantasy.NewAgent(model, fantasy.WithTools(tools...))
agent, _ := fantasy.WithTracing(baseAgent, tracing.TracingConfig{
    Client:       mlflowClient,
    ExperimentID: expID,
    AgentName:    "my-agent",
    ModelName:    "gpt-4",
})

// Use normally - traces are automatically captured!
result, _ := agent.Generate(ctx, fantasy.AgentCall{Prompt: "Hello"})
```

Traces include:
- Full span hierarchy with timing information
- LLM token usage and model metadata
- Tool invocations with inputs/outputs
- Custom tags for filtering and organization

See the [tracing documentation](tracing/README.md) for more details.

### Evaluation

Systematically evaluate agent quality with built-in scorers and optional MLflow export:

```go
import "charm.land/fantasy/eval"

// Define test dataset
dataset := &eval.Dataset{
    Name: "qa-test",
    TestCases: []eval.TestCase{
        {
            Inputs:       map[string]any{"question": "What is 2+2?"},
            Expectations: map[string]any{"answer": "4"},
        },
    },
}

// Create scorers
scorers := []eval.Scorer{
    eval.NewExactMatch("answer", "answer"),           // Heuristic
    eval.NewCorrectness(judgeModel),                  // LLM-as-judge
    eval.NewToolCallTrajectory([]string{"search"}),   // Agent-specific
}

// Run evaluation
evaluator := eval.NewEvaluator()
results, _ := evaluator.Run(ctx, dataset, scorers,
    eval.WithPredict(myPredictFunc),
    eval.WithParallelism(10),
    eval.WithMLflowExport(mlflowClient, expID),
)

fmt.Printf("Passed: %d/%d\n", results.PassedCases, results.TotalCases)
```

Built-in scorers include:
- **Heuristic**: ExactMatch, Contains, Regex, JSONMatch, NumericRange
- **LLM-as-judge**: Correctness, Guidelines, Relevance, Groundedness
- **Agent-specific**: ToolCallTrajectory, StepValidation

See the [eval documentation](eval/README.md) for more details.

### MLflow Client

Direct access to MLflow REST API with type-safe protobuf definitions:

```go
import "charm.land/fantasy/mlflowclient"

client := mlflowclient.New("http://localhost:5000",
    mlflowclient.WithToken("your-token"),
)

// Experiment management
expID, _ := client.CreateExperiment(ctx, "my-experiment", tags)
exp, _ := client.GetExperiment(ctx, expID)

// Run tracking
run, _ := client.CreateRun(ctx, expID, mlflowclient.WithRunName("run-1"))
_ = client.LogBatch(ctx, run.GetRunId(),
    mlflowclient.WithMetrics(metrics),
    mlflowclient.WithParams(params),
)

// Trace management
trace, _ := client.GetTrace(ctx, traceID)
traces, _ := client.SearchTraces(ctx, expID, "tags.agent = 'my-agent'", 100)
```

See the [mlflowclient documentation](mlflowclient/README.md) for more details.

## Work in Progress

We built Fantasy to power [Crush](https://github.com/charmbracelet/crush), a hot coding agent for glamourously invincible development. Given that, Fantasy does not yet support things like:

- Image models
- Audio models
- PDF uploads
- Provider tools (e.g. web_search)

For things you’d like to see supported, PRs are welcome.

## Whatcha think?

We’d love to hear your thoughts on this project. Need help? We gotchu. You can find us on:

- [Slack](https://charm.land/slack)
- [Discord][discord]
- [Twitter](https://twitter.com/charmcli)
- [The Fediverse](https://mastodon.social/@charmcli)
- [Bluesky](https://bsky.app/profile/charm.land)

[discord]: https://charm.land/discord

---

Part of [Charm](https://charm.land).

<a href="https://charm.land/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-banner-next.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source
