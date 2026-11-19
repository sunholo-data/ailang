package core

import (
	"strings"
	"testing"
)

// The #2375 shape shadows; after UniqueLocals it does not.
func TestCheckNoShadowing_BeforeAndAfterUniqueLocals(t *testing.T) {
	in := blockShadowCore()
	if err := CheckNoShadowing(in); err == nil || !strings.Contains(err.Error(), `"r" is bound while an outer "r"`) {
		t.Fatalf("shadowing input: %v", err)
	}
	if err := CheckNoShadowing(UniqueLocals(in)); err != nil {
		t.Fatalf("after UniqueLocals: %v", err)
	}
}

// A local named like a free reference is flagged; sibling scopes are not.
func TestCheckNoShadowing_FreeNamesAndSiblings(t *testing.T) {
	// let s = (let f = 2 in f) in f(s): the local f has the name of the free f.
	free := coreLet(1, "s", coreLet(2, "f", litInt(3, 2), coreVar(4, "f")),
		&App{CoreNode: node(5), Func: coreVar(6, "f"), Args: []CoreExpr{coreVar(7, "s")}})
	if err := CheckNoShadowing(free); err == nil || !strings.Contains(err.Error(), "free") {
		t.Fatalf("local named like a free reference: %v", err)
	}
	some := func(n string) CorePattern {
		return &ConstructorPattern{Name: "Some", Args: []CorePattern{&VarPattern{Name: n}}}
	}
	// match o { Some(x) => x, Some(x) => x } and two sibling lambdas \y. y.
	siblings := &Tuple{CoreNode: node(10), Elements: []CoreExpr{
		&Match{CoreNode: node(11), Scrutinee: coreVar(12, "o"), Arms: []MatchArm{
			{Pattern: some("x"), Body: coreVar(13, "x")},
			{Pattern: some("x"), Guard: coreVar(14, "x"), Body: coreVar(15, "x")},
		}},
		&Lambda{CoreNode: node(16), Params: []string{"y"}, Body: coreVar(17, "y")},
		&Lambda{CoreNode: node(18), Params: []string{"y"}, Body: coreVar(19, "y")},
	}}
	if err := CheckNoShadowing(siblings); err != nil {
		t.Fatalf("siblings flagged: %v", err)
	}
}

// Every binder form is checked: a parameter, a letrec member, a pattern
// variable of each pattern kind, under every node kind that holds one.
func TestCheckNoShadowing_EveryBinderAndNode(t *testing.T) {
	x := func(id uint64) CoreExpr { return coreVar(id, "x") }
	reuse := func(id uint64) CoreExpr { // binds x again
		return &Lambda{CoreNode: node(id), Params: []string{"x"}, Body: x(id + 1)}
	}
	holders := map[string]CoreExpr{
		"app":          &App{CoreNode: node(20), Func: x(21), Args: []CoreExpr{reuse(22)}},
		"if":           &If{CoreNode: node(20), Cond: x(21), Then: x(22), Else: reuse(23)},
		"binop":        &BinOp{CoreNode: node(20), Op: "+", Left: x(21), Right: reuse(22)},
		"unop":         &UnOp{CoreNode: node(20), Op: "-", Operand: reuse(21)},
		"intrinsic":    &Intrinsic{CoreNode: node(20), Op: OpAdd, Args: []CoreExpr{reuse(21)}},
		"record":       &Record{CoreNode: node(20), Fields: map[string]CoreExpr{"a": reuse(21)}},
		"access":       &RecordAccess{CoreNode: node(20), Record: reuse(21), Field: "a"},
		"update":       &RecordUpdate{CoreNode: node(20), Base: x(21), Updates: map[string]CoreExpr{"a": reuse(22)}},
		"list":         &List{CoreNode: node(20), Elements: []CoreExpr{reuse(21)}},
		"array":        &Array{CoreNode: node(20), Elements: []CoreExpr{reuse(21)}},
		"tuple":        &Tuple{CoreNode: node(20), Elements: []CoreExpr{reuse(21)}},
		"dictabs":      &DictAbs{CoreNode: node(20), Body: reuse(21)},
		"dictapp":      &DictApp{CoreNode: node(20), Dict: x(21), Args: []CoreExpr{reuse(22)}},
		"forall":       &Forall{CoreNode: node(20), Var: "i", Lo: x(21), Hi: x(22), Body: reuse(23)},
		"letrec":       &LetRec{CoreNode: node(20), Bindings: []RecBinding{{Name: "x", Value: x(21)}}, Body: x(22)},
		"let":          coreLet(20, "x", x(21), x(22)),
		"pattern-var":  &Match{CoreNode: node(20), Scrutinee: x(21), Arms: []MatchArm{{Pattern: &VarPattern{Name: "x"}, Body: x(22)}}},
		"pattern-list": &Match{CoreNode: node(20), Scrutinee: x(21), Arms: []MatchArm{{Pattern: &ListPattern{Elements: []CorePattern{&VarPattern{Name: "x"}}}, Body: x(22)}}},
		"pattern-tail": &Match{CoreNode: node(20), Scrutinee: x(21), Arms: []MatchArm{{Pattern: &ListPattern{Tail: patPtr(&VarPattern{Name: "x"})}, Body: x(22)}}},
		"pattern-rec":  &Match{CoreNode: node(20), Scrutinee: x(21), Arms: []MatchArm{{Pattern: &RecordPattern{Fields: map[string]CorePattern{"a": &VarPattern{Name: "x"}}}, Body: x(22)}}},
		"pattern-tup":  &Match{CoreNode: node(20), Scrutinee: x(21), Arms: []MatchArm{{Pattern: &TuplePattern{Elements: []CorePattern{&VarPattern{Name: "x"}}}, Body: x(22)}}},
		"pattern-ctor": &Match{CoreNode: node(20), Scrutinee: x(21), Arms: []MatchArm{{Pattern: &ConstructorPattern{Name: "C", Args: []CorePattern{&VarPattern{Name: "x"}}}, Body: x(22)}}},
	}
	for name, h := range holders {
		decl := &Lambda{CoreNode: node(1), Params: []string{"x"}, Body: h}
		if err := CheckNoShadowing(decl); err == nil {
			t.Errorf("%s: a second binder of x was not flagged", name)
		}
	}
	if err := CheckNoShadowing(exprShim{}); err == nil {
		t.Error("an unknown node type passed the check")
	}
}

