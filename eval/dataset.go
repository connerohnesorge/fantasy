package eval

import (
	"encoding/json"
	"fmt"
	"os"
)

// TestCase represents a single test case in a dataset.
type TestCase struct {
	ID           string            // Optional unique identifier
	Inputs       map[string]any    // Required: input data
	Expectations map[string]any    // Expected output values
	Outputs      map[string]any    // Optional: pre-generated outputs
	Tags         map[string]string // Optional: metadata tags
}

// Dataset represents a collection of test cases.
type Dataset struct {
	Name      string
	TestCases []TestCase
	Metadata  map[string]any
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

	if err := dataset.Validate(); err != nil {
		return nil, fmt.Errorf("dataset validation failed: %w", err)
	}

	return &dataset, nil
}

// Validate checks if the dataset is valid.
func (d *Dataset) Validate() error {
	if d.Name == "" {
		return fmt.Errorf("dataset name is required")
	}

	if len(d.TestCases) == 0 {
		return fmt.Errorf("dataset must have at least one test case")
	}

	for i, tc := range d.TestCases {
		if tc.Inputs == nil {
			return fmt.Errorf("test case %d: Inputs map is required (cannot be nil)", i)
		}

		// Validate that inputs and expectations are JSON-serializable
		if err := validateJSONSerializable(tc.Inputs); err != nil {
			return fmt.Errorf("test case %d: Inputs contain non-JSON-serializable values: %w", i, err)
		}

		if tc.Expectations != nil {
			if err := validateJSONSerializable(tc.Expectations); err != nil {
				return fmt.Errorf("test case %d: Expectations contain non-JSON-serializable values: %w", i, err)
			}
		}
	}

	return nil
}

// validateJSONSerializable checks if a value can be JSON-serialized.
func validateJSONSerializable(v any) error {
	_, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("value is not JSON-serializable: %w", err)
	}
	return nil
}
