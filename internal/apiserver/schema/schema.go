// Package schema converts AILANG type signatures to JSON Schema objects.
//
// This is the shared foundation for OpenAPI, MCP, and A2A protocol support.
// It parses type strings like "int -> string -> bool" and produces JSON Schema
// representations for request parameters and return values.
package schema

import (
	"strings"
)

// JSONSchema represents a JSON Schema object.
// We use a simple map rather than a struct to support arbitrary schema shapes.
type JSONSchema = map[string]any

// FunctionSchema holds the decomposed JSON Schema for a function type.
type FunctionSchema struct {
	// Parameters is the list of parameter schemas (one per function argument).
	Parameters []JSONSchema
	// Return is the schema for the return type.
	Return JSONSchema
	// Arity is the number of parameters.
	Arity int
}

// FromTypeString converts an AILANG type signature string to a FunctionSchema.
// For non-function types (no "->"), it returns a FunctionSchema with zero parameters
// and the type itself as the return schema.
//
// Examples:
//
//	"int -> string -> bool"  → params: [integer, string], return: boolean
//	"int"                    → params: [], return: integer
//	"[int]"                  → params: [], return: array of integer
func FromTypeString(typeStr string) *FunctionSchema {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		return &FunctionSchema{Return: anySchema()}
	}

	// Split on top-level " -> " (not inside brackets or parens).
	parts := splitFunctionType(typeStr)
	if len(parts) == 1 {
		// Not a function type.
		return &FunctionSchema{
			Return: typeToSchema(parts[0]),
		}
	}

	// Last part is return type, rest are parameters.
	params := make([]JSONSchema, len(parts)-1)
	for i := 0; i < len(parts)-1; i++ {
		params[i] = typeToSchema(parts[i])
	}

	return &FunctionSchema{
		Parameters: params,
		Return:     typeToSchema(parts[len(parts)-1]),
		Arity:      len(params),
	}
}

// TypeToSchema converts a single AILANG type string to a JSON Schema object.
func TypeToSchema(typeStr string) JSONSchema {
	return typeToSchema(strings.TrimSpace(typeStr))
}

// typeToSchema converts a single type (not a function type) to JSON Schema.
func typeToSchema(t string) JSONSchema {
	t = strings.TrimSpace(t)
	if t == "" {
		return anySchema()
	}

	// Unit literal "()" check before paren handling.
	if t == "()" {
		return JSONSchema{"type": "null"}
	}

	// Parenthesized type — unwrap if not a tuple.
	if strings.HasPrefix(t, "(") && strings.HasSuffix(t, ")") {
		inner := t[1 : len(t)-1]
		// Check if it's a tuple (has top-level commas).
		parts := splitTopLevel(inner, ',')
		if len(parts) > 1 {
			return tupleSchema(parts)
		}
		// Single parenthesized type — unwrap.
		return typeToSchema(inner)
	}

	// List type: [T]
	if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
		inner := t[1 : len(t)-1]
		return JSONSchema{
			"type":  "array",
			"items": typeToSchema(inner),
		}
	}

	// Record type: {f1: T1, f2: T2}
	if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
		return recordSchema(t[1 : len(t)-1])
	}

	// Primitive types.
	switch t {
	case "int":
		return JSONSchema{"type": "integer"}
	case "float":
		return JSONSchema{"type": "number"}
	case "string":
		return JSONSchema{"type": "string"}
	case "bool":
		return JSONSchema{"type": "boolean"}
	case "unit", "()":
		return JSONSchema{"type": "null"}
	case "bytes":
		return JSONSchema{"type": "string", "contentEncoding": "base64"}
	}

	// Type variable (single lowercase letter or lowercase name) → any.
	if isTypeVariable(t) {
		return anySchema()
	}

	// Type application like Result[a, e] or Array[int].
	if idx := strings.Index(t, "["); idx > 0 && strings.HasSuffix(t, "]") {
		name := t[:idx]
		argsStr := t[idx+1 : len(t)-1]
		args := splitTopLevel(argsStr, ',')
		return typeAppSchema(name, args)
	}

	// Named type (ADT, unknown) — return as described object.
	return JSONSchema{"type": "object", "description": "AILANG type: " + t}
}

