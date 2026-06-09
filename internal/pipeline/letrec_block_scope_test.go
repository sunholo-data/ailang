package pipeline

import "testing"

// TestLetRecBlockScope guards M-LETREC-BLOCK-SCOPE.
//
// A statement-form `letrec` inside a block (`letrec f = ...; <rest>`) must keep
// the bound name in scope both recursively (in its own value) and for the rest
// of the block. The elaborator's normalizeBlock previously had no *ast.LetRec
// case, so the binding was dropped (bound to a discarded _block_N wildcard) and
// any later use — including the recursive self-call — failed type-checking with
// "undefined variable". This is the form the teaching prompt shows, yet it had
// zero examples/ coverage (the existing letrec example used only the always-
// working `letrec f = ... in ...` expression form), so CI never caught it.
//
// Sources avoid stdlib imports (return int, no println) so the check isolates
// letrec scoping rather than module resolution in the test sandbox.
func TestLetRecBlockScope(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "simple recursive letrec statement",
			code: `module test
export func main() -> int {
  letrec factorial = \n. if n <= 1 then 1 else n * factorial(n - 1);
  factorial(5)
}`,
		},
		{
			name: "curried recursive letrec statement",
			code: `module test
export func main() -> int {
  letrec sumTo = \i. \acc. if i <= 0 then acc else sumTo(i - 1)(acc + i);
  sumTo(10)(0)
}`,
		},
		{
			name: "letrec after another statement in the block",
			code: `module test
export func main() -> int {
  let base = 5;
  letrec countdown = \k. if k <= 0 then 0 else countdown(k - 1);
  countdown(base)
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Run(Config{Mode: ModeCheck}, Source{Code: tt.code, IsREPL: false})
			if err != nil {
				t.Fatalf("block-form letrec should type-check, got: %v", err)
			}
		})
	}
}
