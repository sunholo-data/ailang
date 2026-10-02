package lower

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// --- Match lowering tests ---

func TestLowerMatchStmt_Constructor(t *testing.T) {
	cti := makeCTI(nil)

	m := &core.Match{
		CoreNode:   core.CoreNode{NodeID: 1},
		Scrutinee:  coreVar(2, "c"),
		Exhaustive: true,
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{Name: "Red"},
				Body:    litInt(3, 0),
			},
			{
				Pattern: &core.ConstructorPattern{Name: "Green"},
				Body:    litInt(4, 1),
			},
			{
				Pattern: &core.WildcardPattern{},
				Body:    litInt(5, 2),
			},
		},
	}

	result := LowerMatchStmt(m, cti)
	sw, ok := result.(stmt.SwitchStmt)
	if !ok {
		t.Fatalf("expected SwitchStmt, got %T", result)
	}
	if len(sw.Cases) != 2 {
		t.Errorf("expected 2 cases, got %d", len(sw.Cases))
	}
	if sw.Cases[0].Tag != "Red" {
		t.Errorf("expected first case Red, got %s", sw.Cases[0].Tag)
	}
	if len(sw.Default) == 0 {
		t.Error("expected default branch")
	}
}

func TestLowerMatchStmt_ConstructorWithBindings(t *testing.T) {
	cti := makeCTI(nil)

	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "opt"),
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{
					Name: "Some",
					Args: []core.CorePattern{&core.VarPattern{Name: "x"}},
				},
				Body: coreVar(3, "x"),
			},
			{
				Pattern: &core.ConstructorPattern{Name: "None"},
				Body:    litInt(4, 0),
			},
		},
	}

	result := LowerMatchStmt(m, cti)
	sw, ok := result.(stmt.SwitchStmt)
	if !ok {
		t.Fatalf("expected SwitchStmt, got %T", result)
	}
	if len(sw.Cases) != 2 {
		t.Errorf("expected 2 cases, got %d", len(sw.Cases))
	}
	// First case should have a binding.
	if len(sw.Cases[0].Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(sw.Cases[0].Bindings))
	}
	if sw.Cases[0].Bindings[0].Name != "x" {
		t.Errorf("expected binding name x, got %s", sw.Cases[0].Bindings[0].Name)
	}
}

// TestLowerMatchStmt_ConstructorWithLiteralArg is the regression test for
// Bug A.2 (M-LOWER-FIX follow-up): a constructor pattern with a literal
// sub-argument (`Num(0) => true, _ => false`) must check the literal, not
// match any `Num(_)`. Since m-vm-match-lowering a refutable constructor
// argument routes the match to the if-chain, whose condition is the tag
// check AND the literal comparison on the positional field.
func TestLowerMatchStmt_ConstructorWithLiteralArg(t *testing.T) {
	cti := makeCTI(nil)

	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "e"),
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{
					Name: "Num",
					Args: []core.CorePattern{&core.LitPattern{Value: int64(0)}},
				},
				Body: litBool(3, true),
			},
			{
				Pattern: &core.WildcardPattern{},
				Body:    litBool(4, false),
			},
		},
	}

	result := LowerMatchStmt(m, cti)
	ifStmt, ok := result.(stmt.IfStmt)
	if !ok {
		t.Fatalf("expected if-chain IfStmt, got %T", result)
	}
	and, ok := ifStmt.Cond.(stmt.BinOp)
	if !ok || and.Op != stmt.OpAnd {
		t.Fatalf("expected tag-check && literal-check, got %s", stmtCondString(ifStmt.Cond))
	}
	if tag, ok := and.Left.(stmt.ADTTagEq); !ok || tag.Tag != "Num" {
		t.Errorf("expected ADTTagEq{Num} first, got %#v", and.Left)
	}
	eq, ok := and.Right.(stmt.BinOp)
	if !ok || eq.Op != stmt.OpEq {
		t.Fatalf("expected literal Eq second, got %#v", and.Right)
	}
	if fa, ok := eq.Left.(stmt.FieldAccess); !ok || fa.Field != "_0" {
		t.Errorf("expected literal compared against field _0, got %#v", eq.Left)
	}
	if v, ok := eq.Right.(stmt.LitInt); !ok || v.Value != 0 {
		t.Errorf("expected literal 0, got %#v", eq.Right)
	}
	// The wildcard arm is the else branch.
	if len(ifStmt.Else) == 0 {
		t.Error("expected the wildcard arm in the else branch, got empty")
	}
}

