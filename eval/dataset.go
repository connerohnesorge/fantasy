package eval

import (
	"encoding/json"
	"fmt"
	"os"
)

// TestCase represents a single test case in an evaluation dataset.
type TestCase struct {
	// Inputs contains the input data for this test case.
	Inputs map[string]any `json:"inputs"`

	// Expectations contains the expected values for comparison.
	Expectations map[string]any `json:"expectations"`

	// Outputs contains pre-generated outputs (optional).
	// If nil, the predict function will be used to generate outputs.
	Outputs map[string]any `json:"outputs,omitempty"`

	// Tags contains optional metadata tags for this test case.
	Tags map[string]string `json:"tags,omitempty"`
}

// Dataset represents a collection of test cases for evaluation.
type Dataset struct {
	// Name is the identifier for this dataset.
	Name string `json:"name"`

	// TestCases contains the individual test cases.
	TestCases []TestCase `json:"test_cases"`

	// Metadata contains optional dataset-level metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// LoadDataset loads a dataset from a JSON file.
func LoadDataset(path string) (*Dataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read dataset file: %w", err)
	}

	var dataset Dataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, fmt.Errorf("failed to parse dataset JSON: %w", err)
	}

	// Validate the dataset
	if err := validateDataset(&dataset); err != nil {
		return nil, fmt.Errorf("dataset validation failed: %w", err)
	}

	return &dataset, nil
}

// validateDataset checks that a dataset meets the required constraints.
func validateDataset(dataset *Dataset) error {
	if dataset.Name == "" {
		return fmt.Errorf("dataset must have a non-empty name")
	}

	if len(dataset.TestCases) == 0 {
		return fmt.Errorf("dataset must have at least one test case")
	}

	for i, tc := range dataset.TestCases {
		if tc.Inputs == nil {
			return fmt.Errorf("test case %d: inputs must not be nil", i)
		}

		// Check that inputs and expectations are JSON-serializable
		if err := validateJSONSerializable(tc.Inputs); err != nil {
			return fmt.Errorf("test case %d inputs: %w", i, err)
		}

		if tc.Expectations != nil {
			if err := validateJSONSerializable(tc.Expectations); err != nil {
				return fmt.Errorf("test case %d expectations: %w", i, err)
			}
		}

		if tc.Outputs != nil {
			if err := validateJSONSerializable(tc.Outputs); err != nil {
				return fmt.Errorf("test case %d outputs: %w", i, err)
			}
		}
	}

	return nil
}

// validateJSONSerializable checks that a value can be serialized to JSON.
func validateJSONSerializable(v any) error {
	_, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("value is not JSON-serializable: %w", err)
	}
	return nil
}
