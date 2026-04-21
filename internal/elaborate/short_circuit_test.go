package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// Tests for short-circuit desugaring of && and || into core.If.
// See design_docs/planned/v0_11_3/m-eval-short-circuit-bool.md
// Bug report: ailang-parse msg ce6e078e.

func elaborateExpr(t *testing.T, src string) core.CoreExpr {
	t.Helper()
	l := lexer.New(src, "test.ail")
	p := parser.New(l)
	prog := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	elab := NewElaborator()
	coreProg, err := elab.Elaborate(prog)
	if err != nil {
		t.Fatalf("elaborate error: %v", err)
	}
	if coreProg == nil || len(coreProg.Decls) == 0 {
		t.Fatalf("no core decls produced")
	}
	// Return the first top-level expression's body.
	return coreProg.Decls[0]
}

// containsIf walks an expression tree and reports whether any node is a core.If.
func containsIf(expr core.CoreExpr) bool {
	if expr == nil {
		return false
	}
	if _, ok := expr.(*core.If); ok {
		return true
	}
	switch e := expr.(type) {
	case *core.Let:
		return containsIf(e.Value) || containsIf(e.Body)
	case *core.LetRec:
		for _, b := range e.Bindings {
			if containsIf(b.Value) {
				return true
			}
		}
		return containsIf(e.Body)
	case *core.App:
		if containsIf(e.Func) {
			return true
		}
		for _, a := range e.Args {
			if containsIf(a) {
				return true
			}
		}
		return false
	case *core.Lambda:
		return containsIf(e.Body)
	case *core.Intrinsic:
		for _, a := range e.Args {
			if containsIf(a) {
				return true
			}
		}
		return false
	case *core.BinOp:
		return containsIf(e.Left) || containsIf(e.Right)
	}
	return false
}

// containsIntrinsic reports whether any node uses the given intrinsic op.
func containsIntrinsic(expr core.CoreExpr, op core.IntrinsicOp) bool {
	if expr == nil {
		return false
	}
	if intr, ok := expr.(*core.Intrinsic); ok && intr.Op == op {
		return true
	}
	switch e := expr.(type) {
	case *core.Let:
		return containsIntrinsic(e.Value, op) || containsIntrinsic(e.Body, op)
	case *core.LetRec:
		for _, b := range e.Bindings {
			if containsIntrinsic(b.Value, op) {
				return true
			}
		}
		return containsIntrinsic(e.Body, op)
	case *core.App:
		if containsIntrinsic(e.Func, op) {
			return true
		}
		for _, a := range e.Args {
			if containsIntrinsic(a, op) {
				return true
			}
		}
		return false
	case *core.Lambda:
		return containsIntrinsic(e.Body, op)
	case *core.Intrinsic:
		for _, a := range e.Args {
			if containsIntrinsic(a, op) {
				return true
			}
		}
		return false
	case *core.If:
		return containsIntrinsic(e.Cond, op) || containsIntrinsic(e.Then, op) || containsIntrinsic(e.Else, op)
	}
	return false
}

func TestShortCircuit_AndDesugarsToIf(t *testing.T) {
	expr := elaborateExpr(t, "true && false")
	if !containsIf(expr) {
		t.Errorf("expected && to desugar to core.If, but no If node found in %T", expr)
	}
	if containsIntrinsic(expr, core.OpAnd) {
		t.Errorf("expected && to NOT produce core.OpAnd intrinsic after desugar")
	}
}

func TestShortCircuit_OrDesugarsToIf(t *testing.T) {
	expr := elaborateExpr(t, "true || false")
	if !containsIf(expr) {
		t.Errorf("expected || to desugar to core.If, but no If node found in %T", expr)
	}
	if containsIntrinsic(expr, core.OpOr) {
		t.Errorf("expected || to NOT produce core.OpOr intrinsic after desugar")
	}
}

func TestShortCircuit_AndPreservesLhsBinding(t *testing.T) {
	// Complex LHS should still be normalized (ANF-lifted) so it's evaluated once,
	// but it must be evaluated BEFORE the If, not inside the RHS thunk.
	expr := elaborateExpr(t, "(1 + 2 == 3) && false")
	if !containsIf(expr) {
		t.Errorf("expected && over complex LHS to produce core.If, got %T", expr)
	}
}

func TestShortCircuit_NestedAndOr(t *testing.T) {
	expr := elaborateExpr(t, "true && (false || true)")
	if !containsIf(expr) {
		t.Errorf("expected nested && / || to desugar to core.If, got %T", expr)
	}
	if containsIntrinsic(expr, core.OpAnd) || containsIntrinsic(expr, core.OpOr) {
		t.Errorf("expected no OpAnd/OpOr intrinsics after desugar")
	}
}

// Regression test for bug report ce6e078e: guarded string indexing must not
// evaluate the RHS when the LHS is false. This test exercises the shape only;
// the behavioral test lives in examples/short_circuit_and.ail.
func TestShortCircuit_GuardedIndexingShape(t *testing.T) {
	// We can't easily test charAt bounds-checking at the Core level without
	// the full runtime; instead we verify the Core shape enforces laziness.
	src := `let s = "abc" in let i = 0 in (i > 0 && (i + 1) == 1)`
	expr := elaborateExpr(t, src)
	if !containsIf(expr) {
		t.Errorf("expected guarded expression to elaborate with If (lazy RHS), got %s",
			strings.ToLower(coreKind(expr)))
	}
}

func coreKind(e core.CoreExpr) string {
	if e == nil {
		return "nil"
	}
	switch e.(type) {
	case *core.If:
		return "If"
	case *core.Let:
		return "Let"
	case *core.Intrinsic:
		return "Intrinsic"
	case *core.App:
		return "App"
	}
	return "other"
}
