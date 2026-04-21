package effects

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

func TestNewContractContext(t *testing.T) {
	ctx := NewContractContext()

	if ctx == nil {
		t.Fatal("NewContractContext returned nil")
	}

	if ctx.Mode != ContractModePanic {
		t.Errorf("expected default mode Panic, got %v", ctx.Mode)
	}

	if len(ctx.checks) != 0 {
		t.Errorf("expected empty checks, got %d", len(ctx.checks))
	}
}

func TestNewContractContextWithMode(t *testing.T) {
	ctx := NewContractContextWithMode(ContractModeReport)

	if ctx.Mode != ContractModeReport {
		t.Errorf("expected Report mode, got %v", ctx.Mode)
	}
}

func TestContractMode_String(t *testing.T) {
	tests := []struct {
		mode     ContractMode
		expected string
	}{
		{ContractModePanic, "panic"},
		{ContractModeReport, "report"},
		{ContractModeOff, "off"},
		{ContractMode(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.expected {
			t.Errorf("ContractMode(%d).String() = %q, want %q", tt.mode, got, tt.expected)
		}
	}
}

func TestCheckRequires_PanicMode_Pass(t *testing.T) {
	ctx := NewContractContext()
	ctx.SetFunction("foo")

	err := ctx.CheckRequires(true, "x >= 0", "test.ail:10")
	if err != nil {
		t.Errorf("CheckRequires should pass, got error: %v", err)
	}

	output := ctx.Collect()
	if output.TotalCount != 1 {
		t.Errorf("expected 1 check, got %d", output.TotalCount)
	}
	if output.PassCount != 1 {
		t.Errorf("expected 1 pass, got %d", output.PassCount)
	}
	if output.FailCount != 0 {
		t.Errorf("expected 0 fails, got %d", output.FailCount)
	}
}

func TestCheckRequires_PanicMode_Fail(t *testing.T) {
	ctx := NewContractContext()
	ctx.SetFunction("foo")

	err := ctx.CheckRequires(false, "x >= 0", "test.ail:10")
	if err == nil {
		t.Error("CheckRequires should fail with error in Panic mode")
	}

	expected := "contract violation: requires failed in foo at test.ail:10: x >= 0"
	if err.Error() != expected {
		t.Errorf("error message = %q, want %q", err.Error(), expected)
	}
}

func TestCheckEnsures_ReportMode(t *testing.T) {
	ctx := NewContractContextWithMode(ContractModeReport)
	ctx.SetFunction("bar")

	// Both passes and failures should be recorded
	err1 := ctx.CheckEnsures(true, "result > 0", "test.ail:20")
	err2 := ctx.CheckEnsures(false, "result < 100", "test.ail:21")

	if err1 != nil || err2 != nil {
		t.Error("Report mode should not return errors")
	}

	output := ctx.Collect()
	if output.TotalCount != 2 {
		t.Errorf("expected 2 checks, got %d", output.TotalCount)
	}
	if output.PassCount != 1 {
		t.Errorf("expected 1 pass, got %d", output.PassCount)
	}
	if output.FailCount != 1 {
		t.Errorf("expected 1 fail, got %d", output.FailCount)
	}
}

func TestCheckInvariant_OffMode(t *testing.T) {
	ctx := NewContractContextWithMode(ContractModeOff)
	ctx.SetFunction("invariant_check")

	// In Off mode, nothing is recorded
	err := ctx.CheckInvariant(false, "invariant violated", "test.ail:30")
	if err != nil {
		t.Error("Off mode should not return errors")
	}

	output := ctx.Collect()
	if output.TotalCount != 0 {
		t.Errorf("expected 0 checks in Off mode, got %d", output.TotalCount)
	}
}

func TestHasViolations(t *testing.T) {
	ctx := NewContractContextWithMode(ContractModeReport)
	ctx.SetFunction("test")

	if ctx.HasViolations() {
		t.Error("should have no violations initially")
	}

	ctx.CheckRequires(true, "ok", "test.ail:1")
	if ctx.HasViolations() {
		t.Error("should have no violations after passing check")
	}

	ctx.CheckRequires(false, "fail", "test.ail:2")
	if !ctx.HasViolations() {
		t.Error("should have violations after failing check")
	}
}

func TestViolations(t *testing.T) {
	ctx := NewContractContextWithMode(ContractModeReport)
	ctx.SetFunction("test")

	ctx.CheckRequires(true, "pass1", "test.ail:1")
	ctx.CheckRequires(false, "fail1", "test.ail:2")
	ctx.CheckEnsures(false, "fail2", "test.ail:3")
	ctx.CheckEnsures(true, "pass2", "test.ail:4")

	violations := ctx.Violations()
	if len(violations) != 2 {
		t.Errorf("expected 2 violations, got %d", len(violations))
	}

	// Check violation details
	if violations[0].Message != "fail1" {
		t.Errorf("first violation should be fail1, got %q", violations[0].Message)
	}
	if violations[1].Message != "fail2" {
		t.Errorf("second violation should be fail2, got %q", violations[1].Message)
	}
}

func TestViolationsByKind(t *testing.T) {
	ctx := NewContractContextWithMode(ContractModeReport)
	ctx.SetFunction("test")

	ctx.CheckRequires(false, "req1", "test.ail:1")
	ctx.CheckRequires(false, "req2", "test.ail:2")
	ctx.CheckEnsures(false, "ens1", "test.ail:3")
	ctx.CheckInvariant(false, "inv1", "test.ail:4")

	reqViolations := ctx.ViolationsByKind(ast.RequiresKind)
	if len(reqViolations) != 2 {
		t.Errorf("expected 2 requires violations, got %d", len(reqViolations))
	}

	ensViolations := ctx.ViolationsByKind(ast.EnsuresKind)
	if len(ensViolations) != 1 {
		t.Errorf("expected 1 ensures violation, got %d", len(ensViolations))
	}

	invViolations := ctx.ViolationsByKind(ast.InvariantKind)
	if len(invViolations) != 1 {
		t.Errorf("expected 1 invariant violation, got %d", len(invViolations))
	}
}

func TestReset(t *testing.T) {
	ctx := NewContractContextWithMode(ContractModeReport)
	ctx.SetFunction("test")

	ctx.CheckRequires(false, "fail", "test.ail:1")
	if !ctx.HasViolations() {
		t.Error("should have violations before reset")
	}

	ctx.Reset()

	if ctx.HasViolations() {
		t.Error("should have no violations after reset")
	}

	output := ctx.Collect()
	if output.TotalCount != 0 {
		t.Errorf("expected 0 checks after reset, got %d", output.TotalCount)
	}

	// Mode should be preserved
	if ctx.Mode != ContractModeReport {
		t.Errorf("mode should be preserved after reset, got %v", ctx.Mode)
	}
}

func TestSetMode(t *testing.T) {
	ctx := NewContractContext()

	if ctx.Mode != ContractModePanic {
		t.Errorf("expected initial mode Panic, got %v", ctx.Mode)
	}

	ctx.SetMode(ContractModeOff)
	if ctx.Mode != ContractModeOff {
		t.Errorf("expected Off mode after SetMode, got %v", ctx.Mode)
	}
}

func TestContractCheck_Function(t *testing.T) {
	ctx := NewContractContextWithMode(ContractModeReport)

	ctx.SetFunction("func1")
	ctx.CheckRequires(false, "in func1", "test.ail:1")

	ctx.SetFunction("func2")
	ctx.CheckEnsures(false, "in func2", "test.ail:2")

	violations := ctx.Violations()
	if violations[0].Function != "func1" {
		t.Errorf("expected func1, got %q", violations[0].Function)
	}
	if violations[1].Function != "func2" {
		t.Errorf("expected func2, got %q", violations[1].Function)
	}
}

func TestCollect_Output(t *testing.T) {
	ctx := NewContractContextWithMode(ContractModeReport)
	ctx.SetFunction("test")

	ctx.CheckRequires(true, "pass", "test.ail:1")
	ctx.CheckEnsures(false, "fail", "test.ail:2")

	output := ctx.Collect()

	if output.Mode != ContractModeReport {
		t.Errorf("expected Report mode in output, got %v", output.Mode)
	}

	if len(output.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(output.Checks))
	}

	// Verify check details
	if output.Checks[0].Kind != ast.RequiresKind {
		t.Errorf("first check should be requires, got %v", output.Checks[0].Kind)
	}
	if output.Checks[0].Passed != true {
		t.Error("first check should have passed")
	}

	if output.Checks[1].Kind != ast.EnsuresKind {
		t.Errorf("second check should be ensures, got %v", output.Checks[1].Kind)
	}
	if output.Checks[1].Passed != false {
		t.Error("second check should have failed")
	}
}
