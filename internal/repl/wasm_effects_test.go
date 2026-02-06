package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// TestGetEffContext verifies the REPL exposes its effect context.
func TestGetEffContext(t *testing.T) {
	r := New()
	ctx := r.GetEffContext()
	if ctx == nil {
		t.Fatal("GetEffContext() returned nil")
	}
	// REPL grants IO by default
	if !ctx.HasCap("IO") {
		t.Error("Expected REPL to have IO capability by default")
	}
}

// TestGrantCapability verifies granting capabilities through the REPL.
func TestGrantCapability(t *testing.T) {
	r := New()
	ctx := r.GetEffContext()

	// AI should not be granted by default
	if ctx.HasCap("AI") {
		t.Fatal("AI capability should not be granted by default")
	}

	r.GrantCapability("AI")
	if !ctx.HasCap("AI") {
		t.Error("Expected AI capability after GrantCapability")
	}

	r.GrantCapability("Net")
	if !ctx.HasCap("Net") {
		t.Error("Expected Net capability after GrantCapability")
	}
}

// TestSetAIHandler verifies the AI handler is wired and auto-grants AI cap.
func TestSetAIHandler(t *testing.T) {
	r := New()
	ctx := r.GetEffContext()

	// Before: no AI handler
	if ctx.AI != nil {
		t.Fatal("Expected no AI context before SetAIHandler")
	}

	// Set handler
	stub := effects.NewStubAIHandler()
	stub.SetDefaultResponse("test response")
	r.SetAIHandler(stub)

	// After: AI handler set and capability granted
	if ctx.AI == nil {
		t.Fatal("Expected AI context after SetAIHandler")
	}
	if !ctx.HasCap("AI") {
		t.Error("Expected AI capability auto-granted by SetAIHandler")
	}

	// Verify handler works
	result, err := ctx.AI.Call("hello")
	if err != nil {
		t.Fatalf("AI.Call failed: %v", err)
	}
	if result != "test response" {
		t.Errorf("Expected 'test response', got %q", result)
	}
}

// TestAICallThroughInvokeExport verifies that a module using AI.call works
// when an AI handler is configured via the REPL + registry EffContext sharing.
func TestAICallThroughInvokeExport(t *testing.T) {
	// Create REPL and registry, wired like WASM does
	r := New()
	reg := NewModuleRegistry()
	r.SetRegistry(reg)
	reg.SetEffContext(r.GetEffContext())

	// Configure AI handler via REPL method
	stub := effects.NewStubAIHandler()
	stub.SetResponse("What is 2+2?", "4")
	stub.SetDefaultResponse("I don't know")
	r.SetAIHandler(stub)

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

	// Load std/ai (wraps _ai_call builtin)
	aiPath := filepath.Join("..", "..", "std", "ai.ail")
	aiContent, err := os.ReadFile(aiPath)
	if err != nil {
		t.Fatalf("Failed to read std/ai.ail: %v", err)
	}
	_, err = reg.LoadModule("std/ai", string(aiContent))
	if err != nil {
		t.Fatalf("Failed to load std/ai: %v", err)
	}

	// Module that uses AI.call
	code := `
module test/ai_demo
import std/ai as AI

export func askAI(question: string) -> string ! {AI} =
    AI.call(question)
`
	_, err = reg.LoadModule("test/ai_demo", code)
	if err != nil {
		t.Fatalf("Failed to load test/ai_demo: %v", err)
	}

	// InvokeExport with the AI function
	result, err := reg.InvokeExport("test/ai_demo", "askAI", []eval.Value{
		&eval.StringValue{Value: "What is 2+2?"},
	})
	if err != nil {
		t.Fatalf("askAI failed: %v", err)
	}
	sv, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T: %v", result, result)
	}
	if sv.Value != "4" {
		t.Errorf("Expected '4', got %q", sv.Value)
	}
}

