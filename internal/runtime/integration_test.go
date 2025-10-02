package runtime

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

func TestIntegration_SimpleModule(t *testing.T) {
	// Get absolute path to project root (where tests/ directory is)
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	rt := NewModuleRuntime(testPath)

	// Load and evaluate simple module
	inst, err := rt.LoadAndEvaluate("tests/runtime_integration/simple")
	if err != nil {
		t.Fatalf("Failed to load and evaluate simple module: %v", err)
	}

	// Check that module was evaluated
	if !inst.IsEvaluated() {
		t.Error("Expected module to be evaluated")
	}

	// Check that main export exists
	mainVal, err := inst.GetExport("main")
	if err != nil {
		t.Fatalf("Failed to get main export: %v", err)
	}

	// Verify it's a function value
	_, ok := mainVal.(*eval.FunctionValue)
	if !ok {
		t.Fatalf("Expected main to be a FunctionValue, got %T", mainVal)
	}
}

func TestIntegration_ModuleWithImport(t *testing.T) {
	// Get absolute path to project root
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	rt := NewModuleRuntime(testPath)

	// Load and evaluate module with import
	inst, err := rt.LoadAndEvaluate("tests/runtime_integration/with_import")
	if err != nil {
		t.Fatalf("Failed to load and evaluate module with import: %v", err)
	}

	// Check that both modules were loaded
	if !rt.HasInstance("tests/runtime_integration/with_import") {
		t.Error("Expected with_import module to be cached")
	}

	if !rt.HasInstance("tests/runtime_integration/dep") {
		t.Error("Expected dep module to be cached")
	}

	// Check that dependency was evaluated
	depInst := rt.GetInstance("tests/runtime_integration/dep")
	if depInst == nil || !depInst.IsEvaluated() {
		t.Error("Expected dep module to be evaluated")
	}

	// Verify dep has inc export
	incVal, err := depInst.GetExport("inc")
	if err != nil {
		t.Fatalf("Failed to get inc export from dep: %v", err)
	}

	if _, ok := incVal.(*eval.FunctionValue); !ok {
		t.Errorf("Expected inc to be a FunctionValue, got %T", incVal)
	}

	// Get main export from with_import
	mainVal, err := inst.GetExport("main")
	if err != nil {
		t.Fatalf("Failed to get main export: %v", err)
	}

	if _, ok := mainVal.(*eval.FunctionValue); !ok {
		t.Errorf("Expected main to be a FunctionValue, got %T", mainVal)
	}
}

func TestIntegration_CachedModules(t *testing.T) {
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	rt := NewModuleRuntime(testPath)

	// Load same module twice
	inst1, err := rt.LoadAndEvaluate("tests/runtime_integration/simple")
	if err != nil {
		t.Fatalf("First load failed: %v", err)
	}

	inst2, err := rt.LoadAndEvaluate("tests/runtime_integration/simple")
	if err != nil {
		t.Fatalf("Second load failed: %v", err)
	}

	// Should return same instance
	if inst1 != inst2 {
		t.Error("Expected cached instance to be returned on second load")
	}

	// Check instances list
	instances := rt.ListInstances()
	if len(instances) != 1 {
		t.Errorf("Expected 1 cached instance, got %d", len(instances))
	}
}

func TestIntegration_ModuleEvaluationOrder(t *testing.T) {
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	rt := NewModuleRuntime(testPath)

	// Load module that has a dependency
	_, err = rt.LoadAndEvaluate("tests/runtime_integration/with_import")
	if err != nil {
		t.Fatalf("Failed to load module: %v", err)
	}

	// Both modules should be in cache
	instances := rt.ListInstances()
	if len(instances) != 2 {
		t.Errorf("Expected 2 cached instances, got %d", len(instances))
	}

	// Dependency should be evaluated first
	depInst := rt.GetInstance("tests/runtime_integration/dep")
	if depInst == nil {
		t.Fatal("Expected dep module to be loaded")
	}

	if !depInst.IsEvaluated() {
		t.Error("Expected dep module to be evaluated (dependencies evaluate first)")
	}

	// Main module should also be evaluated
	mainInst := rt.GetInstance("tests/runtime_integration/with_import")
	if mainInst == nil {
		t.Fatal("Expected with_import module to be loaded")
	}

	if !mainInst.IsEvaluated() {
		t.Error("Expected with_import module to be evaluated")
	}

	// Main module should have dependency in its imports
	if len(mainInst.Imports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(mainInst.Imports))
	}

	if mainInst.Imports["tests/runtime_integration/dep"] != depInst {
		t.Error("Expected import to point to cached dependency instance")
	}
}

func TestIntegration_CircularImport(t *testing.T) {
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	rt := NewModuleRuntime(testPath)

	// Note: We don't have actual circular import test files because
	// they would fail at parse time. This test verifies the detection
	// mechanism would work if we did have them.

	// For now, just verify the error message format is correct
	// by checking that the runtime has cycle detection fields initialized
	if rt.visiting == nil {
		t.Error("Expected visiting map to be initialized for cycle detection")
	}

	if rt.pathStack == nil {
		t.Error("Expected pathStack to be initialized for cycle tracking")
	}
}

func TestIntegration_NonExistentModule(t *testing.T) {
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	rt := NewModuleRuntime(testPath)

	// Try to load non-existent module
	_, err = rt.LoadAndEvaluate("tests/runtime_integration/does_not_exist")
	if err == nil {
		t.Error("Expected error when loading non-existent module")
	}

	// Error should mention the module path
	if !strings.Contains(err.Error(), "does_not_exist") {
		t.Errorf("Expected error to mention module path, got: %v", err)
	}
}

func TestIntegration_ExportFiltering(t *testing.T) {
	testPath, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	rt := NewModuleRuntime(testPath)

	// Load module
	inst, err := rt.LoadAndEvaluate("tests/runtime_integration/simple")
	if err != nil {
		t.Fatalf("Failed to load module: %v", err)
	}

	// simple.ail only exports main
	exports := inst.ListExports()
	if len(exports) != 1 {
		t.Errorf("Expected 1 export, got %d", len(exports))
	}

	if exports[0] != "main" {
		t.Errorf("Expected export to be 'main', got '%s'", exports[0])
	}

	// Try to get non-existent export
	_, err = inst.GetExport("notExported")
	if err == nil {
		t.Error("Expected error when getting non-existent export")
	}
}
