# fantasy Context

## Purpose

Fantasy is a Go library for building AI agents with multi-provider, multi-model support through a unified API. Built to power [Crush](https://github.com/charmbracelet/crush) (Charm's AI coding agent), Fantasy provides:

- **Unified API**: Work with multiple AI providers (Anthropic, OpenAI, Google, Azure, Bedrock, OpenRouter) through one consistent interface
- **Tool Calling**: Function calling abstraction with automatic JSON schema generation
- **Streaming Support**: Real-time response streaming for all providers
- **Structured Outputs**: Generate validated JSON objects from language models
- **Multi-Step Agents**: Orchestrate complex agent workflows with stop conditions

**Status**: Currently in preview/alpha. Expect API changes as the library evolves.

## Tech Stack

### Language
- **Go 1.25+** (required minimum version)

### AI Provider SDKs
- **Anthropic SDK**: `charmbracelet/anthropic-sdk-go`
- **OpenAI SDK**: `openai/openai-go/v2`
- **Google Generative AI**: `google.golang.org/genai`
- **AWS SDK**: Bedrock support via `aws-sdk-go-v2`
- **Azure SDK**: OpenAI service integration

### Build Tools
- **Task**: Task runner (see `Taskfile.yaml`)
- **golangci-lint**: Code quality and linting
- **gofumpt**: Stricter formatting than standard gofmt
- **goimports**: Import organization
- **modernize**: Go code modernization

### Testing
- **Go testing**: Standard `testing` package
- **VCR**: `charm.land/x/vcr` for HTTP recording/replay

### Key Libraries
- `kaptinlin/jsonschema`: JSON schema generation for tools
- `charmbracelet/x/json`: JSON utilities
- `github.com/google/uuid`: UUID generation
- `go-viper/mapstructure/v2`: Struct mapping

## Project Conventions

### Code Style

- **Formatting**: Use `gofumpt` (stricter than standard gofmt)
  - Run: `task fmt`
- **Imports**: Organized with `goimports`
- **Naming**: Follow standard Go conventions
  - Exported identifiers: Capitalized (e.g., `Agent`, `LanguageModel`)
  - Unexported identifiers: Lowercase (e.g., `agent`, `stepExecutionResult`)
- **Comments**: Must end with periods (enforced by godot linter)
- **Linting**:
  - View issues: `task lint`
  - Auto-fix: `task lint:fix`

**Enabled Linters**: bodyclose, godot, gosec, misspell, nakedret, nilerr, noctx, nolintlint, prealloc, revive, rowserrcheck, sqlclosecheck, tparallel, unconvert, unparam, whitespace

### Architecture Patterns

- **Provider Abstraction**: All AI providers implement the `LanguageModel` interface for consistency across different backends
- **Immutable Data**: `Message` and `Content` structures treated as immutable - operations return new instances
- **Functional Options**: Use `WithX(...)` pattern for configuration
  - Example: `NewAgent(model, WithSystemPrompt("..."), WithTools(...))`
- **OpenAI Compatibility**: New providers should leverage the `openaicompat` package when possible to reduce implementation effort

**Core Components**:
- `Agent`: Orchestrates multi-step interactions with language models
- `LanguageModel`: Interface implemented by all AI providers
- `Tool`: Function calling abstraction with automatic JSON schema generation
- `Message`/`Content`: Immutable conversation representation
- `Provider`: Factory for creating language models with provider-specific configuration

### Testing Strategy

**VCR for Provider Tests** (CRITICAL):
- MUST use VCR cassettes to record/replay HTTP interactions with AI providers
- Cassettes stored in `providertests/testdata/`
- Prevents hitting real APIs during tests
- Ensures consistent, fast test runs without API costs

**Running Tests**:
- Standard: `task test` or `go test ./... -count=1`
- Always use `-count=1` to disable test caching (ensures fresh runs)

**Test Organization**:
- Unit tests: `*_test.go` files alongside source code
- Provider integration: `providertests/` directory
- Examples: `examples/` directory (not run in CI)

**Requirements**:
- All provider integration tests in `providertests/` must pass before merging
- VCR cassettes must be committed with test changes
- Tests should cover both streaming and non-streaming modes

### Git Workflow

- **Main Branch**: `main` (default branch for all PRs)
- **Versioning**: Semantic versioning managed with `svu`
- **Commits**: Conventional commit style preferred
- **CI**: GitHub Actions for build, lint, and release workflows

**Releases**:
- Use `task release` to create new version
- Requires clean git state on main branch
- Creates signed, annotated tags
- Automatically pushes to origin with tags

## Domain Context

**Built for Crush**: Fantasy was created to power Crush, Charm's AI coding agent. Feature priorities reflect coding agent use cases.

**Agent Patterns**:
- Multi-step execution with configurable stop conditions
- Tool calling with automatic response handling
- Streaming support for real-time user feedback
- Structured output generation (JSON objects with validation)
- Token usage tracking across multiple steps

**Provider Coverage**: Must support Anthropic (Claude), OpenAI (GPT), Google (Gemini/Vertex), Azure OpenAI, AWS Bedrock, and OpenRouter.

**Not Yet Supported**:
- Image generation models
- Audio models
- PDF uploads
- Provider-specific tools (e.g., Anthropic's web_search)

## Important Constraints

- **Go Version**: Must maintain compatibility with Go 1.25 and above
- **OpenAI Compatibility**: New providers should work with the `openaicompat` layer when possible to reduce implementation complexity
- **Breaking Changes**: Acceptable during preview/alpha phase - API stability not yet guaranteed
- **No Vendor Lock-in**: Unified API must work across all providers without provider-specific code in user applications
- **Streaming Support**: All provider implementations must support both streaming and non-streaming modes

## External Dependencies

### AI Provider APIs
- **Anthropic API**: Claude models (claude-3-opus, claude-3-sonnet, claude-3-haiku, etc.)
- **OpenAI API**: GPT models (gpt-4, gpt-3.5-turbo, etc.)
- **Google Generative AI**: Gemini models (gemini-pro, gemini-flash, etc.)
- **Google Vertex AI**: Vertex-hosted models (vertex-claude, vertex-gemini, etc.)
- **Azure OpenAI Service**: Azure-hosted OpenAI models
- **AWS Bedrock**: Anthropic models via Bedrock
- **OpenRouter**: Multi-provider proxy service

### Authentication Requirements
- API keys for each provider (Anthropic, OpenAI, OpenRouter)
- AWS credentials for Bedrock (IAM roles or access keys)
- Google Cloud credentials for Vertex AI (service accounts)
- Azure credentials for Azure OpenAI (API keys or managed identity)

**Note**: Fantasy is a library, not a service - no database or persistent storage required.