// TestStdlibSymbolImport verifies that user modules can import specific symbols
// from stdlib modules (e.g., import std/math (intToFloat)).
// This reproduces the bug where intToFloat from std/math is undefined at runtime.
func TestStdlibSymbolImport(t *testing.T) {
	reg := NewModuleRegistry()

	// Load stdlib dependencies needed by std/math
	for _, modName := range []string{"option", "result", "list", "math"} {
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

	// Load a user module that imports intToFloat from std/math
	code := `
module test/math_user
import std/math (intToFloat)

export func convertQuantity(qty: int) -> float =
    intToFloat(qty)
`
	_, err := reg.LoadModule("test/math_user", code)
	if err != nil {
		t.Fatalf("Failed to load test/math_user: %v", err)
	}

	// InvokeExport to call the function
	result, err := reg.InvokeExport("test/math_user", "convertQuantity", []eval.Value{
		&eval.IntValue{Value: 42},
	})
	if err != nil {
		t.Fatalf("convertQuantity failed: %v", err)
	}
	fv, ok := result.(*eval.FloatValue)
	if !ok {
		t.Fatalf("Expected FloatValue, got %T: %v", result, result)
	}
	if fv.Value != 42.0 {
		t.Errorf("Expected 42.0, got %f", fv.Value)
	}
}

// TestStdlibModuleAliasImport verifies that modules using module alias imports
// (import std/math as Math) can access functions via qualified names.
func TestStdlibModuleAliasImport(t *testing.T) {
	reg := NewModuleRegistry()

	// Load all stdlib modules needed
	for _, modName := range []string{"option", "result", "list", "math", "json", "string"} {
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

	// Load a user module using both module alias and symbol imports
	code := `
module test/invoice
import std/math as Math
import std/string (intToStr)

export func formatQty(qty: int) -> string =
    let f = Math.intToFloat(qty)
    in intToStr(qty) ++ " units = " ++ intToStr(qty)
`
	exports, err := reg.LoadModule("test/invoice", code)
	if err != nil {
		t.Fatalf("Failed to load test/invoice: %v", err)
	}
	t.Logf("Loaded exports: %v", exports)

	// InvokeExport to call the function
	result, err := reg.InvokeExport("test/invoice", "formatQty", []eval.Value{
		&eval.IntValue{Value: 5},
	})
	if err != nil {
		t.Fatalf("formatQty failed: %v", err)
	}
	sv, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T: %v", result, result)
	}
	t.Logf("Result: %s", sv.Value)
}

// TestEmbeddedStdlibLoading simulates how WASM loads stdlib from the embedded FS.
// Verifies that all stdlib modules load successfully (like loadEmbeddedStdlib does).
func TestEmbeddedStdlibLoading(t *testing.T) {
	reg := NewModuleRegistry()

	// Read all .ail files from std/ directory (simulates std.FS.ReadDir)
	stdDir := filepath.Join("..", "..", "std")
	entries, err := os.ReadDir(stdDir)
	if err != nil {
		t.Fatalf("Failed to read std directory: %v", err)
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
		content, err := os.ReadFile(filepath.Join(stdDir, entry.Name()))
		if err != nil {
			continue
		}
		moduleName := "std/" + strings.TrimSuffix(entry.Name(), ".ail")
		pending = append(pending, moduleSource{name: moduleName, content: string(content)})
	}

	// Multi-pass loading (same as loadEmbeddedStdlib)
	const maxPasses = 5
	for pass := 0; pass < maxPasses && len(pending) > 0; pass++ {
		var stillPending []moduleSource
		for _, mod := range pending {
			_, err := reg.LoadModule(mod.name, mod.content)
			if err != nil {
				stillPending = append(stillPending, mod)
			}
		}
		if len(stillPending) == len(pending) {
			// No progress
			for _, mod := range stillPending {
				t.Errorf("Failed to load stdlib module: %s", mod.name)
			}
			break
		}
		pending = stillPending
	}

	// Verify std/math loaded
	mathMod, ok := reg.GetModule("std/math")
	if !ok {
		t.Fatal("std/math not loaded into registry")
	}
	if _, exists := mathMod.Exports["intToFloat"]; !exists {
		t.Error("std/math missing intToFloat export")
	}

	// Now try loading a user module that imports intToFloat
	code := `
module test/use_math
import std/math (intToFloat)

export func convert(x: int) -> float = intToFloat(x)
`
	exports, err := reg.LoadModule("test/use_math", code)
	if err != nil {
		t.Fatalf("Failed to load user module: %v", err)
	}
	t.Logf("User module exports: %v", exports)

	result, err := reg.InvokeExport("test/use_math", "convert", []eval.Value{
		&eval.IntValue{Value: 7},
	})
	if err != nil {
		t.Fatalf("convert(7) failed: %v", err)
	}
	fv, ok2 := result.(*eval.FloatValue)
	if !ok2 {
		t.Fatalf("Expected FloatValue, got %T", result)
	}
	if fv.Value != 7.0 {
		t.Errorf("Expected 7.0, got %f", fv.Value)
	}
}
