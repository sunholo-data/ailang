package repl

import (
	"strings"
	"testing"
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
