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

func TestSetTraceTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces/tr-123/tags") {
			t.Errorf("expected trace tags path, got %s", r.URL.Path)
		}

		resp := &pb.SetTraceTagV3_Response{}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	err := client.SetTraceTag(ctx, "tr-123", "env", "prod")
	if err != nil {
		t.Fatalf("SetTraceTag failed: %v", err)
	}
}

func TestDeleteTraceTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces/tr-123/tags/env") {
			t.Errorf("expected trace tag delete path, got %s", r.URL.Path)
		}

		resp := &pb.DeleteTraceTagV3_Response{}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	err := client.DeleteTraceTag(ctx, "tr-123", "env")
	if err != nil {
		t.Fatalf("DeleteTraceTag failed: %v", err)
	}
}

func TestSetRunTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/2.0/mlflow/runs/set-tag") {
			t.Errorf("expected runs/set-tag path, got %s", r.URL.Path)
		}

		resp := &pb.SetTag_Response{}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	err := client.SetRunTag(ctx, "run-123", "env", "prod")
	if err != nil {
		t.Fatalf("SetRunTag failed: %v", err)
	}
}

func TestTagValidation(t *testing.T) {
	client := New("http://localhost:5000")
	ctx := context.Background()

	// Test empty trace ID
	err := client.SetTraceTag(ctx, "", "key", "value")
	if err == nil {
		t.Fatal("expected validation error for empty trace ID")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "traceID" {
		t.Errorf("expected field traceID, got %s", valErr.Field)
	}

	// Test empty key
	err = client.SetTraceTag(ctx, "tr-123", "", "value")
	if err == nil {
		t.Fatal("expected validation error for empty key")
	}

	valErr, ok = err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if valErr.Field != "key" {
		t.Errorf("expected field key, got %s", valErr.Field)
	}
}
