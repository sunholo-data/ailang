package repl

import (
	"os"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TestStringEqDispatchInNonExportedHelper reproduces Bug 2 from M-WASM-DICTIONARY-DISPATCH:
// Non-exported helper functions with lambdas using == on strings dispatch eq_Int
// instead of eq_String in the WASM/ModuleRegistry path.
//
// Root cause: Without monomorphization, the lambda inside the helper remains
// polymorphic, and operator lowering defaults to eq_Int for the == operator.
func TestStringEqDispatchInNonExportedHelper(t *testing.T) {
	reg := NewModuleRegistry()

	// Load std/option first (dependency)
	loadStdModule(t, reg, "std/option")
	loadStdModule(t, reg, "std/result")
	loadStdModule(t, reg, "std/list")

	// Module with a non-exported helper that uses == on strings inside a lambda
	code := `module test/eq_dispatch

import std/list (any)

-- Non-exported helper — was crashing with eq_Int in WASM
pure func hasMatch(xs: [string], target: string) -> bool =
  any(\x. x == target, xs)

export pure func test(target: string) -> bool =
  hasMatch(["hello", "world", "foo"], target)
`
	_, err := reg.LoadModule("test/eq_dispatch", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	// Test: "world" should be found
	result, err := reg.InvokeExport("test/eq_dispatch", "test",
		[]eval.Value{&eval.StringValue{Value: "world"}})
	if err != nil {
		t.Fatalf("InvokeExport failed (string == dispatched to eq_Int?): %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T: %v", result, result)
	}
	if !boolVal.Value {
		t.Error("expected true for hasMatch(['hello','world','foo'], 'world')")
	}

	// Test: "missing" should not be found
	result2, err := reg.InvokeExport("test/eq_dispatch", "test",
		[]eval.Value{&eval.StringValue{Value: "missing"}})
	if err != nil {
		t.Fatalf("InvokeExport failed for negative case: %v", err)
	}

	boolVal2, ok := result2.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T: %v", result2, result2)
	}
	if boolVal2.Value {
		t.Error("expected false for hasMatch(['hello','world','foo'], 'missing')")
	}
}

// TestStringEqWithLocalAny tests == on strings inside a lambda passed to a locally-defined any.
func TestStringEqWithLocalAny(t *testing.T) {
	reg := NewModuleRegistry()

	code := `module test/local_any

pure func myAny(p: (string) -> bool, xs: [string]) -> bool {
  match xs {
    [] => false,
    [x, ...rest] => if p(x) then true else myAny(p, rest)
  }
}

export pure func test(target: string) -> bool =
  myAny(\x. x == target, ["hello", "world", "foo"])
`
	_, err := reg.LoadModule("test/local_any", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	result, err := reg.InvokeExport("test/local_any", "test",
		[]eval.Value{&eval.StringValue{Value: "world"}})
	if err != nil {
		t.Fatalf("InvokeExport failed: %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T: %v", result, result)
	}
	if !boolVal.Value {
		t.Error("expected true for myAny with 'world'")
	}
}

// TestStringEqInlinedInExport verifies that the inlined version works (control test).
// This worked even before the fix — confirms the issue was specific to helper extraction.
func TestStringEqInlinedInExport(t *testing.T) {
	reg := NewModuleRegistry()

	loadStdModule(t, reg, "std/option")
	loadStdModule(t, reg, "std/result")
	loadStdModule(t, reg, "std/list")

	code := `module test/eq_inline

import std/list (any)

export pure func test(target: string) -> bool =
  any(\x. x == target, ["hello", "world", "foo"])
`
	_, err := reg.LoadModule("test/eq_inline", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	result, err := reg.InvokeExport("test/eq_inline", "test",
		[]eval.Value{&eval.StringValue{Value: "world"}})
	if err != nil {
		t.Fatalf("InvokeExport failed: %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T: %v", result, result)
	}
	if !boolVal.Value {
		t.Error("expected true for inlined any with 'world'")
	}
}

// TestIntEqDispatchInHelper verifies that int == also works correctly in helpers.
func TestIntEqDispatchInHelper(t *testing.T) {
	reg := NewModuleRegistry()

	loadStdModule(t, reg, "std/option")
	loadStdModule(t, reg, "std/result")
	loadStdModule(t, reg, "std/list")

	code := `module test/eq_int

import std/list (any)

pure func hasNum(xs: [int], target: int) -> bool =
  any(\x. x == target, xs)

export pure func test(target: int) -> bool =
  hasNum([1, 2, 3], target)
`
	_, err := reg.LoadModule("test/eq_int", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	result, err := reg.InvokeExport("test/eq_int", "test",
		[]eval.Value{&eval.IntValue{Value: 2}})
	if err != nil {
		t.Fatalf("InvokeExport failed: %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T: %v", result, result)
	}
	if !boolVal.Value {
		t.Error("expected true for hasNum([1,2,3], 2)")
	}
}

// TestDualLengthImportNoShadowing reproduces Bug 1 from M-WASM-DICTIONARY-DISPATCH:
// When a module imports length from both std/string and std/list (aliased),
// the WASM runtime should dispatch to the correct version based on type.
//
// Root cause: Without monomorphization, both functions remain as unqualified
// "length" and the last import wins, causing type mismatches.
func TestDualLengthImportNoShadowing(t *testing.T) {
	reg := NewModuleRegistry()

	// Load real stdlib modules in dependency order
	loadStdModule(t, reg, "std/option")
	loadStdModule(t, reg, "std/result")
	loadStdModule(t, reg, "std/list")
	loadStdModule(t, reg, "std/string")

	// Module that imports length from both std/string and std/list
	code := `module test/dual_length

import std/string (length)
import std/list (length as listLength)

export pure func stringLen(s: string) -> int = length(s)
export pure func listLen(xs: [int]) -> int = listLength(xs)
`
	_, err := reg.LoadModule("test/dual_length", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	// Test string length
	result1, err := reg.InvokeExport("test/dual_length", "stringLen",
		[]eval.Value{&eval.StringValue{Value: "hello"}})
	if err != nil {
		t.Fatalf("stringLen InvokeExport failed: %v", err)
	}
	intVal1, ok := result1.(*eval.IntValue)
	if !ok {
		t.Fatalf("stringLen: expected IntValue, got %T: %v", result1, result1)
	}
	if intVal1.Value != 5 {
		t.Errorf("stringLen('hello') = %d, want 5", intVal1.Value)
	}

	// Test list length
	listArg := &eval.ListValue{
		Elements: []eval.Value{
			&eval.IntValue{Value: 10},
			&eval.IntValue{Value: 20},
			&eval.IntValue{Value: 30},
		},
	}
	result2, err := reg.InvokeExport("test/dual_length", "listLen",
		[]eval.Value{listArg})
	if err != nil {
		t.Fatalf("listLen InvokeExport failed (length shadowing?): %v", err)
	}
	intVal2, ok := result2.(*eval.IntValue)
	if !ok {
		t.Fatalf("listLen: expected IntValue, got %T: %v", result2, result2)
	}
	if intVal2.Value != 3 {
		t.Errorf("listLen([10,20,30]) = %d, want 3", intVal2.Value)
	}
}

// TestStringEqInLambdaWithoutHOF tests string == inside a lambda without HOFs.
func TestStringEqInLambdaWithoutHOF(t *testing.T) {
	reg := NewModuleRegistry()

	code := `module test/lambda_eq

pure func checker(x: string) -> bool = x == "hello"

export pure func check(s: string) -> bool = checker(s)
`
	_, err := reg.LoadModule("test/lambda_eq", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	result, err := reg.InvokeExport("test/lambda_eq", "check",
		[]eval.Value{&eval.StringValue{Value: "hello"}})
	if err != nil {
		t.Fatalf("InvokeExport failed: %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T: %v", result, result)
	}
	if !boolVal.Value {
		t.Error("expected true")
	}
}

// TestSimpleStringEq tests basic string == without any HOF involvement.
func TestSimpleStringEq(t *testing.T) {
	reg := NewModuleRegistry()

	code := `module test/simple_eq

export pure func isHello(s: string) -> bool = s == "hello"
`
	_, err := reg.LoadModule("test/simple_eq", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	result, err := reg.InvokeExport("test/simple_eq", "isHello",
		[]eval.Value{&eval.StringValue{Value: "hello"}})
	if err != nil {
		t.Fatalf("InvokeExport failed: %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T: %v", result, result)
	}
	if !boolVal.Value {
		t.Error("expected true for isHello('hello')")
	}
}

// loadStdModule loads a real stdlib module from disk into the registry.
func loadStdModule(t *testing.T, reg *ModuleRegistry, modName string) {
	t.Helper()
	filename := "../../std/" + modName[len("std/"):] + ".ail"
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", filename, err)
	}
	_, err = reg.LoadModule(modName, string(content))
	if err != nil {
		t.Fatalf("Failed to load %s: %v", modName, err)
	}
}
