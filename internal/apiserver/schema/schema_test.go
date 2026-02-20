package schema

import (
	"encoding/json"
	"testing"
)

func TestPrimitiveTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected string // JSON-encoded schema
	}{
		{"int", `{"type":"integer"}`},
		{"float", `{"type":"number"}`},
		{"string", `{"type":"string"}`},
		{"bool", `{"type":"boolean"}`},
		{"unit", `{"type":"null"}`},
		{"()", `{"type":"null"}`},
		{"bytes", `{"contentEncoding":"base64","type":"string"}`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			schema := TypeToSchema(tt.input)
			got := mustJSON(t, schema)
			if got != tt.expected {
				t.Errorf("TypeToSchema(%q) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTypeVariables(t *testing.T) {
	tests := []string{"a", "b", "e", "t"}
	for _, tv := range tests {
		t.Run(tv, func(t *testing.T) {
			schema := TypeToSchema(tv)
			got := mustJSON(t, schema)
			if got != "{}" {
				t.Errorf("TypeToSchema(%q) = %s, want {}", tv, got)
			}
		})
	}
}

func TestListType(t *testing.T) {
	schema := TypeToSchema("[int]")
	got := mustJSON(t, schema)
	expected := `{"items":{"type":"integer"},"type":"array"}`
	if got != expected {
		t.Errorf("got %s, want %s", got, expected)
	}
}

func TestNestedListType(t *testing.T) {
	schema := TypeToSchema("[[string]]")
	got := mustJSON(t, schema)
	expected := `{"items":{"items":{"type":"string"},"type":"array"},"type":"array"}`
	if got != expected {
		t.Errorf("got %s, want %s", got, expected)
	}
}

func TestTupleType(t *testing.T) {
	schema := TypeToSchema("(int, string)")
	items, ok := schema["items"].([]JSONSchema)
	if !ok {
		t.Fatalf("expected items to be []JSONSchema, got %T", schema["items"])
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0]["type"] != "integer" {
		t.Errorf("first item type = %v, want integer", items[0]["type"])
	}
	if items[1]["type"] != "string" {
		t.Errorf("second item type = %v, want string", items[1]["type"])
	}
	if schema["minItems"] != 2 || schema["maxItems"] != 2 {
		t.Errorf("tuple should have minItems=2, maxItems=2")
	}
}

func TestRecordType(t *testing.T) {
	schema := TypeToSchema("{name: string, age: int}")
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %T", schema["properties"])
	}

	nameSchema, ok := props["name"].(JSONSchema)
	if !ok {
		t.Fatalf("expected name to be JSONSchema")
	}
	if nameSchema["type"] != "string" {
		t.Errorf("name type = %v, want string", nameSchema["type"])
	}

	ageSchema, ok := props["age"].(JSONSchema)
	if !ok {
		t.Fatalf("expected age to be JSONSchema")
	}
	if ageSchema["type"] != "integer" {
		t.Errorf("age type = %v, want integer", ageSchema["type"])
	}
}

func TestFunctionTypeDecomposition(t *testing.T) {
	fs := FromTypeString("int -> string -> bool")
	if fs.Arity != 2 {
		t.Fatalf("arity = %d, want 2", fs.Arity)
	}
	if len(fs.Parameters) != 2 {
		t.Fatalf("params = %d, want 2", len(fs.Parameters))
	}
	if fs.Parameters[0]["type"] != "integer" {
		t.Errorf("param[0] type = %v, want integer", fs.Parameters[0]["type"])
	}
	if fs.Parameters[1]["type"] != "string" {
		t.Errorf("param[1] type = %v, want string", fs.Parameters[1]["type"])
	}
	if fs.Return["type"] != "boolean" {
		t.Errorf("return type = %v, want boolean", fs.Return["type"])
	}
}

func TestFunctionWithListParam(t *testing.T) {
	fs := FromTypeString("[int] -> int")
	if fs.Arity != 1 {
		t.Fatalf("arity = %d, want 1", fs.Arity)
	}
	if fs.Parameters[0]["type"] != "array" {
		t.Errorf("param type = %v, want array", fs.Parameters[0]["type"])
	}
	if fs.Return["type"] != "integer" {
		t.Errorf("return type = %v, want integer", fs.Return["type"])
	}
}

func TestFunctionWithRecordReturn(t *testing.T) {
	fs := FromTypeString("string -> {name: string, age: int}")
	if fs.Arity != 1 {
		t.Fatalf("arity = %d, want 1", fs.Arity)
	}
	if fs.Return["type"] != "object" {
		t.Errorf("return type = %v, want object", fs.Return["type"])
	}
}

func TestNonFunctionType(t *testing.T) {
	fs := FromTypeString("int")
	if fs.Arity != 0 {
		t.Fatalf("arity = %d, want 0", fs.Arity)
	}
	if len(fs.Parameters) != 0 {
		t.Fatalf("params = %d, want 0", len(fs.Parameters))
	}
	if fs.Return["type"] != "integer" {
		t.Errorf("return type = %v, want integer", fs.Return["type"])
	}
}

func TestEmptyTypeString(t *testing.T) {
	fs := FromTypeString("")
	if fs.Arity != 0 {
		t.Fatalf("arity = %d, want 0", fs.Arity)
	}
	// Should return any schema.
	got := mustJSON(t, fs.Return)
	if got != "{}" {
		t.Errorf("empty type return = %s, want {}", got)
	}
}

func TestTypeApplication(t *testing.T) {
	t.Run("Option[int]", func(t *testing.T) {
		schema := TypeToSchema("Option[int]")
		oneOf, ok := schema["oneOf"].([]JSONSchema)
		if !ok {
			t.Fatalf("expected oneOf, got %v", schema)
		}
		if len(oneOf) != 2 {
			t.Fatalf("expected 2 oneOf variants, got %d", len(oneOf))
		}
	})

	t.Run("Result[int, string]", func(t *testing.T) {
		schema := TypeToSchema("Result[int, string]")
		oneOf, ok := schema["oneOf"].([]JSONSchema)
		if !ok {
			t.Fatalf("expected oneOf, got %v", schema)
		}
		if len(oneOf) != 2 {
			t.Fatalf("expected 2 oneOf variants, got %d", len(oneOf))
		}
	})

	t.Run("Array[float]", func(t *testing.T) {
		schema := TypeToSchema("Array[float]")
		if schema["type"] != "array" {
			t.Errorf("type = %v, want array", schema["type"])
		}
	})
}

func TestRequestSchema(t *testing.T) {
	fs := FromTypeString("int -> string -> bool")
	req := RequestSchema(fs)

	if req["type"] != "object" {
		t.Errorf("request type = %v, want object", req["type"])
	}

	props, ok := req["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties")
	}
	_, hasArgs := props["args"]
	if !hasArgs {
		t.Error("expected args property")
	}
}

func TestRequestSchemaNullary(t *testing.T) {
	fs := FromTypeString("int")
	req := RequestSchema(fs)

	props, ok := req["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties")
	}
	if len(props) != 0 {
		t.Errorf("nullary request should have no properties, got %d", len(props))
	}
}

