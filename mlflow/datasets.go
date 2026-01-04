package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Dataset represents an MLflow evaluation dataset.
type Dataset struct {
	DatasetID     string   `json:"dataset_id,omitempty"`
	Name          string   `json:"name"`
	Tags          string   `json:"tags,omitempty"`           // JSON string
	Schema        string   `json:"schema,omitempty"`         // JSON string
	Profile       string   `json:"profile,omitempty"`        // JSON string
	Digest        string   `json:"digest,omitempty"`
	CreatedTime   int64    `json:"created_time,omitempty"`
	LastUpdateTime int64   `json:"last_update_time,omitempty"`
	CreatedBy     string   `json:"created_by,omitempty"`
	LastUpdatedBy string   `json:"last_updated_by,omitempty"`
	ExperimentIDs []string `json:"experiment_ids,omitempty"`
}

// DatasetRecordSourceType represents the source type for a dataset record.
type DatasetRecordSourceType string

const (
	SourceTypeUnspecified DatasetRecordSourceType = "SOURCE_TYPE_UNSPECIFIED"
	SourceTypeTrace       DatasetRecordSourceType = "TRACE"
	SourceTypeHuman       DatasetRecordSourceType = "HUMAN"
	SourceTypeDocument    DatasetRecordSourceType = "DOCUMENT"
	SourceTypeCode        DatasetRecordSourceType = "CODE"
)

// DatasetRecord represents a single record in an evaluation dataset.
type DatasetRecord struct {
	DatasetRecordID string                  `json:"dataset_record_id,omitempty"`
	DatasetID       string                  `json:"dataset_id,omitempty"`
	Inputs          string                  `json:"inputs,omitempty"`       // JSON string
	Expectations    string                  `json:"expectations,omitempty"` // JSON string
	Outputs         string                  `json:"outputs,omitempty"`      // JSON string
	Tags            string                  `json:"tags,omitempty"`         // JSON string
	Source          string                  `json:"source,omitempty"`       // JSON string
	SourceID        string                  `json:"source_id,omitempty"`
	SourceType      DatasetRecordSourceType `json:"source_type,omitempty"`
	CreatedTime     int64                   `json:"created_time,omitempty"`
	LastUpdateTime  int64                   `json:"last_update_time,omitempty"`
	CreatedBy       string                  `json:"created_by,omitempty"`
	LastUpdatedBy   string                  `json:"last_updated_by,omitempty"`
}

