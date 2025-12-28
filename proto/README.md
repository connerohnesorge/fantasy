# Proto Infrastructure

This directory contains Protocol Buffer definitions for MLflow integration with Fantasy.

## Directory Structure

```
proto/
├── buf.yaml                    # Buf configuration
├── buf.gen.yaml                # Code generation config
├── mlflow/                     # Curated MLflow protos
│   ├── service.proto           # Core trace/assessment messages
│   ├── assessments.proto       # Assessment types
│   ├── experiments.proto       # Experiment/run messages
│   └── scorers.proto           # Scorer registration messages
├── opentelemetry/proto/        # OTel protos (downloaded)
│   ├── trace/v1/trace.proto
│   ├── common/v1/common.proto
│   └── resource/v1/resource.proto
├── google/protobuf/            # Google well-known types
│   ├── timestamp.proto
│   ├── duration.proto
│   ├── struct.proto
│   ├── any.proto
│   ├── field_mask.proto
│   └── descriptor.proto
├── scalapb/                    # Stub for ScalaPB options
│   └── scalapb.proto
└── gen/                        # Generated code (git-ignored)
    └── mlflow/
        ├── service.pb.go
        ├── assessments.pb.go
        ├── experiments.pb.go
        └── scorers.pb.go
```

## Proto Generation Workflow

### Prerequisites

1. Install buf:
   ```bash
   task proto:deps
   ```

### Generate Go Code

```bash
task proto:gen
```

This runs `buf generate` which:
- Compiles all `.proto` files in `proto/mlflow/`
- Generates Go code to `proto/gen/` using `protoc-gen-go`
- Uses `paths=source_relative` to maintain directory structure

### Lint Protos

```bash
task proto:lint
```

This runs `buf lint` which checks:
- Default style rules (naming, package conventions)
- File breaking changes (when comparing versions)

## Updating Protos from MLflow

MLflow protos are curated from the official MLflow repository (v3.8.0). To update:

### Step 1: Clone MLflow (if not already done)

```bash
git clone --depth 1 --branch v3.8.0 https://github.com/mlflow/mlflow.git mlflow-ref/mlflow
```

### Step 2: Copy and Curate Protos

```bash
# Copy relevant proto files from mlflow-ref/mlflow/mlflow/protos/
# to proto/mlflow/

# Example files to copy:
# - service.proto (core trace messages)
# - assessments.proto (assessment types)
# - experiments.proto (experiment/run types)
# - scorers.proto (scorer registration)
```

### Step 3: Strip ScalaPB Extensions

MLflow protos include ScalaPB-specific extensions that need to be removed or stubbed:

1. **Option 1: Remove extensions** (recommended)
   ```bash
   # Remove lines like:
   # import "scalapb/scalapb.proto";
   # option (scalapb.options) = {flat_package: true};
   # option (scalapb.message).extends = "...";
   ```

2. **Option 2: Keep imports** (they resolve to our stub)
   - Leave `import "scalapb/scalapb.proto";` lines
   - Our stub at `proto/scalapb/scalapb.proto` provides empty extension definitions

### Step 4: Remove Databricks-Specific Options

Remove any Databricks-specific options:
```protobuf
// Remove lines like:
option (scalapb.message).extends = "com.databricks.rpc.RPC[$this.Response]";
```

### Step 5: Update Version Comment

Update the version comment in `buf.yaml`:
```yaml
version: v2
# MLflow v3.8.0  <-- Update this line
```

### Step 6: Regenerate and Test

```bash
# Lint the updated protos
task proto:lint

# Generate Go code
task proto:gen

# Verify compilation
go build ./proto/gen/...
```

## Troubleshooting

### Error: "import not found"

If you see errors like `import "scalapb/scalapb.proto" not found`:
- Ensure `proto/scalapb/scalapb.proto` exists
- Check that the import path matches the directory structure

### Error: "unknown field option"

If you see errors about unknown field options:
- You may have missed removing some ScalaPB or Databricks-specific options
- Search for `scalapb` or `databricks` in the proto files and remove those lines

### Error: "buf generate failed"

If `buf generate` fails:
1. Check that buf is installed: `buf --version`
2. Verify `buf.yaml` syntax: `buf lint`
3. Ensure all imports are resolvable
4. Check `buf.gen.yaml` plugin configuration

### Generated Code Not Found

If you can't import generated code:
1. Ensure `task proto:gen` completed successfully
2. Check that `proto/gen/` exists and contains `.pb.go` files
3. Verify the Go module path matches your imports

### Buf Lint Warnings

Common lint warnings and fixes:
- **Package naming**: Use lowercase with dots (e.g., `package mlflow.service;`)
- **File naming**: Use `snake_case.proto` (e.g., `service.proto`)
- **Message naming**: Use `PascalCase` (e.g., `TraceInfo`)

## Versioning

This proto infrastructure tracks MLflow v3.8.0. When upgrading:

1. Update the version comment in `buf.yaml`
2. Follow the "Updating Protos from MLflow" workflow above
3. Run `buf breaking` to check for breaking changes
4. Update the Fantasy codebase to handle any breaking changes
5. Update this README with any new proto files or changes

## Dependencies

- **MLflow**: v3.8.0 (proto definitions source)
- **OpenTelemetry**: v1.0.0 (trace span types)
- **Google Protobuf**: main branch (well-known types)
- **Buf**: latest (proto tooling)

## See Also

- [MLflow Proto Source](https://github.com/mlflow/mlflow/tree/v3.8.0/mlflow/protos)
- [Buf Documentation](https://buf.build/docs)
- [Protocol Buffers](https://protobuf.dev/)
- [OpenTelemetry Proto](https://github.com/open-telemetry/opentelemetry-proto)
