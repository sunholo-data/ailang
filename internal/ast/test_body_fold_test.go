package ast

import "testing"

// The fold shared by the test harness and the compiler (#2232). The harness's
// own tests (internal/testing) pin the printed shapes end to end; these pin
// the AST the compiler type-checks.

func foldIdent(n string) Expr { return &Identifier{Name: n, Pos: Pos{Line: 1}} }

func TestFoldTestBody_LegacyPath(t *testing.T) {
	if got, checks := FoldTestBody(nil); got != nil || checks != nil {
		t.Fatalf("empty body: got %v, %v", got, checks)
	}
	// lets + a final expression: chained lets, no checks, final not forced bool
	body := []Expr{&Let{Name: "a", Value: foldIdent("x")}, &LetRec{Name: "f", Value: foldIdent("g")}, foldIdent("a")}
	got, checks := FoldTestBody(body)
	if checks != nil {
		t.Fatalf("legacy path must report nil checks, got %v", checks)
	}
	let, ok := got.(*Let)
	if !ok || let.Name != "a" {
		t.Fatalf("want let a ..., got %T %v", got, got)
	}
	rec, ok := let.Body.(*LetRec)
	if !ok || rec.Name != "f" || rec.Body != body[2] {
		t.Fatalf("want letrec f ... in a, got %v", let.Body)
	}
	if body[0].(*Let).Body != nil {
		t.Fatalf("the parser's let was mutated")
	}
}

func TestFoldBodyExprs_BareNonFinalIsSequenced(t *testing.T) {
	got := FoldBodyExprs([]Expr{foldIdent("side"), foldIdent("final")})
	let, ok := got.(*Let)
	if !ok || let.Name != "_seq" || let.Body.(*Identifier).Name != "final" {
		t.Fatalf("want let _seq = side in final, got %v", got)
	}
	if FoldBodyExprs(nil) != nil {
		t.Fatalf("empty body must fold to nil")
	}
}

func TestFoldTestBody_SentinelPath(t *testing.T) {
	// assert c1; let a = v; c2 (bare, final)  →  if c1 then let a = v in (if c2 then 0 else 2) else 1
	body := []Expr{
		&AssertStmt{Condition: foldIdent("c1"), Pos: Pos{Line: 2}},
		&Let{Name: "a", Value: foldIdent("v")},
		&LetRec{Name: "r", Value: foldIdent("w")},
		foldIdent("c2"),
	}
	got, checks := FoldTestBody(body)
	if len(checks) != 2 || checks[0].Ordinal != 1 || !checks[0].IsAssert || checks[1].Ordinal != 2 || checks[1].IsAssert {
		t.Fatalf("checks = %+v", checks)
	}
	if checks[0].Cond.(*Identifier).Name != "c1" || checks[1].Cond.(*Identifier).Name != "c2" {
		t.Fatalf("check conditions = %v, %v", checks[0].Cond, checks[1].Cond)
	}
	outer, ok := got.(*If)
	if !ok || outer.Condition.(*Identifier).Name != "c1" || outer.Else.(*Literal).Value != int64(1) {
		t.Fatalf("outer = %v", got)
	}
	inner := outer.Then.(*Let).Body.(*LetRec).Body.(*If)
	if inner.Then.(*Literal).Value != int64(0) || inner.Else.(*Literal).Value != int64(2) {
		t.Fatalf("final check = %v", inner)
	}
}

func TestFoldTestBody_BareNonFinalIsACheck(t *testing.T) {
	got, checks := FoldTestBody([]Expr{foldIdent("c1"), foldIdent("c2")})
	if len(checks) != 2 {
		t.Fatalf("both bare expressions are checks, got %+v", checks)
	}
	if outer := got.(*If); outer.Condition.(*Identifier).Name != "c1" {
		t.Fatalf("outer = %v", got)
	}
	// A trailing let is a degenerate body: kept as written, not a check.
	trailing := &Let{Name: "z", Value: foldIdent("v")}
	got, checks = FoldTestBody([]Expr{&AssertStmt{Condition: foldIdent("c")}, trailing})
	if len(checks) != 1 || got.(*If).Then != trailing {
		t.Fatalf("trailing let: got %v, checks %+v", got, checks)
	}
}
