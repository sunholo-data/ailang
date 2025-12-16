package types

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/typedast"
)

// inferMatch infers type of pattern matching
func (tc *CoreTypeChecker) inferMatch(ctx *InferenceContext, match *core.Match) (*typedast.TypedMatch, *TypeEnv, error) {
	// Infer scrutinee type
	scrutineeNode, _, err := tc.inferCore(ctx, match.Scrutinee)
	if err != nil {
		return nil, ctx.env, err
	}

	// Check exhaustiveness (simplified for now)
	// TODO: Implement full exhaustiveness checking
	exhaustive := match.Exhaustive

	// Infer types of all arms
	var arms []typedast.TypedMatchArm
	var resultType Type
	var allEffects []*Row

	for i, arm := range match.Arms {
		// Type check pattern and get bindings
		patternBindings, typedPattern, err := tc.checkPattern(arm.Pattern, getType(scrutineeNode), ctx)
		if err != nil {
			return nil, ctx.env, err
		}

		// Extend environment with pattern bindings
		armEnv := ctx.env
		for name, typ := range patternBindings {
			armEnv = armEnv.Extend(name, typ)
		}

		// Save and update environment
		oldEnv := ctx.env
		ctx.env = armEnv

		// Check guard if present
		var guardNode typedast.TypedNode
		if arm.Guard != nil {
			guardNode, _, err = tc.inferCore(ctx, arm.Guard)
			if err != nil {
				return nil, oldEnv, err
			}
			// Guard must be boolean
			ctx.addConstraint(TypeEq{
				Left:  getType(guardNode),
				Right: TBool,
				Path:  []string{fmt.Sprintf("match guard %d at %s", i, match.Span())},
			})
			allEffects = append(allEffects, getEffectRow(guardNode))
		}

		// Type check body with expected type (if available)
		bodyCtx := ctx
		if ctx.expectedType != nil {
			// Thread expected type to arm body (tail position)
			bodyCtx = ctx.withExpectedType(*ctx.expectedType)
		}
		bodyNode, _, err := tc.inferCore(bodyCtx, arm.Body)
		if err != nil {
			return nil, oldEnv, err
		}
		allEffects = append(allEffects, getEffectRow(bodyNode))

		// Restore environment
		ctx.env = oldEnv

		// All arms must have same result type
		if i == 0 {
			resultType = getType(bodyNode)
		} else {
			ctx.addConstraint(TypeEq{
				Left:  getType(bodyNode),
				Right: resultType,
				Path:  []string{fmt.Sprintf("match arm %d at %s", i, match.Span())},
			})
		}

		arms = append(arms, typedast.TypedMatchArm{
			Pattern: typedPattern,
			Guard:   guardNode,
			Body:    bodyNode,
		})
	}

	// Add scrutinee effects
	allEffects = append(allEffects, getEffectRow(scrutineeNode))

	return &typedast.TypedMatch{
		TypedExpr: typedast.TypedExpr{
			NodeID:    match.ID(),
			Span:      match.Span(),
			Type:      resultType,
			EffectRow: combineEffectList(allEffects),
			Core:      match,
		},
		Scrutinee:  scrutineeNode,
		Arms:       arms,
		Exhaustive: exhaustive,
	}, ctx.env, nil
}

