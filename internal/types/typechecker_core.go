package types

import (
	"fmt"
	"strings"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/typedast"
)

// CoreTypeChecker type checks Core AST and produces TypedAST
type CoreTypeChecker struct {
	errors []error
}

// NewCoreTypeChecker creates a new Core type checker
func NewCoreTypeChecker() *CoreTypeChecker {
	return &CoreTypeChecker{
		errors: []error{},
	}
}

// CheckCoreProgram type checks a Core program and produces TypedAST
func (tc *CoreTypeChecker) CheckCoreProgram(prog *core.Program) (*typedast.TypedProgram, error) {
	typed := &typedast.TypedProgram{
		Decls: make([]typedast.TypedNode, 0),
	}
	
	// Create global environment with builtins
	globalEnv := NewTypeEnvWithBuiltins()
	
	for _, decl := range prog.Decls {
		typedNode, env, err := tc.checkCoreExpr(decl, globalEnv)
		if err != nil {
			tc.errors = append(tc.errors, err)
			continue
		}
		typed.Decls = append(typed.Decls, typedNode)
		globalEnv = env // Update environment with new bindings
	}
	
	// Report all errors
	if len(tc.errors) > 0 {
		return nil, tc.formatErrors()
	}
	
	return typed, nil
}

// checkCoreExpr type checks a Core expression
func (tc *CoreTypeChecker) checkCoreExpr(expr core.CoreExpr, env *TypeEnv) (typedast.TypedNode, *TypeEnv, error) {
	ctx := NewInferenceContext()
	ctx.env = env
	
	// Infer type and effects
	typedNode, newEnv, err := tc.inferCore(ctx, expr)
	if err != nil {
		return nil, env, err
	}
	
	// Solve constraints
	sub, unsolved, err := ctx.SolveConstraints()
	if err != nil {
		return nil, env, err
	}
	
	// Fail on ANY unsolved constraints
	if len(unsolved) > 0 {
		for _, c := range unsolved {
			tc.errors = append(tc.errors, 
				NewUnsolvedConstraintError(c.Class, c.Type, c.Path))
		}
		return nil, env, fmt.Errorf("unsolved constraints")
	}
	
	// Apply substitution to typed node
	typedNode = tc.applySubstitutionToTyped(sub, typedNode)
	
	return typedNode, newEnv, nil
}

// inferCore performs type inference on Core expressions
func (tc *CoreTypeChecker) inferCore(ctx *InferenceContext, expr core.CoreExpr) (typedast.TypedNode, *TypeEnv, error) {
	switch e := expr.(type) {
	case *core.Lit:
		return tc.inferLit(ctx, e)
		
	case *core.Var:
		return tc.inferVar(ctx, e)
		
	case *core.Lambda:
		return tc.inferLambda(ctx, e)
		
	case *core.Let:
		return tc.inferLet(ctx, e)
		
	case *core.LetRec:
		return tc.inferLetRec(ctx, e)
		
	case *core.App:
		return tc.inferApp(ctx, e)
		
	case *core.If:
		return tc.inferIf(ctx, e)
		
	case *core.BinOp:
		return tc.inferBinOp(ctx, e)
		
	case *core.UnOp:
		return tc.inferUnOp(ctx, e)
		
	case *core.Record:
		return tc.inferRecord(ctx, e)
		
	case *core.RecordAccess:
		return tc.inferRecordAccess(ctx, e)
		
	case *core.List:
		return tc.inferList(ctx, e)
		
	case *core.Match:
		return tc.inferMatch(ctx, e)
		
	default:
		return nil, ctx.env, fmt.Errorf("type inference not implemented for %T", expr)
	}
}

