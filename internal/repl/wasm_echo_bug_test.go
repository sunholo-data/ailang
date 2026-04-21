package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TestInvokeExportWithArithmetic verifies that InvokeExport works for functions
// that use arithmetic and comparison operations (requires type class dictionaries).
// This is the actual root cause of the "expected int arguments" bug in WASM:
// InvokeExport was creating a fresh evaluator without type class dictionaries.
func TestInvokeExportWithArithmetic(t *testing.T) {
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

	// Module with arithmetic and comparison operations
	code := `
module test/arith
import std/json (decode, getNumber)
import std/result (Ok, Err)
import std/option (Some, None)

export func addInts(a: int, b: int) -> int = a + b
export func isPositive(n: int) -> bool = n > 0
export func validateQuantity(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => match getNumber(json, "qty") {
      Some(q) => if q > 0.0 then "valid" else "invalid",
      None => "missing"
    },
    Err(msg) => "parse error"
  }
}
`
	_, err := reg.LoadModule("test/arith", code)
	if err != nil {
		t.Fatalf("Failed to load test/arith: %v", err)
	}

	// Test integer arithmetic
	t.Run("addInts", func(t *testing.T) {
		result, err := reg.InvokeExport("test/arith", "addInts", []eval.Value{
			&eval.IntValue{Value: 3},
			&eval.IntValue{Value: 7},
		})
		if err != nil {
			t.Fatalf("addInts(3, 7) failed: %v", err)
		}
		iv, ok := result.(*eval.IntValue)
		if !ok {
			t.Fatalf("Expected IntValue, got %T: %v", result, result)
		}
		if iv.Value != 10 {
			t.Errorf("Expected 10, got %d", iv.Value)
		}
	})

	// Test integer comparison
	t.Run("isPositive", func(t *testing.T) {
		result, err := reg.InvokeExport("test/arith", "isPositive", []eval.Value{
			&eval.IntValue{Value: 5},
		})
		if err != nil {
			t.Fatalf("isPositive(5) failed: %v", err)
		}
		bv, ok := result.(*eval.BoolValue)
		if !ok {
			t.Fatalf("Expected BoolValue, got %T: %v", result, result)
		}
		if !bv.Value {
			t.Errorf("Expected true, got false")
		}
	})

	// Test JSON with number comparison (the exact WASM demo failure)
	t.Run("validateQuantity", func(t *testing.T) {
		result, err := reg.InvokeExport("test/arith", "validateQuantity", []eval.Value{
			&eval.StringValue{Value: `{"qty":10}`},
		})
		if err != nil {
			t.Fatalf("validateQuantity failed: %v", err)
		}
		sv, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if sv.Value != "valid" {
			t.Errorf("Expected 'valid', got %q", sv.Value)
		}
	})
}

// TestWasmEchoBug reproduces the exact bug from the invoice_processor_wasm demo:
// - Module imports std/json and std/result
// - Simple echo function: export pure func echo(s: string) -> string = s
// - Calling echo with '{"quantity":10}' fails with "expected int arguments"
// - But echo('{}') and echo('{"name":"alice"}') work fine
func TestWasmEchoBug(t *testing.T) {
	reg := NewModuleRegistry()

	// Load stdlib dependencies (same as WASM loadEmbeddedStdlib)
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

	// Exact module from bug report
	code := `
module test
import std/json (decode, encode, jo, kv, js, jb)
import std/result (Ok, Err)
export pure func echo(s: string) -> string = s
`

	exports, err := reg.LoadModule("test", code)
	if err != nil {
		t.Fatalf("Failed to load test module: %v", err)
	}
	t.Logf("Exports: %v", exports)

	// Check what echo looks like
	echoExport, err := reg.GetExport("test", "echo")
	if err != nil {
		t.Fatalf("GetExport echo: %v", err)
	}
	t.Logf("echo value: %T", echoExport.Value)
	if fn, ok := echoExport.Value.(*eval.FunctionValue); ok {
		t.Logf("  Params: %v", fn.Params)
		t.Logf("  Body type: %T", fn.Body)
	}

	// Step 2 from bug report: These PASS
	tests := []struct {
		name  string
		input string
	}{
		{"empty object", `{}`},
		{"strings only", `{"name":"alice"}`},
		// Step 3 from bug report: This FAILS
		{"with number", `{"quantity":10}`},
		{"with float", `{"price":9.99}`},
		{"with negative", `{"balance":-42}`},
		{"mixed types", `{"name":"alice","age":30,"active":true}`},
		{"nested", `{"item":{"qty":5}}`},
		{"array with numbers", `[1,2,3]`},
		{"just a number", `42`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reg.InvokeExport("test", "echo", []eval.Value{
				&eval.StringValue{Value: tt.input},
			})
			if err != nil {
				t.Fatalf("echo(%q) failed: %v", tt.input, err)
			}
			strVal, ok := result.(*eval.StringValue)
			if !ok {
				t.Fatalf("Expected StringValue, got %T: %v", result, result)
			}
			if strVal.Value != tt.input {
				t.Errorf("Expected %q, got %q", tt.input, strVal.Value)
			}
		})
	}
}
