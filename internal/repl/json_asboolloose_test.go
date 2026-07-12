package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TestAsBoolLoose_FirestoreBooleanField reproduces the M-DX-JSON-BOOL data-
// integrity scenario at the std/json boundary.
//
// The bug: a package encodes a boolean as js("true") (a JSON *string*) while
// the external API (Firestore) normalises and returns it as a JSON *boolean*.
// asBool() only accepts JBool, so a value that round-trips through the API
// reads back as None and is silently treated as false.
//
// asBoolLoose() is the system-boundary fix: it accepts BOTH the real JSON
// boolean AND the stringified "true"/"false", while still returning None
// (structured failure, never a silent default) for genuinely non-boolean input.
//
// This mirrors sunholo/firestore/fields.ail's asBoolField decoder using the
// nested {"field": {"booleanValue": ...}} envelope shape.
func TestAsBoolLoose_FirestoreBooleanField(t *testing.T) {
	reg := NewModuleRegistry()

	for _, modName := range []string{"option", "result", "list", "math", "json"} {
		path := filepath.Join("..", "..", "std", modName+".ail")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read std/%s.ail: %v", modName, err)
		}
		if _, err := reg.LoadModule("std/"+modName, string(content)); err != nil {
			t.Fatalf("Failed to load std/%s: %v", modName, err)
		}
	}

	// A Firestore-style boolean-field decoder built on asBoolLoose.
	pkgCode := `
module pkg/test/boolfields

import std/json (get, asBoolLoose, decode)
import std/option (Some, None)
import std/result (Ok, Err)

-- Decode a Firestore {"field": {"booleanValue": <bool|string>}} envelope.
-- Uses asBoolLoose so a boolean survives whether the API returns it as a JSON
-- boolean OR a stringified "true"/"false".
export func asBoolField(fields: Json, fieldName: string) -> string =
  match get(fields, fieldName) {
    None => "MISSING",
    Some(field) =>
      match get(field, "booleanValue") {
        None => "MISSING",
        Some(bv) => match asBoolLoose(bv) {
          Some(b) => if b then "true" else "false",
          None => "NOT_BOOL"
        }
      }
  }

export func readFlag(jsonStr: string, fieldName: string) -> string =
  match decode(jsonStr) {
    Ok(fields) => asBoolField(fields, fieldName),
    Err(_) => "DECODE_ERROR"
  }
`
	if _, err := reg.LoadModule("pkg/test/boolfields", pkgCode); err != nil {
		t.Fatalf("Failed to load pkg/test/boolfields: %v", err)
	}

	cases := []struct {
		name string
		json string
		want string
	}{
		// Firestore-normalised real JSON booleans (the shape the API returns).
		{"real_bool_true", `{"active":{"booleanValue":true}}`, "true"},
		{"real_bool_false", `{"active":{"booleanValue":false}}`, "false"},
		// Stringified booleans (a naive js("true") encoder) — the bug, now tolerated.
		{"string_true", `{"active":{"booleanValue":"true"}}`, "true"},
		{"string_false", `{"active":{"booleanValue":"false"}}`, "false"},
		// Genuinely non-boolean input is structured failure, never a silent false.
		{"number_not_bool", `{"active":{"booleanValue":42}}`, "NOT_BOOL"},
		{"string_not_bool", `{"active":{"booleanValue":"maybe"}}`, "NOT_BOOL"},
		// Absent field.
		{"missing_field", `{"other":{"booleanValue":true}}`, "MISSING"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := reg.InvokeExport("pkg/test/boolfields", "readFlag", []eval.Value{
				&eval.StringValue{Value: tc.json},
				&eval.StringValue{Value: "active"},
			})
			if err != nil {
				t.Fatalf("InvokeExport readFlag failed: %v", err)
			}
			strVal, ok := result.(*eval.StringValue)
			if !ok {
				t.Fatalf("Expected StringValue, got %T: %v", result, result)
			}
			if strVal.Value != tc.want {
				t.Errorf("readFlag(%s) = %q, want %q", tc.json, strVal.Value, tc.want)
			}
		})
	}
}
