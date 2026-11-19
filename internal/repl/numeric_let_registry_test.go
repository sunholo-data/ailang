package repl

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// #2378: the module registry compiles no numeric variants, so a let-bound
// function with its own integer literal keeps one type decided by its uses
// there (literalWithhold), as before #2393, instead of being generalised and
// run at its default int: used at float only, it computes at float.
func TestModuleRegistry_LetBoundLiteralKeepsOneType(t *testing.T) {
	reg := NewModuleRegistry()
	code := "module numlet\n\nexport pure func atFloat() -> string { let f = \\x. x + 1; show(f(2.5)) }\n"
	if _, err := reg.LoadModule("numlet", code); err != nil {
		t.Fatalf("load: %v", err)
	}
	got, err := reg.InvokeExport("numlet", "atFloat", []eval.Value{})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if s, ok := got.(*eval.StringValue); !ok || s.Value != "3.5" {
		t.Fatalf("atFloat() = %v, want 3.5", got)
	}
}
