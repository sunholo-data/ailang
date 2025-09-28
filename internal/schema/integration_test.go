package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/sunholo/ailang/internal/errors"
	"github.com/sunholo/ailang/internal/schema"
	"github.com/sunholo/ailang/internal/test"
)

// TestErrorSchemaIntegration verifies error JSON schemas work end-to-end
func TestErrorSchemaIntegration(t *testing.T) {
	// Create an error through the errors package
	err := errors.NewTypecheck("TC#123", errors.TC001, "Type mismatch", nil)
	
	// Convert to JSON
	jsonData, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf("Failed to convert error to JSON: %v", jsonErr)
	}
	
	// Parse the JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	// Verify schema field exists and is correct
	schemaField, ok := parsed["schema"].(string)
	if !ok {
		t.Fatal("Missing or invalid schema field")
	}
	
	if !schema.Accepts(schemaField, schema.ErrorV1) {
		t.Errorf("Schema %q not accepted by %q", schemaField, schema.ErrorV1)
	}
	
	// Verify all required fields are present
	requiredFields := []string{"schema", "sid", "phase", "code", "message", "fix"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}
}

// TestTestReportSchemaIntegration verifies test report JSON schemas work end-to-end
func TestTestReportSchemaIntegration(t *testing.T) {
	// Create a test report
	runner := test.NewRunner()
	runner.RunTest("integration", "test1", func() error { return nil })
	report := runner.GetReport()
	
	// Convert to JSON
	jsonData, err := report.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert report to JSON: %v", err)
	}
	
	// Parse the JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	// Verify schema field exists and is correct
	schemaField, ok := parsed["schema"].(string)
	if !ok {
		t.Fatal("Missing or invalid schema field")
	}
	
	if !schema.Accepts(schemaField, schema.TestV1) {
		t.Errorf("Schema %q not accepted by %q", schemaField, schema.TestV1)
	}
	
	// Verify all required fields are present
	requiredFields := []string{"schema", "run_id", "duration_ms", "counts", "cases", "platform"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}
}

// TestCompactModeIntegration verifies compact mode works with real data
func TestCompactModeIntegration(t *testing.T) {
	// Create a test report
	runner := test.NewRunner()
	runner.RunTest("compact", "test1", func() error { return nil })
	report := runner.GetReport()
	
	// Test pretty mode (default)
	schema.SetCompactMode(false)
	prettyJSON, err := report.ToJSON()
	if err != nil {
		t.Fatalf("Failed to generate pretty JSON: %v", err)
	}
	
	// Test compact mode
	schema.SetCompactMode(true)
	compactJSON, err := report.ToJSON()
	if err != nil {
		t.Fatalf("Failed to generate compact JSON: %v", err)
	}
	
	// Pretty should have newlines, compact should not
	prettyStr := string(prettyJSON)
	compactStr := string(compactJSON)
	
	if len(prettyStr) <= len(compactStr) {
		t.Error("Pretty JSON should be longer than compact JSON")
	}
	
	// Both should parse to the same data
	var prettyParsed, compactParsed interface{}
	if err := json.Unmarshal(prettyJSON, &prettyParsed); err != nil {
		t.Fatalf("Failed to parse pretty JSON: %v", err)
	}
	if err := json.Unmarshal(compactJSON, &compactParsed); err != nil {
		t.Fatalf("Failed to parse compact JSON: %v", err)
	}
	
	// Reset to default
	schema.SetCompactMode(false)
}

// TestDeterministicOutput verifies JSON output is deterministic
func TestDeterministicOutput(t *testing.T) {
	// Run the same test multiple times and verify identical output
	outputs := make([]string, 3)
	
	for i := 0; i < 3; i++ {
		runner := test.NewRunner()
		runner.RunTest("deterministic", "test1", func() error { return nil })
		runner.RunTest("deterministic", "test2", func() error { return nil })
		report := runner.GetReport()
		
		// Override random fields for deterministic comparison
		report.RunID = "fixed_run_id"
		report.Platform.Timestamp = "2024-01-01T00:00:00Z"
		report.DurationMs = 100
		for j := range report.Cases {
			report.Cases[j].TimeMs = 10
		}
		
		jsonData, err := report.ToJSON()
		if err != nil {
			t.Fatalf("Failed to generate JSON (iteration %d): %v", i, err)
		}
		
		outputs[i] = string(jsonData)
	}
	
	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("Output %d differs from output 0:\nOutput 0:\n%s\nOutput %d:\n%s", 
				i, outputs[0], i, outputs[i])
		}
	}
}