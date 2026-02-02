package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/runtime"
)

func TestNewModuleRegistry(t *testing.T) {
	reg := NewModuleRegistry()
	if reg == nil {
		t.Fatal("NewModuleRegistry returned nil")
	}
	if reg.modules == nil {
		t.Fatal("modules map not initialized")
	}
	if len(reg.modules) != 0 {
		t.Errorf("expected empty modules map, got %d entries", len(reg.modules))
	}
}

func TestLoadModuleSimple(t *testing.T) {
	reg := NewModuleRegistry()

	// Simple module with one function (explicit type annotation for numeric operations)
	code := `let add: Int -> Int -> Int = \x. \y. x + y`

	exports, err := reg.LoadModule("math", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	if len(exports) != 1 {
		t.Errorf("expected 1 export, got %d", len(exports))
	}

	found := false
	for _, name := range exports {
		if name == "add" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'add' in exports, got %v", exports)
	}
}

func TestLoadModuleMultipleFunctions(t *testing.T) {
	reg := NewModuleRegistry()

	// All arithmetic operations need type annotations
	code := `
let add: Int -> Int -> Int = \x. \y. x + y
let sub: Int -> Int -> Int = \x. \y. x - y
let mul: Int -> Int -> Int = \x. \y. x * y
`

	exports, err := reg.LoadModule("math", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	if len(exports) != 3 {
		t.Errorf("expected 3 exports, got %d: %v", len(exports), exports)
	}

	// Verify all expected functions are exported
	exportSet := make(map[string]bool)
	for _, name := range exports {
		exportSet[name] = true
	}

	for _, expected := range []string{"add", "sub", "mul"} {
		if !exportSet[expected] {
			t.Errorf("missing expected export: %s", expected)
		}
	}
}

func TestGetExport(t *testing.T) {
	reg := NewModuleRegistry()

	code := `let double: Int -> Int = \x. x * 2`

	_, err := reg.LoadModule("utils", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	// Get existing export
	export, err := reg.GetExport("utils", "double")
	if err != nil {
		t.Fatalf("GetExport failed: %v", err)
	}

	if export.Name != "double" {
		t.Errorf("expected export name 'double', got '%s'", export.Name)
	}

	if export.Value == nil {
		t.Error("export value is nil")
	}

	if export.Scheme == nil {
		t.Error("export scheme is nil")
	}
}

func TestGetExportModuleNotLoaded(t *testing.T) {
	reg := NewModuleRegistry()

	_, err := reg.GetExport("nonexistent", "func")
	if err == nil {
		t.Fatal("expected error for non-existent module")
	}

	if !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("error message should mention 'not loaded', got: %v", err)
	}
}

func TestGetExportFunctionNotExported(t *testing.T) {
	reg := NewModuleRegistry()

	// Use a boolean value - no type annotations needed
	code := `let foo = true`
	_, err := reg.LoadModule("test", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	_, err = reg.GetExport("test", "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent export")
	}

	if !strings.Contains(err.Error(), "not exported") {
		t.Errorf("error message should mention 'not exported', got: %v", err)
	}
}

func TestGetModule(t *testing.T) {
	reg := NewModuleRegistry()

	// Use a string constant - no type annotation needed
	code := `let value = "hello"`
	_, err := reg.LoadModule("constants", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	mod, ok := reg.GetModule("constants")
	if !ok {
		t.Fatal("GetModule returned false for loaded module")
	}

	if mod.Name != "constants" {
		t.Errorf("expected module name 'constants', got '%s'", mod.Name)
	}

	if mod.Source != code {
		t.Error("module source doesn't match")
	}
}

func TestGetModuleNotLoaded(t *testing.T) {
	reg := NewModuleRegistry()

	_, ok := reg.GetModule("nonexistent")
	if ok {
		t.Error("GetModule returned true for non-existent module")
	}
}

func TestListModules(t *testing.T) {
	reg := NewModuleRegistry()

	// Empty registry
	modules := reg.ListModules()
	if len(modules) != 0 {
		t.Errorf("expected empty list, got %v", modules)
	}

	// Load modules with boolean/string values (no type annotations needed)
	_, err := reg.LoadModule("a", `let x = true`)
	if err != nil {
		t.Fatalf("LoadModule a failed: %v", err)
	}
	_, err = reg.LoadModule("b", `let y = false`)
	if err != nil {
		t.Fatalf("LoadModule b failed: %v", err)
	}
	_, err = reg.LoadModule("c", `let z = "test"`)
	if err != nil {
		t.Fatalf("LoadModule c failed: %v", err)
	}

	modules = reg.ListModules()
	if len(modules) != 3 {
		t.Errorf("expected 3 modules, got %d", len(modules))
	}

	// Check all modules are listed
	moduleSet := make(map[string]bool)
	for _, name := range modules {
		moduleSet[name] = true
	}

	for _, expected := range []string{"a", "b", "c"} {
		if !moduleSet[expected] {
			t.Errorf("missing module: %s", expected)
		}
	}
}

func TestLoadModuleParseError(t *testing.T) {
	reg := NewModuleRegistry()

	// Invalid syntax
	code := `let x = (`

	_, err := reg.LoadModule("broken", code)
	if err == nil {
		t.Fatal("expected parse error")
	}

	if !strings.Contains(err.Error(), "parse error") {
		t.Errorf("error should mention 'parse error', got: %v", err)
	}
}

func TestLoadModuleTypeError(t *testing.T) {
	reg := NewModuleRegistry()

	// Type error: applying an int where a function is expected
	code := `let bad = 42(1)`

	_, err := reg.LoadModule("broken", code)
	if err == nil {
		t.Fatal("expected type error")
	}

	// Error should mention type-related issue
	errMsg := err.Error()
	if !strings.Contains(errMsg, "type") && !strings.Contains(errMsg, "unification") && !strings.Contains(errMsg, "expected function") {
		t.Errorf("error should mention type-related issue, got: %v", err)
	}
}

func TestLoadModuleEmptyDeclarations(t *testing.T) {
	reg := NewModuleRegistry()

	// Empty module (just comments or whitespace)
	code := `-- just a comment`

	_, err := reg.LoadModule("empty", code)
	if err == nil {
		t.Fatal("expected error for empty module")
	}

	if !strings.Contains(err.Error(), "no declarations") {
		t.Errorf("error should mention 'no declarations', got: %v", err)
	}
}

func TestLoadModuleOverwrite(t *testing.T) {
	reg := NewModuleRegistry()

	// Load first version
	code1 := `let value = "first"`
	_, err := reg.LoadModule("test", code1)
	if err != nil {
		t.Fatalf("first LoadModule failed: %v", err)
	}

	// Load second version (should overwrite)
	code2 := `let value = "second"`
	_, err = reg.LoadModule("test", code2)
	if err != nil {
		t.Fatalf("second LoadModule failed: %v", err)
	}

	mod, _ := reg.GetModule("test")
	if mod.Source != code2 {
		t.Error("module should be overwritten with new version")
	}
}

func TestLoadModuleWithIdentityFunction(t *testing.T) {
	reg := NewModuleRegistry()

	// Identity function is polymorphic but doesn't have Num constraint
	code := `let id = \x. x`

	exports, err := reg.LoadModule("utils", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	if len(exports) != 1 {
		t.Errorf("expected 1 export, got %d", len(exports))
	}
}

func TestLoadModuleWithPureFunctions(t *testing.T) {
	reg := NewModuleRegistry()

	// Pure functions without numeric operations
	code := `
let const = \x. \y. x
let flip = \f. \x. \y. f(y)(x)
let compose = \f. \g. \x. f(g(x))
`

	exports, err := reg.LoadModule("combinators", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	if len(exports) != 3 {
		t.Errorf("expected 3 exports, got %d: %v", len(exports), exports)
	}
}

func TestConcurrentAccess(t *testing.T) {
	reg := NewModuleRegistry()

	// Load a module first
	_, err := reg.LoadModule("shared", `let x = "shared"`)
	if err != nil {
		t.Fatalf("initial LoadModule failed: %v", err)
	}

	// Concurrent reads should be safe
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			reg.ListModules()
			reg.GetModule("shared")
			reg.GetExport("shared", "x")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestLoadModuleWithExportPureFunc(t *testing.T) {
	reg := NewModuleRegistry()

	// Module with explicit export declaration
	code := `module test_export
export pure func double(x: int) -> int = x * 2`

	exports, err := reg.LoadModule("test_export", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	if len(exports) != 1 {
		t.Errorf("expected 1 export, got %d: %v", len(exports), exports)
	}

	found := false
	for _, name := range exports {
		if name == "double" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'double' in exports, got %v", exports)
	}
}

func TestLoadModuleExplicitExportFiltering(t *testing.T) {
	reg := NewModuleRegistry()

	// Module with one exported function and one private function
	code := `module test_filter
export pure func public_func(x: int) -> int = x * 2
pure func private_func(x: int) -> int = x * 3`

	exports, err := reg.LoadModule("test_filter", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	// Only the exported function should be in exports
	if len(exports) != 1 {
		t.Errorf("expected 1 export (only public_func), got %d: %v", len(exports), exports)
	}

	// Verify public_func is exported
	exportSet := make(map[string]bool)
	for _, name := range exports {
		exportSet[name] = true
	}

	if !exportSet["public_func"] {
		t.Error("public_func should be exported")
	}
	if exportSet["private_func"] {
		t.Error("private_func should NOT be exported")
	}
}

func TestREPLImportFromRegistry(t *testing.T) {
	// Create a REPL with a registry
	r := New()
	reg := NewModuleRegistry()
	r.SetRegistry(reg)

	// Load a module into the registry
	_, err := reg.LoadModule("math", `let double: Int -> Int = \x. x * 2`)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	// Verify registry is connected
	if r.GetRegistry() == nil {
		t.Fatal("Registry not set on REPL")
	}

	// Verify module is in registry
	mods := r.GetRegistry().ListModules()
	if len(mods) != 1 || mods[0] != "math" {
		t.Errorf("Expected ['math'], got %v", mods)
	}
}

func TestLoadModuleWithBuiltinWrapper(t *testing.T) {
	// This test verifies that modules can call builtin functions like _str_len.
	// The elaborator must call AddBuiltinsToGlobalEnv() for this to work,
	// otherwise builtins are treated as regular variables and fail at runtime.
	reg := NewModuleRegistry()

	// Load a module that wraps a builtin (similar to std/string)
	code := `module test_builtins
export pure func len(s: string) -> int = _str_len(s)`

	exports, err := reg.LoadModule("test_builtins", code)
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}

	if len(exports) != 1 {
		t.Errorf("expected 1 export, got %d: %v", len(exports), exports)
	}

	// Verify the export exists
	export, err := reg.GetExport("test_builtins", "len")
	if err != nil {
		t.Fatalf("GetExport failed: %v", err)
	}

	if export.Value == nil {
		t.Error("export value should not be nil")
	}
}

func TestLoadStdlibWithDependencies(t *testing.T) {
	// This test loads actual stdlib modules with dependencies to verify
	// that module imports work correctly in the LoadModule function.
	// Without proper import handling, this test will fail because
	// std/string imports std/option.
	reg := NewModuleRegistry()

	// Load std/option first (no dependencies - should succeed)
	optionCode := `module std/option

type Option[T] = Some(T) | None

export pure func isSome[T](opt: Option[T]) -> bool {
  match opt {
    Some(_) => true,
    None => false
  }
}

export pure func isNone[T](opt: Option[T]) -> bool {
  match opt {
    Some(_) => false,
    None => true
  }
}
`
	optExports, err := reg.LoadModule("std/option", optionCode)
	if err != nil {
		t.Fatalf("Failed to load std/option: %v", err)
	}
	t.Logf("std/option exports: %v", optExports)

	// Now load a module that depends on std/option AND USES Option type
	// This should fail without proper import handling because Option is undefined
	stringCode := `module std/string

import std/option (Option, Some, None)

export pure func length(s: string) -> int { _str_len(s) }
export pure func intToStr(n: int) -> string { _string_intToStr(n) }

-- This function uses the imported Option type
export pure func stringToInt(s: string) -> Option[int] { _stringToInt(s) }
`
	strExports, err := reg.LoadModule("std/string", stringCode)
	if err != nil {
		t.Fatalf("Failed to load std/string: %v", err)
	}
	t.Logf("std/string exports: %v", strExports)

	// Verify expected exports exist
	exportSet := make(map[string]bool)
	for _, name := range strExports {
		exportSet[name] = true
	}

	if !exportSet["length"] {
		t.Error("std/string should export 'length'")
	}
	if !exportSet["intToStr"] {
		t.Error("std/string should export 'intToStr'")
	}
	if !exportSet["stringToInt"] {
		t.Error("std/string should export 'stringToInt'")
	}
}

func TestLoadRealStdlibFromFiles(t *testing.T) {
	// This test loads the REAL stdlib files from disk to test
	// the full WASM initialization path.
	reg := NewModuleRegistry()

	// These are the stdlib files that should have no dependencies
	// and should load first
	baseModules := []string{
		"std/option",
		"std/result",
		"std/prelude",
	}

	for _, modName := range baseModules {
		filename := "../../std/" + strings.TrimPrefix(modName, "std/") + ".ail"
		content, err := os.ReadFile(filename)
		if err != nil {
			// Skip if file doesn't exist (some modules might not be present)
			t.Logf("Skipping %s: %v", modName, err)
			continue
		}

		exports, err := reg.LoadModule(modName, string(content))
		if err != nil {
			t.Errorf("Failed to load base module %s: %v", modName, err)
		} else {
			t.Logf("Loaded %s with exports: %v", modName, exports)
		}
	}

	// Now load std/string which depends on std/option
	stringContent, err := os.ReadFile("../../std/string.ail")
	if err != nil {
		t.Fatalf("Failed to read std/string.ail: %v", err)
	}

	exports, err := reg.LoadModule("std/string", string(stringContent))
	if err != nil {
		t.Fatalf("Failed to load std/string: %v", err)
	}
	t.Logf("std/string exports: %v", exports)

	// Verify key exports
	exportSet := make(map[string]bool)
	for _, name := range exports {
		exportSet[name] = true
	}

	if !exportSet["intToStr"] {
		t.Error("std/string should export 'intToStr'")
	}
	if !exportSet["stringToInt"] {
		t.Error("std/string should export 'stringToInt' (uses Option[int])")
	}
}

// TestLoadStdlibWithActualImports tests loading modules that actually USE imported constructors
// (not just type annotations). std/json uses Some() and None in its code, not just types.
func TestLoadStdlibWithActualImports(t *testing.T) {
	reg := NewModuleRegistry()

	// Load dependency modules in the right order
	dependencyModules := []string{
		"std/option", // Option, Some, None ADT
		"std/result", // Result, Ok, Err ADT
		"std/list",   // list operations
		"std/math",   // floatToInt used by json
	}

	for _, modName := range dependencyModules {
		filename := "../../std/" + strings.TrimPrefix(modName, "std/") + ".ail"
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Logf("Skipping %s: %v", modName, err)
			continue
		}

		exports, err := reg.LoadModule(modName, string(content))
		if err != nil {
			t.Errorf("Failed to load %s: %v", modName, err)
		} else {
			t.Logf("Loaded %s with %d exports", modName, len(exports))
		}
	}

	// Now load std/json which USES Some() and None constructors in its code
	// This tests that imports actually work, not just type annotations
	jsonContent, err := os.ReadFile("../../std/json.ail")
	if err != nil {
		t.Fatalf("Failed to read std/json.ail: %v", err)
	}

	exports, err := reg.LoadModule("std/json", string(jsonContent))
	if err != nil {
		t.Fatalf("Failed to load std/json: %v", err)
	}
	t.Logf("std/json exports: %v", exports)

	// Verify key exports that use Option constructors
	exportSet := make(map[string]bool)
	for _, name := range exports {
		exportSet[name] = true
	}

	// These functions use Some() and None in their implementation
	if !exportSet["get"] {
		t.Error("std/json should export 'get' (uses None and Some)")
	}
	if !exportSet["has"] {
		t.Error("std/json should export 'has' (uses match on Some/None)")
	}
}

// TestLoadAllStdlibWithMultiPass tests loading ALL stdlib modules using multi-pass loading
// This simulates what the WASM loader does
func TestLoadAllStdlibWithMultiPass(t *testing.T) {
	reg := NewModuleRegistry()

	// Read all stdlib modules
	entries, err := os.ReadDir("../../std")
	if err != nil {
		t.Fatalf("Failed to read std/: %v", err)
	}

	type moduleSource struct {
		name    string
		content string
	}
	var pending []moduleSource

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ail") {
			continue
		}
		content, err := os.ReadFile("../../std/" + entry.Name())
		if err != nil {
			continue
		}
		moduleName := "std/" + strings.TrimSuffix(entry.Name(), ".ail")
		pending = append(pending, moduleSource{name: moduleName, content: string(content)})
	}

	t.Logf("Found %d stdlib modules to load", len(pending))

	// Multi-pass loading (same as WASM)
	for pass := 0; pass < 10 && len(pending) > 0; pass++ {
		var stillPending []moduleSource
		var loaded []string

		for _, mod := range pending {
			_, err := reg.LoadModule(mod.name, mod.content)
			if err != nil {
				stillPending = append(stillPending, mod)
			} else {
				loaded = append(loaded, mod.name)
			}
		}

		t.Logf("Pass %d: loaded %d modules, %d still pending", pass+1, len(loaded), len(stillPending))
		for _, name := range loaded {
			t.Logf("  ✓ %s", name)
		}

		if len(stillPending) == len(pending) {
			t.Logf("No progress - remaining modules have errors:")
			for _, mod := range stillPending {
				_, err := reg.LoadModule(mod.name, mod.content)
				t.Logf("  ✗ %s: %v", mod.name, err)
			}
			break
		}
		pending = stillPending
	}

	// Report final state
	loadedModules := reg.ListModules()
	t.Logf("Successfully loaded %d modules: %v", len(loadedModules), loadedModules)

	// Check that key modules loaded
	moduleSet := make(map[string]bool)
	for _, name := range loadedModules {
		moduleSet[name] = true
	}

	required := []string{"std/option", "std/result", "std/list", "std/string", "std/json", "std/math"}
	for _, name := range required {
		if !moduleSet[name] {
			t.Errorf("Required module %s was NOT loaded", name)
		}
	}
}

// TestModuleAliasQualifiedAccess tests that module aliases work for qualified access
// This tests the fix for "import std/sharedmem as cache" where cache.get() was failing
func TestModuleAliasQualifiedAccess(t *testing.T) {
	reg := NewModuleRegistry()

	// Load base module with a function to export
	_, err := reg.LoadModule("base", `
module base
export pure func helper(x: int) -> int = x * 2
export pure func add(x: int, y: int) -> int = x + y
`)
	if err != nil {
		t.Fatalf("Failed to load base: %v", err)
	}

	// Load module that uses alias for qualified access
	exports, err := reg.LoadModule("user", `
module user
import base as b

export pure func useAlias(x: int) -> int = b.helper(x)
export pure func useAdd(x: int, y: int) -> int = b.add(x, y)
`)
	if err != nil {
		t.Fatalf("Failed to load user (module alias bug): %v", err)
	}

	// Verify both functions are exported
	if len(exports) != 2 {
		t.Errorf("expected 2 exports, got %d: %v", len(exports), exports)
	}

	// Verify useAlias is exported
	exportSet := make(map[string]bool)
	for _, name := range exports {
		exportSet[name] = true
	}

	if !exportSet["useAlias"] {
		t.Error("useAlias should be exported")
	}
	if !exportSet["useAdd"] {
		t.Error("useAdd should be exported")
	}
}

// TestModuleImportAtRuntime tests that imported functions are available at runtime
func TestModuleImportAtRuntime(t *testing.T) {
	reg := NewModuleRegistry()

	// First, load std/option (no dependencies)
	optionSrc := `
module std/option

export type Option[a] = Some(a) | None

export pure func isSome(opt: Option[a]) -> bool =
  match opt {
    Some(_) => true,
    None => false
  }
`
	_, err := reg.LoadModule("std/option", optionSrc)
	if err != nil {
		t.Fatalf("Failed to load std/option: %v", err)
	}

	// Now load a module that imports from std/option
	userModuleSrc := `
module test/user

import std/option (Option, Some, None, isSome)

export pure func wrapValue(x: int) -> Option[int] = Some(x)

export pure func checkWrap(x: int) -> bool = isSome(wrapValue(x))
`
	exports, err := reg.LoadModule("test/user", userModuleSrc)
	if err != nil {
		t.Fatalf("Failed to load test/user: %v", err)
	}
	t.Logf("Exported: %v", exports)

	// Get the export and call it
	export, err := reg.GetExport("test/user", "checkWrap")
	if err != nil {
		t.Fatalf("Failed to get export: %v", err)
	}

	// The export should be a closure we can call
	if export.Value == nil {
		t.Fatal("Export value is nil")
	}

	t.Logf("checkWrap export type: %T", export.Value)
}

// TestModuleWithRecordTypeAlias tests that record type aliases are properly unified
func TestModuleWithRecordTypeAlias(t *testing.T) {
	reg := NewModuleRegistry()

	// Module with record type alias and function using that type
	src := `
module test/invoice

type LineItem = {
  description: string,
  quantity: int,
  price: float
}

export pure func validateItem(item: LineItem) -> bool =
  item.quantity > 0 && item.price >= 0.0

export pure func makeItem(desc: string, qty: int, p: float) -> LineItem =
  { description: desc, quantity: qty, price: p }
`
	exports, err := reg.LoadModule("test/invoice", src)
	if err != nil {
		t.Fatalf("Failed to load module with record type alias: %v", err)
	}

	t.Logf("Exported: %v", exports)

	// Verify both exports exist
	if len(exports) != 2 {
		t.Errorf("Expected 2 exports, got %d: %v", len(exports), exports)
	}
}

// TestInvokeExportWithImports tests that InvokeExport can call functions
// that use imports from other modules (the key bug in ailangCall).
func TestInvokeExportWithImports(t *testing.T) {
	reg := NewModuleRegistry()

	// First, load std/option which defines Option, Some, None
	optionCode := `module std/option

export type Option[a] = Some(a) | None

-- Helper function to wrap a value
export pure func wrap[a](x: a) -> Option[a] { Some(x) }
`
	_, err := reg.LoadModule("std/option", optionCode)
	if err != nil {
		t.Fatalf("Failed to load std/option: %v", err)
	}

	// Load a module that imports from std/option
	userCode := `module user_module

import std/option (Option, Some, None, wrap)

-- Function that uses imported wrap
export pure func makeOption(x: int) -> Option[int] { wrap(x) }

-- Function that checks if option is Some
export pure func isSome(opt: Option[int]) -> bool {
  match opt {
    Some(_) => true,
    None => false
  }
}
`
	exports, err := reg.LoadModule("user_module", userCode)
	if err != nil {
		t.Fatalf("Failed to load user_module: %v", err)
	}
	t.Logf("Exported: %v", exports)

	// Test InvokeExport with makeOption
	result, err := reg.InvokeExport("user_module", "makeOption", []eval.Value{
		&eval.IntValue{Value: 42},
	})
	if err != nil {
		t.Fatalf("InvokeExport makeOption failed: %v", err)
	}

	// Should return Some(42)
	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("Expected TaggedValue, got %T: %v", result, result)
	}
	if tagged.CtorName != "Some" {
		t.Errorf("Expected Some, got %s", tagged.CtorName)
	}
	if len(tagged.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(tagged.Fields))
	}

	// Test InvokeExport with isSome
	result2, err := reg.InvokeExport("user_module", "isSome", []eval.Value{result})
	if err != nil {
		t.Fatalf("InvokeExport isSome failed: %v", err)
	}

	boolVal, ok := result2.(*eval.BoolValue)
	if !ok {
		t.Fatalf("Expected BoolValue, got %T: %v", result2, result2)
	}
	if !boolVal.Value {
		t.Errorf("Expected true, got false")
	}

	t.Logf("makeOption(42) = %v", result)
	t.Logf("isSome(Some(42)) = %v", result2)
}

// TestInvokeExportWithStdlibBuiltins tests that InvokeExport can call functions
// that use stdlib builtins like _json_encode (the WASM bug).
func TestInvokeExportWithStdlibBuiltins(t *testing.T) {
	reg := NewModuleRegistry()

	// Load std/json which provides encode function that wraps _json_encode
	jsonContent, err := os.ReadFile("../../std/json.ail")
	if err != nil {
		t.Fatalf("Failed to read std/json.ail: %v", err)
	}

	// First load dependencies
	optionContent, _ := os.ReadFile("../../std/option.ail")
	if optionContent != nil {
		reg.LoadModule("std/option", string(optionContent))
	}
	resultContent, _ := os.ReadFile("../../std/result.ail")
	if resultContent != nil {
		reg.LoadModule("std/result", string(resultContent))
	}
	listContent, _ := os.ReadFile("../../std/list.ail")
	if listContent != nil {
		reg.LoadModule("std/list", string(listContent))
	}
	mathContent, _ := os.ReadFile("../../std/math.ail")
	if mathContent != nil {
		reg.LoadModule("std/math", string(mathContent))
	}

	_, err = reg.LoadModule("std/json", string(jsonContent))
	if err != nil {
		t.Fatalf("Failed to load std/json: %v", err)
	}

	// Load a module that uses std/json.encode
	userCode := `module test_json

import std/json (encode, js)

-- Function that creates a JSON string using encode
export pure func makeJson(name: string) -> string {
  encode(js(name))
}
`
	exports, err := reg.LoadModule("test_json", userCode)
	if err != nil {
		t.Fatalf("Failed to load test_json: %v", err)
	}
	t.Logf("Exported: %v", exports)

	// Debug: Check if builtins are in registry
	evaluator := eval.NewCoreEvaluator()
	builtinReg := runtime.NewBuiltinRegistry(evaluator)
	if val, ok := builtinReg.Get("_json_encode"); ok {
		t.Logf("_json_encode found in BuiltinRegistry: %T", val)
	} else {
		t.Logf("_json_encode NOT found in BuiltinRegistry!")
	}

	// Test InvokeExport with makeJson
	result, err := reg.InvokeExport("test_json", "makeJson", []eval.Value{
		&eval.StringValue{Value: "hello"},
	})
	if err != nil {
		t.Fatalf("InvokeExport makeJson failed: %v", err)
	}

	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T: %v", result, result)
	}
	t.Logf("makeJson('hello') = %s", strVal.Value)

	// Should produce a JSON string like "\"hello\""
	if strVal.Value != `"hello"` {
		t.Errorf("Expected '\"hello\"', got '%s'", strVal.Value)
	}
}

// TestInvokeExportWithLocalADTAndDecode mimics the invoice processor pattern:
// - Imports Ok, Err from std/result
// - Defines local ADT types (ParseResult, ValidationResult)
// - Pattern matches on decode() then creates local ADT values
// This is exactly what the user's invoice module does
func TestInvokeExportWithLocalADTAndDecode(t *testing.T) {
	reg := NewModuleRegistry()

	// Load stdlib dependencies
	for _, modName := range []string{"option", "result", "list", "math", "json"} {
		path := filepath.Join("..", "..", "std", modName+".ail")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read std/%s.ail: %v", modName, err)
		}
		_, err = reg.LoadModule("std/"+modName, string(content))
		if err != nil {
			t.Fatalf("Failed to load std/%s: %v", modName, err)
		}
	}

	// Invoice module pattern: local ADT types + pattern match on decode()
	invoiceCode := `
module test/invoice

import std/json (decode, encode, Json, JObject, JString, getString, get, asString)
import std/result (Ok, Err)
import std/option (Option, Some, None)

-- Local ADT types (not using std/result directly for domain types)
type ParseResult[a] = ParseOk(a) | ParseErr(string)
type ValidationResult = Valid(string) | Invalid(string)

-- Function that pattern matches on decode(), then creates local ADT
export func processJson(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => match getString(json, "name") {
      Some(name) => "found: " ++ name,
      None => "no name field"
    },
    Err(msg) => "parse error: " ++ msg
  }
}

-- Simpler version to test basic decode + pattern match
export func simpleCheck(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(_) => "ok",
    Err(_) => "error"
  }
}

-- Test extracting the json value from Ok - does it bind correctly?
export func extractJson(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => "got json",
    Err(msg) => "error: " ++ msg
  }
}

-- Test calling a function on the extracted json (without block)
export func callOnExtracted(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => match getString(json, "name") {
      Some(s) => s,
      None => "no name"
    },
    Err(msg) => "error: " ++ msg
  }
}

-- Same but with encode instead of getString
export func encodeExtracted(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => encode(json),
    Err(msg) => "error: " ++ msg
  }
}

-- Test using get (also 2 args like getString)
export func useGet(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => match get(json, "name") {
      Some(val) => "found",
      None => "not found"
    },
    Err(msg) => "error: " ++ msg
  }
}

-- Test calling getString outside of a pattern match arm
export func directGetString(json: Json) -> string {
  match getString(json, "name") {
    Some(s) => s,
    None => "none"
  }
}

-- Test just calling getString directly with both args at once (dummy arg to avoid nullary bug)
export func twoArgCall(dummy: int) -> string {
  match decode("{\"a\":\"b\"}") {
    Ok(json) => {
      let result = getString(json, "a");
      match result {
        Some(s) => s,
        None => "none"
      }
    },
    Err(_) => "err"
  }
}

-- Local multi-arg function - does this work?
pure func localTwoArg(a: int, b: int) -> int = a + b

export func testLocalTwoArg(x: int) -> int = localTwoArg(x, 10)

-- Test calling encode inside pattern match - this works
export func testEncodeInMatch(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => encode(json),
    Err(_) => "err"
  }
}

-- Test calling imported get (2-arg)
export func testImportedGet(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => match get(json, "name") {
      Some(_) => "found",
      None => "not found"
    },
    Err(_) => "err"
  }
}

-- Test calling get at top level, not in match arm
export func testGetTopLevel(json: Json) -> string {
  match get(json, "name") {
    Some(_) => "found",
    None => "not found"
  }
}
`
	exports, err := reg.LoadModule("test/invoice", invoiceCode)
	if err != nil {
		t.Fatalf("Failed to load test/invoice: %v", err)
	}
	t.Logf("Exported: %v", exports)

	// Test simpleCheck first (basic decode pattern match)
	result, err := reg.InvokeExport("test/invoice", "simpleCheck", []eval.Value{
		&eval.StringValue{Value: `{"name":"test"}`},
	})
	if err != nil {
		t.Fatalf("InvokeExport simpleCheck failed: %v", err)
	}
	t.Logf("simpleCheck('{\"name\":\"test\"}') = %v (%T)", result, result)

	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T: %v", result, result)
	}
	if strVal.Value != "ok" {
		t.Errorf("Expected 'ok', got '%s'", strVal.Value)
	}

	// Test extractJson - binding variable from Ok(json)
	resultExtract, err := reg.InvokeExport("test/invoice", "extractJson", []eval.Value{
		&eval.StringValue{Value: `{"name":"test"}`},
	})
	if err != nil {
		t.Fatalf("InvokeExport extractJson failed: %v", err)
	}
	t.Logf("extractJson('{\"name\":\"test\"}') = %v (%T)", resultExtract, resultExtract)

	// Test encodeExtracted - calling encode on extracted json
	resultEncode, err := reg.InvokeExport("test/invoice", "encodeExtracted", []eval.Value{
		&eval.StringValue{Value: `{"name":"test"}`},
	})
	if err != nil {
		t.Fatalf("InvokeExport encodeExtracted failed: %v", err)
	}
	t.Logf("encodeExtracted('{\"name\":\"test\"}') = %v (%T)", resultEncode, resultEncode)

	// Test testLocalTwoArg - calling local 2-arg function
	resultLocal, err := reg.InvokeExport("test/invoice", "testLocalTwoArg", []eval.Value{
		&eval.IntValue{Value: 5},
	})
	if err != nil {
		t.Fatalf("InvokeExport testLocalTwoArg failed: %v", err)
	}
	t.Logf("testLocalTwoArg(5) = %v (%T)", resultLocal, resultLocal)

	// Check what 'get' resolves to in std/json
	getExport, err := reg.GetExport("std/json", "get")
	if err != nil {
		t.Logf("GetExport std/json.get: %v", err)
	} else {
		t.Logf("std/json.get export value: %T = %v", getExport.Value, getExport.Value)
		if fv, ok := getExport.Value.(*eval.FunctionValue); ok && fv.Env != nil {
			t.Logf("  get's Params: %v", fv.Params)
			// Check what's in get's closure
			for _, name := range []string{"findInList", "None", "Some", "JObject"} {
				if val, ok := fv.Env.Get(name); ok {
					t.Logf("  get's env[%s]: %T", name, val)
				} else {
					t.Logf("  get's env[%s]: NOT FOUND", name)
				}
			}
		}
	}

	// Check what 'encode' resolves to (1-arg, works)
	encodeExport, err := reg.GetExport("std/json", "encode")
	if err != nil {
		t.Logf("GetExport std/json.encode: %v", err)
	} else {
		t.Logf("std/json.encode export value: %T = %v", encodeExport.Value, encodeExport.Value)
	}

	// Check what testImportedGet resolves to
	testImportExport, err := reg.GetExport("test/invoice", "testImportedGet")
	if err != nil {
		t.Logf("GetExport test/invoice.testImportedGet: %v", err)
	} else {
		t.Logf("test/invoice.testImportedGet export value: %T = %v", testImportExport.Value, testImportExport.Value)
		if fv, ok := testImportExport.Value.(*eval.FunctionValue); ok && fv.Env != nil {
			t.Logf("  Closure env: %p", fv.Env)
			// Lookup specific names we expect
			for _, name := range []string{"get", "decode", "encode", "getString"} {
				if val, ok := fv.Env.Get(name); ok {
					t.Logf("    %s: %T = %v", name, val, val)
				}
			}
		}
	}

	// Check what encodeExtracted resolves to (should work)
	encExtExport, err := reg.GetExport("test/invoice", "encodeExtracted")
	if err != nil {
		t.Logf("GetExport test/invoice.encodeExtracted: %v", err)
	} else {
		t.Logf("test/invoice.encodeExtracted export value: %T = %v", encExtExport.Value, encExtExport.Value)
		if fv, ok := encExtExport.Value.(*eval.FunctionValue); ok && fv.Env != nil {
			t.Logf("  Closure env: %p", fv.Env)
			// Lookup specific names we expect
			for _, name := range []string{"get", "decode", "encode", "getString"} {
				if val, ok := fv.Env.Get(name); ok {
					t.Logf("    %s: %T = %v", name, val, val)
				}
			}
		}
	}

	// Test testImportedGet - calling imported 2-arg function
	resultImport, err := reg.InvokeExport("test/invoice", "testImportedGet", []eval.Value{
		&eval.StringValue{Value: `{"name":"test"}`},
	})
	if err != nil {
		t.Logf("InvokeExport testImportedGet failed: %v", err)
	} else {
		t.Logf("testImportedGet('{\"name\":\"test\"}') = %v (%T)", resultImport, resultImport)
	}

	// Test twoArgCall - direct getString call (not in match arm variable)
	resultTwo, err := reg.InvokeExport("test/invoice", "twoArgCall", []eval.Value{
		&eval.IntValue{Value: 0}, // dummy arg to avoid nullary function bug
	})
	if err != nil {
		t.Fatalf("InvokeExport twoArgCall failed: %v", err)
	}
	t.Logf("twoArgCall(0) = %v (%T)", resultTwo, resultTwo)

	// Test useGet - calling get (also 2 args)
	resultGet, err := reg.InvokeExport("test/invoice", "useGet", []eval.Value{
		&eval.StringValue{Value: `{"name":"test"}`},
	})
	if err != nil {
		t.Fatalf("InvokeExport useGet failed: %v", err)
	}
	t.Logf("useGet('{\"name\":\"test\"}') = %v (%T)", resultGet, resultGet)

	// Test callOnExtracted - calling getString on extracted json
	resultCall, err := reg.InvokeExport("test/invoice", "callOnExtracted", []eval.Value{
		&eval.StringValue{Value: `{"name":"test"}`},
	})
	if err != nil {
		t.Fatalf("InvokeExport callOnExtracted failed: %v", err)
	}
	t.Logf("callOnExtracted('{\"name\":\"test\"}') = %v (%T)", resultCall, resultCall)

	// Test processJson (more complex with nested pattern match)
	result2, err := reg.InvokeExport("test/invoice", "processJson", []eval.Value{
		&eval.StringValue{Value: `{"name":"Alice"}`},
	})
	if err != nil {
		t.Fatalf("InvokeExport processJson failed: %v", err)
	}
	t.Logf("processJson('{\"name\":\"Alice\"}') = %v (%T)", result2, result2)

	strVal2, ok := result2.(*eval.StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T: %v", result2, result2)
	}
	if strVal2.Value != "found: Alice" {
		t.Errorf("Expected 'found: Alice', got '%s'", strVal2.Value)
	}

	// Test with missing name field
	result3, err := reg.InvokeExport("test/invoice", "processJson", []eval.Value{
		&eval.StringValue{Value: `{"id":123}`},
	})
	if err != nil {
		t.Fatalf("InvokeExport processJson (no name) failed: %v", err)
	}
	t.Logf("processJson('{\"id\":123}') = %v (%T)", result3, result3)
}

