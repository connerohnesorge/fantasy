package mlflowclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pb "charm.land/fantasy/proto/gen/mlflow"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestStartTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces") {
			t.Errorf("expected traces path, got %s", r.URL.Path)
		}

		// Return trace response
		traceInfo := &pb.TraceInfoV3{
			TraceId: stringPtr("tr-123"),
			State:   pb.TraceInfoV3_IN_PROGRESS.Enum(),
		}
		resp := &pb.StartTraceV3_Response{
			Trace: &pb.Trace{
				TraceInfo: traceInfo,
			},
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	trace := &Trace{
		TraceInfoV3: &pb.TraceInfoV3{
			TraceId: stringPtr("tr-123"),
			State:   pb.TraceInfoV3_IN_PROGRESS.Enum(),
		},
	}

	result, err := client.StartTrace(ctx, trace)
	if err != nil {
		t.Fatalf("StartTrace failed: %v", err)
	}

	if result.GetTraceId() != "tr-123" {
		t.Errorf("expected trace ID tr-123, got %s", result.GetTraceId())
	}
}

func TestGetTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces/get") {
			t.Errorf("expected traces/get path, got %s", r.URL.Path)
		}

		// Check query params
		if !strings.Contains(r.URL.RawQuery, "trace_id=tr-123") {
			t.Errorf("expected trace_id in query params, got %s", r.URL.RawQuery)
		}

		// Return trace
		trace := &pb.Trace{
			TraceInfo: &pb.TraceInfoV3{
				TraceId: stringPtr("tr-123"),
			},
		}
		resp := &pb.GetTrace_Response{
			Trace: trace,
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	trace, err := client.GetTrace(ctx, "tr-123", false)
	if err != nil {
		t.Fatalf("GetTrace failed: %v", err)
	}

	if trace.TraceInfo.GetTraceId() != "tr-123" {
		t.Errorf("expected trace ID tr-123, got %s", trace.TraceInfo.GetTraceId())
	}
}

func TestSearchTraces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces/search") {
			t.Errorf("expected traces/search path, got %s", r.URL.Path)
		}

		// Return search results
		traces := []*pb.TraceInfoV3{
			{
				TraceId: stringPtr("tr-123"),
			},
			{
				TraceId: stringPtr("tr-456"),
			},
		}
		resp := &pb.SearchTracesV3_Response{
			Traces:        traces,
			NextPageToken: stringPtr("next-token"),
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	opts := &SearchTracesOptions{
		ExperimentIDs: []string{"exp-123"},
		Filter:        "status = 'OK'",
		MaxResults:    10,
	}

	result, err := client.SearchTraces(ctx, opts)
	if err != nil {
		t.Fatalf("SearchTraces failed: %v", err)
	}

	if len(result.Traces) != 2 {
		t.Errorf("expected 2 traces, got %d", len(result.Traces))
	}

	if result.NextPageToken != "next-token" {
		t.Errorf("expected next-token, got %s", result.NextPageToken)
	}
}

func TestDeleteTraces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces/delete-traces") {
			t.Errorf("expected traces/delete-traces path, got %s", r.URL.Path)
		}

		// Return deletion count
		deleted := int32(5)
		resp := &pb.DeleteTracesV3_Response{
			TracesDeleted: &deleted,
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	opts := &DeleteTracesOptions{
		MaxTraces: 5,
	}

	count, err := client.DeleteTraces(ctx, "exp-123", opts)
	if err != nil {
		t.Fatalf("DeleteTraces failed: %v", err)
	}

	if count != 5 {
		t.Errorf("expected 5 traces deleted, got %d", count)
	}
}

func TestDeleteTracesValidation(t *testing.T) {
	client := New("http://localhost:5000")
	ctx := context.Background()

	// Test missing experiment ID
	_, err := client.DeleteTraces(ctx, "", &DeleteTracesOptions{MaxTraces: 5})
	if err == nil {
		t.Fatal("expected validation error for empty experiment ID")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "experimentID" {
		t.Errorf("expected field experimentID, got %s", valErr.Field)
	}

	// Test missing options
	_, err = client.DeleteTraces(ctx, "exp-123", nil)
	if err == nil {
		t.Fatal("expected validation error for nil options")
	}

	valErr, ok = err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "options" {
		t.Errorf("expected field options, got %s", valErr.Field)
	}
}
