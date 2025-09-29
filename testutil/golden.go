// Package testutil provides utilities for golden file testing.
package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// UpdateGoldens controls whether to update golden files
// Set via environment variable: UPDATE_GOLDENS=true go test ./...
var UpdateGoldens = os.Getenv("UPDATE_GOLDENS") == "true"

// GoldenMeta captures platform information for reproducibility
type GoldenMeta struct {
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GoldenFile represents a golden test file with metadata
type GoldenFile struct {
	Meta GoldenMeta  `json:"meta"`
	Data interface{} `json:"data"`
}

// GetGoldenPath returns the path to a golden file
func GetGoldenPath(feature, name string) string {
	return filepath.Join("testdata", feature, name+".golden.json")
}

// CompareWithGolden compares actual output with golden file
func CompareWithGolden(t *testing.T, feature, name string, actual interface{}) {
	t.Helper()

	goldenPath := GetGoldenPath(feature, name)

	// Create golden file structure
	goldenData := GoldenFile{
		Meta: GoldenMeta{
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		},
		Data: actual,
	}

	// Marshal to deterministic JSON
	actualJSON, err := marshalDeterministic(goldenData)
	if err != nil {
		t.Fatalf("failed to marshal actual data: %v", err)
	}

	if UpdateGoldens {
		// Update mode: write the golden file
		err := os.MkdirAll(filepath.Dir(goldenPath), 0755)
		if err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}

		err = os.WriteFile(goldenPath, actualJSON, 0644)
		if err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}

		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Compare mode: read and compare
	expectedJSON, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file does not exist: %s\nRun with UPDATE_GOLDENS=true to create", goldenPath)
		}
		t.Fatalf("failed to read golden file: %v", err)
	}

	// Compare JSON content (ignoring whitespace differences)
	if !jsonEqual(actualJSON, expectedJSON) {
		t.Errorf("golden file mismatch for %s/%s\nExpected:\n%s\nActual:\n%s",
			feature, name, string(expectedJSON), string(actualJSON))
	}
}

// AssertGoldenJSON compares JSON output with golden file
func AssertGoldenJSON(t *testing.T, feature, name string, actualJSON []byte) {
	t.Helper()

	var actual interface{}
	if err := json.Unmarshal(actualJSON, &actual); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	CompareWithGolden(t, feature, name, actual)
}

// marshalDeterministic marshals with sorted keys
func marshalDeterministic(v interface{}) ([]byte, error) {
	// First marshal to get a map
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	// Unmarshal to generic interface
	var m interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	// Re-marshal with indentation for readability
	return json.MarshalIndent(m, "", "  ")
}

// jsonEqual compares two JSON byte slices for equality
func jsonEqual(a, b []byte) bool {
	var aData, bData interface{}

	if err := json.Unmarshal(a, &aData); err != nil {
		return false
	}

	if err := json.Unmarshal(b, &bData); err != nil {
		return false
	}

	aJSON, _ := json.Marshal(aData)
	bJSON, _ := json.Marshal(bData)

	return bytes.Equal(aJSON, bJSON)
}

// CreateGoldenTest creates a test that compares with golden files
func CreateGoldenTest(t *testing.T, feature string, tests []struct {
	Name string
	Data interface{}
}) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			CompareWithGolden(t, feature, tt.Name, tt.Data)
		})
	}
}

// LoadGoldenFile loads and returns a golden file's data
func LoadGoldenFile(t *testing.T, feature, name string) interface{} {
	t.Helper()

	goldenPath := GetGoldenPath(feature, name)
	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to load golden file %s: %v", goldenPath, err)
	}

	var golden GoldenFile
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("failed to unmarshal golden file: %v", err)
	}

	return golden.Data
}

// DiffJSON returns a string showing the differences between two JSON values
func DiffJSON(expected, actual interface{}) string {
	expJSON, _ := json.MarshalIndent(expected, "", "  ")
	actJSON, _ := json.MarshalIndent(actual, "", "  ")

	expLines := strings.Split(string(expJSON), "\n")
	actLines := strings.Split(string(actJSON), "\n")

	var diff strings.Builder
	diff.WriteString("JSON Diff:\n")

	maxLines := len(expLines)
	if len(actLines) > maxLines {
		maxLines = len(actLines)
	}

	for i := 0; i < maxLines; i++ {
		var expLine, actLine string

		if i < len(expLines) {
			expLine = expLines[i]
		}
		if i < len(actLines) {
			actLine = actLines[i]
		}

		if expLine != actLine {
			fmt.Fprintf(&diff, "- %s\n", expLine)
			fmt.Fprintf(&diff, "+ %s\n", actLine)
		}
	}

	return diff.String()
}
