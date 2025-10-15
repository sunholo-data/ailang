package schema

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestGoldenErrorJSON tests that error JSON is deterministic and matches schema
func TestGoldenErrorJSON(t *testing.T) {
	tests := []struct {
		name     string
		err      map[string]interface{}
		wantJSON string // Exact expected JSON output
	}{
		{
			name: "type_mismatch_error",
			err: map[string]interface{}{
				"schema":  ErrorV1,
				"sid":     "TC#001",
				"phase":   "typecheck",
				"code":    "TC001",
				"message": "Type mismatch: expected Int, got String",
				"fix": map[string]interface{}{
					"suggestion": "",
					"confidence": 0.0,
				},
				"context": map[string]interface{}{
					"constraints": []string{"Num a", "a = String"},
					"trace_slice": "TC#001 -> TC#002 -> TC#003",
				},
			},
			wantJSON: `{
  "code": "TC001",
  "context": {
    "constraints": [
      "Num a",
      "a = String"
    ],
    "trace_slice": "TC#001 -> TC#002 -> TC#003"
  },
  "fix": {
    "confidence": 0,
    "suggestion": ""
  },
  "message": "Type mismatch: expected Int, got String",
  "phase": "typecheck",
  "schema": "ailang.error/v1",
  "sid": "TC#001"
}`,
		},
		{
			name: "linking_error_with_fix",
			err: map[string]interface{}{
				"schema":  ErrorV1,
				"sid":     "LNK#042",
				"phase":   "linking",
				"code":    "LNK001",
				"message": "Missing instance: Num String",
				"fix": map[string]interface{}{
					"suggestion": "Import std/num/string or define instance",
					"confidence": 0.85,
				},
			},
			wantJSON: `{
  "code": "LNK001",
  "fix": {
    "confidence": 0.85,
    "suggestion": "Import std/num/string or define instance"
  },
  "message": "Missing instance: Num String",
  "phase": "linking",
  "schema": "ailang.error/v1",
  "sid": "LNK#042"
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use MarshalDeterministic which should produce sorted keys
			got, err := MarshalDeterministic(tt.err)
			if err != nil {
				t.Fatalf("MarshalDeterministic() error = %v", err)
			}

			formatted, err := FormatJSON(got)
			if err != nil {
				t.Fatalf("FormatJSON() error = %v", err)
			}

			// Normalize whitespace for comparison
			wantNorm := normalizeJSON(t, tt.wantJSON)
			gotNorm := normalizeJSON(t, string(formatted))

			if gotNorm != wantNorm {
				t.Errorf("JSON mismatch:\nGot:\n%s\nWant:\n%s", gotNorm, wantNorm)
			}

			// Verify schema acceptance
			var parsed map[string]interface{}
			if err := json.Unmarshal(got, &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if schemaField, ok := parsed["schema"].(string); ok {
				if !Accepts(schemaField, ErrorV1) {
					t.Errorf("Schema %q does not accept %q", schemaField, ErrorV1)
				}
			} else {
				t.Error("Missing schema field in JSON output")
			}
		})
	}
}

// TestGoldenTestReportJSON tests that test report JSON is deterministic
func TestGoldenTestReportJSON(t *testing.T) {
	// Create a mock test report structure similar to what test package would produce
	report := map[string]interface{}{
		"schema":      TestV1,
		"run_id":      "test_run_001",
		"seed":        42,
		"duration_ms": 38,
		"cases": []interface{}{
			map[string]interface{}{
				"sid":     "T#abc123",
				"suite":   "parser",
				"name":    "parse_lambda",
				"status":  "passed",
				"time_ms": 15,
			},
			map[string]interface{}{
				"sid":     "T#def456",
				"suite":   "typecheck",
				"name":    "infer_lambda",
				"status":  "failed",
				"time_ms": 23,
				"error":   "Type mismatch",
			},
		},
		"counts": map[string]interface{}{
			"passed":  1,
			"failed":  1,
			"errored": 0,
			"skipped": 0,
			"total":   2,
		},
		"platform": map[string]interface{}{
			"go_version": "go1.21.0",
			"os":         "darwin",
			"arch":       "arm64",
			"timestamp":  "2024-01-01T00:00:00Z",
		},
	}

	wantJSON := `{
  "cases": [
    {
      "name": "parse_lambda",
      "sid": "T#abc123",
      "status": "passed",
      "suite": "parser",
      "time_ms": 15
    },
    {
      "error": "Type mismatch",
      "name": "infer_lambda",
      "sid": "T#def456",
      "status": "failed",
      "suite": "typecheck",
      "time_ms": 23
    }
  ],
  "counts": {
    "errored": 0,
    "failed": 1,
    "passed": 1,
    "skipped": 0,
    "total": 2
  },
  "duration_ms": 38,
  "platform": {
    "arch": "arm64",
    "go_version": "go1.21.0",
    "os": "darwin",
    "timestamp": "2024-01-01T00:00:00Z"
  },
  "run_id": "test_run_001",
  "schema": "ailang.test/v1",
  "seed": 42
}`

	got, err := MarshalDeterministic(report)
	if err != nil {
		t.Fatalf("MarshalDeterministic() error = %v", err)
	}

	formatted, err := FormatJSON(got)
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}

	wantNorm := normalizeJSON(t, wantJSON)
	gotNorm := normalizeJSON(t, string(formatted))

	if gotNorm != wantNorm {
		t.Errorf("JSON mismatch:\nGot:\n%s\nWant:\n%s", gotNorm, wantNorm)
	}
}

// TestGoldenCompactMode tests that compact mode works correctly
func TestGoldenCompactMode(t *testing.T) {
	data := map[string]interface{}{
		"schema": "ailang.test/v1",
		"counts": map[string]interface{}{
			"passed": 10,
			"failed": 2,
		},
	}

	// Test pretty mode
	SetCompactMode(false)
	pretty, err := MarshalDeterministic(data)
	if err != nil {
		t.Fatalf("MarshalDeterministic error = %v", err)
	}
	prettyFormatted, err := FormatJSON(pretty)
	if err != nil {
		t.Fatalf("FormatJSON error = %v", err)
	}

	if !strings.Contains(string(prettyFormatted), "\n") {
		t.Error("Pretty mode should contain newlines")
	}

	// Test compact mode
	SetCompactMode(true)
	compact, err := MarshalDeterministic(data)
	if err != nil {
		t.Fatalf("MarshalDeterministic error = %v", err)
	}
	compactFormatted, err := FormatJSON(compact)
	if err != nil {
		t.Fatalf("FormatJSON error = %v", err)
	}

	if strings.Contains(string(compactFormatted), "\n") {
		t.Error("Compact mode should not contain newlines")
	}

	// Verify JSON is still valid and deterministic
	wantCompact := `{"counts":{"failed":2,"passed":10},"schema":"ailang.test/v1"}`
	if string(compactFormatted) != wantCompact {
		t.Errorf("Compact JSON mismatch:\nGot:  %s\nWant: %s", string(compactFormatted), wantCompact)
	}

	// Reset to default
	SetCompactMode(false)
}

// TestAcceptsCompatibility tests schema version compatibility
func TestAcceptsCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		want     string
		expected bool
	}{
		// Exact matches
		{"exact error v1", "ailang.error/v1", ErrorV1, true},
		{"exact test v1", "ailang.test/v1", TestV1, true},
		{"exact effects v1", "ailang.effects/v1", EffectsV1, true},

		// Minor versions should be accepted
		{"error v1.1", "ailang.error/v1.1", ErrorV1, true},
		{"test v1.2.3", "ailang.test/v1.2.3", TestV1, true},

		// Major version mismatches should be rejected
		{"error v2", "ailang.error/v2", ErrorV1, false},
		{"test v2", "ailang.test/v2", TestV1, false},

		// Different schemas should be rejected
		{"wrong schema", "ailang.test/v1", ErrorV1, false},
		{"wrong schema 2", "ailang.error/v1", TestV1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Accepts(tt.got, tt.want); got != tt.expected {
				t.Errorf("Accepts(%q, %q) = %v, want %v", tt.got, tt.want, got, tt.expected)
			}
		})
	}
}

// normalizeJSON normalizes JSON for comparison by parsing and re-formatting
func normalizeJSON(t *testing.T, jsonStr string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("Invalid JSON: %v\nJSON: %s", err, jsonStr)
	}

	normalized, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to normalize JSON: %v", err)
	}

	return string(normalized)
}
