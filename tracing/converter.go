package tracing

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	pb "charm.land/fantasy/proto/gen/mlflow"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConvertTrace converts a tracing.Trace to MLflow protobuf format.
func ConvertTrace(trace *Trace) (*pb.Trace, error) {
	if trace == nil {
		return nil, fmt.Errorf("trace cannot be nil")
	}

	trace.mu.RLock()
	defer trace.mu.RUnlock()

	// Convert trace state
	var state pb.TraceInfoV3_State
	switch trace.State {
	case TraceStateOK:
		state = pb.TraceInfoV3_OK
	case TraceStateError:
		state = pb.TraceInfoV3_ERROR
	case TraceStateInProgress:
		state = pb.TraceInfoV3_IN_PROGRESS
	default:
		state = pb.TraceInfoV3_STATE_UNSPECIFIED
	}

	// Create trace location
	locType := pb.TraceLocation_MLFLOW_EXPERIMENT
	location := &pb.TraceLocation{
		Type: &locType,
		Identifier: &pb.TraceLocation_MlflowExperiment{
			MlflowExperiment: &pb.TraceLocation_MlflowExperimentLocation{
				ExperimentId: &trace.ExperimentID,
			},
		},
	}

	// Create TraceInfoV3
	traceInfo := &pb.TraceInfoV3{
		TraceId:           &trace.TraceID,
		TraceLocation:     location,
		RequestPreview:    &trace.RequestPreview,
		ResponsePreview:   &trace.ResponsePreview,
		RequestTime:       timestamppb.New(millisToTime(trace.RequestTime)),
		ExecutionDuration: durationpb.New(millisToDuration(trace.ExecutionDuration)),
		State:             &state,
		TraceMetadata:     trace.TraceMetadata,
		Tags:              trace.Tags,
	}

	// Convert spans to OpenTelemetry format
	otSpans := make([]*tracev1.Span, 0, len(trace.Spans))
	for _, span := range trace.Spans {
		otSpan, err := convertSpan(span)
		if err != nil {
			return nil, fmt.Errorf("failed to convert span %s: %w", span.SpanID, err)
		}
		otSpans = append(otSpans, otSpan)
	}

	// Create the pb.Trace
	pbTrace := &pb.Trace{
		TraceInfo: traceInfo,
		Spans:     otSpans,
	}

	return pbTrace, nil
}

// convertSpan converts a tracing.Span to OpenTelemetry protobuf format.
func convertSpan(span *Span) (*tracev1.Span, error) {
	if span == nil {
		return nil, fmt.Errorf("span cannot be nil")
	}

	span.mu.Lock()
	defer span.mu.Unlock()

	// Convert IDs from hex strings to bytes
	traceID, err := hexToBytes(span.TraceID, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid trace ID: %w", err)
	}

	spanID, err := hexToBytes(span.SpanID, 8)
	if err != nil {
		return nil, fmt.Errorf("invalid span ID: %w", err)
	}

	var parentSpanID []byte
	if span.ParentID != "" {
		parentSpanID, err = hexToBytes(span.ParentID, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid parent span ID: %w", err)
		}
	}

	// Convert span kind (always internal for GenAI traces)
	kind := tracev1.Span_SPAN_KIND_INTERNAL

	// Convert status
	var status *tracev1.Status
	switch span.Status.Code {
	case SpanStatusOK:
		status = &tracev1.Status{
			Code:    tracev1.Status_STATUS_CODE_OK,
			Message: span.Status.Description,
		}
	case SpanStatusError:
		status = &tracev1.Status{
			Code:    tracev1.Status_STATUS_CODE_ERROR,
			Message: span.Status.Description,
		}
	default:
		status = &tracev1.Status{
			Code:    tracev1.Status_STATUS_CODE_UNSET,
			Message: "",
		}
	}

	// Convert attributes
	attributes := make([]*commonv1.KeyValue, 0, len(span.Attributes))
	for key, value := range span.Attributes {
		kv, err := convertAttribute(key, value)
		if err != nil {
			// Skip invalid attributes
			continue
		}
		attributes = append(attributes, kv)
	}

	// Convert events
	events := make([]*tracev1.Span_Event, 0, len(span.Events))
	for _, event := range span.Events {
		otEvent, err := convertEvent(event)
		if err != nil {
			// Skip invalid events
			continue
		}
		events = append(events, otEvent)
	}

	// Create OpenTelemetry span
	otSpan := &tracev1.Span{
		TraceId:           traceID,
		SpanId:            spanID,
		ParentSpanId:      parentSpanID,
		Name:              span.Name,
		Kind:              kind,
		StartTimeUnixNano: uint64(span.StartTimeNs),
		EndTimeUnixNano:   uint64(span.EndTimeNs),
		Attributes:        attributes,
		Events:            events,
		Status:            status,
	}

	return otSpan, nil
}

