package types

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/typedast"
)

// TC_FORALL_001: a bounded quantifier outside a contract clause.
const TC_FORALL_001 = "TC_FORALL_001"

// inferCoreOther handles the Core forms inferCore's main switch does not:
// today only the bounded quantifier. Anything else is an inference error,
// never a silent pass (#2248: a quantifier used to fall through to here and
// its contract's whole inference was dropped).
//
// `forall` is a contract-only form: only the SMT encoder knows it (no
// evaluator, VM, lowering or CoreTI-validator case), so it types only under an
// InferPredicate root. Anywhere else — a function body, a let, a test or
// property body — it is TC_FORALL_001, as located as any type error.
func (tc *CoreTypeChecker) inferCoreOther(ctx *InferenceContext, expr core.CoreExpr) (typedast.TypedNode, *TypeEnv, error) {
	if fa, ok := expr.(*core.Forall); ok {
		if tc.predicateRoot == nil {
			return nil, ctx.env, forallOutsideContract(fa)
		}
		return tc.inferForall(ctx, fa)
	}
	return nil, ctx.env, fmt.Errorf("type inference not implemented for %T", expr)
}

func forallOutsideContract(fa *core.Forall) error {
	pos := fa.Span()
	return ailerrors.WrapReport(&ailerrors.Report{
		Schema:  "ailang.error/v1",
		Code:    TC_FORALL_001,
		Phase:   "typecheck",
		Message: fmt.Sprintf("`forall %s: lo..hi => p` at %s is a contract quantifier: it is allowed only in a requires/ensures clause", fa.Var, pos),
		Span:    &ast.Span{Start: pos, End: pos},
		Fix:     &ailerrors.Fix{Suggestion: "move the quantified condition into a requires/ensures clause, or compute it with recursion or a std/list function such as any"},
	})
}

// InferPredicate infers a contract clause (#2248): InferWithConstraints, plus
// the requirement that the expression is a bool. Every error is the caller's
// to report; a contract whose inference fails is not checked at all.
func (tc *CoreTypeChecker) InferPredicate(expr core.CoreExpr, env *TypeEnv) (Type, error) {
	tc.predicateRoot = expr
	defer func() { tc.predicateRoot = nil }()
	_, _, typ, _, err := tc.InferWithConstraints(expr, env)
	return typ, err
}

// constrainPredicateRoot adds `root : bool` when root is InferPredicate's.
func (tc *CoreTypeChecker) constrainPredicateRoot(ctx *InferenceContext, root core.CoreExpr, node typedast.TypedNode) {
	if root == nil || root != tc.predicateRoot {
		return
	}
	ctx.addConstraint(TypeEq{Left: getType(node), Right: TBool, Path: []string{"contract at " + root.Span().String()}})
}

// inferForall types `forall v: lo..hi => body` (#2248): both bounds are int,
// v is an int in the body, the body is bool, and so is the quantifier.
// The SMT encoding (smt/codegen_quantifier.go) assumes exactly these types.
func (tc *CoreTypeChecker) inferForall(ctx *InferenceContext, fa *core.Forall) (*typedast.TypedForall, *TypeEnv, error) {
	where := fa.Span().String()
	loNode, _, err := tc.inferCore(ctx, fa.Lo)
	if err != nil {
		return nil, ctx.env, err
	}
	ctx.addConstraint(TypeEq{Left: getType(loNode), Right: TInt, Path: []string{"forall lower bound at " + where}})

	hiNode, _, err := tc.inferCore(ctx, fa.Hi)
	if err != nil {
		return nil, ctx.env, err
	}
	ctx.addConstraint(TypeEq{Left: getType(hiNode), Right: TInt, Path: []string{"forall upper bound at " + where}})

	oldEnv := ctx.env
	ctx.env = ctx.env.Extend(fa.Var, TInt)
	bodyNode, _, err := tc.inferCore(ctx, fa.Body)
	ctx.env = oldEnv
	if err != nil {
		return nil, ctx.env, err
	}
	ctx.addConstraint(TypeEq{Left: getType(bodyNode), Right: TBool, Path: []string{"forall body at " + where}})

	return &typedast.TypedForall{
		TypedExpr: typedast.TypedExpr{
			NodeID: fa.ID(),
			Span:   fa.Span(),
			Type:   TBool,
			EffectRow: ctx.unionEffectList([]*Row{
				getEffectRow(loNode), getEffectRow(hiNode), getEffectRow(bodyNode),
			}),
			Core: fa,
		},
		Var:  fa.Var,
		Lo:   loNode,
		Hi:   hiNode,
		Body: bodyNode,
	}, ctx.env, nil
}