func patPtr(p CorePattern) *CorePattern { return &p }

// UniqueProgramLocals renames inside every declaration value, keeps
// top-level names, and returns the program itself when nothing changed.
func TestUniqueProgramLocals(t *testing.T) {
	clean := &Program{Decls: []CoreExpr{coreLet(1, "f", &Lambda{CoreNode: node(2), Params: []string{"x"}, Body: coreVar(3, "x")}, coreVar(4, "f"))}}
	if UniqueProgramLocals(clean) != clean {
		t.Fatal("a program without shadowing was copied")
	}
	if UniqueProgramLocals(nil) != nil {
		t.Fatal("nil program")
	}
	shadowLet := coreLet(10, "g", blockShadowCore(), coreVar(9, "g"))
	shadowRec := &LetRec{CoreNode: node(11), Body: &Lit{Kind: UnitLit}, Bindings: []RecBinding{
		{Name: "h", Value: &Lambda{CoreNode: node(12), Params: []string{"n"}, Body: coreVar(13, "n")}},
		{Name: "k", Value: blockShadowCore()},
	}}
	expr := &Tuple{CoreNode: node(30), Elements: []CoreExpr{blockShadowCore()}} // a top-level expression (REPL)
	replLet := blockShadowCore()                                                // `let r = 1 in …`: an expression, not a declaration
	recExpr := &LetRec{CoreNode: node(31), Bindings: []RecBinding{{Name: "h", Value: coreVar(32, "h")}}, Body: blockShadowCore()}
	prog := &Program{Decls: []CoreExpr{clean.Decls[0], shadowLet, shadowRec, expr, replLet, recExpr}}
	out := UniqueProgramLocals(prog)
	if out == prog || out.Decls[0] != prog.Decls[0] {
		t.Fatal("copy-on-write: changed program not copied, or unchanged declaration copied")
	}
	if l := out.Decls[1].(*Let); l.Name != "g" || CheckNoShadowing(l.Value) != nil {
		t.Fatalf("let declaration: %s", l)
	}
	r := out.Decls[2].(*LetRec)
	if r.Bindings[0].Name != "h" || r.Bindings[0].Value != shadowRec.Bindings[0].Value || CheckNoShadowing(r.Bindings[1].Value) != nil {
		t.Fatalf("letrec declaration: %s", r)
	}
	if CheckNoShadowing(out.Decls[3]) != nil {
		t.Fatal("top-level expression not renamed")
	}
	if l := out.Decls[4].(*Let); l.Name != "r" || CheckNoShadowing(l) != nil {
		t.Fatalf("a top-level let expression must be renamed inside its body, keeping its own name: %s", l)
	}
	if CheckNoShadowing(out.Decls[5]) != nil {
		t.Fatal("a letrec with a real body is an expression and must be renamed whole")
	}
	if CheckNoShadowing(prog.Decls[1].(*Let).Value) == nil {
		t.Fatal("input program mutated")
	}
}

func TestFreeVars(t *testing.T) {
	// \x. f(x, y): free f and y, not x.
	lam := &Lambda{CoreNode: node(1), Params: []string{"x"},
		Body: &App{CoreNode: node(2), Func: coreVar(3, "f"), Args: []CoreExpr{coreVar(4, "x"), coreVar(5, "y")}}}
	fv := FreeVars(lam)
	if !fv["f"] || !fv["y"] || fv["x"] || len(fv) != 2 {
		t.Fatalf("FreeVars = %v", fv)
	}
}
