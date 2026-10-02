package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// Lowered-IR shape tests for m-vm-match-lowering. End-to-end parity (the
// strict VM agrees with the evaluator) lives in
// cmd/ailang/run_bytecode_match_test.go; these pin the structural
// invariants that parity depends on.

func patPtr(p core.CorePattern) *core.CorePattern { return &p }

func consPat(head core.CorePattern, tail core.CorePattern) *core.ListPattern {
	return &core.ListPattern{Elements: []core.CorePattern{head}, Tail: patPtr(tail)}
}

// gtGuard builds the Core guard `name > n`.
func gtGuard(id uint64, name string, n int64) core.CoreExpr {
	return &core.BinOp{CoreNode: core.CoreNode{NodeID: id}, Op: ">", Left: coreVar(id+1, name), Right: litInt(id+2, n)}
}

// exprRefs reports whether e reads the variable name.
func exprRefs(e stmt.Expr, name string) bool {
	switch e := e.(type) {
	case stmt.VarRef:
		return e.Name == name
	case stmt.BinOp:
		return exprRefs(e.Left, name) || exprRefs(e.Right, name)
	case stmt.UnOp:
		return exprRefs(e.Operand, name)
	case stmt.FieldAccess:
		return exprRefs(e.Record, name)
	case stmt.BuiltinCall:
		for _, a := range e.Args {
			if exprRefs(a, name) {
				return true
			}
		}
	case stmt.ADTTagEq:
		return exprRefs(e.Value, name)
	}
	return false
}

// walkStmtExprs visits every expression in a statement tree.
func walkStmtExprs(ss []stmt.Stmt, visit func(stmt.Expr)) {
	var expr func(e stmt.Expr)
	expr = func(e stmt.Expr) {
		if e == nil {
			return
		}
		visit(e)
		switch e := e.(type) {
		case stmt.BinOp:
			expr(e.Left)
			expr(e.Right)
		case stmt.UnOp:
			expr(e.Operand)
		case stmt.FieldAccess:
			expr(e.Record)
		case stmt.BuiltinCall:
			for _, a := range e.Args {
				expr(a)
			}
		case stmt.ADTTagEq:
			expr(e.Value)
		}
	}
	for _, s := range ss {
		switch s := s.(type) {
		case stmt.VarDecl:
			expr(s.Value)
		case stmt.AssignStmt:
			expr(s.Value)
		case stmt.ReturnStmt:
			expr(s.Value)
		case stmt.IfStmt:
			expr(s.Cond)
			walkStmtExprs(s.Then, visit)
			walkStmtExprs(s.Else, visit)
		case stmt.SwitchStmt:
			expr(s.Scrutinee)
			for _, c := range s.Cases {
				walkStmtExprs(c.Body, visit)
			}
			walkStmtExprs(s.Default, visit)
		}
	}
}

// declaredBefore reports whether, walking ss in order, a VarDecl for name
// appears before the first IfStmt whose condition reads name.
func declaredBefore(ss []stmt.Stmt, name string) (declared, used bool) {
	for _, s := range ss {
		switch s := s.(type) {
		case stmt.VarDecl:
			if s.Name == name {
				return true, false
			}
		case stmt.IfStmt:
			if exprRefs(s.Cond, name) {
				return false, true
			}
			if d, u := declaredBefore(s.Then, name); d || u {
				return d, u
			}
			if d, u := declaredBefore(s.Else, name); d || u {
				return d, u
			}
		}
	}
	return false, false
}

