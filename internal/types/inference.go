package types

import (
	"fmt"
	"github.com/sunholo/ailang/internal/ast"
)

// InferenceContext maintains state during type inference
type InferenceContext struct {
	env                   *TypeEnv
	unifier               *Unifier
	constraints           []TypeConstraint
	freshCounter          int
	path                  []string         // For error reporting
	qualifiedConstraints  []ClassConstraint // Non-ground constraints for qualified types
}

// TypeConstraint represents a constraint to be solved
type TypeConstraint interface {
	constraint()
	String() string
}

// TypeEq represents a type equality constraint
type TypeEq struct {
	Left  Type
	Right Type
	Path  []string // Source location path
}

func (c TypeEq) constraint()   {}
func (c TypeEq) String() string { return fmt.Sprintf("%s ~ %s", c.Left, c.Right) }

// RowEq represents a row equality constraint
type RowEq struct {
	Left  *Row
	Right *Row
	Path  []string
}

func (c RowEq) constraint()   {}
func (c RowEq) String() string { return fmt.Sprintf("%s ~ %s", c.Left, c.Right) }

// ClassConstraint represents a type class constraint
type ClassConstraint struct {
	Class  string
	Type   Type
	Path   []string
	NodeID uint64 // NodeID of the Core expression that generated this constraint
}

func (c ClassConstraint) constraint()   {}
func (c ClassConstraint) String() string { return fmt.Sprintf("%s[%s]", c.Class, c.Type) }

// NewInferenceContext creates a new inference context
func NewInferenceContext() *InferenceContext {
	return &InferenceContext{
		env:         NewTypeEnv(),
		unifier:     NewUnifier(),
		constraints: []TypeConstraint{},
		freshCounter: 0,
		path:        []string{},
	}
}

// SetEnv sets the type environment for the inference context
func (ctx *InferenceContext) SetEnv(env *TypeEnv) {
	ctx.env = env
}

