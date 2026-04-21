package repl

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/runtime"
)

// TestConsExpression_REPLEval tests that :: expressions evaluate correctly
// through the REPL module registry with InvokeExport.
func TestConsExpression_REPLEval(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		arg      eval.Value
		expected string
	}{
		{
			name: "basic prepend",
			code: `module test
export func result(x: int) -> [int] = x :: [2, 3]`,
			arg:      &eval.IntValue{Value: 1},
			expected: "[1, 2, 3]",
		},
		{
			name: "chained cons",
			code: `module test
export func result(x: int) -> [int] = x :: 2 :: 3 :: []`,
			arg:      &eval.IntValue{Value: 1},
			expected: "[1, 2, 3]",
		},
		{
			name: "cons onto empty",
			code: `module test
export func result(x: int) -> [int] = x :: []`,
			arg:      &eval.IntValue{Value: 42},
			expected: "[42]",
		},
		{
			name: "cons in if branch",
			code: `module test
export func result(x: int) -> [int] =
  if x > 0 then x :: [2] else [3]`,
			arg:      &eval.IntValue{Value: 1},
			expected: "[1, 2]",
		},
		{
			name: "cons with string list",
			code: `module test
export func result(x: string) -> [string] = x :: ["b", "c"]`,
			arg:      &eval.StringValue{Value: "a"},
			expected: `[a, b, c]`,
		},
		{
			name: "cons in match pattern and expression",
			code: `module test
export func result(xs: [int]) -> [int] =
  match xs {
    [] => [],
    x :: rest => x :: x :: rest
  }`,
			arg:      &eval.ListValue{Elements: []eval.Value{&eval.IntValue{Value: 1}}},
			expected: "[1, 1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewModuleRegistry()
			_, err := reg.LoadModule("test", tt.code)
			if err != nil {
				t.Fatalf("LoadModule error: %v", err)
			}

			val, err := reg.InvokeExport("test", "result", []eval.Value{tt.arg})
			if err != nil {
				t.Fatalf("InvokeExport error: %v", err)
			}

			got := runtime.Show(val)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}
