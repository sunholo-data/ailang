package runtime

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-PRELUDE-OPTION-RESULT: entry modules can construct + match Option/Result
// with NO explicit `import std/option` / `import std/result`. These tests load
// and evaluate no-import fixtures end-to-end through the runtime — exercising
// the implicit-import injection at the loader AND the constructor resolution
// path in evaluateModule (the runtime side the plan flags as R1).
func TestPreludeOptionNoImport_LoadsAndEvaluates(t *testing.T) {
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	rt := NewModuleRuntime(testPath)

	inst, err := rt.LoadAndEvaluate("tests/runtime_integration/prelude_option_noimport")
	if err != nil {
		t.Fatalf("no-import Option module failed to load/evaluate: %v", err)
	}
	if !inst.IsEvaluated() {
		t.Error("expected no-import Option module to be evaluated")
	}
	// The implicit std/option must have been loaded as a dependency.
	if rt.GetInstance("std/option") == nil {
		t.Error("std/option was not implicitly loaded for the entry module")
	}
	if _, err := inst.GetExport("main"); err != nil {
		t.Fatalf("main export missing: %v", err)
	}
}

func TestPreludeResultNoImport_LoadsAndEvaluates(t *testing.T) {
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	rt := NewModuleRuntime(testPath)

	inst, err := rt.LoadAndEvaluate("tests/runtime_integration/prelude_result_noimport")
	if err != nil {
		t.Fatalf("no-import Result module failed to load/evaluate: %v", err)
	}
	if !inst.IsEvaluated() {
		t.Error("expected no-import Result module to be evaluated")
	}
	if rt.GetInstance("std/result") == nil {
		t.Error("std/result was not implicitly loaded for the entry module")
	}
	mainVal, err := inst.GetExport("main")
	if err != nil {
		t.Fatalf("main export missing: %v", err)
	}
	if _, ok := mainVal.(*eval.FunctionValue); !ok {
		t.Fatalf("expected main to be a FunctionValue, got %T", mainVal)
	}
}