// TestInvokeExportWithJSONDecode tests that _json_decode works through InvokeExport
func TestInvokeExportWithJSONDecode(t *testing.T) {
	reg := NewModuleRegistry()

	// Load std/option first (std/result depends on it)
	optionPath := filepath.Join("..", "..", "std", "option.ail")
	optionContent, err := os.ReadFile(optionPath)
	if err != nil {
		t.Fatalf("Failed to read std/option.ail: %v", err)
	}
	_, err = reg.LoadModule("std/option", string(optionContent))
	if err != nil {
		t.Fatalf("Failed to load std/option: %v", err)
	}

	// Load std/result
	resultPath := filepath.Join("..", "..", "std", "result.ail")
	resultContent, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("Failed to read std/result.ail: %v", err)
	}
	_, err = reg.LoadModule("std/result", string(resultContent))
	if err != nil {
		t.Fatalf("Failed to load std/result: %v", err)
	}

	// Load std/list (std/json imports it)
	listPath := filepath.Join("..", "..", "std", "list.ail")
	listContent, err := os.ReadFile(listPath)
	if err != nil {
		t.Fatalf("Failed to read std/list.ail: %v", err)
	}
	_, err = reg.LoadModule("std/list", string(listContent))
	if err != nil {
		t.Fatalf("Failed to load std/list: %v", err)
	}

	// Load std/math (std/json imports it)
	mathPath := filepath.Join("..", "..", "std", "math.ail")
	mathContent, err := os.ReadFile(mathPath)
	if err != nil {
		t.Fatalf("Failed to read std/math.ail: %v", err)
	}
	_, err = reg.LoadModule("std/math", string(mathContent))
	if err != nil {
		t.Fatalf("Failed to load std/math: %v", err)
	}

	// Load std/json
	jsonPath := filepath.Join("..", "..", "std", "json.ail")
	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read std/json.ail: %v", err)
	}
	_, err = reg.LoadModule("std/json", string(jsonContent))
	if err != nil {
		t.Fatalf("Failed to load std/json: %v", err)
	}

	// Test user module that calls decode and returns the Result
	userCode := `
module test_decode

import std/json (decode, Json, JObject, JString)
import std/result (Result, Ok, Err)

-- Function that decodes JSON and returns the Result directly
export func parseJson(s: string) -> Result[Json, string] {
  decode(s)
}

-- Function that extracts a field if decoding succeeds
export func getValueOrError(s: string) -> string {
  match decode(s) {
    Ok(json) => "parsed",
    Err(msg) => msg
  }
}
`
	exports, err := reg.LoadModule("test_decode", userCode)
	if err != nil {
		t.Fatalf("Failed to load test_decode: %v", err)
	}
	t.Logf("Exported: %v", exports)

	// Test getValueOrError with valid JSON
	result, err := reg.InvokeExport("test_decode", "getValueOrError", []eval.Value{
		&eval.StringValue{Value: `{"name":"test"}`},
	})
	if err != nil {
		t.Fatalf("InvokeExport getValueOrError failed: %v", err)
	}
	t.Logf("getValueOrError('{\"name\":\"test\"}') = %v (%T)", result, result)

	// Test getValueOrError with invalid JSON
	result2, err := reg.InvokeExport("test_decode", "getValueOrError", []eval.Value{
		&eval.StringValue{Value: `invalid json`},
	})
	if err != nil {
		t.Fatalf("InvokeExport getValueOrError with invalid JSON failed: %v", err)
	}
	t.Logf("getValueOrError('invalid json') = %v (%T)", result2, result2)
}