// CreateDatasetOptions configures dataset creation.
type CreateDatasetOptions struct {
	ExperimentIDs []string          `json:"experiment_ids,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

// CreateDataset creates a new evaluation dataset.
//
// API endpoint: POST /api/3.0/mlflow/datasets/create
//
// Parameters:
//   - ctx: Context for the request
//   - name: Name of the dataset
//   - opts: Optional configuration
//
// Returns the created dataset.
func (c *Client) CreateDataset(ctx context.Context, name string, opts *CreateDatasetOptions) (*Dataset, error) {
	if name == "" {
		return nil, &ValidationError{
			Field:   "name",
			Message: "name cannot be empty",
			Value:   "",
		}
	}

	// Build request body
	reqBody := map[string]any{
		"name": name,
	}

	if opts != nil {
		if len(opts.ExperimentIDs) > 0 {
			reqBody["experiment_ids"] = opts.ExperimentIDs
		}
		if len(opts.Tags) > 0 {
			tagsJSON, err := json.Marshal(opts.Tags)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tags: %w", err)
			}
			reqBody["tags"] = string(tagsJSON)
		}
	}

	// Make request
	var resp struct {
		Dataset Dataset `json:"dataset"`
	}

	if err := c.doJSONRequest(ctx, http.MethodPost, "/api/3.0/mlflow/datasets/create", reqBody, &resp); err != nil {
		return nil, err
	}

	return &resp.Dataset, nil
}

// GetDataset retrieves a dataset by ID.
//
// API endpoint: GET /api/3.0/mlflow/datasets/{dataset_id}
func (c *Client) GetDataset(ctx context.Context, datasetID string) (*Dataset, error) {
	if datasetID == "" {
		return nil, &ValidationError{
			Field:   "datasetID",
			Message: "datasetID cannot be empty",
			Value:   "",
		}
	}

	path := fmt.Sprintf("/api/3.0/mlflow/datasets/%s", datasetID)

	var resp struct {
		Dataset Dataset `json:"dataset"`
	}

	if err := c.doJSONRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &resp.Dataset, nil
}

// DeleteDataset deletes a dataset by ID.
//
// API endpoint: DELETE /api/3.0/mlflow/datasets/{dataset_id}
func (c *Client) DeleteDataset(ctx context.Context, datasetID string) error {
	if datasetID == "" {
		return &ValidationError{
			Field:   "datasetID",
			Message: "datasetID cannot be empty",
			Value:   "",
		}
	}

	path := fmt.Sprintf("/api/3.0/mlflow/datasets/%s", datasetID)

	if err := c.doJSONRequest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return err
	}

	return nil
}

// UpsertDatasetRecordsResult contains the result of upserting records.
type UpsertDatasetRecordsResult struct {
	InsertedCount int `json:"inserted_count"`
	UpdatedCount  int `json:"updated_count"`
}

// UpsertDatasetRecords inserts or updates records in a dataset.
//
// API endpoint: POST /api/3.0/mlflow/datasets/{dataset_id}/records
//
// Parameters:
//   - ctx: Context for the request
//   - datasetID: ID of the dataset
//   - records: Records to upsert (must be JSON-serializable)
//
// Returns counts of inserted and updated records.
func (c *Client) UpsertDatasetRecords(ctx context.Context, datasetID string, records []map[string]any) (*UpsertDatasetRecordsResult, error) {
	if datasetID == "" {
		return nil, &ValidationError{
			Field:   "datasetID",
			Message: "datasetID cannot be empty",
			Value:   "",
		}
	}

	if len(records) == 0 {
		return &UpsertDatasetRecordsResult{}, nil
	}

	// Serialize records to JSON string
	recordsJSON, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal records: %w", err)
	}

	path := fmt.Sprintf("/api/3.0/mlflow/datasets/%s/records", datasetID)

	reqBody := map[string]any{
		"records": string(recordsJSON),
	}

	var resp struct {
		InsertedCount int `json:"inserted_count"`
		UpdatedCount  int `json:"updated_count"`
	}

	if err := c.doJSONRequest(ctx, http.MethodPost, path, reqBody, &resp); err != nil {
		return nil, err
	}

	return &UpsertDatasetRecordsResult{
		InsertedCount: resp.InsertedCount,
		UpdatedCount:  resp.UpdatedCount,
	}, nil
}

// GetDatasetRecordsOptions configures record retrieval.
type GetDatasetRecordsOptions struct {
	MaxResults int    `json:"max_results,omitempty"`
	PageToken  string `json:"page_token,omitempty"`
}

// GetDatasetRecordsResult contains retrieved records and pagination info.
type GetDatasetRecordsResult struct {
	Records       []DatasetRecord `json:"records"`
	NextPageToken string          `json:"next_page_token,omitempty"`
}

// GetDatasetRecords retrieves records from a dataset.
//
// API endpoint: GET /api/3.0/mlflow/datasets/{dataset_id}/records
func (c *Client) GetDatasetRecords(ctx context.Context, datasetID string, opts *GetDatasetRecordsOptions) (*GetDatasetRecordsResult, error) {
	if datasetID == "" {
		return nil, &ValidationError{
			Field:   "datasetID",
			Message: "datasetID cannot be empty",
			Value:   "",
		}
	}

	path := fmt.Sprintf("/api/3.0/mlflow/datasets/%s/records", datasetID)

	// Build query params
	reqBody := map[string]any{}
	if opts != nil {
		if opts.MaxResults > 0 {
			reqBody["max_results"] = opts.MaxResults
		}
		if opts.PageToken != "" {
			reqBody["page_token"] = opts.PageToken
		}
	}

	var resp struct {
		Records       string `json:"records"` // JSON string
		NextPageToken string `json:"next_page_token,omitempty"`
	}

	if err := c.doJSONRequest(ctx, http.MethodGet, path, reqBody, &resp); err != nil {
		return nil, err
	}

	// Parse records JSON
	var records []DatasetRecord
	if resp.Records != "" {
		if err := json.Unmarshal([]byte(resp.Records), &records); err != nil {
			return nil, fmt.Errorf("failed to parse records: %w", err)
		}
	}

	return &GetDatasetRecordsResult{
		Records:       records,
		NextPageToken: resp.NextPageToken,
	}, nil
}

// SearchEvaluationDatasetsOptions configures dataset search.
type SearchEvaluationDatasetsOptions struct {
	ExperimentIDs []string `json:"experiment_ids,omitempty"`
	MaxResults    int      `json:"max_results,omitempty"`
	PageToken     string   `json:"page_token,omitempty"`
}

// SearchEvaluationDatasetsResult contains search results.
type SearchEvaluationDatasetsResult struct {
	Datasets      []Dataset `json:"datasets"`
	NextPageToken string    `json:"next_page_token,omitempty"`
}

// SearchEvaluationDatasets searches for datasets.
//
// API endpoint: GET /api/3.0/mlflow/datasets
func (c *Client) SearchEvaluationDatasets(ctx context.Context, opts *SearchEvaluationDatasetsOptions) (*SearchEvaluationDatasetsResult, error) {
	reqBody := map[string]any{}
	if opts != nil {
		if len(opts.ExperimentIDs) > 0 {
			reqBody["experiment_ids"] = opts.ExperimentIDs
		}
		if opts.MaxResults > 0 {
			reqBody["max_results"] = opts.MaxResults
		}
		if opts.PageToken != "" {
			reqBody["page_token"] = opts.PageToken
		}
	}

	var resp struct {
		Datasets      []Dataset `json:"datasets"`
		NextPageToken string    `json:"next_page_token,omitempty"`
	}

	if err := c.doJSONRequest(ctx, http.MethodGet, "/api/3.0/mlflow/datasets", reqBody, &resp); err != nil {
		return nil, err
	}

	return &SearchEvaluationDatasetsResult{
		Datasets:      resp.Datasets,
		NextPageToken: resp.NextPageToken,
	}, nil
}
