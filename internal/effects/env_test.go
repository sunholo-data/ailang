package effects

import (
	"os"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// TestEnvGetEnv_Success tests successful environment variable retrieval
func TestEnvGetEnv_Success(t *testing.T) {
	// Setup
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))
	ctx.EnvSnapshot = map[string]string{
		"TEST_VAR": "test_value",
		"EMPTY":    "",
	}

	// Test: Get existing variable
	result, err := envGetEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "TEST_VAR"},
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return Ok(value) Result
	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}

	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}

	if len(tagged.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(tagged.Fields))
	}

	strVal, ok := tagged.Fields[0].(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue field, got %T", tagged.Fields[0])
	}

	if strVal.Value != "test_value" {
		t.Errorf("expected 'test_value', got '%s'", strVal.Value)
	}
}

// TestEnvGetEnv_EmptyValue tests retrieval of empty environment variable
func TestEnvGetEnv_EmptyValue(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))
	ctx.EnvSnapshot = map[string]string{
		"EMPTY": "",
	}

	result, err := envGetEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "EMPTY"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return Ok("") for empty value
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok for empty value, got %s", tagged.CtorName)
	}

	strVal := tagged.Fields[0].(*eval.StringValue)
	if strVal.Value != "" {
		t.Errorf("expected empty string, got '%s'", strVal.Value)
	}
}

// TestEnvGetEnv_NotFound tests missing environment variable
func TestEnvGetEnv_NotFound(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))
	ctx.EnvSnapshot = map[string]string{
		"EXISTS": "value",
	}

	result, err := envGetEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "MISSING"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return Err(NotFound) Result
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err Result, got %s", tagged.CtorName)
	}

	// Unwrap the Err to get the EnvError
	envErr := tagged.Fields[0].(*eval.TaggedValue)
	if envErr.CtorName != "NotFound" {
		t.Fatalf("expected NotFound error, got %s", envErr.CtorName)
	}

	// Check error message
	msgVal := envErr.Fields[0].(*eval.StringValue)
	if msgVal.Value == "" {
		t.Error("expected non-empty error message")
	}
}

// TestEnvGetEnv_NoCapability tests capability requirement
func TestEnvGetEnv_NoCapability(t *testing.T) {
	ctx := NewEffContext([]string{})
	// Don't grant Env capability
	ctx.EnvSnapshot = map[string]string{
		"TEST": "value",
	}

	_, err := envGetEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "TEST"},
	})

	// Should return error, not Result
	if err == nil {
		t.Fatal("expected error for missing capability")
	}

	if err.Error() == "" {
		t.Error("expected descriptive error message")
	}
}

// TestEnvGetEnv_Allowlist tests allowlist enforcement
func TestEnvGetEnv_Allowlist(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))
	ctx.EnvSnapshot = map[string]string{
		"ALLOWED":     "allowed_value",
		"NOT_ALLOWED": "secret_value",
	}
	ctx.EnvAllowlist = []string{"ALLOWED"}

	// Test: Allowed variable succeeds
	result, err := envGetEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "ALLOWED"},
	})
	if err != nil {
		t.Fatalf("unexpected error for allowed variable: %v", err)
	}

	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Errorf("expected Ok for allowed variable, got %s", tagged.CtorName)
	}

	// Test: Non-allowed variable returns Err(NotAllowed)
	result, err = envGetEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "NOT_ALLOWED"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tagged = result.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Errorf("expected Err Result for non-allowed variable, got %s", tagged.CtorName)
	}

	// Unwrap the Err to get the EnvError
	envErr2 := tagged.Fields[0].(*eval.TaggedValue)
	if envErr2.CtorName != "NotAllowed" {
		t.Errorf("expected NotAllowed error, got %s", envErr2.CtorName)
	}
}

// TestEnvHasEnv_Success tests successful existence check
func TestEnvHasEnv_Success(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))
	ctx.EnvSnapshot = map[string]string{
		"EXISTS": "value",
	}

	// Test: Variable exists
	result, err := envHasEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "EXISTS"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", result)
	}

	if !boolVal.Value {
		t.Error("expected true for existing variable")
	}

	// Test: Variable doesn't exist
	result, err = envHasEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "MISSING"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boolVal = result.(*eval.BoolValue)
	if boolVal.Value {
		t.Error("expected false for missing variable")
	}
}