// No constructor tag check may be spelled as a record field read: the VM's
// by-name _record_get rejects ADT values (#1503).
func TestMatchLowering_NoRecordTagAccess(t *testing.T) {
	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "xs"),
		Arms: []core.MatchArm{
			{Pattern: consPat(&core.ConstructorPattern{Name: "Some", Args: []core.CorePattern{&core.VarPattern{Name: "x"}}}, &core.WildcardPattern{}), Body: coreVar(3, "x")},
			{Pattern: &core.ListPattern{Elements: []core.CorePattern{&core.ConstructorPattern{Name: "None"}}}, Body: litInt(4, 0)},
			{Pattern: &core.WildcardPattern{}, Body: litInt(5, 1)},
		},
	}
	tags := 0
	walkStmtExprs([]stmt.Stmt{LowerMatchStmt(m, makeCTI(nil))}, func(e stmt.Expr) {
		if fa, ok := e.(stmt.FieldAccess); ok && fa.Field == "Tag" {
			t.Errorf("tag check lowered as record field access: %#v", fa)
		}
		if _, ok := e.(stmt.ADTTagEq); ok {
			tags++
		}
	})
	if tags != 2 {
		t.Errorf("expected 2 ADTTagEq checks (Some, None), got %d", tags)
	}
}

// `a :: b :: rest` binds every variable and needs one `len >= 2` check
// (#1420, #1505, #1517).
func TestMatchLowering_NestedConsBindsAll(t *testing.T) {
	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "xs"),
		Arms: []core.MatchArm{
			{Pattern: consPat(&core.VarPattern{Name: "a"}, consPat(&core.VarPattern{Name: "b"}, &core.VarPattern{Name: "rest"})), Body: coreVar(3, "b")},
			{Pattern: &core.WildcardPattern{}, Body: litInt(4, 0)},
		},
	}
	ifStmt, ok := LowerMatchStmt(m, makeCTI(nil)).(stmt.IfStmt)
	if !ok {
		t.Fatalf("expected if-chain")
	}
	cond, ok := ifStmt.Cond.(stmt.BinOp)
	if !ok || cond.Op != stmt.OpGte {
		t.Fatalf("expected a single len >= 2 check, got %s", stmtCondString(ifStmt.Cond))
	}
	if n, ok := cond.Right.(stmt.LitInt); !ok || n.Value != 2 {
		t.Errorf("expected length bound 2, got %#v", cond.Right)
	}
	declared := map[string]bool{}
	for _, s := range ifStmt.Then {
		if vd, ok := s.(stmt.VarDecl); ok {
			declared[vd.Name] = true
		}
	}
	for _, v := range []string{"a", "b", "rest"} {
		if !declared[v] {
			t.Errorf("expected VarDecl for %q, got %#v", v, ifStmt.Then)
		}
	}
}

// A guard that reads its arm's bindings must be evaluated after them; the
// evaluator binds first and falls through to the next arm on a false guard.
func TestMatchLowering_GuardAfterBindings(t *testing.T) {
	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "xs"),
		Arms: []core.MatchArm{
			{Pattern: consPat(&core.VarPattern{Name: "x"}, &core.WildcardPattern{}), Guard: gtGuard(10, "x", 5), Body: litStr(3, "big")},
			{Pattern: &core.WildcardPattern{}, Body: litStr(4, "small")},
		},
	}
	out := LowerMatchStmt(m, makeCTI(nil))
	declared, usedFirst := declaredBefore([]stmt.Stmt{out}, "x")
	if usedFirst || !declared {
		t.Fatalf("guard reads x before its VarDecl (declared=%v, usedFirst=%v): %#v", declared, usedFirst, out)
	}
	// The fall-through to the wildcard arm must survive a false guard: the
	// wildcard body is reachable outside the arm's own Then branch.
	block, ok := out.(stmt.IfStmt)
	if !ok || len(block.Then) < 3 {
		t.Fatalf("expected flag decl + guarded arm + fall-through, got %#v", out)
	}
	if fall, ok := block.Then[len(block.Then)-1].(stmt.IfStmt); !ok {
		t.Errorf("expected trailing fall-through If, got %#v", block.Then[len(block.Then)-1])
	} else if _, ok := fall.Cond.(stmt.UnOp); !ok {
		t.Errorf("expected fall-through guarded by !matched flag, got %s", stmtCondString(fall.Cond))
	}
}

