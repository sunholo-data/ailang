package pipeline_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// M-SOUND-NUMERIC-LITERALS: an integer literal is `Num a => a`. Wherever the
// type checker resolves it at Float, the runtime must see a float — before,
// these all type-checked and then failed with "expected float arguments" /
// "missing dictionary method: prelude::Fractional::Int::add", or silently
// produced an Int where the type said float.
//
// Expression mode (REPL / `ailang eval`) has no module declarations, so every
// let — including the outermost — is under the integer-literal restriction.
func TestNumericLiteral_ExpressionMode(t *testing.T) {
	cases := []struct {
		name, code, want string
	}{
		{"ground literal at float", `2.5 >= 0`, "true"},
		{"literal left operand", `2 * 1.5`, "3.0"},
		{"let-bound literal used at float", `let w = 70 in w / 2.5`, "28.0"},
		{"let-bound lambda with literal used at float", `let inc = \x. x + 1 in inc(1.5)`, "2.5"},
		{"tuple of literals at float", `let t: (float, float) = (1, 2) in t`, "(1.0, 2.0)"},
		{"list of literals at float", `let xs: [float] = [1, 2.5] in xs`, "[1.0, 2.5]"},
		// Literal-free let-polymorphism keeps working at two types (M-POLY-ARITH).
		{"literal-free poly lambda", `let add = \x. \y. x + y in add(3.14)(2.71)`, "5.85"},
		{"int literals stay int", `let n = 7 in n / 2`, "3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, pipeline.Source{Code: c.code})
			if err != nil {
				t.Fatalf("%s: %v", c.code, err)
			}
			if got := res.Value.String(); got != c.want {
				t.Errorf("%s = %s, want %s", c.code, got, c.want)
			}
		})
	}
}

// A let whose literal makes it monomorphic cannot also be used at Int: that
// is now a type error, where the Float use used to crash at runtime.
func TestNumericLiteral_RestrictedLetAtTwoTypesIsTypeError(t *testing.T) {
	code := `let k: int = 3 in let inc = \x. x + 1 in let a = inc(1.5) in inc(k)`
	_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeEval}, pipeline.Source{Code: code})
	if err == nil {
		t.Fatalf("%s: expected a type error (inc used at float and at int), got none", code)
	}
	if !strings.Contains(err.Error(), "float") || !strings.Contains(err.Error(), "int") {
		t.Errorf("type error should name the float/int conflict, got: %v", err)
	}
}
