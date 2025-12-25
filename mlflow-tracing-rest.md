# MLflow Tracing REST API Reference

This document provides a comprehensive reference for the MLflow Tracing REST API, covering all endpoints, data models, and schemas for managing traces, spans, and assessments.

## Table of Contents

1. [Overview](#overview)
2. [API Versioning](#api-versioning)
3. [Base URLs](#base-urls)
4. [Data Models](#data-models)
   - [Trace](#trace)
   - [TraceInfo](#traceinfo)
   - [TraceData](#tracedata)
   - [Span](#span)
   - [SpanStatus](#spanstatus)
   - [SpanEvent](#spanevent)
   - [TraceLocation](#tracelocation)
   - [Assessment](#assessment)
5. [REST Endpoints](#rest-endpoints)
   - [Trace Management](#trace-management)
   - [Span Management](#span-management)
   - [Tag Management](#tag-management)
   - [Assessment Management](#assessment-management)
   - [Trace-to-Run Linking](#trace-to-run-linking)
   - [Location Management](#location-management)
   - [Metrics](#metrics)
6. [OpenTelemetry Integration](#opentelemetry-integration)
7. [Error Codes](#error-codes)
8. [Examples](#examples)

---

## Overview

MLflow Tracing provides distributed tracing capabilities for machine learning workflows, enabling observability for LLM applications, agents, and complex ML pipelines. The Tracing API allows you to:

- Create and manage traces with hierarchical spans
- Attach metadata, tags, and assessments to traces
- Search and filter traces across experiments
- Integrate with OpenTelemetry protocols
- Calculate correlations between trace filters

---

## API Versioning

MLflow Tracing supports multiple API versions:

| Version | Path Prefix | Status | Description |
|---------|-------------|--------|-------------|
| V2 | `/api/2.0/mlflow/traces` | Deprecated | Legacy trace API |
| V3 | `/api/3.0/mlflow/traces` | Current | Primary tracing API |
| V4 | `/api/4.0/mlflow/traces` | Databricks | Databricks-specific extensions |

> **Note**: V3 APIs are recommended for new implementations. V2 APIs are maintained for backward compatibility.

---

## Base URLs

```
# OSS MLflow Server
https://<mlflow-server-host>/api/3.0/mlflow/traces

# Databricks Workspace
https://<workspace-url>/api/4.0/mlflow/traces
```

---

## Data Models

### Trace

A complete trace object containing both metadata and span data.

```typescript
interface Trace {
  info: TraceInfo;      // Trace metadata
  data: TraceData;      // Trace span data
}
```

**JSON Schema:**
```json
{
  "info": {
    "trace_id": "tr-abc123",
    "trace_location": {
      "type": "MLFLOW_EXPERIMENT",
      "mlflow_experiment": {
        "experiment_id": "0"
      }
    },
    "request_time": "2024-01-15T10:30:00Z",
    "execution_duration_ms": 1500,
    "state": "OK",
    "request_preview": "{\"query\": \"What is MLflow?\"}",
    "response_preview": "{\"answer\": \"MLflow is...\"}",
    "trace_metadata": {
      "mlflow.modelId": "model-123",
      "mlflow.sourceRun": "run-456"
    },
    "tags": {
      "environment": "production"
    },
    "assessments": []
  },
  "data": {
    "spans": []
  }
}
```

---

### TraceInfo

Metadata about a trace including location, timing, and state.

```typescript
interface TraceInfo {
  // Primary identifier for the trace
  trace_id: string;

  // Client-supplied request ID (optional)
  client_request_id?: string;

  // Location where trace is stored
  trace_location: TraceLocation;

  // Request/response previews (JSON strings, truncated to 10KB)
  request_preview?: string;
  response_preview?: string;

  // Timing information
  request_time: Timestamp;          // Start time (ISO 8601)
  execution_duration?: Duration;    // Duration in milliseconds

  // Trace state
  state: TraceState;

  // Key-value metadata (immutable after creation)
  trace_metadata: Record<string, string>;

  // Mutable tags
  tags: Record<string, string>;

  // Associated assessments
  assessments: Assessment[];
}
```

**TraceState Enum:**
```typescript
enum TraceState {
  STATE_UNSPECIFIED = "STATE_UNSPECIFIED",
  OK = "OK",
  ERROR = "ERROR",
  IN_PROGRESS = "IN_PROGRESS"
}
```

**Common Trace Metadata Keys:**
| Key | Description |
|-----|-------------|
| `mlflow.sourceRun` | ID of the MLflow run that produced the trace |
| `mlflow.modelId` | ID of the associated model |
| `mlflow.datasetId` | ID of the associated dataset |
| `mlflow.datasetRecordId` | ID of the dataset record |
| `mlflow.sessionId` | Session/conversation ID |
| `mlflow.tokenUsage` | Aggregated token usage (JSON) |

---

### TraceData

Container for span data within a trace.

```typescript
interface TraceData {
  spans: Span[];
}
```

---

### Span

A span represents a unit of work within a trace.

```typescript
interface Span {
  // Identifiers (base64-encoded bytes)
  trace_id: string;           // 16 bytes, base64
  span_id: string;            // 8 bytes, base64
  parent_span_id?: string;    // 8 bytes, base64 (null for root span)

  // Descriptive information
  name: string;

  // Timing (Unix nanoseconds)
  start_time_unix_nano: number;
  end_time_unix_nano?: number;

  // Status
  status: SpanStatus;

  // Attributes (JSON-encoded values)
  attributes: Record<string, any>;

  // Events that occurred during the span
  events: SpanEvent[];
}
```

**Span Attribute Keys:**
| Key | Description |
|-----|-------------|
| `mlflow.traceRequestId` | MLflow trace ID |
| `mlflow.spanType` | Type of span (LLM, CHAIN, TOOL, etc.) |
| `mlflow.spanInputs` | Input values (JSON) |
| `mlflow.spanOutputs` | Output values (JSON) |

**SpanType Constants:**
```typescript
enum SpanType {
  LLM = "LLM",
  CHAIN = "CHAIN",
  AGENT = "AGENT",
  TOOL = "TOOL",
  CHAT_MODEL = "CHAT_MODEL",
  RETRIEVER = "RETRIEVER",
  PARSER = "PARSER",
  EMBEDDING = "EMBEDDING",
  RERANKER = "RERANKER",
  MEMORY = "MEMORY",
  UNKNOWN = "UNKNOWN",
  WORKFLOW = "WORKFLOW",
  TASK = "TASK",
  GUARDRAIL = "GUARDRAIL",
  EVALUATOR = "EVALUATOR"
}
```

---

### SpanStatus

Status information for a span.

```typescript
interface SpanStatus {
  code: SpanStatusCode;
  message?: string;
}

enum SpanStatusCode {
  STATUS_CODE_UNSET = "STATUS_CODE_UNSET",
  STATUS_CODE_OK = "STATUS_CODE_OK",
  STATUS_CODE_ERROR = "STATUS_CODE_ERROR"
}
```

---

### SpanEvent

An event that occurred during a span's lifetime.

```typescript
interface SpanEvent {
  name: string;
  time_unix_nano: number;
  attributes?: Record<string, any>;
}
```

---

### TraceLocation

Specifies where a trace is stored.

```typescript
interface TraceLocation {
  type: TraceLocationType;
  mlflow_experiment?: MlflowExperimentLocation;
  inference_table?: InferenceTableLocation;    // Deprecated
  uc_schema?: UCSchemaLocation;
}

enum TraceLocationType {
  TRACE_LOCATION_TYPE_UNSPECIFIED = "TRACE_LOCATION_TYPE_UNSPECIFIED",
  MLFLOW_EXPERIMENT = "MLFLOW_EXPERIMENT",
  INFERENCE_TABLE = "INFERENCE_TABLE",   // Deprecated
  UC_SCHEMA = "UC_SCHEMA"
}

interface MlflowExperimentLocation {
  experiment_id: string;
}

interface UCSchemaLocation {
  catalog_name: string;
  schema_name: string;
  otel_spans_table_name?: string;   // Default: mlflow_experiment_trace_otel_spans
  otel_logs_table_name?: string;    // Default: mlflow_experiment_trace_otel_logs
}
```

---

### Assessment

Feedback or expectations attached to a trace.

```typescript
interface Assessment {
  assessment_id?: string;
  assessment_name: string;
  trace_id: string;
  span_id?: string;
  source: AssessmentSource;
  create_time: Timestamp;
  last_update_time: Timestamp;

  // One of:
  feedback?: Feedback;
  expectation?: Expectation;

  rationale?: string;
  metadata?: Record<string, string>;
  overrides?: string;        // ID of overridden assessment
  valid?: boolean;           // True if not superseded
}

interface AssessmentSource {
  source_type: AssessmentSourceType;
  source_id: string;
}

enum AssessmentSourceType {
  SOURCE_TYPE_UNSPECIFIED = "SOURCE_TYPE_UNSPECIFIED",
  HUMAN = "HUMAN",
  LLM_JUDGE = "LLM_JUDGE",
  CODE = "CODE"
}

interface Feedback {
  value: any;                // number, boolean, string, or list
  error?: AssessmentError;
}

interface Expectation {
  value?: any;
  serialized_value?: {
    serialization_format: string;   // e.g., "JSON_FORMAT"
    value: string;
  };
}

interface AssessmentError {
  error_code?: string;
  error_message?: string;
  stack_trace?: string;     // Truncated to 1000 chars
}
```

---

## REST Endpoints

### Trace Management

#### Create Trace (Start Trace V3)

Creates a new trace with metadata.

```http
POST /api/3.0/mlflow/traces
Content-Type: application/json
```

**Request Body:**
```json
{
  "trace": {
    "trace_info": {
      "trace_id": "tr-abc123",
      "trace_location": {
        "type": "MLFLOW_EXPERIMENT",
        "mlflow_experiment": {
          "experiment_id": "0"
        }
      },
      "request_time": "2024-01-15T10:30:00Z",
      "state": "OK",
      "trace_metadata": {},
      "tags": {}
    },
    "spans": []
  }
}
```

**Response:**
```json
{
  "trace": {
    "trace_info": {
      "trace_id": "tr-abc123",
      "trace_location": {...},
      "request_time": "2024-01-15T10:30:00Z",
      "state": "OK"
    }
  }
}
```

---

#### Get Trace

Retrieves a complete trace with spans.

```http
GET /api/3.0/mlflow/traces/get
```

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `trace_id` | string | Yes | Trace ID to fetch |
| `allow_partial` | boolean | No | Allow incomplete traces |

**Response:**
```json
{
  "trace": {
    "trace_info": {...},
    "spans": [...]
  }
}
```

---

#### Get Trace Info

Retrieves trace metadata only (without spans).

```http
GET /api/3.0/mlflow/traces/{trace_id}
```

**Path Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `trace_id` | string | Yes | Trace ID |

**Response:**
```json
{
  "trace": {
    "trace_info": {
      "trace_id": "tr-abc123",
      "trace_location": {...},
      "request_time": "2024-01-15T10:30:00Z",
      "execution_duration": {"nanos": 1500000000},
      "state": "OK",
      "trace_metadata": {},
      "tags": {},
      "assessments": []
    }
  }
}
```

---

#### Batch Get Traces

Retrieves multiple traces in a single request.

```http
GET /api/3.0/mlflow/traces/batchGet
```

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `trace_ids` | string[] | Yes | List of trace IDs (max 10) |

**Response:**
```json
{
  "traces": [
    {
      "trace_info": {...},
      "spans": [...]
    }
  ]
}
```

---

#### Search Traces

Search for traces matching specified criteria.

```http
POST /api/3.0/mlflow/traces/search
Content-Type: application/json
```

**Request Body:**
```json
{
  "locations": [
    {
      "type": "MLFLOW_EXPERIMENT",
      "mlflow_experiment": {
        "experiment_id": "0"
      }
    }
  ],
  "filter": "trace.state = 'OK' AND trace.request_time > 1704067200000",
  "max_results": 100,
  "order_by": ["request_time DESC"],
  "page_token": null
}
```

**Filter Syntax:**

| Field | Description | Operators |
|-------|-------------|-----------|
| `trace.state` | Trace state | `=`, `!=` |
| `trace.request_time` | Start timestamp (ms) | `=`, `!=`, `<`, `<=`, `>`, `>=` |
| `trace.execution_duration` | Duration (ms) | `=`, `!=`, `<`, `<=`, `>`, `>=` |
| `attribute.run_id` | Associated run ID | `=`, `!=` |
| `request_metadata.<key>` | Metadata value | `=`, `!=` |
| `tag.<key>` | Tag value | `=`, `!=` |
| `span.type` | Span type | `=`, `!=` |
| `feedback.<name>` | Assessment feedback | `=`, `!=`, `<`, `>` |

**Response:**
```json
{
  "traces": [
    {
      "trace_id": "tr-abc123",
      "trace_location": {...},
      "request_time": "2024-01-15T10:30:00Z",
      "state": "OK"
    }
  ],
  "next_page_token": "eyJ..."
}
```

---

#### Delete Traces

Delete traces by criteria or IDs.

```http
POST /api/2.0/mlflow/traces/delete-traces
Content-Type: application/json
```

**Request Body (by timestamp):**
```json
{
  "experiment_id": "0",
  "max_timestamp_millis": 1704067200000,
  "max_traces": 100
}
```

**Request Body (by IDs):**
```json
{
  "experiment_id": "0",
  "request_ids": ["tr-abc123", "tr-def456"]
}
```

**Response:**
```json
{
  "traces_deleted": 42
}
```

---

### Span Management

#### Log Spans (OTLP)

Log spans using the OpenTelemetry Protocol (OTLP).

```http
POST /v1/traces
Content-Type: application/x-protobuf
X-MLflow-Experiment-Id: <experiment_id>
```

**Request Body:** OpenTelemetry `ExportTraceServiceRequest` protobuf

**Response:** `ExportTraceServiceResponse` protobuf

---

### Tag Management

#### Set Trace Tag

Set or update a tag on a trace.

```http
PATCH /api/3.0/mlflow/traces/{trace_id}/tags
Content-Type: application/json
```

**Request Body:**
```json
{
  "key": "environment",
  "value": "production"
}
```

**Response:** Empty on success

> **Note:** Tag keys and values are limited to 250 characters.

---

#### Delete Trace Tag

Delete a tag from a trace.

```http
DELETE /api/3.0/mlflow/traces/{trace_id}/tags/{key}
```

**Path Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `trace_id` | string | Yes | Trace ID |
| `key` | string | Yes | Tag key to delete |

**Response:** Empty on success

---

### Assessment Management

#### Create Assessment

Create a new assessment on a trace.

```http
POST /api/3.0/mlflow/traces/{trace_id}/assessments
Content-Type: application/json
```

**Request Body (Feedback):**
```json
{
  "assessment": {
    "assessment_name": "correctness",
    "trace_id": "tr-abc123",
    "source": {
      "source_type": "HUMAN",
      "source_id": "user@example.com"
    },
    "feedback": {
      "value": true
    },
    "rationale": "The response correctly answered the question.",
    "metadata": {
      "reviewer_notes": "Verified against documentation"
    }
  }
}
```

**Request Body (Expectation):**
```json
{
  "assessment": {
    "assessment_name": "expected_response",
    "trace_id": "tr-abc123",
    "source": {
      "source_type": "HUMAN",
      "source_id": "labeler@example.com"
    },
    "expectation": {
      "value": "The capital of France is Paris."
    }
  }
}
```

**Response:**
```json
{
  "assessment": {
    "assessment_id": "asmt-789",
    "assessment_name": "correctness",
    "trace_id": "tr-abc123",
    "source": {...},
    "create_time": "2024-01-15T10:35:00Z",
    "last_update_time": "2024-01-15T10:35:00Z",
    "feedback": {"value": true},
    "valid": true
  }
}
```

---

#### Get Assessment

Retrieve a specific assessment.

```http
GET /api/3.0/mlflow/traces/{trace_id}/assessments/{assessment_id}
```

**Response:**
```json
{
  "assessment": {
    "assessment_id": "asmt-789",
    "assessment_name": "correctness",
    "trace_id": "tr-abc123",
    ...
  }
}
```

---

#### Update Assessment

Update an existing assessment.

```http
PATCH /api/3.0/mlflow/traces/{trace_id}/assessments/{assessment_id}
Content-Type: application/json
```

**Request Body:**
```json
{
  "assessment": {
    "assessment_id": "asmt-789",
    "trace_id": "tr-abc123",
    "feedback": {
      "value": false
    },
    "rationale": "Updated: Found an error in the response."
  },
  "update_mask": {
    "paths": ["feedback", "rationale"]
  }
}
```

**Updatable Fields:**
- `assessment_name`
- `span_id`
- `source`
- `feedback`
- `expectation`
- `rationale`
- `metadata`
- `overrides`
- `valid`

**Response:**
```json
{
  "assessment": {
    "assessment_id": "asmt-789",
    "last_update_time": "2024-01-15T11:00:00Z",
    ...
  }
}
```

---

#### Delete Assessment

Delete an assessment from a trace.

```http
DELETE /api/3.0/mlflow/traces/{trace_id}/assessments/{assessment_id}
```

**Response:** Empty on success

---

### Trace-to-Run Linking

#### Link Traces to Run

Associate traces with an MLflow run.

```http
POST /api/3.0/mlflow/traces/link-to-run
Content-Type: application/json
```

**Request Body:**
```json
{
  "trace_ids": ["tr-abc123", "tr-def456"],
  "run_id": "run-789"
}
```

**Response:** Empty on success

> **Note:** Maximum 100 traces per request.

---

### Location Management

#### Create UC Storage Location (Databricks)

Create a Unity Catalog schema for trace storage.

```http
POST /api/4.0/mlflow/traces/location
Content-Type: application/json
```

**Request Body:**
```json
{
  "uc_schema": {
    "catalog_name": "main",
    "schema_name": "tracing"
  },
  "sql_warehouse_id": "abc123def456"
}
```

**Response:**
```json
{
  "uc_schema": {
    "catalog_name": "main",
    "schema_name": "tracing",
    "otel_spans_table_name": "mlflow_experiment_trace_otel_spans",
    "otel_logs_table_name": "mlflow_experiment_trace_otel_logs"
  }
}
```

---

#### Link Experiment to UC Location (Databricks)

Link an experiment to a UC trace location.

```http
POST /api/4.0/mlflow/traces/{experiment_id}/link-location
Content-Type: application/json
```

**Request Body:**
```json
{
  "uc_schema": {
    "catalog_name": "main",
    "schema_name": "tracing"
  }
}
```

**Response:** Empty on success

---

#### Unlink Experiment from UC Location (Databricks)

Unlink an experiment from a UC trace location.

```http
POST /api/4.0/mlflow/traces/{experiment_id}/unlink-location
Content-Type: application/json
```

**Request Body:**
```json
{
  "uc_schema": {
    "catalog_name": "main",
    "schema_name": "tracing"
  }
}
```

**Response:** Empty on success

---

### Metrics

#### Query Trace Metrics

Query aggregated metrics from traces.

```http
POST /api/3.0/mlflow/traces/metrics
Content-Type: application/json
```

**Request Body:**
```json
{
  "experiment_ids": ["0", "1"],
  "view_type": "TRACE",
  "metric_name": "execution_duration",
  "aggregations": ["AVG", "P50", "P99"],
  "dimensions": ["trace.state"],
  "filters": ["trace.state = 'OK'"],
  "time_interval_seconds": 3600,
  "start_time_ms": 1704067200000,
  "end_time_ms": 1704153600000,
  "max_results": 1000
}
```

**Response:**
```json
{
  "data_points": [
    {
      "timestamp_ms": 1704067200000,
      "dimensions": {"trace.state": "OK"},
      "aggregations": {
        "AVG": 1500.5,
        "P50": 1200.0,
        "P99": 5000.0
      }
    }
  ],
  "next_page_token": null
}
```

---

#### Calculate Filter Correlation

Calculate NPMI correlation between two filter conditions.

```http
POST /api/3.0/mlflow/traces/calculate-filter-correlation
Content-Type: application/json
```

**Request Body:**
```json
{
  "experiment_ids": ["0"],
  "filter_string1": "span.type = 'LLM'",
  "filter_string2": "feedback.quality > 0.8",
  "base_filter": "trace.request_time > 1704067200000"
}
```

**Response:**
```json
{
  "npmi": 0.456,
  "npmi_smoothed": 0.423,
  "filter1_count": 150,
  "filter2_count": 80,
  "joint_count": 45,
  "total_count": 500
}
```

---

## OpenTelemetry Integration

MLflow Tracing supports the OpenTelemetry Protocol (OTLP) for span ingestion.

### OTLP Endpoint

```http
POST /v1/traces
Content-Type: application/x-protobuf
X-MLflow-Experiment-Id: <experiment_id>
```

### Supported Translators

MLflow supports translation from various OpenTelemetry instrumentation formats:

| Translator | Source |
|------------|--------|
| `traceloop` | Traceloop instrumentation |
| `vercel_ai` | Vercel AI SDK |
| `google_adk` | Google ADK |
| `open_inference` | Open Inference standard |
| `voltagent` | Voltagent framework |
| `genai_semconv` | GenAI Semantic Conventions |

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `RESOURCE_DOES_NOT_EXIST` | 404 | Trace or resource not found |
| `INVALID_PARAMETER_VALUE` | 400 | Invalid request parameters |
| `BAD_REQUEST` | 400 | Malformed request |
| `NOT_FOUND` | 404 | Resource not found |
| `ENDPOINT_NOT_FOUND` | 404 | API endpoint not available |
| `INTERNAL_ERROR` | 500 | Server-side error |

**Error Response Format:**
```json
{
  "error_code": "RESOURCE_DOES_NOT_EXIST",
  "message": "Trace with ID 'tr-abc123' not found."
}
```

---

## Examples

### Python: Create and Log a Trace

```python
import mlflow
from mlflow.entities import TraceInfo, TraceLocation, TraceState

# Configure tracing
mlflow.set_tracking_uri("http://localhost:5000")
mlflow.set_experiment("my-experiment")

# Using the fluent API
@mlflow.trace
def my_function(query: str) -> str:
    return f"Response to: {query}"

# Execute and generate trace
result = my_function("Hello, MLflow!")

# Get the trace
trace_id = mlflow.get_last_active_trace_id()
trace = mlflow.get_trace(trace_id)
print(trace.to_json(pretty=True))
```

### Python: Search Traces

```python
import mlflow

# Search for successful traces
traces = mlflow.search_traces(
    experiment_ids=["0"],
    filter_string="trace.state = 'OK'",
    max_results=100,
    order_by=["request_time DESC"]
)

for trace in traces:
    print(f"Trace: {trace.info.trace_id}")
    print(f"  Duration: {trace.info.execution_duration}ms")
    print(f"  Spans: {len(trace.data.spans)}")
```

### Python: Add Assessment

```python
import mlflow
from mlflow.entities import AssessmentSource, Feedback

# Log feedback on a trace
mlflow.log_feedback(
    trace_id="tr-abc123",
    name="correctness",
    value=True,
    source=AssessmentSource(
        source_type="HUMAN",
        source_id="reviewer@example.com"
    ),
    rationale="Response was accurate and helpful"
)
```

### cURL: Get Trace

```bash
curl -X GET \
  "http://localhost:5000/api/3.0/mlflow/traces/tr-abc123" \
  -H "Content-Type: application/json"
```

### cURL: Search Traces

```bash
curl -X POST \
  "http://localhost:5000/api/3.0/mlflow/traces/search" \
  -H "Content-Type: application/json" \
  -d '{
    "locations": [
      {
        "type": "MLFLOW_EXPERIMENT",
        "mlflow_experiment": {"experiment_id": "0"}
      }
    ],
    "filter": "trace.state = '\''OK'\''",
    "max_results": 10
  }'
```

### cURL: Create Assessment

```bash
curl -X POST \
  "http://localhost:5000/api/3.0/mlflow/traces/tr-abc123/assessments" \
  -H "Content-Type: application/json" \
  -d '{
    "assessment": {
      "assessment_name": "quality",
      "trace_id": "tr-abc123",
      "source": {
        "source_type": "CODE",
        "source_id": "auto-scorer"
      },
      "feedback": {
        "value": 0.95
      }
    }
  }'
```

---

## Databricks-Specific APIs (V4)

The V4 API provides Databricks-specific extensions for trace management with Unity Catalog integration.

### V4 Endpoints Overview

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/4.0/mlflow/traces/{location_id}/{trace_id}/info` | POST | Create trace info |
| `/api/4.0/mlflow/traces/{location_id}/batchGet` | GET | Batch get traces |
| `/api/4.0/mlflow/traces/{location}/{trace_id}/info` | GET | Get trace info |
| `/api/4.0/mlflow/traces/{location_id}/{trace_id}/tags` | PATCH | Set trace tag |
| `/api/4.0/mlflow/traces/{location_id}/{trace_id}/tags/{key}` | DELETE | Delete trace tag |
| `/api/4.0/mlflow/traces/search` | POST | Search traces |
| `/api/4.0/mlflow/traces/location` | POST | Create UC storage location |
| `/api/4.0/mlflow/traces/{experiment_id}/link-location` | POST | Link experiment to UC location |
| `/api/4.0/mlflow/traces/{experiment_id}/unlink-location` | POST | Unlink experiment from UC location |
| `/api/4.0/mlflow/traces/{location_id}/{trace_id}/assessments` | POST | Create assessment |
| `/api/4.0/mlflow/traces/{location_id}/{trace_id}/assessments/{assessment_id}` | GET | Get assessment |
| `/api/4.0/mlflow/traces/{location_id}/{trace_id}/assessments/{assessment_id}` | PATCH | Update assessment |
| `/api/4.0/mlflow/traces/{location_id}/{trace_id}/assessments/{assessment_id}` | DELETE | Delete assessment |
| `/api/4.0/mlflow/traces/{location_id}/link-to-run/batchCreate` | POST | Link traces to run |
| `/api/4.0/mlflow/traces/{location_id}/unlink-from-run/batchDelete` | DELETE | Unlink traces from run |

### V4 Trace ID Format

V4 trace IDs include location information:

```
<location_id>:<trace_uuid>

Example: main.tracing:tr-abc123def456
```

---

## Rate Limits and Quotas

| Resource | Limit |
|----------|-------|
| Traces per search request | 500 max |
| Trace IDs per batch get | 10 max |
| Traces per link-to-run | 100 max |
| Tag key length | 250 characters |
| Tag value length | 250 characters |
| Request/response preview | 10KB |
| Stack trace in assessment error | 1000 characters |

---

## Best Practices

1. **Use meaningful span names**: Choose descriptive names that reflect the operation being traced.

2. **Set appropriate span types**: Use predefined `SpanType` values for better filtering and analysis.

3. **Include context in metadata**: Add relevant context like model IDs, session IDs, and user identifiers.

4. **Use tags for mutable labels**: Tags can be updated after trace creation, unlike metadata.

5. **Paginate large searches**: Use `page_token` for efficient retrieval of large result sets.

6. **Handle partial traces**: Use `allow_partial=True` when immediate results are acceptable.

7. **Leverage assessments**: Use feedback and expectations for quality tracking and labeling.

---

## Changelog

| Version | Changes |
|---------|---------|
| V3 (3.0) | TraceInfo V3 schema, improved location support |
| V4 (4.0) | Databricks UC integration, enhanced assessment APIs |

---

## See Also

- [MLflow Tracing Documentation](https://mlflow.org/docs/latest/llms/tracing/index.html)
- [OpenTelemetry Specification](https://opentelemetry.io/docs/specs/)
- [MLflow Python API Reference](https://mlflow.org/docs/latest/python_api/index.html)