// inferLit infers type of literal
func (tc *CoreTypeChecker) inferLit(ctx *InferenceContext, lit *core.Lit) (*typedast.TypedLit, *TypeEnv, error) {
	var typ Type
	switch lit.Kind {
	case core.IntLit:
		typ = TInt
	case core.FloatLit:
		typ = TFloat
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
		monotype = scheme.Instantiate(ctx.freshType).(Type)
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
	
	// Lambda type
	var bodyEffectRow *Row
	if effRow := bodyNode.GetEffectRow(); effRow != nil {
		bodyEffectRow = effRow.(*Row)
	}
	funcType := &TFunc2{
		Params:    paramTypes,
		EffectRow: bodyEffectRow,
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
	
	// Generalize if value is syntactic value (value restriction)
	var binding interface{}
	if isCoreValue(let.Value) {
		binding = ctx.generalize(getType(valueNode), getEffectRow(valueNode))
	} else {
		binding = valueNode.GetType()
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
			EffectRow: combineEffects(getEffectRow(valueNode), getEffectRow(bodyNode)),
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
	
	// Infer types of all values
	typedBindings := make([]typedast.TypedRecBinding, len(letrec.Bindings))
	for i, binding := range letrec.Bindings {
		valueNode, _, err := tc.inferCore(ctx, binding.Value)
		if err != nil {
			return nil, oldEnv, err
		}
		
		// Unify with expected type
		ctx.addConstraint(TypeEq{
			Left:  bindingTypes[binding.Name],
			Right: getType(valueNode),
			Path:  []string{binding.Name},
		})
		
		// Generalize for recursion
		scheme := ctx.generalize(getType(valueNode), getEffectRow(valueNode))
		
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
		if node, ok := binding.Value.(typedast.TypedNode); ok {
			allEffects = append(allEffects, getEffectRow(node))
		}
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

// inferBinOp infers type of binary operation
func (tc *CoreTypeChecker) inferBinOp(ctx *InferenceContext, binop *core.BinOp) (*typedast.TypedBinOp, *TypeEnv, error) {
	// Infer operand types
	leftNode, _, err := tc.inferCore(ctx, binop.Left)
	if err != nil {
		return nil, ctx.env, err
	}
	
	rightNode, _, err := tc.inferCore(ctx, binop.Right)
	if err != nil {
		return nil, ctx.env, err
	}
	
	// Determine result type based on operator
	var resultType Type
	
	switch binop.Op {
	case "+", "-", "*", "/", "%":
		// Arithmetic operators - require Num constraint
		ctx.addConstraint(ClassConstraint{
			Class: "Num",
			Type:  getType(leftNode),
			Path:  []string{binop.Span().String()},
		})
		ctx.addConstraint(ClassConstraint{
			Class: "Num",
			Type:  getType(rightNode),
			Path:  []string{binop.Span().String()},
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: getType(rightNode),
			Path:  []string{"arithmetic at " + binop.Span().String()},
		})
		resultType = getType(leftNode)
		
	case "++":
		// String concatenation
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: TString,
			Path:  []string{"string concat at " + binop.Span().String()},
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(rightNode),
			Right: TString,
			Path:  []string{"string concat at " + binop.Span().String()},
		})
		resultType = TString
		
	case "<", ">", "<=", ">=":
		// Comparison operators - require Ord constraint
		ctx.addConstraint(ClassConstraint{
			Class: "Ord",
			Type:  getType(leftNode),
			Path:  []string{binop.Span().String()},
		})
		ctx.addConstraint(ClassConstraint{
			Class: "Ord",
			Type:  getType(rightNode),
			Path:  []string{binop.Span().String()},
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: getType(rightNode),
			Path:  []string{"comparison at " + binop.Span().String()},
		})
		resultType = TBool
		
	case "==", "!=":
		// Equality - require Eq constraint
		ctx.addConstraint(ClassConstraint{
			Class: "Eq",
			Type:  getType(leftNode),
			Path:  []string{binop.Span().String()},
		})
		ctx.addConstraint(ClassConstraint{
			Class: "Eq",
			Type:  getType(rightNode),
			Path:  []string{binop.Span().String()},
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: getType(rightNode),
			Path:  []string{"equality at " + binop.Span().String()},
		})
		resultType = TBool
		
	case "&&", "||":
		// Boolean operators
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: TBool,
			Path:  []string{"boolean op at " + binop.Span().String()},
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(rightNode),
			Right: TBool,
			Path:  []string{"boolean op at " + binop.Span().String()},
		})
		resultType = TBool
		
	default:
		return nil, ctx.env, fmt.Errorf("unknown binary operator: %s", binop.Op)
	}
	
	// Combine effects
	effects := combineEffects(getEffectRow(leftNode), getEffectRow(rightNode))
	
	return &typedast.TypedBinOp{
		TypedExpr: typedast.TypedExpr{
			NodeID:    binop.ID(),
			Span:      binop.Span(),
			Type:      resultType,
			EffectRow: effects,
			Core:      binop,
		},
		Op:    binop.Op,
		Left:  leftNode,
		Right: rightNode,
	}, ctx.env, nil
}

// inferUnOp infers type of unary operation
func (tc *CoreTypeChecker) inferUnOp(ctx *InferenceContext, unop *core.UnOp) (*typedast.TypedUnOp, *TypeEnv, error) {
	// Infer operand type
	operandNode, _, err := tc.inferCore(ctx, unop.Operand)
	if err != nil {
		return nil, ctx.env, err
	}
	
	var resultType Type
	
	switch unop.Op {
	case "-":
		// Negation - requires Num constraint
		ctx.addConstraint(ClassConstraint{
			Class: "Num",
			Type:  getType(operandNode),
			Path:  []string{unop.Span().String()},
		})
		resultType = getType(operandNode)
		
	case "not":
		// Boolean negation
		ctx.addConstraint(TypeEq{
			Left:  getType(operandNode),
			Right: TBool,
			Path:  []string{"not at " + unop.Span().String()},
		})
		resultType = TBool
		
	default:
		return nil, ctx.env, fmt.Errorf("unknown unary operator: %s", unop.Op)
	}
	
	return &typedast.TypedUnOp{
		TypedExpr: typedast.TypedExpr{
			NodeID:    unop.ID(),
			Span:      unop.Span(),
			Type:      resultType,
			EffectRow: getEffectRow(operandNode),
			Core:      unop,
		},
		Op:      unop.Op,
		Operand: operandNode,
	}, ctx.env, nil
}

// inferRecord infers type of record construction
func (tc *CoreTypeChecker) inferRecord(ctx *InferenceContext, rec *core.Record) (*typedast.TypedRecord, *TypeEnv, error) {
	fields := make(map[string]typedast.TypedNode)
	fieldTypes := make(map[string]Type)
	var allEffects []*Row
	
	for name, value := range rec.Fields {
		valueNode, _, err := tc.inferCore(ctx, value)
		if err != nil {
			return nil, ctx.env, err
		}
		fields[name] = valueNode
		fieldTypes[name] = getType(valueNode)
		allEffects = append(allEffects, getEffectRow(valueNode))
	}
	
	// Create record type
	recordType := &TRecord{
		Fields: fieldTypes,
		Row: &Row{
			Kind:   RecordRow,
			Labels: fieldTypes,
			Tail:   nil, // Closed record for now
		},
	}
	
	return &typedast.TypedRecord{
		TypedExpr: typedast.TypedExpr{
			NodeID:    rec.ID(),
			Span:      rec.Span(),
			Type:      recordType,
			EffectRow: combineEffectList(allEffects),
			Core:      rec,
		},
		Fields: fields,
	}, ctx.env, nil
}

// inferRecordAccess infers type of field access
func (tc *CoreTypeChecker) inferRecordAccess(ctx *InferenceContext, acc *core.RecordAccess) (*typedast.TypedRecordAccess, *TypeEnv, error) {
	// Infer record type
	recordNode, _, err := tc.inferCore(ctx, acc.Record)
	if err != nil {
		return nil, ctx.env, err
	}
	
	// Create fresh type variable for field
	fieldType := ctx.freshTypeVar()
	rowVar := &RowVar{Name: "r", Kind: RecordRow}
	
	// Expected record type with field
	expectedRecord := &TRecord{
		Fields: map[string]Type{acc.Field: fieldType},
		Row: &Row{
			Kind:   RecordRow,
			Labels: map[string]Type{acc.Field: fieldType},
			Tail:   rowVar, // Row polymorphic
		},
	}
	
	ctx.addConstraint(TypeEq{
		Left:  getType(recordNode),
		Right: expectedRecord,
		Path:  []string{"field access at " + acc.Span().String()},
	})
	
	return &typedast.TypedRecordAccess{
		TypedExpr: typedast.TypedExpr{
			NodeID:    acc.ID(),
			Span:      acc.Span(),
			Type:      fieldType,
			EffectRow: getEffectRow(recordNode),
			Core:      acc,
		},
		Record: recordNode,
		Field:  acc.Field,
	}, ctx.env, nil
}

// inferList infers type of list construction
func (tc *CoreTypeChecker) inferList(ctx *InferenceContext, list *core.List) (*typedast.TypedList, *TypeEnv, error) {
	if len(list.Elements) == 0 {
		// Empty list - polymorphic
		elemType := ctx.freshTypeVar()
		return &typedast.TypedList{
			TypedExpr: typedast.TypedExpr{
				NodeID:    list.ID(),
				Span:      list.Span(),
				Type:      &TList{Element: elemType},
				EffectRow: EmptyEffectRow(),
				Core:      list,
			},
			Elements: nil,
		}, ctx.env, nil
	}
	
	// Non-empty list - all elements must have same type
	var elements []typedast.TypedNode
	var allEffects []*Row
	
	firstElem, _, err := tc.inferCore(ctx, list.Elements[0])
	if err != nil {
		return nil, ctx.env, err
	}
	elements = append(elements, firstElem)
	allEffects = append(allEffects, getEffectRow(firstElem))
	elemType := getType(firstElem)
	
	for i := 1; i < len(list.Elements); i++ {
		elemNode, _, err := tc.inferCore(ctx, list.Elements[i])
		if err != nil {
			return nil, ctx.env, err
		}
		elements = append(elements, elemNode)
		allEffects = append(allEffects, getEffectRow(elemNode))
		
		// All elements must have same type
		ctx.addConstraint(TypeEq{
			Left:  getType(elemNode),
			Right: elemType,
			Path:  []string{fmt.Sprintf("list element %d at %s", i, list.Span())},
		})
	}
	
	return &typedast.TypedList{
		TypedExpr: typedast.TypedExpr{
			NodeID:    list.ID(),
			Span:      list.Span(),
			Type:      &TList{Element: elemType},
			EffectRow: combineEffectList(allEffects),
			Core:      list,
		},
		Elements: elements,
	}, ctx.env, nil
}

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
		
		// Type check body
		bodyNode, _, err := tc.inferCore(ctx, arm.Body)
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
		case int:
			litType = TInt
		case float64:
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
		
	default:
		return nil, nil, fmt.Errorf("pattern type checking not implemented for %T", pat)
	}
}