// convertAttribute converts a Go value to an OpenTelemetry KeyValue.
func convertAttribute(key string, value any) (*commonv1.KeyValue, error) {
	anyValue := convertToAnyValue(value)
	if anyValue == nil {
		return nil, fmt.Errorf("failed to convert attribute value")
	}

	return &commonv1.KeyValue{
		Key:   key,
		Value: anyValue,
	}, nil
}

// convertToAnyValue converts a Go value to an OpenTelemetry AnyValue.
func convertToAnyValue(value any) *commonv1.AnyValue {
	switch v := value.(type) {
	case string:
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: v},
		}
	case int:
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_IntValue{IntValue: int64(v)},
		}
	case int64:
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_IntValue{IntValue: v},
		}
	case float64:
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_DoubleValue{DoubleValue: v},
		}
	case bool:
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_BoolValue{BoolValue: v},
		}
	case []byte:
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_BytesValue{BytesValue: v},
		}
	case TokenUsage:
		// Marshal to JSON string
		data, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: string(data)},
		}
	case map[string]any:
		// Convert to KeyValueList
		kvs := make([]*commonv1.KeyValue, 0, len(v))
		for k, val := range v {
			anyVal := convertToAnyValue(val)
			if anyVal != nil {
				kvs = append(kvs, &commonv1.KeyValue{
					Key:   k,
					Value: anyVal,
				})
			}
		}
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_KvlistValue{
				KvlistValue: &commonv1.KeyValueList{Values: kvs},
			},
		}
	case []any:
		// Convert to ArrayValue
		vals := make([]*commonv1.AnyValue, 0, len(v))
		for _, val := range v {
			anyVal := convertToAnyValue(val)
			if anyVal != nil {
				vals = append(vals, anyVal)
			}
		}
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_ArrayValue{
				ArrayValue: &commonv1.ArrayValue{Values: vals},
			},
		}
	default:
		// Try to marshal to JSON as fallback
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		return &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: string(data)},
		}
	}
}

// convertEvent converts a SpanEvent to OpenTelemetry format.
func convertEvent(event SpanEvent) (*tracev1.Span_Event, error) {
	// Convert attributes
	attributes := make([]*commonv1.KeyValue, 0, len(event.Attributes))
	for key, value := range event.Attributes {
		kv, err := convertAttribute(key, value)
		if err != nil {
			continue
		}
		attributes = append(attributes, kv)
	}

	return &tracev1.Span_Event{
		TimeUnixNano: uint64(event.Timestamp),
		Name:         event.Name,
		Attributes:   attributes,
	}, nil
}

// hexToBytes converts a hex string to bytes, padding or truncating to the specified length.
func hexToBytes(hexStr string, expectedLen int) ([]byte, error) {
	// Decode hex string
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}

	// Pad or truncate to expected length
	if len(bytes) < expectedLen {
		// Pad with zeros
		padded := make([]byte, expectedLen)
		copy(padded[expectedLen-len(bytes):], bytes)
		return padded, nil
	} else if len(bytes) > expectedLen {
		// Truncate from the left
		return bytes[len(bytes)-expectedLen:], nil
	}

	return bytes, nil
}
