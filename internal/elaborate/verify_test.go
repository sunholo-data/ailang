package elaborate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// VerifyANF must accept what the elaborator produces (#2407: since #2388 a
// let the program wrote stays nested as a let's value, and the verifier
// rejected it, so the REPL and the wasm playground refused the input).
func TestVerifyANF_AcceptsElaboratorOutput(t *testing.T) {
	inputs := []string{
		`{ let a = 1; a } + 2`,
		`let r = 1 in (let r = 2 in r * 10) * 7 + r`,
		`let r = 1 in let s = { let r = 2; r * 10 } in r + s`,
		`let r = 1 in [{ let r = 2; r }, r]`,
		`let r = 1 in ({ let r = 2; r }, r)`,
		`let r = 1 in {a: { let r = 2; r }, b: r}`,
		`let r = 1 in if { let r = 0; r > 0 } then r else r + 10`,
		`let r = 1 in match { let r = 5; r } { 5 => r, _ => 0 }`,
		`let r = 1 in let f = \x. { let r = x; r * 2 } + r in f(5)`,
		`let f = (let k = 1 in \x. x) in f(2)`,
		`let x = if true then 1 else 2 in x`,
		`let y = match 1 { 1 => 2, _ => 3 } in y`,
		`let t = (1, 2) in t`,
		`(1, 2)`,
		`match 3 { n if { let m = 2; n > m } => 1, _ => 0 }`,
	}
	for _, in := range inputs {
		p := parser.New(lexer.New(in, "<test>"))
		prog := p.Parse()
		if len(p.Errors()) > 0 {
			t.Fatalf("%s: parse: %v", in, p.Errors())
		}
		coreProg, err := NewElaborator().Elaborate(prog)
		if err != nil {
			t.Fatalf("%s: elaborate: %v", in, err)
		}
		if err := VerifyANF(coreProg); err != nil {
			t.Errorf("%s: %v", in, err)
		}
	}
}

// The verifier describes the elaborator on every module under examples/: a
// shape check the elaborator does not meet would reject real programs.
func TestVerifyANF_AcceptsExamples(t *testing.T) {
	var files []string
	_ = filepath.WalkDir(filepath.Join("..", "..", "examples"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".ail") {
			files = append(files, path)
		}
		return nil
	})
	if len(files) < 50 {
		t.Fatalf("found %d example files; the glob is wrong", len(files))
	}
	checked := 0
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		p := parser.New(lexer.New(string(src), f))
		file := p.ParseFile()
		if len(p.Errors()) > 0 || file == nil {
			continue // not parseable on its own; the parser tests cover it
		}
		coreProg, err := NewElaborator().ElaborateFile(file)
		if err != nil {
			continue
		}
		checked++
		if err := VerifyANF(coreProg); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	if checked < 50 {
		t.Fatalf("verified only %d example modules", checked)
	}
}

// Operands stay atomic: a non-atomic operand in any operand position is an
// error, so the verifier still checks something.
func TestVerifyANF_RejectsNonAtomicOperands(t *testing.T) {
	lit := func(n int) core.CoreExpr { return &core.Lit{Kind: core.IntLit, Value: n} }
	v := &core.Var{Name: "x"}
	call := &core.App{Func: &core.Var{Name: "f"}, Args: []core.CoreExpr{v}}
	nestedLet := &core.Let{Name: "a", Value: lit(1), Body: &core.Var{Name: "a"}}
	cases := map[string]core.CoreExpr{
		"call argument":   &core.App{Func: &core.Var{Name: "f"}, Args: []core.CoreExpr{call}},
		"called function": &core.App{Func: nestedLet, Args: []core.CoreExpr{v}},
		"operator left":   &core.BinOp{Op: "+", Left: call, Right: lit(1)},
		"operator right":  &core.BinOp{Op: "+", Left: lit(1), Right: nestedLet},
		"if condition":    &core.If{Cond: nestedLet, Then: lit(1), Else: lit(2)},
		"match scrutinee": &core.Match{Scrutinee: call, Arms: []core.MatchArm{{Pattern: &core.WildcardPattern{}, Body: lit(1)}}},
		"list element":    &core.List{Elements: []core.CoreExpr{v, call}},
		"tuple element":   &core.Tuple{Elements: []core.CoreExpr{nestedLet, v}},
		"record field":    &core.Record{Fields: map[string]core.CoreExpr{"a": call}},
		"field access":    &core.RecordAccess{Record: call, Field: "a"},
		"in a let value":  &core.Let{Name: "s", Value: &core.BinOp{Op: "+", Left: call, Right: lit(1)}, Body: v},
		"in a let body":   &core.Let{Name: "s", Value: lit(1), Body: &core.Tuple{Elements: []core.CoreExpr{call}}},
		"in a lambda":     &core.Lambda{Params: []string{"x"}, Body: &core.List{Elements: []core.CoreExpr{call}}},
		"in a branch":     &core.If{Cond: v, Then: &core.List{Elements: []core.CoreExpr{call}}, Else: lit(2)},
		"in a guard":      &core.Match{Scrutinee: v, Arms: []core.MatchArm{{Pattern: &core.WildcardPattern{}, Guard: &core.BinOp{Op: ">", Left: call, Right: lit(0)}, Body: lit(1)}}},
	}
	for name, expr := range cases {
		err := VerifyANF(&core.Program{Decls: []core.CoreExpr{expr}})
		if err == nil || !strings.Contains(err.Error(), "must be atomic") {
			t.Errorf("%s: want a 'must be atomic' error, got %v", name, err)
		}
	}
}
