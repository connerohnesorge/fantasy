# Proto Generation

This directory contains Protocol Buffer definitions for the MLflow integration.

## Prerequisites

Install the Buf CLI and protoc-gen-go:

```bash
task proto:deps
```

## Proto Generation Workflow

### Generate Go Code

```bash
task proto:gen
```

This generates Go code to `proto/gen/mlflow/`.

### Lint Protos

```bash
task proto:lint
```

## Directory Structure

```
proto/
├── buf.yaml           # Buf module configuration
├── buf.gen.yaml       # Code generation configuration
├── mlflow/            # Curated MLflow protos
│   ├── service.proto      # Core trace/span messages
│   ├── assessments.proto  # Assessment types
│   ├── experiments.proto  # Experiment/run messages
│   └── scorers.proto      # Scorer registration
├── opentelemetry/     # OpenTelemetry protos (from MLflow)
│   └── proto/
│       ├── common/v1/common.proto
│       ├── resource/v1/resource.proto
│       └── trace/v1/trace.proto
├── scalapb/           # ScalaPB stub for compatibility
│   └── scalapb.proto
├── google/            # Google well-known types
│   └── protobuf/
│       ├── any.proto
│       ├── descriptor.proto
│       ├── duration.proto
│       ├── field_mask.proto
│       ├── struct.proto
│       └── timestamp.proto
└── gen/               # Generated Go code (git-ignored)
    └── mlflow/
```

## Updating Protos from MLflow

When MLflow releases a new version:

1. Update the MLflow reference:
   ```bash
   cd mlflow-ref/mlflow
   git fetch --tags
   git checkout v3.9.0  # or desired version
   ```

2. Copy new protos:
   ```bash
   cp mlflow-ref/mlflow/mlflow/protos/service.proto proto/mlflow/
   cp mlflow-ref/mlflow/mlflow/protos/assessments.proto proto/mlflow/
   # ... copy other needed protos
   ```

3. Strip ScalaPB extensions if needed (the stub handles most cases).

4. Update the version comment in `buf.yaml`.

5. Regenerate and verify:
   ```bash
   task proto:gen
   go build ./...
   ```

## Troubleshooting

### "scalapb.proto not found"

Ensure the `proto/scalapb/scalapb.proto` stub file exists.

### "google/protobuf/*.proto not found"

Download Google well-known types:

```bash
cd proto/google/protobuf
curl -sLO https://raw.githubusercontent.com/protocolbuffers/protobuf/main/src/google/protobuf/descriptor.proto
# ... and other needed types
```

### Import path issues

Ensure `buf.yaml` is configured correctly with the module name and `buf.gen.yaml` has the correct `go_package_prefix`.

## Version

Currently tracking: **MLflow v3.8.0**
