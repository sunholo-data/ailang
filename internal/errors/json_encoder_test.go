package errors

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/schema"
)

func TestNewTypecheck(t *testing.T) {
	err := NewTypecheck("N#42", TC001, "Type mismatch", nil)

	if err.Schema != schema.ErrorV1 {
		t.Errorf("Expected schema %s, got %s", schema.ErrorV1, err.Schema)
	}

	if err.Phase != "typecheck" {
		t.Errorf("Expected phase typecheck, got %s", err.Phase)
	}

	if err.Code != TC001 {
		t.Errorf("Expected code %s, got %s", TC001, err.Code)
	}

	if err.SID != "N#42" {
		t.Errorf("Expected SID N#42, got %s", err.SID)
	}

	// Test with empty SID
	err2 := NewTypecheck("", TC002, "Unbound variable", nil)
	if err2.SID != "unknown" {
		t.Errorf("Expected SID unknown for empty input, got %s", err2.SID)
	}
}

func TestWithFix(t *testing.T) {
	err := NewTypecheck("N#1", TC006, "Missing type annotation", nil)
	err = err.WithFix("Add type annotation: x: Int", 0.9)

	if err.Fix.Suggestion != "Add type annotation: x: Int" {
		t.Errorf("Expected fix suggestion, got %s", err.Fix.Suggestion)
	}

	if err.Fix.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", err.Fix.Confidence)
	}
}

func TestWithSourceSpan(t *testing.T) {
	err := NewElaboration("N#2", ELB001, "Invalid AST", nil)
	err = err.WithSourceSpan("main.ail:10:5")

	if err.SourceSpan != "main.ail:10:5" {
		t.Errorf("Expected source span main.ail:10:5, got %s", err.SourceSpan)
	}
}

func TestWithMeta(t *testing.T) {
	meta := map[string]string{
		"hint": "Check variable scoping",
		"severity": "error",
	}

	err := NewLinking("N#3", LNK001, "Missing instance", nil)
	err = err.WithMeta(meta)

	if err.Meta == nil {
		t.Error("Expected meta to be set")
	}
}

func TestToJSON(t *testing.T) {
	ctx := ErrorContext{
		Constraints: []string{"Num a", "a ~ Int"},
		Decisions:   []string{"defaulted a -> Int"},
	}

	err := NewTypecheck("N#42", TC007, "Defaulting ambiguity", ctx).
		WithFix("Add explicit type annotation", 0.85).
		WithSourceSpan("test.ail:5:10")

	jsonData, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf("ToJSON failed: %v", jsonErr)
	}

	// Parse back to verify structure
	var result map[string]interface{}
	if parseErr := json.Unmarshal(jsonData, &result); parseErr != nil {
		t.Fatalf("Failed to parse JSON: %v", parseErr)
	}

	// Check required fields
	if result["schema"] != schema.ErrorV1 {
		t.Errorf("Expected schema %s, got %v", schema.ErrorV1, result["schema"])
	}

	if result["phase"] != "typecheck" {
		t.Errorf("Expected phase typecheck, got %v", result["phase"])
	}

	if result["code"] != TC007 {
		t.Errorf("Expected code %s, got %v", TC007, result["code"])
	}

	// Check fix is always present
	if _, ok := result["fix"]; !ok {
		t.Error("Fix field should always be present")
	}
}

func TestSafeEncodeError(t *testing.T) {
	// Test with nil error
	result := SafeEncodeError(nil, "typecheck")
	if result != nil {
		t.Error("Expected nil for nil error")
	}

	// Test with regular error
	testErr := &testError{msg: "test error"}
	result = SafeEncodeError(testErr, "runtime")

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if parsed["phase"] != "runtime" {
		t.Errorf("Expected phase runtime, got %v", parsed["phase"])
	}

	if !strings.Contains(parsed["message"].(string), "test error") {
		t.Errorf("Expected message to contain 'test error', got %v", parsed["message"])
	}
}

func TestFormatSourceSpan(t *testing.T) {
	tests := []struct {
		file     string
		line     int
		col      int
		expected string
	}{
		{"main.ail", 10, 5, "main.ail:10:5"},
		{"test.ail", 1, 1, "test.ail:1:1"},
		{"/path/to/file.ail", 100, 25, "/path/to/file.ail:100:25"},
	}

	for _, tt := range tests {
		result := FormatSourceSpan(tt.file, tt.line, tt.col)
		if result != tt.expected {
			t.Errorf("FormatSourceSpan(%s, %d, %d) = %s, want %s",
				tt.file, tt.line, tt.col, result, tt.expected)
		}
	}
}

func TestErrorCodes(t *testing.T) {
	// Verify error codes follow the taxonomy
	typecheckCodes := []string{TC001, TC002, TC003, TC004, TC005, TC006, TC007}
	for _, code := range typecheckCodes {
		if !strings.HasPrefix(code, "TC") {
			t.Errorf("Typecheck code %s should start with TC", code)
		}
	}

	elaborationCodes := []string{ELB001, ELB002, ELB003, ELB004}
	for _, code := range elaborationCodes {
		if !strings.HasPrefix(code, "ELB") {
			t.Errorf("Elaboration code %s should start with ELB", code)
		}
	}

	linkingCodes := []string{LNK001, LNK002, LNK003, LNK004}
	for _, code := range linkingCodes {
		if !strings.HasPrefix(code, "LNK") {
			t.Errorf("Linking code %s should start with LNK", code)
		}
	}

	runtimeCodes := []string{RT001, RT002, RT003, RT004, RT005, RT006}
	for _, code := range runtimeCodes {
		if !strings.HasPrefix(code, "RT") {
			t.Errorf("Runtime code %s should start with RT", code)
		}
	}
}

// Helper type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}