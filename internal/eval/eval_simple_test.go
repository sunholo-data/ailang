package eval

import (
	"math"
	"testing"
)

func TestShowFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected string
	}{
		// Basic types
		{"integer", &IntValue{Value: 42}, "42"},
		{"negative integer", &IntValue{Value: -42}, "-42"},
		{"float", &FloatValue{Value: 3.14159}, "3.14159"},
		{"negative float", &FloatValue{Value: -3.14159}, "-3.14159"},
		{"boolean true", &BoolValue{Value: true}, "true"},
		{"boolean false", &BoolValue{Value: false}, "false"},
		{"unit", &UnitValue{}, "()"},
		
		// String with escaping (JSON rules)
		{"simple string", &StringValue{Value: "hello"}, `"hello"`},
		{"string with newline", &StringValue{Value: "hello\nworld"}, `"hello\nworld"`},
		{"string with tab", &StringValue{Value: "a\tb"}, `"a\tb"`},
		{"string with quotes", &StringValue{Value: `say "hi"`}, `"say \"hi\""`},
		{"string with backslash", &StringValue{Value: `path\to\file`}, `"path\\to\\file"`},
		
		// Special float values
		{"positive infinity", &FloatValue{Value: math.Inf(1)}, "Inf"},
		{"negative infinity", &FloatValue{Value: math.Inf(-1)}, "-Inf"},
		{"NaN", &FloatValue{Value: math.NaN()}, "NaN"},
		{"negative zero", &FloatValue{Value: math.Copysign(0, -1)}, "-0.0"},
		
		// Lists
		{"empty list", &ListValue{Elements: []Value{}}, "[]"},
		{"simple list", &ListValue{Elements: []Value{
			&IntValue{Value: 1},
			&IntValue{Value: 2},
			&IntValue{Value: 3},
		}}, "[1, 2, 3]"},
		{"nested list", &ListValue{Elements: []Value{
			&ListValue{Elements: []Value{&IntValue{Value: 1}}},
			&ListValue{Elements: []Value{&IntValue{Value: 2}}},
		}}, "[[1], [2]]"},
		
		// Records (with deterministic key sorting)
		{"empty record", &RecordValue{Fields: map[string]Value{}}, "{}"},
		{"simple record", &RecordValue{Fields: map[string]Value{
			"name": &StringValue{Value: "Alice"},
			"age":  &IntValue{Value: 30},
		}}, `{age: 30, name: "Alice"}`},
		{"record with sorted keys", &RecordValue{Fields: map[string]Value{
			"z": &IntValue{Value: 3},
			"a": &IntValue{Value: 1},
			"m": &IntValue{Value: 2},
		}}, "{a: 1, m: 2, z: 3}"},
		
		// Functions
		{"function", &FunctionValue{Params: []string{"x", "y"}}, "<function>"},
		{"builtin", &BuiltinFunction{Name: "print"}, "<builtin: print>"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := showValue(tt.input, 0)
			if result != tt.expected {
				t.Errorf("showValue(%s) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

func TestShowDepthLimit(t *testing.T) {
	// Create a deeply nested list
	deepList := &ListValue{Elements: []Value{
		&ListValue{Elements: []Value{
			&ListValue{Elements: []Value{
				&ListValue{Elements: []Value{
					&IntValue{Value: 1},
				}},
			}},
		}},
	}}
	
	result := showValue(deepList, 0)
	expected := "[[[[...]]]]"
	if result != expected {
		t.Errorf("showValue(deep nested) = %q, want %q", result, expected)
	}
}

func TestStringConcatenation(t *testing.T) {
	eval := NewSimple()
	
	tests := []struct {
		name      string
		left      Value
		op        string
		right     Value
		expected  Value
		wantError bool
	}{
		// ++ operator tests
		{
			name:     "string concatenation",
			left:     &StringValue{Value: "hello "},
			op:       "++",
			right:    &StringValue{Value: "world"},
			expected: &StringValue{Value: "hello world"},
		},
		{
			name:      "++ with non-string left",
			left:      &IntValue{Value: 42},
			op:        "++",
			right:     &StringValue{Value: "world"},
			wantError: true,
		},
		{
			name:      "++ with non-string right",
			left:      &StringValue{Value: "hello"},
			op:        "++",
			right:     &IntValue{Value: 42},
			wantError: true,
		},
		
		// + operator should not work with strings
		{
			name:      "+ with strings",
			left:      &StringValue{Value: "hello"},
			op:        "+",
			right:     &StringValue{Value: "world"},
			wantError: true,
		},
		{
			name:     "+ with numbers still works",
			left:     &IntValue{Value: 10},
			op:       "+",
			right:    &IntValue{Value: 20},
			expected: &IntValue{Value: 30},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.evalBinOp(tt.op, tt.left, tt.right)
			if tt.wantError {
				if err == nil {
					t.Errorf("evalBinOp(%s) expected error, got %v", tt.name, result)
				}
			} else {
				if err != nil {
					t.Errorf("evalBinOp(%s) unexpected error: %v", tt.name, err)
				}
				if result.String() != tt.expected.String() {
					t.Errorf("evalBinOp(%s) = %v, want %v", tt.name, result, tt.expected)
				}
			}
		})
	}
}

func TestToTextFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected string
	}{
		// toText removes quotes from strings
		{"simple string", &StringValue{Value: "hello"}, "hello"},
		{"string with newline", &StringValue{Value: "hello\nworld"}, "hello\nworld"},
		
		// Other types remain the same
		{"integer", &IntValue{Value: 42}, "42"},
		{"boolean", &BoolValue{Value: true}, "true"},
		{"list", &ListValue{Elements: []Value{
			&IntValue{Value: 1},
			&IntValue{Value: 2},
		}}, "[1, 2]"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toTextValue(tt.input)
			if result != tt.expected {
				t.Errorf("toTextValue(%s) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

func TestTruncation(t *testing.T) {
	// Create a very long string (> 80 chars)
	longStr := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyz"
	result := truncateIfNeeded(longStr)
	
	// Should preserve first 20 and last 20 chars
	if len(result) > maxWidth {
		t.Errorf("truncateIfNeeded did not truncate: len = %d", len(result))
	}
	
	expected := "abcdefghijklmnopqrst...ghijklmnopqrstuvwxyz"
	if result != expected {
		t.Errorf("truncateIfNeeded = %q, want %q", result, expected)
	}
}

func TestShowDeterminism(t *testing.T) {
	// Test that the same complex value produces identical output
	record := &RecordValue{Fields: map[string]Value{
		"z": &IntValue{Value: 3},
		"a": &ListValue{Elements: []Value{
			&IntValue{Value: 1},
			&StringValue{Value: "test"},
		}},
		"m": &BoolValue{Value: true},
	}}
	
	// Call show multiple times
	result1 := showValue(record, 0)
	result2 := showValue(record, 0)
	result3 := showValue(record, 0)
	
	if result1 != result2 || result2 != result3 {
		t.Errorf("showValue is not deterministic: %q vs %q vs %q", result1, result2, result3)
	}
	
	// Check that keys are sorted
	expected := `{a: [1, "test"], m: true, z: 3}`
	if result1 != expected {
		t.Errorf("showValue = %q, want %q", result1, expected)
	}
}