package types

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/typedast"
)

// inferLambda infers type of lambda with linear capture analysis
func (tc *CoreTypeChecker) inferLambda(ctx *InferenceContext, lam *core.Lambda) (*typedast.TypedLambda, *TypeEnv, error) {
	// Fresh type variables for parameters
	paramTypes := make([]Type, len(lam.Params))
	newEnv := ctx.env

	for i, param := range lam.Params {
		paramType := ctx.freshTypeVar()
		paramTypes[i] = paramType
		newEnv = newEnv.Extend(param, paramType)
	}

	// Save old env and use new one for body
	oldEnv := ctx.env
	ctx.env = newEnv

	// Infer body type
	bodyNode, _, err := tc.inferCore(ctx, lam.Body)
	if err != nil {
		return nil, oldEnv, err
	}

	// Check for linear capture violations
	captured := tc.findCapturedVars(lam, oldEnv)
	for _, cap := range captured {
		if tc.isLinearCapability(cap) {
			return nil, oldEnv, fmt.Errorf("lambda at %s captures linear capability %s; pass it as a parameter instead",
				lam.Span(), cap)
		}
	}

	// Restore environment
	ctx.env = oldEnv

	// Lambda type with effect annotation
	var funcEffectRow *Row

	// Check for explicit effect annotation from AST
	if effectNames := tc.effectAnnots[lam.ID()]; len(effectNames) > 0 {
		// Use explicit annotation
		var err error
		funcEffectRow, err = ElaborateEffectRow(effectNames)
		if err != nil {
			return nil, oldEnv, fmt.Errorf("invalid effect annotation at %s: %w", lam.Span(), err)
		}
	} else {
		// Infer from body (existing behavior)
		if effRow := bodyNode.GetEffectRow(); effRow != nil {
			funcEffectRow = effRow.(*Row)
		}
	}

	funcType := &TFunc2{
		Params:    paramTypes,
		EffectRow: funcEffectRow,
		Return:    bodyNode.GetType().(Type),
	}

	return &typedast.TypedLambda{
		TypedExpr: typedast.TypedExpr{
			NodeID:    lam.ID(),
			Span:      lam.Span(),
			Type:      funcType,
			EffectRow: interface{}(EmptyEffectRow()), // Lambda itself is pure
			Core:      lam,
		},
		Params:     lam.Params,
		ParamTypes: toInterfaceSlice(paramTypes),
		Body:       bodyNode,
	}, ctx.env, nil
}

// inferLet infers type of let binding with generalization
func (tc *CoreTypeChecker) inferLet(ctx *InferenceContext, let *core.Let) (*typedast.TypedLet, *TypeEnv, error) {
	// Infer value type
	valueNode, _, err := tc.inferCore(ctx, let.Value)
	if err != nil {
		return nil, ctx.env, err
	}

	// CRITICAL: Apply defaulting BEFORE generalization
	// This is a generalization boundary where defaulting must happen
	valueType := getType(valueNode)
	valueEffects := getEffectRow(valueNode)

	// Get unsolved constraints from current context
	_, unsolvedConstraints, err := ctx.SolveConstraints()
	if err != nil {
		return nil, ctx.env, err
	}

	// Apply defaulting at this generalization boundary
	defaultingSub, defaultedType, defaultedConstraints, err := tc.defaultAmbiguities(valueType, unsolvedConstraints)
	if err != nil {
		return nil, ctx.env, fmt.Errorf("defaulting failed for let binding %s: %w", let.Name, err)
	}

	// Apply defaulting substitution everywhere if any defaults were applied
	if len(defaultingSub) > 0 {
		defaultedType, defaultedConstraints, valueNode, _ = tc.ApplySubstEverywhere(
			defaultingSub, defaultedType, defaultedConstraints, valueNode, nil, let.Name)
	}

	// Generalize if value is syntactic value (value restriction)
	var binding interface{}
	if isCoreValue(let.Value) {
		// After defaulting, only non-ground constraints should remain for generalization
		nonGroundConstraints := []ClassConstraint{}
		for _, c := range defaultedConstraints {
			if !isGround(c.Type) {
				nonGroundConstraints = append(nonGroundConstraints, c)
			}
		}
		binding = tc.generalizeWithConstraints(defaultedType, valueEffects, nonGroundConstraints)
	} else {
		binding = defaultedType
	}

	// Extend environment
	var newEnv *TypeEnv
	var scheme *Scheme
	if s, ok := binding.(*Scheme); ok {
		scheme = s
		newEnv = ctx.env.ExtendScheme(let.Name, s)
	} else {
		newEnv = ctx.env.Extend(let.Name, binding.(Type))
		// Create trivial scheme for consistency
		scheme = &Scheme{Type: binding.(Type)}
	}

	// Save env and infer body
	oldEnv := ctx.env
	ctx.env = newEnv
	bodyNode, finalEnv, err := tc.inferCore(ctx, let.Body)
	if err != nil {
		return nil, oldEnv, err
	}
	ctx.env = oldEnv

	return &typedast.TypedLet{
		TypedExpr: typedast.TypedExpr{
			NodeID:    let.ID(),
			Span:      let.Span(),
			Type:      bodyNode.GetType(),
			EffectRow: combineEffects(valueEffects, getEffectRow(bodyNode)),
			Core:      let,
		},
		Name:   let.Name,
		Scheme: scheme, // Generalized type at binding site
		Value:  valueNode,
		Body:   bodyNode,
	}, finalEnv, nil
}

