package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

func elaborateCheckFile(t *testing.T, src string) (*Elaborator, *core.Program) {
	t.Helper()
	p := parser.New(lexer.New(src, "m.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	elab := NewElaborator()
	prog, err := elab.ElaborateFile(file)
	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}
	return elab, prog
}

// #2232: every test-only body is elaborated, in the harness's entry shape,
// for the module pipeline to type-check; none of them enters the program.
func TestCheckBodies_ElaboratedInHarnessShape(t *testing.T) {
	elab, prog := elaborateCheckFile(t, `module m
export pure func f(x: int) -> int
  tests [(1, 1), (2, 2)]
  properties [ forall(y: int) => f(y) == y ]
{ x }
export pure func g(a: int, b: int) -> int tests [((1, 2), 3)] { a + b }
test "t" {
  let a = f(1);
  assert a == 1;
  a > 0
}
property "p" { forall(n: int, m: int) => g(n, m) == g(m, n) }
`)
	bodies := elab.CheckBodies()
	type want struct {
		kind, name string
		params     int
		pure       bool
	}
	wants := []want{
		{"test", "t", 0, true},
		{"property", "p", 2, true},
		{"property", "f property 1", 1, true},
		{"test row", "f row 1", -1, false},
		{"test row", "f row 2", -1, false},
		{"test row", "g row 1", -1, false},
	}
	if len(bodies) != len(wants) {
		t.Fatalf("got %d bodies, want %d: %+v", len(bodies), len(wants), bodies)
	}
	for i, w := range wants {
		b := bodies[i]
		if b.Kind != w.kind || b.Name != w.name || b.Pure != w.pure || b.Pos.Line == 0 {
			t.Errorf("body %d = %s %q pure=%v line=%d, want %s %q pure=%v", i, b.Kind, b.Name, b.Pure, b.Pos.Line, w.kind, w.name, w.pure)
		}
		lam, isLam := b.Expr.(*core.Lambda)
		switch {
		case w.params < 0 && isLam:
			t.Errorf("%s: a row is an expression, not an entry lambda", b.Name)
		case w.params >= 0 && (!isLam || len(lam.Params) != w.params):
			t.Errorf("%s: want a %d-parameter entry lambda, got %T", b.Name, w.params, b.Expr)
		}
	}
	// Property binders carry their annotated types, as the harness's entry does.
	if annots := elab.GetParamTypeAnnotations()[bodies[1].Expr.ID()]; len(annots) != 2 {
		t.Errorf("property binder annotations = %v", annots)
	}
	// The bodies are checked only: nothing new is declared.
	for _, d := range prog.Decls {
		if strings.Contains(d.String(), "__namedtest") {
			t.Errorf("a test body leaked into the program: %s", d)
		}
	}
	if len(prog.Decls) != 2 {
		t.Errorf("program has %d decls, want the 2 functions", len(prog.Decls))
	}
}

// A module of test blocks only (no functions, no imports) still has its
// bodies elaborated; a second ElaborateFile does not keep the first's.
func TestCheckBodies_TestOnlyModuleAndReset(t *testing.T) {
	p := parser.New(lexer.New("module m\ntest \"only\" { 1 == 1 }\n", "m.ail"))
	file := p.ParseFile()
	elab := NewElaborator()
	if _, err := elab.ElaborateFile(file); err != nil {
		t.Fatal(err)
	}
	if got := elab.CheckBodies(); len(got) != 1 || got[0].Name != "only" {
		t.Fatalf("test-only module bodies = %+v", got)
	}
	p2 := parser.New(lexer.New("module m\nexport pure func k(x: int) -> int = x\n", "m.ail"))
	if _, err := elab.ElaborateFile(p2.ParseFile()); err != nil {
		t.Fatal(err)
	}
	if got := elab.CheckBodies(); len(got) != 0 {
		t.Fatalf("bodies of the previous file survived: %+v", got)
	}
}

func TestTestBodyError_IsAReport(t *testing.T) {
	b := CheckBody{Kind: "test", Name: "t", Pos: ast.Pos{File: "m.ail", Line: 3, Column: 1}}
	rep, ok := ailerrors.AsReport(TestBodyError(b, errString("boom")))
	if !ok || rep.Code != CodeTestBody || rep.Span.Start.Line != 3 || !strings.Contains(rep.Message, `test "t" at m.ail:3:1 does not type-check: boom`) {
		t.Fatalf("report = %+v", rep)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
