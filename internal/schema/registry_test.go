package schema

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAccepts(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		want     string
		expected bool
	}{
		{"exact match", "ailang.error/v1", "ailang.error/v1", true},
		{"minor version", "ailang.error/v1.1", "ailang.error/v1", true},
		{"patch version", "ailang.error/v1.0.1", "ailang.error/v1", true},
		{"major mismatch", "ailang.error/v2", "ailang.error/v1", false},
		{"different schema", "ailang.test/v1", "ailang.error/v1", false},
		{"missing version", "ailang.error", "ailang.error/v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Accepts(tt.got, tt.want); got != tt.expected {
				t.Errorf("Accepts(%q, %q) = %v, want %v", tt.got, tt.want, got, tt.expected)
			}
		})
	}
}

func TestMarshalDeterministic(t *testing.T) {
	// Test that keys are sorted
	data := map[string]interface{}{
		"zebra":  "last",
		"alpha":  "first",
		"middle": "middle",
	}

	result, err := MarshalDeterministic(data)
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}

	// Check that keys appear in alphabetical order
	expected := `{"alpha":"first","middle":"middle","zebra":"last"}`
	if string(result) != expected {
		t.Errorf("Got %s, want %s", string(result), expected)
	}
}

func TestMarshalDeterministic_Nested(t *testing.T) {
	// Test nested objects are also sorted
	data := map[string]interface{}{
		"outer2": map[string]interface{}{
			"inner2": 2,
			"inner1": 1,
		},
		"outer1": "value",
	}

	result, err := MarshalDeterministic(data)
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}

	// Verify both outer and inner keys are sorted
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Check the JSON string contains keys in order
	str := string(result)
	if !strings.Contains(str, `"outer1":"value"`) ||
		!strings.Contains(str, `"inner1":1`) ||
		!strings.Contains(str, `"inner2":2`) {
		t.Errorf("Keys not in expected order: %s", str)
	}
}

func TestFormatJSON(t *testing.T) {
	data := []byte(`{"test":"value","number":42}`)

	// Test pretty format (default)
	SetCompactMode(false)
	result, err := FormatJSON(data)
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	if !strings.Contains(string(result), "\n") {
		t.Error("Expected pretty format with newlines")
	}

	// Test compact format
	SetCompactMode(true)
	result, err = FormatJSON(data)
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	if strings.Contains(string(result), "\n") {
		t.Error("Expected compact format without newlines")
	}

	// Reset to default
	SetCompactMode(false)
}

func TestMustValidate(t *testing.T) {
	// Test with matching schema
	data := map[string]interface{}{
		"schema":  "ailang.error/v1",
		"message": "test error",
	}

	err := MustValidate(ErrorV1, data)
	if err != nil {
		t.Errorf("MustValidate failed for valid schema: %v", err)
	}

	// Test with mismatched schema
	data["schema"] = "ailang.test/v1"
	err = MustValidate(ErrorV1, data)
	if err == nil {
		t.Error("MustValidate should have failed for mismatched schema")
	}

	// Test with missing schema field
	delete(data, "schema")
	err = MustValidate(ErrorV1, data)
	if err != nil {
		t.Error("MustValidate should pass when schema field is missing (no-op)")
	}
}