// Infer performs type inference on an expression
func (ctx *InferenceContext) Infer(expr ast.Expr) (Type, *Row, error) {
	switch e := expr.(type) {
	case *ast.Literal:
		switch e.Kind {
		case ast.IntLit:
			return TInt, EmptyEffectRow(), nil
		case ast.FloatLit:
			return TFloat, EmptyEffectRow(), nil
		case ast.StringLit:
			return TString, EmptyEffectRow(), nil
		case ast.BoolLit:
			return TBool, EmptyEffectRow(), nil
		case ast.UnitLit:
			return TUnit, EmptyEffectRow(), nil
		default:
			return nil, nil, fmt.Errorf("unknown literal kind: %v", e.Kind)
		}

	case *ast.Identifier:
		typ, err := ctx.env.Lookup(e.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("undefined variable: %s", e.Name)
		}
		// Instantiate if it's a scheme
		if scheme, ok := typ.(*Scheme); ok {
			instType := scheme.Instantiate(ctx.freshType)
			return instType.(Type), EmptyEffectRow(), nil
		}
		if t, ok := typ.(Type); ok {
			return t, EmptyEffectRow(), nil
		}
		return nil, nil, fmt.Errorf("invalid type in environment: %T", typ)

	case *ast.Lambda:
		// Fresh type variables for parameters
		paramTypes := make([]Type, len(e.Params))
		for i, param := range e.Params {
			paramType := ctx.freshTypeVar()
			paramTypes[i] = paramType
			
			// Bind parameter in environment
			ctx.env = ctx.env.Extend(param.Name, paramType)
		}

		// Infer body type and effects
		bodyType, bodyEffects, err := ctx.Infer(e.Body)
		if err != nil {
			return nil, nil, err
		}

		// Check for linear capture violations
		if err := ctx.checkLinearCapture(e, paramTypes); err != nil {
			return nil, nil, err
		}

		// Lambda itself is pure, but carries latent effects
		return &TFunc2{
			Params:    paramTypes,
			EffectRow: bodyEffects,
			Return:    bodyType,
		}, EmptyEffectRow(), nil

	case *ast.FuncCall:
		// Infer function type and effects
		fnType, fnEffects, err := ctx.Infer(e.Func)
		if err != nil {
			return nil, nil, err
		}

		// Infer argument types and collect effects
		argTypes := make([]Type, len(e.Args))
		allEffects := []*Row{fnEffects}

		for i, arg := range e.Args {
			argType, argEffects, err := ctx.Infer(arg)
			if err != nil {
				return nil, nil, err
			}
			argTypes[i] = argType
			allEffects = append(allEffects, argEffects)
		}

		// Create fresh result type and effect row
		resultType := ctx.freshTypeVar()
		resultEffects := ctx.freshEffectRow()

		// Generate function type constraint
		expectedFnType := &TFunc2{
			Params:    argTypes,
			EffectRow: resultEffects,
			Return:    resultType,
		}

		ctx.addConstraint(TypeEq{
			Left:  fnType,
			Right: expectedFnType,
			Path:  ctx.path,
		})

		// Union all effects (function eval + args + function's latent effects)
		totalEffects := UnionEffects(allEffects...)
		totalEffects = UnionEffects(totalEffects, resultEffects)

		return resultType, totalEffects, nil

	case *ast.Let:
		// Infer binding type and effects
		bindType, bindEffects, err := ctx.Infer(e.Value)
		if err != nil {
			return nil, nil, err
		}

		// Value restriction: only generalize if RHS is a syntactic value
		if isValue(e.Value) {
			// Generalize free variables
			scheme := ctx.generalize(bindType, bindEffects)
			ctx.env = ctx.env.ExtendScheme(e.Name, scheme)
		} else {
			// Monomorphic binding (don't generalize)
			ctx.env = ctx.env.Extend(e.Name, bindType)
		}

		// Infer body type
		return ctx.Infer(e.Body)

	case *ast.If:
		// Infer condition
		condType, condEffects, err := ctx.Infer(e.Condition)
		if err != nil {
			return nil, nil, err
		}

		// Condition must be boolean
		ctx.addConstraint(TypeEq{
			Left:  condType,
			Right: TBool,
			Path:  append(ctx.path, "condition"),
		})

		// Infer both branches
		thenType, thenEffects, err := ctx.Infer(e.Then)
		if err != nil {
			return nil, nil, err
		}

		elseType, elseEffects, err := ctx.Infer(e.Else)
		if err != nil {
			return nil, nil, err
		}

		// Branches must have same type
		ctx.addConstraint(TypeEq{
			Left:  thenType,
			Right: elseType,
			Path:  ctx.path,
		})

		// Union all effects
		totalEffects := UnionEffects(condEffects, thenEffects, elseEffects)

		return thenType, totalEffects, nil

	case *ast.BinaryOp:
		// Infer operand types
		leftType, leftEffects, err := ctx.Infer(e.Left)
		if err != nil {
			return nil, nil, err
		}

		rightType, rightEffects, err := ctx.Infer(e.Right)
		if err != nil {
			return nil, nil, err
		}

		// Determine result type based on operator
		var resultType Type
		switch e.Op {
		case "+", "-", "*", "/", "%":
			// Numeric operators - need Num constraint
			ctx.addConstraint(ClassConstraint{
				Class: "Num",
				Type:  leftType,
				Path:  ctx.path,
			})
			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: rightType,
				Path:  ctx.path,
			})
			resultType = leftType

		case "<", ">", "<=", ">=":
			// Comparison operators - need Ord constraint
			ctx.addConstraint(ClassConstraint{
				Class: "Ord",
				Type:  leftType,
				Path:  ctx.path,
			})
			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: rightType,
				Path:  ctx.path,
			})
			resultType = TBool

		case "==", "!=":
			// Equality operators - need Eq constraint
			ctx.addConstraint(ClassConstraint{
				Class: "Eq",
				Type:  leftType,
				Path:  ctx.path,
			})
			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: rightType,
				Path:  ctx.path,
			})
			resultType = TBool

		case "&&", "||":
			// Boolean operators
			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: TBool,
				Path:  append(ctx.path, "left"),
			})
			ctx.addConstraint(TypeEq{
				Left:  rightType,
				Right: TBool,
				Path:  append(ctx.path, "right"),
			})
			resultType = TBool

		default:
			return nil, nil, fmt.Errorf("unknown operator: %s", e.Op)
		}

		// Union effects
		totalEffects := UnionEffects(leftEffects, rightEffects)
		return resultType, totalEffects, nil

	case *ast.List:
		if len(e.Elements) == 0 {
			// Empty list - fresh type variable for element
			elemType := ctx.freshTypeVar()
			return &TList{Element: elemType}, EmptyEffectRow(), nil
		}

		// Infer first element type
		elemType, effects, err := ctx.Infer(e.Elements[0])
		if err != nil {
			return nil, nil, err
		}

		allEffects := []*Row{effects}

		// All elements must have same type
		for i := 1; i < len(e.Elements); i++ {
			otherType, otherEffects, err := ctx.Infer(e.Elements[i])
			if err != nil {
				return nil, nil, err
			}
			ctx.addConstraint(TypeEq{
				Left:  elemType,
				Right: otherType,
				Path:  append(ctx.path, fmt.Sprintf("element[%d]", i)),
			})
			allEffects = append(allEffects, otherEffects)
		}

		totalEffects := UnionEffects(allEffects...)
		return &TList{Element: elemType}, totalEffects, nil

	case *ast.Tuple:
		elemTypes := make([]Type, len(e.Elements))
		allEffects := []*Row{}

		for i, elem := range e.Elements {
			elemType, elemEffects, err := ctx.Infer(elem)
			if err != nil {
				return nil, nil, err
			}
			elemTypes[i] = elemType
			allEffects = append(allEffects, elemEffects)
		}

		totalEffects := UnionEffects(allEffects...)
		return &TTuple{Elements: elemTypes}, totalEffects, nil

	case *ast.Record:
		labels := make(map[string]Type)
		allEffects := []*Row{}

		for _, field := range e.Fields {
			valType, valEffects, err := ctx.Infer(field.Value)
			if err != nil {
				return nil, nil, err
			}
			labels[field.Name] = valType
			allEffects = append(allEffects, valEffects)
		}

		// Create record with open row
		recordRow := &Row{
			Kind:   RecordRow,
			Labels: labels,
			Tail:   ctx.freshRecordRow(),
		}

		totalEffects := UnionEffects(allEffects...)
		return &TRecord2{Row: recordRow}, totalEffects, nil

	case *ast.RecordAccess:
		// Infer record type
		recordType, recordEffects, err := ctx.Infer(e.Record)
		if err != nil {
			return nil, nil, err
		}

		// Fresh type for the field
		fieldType := ctx.freshTypeVar()

		// Generate constraint that record has this field
		expectedRow := &Row{
			Kind:   RecordRow,
			Labels: map[string]Type{e.Field: fieldType},
			Tail:   ctx.freshRecordRow(),
		}

		ctx.addConstraint(TypeEq{
			Left:  recordType,
			Right: &TRecord2{Row: expectedRow},
			Path:  ctx.path,
		})

		return fieldType, recordEffects, nil

	default:
		return nil, nil, fmt.Errorf("type inference not implemented for %T", expr)
	}
}

