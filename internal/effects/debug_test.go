package effects

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestDebugContext_LogAndCollect(t *testing.T) {
	ctx := NewDebugContext()
	ctx.SetTimestamp(42)

	// Log some messages
	ctx.Log("first message", "test.ail:1")
	ctx.Log("second message", "test.ail:2")

	// Collect and verify
	output := ctx.Collect()

	if len(output.Logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(output.Logs))
	}

	if output.Logs[0].Message != "first message" {
		t.Errorf("expected 'first message', got %q", output.Logs[0].Message)
	}

	if output.Logs[0].Location != "test.ail:1" {
		t.Errorf("expected 'test.ail:1', got %q", output.Logs[0].Location)
	}

	if output.Logs[0].Timestamp != 42 {
		t.Errorf("expected timestamp 42, got %d", output.Logs[0].Timestamp)
	}
}

func TestDebugContext_CheckAndCollect(t *testing.T) {
	ctx := NewDebugContext()

	// Record some assertions (using Check - renamed from Assert because assert is a reserved keyword)
	ctx.Check(true, "this should pass", "test.ail:10")
	ctx.Check(false, "this should fail", "test.ail:11")
	ctx.Check(true, "this also passes", "test.ail:12")

	// Collect and verify
	output := ctx.Collect()

	if len(output.Assertions) != 3 {
		t.Errorf("expected 3 assertions, got %d", len(output.Assertions))
	}

	if output.Assertions[0].Passed != true {
		t.Error("expected first assertion to pass")
	}

	if output.Assertions[1].Passed != false {
		t.Error("expected second assertion to fail")
	}

	if output.Assertions[1].Message != "this should fail" {
		t.Errorf("expected 'this should fail', got %q", output.Assertions[1].Message)
	}
}

func TestDebugContext_Reset(t *testing.T) {
	ctx := NewDebugContext()

	ctx.Log("message", "test.ail:1")
	ctx.Check(false, "assertion", "test.ail:2")

	// Verify data exists
	output := ctx.Collect()
	if len(output.Logs) != 1 || len(output.Assertions) != 1 {
		t.Error("expected data before reset")
	}

	// Reset
	ctx.Reset()

	// Verify data cleared
	output = ctx.Collect()
	if len(output.Logs) != 0 {
		t.Errorf("expected 0 logs after reset, got %d", len(output.Logs))
	}
	if len(output.Assertions) != 0 {
		t.Errorf("expected 0 assertions after reset, got %d", len(output.Assertions))
	}
}

func TestDebugContext_HasFailedAssertions(t *testing.T) {
	ctx := NewDebugContext()

	// No assertions - no failures
	if ctx.HasFailedAssertions() {
		t.Error("expected no failed assertions when empty")
	}

	// Only passing assertions
	ctx.Check(true, "pass", "test.ail:1")
	if ctx.HasFailedAssertions() {
		t.Error("expected no failed assertions when all pass")
	}

	// Add a failed assertion
	ctx.Check(false, "fail", "test.ail:2")
	if !ctx.HasFailedAssertions() {
		t.Error("expected failed assertion detected")
	}
}

func TestDebugContext_FailedAssertions(t *testing.T) {
	ctx := NewDebugContext()

	ctx.Check(true, "pass 1", "test.ail:1")
	ctx.Check(false, "fail 1", "test.ail:2")
	ctx.Check(true, "pass 2", "test.ail:3")
	ctx.Check(false, "fail 2", "test.ail:4")

	failed := ctx.FailedAssertions()

	if len(failed) != 2 {
		t.Errorf("expected 2 failed assertions, got %d", len(failed))
	}

	if failed[0].Message != "fail 1" {
		t.Errorf("expected 'fail 1', got %q", failed[0].Message)
	}

	if failed[1].Message != "fail 2" {
		t.Errorf("expected 'fail 2', got %q", failed[1].Message)
	}
}

func TestDebugLog_EffectOperation(t *testing.T) {
	// Create effect context with Debug capability
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Debug"))
	ctx.Debug = NewDebugContext()
	ctx.Debug.SetTimestamp(100)

	// Call debug log operation
	args := []eval.Value{
		&eval.StringValue{Value: "test message"},
		&eval.StringValue{Value: "module.ail:42"},
	}

	result, err := Call(ctx, "Debug", "log", args)
	if err != nil {
		t.Fatalf("debug log failed: %v", err)
	}

	if _, ok := result.(*eval.UnitValue); !ok {
		t.Errorf("expected UnitValue, got %T", result)
	}

	// Verify log was recorded
	output := ctx.Debug.Collect()
	if len(output.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(output.Logs))
	}

	if output.Logs[0].Message != "test message" {
		t.Errorf("expected 'test message', got %q", output.Logs[0].Message)
	}

	if output.Logs[0].Location != "module.ail:42" {
		t.Errorf("expected 'module.ail:42', got %q", output.Logs[0].Location)
	}

	if output.Logs[0].Timestamp != 100 {
		t.Errorf("expected timestamp 100, got %d", output.Logs[0].Timestamp)
	}
}

func TestDebugCheck_EffectOperation(t *testing.T) {
	// Create effect context with Debug capability
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Debug"))
	ctx.Debug = NewDebugContext()

	// Call debug check operation (passing)
	// Note: renamed from assert to check because assert is a reserved keyword
	argsPass := []eval.Value{
		&eval.BoolValue{Value: true},
		&eval.StringValue{Value: "should pass"},
		&eval.StringValue{Value: "module.ail:10"},
	}

	result, err := Call(ctx, "Debug", "check", argsPass)
	if err != nil {
		t.Fatalf("debug check failed: %v", err)
	}

	if _, ok := result.(*eval.UnitValue); !ok {
		t.Errorf("expected UnitValue, got %T", result)
	}

	// Call debug check operation (failing)
	argsFail := []eval.Value{
		&eval.BoolValue{Value: false},
		&eval.StringValue{Value: "should fail"},
		&eval.StringValue{Value: "module.ail:11"},
	}

	result, err = Call(ctx, "Debug", "check", argsFail)
	if err != nil {
		t.Fatalf("debug check failed: %v", err)
	}

	// Even failing assertions return UnitValue (not an error)
	if _, ok := result.(*eval.UnitValue); !ok {
		t.Errorf("expected UnitValue even for failed assertion, got %T", result)
	}

	// Verify assertions were recorded
	output := ctx.Debug.Collect()
	if len(output.Assertions) != 2 {
		t.Fatalf("expected 2 assertions, got %d", len(output.Assertions))
	}

	if !output.Assertions[0].Passed {
		t.Error("expected first assertion to pass")
	}

	if output.Assertions[1].Passed {
		t.Error("expected second assertion to fail")
	}
}

func TestDebug_AlwaysAvailable(t *testing.T) {
	// Debug is a ghost effect — always auto-granted, no --caps needed
	ctx := NewEffContext(nil)

	args := []eval.Value{
		&eval.StringValue{Value: "test message"},
		&eval.StringValue{Value: "test.ail:1"},
	}

	_, err := Call(ctx, "Debug", "log", args)
	if err != nil {
		t.Fatalf("Debug should always be available (ghost effect), got error: %v", err)
	}

	out := ctx.Debug.Collect()
	if len(out.Logs) != 1 || out.Logs[0].Message != "test message" {
		t.Errorf("expected 1 log entry with 'test message', got %v", out.Logs)
	}
}
