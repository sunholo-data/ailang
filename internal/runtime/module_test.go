package runtime

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/loader"
)

func TestNewModuleInstance(t *testing.T) {
	// Create a mock LoadedModule
	ifaceInst := iface.NewIface("test/module")

	loaded := &loader.LoadedModule{
		Path:    "test/module",
		Iface:   ifaceInst,
		Core:    &core.Program{},
		Imports: []string{"test/dep"},
	}

	// Create ModuleInstance
	inst := NewModuleInstance(loaded)

	// Verify basic fields
	if inst.Path != "test/module" {
		t.Errorf("Expected path 'test/module', got '%s'", inst.Path)
	}

	if inst.Iface == nil {
		t.Error("Expected Iface to be set")
	}

	if inst.Core == nil {
		t.Error("Expected Core to be set")
	}

	// Verify maps are initialized
	if inst.Bindings == nil {
		t.Error("Expected Bindings map to be initialized")
	}

	if inst.Exports == nil {
		t.Error("Expected Exports map to be initialized")
	}

	if inst.Imports == nil {
		t.Error("Expected Imports map to be initialized")
	}

	// Verify maps are empty initially
	if len(inst.Bindings) != 0 {
		t.Errorf("Expected Bindings to be empty, got %d entries", len(inst.Bindings))
	}

	if len(inst.Exports) != 0 {
		t.Errorf("Expected Exports to be empty, got %d entries", len(inst.Exports))
	}

	if len(inst.Imports) != 0 {
		t.Errorf("Expected Imports to be empty, got %d entries", len(inst.Imports))
	}
}

func TestModuleInstance_GetExport(t *testing.T) {
	inst := &ModuleInstance{
		Path:    "test/module",
		Exports: make(map[string]eval.Value),
	}

	// Test: Export not found, no exports
	_, err := inst.GetExport("main")
	if err == nil {
		t.Error("Expected error when getting non-existent export")
	}

	// Add some exports
	inst.Exports["foo"] = &eval.IntValue{Value: 42}
	inst.Exports["bar"] = &eval.IntValue{Value: 100}

	// Test: Export not found, but others exist
	_, err = inst.GetExport("main")
	if err == nil {
		t.Error("Expected error when getting non-existent export")
	}

	// Test: Export found
	val, err := inst.GetExport("foo")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	intVal, ok := val.(*eval.IntValue)
	if !ok {
		t.Errorf("Expected IntValue, got %T", val)
	}

	if intVal.Value != 42 {
		t.Errorf("Expected value 42, got %d", intVal.Value)
	}
}

func TestModuleInstance_HasExport(t *testing.T) {
	inst := &ModuleInstance{
		Path:    "test/module",
		Exports: make(map[string]eval.Value),
	}

	// Test: Export doesn't exist
	if inst.HasExport("main") {
		t.Error("Expected HasExport to return false for non-existent export")
	}

	// Add export
	inst.Exports["main"] = &eval.IntValue{Value: 42}

	// Test: Export exists
	if !inst.HasExport("main") {
		t.Error("Expected HasExport to return true for existing export")
	}

	// Test: Different export doesn't exist
	if inst.HasExport("foo") {
		t.Error("Expected HasExport to return false for different export")
	}
}

func TestModuleInstance_GetBinding(t *testing.T) {
	inst := &ModuleInstance{
		Path:     "test/module",
		Bindings: make(map[string]eval.Value),
		Exports:  make(map[string]eval.Value),
	}

	// Test: Binding not found
	_, err := inst.GetBinding("foo")
	if err == nil {
		t.Error("Expected error when getting non-existent binding")
	}

	// Add private binding (not exported)
	inst.Bindings["helper"] = &eval.IntValue{Value: 10}

	// Add exported binding
	inst.Bindings["main"] = &eval.IntValue{Value: 42}
	inst.Exports["main"] = &eval.IntValue{Value: 42}

	// Test: Get private binding
	val, err := inst.GetBinding("helper")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	intVal, ok := val.(*eval.IntValue)
	if !ok {
		t.Errorf("Expected IntValue, got %T", val)
	}

	if intVal.Value != 10 {
		t.Errorf("Expected value 10, got %d", intVal.Value)
	}

	// Test: Get exported binding (should work via GetBinding too)
	val, err = inst.GetBinding("main")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	intVal, ok = val.(*eval.IntValue)
	if !ok {
		t.Errorf("Expected IntValue, got %T", val)
	}

	if intVal.Value != 42 {
		t.Errorf("Expected value 42, got %d", intVal.Value)
	}
}

func TestModuleInstance_ListExports(t *testing.T) {
	inst := &ModuleInstance{
		Path:    "test/module",
		Exports: make(map[string]eval.Value),
	}

	// Test: No exports
	exports := inst.ListExports()
	if len(exports) != 0 {
		t.Errorf("Expected 0 exports, got %d", len(exports))
	}

	// Add exports
	inst.Exports["foo"] = &eval.IntValue{Value: 1}
	inst.Exports["bar"] = &eval.IntValue{Value: 2}
	inst.Exports["baz"] = &eval.IntValue{Value: 3}

	// Test: List exports
	exports = inst.ListExports()
	if len(exports) != 3 {
		t.Errorf("Expected 3 exports, got %d", len(exports))
	}

	// Verify all exports are present (order doesn't matter)
	found := make(map[string]bool)
	for _, name := range exports {
		found[name] = true
	}

	if !found["foo"] || !found["bar"] || !found["baz"] {
		t.Errorf("Missing exports, got: %v", exports)
	}
}

func TestModuleInstance_IsEvaluated(t *testing.T) {
	inst := &ModuleInstance{
		Path:     "test/module",
		Bindings: make(map[string]eval.Value),
	}

	// Test: Not evaluated initially
	if inst.IsEvaluated() {
		t.Error("Expected IsEvaluated to return false initially")
	}

	// Add bindings (simulating evaluation)
	inst.Bindings["foo"] = &eval.IntValue{Value: 42}

	// Test: Evaluated after adding bindings
	if !inst.IsEvaluated() {
		t.Error("Expected IsEvaluated to return true after adding bindings")
	}
}

func TestModuleInstance_GetEvaluationError(t *testing.T) {
	inst := &ModuleInstance{
		Path:     "test/module",
		Bindings: make(map[string]eval.Value),
	}

	// Test: No error initially
	if inst.GetEvaluationError() != nil {
		t.Error("Expected no evaluation error initially")
	}

	// Set error
	inst.initErr = eval.NewRuntimeError("TEST_ERROR", "test error", nil)

	// Test: Error returned
	if inst.GetEvaluationError() == nil {
		t.Error("Expected evaluation error to be returned")
	}
}