// checkPattern type checks a pattern and returns bindings
func (tc *CoreTypeChecker) checkPattern(pat core.CorePattern, scrutType Type, ctx *InferenceContext) (map[string]Type, typedast.TypedPattern, error) {
	switch p := pat.(type) {
	case *core.VarPattern:
		// Variable pattern binds to scrutinee type
		return map[string]Type{p.Name: scrutType},
			typedast.TypedVarPattern{Name: p.Name, Type: scrutType}, nil

	case *core.LitPattern:
		// Literal pattern - scrutinee must match literal type
		var litType Type
		switch p.Value.(type) {
		case int, int64:
			litType = TInt
		case float32, float64:
			litType = TFloat
		case string:
			litType = TString
		case bool:
			litType = TBool
		default:
			return nil, nil, fmt.Errorf("unknown literal type in pattern: %T", p.Value)
		}

		ctx.addConstraint(TypeEq{
			Left:  scrutType,
			Right: litType,
			Path:  []string{"literal pattern"},
		})

		return nil, typedast.TypedLitPattern{Value: p.Value}, nil

	case *core.WildcardPattern:
		// Wildcard matches anything, binds nothing
		return nil, typedast.TypedWildcardPattern{}, nil

	case *core.ConstructorPattern:
		// M-DX25.4: Look up constructor's ADT type and add constraint
		// This ensures pattern matching on ADTs infers the correct ADT type for the scrutinee
		if tc.constructorTypes != nil {
			if adtTypeName, ok := tc.constructorTypes[p.Name]; ok {
				// M-TAPP-FIX: Check if ADT has type parameters
				// If so, create TApp with fresh type vars, not just TCon
				var adtType Type
				if paramCount, hasParams := tc.adtTypeParams[adtTypeName]; hasParams && paramCount > 0 {
					// ADT has type parameters - create TApp with fresh type vars
					typeArgs := make([]Type, paramCount)
					for i := 0; i < paramCount; i++ {
						typeArgs[i] = ctx.freshTypeVar()
					}
					adtType = &TApp{
						Constructor: &TCon{Name: adtTypeName},
						Args:        typeArgs,
					}
				} else {
					// No type parameters - use simple TCon
					adtType = &TCon{Name: adtTypeName}
				}

				// Add constraint: scrutinee type = ADT type
				ctx.addConstraint(TypeEq{
					Left:  scrutType,
					Right: adtType,
					Path:  []string{fmt.Sprintf("constructor pattern %s (ADT type %s)", p.Name, adtTypeName)},
				})
			}
		}

		// Recursively check nested patterns
		// We need to know the field types of this constructor
		// For now, create fresh type variables for each field
		bindings := make(map[string]Type)
		typedArgs := make([]typedast.TypedPattern, len(p.Args))

		for i, argPat := range p.Args {
			// Create fresh type variable for each argument
			argType := ctx.freshTypeVar()
			argBindings, typedArg, err := tc.checkPattern(argPat, argType, ctx)
			if err != nil {
				return nil, nil, err
			}
			// Merge bindings
			for name, typ := range argBindings {
				if existing, ok := bindings[name]; ok {
					// Variable bound multiple times - must unify
					ctx.addConstraint(TypeEq{
						Left:  existing,
						Right: typ,
						Path:  []string{fmt.Sprintf("pattern variable %s", name)},
					})
				} else {
					bindings[name] = typ
				}
			}
			typedArgs[i] = typedArg
		}

		return bindings, typedast.TypedConstructorPattern{
			Name: p.Name,
			Args: typedArgs,
		}, nil

	case *core.TuplePattern:
		// Tuple pattern - scrutinee must be tuple type
		// Extract element types from scrutinee
		var elemTypes []Type

		// Try to extract tuple type from scrutinee
		if tupleTy, ok := scrutType.(*TTuple); ok {
			elemTypes = tupleTy.Elements
		} else {
			// Create fresh type variables and add constraint
			elemTypes = make([]Type, len(p.Elements))
			for i := range p.Elements {
				elemTypes[i] = ctx.freshTypeVar()
			}
			ctx.addConstraint(TypeEq{
				Left:  scrutType,
				Right: &TTuple{Elements: elemTypes},
				Path:  []string{"tuple pattern"},
			})
		}

		// Check that arity matches
		if len(p.Elements) != len(elemTypes) {
			return nil, nil, fmt.Errorf("tuple pattern has %d elements but scrutinee has %d",
				len(p.Elements), len(elemTypes))
		}

		// Recursively check each element pattern
		bindings := make(map[string]Type)
		typedElems := make([]typedast.TypedPattern, len(p.Elements))

		for i, elemPat := range p.Elements {
			elemBindings, typedElem, err := tc.checkPattern(elemPat, elemTypes[i], ctx)
			if err != nil {
				return nil, nil, err
			}
			// Merge bindings
			for name, typ := range elemBindings {
				if existing, ok := bindings[name]; ok {
					// Variable bound multiple times - must unify
					ctx.addConstraint(TypeEq{
						Left:  existing,
						Right: typ,
						Path:  []string{fmt.Sprintf("pattern variable %s", name)},
					})
				} else {
					bindings[name] = typ
				}
			}
			typedElems[i] = typedElem
		}

		return bindings, typedast.TypedTuplePattern{
			Elements: typedElems,
		}, nil

	case *core.ListPattern:
		// List pattern - scrutinee must be list type
		// Extract element type from scrutinee list
		var elemType Type

		// DX-17: Try to extract list type from scrutinee (handles both TList and TApp("list",...))
		if elem, ok := AsList(scrutType); ok {
			elemType = elem
		} else {
			// Create fresh type variable for elements
			elemType = ctx.freshTypeVar()
			// Constraint uses TList - the unifier handles TList ↔ TApp("list",...) equivalence
			ctx.addConstraint(TypeEq{
				Left:  scrutType,
				Right: &TList{Element: elemType},
				Path:  []string{"list pattern"},
			})
		}

		// Recursively check each element pattern
		bindings := make(map[string]Type)
		typedElems := make([]typedast.TypedPattern, len(p.Elements))

		for i, elemPat := range p.Elements {
			elemBindings, typedElem, err := tc.checkPattern(elemPat, elemType, ctx)
			if err != nil {
				return nil, nil, err
			}
			// Merge bindings
			for name, typ := range elemBindings {
				if existing, ok := bindings[name]; ok {
					// Variable bound multiple times - must unify
					ctx.addConstraint(TypeEq{
						Left:  existing,
						Right: typ,
						Path:  []string{fmt.Sprintf("pattern variable %s", name)},
					})
				} else {
					bindings[name] = typ
				}
			}
			typedElems[i] = typedElem
		}

		// Type check tail pattern if present
		var typedTail *typedast.TypedPattern
		if p.Tail != nil {
			// Tail must have list type (same as scrutinee)
			tailBindings, tail, err := tc.checkPattern(*p.Tail, scrutType, ctx)
			if err != nil {
				return nil, nil, err
			}
			// Merge tail bindings
			for name, typ := range tailBindings {
				if existing, ok := bindings[name]; ok {
					// Variable bound multiple times - must unify
					ctx.addConstraint(TypeEq{
						Left:  existing,
						Right: typ,
						Path:  []string{fmt.Sprintf("pattern variable %s", name)},
					})
				} else {
					bindings[name] = typ
				}
			}
			typedTail = &tail
		}

		return bindings, typedast.TypedListPattern{
			Elements: typedElems,
			Tail:     typedTail,
		}, nil

	default:
		return nil, nil, fmt.Errorf("pattern type checking not implemented for %T", pat)
	}
}
