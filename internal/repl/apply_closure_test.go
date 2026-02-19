package repl

import (
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// TestApplyClosure_ModuleSiblingBinding tests that a closure exported from a module
// can reference sibling bindings (same-module functions) when invoked via ApplyClosure.
// This is the pattern used by WASM bridge: JS invokes an AILANG closure via js.FuncOf.
func TestApplyClosure_ModuleSiblingBinding(t *testing.T) {
	r := New()
	registry := NewModuleRegistry()
	r.SetRegistry(registry)

	// Load a module with two functions: helper and callback (callback references helper)
	// Uses proper module syntax with export pure func
	moduleCode := `module test/closure_env

export pure func helper(x: int) -> int { x + 1 }

export pure func callback(x: int) -> int { helper(x) }
`
	exports, err := registry.LoadModule("test/closure_env", moduleCode)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	// Verify callback was exported
	found := false
	for _, name := range exports {
		if name == "callback" {
			found = true
		}
	}
	if !found {
		t.Fatalf("callback not in exports: %v", exports)
	}

	// First verify InvokeExport works (this path creates its own RegistryResolver)
	invokeResult, err := registry.InvokeExport("test/closure_env", "callback", []eval.Value{&eval.IntValue{Value: 5}})
	if err != nil {
		t.Fatalf("InvokeExport failed: %v", err)
	}
	intVal, ok := invokeResult.(*eval.IntValue)
	if !ok || intVal.Value != 6 {
		t.Fatalf("InvokeExport: expected 6, got %v (%T)", invokeResult, invokeResult)
	}
	t.Log("InvokeExport works for callback(5) = 6")

	// Now test via ApplyClosure (same path as WASM js.FuncOf bridge)
	export, err := registry.GetExport("test/closure_env", "callback")
	if err != nil {
		t.Fatalf("GetExport failed: %v", err)
	}

	fn, ok := export.Value.(eval.Value)
	if !ok {
		t.Fatalf("export.Value is not eval.Value, got %T", export.Value)
	}
	result, err := r.ApplyClosure(fn, []eval.Value{&eval.IntValue{Value: 5}})
	if err != nil {
		t.Fatalf("ApplyClosure failed (sibling binding not resolved): %v", err)
	}

	intVal2, ok2 := result.(*eval.IntValue)
	if !ok2 {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if intVal2.Value != 6 {
		t.Fatalf("expected 6, got %d", intVal2.Value)
	}
}

// TestApplyClosure_ModuleImportedFunction tests that a closure exported from a module
// can reference imported functions (cross-module VarGlobal references) when invoked via ApplyClosure.
// This reproduces the bug: REPL evaluator only has BuiltinOnlyResolver, not RegistryResolver.
func TestApplyClosure_ModuleImportedFunction(t *testing.T) {
	r := New()
	registry := NewModuleRegistry()
	r.SetRegistry(registry)

	// Load a helper module first
	helperModuleCode := `module test/helpers

export pure func double(x: int) -> int { x + x }
`
	_, err := registry.LoadModule("test/helpers", helperModuleCode)
	if err != nil {
		t.Fatalf("LoadModule helpers failed: %v", err)
	}

	// Load module that imports from test/helpers and creates a closure using it
	moduleCode := `module test/importer

import test/helpers (double)

export pure func makeDoubler(x: int) -> int { double(x) }
`
	exports, err := registry.LoadModule("test/importer", moduleCode)
	if err != nil {
		t.Fatalf("LoadModule importer failed: %v", err)
	}

	// Verify makeDoubler was exported
	found := false
	for _, name := range exports {
		if name == "makeDoubler" {
			found = true
		}
	}
	if !found {
		t.Fatalf("makeDoubler not in exports: %v", exports)
	}

	// First verify InvokeExport works
	invokeResult, err := registry.InvokeExport("test/importer", "makeDoubler", []eval.Value{&eval.IntValue{Value: 5}})
	if err != nil {
		t.Fatalf("InvokeExport failed: %v", err)
	}
	intVal, ok := invokeResult.(*eval.IntValue)
	if !ok || intVal.Value != 10 {
		t.Fatalf("InvokeExport: expected 10, got %v (%T)", invokeResult, invokeResult)
	}
	t.Log("InvokeExport works for makeDoubler(5) = 10")

	// Now test via ApplyClosure (same path as WASM js.FuncOf bridge)
	export, err := registry.GetExport("test/importer", "makeDoubler")
	if err != nil {
		t.Fatalf("GetExport failed: %v", err)
	}

	fn, ok := export.Value.(eval.Value)
	if !ok {
		t.Fatalf("export.Value is not eval.Value, got %T", export.Value)
	}
	result, err := r.ApplyClosure(fn, []eval.Value{&eval.IntValue{Value: 5}})
	if err != nil {
		t.Fatalf("ApplyClosure failed (imported function not resolved): %v", err)
	}

	intVal2, ok2 := result.(*eval.IntValue)
	if !ok2 {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if intVal2.Value != 10 {
		t.Fatalf("expected 10, got %d", intVal2.Value)
	}
}

func TestApplyClosure_NilFunction(t *testing.T) {
	r := New()
	_, err := r.ApplyClosure(nil, []eval.Value{})
	if err == nil {
		t.Fatal("expected error for nil function")
	}
}

func TestApplyClosure_NonFunction(t *testing.T) {
	r := New()
	_, err := r.ApplyClosure(&eval.IntValue{Value: 42}, []eval.Value{&eval.IntValue{Value: 1}})
	if err == nil {
		t.Fatal("expected error for non-function value")
	}
}

func TestApplyClosure_BuiltinFunction(t *testing.T) {
	r := New()
	// Create a simple builtin that returns its argument
	identity := &eval.BuiltinFunction{
		Name: "identity",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return args[0], nil
		},
	}
	result, err := r.ApplyClosure(identity, []eval.Value{&eval.IntValue{Value: 42}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if intVal.Value != 42 {
		t.Fatalf("expected 42, got %d", intVal.Value)
	}
}

func TestApplyClosure_ZeroArgs(t *testing.T) {
	r := New()
	// A "thunk" that ignores its arg and returns a constant
	thunk := &eval.BuiltinFunction{
		Name: "thunk",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return &eval.StringValue{Value: "hello"}, nil
		},
	}
	result, err := r.ApplyClosure(thunk, []eval.Value{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}
	if strVal.Value != "hello" {
		t.Fatalf("expected 'hello', got %q", strVal.Value)
	}
}
