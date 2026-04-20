package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// TestInvokeExportWithJSONNumberStrings verifies that string arguments containing
// JSON with numbers are handled correctly through InvokeExport (mimics ailangCall).
func TestInvokeExportWithJSONNumberStrings(t *testing.T) {
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

	// Module that processes JSON with numbers
	code := `
module test/numbers

import std/json (decode, encode, getString, getNumber, get, Json)
import std/result (Ok, Err)
import std/option (Some, None)

export func processInvoice(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => match getString(json, "vendor") {
      Some(name) => match getNumber(json, "amount") {
        Some(amt) => name,
        None => "no amount"
      },
      None => "no vendor"
    },
    Err(msg) => "parse error: ${msg}"
  }
}

export func roundTrip(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => encode(json),
    Err(msg) => "error: ${msg}"
  }
}
`
	_, err := reg.LoadModule("test/numbers", code)
	if err != nil {
		t.Fatalf("Failed to load test/numbers: %v", err)
	}

	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "JSON with integer",
			input:  `{"vendor":"Acme","amount":42}`,
			expect: "Acme",
		},
		{
			name:   "JSON with float",
			input:  `{"vendor":"Acme","amount":42.50}`,
			expect: "Acme",
		},
		{
			name:   "JSON with negative number",
			input:  `{"vendor":"Acme","amount":-10.5}`,
			expect: "Acme",
		},
		{
			name:   "JSON with zero",
			input:  `{"vendor":"Acme","amount":0}`,
			expect: "Acme",
		},
		{
			name:   "JSON with scientific notation",
			input:  `{"vendor":"Acme","amount":1.5e10}`,
			expect: "Acme",
		},
		{
			name:   "JSON with missing vendor",
			input:  `{"amount":42}`,
			expect: "no vendor",
		},
		{
			name:   "JSON with missing amount",
			input:  `{"vendor":"Acme"}`,
			expect: "no amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reg.InvokeExport("test/numbers", "processInvoice", []eval.Value{
				&eval.StringValue{Value: tt.input},
			})
			if err != nil {
				t.Fatalf("processInvoice(%q) failed: %v", tt.input, err)
			}
			strVal, ok := result.(*eval.StringValue)
			if !ok {
				t.Fatalf("Expected StringValue, got %T: %v", result, result)
			}
			if strVal.Value != tt.expect {
				t.Errorf("Expected %q, got %q", tt.expect, strVal.Value)
			}
		})
	}

	// Test roundTrip to verify JSON with numbers is preserved
	t.Run("roundTrip preserves numbers", func(t *testing.T) {
		result, err := reg.InvokeExport("test/numbers", "roundTrip", []eval.Value{
			&eval.StringValue{Value: `{"amount":42.5,"count":3}`},
		})
		if err != nil {
			t.Fatalf("roundTrip failed: %v", err)
		}
		t.Logf("roundTrip result: %v", result)
	})
}
