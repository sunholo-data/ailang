package types

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/typedast"
)

// SetLiteralLambdas marks the lambdas that are function literals (#2238): their
// parameter annotations are enforced as equalities, not seeded (inferLambda).
func (tc *CoreTypeChecker) SetLiteralLambdas(ids map[uint64]bool) {
	tc.literalLambdas = ids
}

// declaredEffectRow is the row a lambda's explicit effect annotation declares.
// An empty annotation (a function literal's `! {}`, #2238) is the closed empty
// row, not the "no annotation" sentinel ElaborateEffectRowWithBudgets returns.
func declaredEffectRow(annots []ast.EffectAnnotation) (*Row, error) {
	row, err := ElaborateEffectRowWithBudgets(annots)
	if err == nil && row == nil {
		row = EmptyEffectRow()
	}
	return row, err
}

// literalParamType is the type of an annotated parameter of a function literal
// (#2238). Like a top-level func's, it is the annotation itself (seeded) —
// except when the annotation contains a function type with a concrete effect
// contract (`cb: (int) -> () ! {IO}`). Such a row carries an open
// compatibility tail (elaborate.astTypeToInternalType), and seeding it lets the
// tail leak into the enclosing function when the callback is passed on to an
// effect-polymorphic callee (#2295, which top-level funcs share; measured: 60
// motoko-fork files). For that parameter the annotation is instead an equality
// on a fresh variable, emitted before the body, as a let annotation is enforced.
//
// Why not the equality for every parameter: while the parameter is still a
// fresh variable, an application of it mints a fresh row, and unifying that row
// with a NAMED annotation row (`g: int -> () ! {e}`) rebinds the named variable
// to a fresh remainder (row unification of two bare tails) — rejecting
// effect-polymorphic literals that top-level funcs accept (review of #2289,
// probes d11e). Seeding keeps the callee's own row, as for top-level funcs.
func (tc *CoreTypeChecker) literalParamType(ctx *InferenceContext, lam *core.Lambda, param string, annot Type) Type {
	if !hasConcreteEffectContract(annot, 0) {
		return annot
	}
	v := ctx.freshTypeVar()
	ctx.addConstraint(TypeEq{Left: v, Right: annot,
		Path: []string{fmt.Sprintf("parameter annotation %s at %s", param, lam.Span())}})
	return v
}

// hasConcreteEffectContract reports whether an annotation type contains a
// function type whose effect row is a concrete contract (explicit labels).
// Cycle-safety: annotation types are built from the AST by the elaborator and
// are acyclic; depth is still bounded.
func hasConcreteEffectContract(t Type, depth int) bool {
	if depth > 32 {
		return false
	}
	switch x := t.(type) {
	case *TFunc2:
		if x.ConcreteEffectContract {
			return true
		}
		for _, p := range x.Params {
			if hasConcreteEffectContract(p, depth+1) {
				return true
			}
		}
		return hasConcreteEffectContract(x.Return, depth+1)
	case *TList:
		return hasConcreteEffectContract(x.Element, depth+1)
	case *TArray:
		return hasConcreteEffectContract(x.Element, depth+1)
	case *TTuple:
		for _, e := range x.Elements {
			if hasConcreteEffectContract(e, depth+1) {
				return true
			}
		}
	case *TApp:
		for _, a := range x.Args {
			if hasConcreteEffectContract(a, depth+1) {
				return true
			}
		}
	case *TRecord:
		for _, f := range x.Fields {
			if hasConcreteEffectContract(f, depth+1) {
				return true
			}
		}
	case *TRecordOpen:
		for _, f := range x.Fields {
			if hasConcreteEffectContract(f, depth+1) {
				return true
			}
		}
	}
	return false
}

// inferLit infers type of literal
func (tc *CoreTypeChecker) inferLit(ctx *InferenceContext, lit *core.Lit) (*typedast.TypedLit, *TypeEnv, error) {
	var typ Type
	switch lit.Kind {
	case core.IntLit:
		// For integer literals, create a type variable with Num constraint
		// This allows defaulting to kick in later
		tv := ctx.freshType(Star)
		ctx.addConstraint(ClassConstraint{
			Class:      "Num",
			Type:       tv,
			Path:       []string{fmt.Sprintf("literal at %v", lit.Span())},
			NodeID:     lit.ID(),
			IntLiteral: true,
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
		return nil, ctx.env, fmt.Errorf("undefined variable: %s at %s%s%s%s", v.Name, v.Span(), tc.localResolutionHint(v.Name), importHint(v.Name), builtinHint(v.Name))
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

		var instConstraints []Constraint
		var accesses []*fieldAccess
		monotype, instConstraints, accesses = scheme.instantiate(ctx.freshType, []string{fmt.Sprintf("use of %s at %v", v.Name, v.Span())})
		// #2239: a let-bound function whose field accesses were undecided when
		// it was generalised decides them afresh at every use.
		for _, fa := range accesses {
			ctx.addConstraint(fa)
		}

		// M-TYPE-LIST-ELEMENT-SOUNDNESS: re-emit the scheme's class constraints
		// as fresh wanted constraints on the instantiated type variables. Without
		// this, a let-bound `[42]` generalized to `forall a. Num a => [a]` loses
		// its `Num` obligation at the use site, so unifying `[a'] ~ [string]`
		// silently accepts `needStrs([42])` (the soundness hole). Re-emitting
		// `Num a'` lets the normal ground-constraint resolver reject `Num[string]`.
		for _, c := range instConstraints {
			ctx.addConstraint(ClassConstraint{
				Class:  c.Class,
				Type:   c.Type,
				Path:   []string{fmt.Sprintf("use of %s at %v", v.Name, v.Span())},
				NodeID: v.ID(),
			})
		}

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
			core.OpBitwiseAnd: "&", core.OpBitwiseXor: "^",
			core.OpShiftLeft: "<<", core.OpShiftRight: ">>",
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
			core.OpBitwiseNot: "~",
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
