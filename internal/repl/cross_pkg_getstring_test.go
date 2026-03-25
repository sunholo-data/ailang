package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// TestRegistryPackageGetString simulates the exact pattern used by registry
// packages like sunholo/firestore/fields and sunholo/gcp_auth/token:
//
//   - Imports getString, get, asString from std/json
//   - Imports getOrElse, isSome from std/option (but NOT Some/None constructors)
//   - Uses Some/None in pattern matches without explicit import
//   - Uses getOrElse(getString(json, "key"), "") pattern (gcp_auth style)
//   - Uses nested get() calls for Firestore field extraction
//
// This validates that the FallbackResolver correctly propagates constructor
// resolution through cross-package stdlib function calls.
func TestRegistryPackageGetString(t *testing.T) {
	reg := NewModuleRegistry()

	// Load stdlib dependencies (same order as other tests)
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

	// Simulate a registry package that does NOT import Some/None explicitly.
	// This mirrors sunholo/firestore/fields.ail and sunholo/gcp_auth/token.ail.
	pkgCode := `
module pkg/test/fields

import std/json (jo, js, kv, get, asString, getString, decode)
import std/option (getOrElse, isSome)
import std/result (Ok, Err)

-- Pattern 1: Firestore-style nested get() with pattern match on Some/None
-- (mirrors sunholo/firestore/fields.ail asStr)
export pure func asStr(fields: Json, fieldName: string) -> string =
  match get(fields, fieldName) {
    None => "",
    Some(field) =>
      match get(field, "stringValue") {
        None => "",
        Some(sv) => getOrElse(asString(sv), "")
      }
  }

-- Pattern 2: getOrElse(getString(...), "") pattern
-- (mirrors sunholo/gcp_auth/token.ail)
export pure func getField(json: Json, key: string) -> string =
  getOrElse(getString(json, key), "")

-- Pattern 3: isSome + getOrElse combo
-- (mirrors sunholo/gcp_auth/token.ail getAccessToken)
export pure func getFieldOrFail(jsonStr: string, key: string) -> string =
  match decode(jsonStr) {
    Ok(json) => {
      let val = getString(json, key);
      if isSome(val) then getOrElse(val, "")
      else "MISSING"
    },
    Err(e) => "ERROR: " ++ e
  }

-- Pattern 4: Direct getString in match arm (cross-module pattern match)
export pure func directMatch(jsonStr: string, key: string) -> string =
  match decode(jsonStr) {
    Ok(json) => match getString(json, key) {
      Some(s) => s,
      None => "NONE"
    },
    Err(_) => "ERROR"
  }

-- Pattern 5: Firestore-style nested extraction from JSON string
-- Build Firestore envelope and extract through nested get() calls
export pure func firestoreExtract(jsonStr: string, fieldName: string) -> string =
  match decode(jsonStr) {
    Ok(fields) => asStr(fields, fieldName),
    Err(_) => "DECODE_ERROR"
  }
`
	exports, err := reg.LoadModule("pkg/test/fields", pkgCode)
	if err != nil {
		t.Fatalf("Failed to load pkg/test/fields: %v", err)
	}
	t.Logf("Exported: %v", exports)

	// Test Pattern 2 & 3 via getFieldOrFail (string-based, no Json construction needed)
	t.Run("getFieldOrFail_found", func(t *testing.T) {
		result, err := reg.InvokeExport("pkg/test/fields", "getFieldOrFail", []eval.Value{
			&eval.StringValue{Value: `{"name":"Alice","age":"30"}`},
			&eval.StringValue{Value: "name"},
		})
		if err != nil {
			t.Fatalf("InvokeExport getFieldOrFail failed: %v", err)
		}
		strVal, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if strVal.Value != "Alice" {
			t.Errorf("Expected 'Alice', got '%s'", strVal.Value)
		}
	})

	t.Run("getFieldOrFail_missing", func(t *testing.T) {
		result, err := reg.InvokeExport("pkg/test/fields", "getFieldOrFail", []eval.Value{
			&eval.StringValue{Value: `{"name":"Alice"}`},
			&eval.StringValue{Value: "email"},
		})
		if err != nil {
			t.Fatalf("InvokeExport getFieldOrFail (missing) failed: %v", err)
		}
		strVal, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if strVal.Value != "MISSING" {
			t.Errorf("Expected 'MISSING', got '%s'", strVal.Value)
		}
	})

	t.Run("getFieldOrFail_invalid_json", func(t *testing.T) {
		result, err := reg.InvokeExport("pkg/test/fields", "getFieldOrFail", []eval.Value{
			&eval.StringValue{Value: `not json`},
			&eval.StringValue{Value: "key"},
		})
		if err != nil {
			t.Fatalf("InvokeExport getFieldOrFail (invalid) failed: %v", err)
		}
		strVal, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if strVal.Value[:5] != "ERROR" {
			t.Errorf("Expected error string, got '%s'", strVal.Value)
		}
	})

	// Test Pattern 4: directMatch - getString in match arm
	t.Run("directMatch_found", func(t *testing.T) {
		result, err := reg.InvokeExport("pkg/test/fields", "directMatch", []eval.Value{
			&eval.StringValue{Value: `{"access_token":"tok_123"}`},
			&eval.StringValue{Value: "access_token"},
		})
		if err != nil {
			t.Fatalf("InvokeExport directMatch failed: %v", err)
		}
		strVal, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if strVal.Value != "tok_123" {
			t.Errorf("Expected 'tok_123', got '%s'", strVal.Value)
		}
	})

	t.Run("directMatch_missing", func(t *testing.T) {
		result, err := reg.InvokeExport("pkg/test/fields", "directMatch", []eval.Value{
			&eval.StringValue{Value: `{"other":"val"}`},
			&eval.StringValue{Value: "access_token"},
		})
		if err != nil {
			t.Fatalf("InvokeExport directMatch (missing) failed: %v", err)
		}
		strVal, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if strVal.Value != "NONE" {
			t.Errorf("Expected 'NONE', got '%s'", strVal.Value)
		}
	})

	// Test Pattern 5: Firestore-style nested extraction
	t.Run("firestoreExtract_found", func(t *testing.T) {
		// Firestore field format: {"name": {"stringValue": "Alice"}}
		result, err := reg.InvokeExport("pkg/test/fields", "firestoreExtract", []eval.Value{
			&eval.StringValue{Value: `{"name":{"stringValue":"Alice"},"age":{"integerValue":"30"}}`},
			&eval.StringValue{Value: "name"},
		})
		if err != nil {
			t.Fatalf("InvokeExport firestoreExtract failed: %v", err)
		}
		strVal, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if strVal.Value != "Alice" {
			t.Errorf("Expected 'Alice', got '%s'", strVal.Value)
		}
	})

	t.Run("firestoreExtract_missing_field", func(t *testing.T) {
		result, err := reg.InvokeExport("pkg/test/fields", "firestoreExtract", []eval.Value{
			&eval.StringValue{Value: `{"name":{"stringValue":"Alice"}}`},
			&eval.StringValue{Value: "email"},
		})
		if err != nil {
			t.Fatalf("InvokeExport firestoreExtract (missing) failed: %v", err)
		}
		strVal, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if strVal.Value != "" {
			t.Errorf("Expected empty string, got '%s'", strVal.Value)
		}
	})

	t.Run("firestoreExtract_not_string", func(t *testing.T) {
		// Field exists but is integerValue, not stringValue
		result, err := reg.InvokeExport("pkg/test/fields", "firestoreExtract", []eval.Value{
			&eval.StringValue{Value: `{"age":{"integerValue":"30"}}`},
			&eval.StringValue{Value: "age"},
		})
		if err != nil {
			t.Fatalf("InvokeExport firestoreExtract (not string) failed: %v", err)
		}
		strVal, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if strVal.Value != "" {
			t.Errorf("Expected empty string for non-stringValue field, got '%s'", strVal.Value)
		}
	})
}