// A catch-all variable arm before a constructor arm keeps its source
// position (#1473): the match must not become a switch whose Default runs
// only when no tag matched.
func TestMatchLowering_VarArmBeforeCtorNotSwitch(t *testing.T) {
	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "j"),
		Arms: []core.MatchArm{
			{Pattern: &core.ConstructorPattern{Name: "Committed", Args: []core.CorePattern{&core.WildcardPattern{}}}, Body: litStr(3, "c")},
			{Pattern: &core.VarPattern{Name: "other"}, Body: litStr(4, "o")},
			{Pattern: &core.ConstructorPattern{Name: "Idle"}, Body: litStr(5, "idle")},
		},
	}
	if _, ok := LowerMatchStmt(m, makeCTI(nil)).(stmt.SwitchStmt); ok {
		t.Fatal("a non-final catch-all must not be lowered to a switch Default")
	}
}

// A final variable catch-all in a switch binds the scrutinee (#1473).
func TestMatchLowering_SwitchDefaultBindsVar(t *testing.T) {
	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "j"),
		Arms: []core.MatchArm{
			{Pattern: &core.ConstructorPattern{Name: "Planned", Args: []core.CorePattern{&core.WildcardPattern{}}}, Body: litStr(3, "p")},
			{Pattern: &core.VarPattern{Name: "other"}, Body: coreVar(4, "other")},
		},
	}
	sw, ok := LowerMatchStmt(m, makeCTI(nil)).(stmt.SwitchStmt)
	if !ok {
		t.Fatal("expected the fast switch path for a final unguarded catch-all")
	}
	if len(sw.Default) == 0 {
		t.Fatal("expected a Default body")
	}
	vd, ok := sw.Default[0].(stmt.VarDecl)
	if !ok || vd.Name != "other" {
		t.Fatalf("expected Default to bind 'other' first, got %#v", sw.Default[0])
	}
	if ref, ok := vd.Value.(stmt.VarRef); !ok || ref.Name != "j" {
		t.Errorf("expected 'other' bound to the scrutinee, got %#v", vd.Value)
	}
}

// Repeated tags or nested arguments leave the switch path: a switch case
// cannot fall through to a later case with the same tag.
func TestMatchLowering_SwitchEligibility(t *testing.T) {
	some := func(arg core.CorePattern) core.CorePattern {
		return &core.ConstructorPattern{Name: "Some", Args: []core.CorePattern{arg}}
	}
	none := &core.ConstructorPattern{Name: "None"}
	cases := []struct {
		name string
		arms []core.MatchArm
		want bool
	}{
		{"plain", []core.MatchArm{{Pattern: some(&core.VarPattern{Name: "x"})}, {Pattern: none}}, true},
		{"guarded ctor + final default", []core.MatchArm{{Pattern: some(&core.VarPattern{Name: "x"}), Guard: gtGuard(10, "x", 1)}, {Pattern: &core.WildcardPattern{}}}, true},
		{"repeated tag", []core.MatchArm{{Pattern: some(&core.VarPattern{Name: "x"}), Guard: gtGuard(10, "x", 1)}, {Pattern: some(&core.VarPattern{Name: "x"})}, {Pattern: none}}, false},
		{"literal arg", []core.MatchArm{{Pattern: some(&core.LitPattern{Value: int64(0)})}, {Pattern: &core.WildcardPattern{}}}, false},
		{"nested arg", []core.MatchArm{{Pattern: some(some(&core.VarPattern{Name: "x"}))}, {Pattern: &core.WildcardPattern{}}}, false},
		{"guarded catch-all", []core.MatchArm{{Pattern: none}, {Pattern: &core.VarPattern{Name: "o"}, Guard: gtGuard(10, "o", 1)}}, false},
		{"all catch-all", []core.MatchArm{{Pattern: &core.VarPattern{Name: "x"}}}, false},
	}
	for _, c := range cases {
		if got := switchEligible(c.arms); got != c.want {
			t.Errorf("%s: switchEligible = %v, want %v", c.name, got, c.want)
		}
	}
}
