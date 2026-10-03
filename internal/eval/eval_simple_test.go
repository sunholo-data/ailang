package eval

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
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

func TestLambdaClosures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic lambda evaluation
		{"identity lambda", `(\x. x)(42)`, "42"},
		{"arithmetic lambda", `(\x. x + 1)(5)`, "6"},
		// M-GAP2: \x y. is multi-param (tuple-style), use \x. \y. for curried
		{"curried lambda", `(\x. \y. x + y)(3)(4)`, "7"},
		{"multi-param lambda", `(\x y. x + y)(3, 4)`, "7"},

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
		// M-GAP2: \x y. is multi-param, use \x. \y. for curried partial application
		{"partial application closure", `let base = 100 in let add = \x. \y. x + y + base in add(1)(2)`, "103"},
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