// inferLetRec infers type of recursive bindings
func (tc *CoreTypeChecker) inferLetRec(ctx *InferenceContext, letrec *core.LetRec) (*typedast.TypedLetRec, *TypeEnv, error) {
	// Create fresh type variables for all bindings
	bindingTypes := make(map[string]Type)
	for _, binding := range letrec.Bindings {
		bindingTypes[binding.Name] = ctx.freshTypeVar()
	}

	// Extend environment with all bindings
	newEnv := ctx.env
	for name, typ := range bindingTypes {
		newEnv = newEnv.Extend(name, typ)
	}

	// Save and update environment
	oldEnv := ctx.env
	ctx.env = newEnv

	// Infer types of all values and collect constraints
	var allValueNodes []typedast.TypedNode
	var allValueTypes []Type
	for _, binding := range letrec.Bindings {
		valueNode, _, err := tc.inferCore(ctx, binding.Value)
		if err != nil {
			return nil, oldEnv, err
		}

		allValueNodes = append(allValueNodes, valueNode)
		allValueTypes = append(allValueTypes, getType(valueNode))

		// Unify with expected type
		ctx.addConstraint(TypeEq{
			Left:  bindingTypes[binding.Name],
			Right: getType(valueNode),
			Path:  []string{binding.Name},
		})
	}

	// CRITICAL: Apply defaulting ONCE for the entire SCC after solving mutual block
	_, unsolvedConstraints, err := ctx.SolveConstraints()
	if err != nil {
		return nil, oldEnv, err
	}

	// Apply defaulting to the entire mutual block (once per SCC)
	for i, binding := range letrec.Bindings {
		valueType := allValueTypes[i]
		valueNode := allValueNodes[i]

		// Apply defaulting at this generalization boundary
		defaultingSub, defaultedType, defaultedConstraints, err := tc.defaultAmbiguities(valueType, unsolvedConstraints)
		if err != nil {
			return nil, oldEnv, fmt.Errorf("defaulting failed for letrec binding %s: %w", binding.Name, err)
		}

		// Apply defaulting substitution everywhere if any defaults were applied
		if len(defaultingSub) > 0 {
			defaultedType, _, valueNode, _ = tc.ApplySubstEverywhere(
				defaultingSub, defaultedType, defaultedConstraints, valueNode, nil, binding.Name)

			// Update the stored values
			allValueTypes[i] = defaultedType
			allValueNodes[i] = valueNode
		}
	}

	// Now generalize each binding after defaulting
	typedBindings := make([]typedast.TypedRecBinding, len(letrec.Bindings))
	for i, binding := range letrec.Bindings {
		valueType := allValueTypes[i]
		valueNode := allValueNodes[i]

		// Get remaining non-ground constraints after defaulting
		_, remainingConstraints, err := ctx.SolveConstraints()
		if err != nil {
			return nil, oldEnv, err
		}

		nonGroundConstraints := []ClassConstraint{}
		for _, c := range remainingConstraints {
			if !isGround(c.Type) {
				nonGroundConstraints = append(nonGroundConstraints, c)
			}
		}

		// Generalize for recursion
		scheme := tc.generalizeWithConstraints(valueType, getEffectRow(valueNode), nonGroundConstraints)

		typedBindings[i] = typedast.TypedRecBinding{
			Name:   binding.Name,
			Scheme: scheme,
			Value:  valueNode,
		}

		// Update environment with generalized type
		newEnv = newEnv.ExtendScheme(binding.Name, scheme)
	}

	// Update context environment for body
	ctx.env = newEnv

	// Infer body type
	bodyNode, finalEnv, err := tc.inferCore(ctx, letrec.Body)
	if err != nil {
		return nil, oldEnv, err
	}

	// Restore environment
	ctx.env = oldEnv

	// Combine effects from all bindings and body
	var allEffects []*Row
	for _, binding := range typedBindings {
		allEffects = append(allEffects, getEffectRow(binding.Value))
	}
	allEffects = append(allEffects, getEffectRow(bodyNode))

	return &typedast.TypedLetRec{
		TypedExpr: typedast.TypedExpr{
			NodeID:    letrec.ID(),
			Span:      letrec.Span(),
			Type:      bodyNode.GetType(),
			EffectRow: combineEffectList(allEffects),
			Core:      letrec,
		},
		Bindings: typedBindings,
		Body:     bodyNode,
	}, finalEnv, nil
}

