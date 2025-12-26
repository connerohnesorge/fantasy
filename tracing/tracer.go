package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	"unicode/utf8"

	"charm.land/fantasy/mlflowclient"
	pb "charm.land/fantasy/proto/gen/mlflow"
	commonpb "charm.land/fantasy/proto/gen/opentelemetry/proto/common/v1"
	otelpb "charm.land/fantasy/proto/gen/opentelemetry/proto/trace/v1"
)

// Span type constants.
const (
	SpanTypeAgent     = "AGENT"
	SpanTypeLLM       = "LLM"
	SpanTypeTool      = "TOOL"
	SpanTypeChain     = "CHAIN"
	SpanTypeRetriever = "RETRIEVER"
	SpanTypeEmbedding = "EMBEDDING"
	SpanTypeUnknown   = "UNKNOWN"
)

// Span attribute key constants.
const (
	AttrSpanInputs     = "mlflow.spanInputs"
	AttrSpanOutputs    = "mlflow.spanOutputs"
	AttrSpanType       = "mlflow.spanType"
	AttrRequestID      = "mlflow.traceRequestId"
	AttrExperimentID   = "mlflow.experimentId"
	AttrTokenUsage     = "mlflow.chat.tokenUsage"
	AttrChatTools      = "mlflow.chat.tools"
	AttrMessageFormat  = "mlflow.message.format"
	AttrLLMReasoning   = "mlflow.llm.reasoning"
	AttrFunctionName   = "mlflow.spanFunctionName"
	AttrLinkedPrompts  = "mlflow.linkedPrompts"
)

// Maximum sizes for span attributes (MLflow limits).
const (
	MaxSpanAttributeSize = 10240 // 10,240 bytes for inputs/outputs
	MaxPreviewSize       = 1000  // 1,000 characters for previews
	OrphanSpanTimeout    = 5 * time.Minute
)

// Default configuration values.
const (
	DefaultFlushTimeout = 10 * time.Second
	DefaultAgentName    = "agent"
)

// SpanStatusCode represents the status of a span.
type SpanStatusCode int

const (
	SpanStatusUnset SpanStatusCode = iota
	SpanStatusOK
	SpanStatusError
)

// SpanStatus represents the completion status of a span.
type SpanStatus struct {
	Code        SpanStatusCode
	Description string
}

// SpanEvent represents an event that occurred during span execution.
type SpanEvent struct {
	Name       string
	Timestamp  int64
	Attributes map[string]any
}

// Span represents a unit of work within a trace.
type Span struct {
	TraceID     string
	SpanID      string
	ParentID    string
	Name        string
	SpanType    string
	StartTimeNs int64
	EndTimeNs   int64
	Status      SpanStatus
	Attributes  map[string]any
	Events      []SpanEvent
	mu          sync.Mutex
}

// End ends the span with the given status.
func (s *Span) End(status SpanStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.EndTimeNs != 0 {
		log.Printf("[DEBUG] tracing: duplicate EndSpan call for span %s", s.SpanID)
		return
	}

	s.EndTimeNs = time.Now().UnixNano()
	s.Status = status
}

// SetAttribute sets an attribute on the span with truncation if needed.
func (s *Span) SetAttribute(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Attributes == nil {
		s.Attributes = make(map[string]any)
	}

	// Truncate large string attributes
	if str, ok := value.(string); ok {
		if len(str) > MaxSpanAttributeSize {
			value = truncateUTF8(str, MaxSpanAttributeSize-3) + "..."
		}
	}

	s.Attributes[key] = value
}

// SetJSONAttribute serializes value to JSON and sets it as an attribute.
func (s *Span) SetJSONAttribute(key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("[WARN] tracing: failed to serialize attribute %s: %v", key, err)
		s.SetAttribute(key, "[serialization error]")
		return
	}

	jsonStr := string(data)
	if len(jsonStr) > MaxSpanAttributeSize {
		jsonStr = truncateUTF8(jsonStr, MaxSpanAttributeSize-3) + "..."
	}

	s.SetAttribute(key, jsonStr)
}

// AddEvent adds an event to the span.
func (s *Span) AddEvent(name string, attrs map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now().UnixNano(),
		Attributes: attrs,
	})
}

// TraceState represents the state of a trace.
type TraceState int

const (
	TraceStateInProgress TraceState = iota
	TraceStateOK
	TraceStateError
)

