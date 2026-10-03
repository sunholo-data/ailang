package testing

import (
	"math"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Regression suite for #1448: a whole-number float literal (4.0) inside a
// `test "name" { }` block evaluated as an Int. EvaluateNamedTestBodyExprs prints
// the folded body back to AILANG source; Literal.String() spelled float64(4) as
// "4", which re-parsed as an IntLit.

func floatFixtureSource(body string) string {
	return `module float_fixture

import std/math (sqrt)

export pure func root(x: float) -> float = sqrt(x)

test "float literal" {
` + body + `
}
`
}

func TestNamedTest_WholeFloatLiteral(t *testing.T) {
	for name, body := range map[string]string{
		"direct arg":  `  root(4.0) == 2.0`,
		"let binding": `  let x = 4.0; root(x) == 2.0`,
		"fractional":  `  root(2.25) == 1.5`,
		"zero":        `  root(0.0) == 0.0`,
	} {
		t.Run(name, func(t *testing.T) {
			result := runInlineTestsOnSource(t, floatFixtureSource(body))
			if result.FailedTests != 0 || result.PassedTests != 1 {
				t.Errorf("passed=%d failed=%d; first error: %s",
					result.PassedTests, result.FailedTests, firstFailureError(result))
			}
		})
	}
}

// TestPrintAILANGSource_FloatLiteralKeepsFloatSpelling pins the printer itself:
// every float literal must print in a form the lexer reads back as a float.
func TestPrintAILANGSource_FloatLiteralKeepsFloatSpelling(t *testing.T) {
	cases := map[float64]string{
		4.0:          "4.0",
		0.0:          "0.0",
		100.0:        "100.0",
		2.25:         "2.25",
		1e20:         "1e+20",
		1e21:         "1e+21",
		math.Inf(1):  "(1.0 / 0.0)",
		math.Inf(-1): "(-1.0 / 0.0)",
	}
	for v, want := range cases {
		got := PrintAILANGSource(&ast.Literal{Kind: ast.FloatLit, Value: v})
		if got != want {
			t.Errorf("PrintAILANGSource(FloatLit %v) = %q, want %q", v, got, want)
		}
	}
	if got := PrintAILANGSource(&ast.Literal{Kind: ast.FloatLit, Value: math.NaN()}); got != "(0.0 / 0.0)" {
		t.Errorf("PrintAILANGSource(FloatLit NaN) = %q, want %q", got, "(0.0 / 0.0)")
	}
}