// generalizeWithConstraints creates a type scheme with explicit constraints
func (tc *CoreTypeChecker) generalizeWithConstraints(typ Type, effects *Row, constraints []ClassConstraint) *Scheme {
	// Find free type variables in type but not in environment
	typeFreeVars := make(map[string]bool)
	collectFreeVars(typ, typeFreeVars)

	// For now, simplified generalization
	// In a full implementation, would check against environment free vars
	generalizedTypeVars := []string{}
	for v := range typeFreeVars {
		generalizedTypeVars = append(generalizedTypeVars, v)
	}

	// Convert class constraints to scheme constraints
	schemeConstraints := []Constraint{}
	for _, c := range constraints {
		schemeConstraints = append(schemeConstraints, Constraint{
			Class: c.Class,
			Type:  c.Type,
		})
	}

	return &Scheme{
		TypeVars:    generalizedTypeVars,
		RowVars:     []string{}, // Simplified for now
		Constraints: schemeConstraints,
		Type:        typ,
	}
}

// inferApp infers type of function application
func (tc *CoreTypeChecker) inferApp(ctx *InferenceContext, app *core.App) (*typedast.TypedApp, *TypeEnv, error) {
	// Infer function type
	funcNode, _, err := tc.inferCore(ctx, app.Func)
	if err != nil {
		return nil, ctx.env, err
	}

	// Infer argument types
	var argNodes []typedast.TypedNode
	var argTypes []Type
	var allEffects []*Row

	for _, arg := range app.Args {
		argNode, _, err := tc.inferCore(ctx, arg)
		if err != nil {
			return nil, ctx.env, err
		}
		argNodes = append(argNodes, argNode)
		argTypes = append(argTypes, getType(argNode))
		allEffects = append(allEffects, getEffectRow(argNode))
	}

	// Create result type variable
	resultType := ctx.freshTypeVar()
	effectRow := ctx.freshEffectRow()

	// Unify function type with expected type
	expectedFuncType := &TFunc2{
		Params:    argTypes,
		EffectRow: effectRow,
		Return:    resultType,
	}

	ctx.addConstraint(TypeEq{
		Left:  getType(funcNode),
		Right: expectedFuncType,
		Path:  []string{"function application at " + app.Span().String()},
	})

	// Combine effects
	allEffects = append(allEffects, getEffectRow(funcNode), effectRow)

	return &typedast.TypedApp{
		TypedExpr: typedast.TypedExpr{
			NodeID:    app.ID(),
			Span:      app.Span(),
			Type:      resultType,
			EffectRow: combineEffectList(allEffects),
			Core:      app,
		},
		Func: funcNode,
		Args: argNodes,
	}, ctx.env, nil
}

// inferIf infers type of conditional
func (tc *CoreTypeChecker) inferIf(ctx *InferenceContext, ifExpr *core.If) (*typedast.TypedIf, *TypeEnv, error) {
	// Infer condition type
	condNode, _, err := tc.inferCore(ctx, ifExpr.Cond)
	if err != nil {
		return nil, ctx.env, err
	}

	// Condition must be bool
	ctx.addConstraint(TypeEq{
		Left:  getType(condNode),
		Right: TBool,
		Path:  []string{"if condition at " + ifExpr.Span().String()},
	})

	// Infer branch types
	thenNode, _, err := tc.inferCore(ctx, ifExpr.Then)
	if err != nil {
		return nil, ctx.env, err
	}

	elseNode, _, err := tc.inferCore(ctx, ifExpr.Else)
	if err != nil {
		return nil, ctx.env, err
	}

	// Branches must have same type
	ctx.addConstraint(TypeEq{
		Left:  getType(thenNode),
		Right: getType(elseNode),
		Path:  []string{"if branches at " + ifExpr.Span().String()},
	})

	// Combine effects from all parts
	effects := combineEffectList([]*Row{
		getEffectRow(condNode),
		getEffectRow(thenNode),
		getEffectRow(elseNode),
	})

	return &typedast.TypedIf{
		TypedExpr: typedast.TypedExpr{
			NodeID:    ifExpr.ID(),
			Span:      ifExpr.Span(),
			Type:      getType(thenNode),
			EffectRow: effects,
			Core:      ifExpr,
		},
		Cond: condNode,
		Then: thenNode,
		Else: elseNode,
	}, ctx.env, nil
}

// Helper functions

// findCapturedVars finds variables captured by a lambda
func (tc *CoreTypeChecker) findCapturedVars(lam *core.Lambda, outerEnv *TypeEnv) []string {
	// This is simplified - full implementation would traverse body
	// and check which variables are from outer scope
	var captured []string
	// TODO: Implement proper free variable analysis
	return captured
}

// isLinearCapability checks if a variable is a linear capability
func (tc *CoreTypeChecker) isLinearCapability(name string) bool {
	// Check if name is a known capability like FS, Net, etc
	capabilities := []string{"FS", "Net", "IO", "Async"}
	for _, cap := range capabilities {
		if name == cap {
			return true
		}
	}
	return false
}