// Trace represents a complete trace with metadata and spans.
type Trace struct {
	TraceID           string
	ExperimentID      string
	RequestTime       int64
	ExecutionDuration int64
	State             TraceState
	RequestPreview    string
	ResponsePreview   string
	TraceMetadata     map[string]string
	Tags              map[string]string
	Spans             []*Span
	mu                sync.RWMutex
}

// AddSpan adds a span to the trace.
func (t *Trace) AddSpan(span *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Spans = append(t.Spans, span)
}

// GetSpan retrieves a span by ID.
func (t *Trace) GetSpan(spanID string) *Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, span := range t.Spans {
		if span.SpanID == spanID {
			return span
		}
	}
	return nil
}

// TracingResult contains the result of a traced agent execution.
type TracingResult struct {
	Trace      *Trace
	FlushError error
}

// Tracer manages trace and span creation for MLflow integration.
type Tracer struct {
	client       *mlflowclient.Client
	experimentID string
	agentName    string
	modelName    string
	sessionID    string
	tags         map[string]string
	flushTimeout time.Duration
	mu           sync.RWMutex
}

// NewTracer creates a new tracer with the given configuration.
func NewTracer(config TracingConfig) (*Tracer, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	agentName := config.AgentName
	if agentName == "" {
		agentName = DefaultAgentName
	}

	sessionID := config.SessionID
	if sessionID == "" {
		// Generate UUID v4
		sessionID = generateUUIDv4()
	}

	flushTimeout := config.FlushTimeout
	if flushTimeout == 0 {
		flushTimeout = DefaultFlushTimeout
	}

	return &Tracer{
		client:       config.Client,
		experimentID: config.ExperimentID,
		agentName:    agentName,
		modelName:    config.ModelName,
		sessionID:    sessionID,
		tags:         config.Tags,
		flushTimeout: flushTimeout,
	}, nil
}

// StartTrace creates a new trace.
func (t *Tracer) StartTrace(ctx context.Context) (*Trace, context.Context, error) {
	// Check if trace already exists in context
	if existing := TraceFromContext(ctx); existing != nil {
		log.Printf("[WARN] tracing: StartTrace called with existing trace in context")
		return existing, ctx, nil
	}

	traceID, err := generateTraceID()
	if err != nil {
		return nil, ctx, fmt.Errorf("failed to generate trace ID: %w", err)
	}

	trace := &Trace{
		TraceID:       traceID,
		ExperimentID:  t.experimentID,
		RequestTime:   time.Now().UnixMilli(),
		State:         TraceStateInProgress,
		TraceMetadata: make(map[string]string),
		Tags:          make(map[string]string),
		Spans:         make([]*Span, 0),
	}

	// Set metadata
	if t.modelName != "" {
		trace.TraceMetadata["model"] = t.modelName
	}
	trace.TraceMetadata["mlflow.trace.session"] = t.sessionID

	// Copy tags
	for k, v := range t.tags {
		trace.Tags[k] = v
	}

	// Store trace in context
	ctx = contextWithTrace(ctx, trace)

	return trace, ctx, nil
}

// StartSpan creates a new span within a trace.
func (t *Tracer) StartSpan(ctx context.Context, name string, spanType string) (*Span, context.Context, error) {
	trace := TraceFromContext(ctx)
	if trace == nil {
		return nil, ctx, fmt.Errorf("no active trace in context")
	}

	spanID, err := generateSpanID()
	if err != nil {
		return nil, ctx, fmt.Errorf("failed to generate span ID: %w", err)
	}

	// Get parent span from context if exists
	parentID := ""
	if parent := SpanFromContext(ctx); parent != nil {
		parentID = parent.SpanID
	}

	span := &Span{
		TraceID:     trace.TraceID,
		SpanID:      spanID,
		ParentID:    parentID,
		Name:        name,
		SpanType:    spanType,
		StartTimeNs: time.Now().UnixNano(),
		Attributes:  make(map[string]any),
		Events:      make([]SpanEvent, 0),
	}

	// Set common attributes
	span.SetAttribute(AttrSpanType, spanType)
	span.SetAttribute(AttrRequestID, trace.TraceID)
	span.SetAttribute(AttrExperimentID, trace.ExperimentID)

	trace.AddSpan(span)

	// Store span in context
	ctx = contextWithSpan(ctx, span)

	return span, ctx, nil
}

