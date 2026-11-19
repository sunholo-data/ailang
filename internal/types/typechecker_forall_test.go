package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// #2248: a bounded quantifier types with int bounds, an int binder and a bool
// body; InferPredicate also requires the whole clause to be a bool.

type forallBuilder struct{ id uint64 }

func (b *forallBuilder) node() core.CoreNode { b.id++; return core.CoreNode{NodeID: b.id} }
func (b *forallBuilder) intLit(v int64) core.CoreExpr {
	return &core.Lit{CoreNode: b.node(), Kind: core.IntLit, Value: v}
}
func (b *forallBuilder) strLit(v string) core.CoreExpr {
	return &core.Lit{CoreNode: b.node(), Kind: core.StringLit, Value: v}
}
func (b *forallBuilder) v(name string) core.CoreExpr {
	return &core.Var{CoreNode: b.node(), Name: name}
}
func (b *forallBuilder) bin(op string, l, r core.CoreExpr) core.CoreExpr {
	return &core.BinOp{CoreNode: b.node(), Op: op, Left: l, Right: r}
}
func (b *forallBuilder) forall(v string, lo, hi, body core.CoreExpr) core.CoreExpr {
	return &core.Forall{CoreNode: b.node(), Var: v, Lo: lo, Hi: hi, Body: body}
}

func TestInferPredicate_Forall(t *testing.T) {
	cases := []struct {
		name    string
		build   func(b *forallBuilder) core.CoreExpr
		wantErr string // "" = must type-check
	}{
		{"well_typed", func(b *forallBuilder) core.CoreExpr {
			return b.forall("i", b.intLit(0), b.intLit(3), b.bin(">=", b.v("i"), b.intLit(0)))
		}, ""},
		{"nested", func(b *forallBuilder) core.CoreExpr {
			return b.forall("i", b.intLit(0), b.intLit(3),
				b.forall("j", b.v("i"), b.intLit(3), b.bin("<=", b.v("i"), b.v("j"))))
		}, ""},
		{"body_type_error", func(b *forallBuilder) core.CoreExpr {
			return b.forall("i", b.intLit(0), b.intLit(2), b.bin("==", b.bin("+", b.v("i"), b.strLit("x")), b.intLit(1)))
		}, "cannot unify"},
		{"bound_not_int", func(b *forallBuilder) core.CoreExpr {
			return b.forall("i", b.strLit("a"), b.intLit(2), b.bin(">=", b.v("i"), b.intLit(0)))
		}, "forall lower bound"},
		{"upper_bound_not_int", func(b *forallBuilder) core.CoreExpr {
			return b.forall("i", b.intLit(0), b.strLit("b"), b.bin(">=", b.v("i"), b.intLit(0)))
		}, "forall upper bound"},
		{"body_not_bool", func(b *forallBuilder) core.CoreExpr {
			return b.forall("i", b.intLit(0), b.intLit(2), b.bin("+", b.v("i"), b.intLit(1)))
		}, "forall body"},
		{"binder_out_of_scope_after", func(b *forallBuilder) core.CoreExpr {
			return b.bin("&&", b.forall("i", b.intLit(0), b.intLit(2), b.bin(">=", b.v("i"), b.intLit(0))),
				b.bin(">=", b.v("i"), b.intLit(0)))
		}, "undefined variable: i"},
		{"clause_not_bool", func(b *forallBuilder) core.CoreExpr {
			return b.bin("+", b.intLit(1), b.intLit(2))
		}, "bool"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc2 := NewCoreTypeChecker()
			expr := tc.build(&forallBuilder{})
			_, err := tc2.InferPredicate(expr, NewTypeEnvWithBuiltins())
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("well-typed clause rejected: %v", err)
				}
				if _, ok := tc2.CoreTI.Get(expr.ID()); !ok {
					t.Fatalf("no CoreTI entry for the clause")
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// InferPredicate's bool requirement applies to its own root only: a plain
// InferWithConstraints of the same int expression still succeeds.
func TestInferPredicate_RootOnly(t *testing.T) {
	b := &forallBuilder{}
	tc := NewCoreTypeChecker()
	if _, _, _, _, err := tc.InferWithConstraints(b.bin("+", b.intLit(1), b.intLit(2)), NewTypeEnvWithBuiltins()); err != nil {
		t.Fatalf("an int expression outside InferPredicate must still infer: %v", err)
	}
	if tc.predicateRoot != nil {
		t.Fatalf("predicateRoot leaked: %v", tc.predicateRoot)
	}
}

// Anything inferCore has no rule for is still an error, never a silent pass.
func TestInferCoreOther_UnknownFormIsAnError(t *testing.T) {
	tc := NewCoreTypeChecker()
	ctx := NewInferenceContext()
	ctx.env = NewTypeEnvWithBuiltins()
	if _, _, err := tc.inferCoreOther(ctx, &core.DictRef{}); err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("want a not-implemented error, got %v", err)
	}
}

// Outside InferPredicate (a body, a let, a test) forall is TC_FORALL_001.
func TestForallOutsidePredicateIsAnError(t *testing.T) {
	b := &forallBuilder{}
	tc := NewCoreTypeChecker()
	expr := b.forall("i", b.intLit(0), b.intLit(3), b.bin(">=", b.v("i"), b.intLit(0)))
	_, _, _, _, err := tc.InferWithConstraints(expr, NewTypeEnvWithBuiltins())
	if err == nil || !strings.Contains(err.Error(), TC_FORALL_001) {
		t.Fatalf("want %s, got %v", TC_FORALL_001, err)
	}
}
