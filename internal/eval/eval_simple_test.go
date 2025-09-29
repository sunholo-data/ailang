package eval

import (
	"math"
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
)

// Helper function to evaluate string expressions for tests
func evalString(evaluator *SimpleEvaluator, input string) (string, error) {
	l := lexer.New(input, "test.ail")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		return "", p.Errors()[0]
	}

	result, err := evaluator.EvalProgram(program)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

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

func TestLambdaClosures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic lambda evaluation
		{"identity lambda", `(\x. x)(42)`, "42"},
		{"arithmetic lambda", `(\x. x + 1)(5)`, "6"},
		{"curried lambda", `(\x y. x + y)(3)(4)`, "7"},

		// Closure capture
		{"simple closure", `let y = 10 in (\x. x + y)(5)`, "15"},
		{"nested closure", `let a = 1 in let b = 2 in (\x. x + a + b)(3)`, "6"},
		{"closure with string", `let greeting = "Hello" in (\name. greeting ++ " " ++ name)("World")`, "Hello World"},

		// Multiple closures with shared environment
		{"shared environment", `let z = 100 in [(\x. x + z)(1), (\x. x * z)(2)]`, "[101, 200]"},

		// Higher-order functions
		{"higher-order closure", `let multiplier = (\n. \x. x * n) in multiplier(3)(4)`, "12"},
		{"function returning closure", `(\y. \x. x + y)(10)(5)`, "15"},

		// Closure with record access
		{"closure with record", `let person = {name: "Alice", age: 30} in (\prefix. prefix ++ person.name)("Ms. ")`, "Ms. Alice"},

		// Partial application preserving closures
		{"partial application closure", `let base = 100 in let add = \x y. x + y + base in add(1)(2)`, "103"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewSimple()
			result, err := evalString(evaluator, tt.input)

			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestLambdaClosureEnvironmentIsolation(t *testing.T) {
	// Test that different lambda instances have isolated environments
	input := `
		let createCounter = \start. 
			let count = start in
			\increment. count + increment
		in
		let counter1 = createCounter(0) in
		let counter2 = createCounter(100) in
		[counter1(1), counter2(5), counter1(3)]
	`

	evaluator := NewSimple()
	result, err := evalString(evaluator, input)

	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	expected := "[1, 105, 3]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
