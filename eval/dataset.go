package eval

import (
	"encoding/json"
	"fmt"
	"os"
)

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
		return nil, err
	}

	return &dataset, nil
}

// SaveDataset saves a dataset to a JSON file.
func SaveDataset(path string, dataset *Dataset) error {
	data, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write dataset file: %w", err)
	}

	return nil
}

// NewDataset creates a new dataset with the given name and test cases.
func NewDataset(name string, testCases ...*TestCase) *Dataset {
	return &Dataset{
		Name:      name,
		TestCases: testCases,
	}
}

// NewTestCase creates a new test case with the given inputs and expectations.
func NewTestCase(inputs, expectations map[string]any) *TestCase {
	return &TestCase{
		Inputs:       inputs,
		Expectations: expectations,
	}
}

// WithOutputs creates a test case with pre-generated outputs.
func (tc *TestCase) WithOutputs(outputs map[string]any) *TestCase {
	tc.Outputs = outputs
	return tc
}

// WithTags adds tags to a test case.
func (tc *TestCase) WithTags(tags map[string]string) *TestCase {
	tc.Tags = tags
	return tc
}
