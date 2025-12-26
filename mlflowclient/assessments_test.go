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

func TestCreateAssessment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces/tr-123/assessments") {
			t.Errorf("expected assessments path, got %s", r.URL.Path)
		}

		// Return created assessment
		assessment := &pb.Assessment{
			AssessmentId:   stringPtr("assessment-123"),
			AssessmentName: stringPtr("correctness"),
		}
		resp := &pb.CreateAssessment_Response{
			Assessment: assessment,
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	assessment := &pb.Assessment{
		AssessmentName: stringPtr("correctness"),
	}

	result, err := client.CreateAssessment(ctx, "tr-123", assessment)
	if err != nil {
		t.Fatalf("CreateAssessment failed: %v", err)
	}

	if result.GetAssessmentId() != "assessment-123" {
		t.Errorf("expected assessment ID assessment-123, got %s", result.GetAssessmentId())
	}
}

func TestGetAssessment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces/tr-123/assessments/assessment-123") {
			t.Errorf("expected assessment path, got %s", r.URL.Path)
		}

		// Return assessment
		assessment := &pb.Assessment{
			AssessmentId:   stringPtr("assessment-123"),
			AssessmentName: stringPtr("correctness"),
		}
		resp := &pb.GetAssessmentRequest_Response{
			Assessment: assessment,
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	result, err := client.GetAssessment(ctx, "tr-123", "assessment-123")
	if err != nil {
		t.Fatalf("GetAssessment failed: %v", err)
	}

	if result.GetAssessmentName() != "correctness" {
		t.Errorf("expected name correctness, got %s", result.GetAssessmentName())
	}
}

func TestUpdateAssessment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces/tr-123/assessments/assessment-123") {
			t.Errorf("expected assessment path, got %s", r.URL.Path)
		}

		// Return updated assessment
		assessment := &pb.Assessment{
			AssessmentId:   stringPtr("assessment-123"),
			AssessmentName: stringPtr("correctness"),
			Rationale:      stringPtr("Updated rationale"),
		}
		resp := &pb.UpdateAssessment_Response{
			Assessment: assessment,
		}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	assessment := &pb.Assessment{
		Rationale: stringPtr("Updated rationale"),
	}

	result, err := client.UpdateAssessment(ctx, "tr-123", "assessment-123", assessment, []string{"rationale"})
	if err != nil {
		t.Fatalf("UpdateAssessment failed: %v", err)
	}

	if result.GetRationale() != "Updated rationale" {
		t.Errorf("expected rationale 'Updated rationale', got %s", result.GetRationale())
	}
}

func TestDeleteAssessment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/3.0/mlflow/traces/tr-123/assessments/assessment-123") {
			t.Errorf("expected assessment path, got %s", r.URL.Path)
		}

		resp := &pb.DeleteAssessment_Response{}
		data, _ := protojson.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	client := New(server.URL)
	ctx := context.Background()

	err := client.DeleteAssessment(ctx, "tr-123", "assessment-123")
	if err != nil {
		t.Fatalf("DeleteAssessment failed: %v", err)
	}
}
