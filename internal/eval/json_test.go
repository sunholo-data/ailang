package eval

import (
	"testing"
)

// TestJSONEncodeNull verifies JNull encodes to "null"
func TestJSONEncodeNull(t *testing.T) {
	jnull := &TaggedValue{
		TypeName: "Json",
		CtorName: "JNull",
		Fields:   []Value{},
	}

	result, err := encodeJSON(jnull)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "null"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestJSONEncodeBool verifies JBool encodes correctly
func TestJSONEncodeBool(t *testing.T) {
	testCases := []struct {
		value    bool
		expected string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tc := range testCases {
		jbool := &TaggedValue{
			TypeName: "Json",
			CtorName: "JBool",
			Fields:   []Value{&BoolValue{Value: tc.value}},
		}

		result, err := encodeJSON(jbool)
		if err != nil {
			t.Fatalf("Unexpected error for %v: %v", tc.value, err)
		}

		if result != tc.expected {
			t.Errorf("Expected %q, got %q", tc.expected, result)
		}
	}
}

// TestJSONEncodeNumber verifies JNumber encodes correctly
func TestJSONEncodeNumber(t *testing.T) {
	testCases := []struct {
		value    float64
		expected string
	}{
		{0.0, "0"},
		{42.0, "42"},
		{3.14, "3.14"},
		{-1.5, "-1.5"},
		{1e10, "1e+10"},
	}

	for _, tc := range testCases {
		jnum := &TaggedValue{
			TypeName: "Json",
			CtorName: "JNumber",
			Fields:   []Value{&FloatValue{Value: tc.value}},
		}

		result, err := encodeJSON(jnum)
		if err != nil {
			t.Fatalf("Unexpected error for %v: %v", tc.value, err)
		}

		if result != tc.expected {
			t.Errorf("Expected %q, got %q", tc.expected, result)
		}
	}
}

// TestJSONEncodeString verifies JString with escaping
func TestJSONEncodeString(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected string
	}{
		{"empty", "", `""`},
		{"simple", "hello", `"hello"`},
		{"with spaces", "hello world", `"hello world"`},
		{"quote", `say "hi"`, `"say \"hi\""`},
		{"backslash", `path\to\file`, `"path\\to\\file"`},
		{"newline", "line1\nline2", `"line1\nline2"`},
		{"tab", "col1\tcol2", `"col1\tcol2"`},
		{"carriage return", "line1\rline2", `"line1\rline2"`},
		{"backspace", "test\b", `"test\b"`},
		{"form feed", "test\f", `"test\f"`},
		{"control char", "test\x01end", `"test\u0001end"`},
		{"unicode", "hello 世界", `"hello 世界"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jstr := &TaggedValue{
				TypeName: "Json",
				CtorName: "JString",
				Fields:   []Value{&StringValue{Value: tc.value}},
			}

			result, err := encodeJSON(jstr)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestJSONEncodeArray verifies JArray encoding
func TestJSONEncodeArray(t *testing.T) {
	testCases := []struct {
		name     string
		elements []Value
		expected string
	}{
		{
			name:     "empty array",
			elements: []Value{},
			expected: "[]",
		},
		{
			name: "single element",
			elements: []Value{
				&TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 42}}},
			},
			expected: "[42]",
		},
		{
			name: "multiple elements",
			elements: []Value{
				&TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 1}}},
				&TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 2}}},
				&TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 3}}},
			},
			expected: "[1,2,3]",
		},
		{
			name: "mixed types",
			elements: []Value{
				&TaggedValue{TypeName: "Json", CtorName: "JNull", Fields: []Value{}},
				&TaggedValue{TypeName: "Json", CtorName: "JBool", Fields: []Value{&BoolValue{Value: true}}},
				&TaggedValue{TypeName: "Json", CtorName: "JString", Fields: []Value{&StringValue{Value: "hi"}}},
			},
			expected: `[null,true,"hi"]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jarr := &TaggedValue{
				TypeName: "Json",
				CtorName: "JArray",
				Fields:   []Value{&ListValue{Elements: tc.elements}},
			}

			result, err := encodeJSON(jarr)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestJSONEncodeObject verifies JObject encoding
func TestJSONEncodeObject(t *testing.T) {
	testCases := []struct {
		name     string
		pairs    []Value
		expected string
	}{
		{
			name:     "empty object",
			pairs:    []Value{},
			expected: "{}",
		},
		{
			name: "single key-value",
			pairs: []Value{
				&RecordValue{
					Fields: map[string]Value{
						"key":   &StringValue{Value: "name"},
						"value": &TaggedValue{TypeName: "Json", CtorName: "JString", Fields: []Value{&StringValue{Value: "Alice"}}},
					},
				},
			},
			expected: `{"name":"Alice"}`,
		},
		{
			name: "multiple key-values (deterministic order)",
			pairs: []Value{
				&RecordValue{
					Fields: map[string]Value{
						"key":   &StringValue{Value: "a"},
						"value": &TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 1}}},
					},
				},
				&RecordValue{
					Fields: map[string]Value{
						"key":   &StringValue{Value: "b"},
						"value": &TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 2}}},
					},
				},
			},
			expected: `{"a":1,"b":2}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jobj := &TaggedValue{
				TypeName: "Json",
				CtorName: "JObject",
				Fields:   []Value{&ListValue{Elements: tc.pairs}},
			}

			result, err := encodeJSON(jobj)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestJSONEncodeNested verifies deep nesting
func TestJSONEncodeNested(t *testing.T) {
	// Build: {"outer": {"inner": [1, 2, 3]}}
	innerArray := &TaggedValue{
		TypeName: "Json",
		CtorName: "JArray",
		Fields: []Value{
			&ListValue{
				Elements: []Value{
					&TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 1}}},
					&TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 2}}},
					&TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 3}}},
				},
			},
		},
	}

	innerObject := &TaggedValue{
		TypeName: "Json",
		CtorName: "JObject",
		Fields: []Value{
			&ListValue{
				Elements: []Value{
					&RecordValue{
						Fields: map[string]Value{
							"key":   &StringValue{Value: "inner"},
							"value": innerArray,
						},
					},
				},
			},
		},
	}

	outerObject := &TaggedValue{
		TypeName: "Json",
		CtorName: "JObject",
		Fields: []Value{
			&ListValue{
				Elements: []Value{
					&RecordValue{
						Fields: map[string]Value{
							"key":   &StringValue{Value: "outer"},
							"value": innerObject,
						},
					},
				},
			},
		},
	}

	result, err := encodeJSON(outerObject)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := `{"outer":{"inner":[1,2,3]}}`
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestJSONEncodeBuiltin verifies the full _json_encode builtin
func TestJSONEncodeBuiltin(t *testing.T) {
	// Test simple object: {"name": "Bob", "age": 30}
	jobj := &TaggedValue{
		TypeName: "Json",
		CtorName: "JObject",
		Fields: []Value{
			&ListValue{
				Elements: []Value{
					&RecordValue{
						Fields: map[string]Value{
							"key":   &StringValue{Value: "name"},
							"value": &TaggedValue{TypeName: "Json", CtorName: "JString", Fields: []Value{&StringValue{Value: "Bob"}}},
						},
					},
					&RecordValue{
						Fields: map[string]Value{
							"key":   &StringValue{Value: "age"},
							"value": &TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: 30}}},
						},
					},
				},
			},
		},
	}

	builtin := Builtins["_json_encode"]
	impl := builtin.Impl.(func(Value) (*StringValue, error))

	result, err := impl(jobj)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := `{"name":"Bob","age":30}`
	if result.Value != expected {
		t.Errorf("Expected %q, got %q", expected, result.Value)
	}
}