// EndTrace finalizes a trace and flushes it to MLflow.
func (t *Tracer) EndTrace(ctx context.Context, trace *Trace, hasError bool) error {
	trace.mu.Lock()

	// Set final state
	if hasError || ctx.Err() != nil {
		trace.State = TraceStateError
	} else {
		trace.State = TraceStateOK
	}

	// Calculate execution duration
	trace.ExecutionDuration = time.Now().UnixMilli() - trace.RequestTime

	// Clean up orphan spans
	t.cleanupOrphanSpans(trace)

	trace.mu.Unlock()

	// Flush to MLflow
	return t.flush(ctx, trace)
}

// cleanupOrphanSpans ends any spans that were never explicitly ended.
func (t *Tracer) cleanupOrphanSpans(trace *Trace) {
	now := time.Now().UnixNano()
	orphanCount := 0

	for _, span := range trace.Spans {
		span.mu.Lock()
		if span.EndTimeNs == 0 {
			// Check if span has timed out
			if now-span.StartTimeNs > OrphanSpanTimeout.Nanoseconds() {
				span.EndTimeNs = now
				span.Status = SpanStatus{
					Code:        SpanStatusError,
					Description: "span timed out (no EndSpan called)",
				}
				orphanCount++
			} else {
				// Force end orphan span with error
				span.EndTimeNs = now
				span.Status = SpanStatus{
					Code:        SpanStatusError,
					Description: "span ended during trace finalization",
				}
				orphanCount++
			}
		}
		span.mu.Unlock()
	}

	if orphanCount > 0 {
		log.Printf("[WARN] tracing: ended %d orphan spans during trace finalization", orphanCount)
	}
}

// flush sends the trace to MLflow server.
func (t *Tracer) flush(ctx context.Context, trace *Trace) error {
	// Create a timeout context for the flush operation
	flushCtx, cancel := context.WithTimeout(ctx, t.flushTimeout)
	defer cancel()

	// Build trace proto
	traceProto := t.traceToProto(trace)

	// Create mlflowclient.Trace wrapper
	mlflowTrace := &mlflowclient.Trace{
		TraceInfoV3: traceProto,
	}

	// Send to MLflow
	resp, err := t.client.StartTrace(flushCtx, mlflowTrace)
	if err != nil {
		log.Printf("[ERROR] tracing: failed to flush trace to MLflow: %v", err)
		return err
	}

	// Note: The actual MLflow API doesn't have a separate "update trace with spans" endpoint.
	// Spans are sent as part of the initial StartTrace request or via separate span endpoints.
	// For simplicity, we'll log the trace ID from the response.
	if resp != nil && resp.TraceId != nil {
		log.Printf("[DEBUG] tracing: flushed trace %s to MLflow", *resp.TraceId)
	}

	return nil
}

// traceToProto converts a Trace to MLflow proto format.
func (t *Tracer) traceToProto(trace *Trace) *pb.TraceInfoV3 {
	// Add session ID to trace metadata
	metadata := make(map[string]string)
	for k, v := range trace.TraceMetadata {
		metadata[k] = v
	}
	metadata["mlflow.trace.session"] = t.sessionID

	return &pb.TraceInfoV3{
		TraceId:           &trace.TraceID,
		ClientRequestId:   &trace.TraceID,
		RequestPreview:    &trace.RequestPreview,
		ResponsePreview:   &trace.ResponsePreview,
		State:             convertTraceStateV3(trace.State).Enum(),
		TraceMetadata:     metadata,
		Tags:              trace.Tags,
	}
}