// Helper functions

// toInterfaceSlice converts []Type to []interface{}
func toInterfaceSlice(types []Type) []interface{} {
	result := make([]interface{}, len(types))
	for i, t := range types {
		result[i] = t
	}
	return result
}

// getType safely extracts Type from interface{}
func getType(node typedast.TypedNode) Type {
	if t := node.GetType(); t != nil {
		return t.(Type)
	}
	return nil
}

// getEffectRow safely extracts Row from interface{}
func getEffectRow(node typedast.TypedNode) *Row {
	if r := node.GetEffectRow(); r != nil {
		return r.(*Row)
	}
	return EmptyEffectRow()
}

// isCoreValue checks if expression is a syntactic value (for value restriction)
func isCoreValue(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.Lit, *core.Lambda:
		return true
	case *core.Var:
		// Variables are values if bound to values
		return true
	case *core.Record:
		// Record is value if all fields are values
		for _, field := range e.Fields {
			if !isCoreValue(field) {
				return false
			}
		}
		return true
	case *core.List:
		// List is value if all elements are values
		for _, elem := range e.Elements {
			if !isCoreValue(elem) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

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

// combineEffects combines two effect rows
func combineEffects(e1, e2 *Row) *Row {
	if e1 == nil || (len(e1.Labels) == 0 && e1.Tail == nil) {
		return e2
	}
	if e2 == nil || (len(e2.Labels) == 0 && e2.Tail == nil) {
		return e1
	}
	
	// Combine labels
	combined := make(map[string]Type)
	for k, v := range e1.Labels {
		combined[k] = v
	}
	for k, v := range e2.Labels {
		combined[k] = v
	}
	
	// For now, ignore tail variables in combination
	// Full implementation would handle row unification
	return &Row{
		Kind:   EffectRow,
		Labels: combined,
		Tail:   nil,
	}
}

// combineEffectList combines multiple effect rows
func combineEffectList(effects []*Row) *Row {
	result := EmptyEffectRow()
	for _, e := range effects {
		result = combineEffects(result, e)
	}
	return result
}

// applySubstitutionToTyped applies substitution to typed nodes
func (tc *CoreTypeChecker) applySubstitutionToTyped(sub Substitution, node typedast.TypedNode) typedast.TypedNode {
	// Apply substitution to type and effect row
	// This is simplified - full implementation would rebuild entire typed tree
	// For now, we assume the typed nodes are built with final types
	return node
}

// formatErrors formats all collected errors
func (tc *CoreTypeChecker) formatErrors() error {
	if len(tc.errors) == 0 {
		return nil
	}
	
	// Format errors with diagnostics
	var messages []string
	for _, err := range tc.errors {
		messages = append(messages, err.Error())
	}
	
	return fmt.Errorf("Type checking failed:\n%s", strings.Join(messages, "\n"))
}