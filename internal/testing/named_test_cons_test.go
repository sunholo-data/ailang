package testing

import "testing"

// Regression suite for #1914: `x :: xs` inside a `test "name" { }` body failed
// with PAR_NO_PREFIX_PARSE "unexpected token in expression: ::". The parser
// desugars a cons to FuncCall{::, [x, xs]}, PrintAILANGSource printed that as
// `::(x, xs)`, and that prefix spelling does not parse as an expression. The
// same expression always worked in a function body.

func consFixtureSource(body string) string {
	return `module cons_fixture

import std/list (length)

pure func identity(xs: [int]) -> [int] = xs

test "cons" {
` + body + `
}
`
}

func TestNamedTest_ConsInBody(t *testing.T) {
	for name, body := range map[string]string{
		// The issue's repro, verbatim.
		"issue 1914":       `  let x = 1; let xs = [2]; length(identity(x :: xs)) == 2`,
		"bare":             `  1 :: [2] == [1, 2]`,
		"parenthesised":    `  let x = 1; let xs = [2]; length(identity((x :: xs))) == 2`,
		"let binding":      `  let ys = 1 :: [2]; length(ys) == 2`,
		"right-assoc":      `  1 :: 2 :: [] == [1, 2]`,
		"operand of ++":    `  (0 :: [1]) ++ [2] == [0, 1, 2]`,
		"inside an assert": `  assert 1 :: [2] == [1, 2]`,
		// Reviewer probes on PR #2222: an if-operand of :: and an if-base of
		// a field access keep their meaning only once both go through
		// printOperand (they failed with "No instance for Num[list[int]]" and
		// a unification error).
		"if operand of ::":        `  (if true then 1 else 2) :: [5] == [1, 5]`,
		"if base of field access": `  (if true then {a: 1} else {a: 2}).a == 1`,
	} {
		t.Run(name, func(t *testing.T) {
			result := runInlineTestsOnSource(t, consFixtureSource(body))
			if result.FailedTests != 0 || result.PassedTests != 1 {
				t.Errorf("passed=%d failed=%d; first error: %s",
					result.PassedTests, result.FailedTests, firstFailureError(result))
			}
		})
	}
}

// TestNamedTest_ConsFailureStillFails is the negative control: a false cons
// equation fails the test (the fix must not turn the body into a pass).
func TestNamedTest_ConsFailureStillFails(t *testing.T) {
	result := runInlineTestsOnSource(t, consFixtureSource(`  1 :: [2] == [2, 1]`))
	if result.FailedTests != 1 || result.PassedTests != 0 {
		t.Fatalf("passed=%d failed=%d; a false cons equation must fail", result.PassedTests, result.FailedTests)
	}
}
