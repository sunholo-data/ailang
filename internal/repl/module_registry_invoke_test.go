package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/runtime"
)

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
      Some(name) => "found: ${name}",
      None => "no name field"
    },
    Err(msg) => "parse error: ${msg}"
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
    Err(msg) => "error: ${msg}"
  }
}

-- Test calling a function on the extracted json (without block)
export func callOnExtracted(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => match getString(json, "name") {
      Some(s) => s,
      None => "no name"
    },
    Err(msg) => "error: ${msg}"
  }
}

-- Same but with encode instead of getString
export func encodeExtracted(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => encode(json),
    Err(msg) => "error: ${msg}"
  }
}

-- Test using get (also 2 args like getString)
export func useGet(jsonStr: string) -> string {
  match decode(jsonStr) {
    Ok(json) => match get(json, "name") {
      Some(val) => "found",
      None => "not found"
    },
    Err(msg) => "error: ${msg}"
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
