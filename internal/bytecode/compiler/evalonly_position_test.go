package compiler

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// A BC001 diagnostic needs file:line for a function the compiler never emitted
// a body for. The compile-error rollback clears LineInfo and the lower-error
// path never runs fc.compile, so the declaration position must be stamped on
// the prototype when it is registered and survive both EvalOnly paths.
func TestCompile_EvalOnlyKeepsDeclarationPosition(t *testing.T) {
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{
		{Name: "good", Exported: true, File: "m.ail", Line: 3, Return: stmt.LitInt{Value: 1}},
		{Name: "badCompile", File: "m.ail", Line: 7, Return: stmt.BuiltinCall{Name: "_dict_foo"}},
		{Name: "badLower", File: "m.ail", Line: 11, LowerError: "lower: unsupported shape"},
	}}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	want := map[string]struct {
		line     int
		evalOnly bool
	}{"good": {3, false}, "badCompile": {7, true}, "badLower": {11, true}}
	for _, p := range img.Prototypes {
		w, ok := want[p.Name]
		if !ok {
			continue
		}
		delete(want, p.Name)
		if p.File != "m.ail" || p.Line != w.line || p.EvalOnly != w.evalOnly {
			t.Errorf("%s: File=%q Line=%d EvalOnly=%v, want m.ail:%d EvalOnly=%v", p.Name, p.File, p.Line, p.EvalOnly, w.line, w.evalOnly)
		}
	}
	for name := range want {
		t.Errorf("prototype %s missing from image", name)
	}
}
