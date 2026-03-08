package runtime

import (
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// helper: create a ModuleRuntime rooted at the project directory
func newTestRuntime(t *testing.T) *ModuleRuntime {
	t.Helper()
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	rt := NewModuleRuntime(testPath)
	rt.evaluator.SetExperimentalBinopShim(true)
	return rt
}

// helper: call an exported function with a single int argument
func callIntFunc(t *testing.T, rt *ModuleRuntime, fn *eval.FunctionValue, arg int) int {
	t.Helper()
	result, err := rt.evaluator.CallFunction(fn, []eval.Value{
		&eval.IntValue{Value: arg},
	})
	if err != nil {
		t.Fatalf("CallFunction failed: %v", err)
	}
	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("Expected IntValue, got %T: %v", result, result)
	}
	return intVal.Value
}

// helper: get an exported FunctionValue from a module instance
func getExportedFunc(t *testing.T, inst *ModuleInstance, name string) *eval.FunctionValue {
	t.Helper()
	val, err := inst.GetExport(name)
	if err != nil {
		t.Fatalf("Failed to get export %q: %v", name, err)
	}
	fn, ok := val.(*eval.FunctionValue)
	if !ok {
		t.Fatalf("Expected %q to be FunctionValue, got %T", name, val)
	}
	return fn
}

// TestModuleScopeCollision_Runtime reproduces the M-MODULE-SCOPE bug:
// Non-exported functions with the same name in different modules collide
// because ModuleRuntime used a single shared evaluator environment.
//
// scope_a defines internal `format(x) = x + 100`
// scope_b defines internal `format(x) = x + 200`
// scope_main imports transformA (calls scope_a.format) and transformB (calls scope_b.format).
//
// Before fix: testA(1) returned 201 (scope_b's format overwrote scope_a's).
// After fix: testA(1) returns 101 (each module's internals are isolated).
func TestModuleScopeCollision_Runtime(t *testing.T) {
	rt := newTestRuntime(t)

	inst, err := rt.LoadAndEvaluate("tests/runtime_integration/scope_main")
	if err != nil {
		t.Fatalf("Failed to load scope_main: %v", err)
	}

	testAFn := getExportedFunc(t, inst, "testA")
	testBFn := getExportedFunc(t, inst, "testB")

	gotA := callIntFunc(t, rt, testAFn, 1)
	gotB := callIntFunc(t, rt, testBFn, 1)

	t.Logf("testA(1) = %d (want 101)", gotA)
	t.Logf("testB(1) = %d (want 201)", gotB)

	if gotA != 101 {
		t.Errorf("testA(1) = %d, want 101 — scope_b's format() leaked into scope_a", gotA)
	}
	if gotB != 201 {
		t.Errorf("testB(1) = %d, want 201", gotB)
	}
}

// TestModuleScopeCollision_ThreeModules verifies isolation with three modules
// that all define the same internal function name `format`.
func TestModuleScopeCollision_ThreeModules(t *testing.T) {
	rt := newTestRuntime(t)

	// Load scope_a, scope_b, scope_c individually
	instA, err := rt.LoadAndEvaluate("tests/runtime_integration/scope_a")
	if err != nil {
		t.Fatalf("Failed to load scope_a: %v", err)
	}
	instB, err := rt.LoadAndEvaluate("tests/runtime_integration/scope_b")
	if err != nil {
		t.Fatalf("Failed to load scope_b: %v", err)
	}
	instC, err := rt.LoadAndEvaluate("tests/runtime_integration/scope_c")
	if err != nil {
		t.Fatalf("Failed to load scope_c: %v", err)
	}

	fnA := getExportedFunc(t, instA, "transformA")
	fnB := getExportedFunc(t, instB, "transformB")
	fnC := getExportedFunc(t, instC, "transformC")

	gotA := callIntFunc(t, rt, fnA, 1)
	gotB := callIntFunc(t, rt, fnB, 1)
	gotC := callIntFunc(t, rt, fnC, 1)

	t.Logf("transformA(1)=%d transformB(1)=%d transformC(1)=%d", gotA, gotB, gotC)

	if gotA != 101 {
		t.Errorf("transformA(1) = %d, want 101", gotA)
	}
	if gotB != 201 {
		t.Errorf("transformB(1) = %d, want 201", gotB)
	}
	if gotC != 301 {
		t.Errorf("transformC(1) = %d, want 301", gotC)
	}
}

// TestModuleScopeCollision_ClosureAfterLoad verifies that closures created in
// module_a still reference module_a's internal `format` even after module_b
// and module_c are loaded later (closure capture is by value, not affected
// by subsequent module loads).
func TestModuleScopeCollision_ClosureAfterLoad(t *testing.T) {
	rt := newTestRuntime(t)

	// Load scope_a first, get its closure
	instA, err := rt.LoadAndEvaluate("tests/runtime_integration/scope_a")
	if err != nil {
		t.Fatalf("Failed to load scope_a: %v", err)
	}
	fnA := getExportedFunc(t, instA, "transformA")

	// Call BEFORE loading other modules — should use scope_a's format
	gotBefore := callIntFunc(t, rt, fnA, 5)

	// Now load scope_b and scope_c (which also define `format`)
	_, err = rt.LoadAndEvaluate("tests/runtime_integration/scope_b")
	if err != nil {
		t.Fatalf("Failed to load scope_b: %v", err)
	}
	_, err = rt.LoadAndEvaluate("tests/runtime_integration/scope_c")
	if err != nil {
		t.Fatalf("Failed to load scope_c: %v", err)
	}

	// Call AFTER loading other modules — should STILL use scope_a's format
	gotAfter := callIntFunc(t, rt, fnA, 5)

	t.Logf("transformA(5) before=%d after=%d (want 105)", gotBefore, gotAfter)

	if gotBefore != 105 {
		t.Errorf("transformA(5) before other loads = %d, want 105", gotBefore)
	}
	if gotAfter != 105 {
		t.Errorf("transformA(5) after loading scope_b/scope_c = %d, want 105 — closure capture broken", gotAfter)
	}
}