// isValue checks if an expression is a syntactic value (for value restriction)
func isValue(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Lambda:
		return true
	case *ast.Literal:
		return true
	case *ast.List:
		// List is a value if all elements are values
		for _, elem := range e.Elements {
			if !isValue(elem) {
				return false
			}
		}
		return true
	case *ast.Tuple:
		// Tuple is a value if all elements are values
		for _, elem := range e.Elements {
			if !isValue(elem) {
				return false
			}
		}
		return true
	case *ast.Record:
		// Record is a value if all fields are values
		for _, field := range e.Fields {
			if !isValue(field.Value) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// generalize creates a type scheme by generalizing free variables
func (ctx *InferenceContext) generalize(typ Type, effects *Row) *Scheme {
	// Find free type variables in type but not in environment
	typeFreeVars := freeTypeVars(typ)
	envFreeVars := ctx.env.FreeTypeVars()
	
	generalizedTypeVars := []string{}
	for v := range typeFreeVars {
		if !envFreeVars[v] {
			generalizedTypeVars = append(generalizedTypeVars, v)
		}
	}

	// Find free row variables in effects but not in environment
	effectFreeVars := freeRowVars(effects)
	envFreeRowVars := ctx.env.FreeRowVars()
	
	generalizedRowVars := []string{}
	for v := range effectFreeVars {
		if !envFreeRowVars[v] {
			generalizedRowVars = append(generalizedRowVars, v)
		}
	}

	// Collect any class constraints on generalized variables
	relevantConstraints := []Constraint{}
	for _, c := range ctx.constraints {
		if cc, ok := c.(ClassConstraint); ok {
			// Check if constraint mentions generalized variables
			// (simplified - full implementation would check properly)
			relevantConstraints = append(relevantConstraints, Constraint{
				Class: cc.Class,
				Type:  cc.Type,
			})
		}
	}

	return &Scheme{
		TypeVars:    generalizedTypeVars,
		RowVars:     generalizedRowVars,
		Constraints: relevantConstraints,
		Type:        typ,
	}
}

// Helper functions for fresh variables

func (ctx *InferenceContext) freshTypeVar() Type {
	ctx.freshCounter++
	return &TVar2{
		Name: fmt.Sprintf("α%d", ctx.freshCounter),
		Kind: Star,
	}
}

func (ctx *InferenceContext) freshEffectRow() *Row {
	ctx.freshCounter++
	return &Row{
		Kind:   EffectRow,
		Labels: make(map[string]Type),
		Tail: &RowVar{
			Name: fmt.Sprintf("ε%d", ctx.freshCounter),
			Kind: EffectRow,
		},
	}
}

func (ctx *InferenceContext) freshRecordRow() *RowVar {
	ctx.freshCounter++
	return &RowVar{
		Name: fmt.Sprintf("ρ%d", ctx.freshCounter),
		Kind: RecordRow,
	}
}

func (ctx *InferenceContext) freshType(kind Kind) Type {
	ctx.freshCounter++
	switch kind {
	case Star:
		return &TVar2{
			Name: fmt.Sprintf("α%d", ctx.freshCounter),
			Kind: Star,
		}
	case EffectRow:
		return &RowVar{
			Name: fmt.Sprintf("ε%d", ctx.freshCounter),
			Kind: EffectRow,
		}
	case RecordRow:
		return &RowVar{
			Name: fmt.Sprintf("ρ%d", ctx.freshCounter),
			Kind: RecordRow,
		}
	default:
		// Fallback
		return &TVar2{
			Name: fmt.Sprintf("τ%d", ctx.freshCounter),
			Kind: kind,
		}
	}
}

func (ctx *InferenceContext) addConstraint(c TypeConstraint) {
	ctx.constraints = append(ctx.constraints, c)
	
	// Also track class constraints separately for qualified types
	if cc, ok := c.(ClassConstraint); ok {
		ctx.qualifiedConstraints = append(ctx.qualifiedConstraints, cc)
	}
}

// SolveConstraints solves all collected constraints
func (ctx *InferenceContext) SolveConstraints() (Substitution, []ClassConstraint, error) {
	sub := make(Substitution)
	
	// Phase 1: Solve all equality constraints first to build up substitution
	for _, c := range ctx.constraints {
		switch constraint := c.(type) {
		case TypeEq:
			var err error
			sub, err = ctx.unifier.Unify(
				ApplySubstitution(sub, constraint.Left),
				ApplySubstitution(sub, constraint.Right),
				sub,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("type unification failed at %v: %w", constraint.Path, err)
			}

		case RowEq:
			var err error
			sub, err = ctx.unifier.rowUnifier.UnifyRows(
				constraint.Left,
				constraint.Right,
				sub,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("row unification failed at %v: %w", constraint.Path, err)
			}
		}
	}
	
	// Phase 2: Apply final substitution to all class constraints
	unsolvedClass := []ClassConstraint{}
	for _, c := range ctx.constraints {
		if constraint, ok := c.(ClassConstraint); ok {
			// Apply the complete substitution from Phase 1
			constraint.Type = ApplySubstitution(sub, constraint.Type)
			unsolvedClass = append(unsolvedClass, constraint)
		}
	}

	return sub, unsolvedClass, nil
}

// Helper functions for free variables

func freeTypeVars(t Type) map[string]bool {
	free := make(map[string]bool)
	collectFreeTypeVars(t, free)
	return free
}

func collectFreeTypeVars(t Type, free map[string]bool) {
	switch t := t.(type) {
	case *TVar2:
		free[t.Name] = true
	case *TFunc2:
		for _, p := range t.Params {
			collectFreeTypeVars(p, free)
		}
		collectFreeTypeVars(t.Return, free)
	case *TList:
		collectFreeTypeVars(t.Element, free)
	case *TTuple:
		for _, e := range t.Elements {
			collectFreeTypeVars(e, free)
		}
	case *TRecord2:
		if t.Row != nil {
			for _, v := range t.Row.Labels {
				collectFreeTypeVars(v, free)
			}
		}
	}
}

func freeRowVars(r *Row) map[string]bool {
	free := make(map[string]bool)
	if r != nil {
		collectFreeRowVars(r, free)
	}
	return free
}

func collectFreeRowVars(r *Row, free map[string]bool) {
	if r.Tail != nil {
		free[r.Tail.Name] = true
	}
	// For record rows, check types in labels
	if r.Kind.Equals(RecordRow) {
		for _, t := range r.Labels {
			// Types might contain row variables in nested records
			if rec, ok := t.(*TRecord2); ok && rec.Row != nil {
				collectFreeRowVars(rec.Row, free)
			}
		}
	}
}

// checkLinearCapture analyzes lambda for linear value capture violations
func (ctx *InferenceContext) checkLinearCapture(lambda *ast.Lambda, paramTypes []Type) error {
	// Find all free variables in the lambda body
	freeVars := findFreeVariables(lambda.Body, getParamNames(lambda.Params))
	
	// Check if any captured variables have linear capabilities
	for varName, _ := range freeVars {
		varType, err := ctx.env.Lookup(varName)
		if err != nil {
			continue // Variable not in scope, will be caught by type inference
		}
		
		// Check if the variable type contains linear capabilities
		if hasLinearCapabilities(varType) {
			// Get the linear capability names for error reporting
			linearCaps := getLinearCapabilities(varType)
			for _, capName := range linearCaps {
				return fmt.Errorf("Lambda captures linear value %s; pass it as a parameter instead", capName)
			}
		}
	}
	
	return nil
}

// findFreeVariables finds all free variables in an expression, excluding bound parameters
func findFreeVariables(expr ast.Expr, boundParams []string) map[string]bool {
	freeVars := make(map[string]bool)
	boundSet := make(map[string]bool)
	
	// Mark parameters as bound
	for _, param := range boundParams {
		boundSet[param] = true
	}
	
	findFreeVarsHelper(expr, freeVars, boundSet)
	return freeVars
}

// findFreeVarsHelper recursively finds free variables
func findFreeVarsHelper(expr ast.Expr, freeVars map[string]bool, bound map[string]bool) {
	if expr == nil {
		return
	}
	
	switch e := expr.(type) {
	case *ast.Identifier:
		// If identifier is not bound, it's a free variable
		if !bound[e.Name] {
			freeVars[e.Name] = true
		}
		
	case *ast.Lambda:
		// Create new bound set including lambda parameters
		newBound := make(map[string]bool)
		for k, v := range bound {
			newBound[k] = v
		}
		for _, param := range e.Params {
			newBound[param.Name] = true
		}
		findFreeVarsHelper(e.Body, freeVars, newBound)
		
	case *ast.Let:
		// Let binding creates a new scope
		findFreeVarsHelper(e.Value, freeVars, bound)
		
		// Create new bound set including let variable
		newBound := make(map[string]bool)
		for k, v := range bound {
			newBound[k] = v
		}
		newBound[e.Name] = true
		findFreeVarsHelper(e.Body, freeVars, newBound)
		
	case *ast.BinaryOp:
		findFreeVarsHelper(e.Left, freeVars, bound)
		findFreeVarsHelper(e.Right, freeVars, bound)
		
	case *ast.UnaryOp:
		findFreeVarsHelper(e.Expr, freeVars, bound)
		
	case *ast.FuncCall:
		findFreeVarsHelper(e.Func, freeVars, bound)
		for _, arg := range e.Args {
			findFreeVarsHelper(arg, freeVars, bound)
		}
		
	case *ast.If:
		findFreeVarsHelper(e.Condition, freeVars, bound)
		findFreeVarsHelper(e.Then, freeVars, bound)
		if e.Else != nil {
			findFreeVarsHelper(e.Else, freeVars, bound)
		}
		
	case *ast.List:
		for _, elem := range e.Elements {
			findFreeVarsHelper(elem, freeVars, bound)
		}
		
	case *ast.Record:
		for _, field := range e.Fields {
			findFreeVarsHelper(field.Value, freeVars, bound)
		}
		
	case *ast.RecordAccess:
		findFreeVarsHelper(e.Record, freeVars, bound)
		
	case *ast.Literal:
		// Literals don't contain variables
		
	default:
		// For other expression types, conservatively assume no free variables
		// In a real implementation, you'd handle all expression types
	}
}

// getParamNames extracts parameter names from lambda parameters
func getParamNames(params []*ast.Param) []string {
	names := make([]string, len(params))
	for i, param := range params {
		names[i] = param.Name
	}
	return names
}

// hasLinearCapabilities checks if a type contains linear capabilities
func hasLinearCapabilities(typ interface{}) bool {
	switch t := typ.(type) {
	case *TFunc2:
		// Check if function effects contain linear capabilities
		return hasLinearEffects(t.EffectRow)
	case *Scheme:
		// Check the underlying type
		if funcType, ok := t.Type.(*TFunc2); ok {
			return hasLinearEffects(funcType.EffectRow)
		}
	}
	return false
}

// hasLinearEffects checks if an effect row contains linear capabilities
func hasLinearEffects(effectRow *Row) bool {
	if effectRow == nil {
		return false
	}
	
	// Check for known linear capabilities in effect labels
	// In a real implementation, this would be configurable
	linearCapabilities := []string{"FS", "Net", "Time", "Rand", "Console"}
	
	for _, capName := range linearCapabilities {
		if _, exists := effectRow.Labels[capName]; exists {
			return true
		}
	}
	
	return false
}

// getLinearCapabilities returns the names of linear capabilities in a type
func getLinearCapabilities(typ interface{}) []string {
	var capabilities []string
	
	switch t := typ.(type) {
	case *TFunc2:
		capabilities = append(capabilities, getLinearEffectNames(t.EffectRow)...)
	case *Scheme:
		if funcType, ok := t.Type.(*TFunc2); ok {
			capabilities = append(capabilities, getLinearEffectNames(funcType.EffectRow)...)
		}
	}
	
	return capabilities
}

// getLinearEffectNames extracts linear capability names from an effect row
func getLinearEffectNames(effectRow *Row) []string {
	if effectRow == nil {
		return nil
	}
	
	var linearCaps []string
	linearCapabilities := []string{"FS", "Net", "Time", "Rand", "Console"}
	
	for _, capName := range linearCapabilities {
		if _, exists := effectRow.Labels[capName]; exists {
			linearCaps = append(linearCaps, capName)
		}
	}
	
	return linearCaps
}