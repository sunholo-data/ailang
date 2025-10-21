package types

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/typedast"
)

// inferLit infers type of literal
func (tc *CoreTypeChecker) inferLit(ctx *InferenceContext, lit *core.Lit) (*typedast.TypedLit, *TypeEnv, error) {
	var typ Type
	switch lit.Kind {
	case core.IntLit:
		// For integer literals, create a type variable with Num constraint
		// This allows defaulting to kick in later
		tv := ctx.freshType(Star)
		ctx.addConstraint(ClassConstraint{
			Class:  "Num",
			Type:   tv,
			Path:   []string{fmt.Sprintf("literal at %v", lit.Span())},
			NodeID: lit.ID(),
		})
		typ = tv
	case core.FloatLit:
		// For float literals, create a type variable with Fractional constraint
		tv := ctx.freshType(Star)
		ctx.addConstraint(ClassConstraint{
			Class:  "Fractional",
			Type:   tv,
			Path:   []string{fmt.Sprintf("literal at %v", lit.Span())},
			NodeID: lit.ID(),
		})
		typ = tv
	case core.StringLit:
		typ = TString
	case core.BoolLit:
		typ = TBool
	case core.UnitLit:
		typ = TUnit
	default:
		return nil, ctx.env, fmt.Errorf("unknown literal kind: %v", lit.Kind)
	}

	return &typedast.TypedLit{
		TypedExpr: typedast.TypedExpr{
			NodeID:    lit.ID(),
			Span:      lit.Span(),
			Type:      typ,
			EffectRow: EmptyEffectRow(),
			Core:      lit,
		},
		Kind:  lit.Kind,
		Value: lit.Value,
	}, ctx.env, nil
}

// inferVar infers type of variable
func (tc *CoreTypeChecker) inferVar(ctx *InferenceContext, v *core.Var) (*typedast.TypedVar, *TypeEnv, error) {
	typ, err := ctx.env.Lookup(v.Name)
	if err != nil {
		return nil, ctx.env, fmt.Errorf("undefined variable: %s at %s", v.Name, v.Span())
	}

	// Instantiate if it's a scheme
	var monotype Type
	if scheme, ok := typ.(*Scheme); ok {
		// Track fresh variables before instantiation
		var freshVars []string
		if tc.trackInstantiations {
			// Capture fresh type variables that will be generated
			for range scheme.TypeVars {
				freshVars = append(freshVars, fmt.Sprintf("t%d", tc.varCounter))
				tc.varCounter++
			}
		}

		monotype = scheme.Instantiate(ctx.freshType)

		// Record instantiation after it happens
		if tc.trackInstantiations {
			tc.instantiations = append(tc.instantiations, Instantiation{
				Location:     v.Span().String(),
				VarName:      v.Name,
				FreshVars:    freshVars,
				Instantiated: monotype,
			})
		}
	} else if t, ok := typ.(Type); ok {
		monotype = t
	} else {
		return nil, ctx.env, fmt.Errorf("invalid type in environment: %T", typ)
	}

	return &typedast.TypedVar{
		TypedExpr: typedast.TypedExpr{
			NodeID:    v.ID(),
			Span:      v.Span(),
			Type:      monotype,
			EffectRow: EmptyEffectRow(),
			Core:      v,
		},
		Name: v.Name,
	}, ctx.env, nil
}

// inferVarGlobal infers type of global variable reference
func (tc *CoreTypeChecker) inferVarGlobal(ctx *InferenceContext, v *core.VarGlobal) (*typedast.TypedVar, *TypeEnv, error) {
	// Look up the type in the global types
	key := fmt.Sprintf("%s.%s", v.Ref.Module, v.Ref.Name)

	scheme, ok := tc.globalTypes[key]
	if !ok {
		return nil, ctx.env, fmt.Errorf("undefined global variable: %s from %s", v.Ref.Name, v.Ref.Module)
	}

	// Track fresh variables before instantiation
	var freshVars []string
	if tc.trackInstantiations {
		// Capture fresh type variables that will be generated
		for range scheme.TypeVars {
			freshVars = append(freshVars, fmt.Sprintf("t%d", tc.varCounter))
			tc.varCounter++
		}
	}

	// Instantiate the scheme
	monotype := scheme.Instantiate(ctx.freshType)

	// Record instantiation after it happens
	if tc.trackInstantiations {
		tc.instantiations = append(tc.instantiations, Instantiation{
			Location:     v.Span().String(),
			VarName:      fmt.Sprintf("%s.%s", v.Ref.Module, v.Ref.Name),
			FreshVars:    freshVars,
			Instantiated: monotype,
		})
	}

	return &typedast.TypedVar{
		TypedExpr: typedast.TypedExpr{
			NodeID:    v.ID(),
			Span:      v.Span(),
			Type:      monotype,
			EffectRow: EmptyEffectRow(), // Variable reference itself has no effects
			Core:      v,
		},
		Name: fmt.Sprintf("%s.%s", v.Ref.Module, v.Ref.Name),
	}, ctx.env, nil
}

// inferIntrinsic infers type of intrinsic operation
func (tc *CoreTypeChecker) inferIntrinsic(ctx *InferenceContext, intrinsic *core.Intrinsic) (*typedast.TypedBinOp, *TypeEnv, error) {
	// For binary intrinsics, delegate to inferBinOp logic
	if len(intrinsic.Args) == 2 {
		// Convert back to BinOp for type checking (temporary)
		opStr := map[core.IntrinsicOp]string{
			core.OpAdd: "+", core.OpSub: "-", core.OpMul: "*", core.OpDiv: "/", core.OpMod: "%",
			core.OpEq: "==", core.OpNe: "!=", core.OpLt: "<", core.OpLe: "<=", core.OpGt: ">", core.OpGe: ">=",
			core.OpConcat: "++", core.OpAnd: "&&", core.OpOr: "||",
		}[intrinsic.Op]

		binop := &core.BinOp{
			CoreNode: intrinsic.CoreNode,
			Op:       opStr,
			Left:     intrinsic.Args[0],
			Right:    intrinsic.Args[1],
		}
		return tc.inferBinOp(ctx, binop)
	}

	// For unary intrinsics
	if len(intrinsic.Args) == 1 {
		opStr := map[core.IntrinsicOp]string{
			core.OpNot: "not", core.OpNeg: "-",
		}[intrinsic.Op]

		unop := &core.UnOp{
			CoreNode: intrinsic.CoreNode,
			Op:       opStr,
			Operand:  intrinsic.Args[0],
		}
		// We need to adapt the unary result
		unResult, env, err := tc.inferUnOp(ctx, unop)
		if err != nil {
			return nil, env, err
		}
		// Convert UnOp result to BinOp result (hack for now)
		return &typedast.TypedBinOp{
			TypedExpr: unResult.TypedExpr,
			Op:        opStr,
			Left:      unResult.Operand,
			Right:     &typedast.TypedLit{TypedExpr: typedast.TypedExpr{Type: TUnit}}, // dummy
		}, env, nil
	}

	return nil, ctx.env, fmt.Errorf("unexpected intrinsic arity: %d", len(intrinsic.Args))
}