// TestEnvHasEnv_NoCapability tests capability requirement
func TestEnvHasEnv_NoCapability(t *testing.T) {
	ctx := NewEffContext([]string{})
	// Don't grant Env capability
	ctx.EnvSnapshot = map[string]string{
		"TEST": "value",
	}

	_, err := envHasEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "TEST"},
	})

	if err == nil {
		t.Fatal("expected error for missing capability")
	}
}

// TestEnvHasEnv_Allowlist tests allowlist enforcement
func TestEnvHasEnv_Allowlist(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))
	ctx.EnvSnapshot = map[string]string{
		"ALLOWED":     "value",
		"NOT_ALLOWED": "secret",
	}
	ctx.EnvAllowlist = []string{"ALLOWED"}

	// Test: Allowed variable returns true
	result, err := envHasEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "ALLOWED"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boolVal := result.(*eval.BoolValue)
	if !boolVal.Value {
		t.Error("expected true for allowed existing variable")
	}

	// Test: Non-allowed variable returns false (don't reveal existence)
	result, err = envHasEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "NOT_ALLOWED"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boolVal = result.(*eval.BoolValue)
	if boolVal.Value {
		t.Error("expected false for non-allowed variable (security)")
	}
}

// TestCaptureEnvSnapshot tests snapshot capture functionality
func TestCaptureEnvSnapshot(t *testing.T) {
	// Set test environment variable
	os.Setenv("AILANG_TEST_VAR", "test_value")
	defer os.Unsetenv("AILANG_TEST_VAR")

	snapshot := captureEnvSnapshot()

	// Should capture the test variable
	value, exists := snapshot["AILANG_TEST_VAR"]
	if !exists {
		t.Fatal("expected AILANG_TEST_VAR in snapshot")
	}

	if value != "test_value" {
		t.Errorf("expected 'test_value', got '%s'", value)
	}
}

// TestSnapshotImmutability tests that snapshot is immutable
func TestSnapshotImmutability(t *testing.T) {
	// Set initial environment variable
	os.Setenv("AILANG_IMMUTABLE_TEST", "initial")
	defer os.Unsetenv("AILANG_IMMUTABLE_TEST")

	// Create context (captures snapshot)
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))

	// Change OS environment
	os.Setenv("AILANG_IMMUTABLE_TEST", "changed")

	// Snapshot should still have initial value
	value, exists := ctx.EnvSnapshot["AILANG_IMMUTABLE_TEST"]
	if !exists {
		t.Fatal("expected variable in snapshot")
	}

	if value != "initial" {
		t.Errorf("snapshot was mutated! expected 'initial', got '%s'", value)
	}

	// Also test via envGetEnv
	result, err := envGetEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "AILANG_IMMUTABLE_TEST"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tagged := result.(*eval.TaggedValue)
	strVal := tagged.Fields[0].(*eval.StringValue)
	if strVal.Value != "initial" {
		t.Errorf("getEnv returned changed value! expected 'initial', got '%s'", strVal.Value)
	}
}

// TestEnvGetEnv_WrongArgCount tests argument validation
func TestEnvGetEnv_WrongArgCount(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))

	// No arguments
	_, err := envGetEnv(ctx, []eval.Value{})
	if err == nil {
		t.Error("expected error for no arguments")
	}

	// Too many arguments
	_, err = envGetEnv(ctx, []eval.Value{
		&eval.StringValue{Value: "VAR1"},
		&eval.StringValue{Value: "VAR2"},
	})
	if err == nil {
		t.Error("expected error for too many arguments")
	}
}

// TestEnvGetEnv_WrongArgType tests type validation
func TestEnvGetEnv_WrongArgType(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Env"))

	// Wrong argument type
	_, err := envGetEnv(ctx, []eval.Value{
		&eval.IntValue{Value: 42},
	})
	if err == nil {
		t.Error("expected error for wrong argument type")
	}
}