func TestResponseSchema(t *testing.T) {
	fs := FromTypeString("int -> bool")
	resp := ResponseSchema(fs)

	props, ok := resp["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties")
	}

	resultSchema, ok := props["result"].(JSONSchema)
	if !ok {
		t.Fatalf("expected result property")
	}
	if resultSchema["type"] != "boolean" {
		t.Errorf("result type = %v, want boolean", resultSchema["type"])
	}
}

func TestNestedFunctionType(t *testing.T) {
	// (int -> int) -> [int] -> [int]
	// This is a higher-order function: first param is a function.
	fs := FromTypeString("(int -> int) -> [int] -> [int]")
	if fs.Arity != 2 {
		t.Fatalf("arity = %d, want 2", fs.Arity)
	}
	// First param is a parenthesized function type — treated as any since
	// functions can't be represented as JSON values.
	if fs.Return["type"] != "array" {
		t.Errorf("return type = %v, want array", fs.Return["type"])
	}
}

func TestParenthesizedSingleType(t *testing.T) {
	schema := TypeToSchema("(int)")
	if schema["type"] != "integer" {
		t.Errorf("(int) should unwrap to integer, got %v", schema["type"])
	}
}

func TestUnknownNamedType(t *testing.T) {
	schema := TypeToSchema("SolarPlanet")
	if schema["type"] != "object" {
		t.Errorf("unknown type should be object, got %v", schema["type"])
	}
	desc, ok := schema["description"].(string)
	if !ok || desc == "" {
		t.Error("unknown type should have description")
	}
}

func TestComplexFunctionType(t *testing.T) {
	// {name: string} -> [int] -> Result[string, int]
	fs := FromTypeString("{name: string} -> [int] -> Result[string, int]")
	if fs.Arity != 2 {
		t.Fatalf("arity = %d, want 2", fs.Arity)
	}
	if fs.Parameters[0]["type"] != "object" {
		t.Errorf("param[0] type = %v, want object", fs.Parameters[0]["type"])
	}
	if fs.Parameters[1]["type"] != "array" {
		t.Errorf("param[1] type = %v, want array", fs.Parameters[1]["type"])
	}
	// Return should be Result with oneOf.
	_, hasOneOf := fs.Return["oneOf"]
	if !hasOneOf {
		t.Error("Result return should have oneOf")
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(b)
}