// TestLowerMatchStmt_ConsConstructorHead — M5 regression test.
//
// Reproduces the "unbound variable t" bug observed in docparse/main.ail:
// when a ListPattern element is a ConstructorPattern (e.g., TextBlock(t) :: rest),
// lowerPatternBindings previously only handled VarPattern elements,
// silently skipping constructor sub-patterns. This left inner bindings like
// `t` undeclared. Additionally, lowerPatternCond only checked list length
// but never verified the constructor tag on the head element.
//
// Shape of the failing pattern:
//
//	match blocks {
//	  [] => "none",
//	  TextBlock(t) :: rest => t.style,
//	  _ :: rest => "other"
//	}
func TestLowerMatchStmt_ConsConstructorHead(t *testing.T) {
	cti := makeCTI(nil)

	tail := core.CorePattern(&core.VarPattern{Name: "rest"})
	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "blocks"),
		Arms: []core.MatchArm{
			{
				// [] => "none"
				Pattern: &core.ListPattern{},
				Body:    litStr(3, "none"),
			},
			{
				// TextBlock(t) :: rest => t
				Pattern: &core.ListPattern{
					Elements: []core.CorePattern{
						&core.ConstructorPattern{
							Name: "TextBlock",
							Args: []core.CorePattern{&core.VarPattern{Name: "t"}},
						},
					},
					Tail: &tail,
				},
				Body: coreVar(4, "t"),
			},
			{
				// _ :: rest => "other"
				Pattern: &core.WildcardPattern{},
				Body:    litStr(5, "other"),
			},
		},
	}

	result := LowerMatchStmt(m, cti)
	// This is an if-chain (mixed pattern types, not pure constructors).
	ifStmt, ok := result.(stmt.IfStmt)
	if !ok {
		t.Fatalf("expected IfStmt (if-chain), got %T", result)
	}

	// The first arm is [] => "none" (empty list check).
	// The second arm is TextBlock(t) :: rest => t.
	// Walk to the second arm (the else branch of the first if).
	if len(ifStmt.Else) == 0 {
		t.Fatalf("expected else branch for second arm, got empty")
	}
	innerIf, ok := ifStmt.Else[0].(stmt.IfStmt)
	if !ok {
		t.Fatalf("expected inner IfStmt for second arm, got %T", ifStmt.Else[0])
	}

	// Bug 1 fix: condition must include a tag check for "TextBlock" (not just length).
	condStr := stmtCondString(innerIf.Cond)
	if condStr == "" {
		t.Fatalf("could not stringify condition: %#v", innerIf.Cond)
	}
	if !containsTagCheck(innerIf.Cond, "TextBlock") {
		t.Errorf("expected tag check for TextBlock in condition, got: %s", condStr)
	}

	// Bug 2 fix: the Then branch must declare `t` before referencing it.
	declared := map[string]bool{}
	for _, s := range innerIf.Then {
		if vd, ok := s.(stmt.VarDecl); ok {
			declared[vd.Name] = true
		}
	}
	if !declared["t"] {
		t.Errorf("expected VarDecl for 't' in Then branch, got stmts: %#v", innerIf.Then)
	}
	// Should also declare `rest`.
	if !declared["rest"] {
		t.Errorf("expected VarDecl for 'rest' in Then branch, got stmts: %#v", innerIf.Then)
	}
}

// containsTagCheck recursively checks whether an expression contains an
// ADTTagEq for tag.
func containsTagCheck(e stmt.Expr, tag string) bool {
	switch e := e.(type) {
	case stmt.ADTTagEq:
		return e.Tag == tag
	case stmt.BinOp:
		return containsTagCheck(e.Left, tag) || containsTagCheck(e.Right, tag)
	default:
		return false
	}
}

// stmtCondString returns a debug-friendly string for a condition expression.
func stmtCondString(e stmt.Expr) string {
	switch e := e.(type) {
	case stmt.BinOp:
		return "(" + stmtCondString(e.Left) + " op" + fmt.Sprintf("%d", e.Op) + " " + stmtCondString(e.Right) + ")"
	case stmt.LitInt:
		return fmt.Sprintf("%d", e.Value)
	case stmt.LitString:
		return fmt.Sprintf("%q", e.Value)
	case stmt.LitBool:
		return fmt.Sprintf("%v", e.Value)
	case stmt.VarRef:
		return e.Name
	case stmt.FieldAccess:
		return stmtCondString(e.Record) + "." + e.Field
	case stmt.BuiltinCall:
		return e.Name + "(...)"
	default:
		return fmt.Sprintf("%T", e)
	}
}

func TestLowerMatchStmt_LitPattern(t *testing.T) {
	cti := makeCTI(nil)

	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "n"),
		Arms: []core.MatchArm{
			{Pattern: &core.LitPattern{Value: int64(0)}, Body: litStr(3, "zero")},
			{Pattern: &core.LitPattern{Value: int64(1)}, Body: litStr(4, "one")},
			{Pattern: &core.WildcardPattern{}, Body: litStr(5, "other")},
		},
	}

	result := LowerMatchStmt(m, cti)
	// Should be an if-chain (lit patterns aren't constructor patterns).
	if _, ok := result.(stmt.IfStmt); !ok {
		t.Errorf("expected IfStmt for lit patterns, got %T", result)
	}
}