// spanToProto converts a Span to OpenTelemetry proto format.
func (t *Tracer) spanToProto(span *Span) *otelpb.Span {
	span.mu.Lock()
	defer span.mu.Unlock()

	// Convert trace ID and span ID to bytes
	traceIDBytes := hexToBytes(span.TraceID[3:]) // Remove "tr-" prefix
	spanIDBytes := hexToBytes(span.SpanID)
	parentIDBytes := hexToBytes(span.ParentID)

	// Convert attributes
	attributes := make([]*commonpb.KeyValue, 0, len(span.Attributes))
	for k, v := range span.Attributes {
		attributes = append(attributes, &commonpb.KeyValue{
			Key:   k,
			Value: convertAttributeValue(v),
		})
	}

	// Convert events
	events := make([]*otelpb.Span_Event, 0, len(span.Events))
	for _, e := range span.Events {
		eventAttrs := make([]*commonpb.KeyValue, 0, len(e.Attributes))
		for k, v := range e.Attributes {
			eventAttrs = append(eventAttrs, &commonpb.KeyValue{
				Key:   k,
				Value: convertAttributeValue(v),
			})
		}
		events = append(events, &otelpb.Span_Event{
			TimeUnixNano: uint64(e.Timestamp),
			Name:         e.Name,
			Attributes:   eventAttrs,
		})
	}

	// Convert status
	status := &otelpb.Status{
		Code:    convertSpanStatusCode(span.Status.Code),
		Message: span.Status.Description,
	}

	return &otelpb.Span{
		TraceId:           traceIDBytes,
		SpanId:            spanIDBytes,
		ParentSpanId:      parentIDBytes,
		Name:              span.Name,
		StartTimeUnixNano: uint64(span.StartTimeNs),
		EndTimeUnixNano:   uint64(span.EndTimeNs),
		Attributes:        attributes,
		Events:            events,
		Status:            status,
	}
}

// Helper functions

func generateTraceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "tr-" + hex.EncodeToString(b), nil
}

func generateSpanID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateUUIDv4() string {
	b := make([]byte, 16)
	rand.Read(b)
	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	// Find the last valid UTF-8 boundary before maxBytes
	for i := maxBytes; i >= 0; i-- {
		if utf8.ValidString(s[:i]) {
			return s[:i]
		}
	}

	return ""
}

func hexToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	b, _ := hex.DecodeString(s)
	return b
}

func convertAttributeValue(v any) *commonpb.AnyValue {
	switch val := v.(type) {
	case string:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: val}}
	case int:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: val}}
	case float64:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: val}}
	case bool:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: val}}
	default:
		// Fallback to JSON string
		data, _ := json.Marshal(v)
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: string(data)}}
	}
}

func convertSpanStatusCode(code SpanStatusCode) otelpb.Status_StatusCode {
	switch code {
	case SpanStatusOK:
		return otelpb.Status_STATUS_CODE_OK
	case SpanStatusError:
		return otelpb.Status_STATUS_CODE_ERROR
	default:
		return otelpb.Status_STATUS_CODE_UNSET
	}
}

func convertTraceState(state TraceState) pb.TraceStatus {
	switch state {
	case TraceStateOK:
		return pb.TraceStatus_OK
	case TraceStateError:
		return pb.TraceStatus_ERROR
	default:
		return pb.TraceStatus_IN_PROGRESS
	}
}

func convertTraceStateV3(state TraceState) pb.TraceInfoV3_State {
	switch state {
	case TraceStateOK:
		return pb.TraceInfoV3_OK
	case TraceStateError:
		return pb.TraceInfoV3_ERROR
	default:
		return pb.TraceInfoV3_IN_PROGRESS
	}
}

func convertMetadataToProto(metadata map[string]string) []*pb.TraceRequestMetadata {
	result := make([]*pb.TraceRequestMetadata, 0, len(metadata))
	for k, v := range metadata {
		key := k
		value := v
		result = append(result, &pb.TraceRequestMetadata{
			Key:   &key,
			Value: &value,
		})
	}
	return result
}

func convertTagsToProto(tags map[string]string) []*pb.TraceTag {
	result := make([]*pb.TraceTag, 0, len(tags))
	for k, v := range tags {
		key := k
		value := v
		result = append(result, &pb.TraceTag{
			Key:   &key,
			Value: &value,
		})
	}
	return result
}

// Context helpers

type contextKey int

const (
	traceContextKey contextKey = iota
	spanContextKey
)

func contextWithTrace(ctx context.Context, trace *Trace) context.Context {
	return context.WithValue(ctx, traceContextKey, trace)
}

func TraceFromContext(ctx context.Context) *Trace {
	trace, _ := ctx.Value(traceContextKey).(*Trace)
	return trace
}

func contextWithSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanContextKey, span)
}

func SpanFromContext(ctx context.Context) *Span {
	span, _ := ctx.Value(spanContextKey).(*Span)
	return span
}

// FlushAll flushes all pending traces (used during shutdown).
func FlushAll(ctx context.Context) error {
	// This is a placeholder for global trace management
	// In a real implementation, we'd maintain a registry of active tracers
	log.Printf("[DEBUG] tracing: FlushAll called")
	return nil
}
