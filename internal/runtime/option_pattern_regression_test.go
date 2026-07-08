package runtime

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TestIntegration_UnimportedNullaryCtorPattern_Issue323 guards against the
// silent-catch-all pattern bug: a bare uppercase identifier in pattern
// position whose constructor is not in scope (Option's None when only
// `std/list (nth)` is imported) must elaborate to a nullary constructor
// pattern, NOT a variable pattern that matches everything.
//
// Before the fix, `match nth(xs, pc) { None => steps, Some(v) => ... }`
// always took the None arm, so the walk below returned 0 instead of 3.
func TestIntegration_UnimportedNullaryCtorPattern_Issue323(t *testing.T) {
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	rt := NewModuleRuntime(testPath)
	inst, err := rt.LoadAndEvaluate("tests/runtime_integration/option_pattern_unimported")
	if err != nil {
		t.Fatalf("Failed to load and evaluate module: %v", err)
	}

	result, err := CallEntrypoint(rt, inst, "run", []eval.Value{})
	if err != nil {
		t.Fatalf("CallEntrypoint failed: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("Expected IntValue, got %T", result)
	}
	if intVal.Value != 3 {
		t.Errorf("count([7,8,9], 0, 0) = %d, want 3 (None pattern matched Some values — see #323)", intVal.Value)
	}
}