// splitFunctionType splits a type string on top-level " -> " arrows,
// respecting nested brackets, parens, and braces.
func splitFunctionType(s string) []string {
	var parts []string
	depth := 0 // tracks [], (), {}
	start := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		}

		// Match " -> " at top level.
		if depth == 0 && i+4 <= len(s) && s[i:i+4] == " -> " {
			parts = append(parts, strings.TrimSpace(s[start:i]))
			start = i + 4
			i += 3 // skip past " -> " (loop will i++)
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}

// splitTopLevel splits on a delimiter at the top level (depth 0).
func splitTopLevel(s string, delim byte) []string {
	var parts []string
	depth := 0
	start := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		}

		if depth == 0 && s[i] == delim {
			parts = append(parts, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}

// tupleSchema creates a JSON Schema for a tuple type.
func tupleSchema(parts []string) JSONSchema {
	items := make([]JSONSchema, len(parts))
	for i, p := range parts {
		items[i] = typeToSchema(p)
	}
	return JSONSchema{
		"type":     "array",
		"items":    items,
		"minItems": len(parts),
		"maxItems": len(parts),
	}
}

// recordSchema creates a JSON Schema from record fields like "name: string, age: int".
func recordSchema(fields string) JSONSchema {
	parts := splitTopLevel(fields, ',')
	properties := make(map[string]any)
	required := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			continue
		}
		name := strings.TrimSpace(part[:colonIdx])
		typ := strings.TrimSpace(part[colonIdx+1:])
		properties[name] = typeToSchema(typ)
		required = append(required, name)
	}

	schema := JSONSchema{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// typeAppSchema handles type applications like Result[a, e], Array[int].
func typeAppSchema(name string, args []string) JSONSchema {
	switch name {
	case "Array":
		if len(args) == 1 {
			return JSONSchema{
				"type":  "array",
				"items": typeToSchema(args[0]),
			}
		}
	case "Option":
		if len(args) == 1 {
			return JSONSchema{
				"oneOf": []JSONSchema{
					{"type": "null", "description": "None"},
					typeToSchema(args[0]),
				},
				"description": "Option[" + args[0] + "]",
			}
		}
	case "Result":
		if len(args) == 2 {
			return JSONSchema{
				"oneOf": []JSONSchema{
					{"type": "object", "properties": map[string]any{"Ok": typeToSchema(args[0])}, "description": "Ok variant"},
					{"type": "object", "properties": map[string]any{"Err": typeToSchema(args[1])}, "description": "Err variant"},
				},
				"description": "Result[" + args[0] + ", " + args[1] + "]",
			}
		}
	}

	// Generic type application — describe it.
	desc := name + "[" + strings.Join(args, ", ") + "]"
	return JSONSchema{"type": "object", "description": "AILANG type: " + desc}
}

// isTypeVariable returns true if the type name looks like a type variable
// (single lowercase letter, or short lowercase name).
func isTypeVariable(t string) bool {
	if len(t) == 0 {
		return false
	}
	// Single lowercase letter is always a type variable.
	if len(t) == 1 && t[0] >= 'a' && t[0] <= 'z' {
		return true
	}
	return false
}

// anySchema returns a JSON Schema that accepts any value.
func anySchema() JSONSchema {
	return JSONSchema{}
}

// RequestSchema creates the JSON Schema for a function call request body.
// This matches the apiserver's FunctionCallRequest format: {"args": [...]}.
func RequestSchema(fs *FunctionSchema) JSONSchema {
	if fs.Arity == 0 {
		return JSONSchema{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	return JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"args": JSONSchema{
				"type":     "array",
				"items":    fs.Parameters,
				"minItems": fs.Arity,
				"maxItems": fs.Arity,
			},
		},
		"required": []string{"args"},
	}
}

// ResponseSchema creates the JSON Schema for a function call response body.
// This matches the apiserver's FunctionCallResponse format.
func ResponseSchema(fs *FunctionSchema) JSONSchema {
	return JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"result":     fs.Return,
			"module":     JSONSchema{"type": "string"},
			"func":       JSONSchema{"type": "string"},
			"elapsed_ms": JSONSchema{"type": "integer"},
			"error":      JSONSchema{"type": "string"},
		},
	}
}
