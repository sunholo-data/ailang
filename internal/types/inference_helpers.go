package types

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Helper functions for type inference

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

// ============================================================================
// Free variable analysis
// ============================================================================

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

// solveLocalEqualities solves ONLY the TypeEq/RowEq constraints in the suffix
// ctx.constraints[from:], starting from a fresh substitution, using the SAME
// Unifier/RowUnifier as SolveConstraints. It returns the accumulated
// substitution WITHOUT mutating ctx.constraints.
//
// M-EFFECT-ROW-SHOW-INTERP (#386) Section A: inferApp uses this to resolve the
// equality constraints an application subtree just added (notably the callee's
// funcType ~ expectedFuncType equality that binds the callee's effect row)
// BEFORE combining effect rows. This makes `show` resolve to closed {} and
// `println` to closed {IO} so their union is {IO}, instead of combineEffects
// prematurely dropping an unresolved open-row tail to closed {}.
//
// Class constraints are NOT solved here (they are handled at the normal
// generalization boundary); only equality constraints are processed so that no
// join representation is ever created and RowUnifier.UnifyRows only sees
// ordinary *Row values.
func (ctx *InferenceContext) solveLocalEqualities(from int) (Substitution, error) {
	sub := make(Substitution)
	if from < 0 {
		from = 0
	}
	for _, c := range ctx.constraints[from:] {
		switch constraint := c.(type) {
		case TypeEq:
			var err error
			sub, err = ctx.unifier.Unify(
				ApplySubstitution(sub, constraint.Left),
				ApplySubstitution(sub, constraint.Right),
				sub,
			)
			if err != nil {
				return nil, fmt.Errorf("type unification failed at %v: %w", constraint.Path, err)
			}
		case RowEq:
			var err error
			sub, err = ctx.unifier.rowUnifier.UnifyRows(
				constraint.Left,
				constraint.Right,
				sub,
			)
			if err != nil {
				return nil, fmt.Errorf("row unification failed at %v: %w", constraint.Path, err)
			}
		}
	}
	return sub, nil
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
	case *TArray:
		collectFreeTypeVars(t.Element, free)
	case *TMap:
		collectFreeTypeVars(t.Key, free)
		collectFreeTypeVars(t.Value, free)
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

// collectFreeEffectRowVarsInType walks a COMPLETE type tree and records the name
// of every free EffectRow-kinded row variable it finds: outer function effect
// rows, nested callback function effect rows, and rows nested inside collections,
// tuples, maps, records, and type applications (ADTs).
//
// M-EFFECT-ROW-SHOW-INTERP (#386) Section B: generalizeWithConstraints uses this
// to quantify a scheme's free effect-row variables so each imported
// row-polymorphic combinator use (mapE/filterE/foldlE/flatMapE/forEachE) receives
// FRESH row variables. The existing collectFreeRowVars only inspected a single
// row's tail and record labels — it never descended into function types, so the
// combinator callback/outer effect rows were missed.
func collectFreeEffectRowVarsInType(t Type, free map[string]bool, visited map[Type]bool) {
	if t == nil || visited[t] {
		return
	}
	visited[t] = true

	switch typ := t.(type) {
	case *TFunc2:
		for _, p := range typ.Params {
			collectFreeEffectRowVarsInType(p, free, visited)
		}
		collectFreeEffectRowVarsInType(typ.Return, free, visited)
		if typ.EffectRow != nil {
			collectFreeEffectRowVarsInRow(typ.EffectRow, free, visited)
		}
	case *TList:
		collectFreeEffectRowVarsInType(typ.Element, free, visited)
	case *TArray:
		collectFreeEffectRowVarsInType(typ.Element, free, visited)
	case *TMap:
		collectFreeEffectRowVarsInType(typ.Key, free, visited)
		collectFreeEffectRowVarsInType(typ.Value, free, visited)
	case *TTuple:
		for _, e := range typ.Elements {
			collectFreeEffectRowVarsInType(e, free, visited)
		}
	case *TApp:
		collectFreeEffectRowVarsInType(typ.Constructor, free, visited)
		for _, a := range typ.Args {
			collectFreeEffectRowVarsInType(a, free, visited)
		}
	case *TRecord:
		for _, ft := range typ.Fields {
			collectFreeEffectRowVarsInType(ft, free, visited)
		}
		if typ.Row != nil {
			collectFreeEffectRowVarsInType(typ.Row, free, visited)
		}
	case *TRecord2:
		if typ.Row != nil {
			collectFreeEffectRowVarsInRow(typ.Row, free, visited)
		}
	case *Row:
		collectFreeEffectRowVarsInRow(typ, free, visited)
	}
}

// collectFreeEffectRowVarsInRow records an EffectRow tail var and recurses into
// the row's label types (which may themselves contain nested function/effect
// rows). Record-row tails are intentionally NOT collected as effect vars.
func collectFreeEffectRowVarsInRow(r *Row, free map[string]bool, visited map[Type]bool) {
	if r == nil {
		return
	}
	if r.Tail != nil && r.Tail.Kind.Equals(EffectRow) {
		free[r.Tail.Name] = true
	}
	for _, lt := range r.Labels {
		collectFreeEffectRowVarsInType(lt, free, visited)
	}
}

// freeEffectRowVarsInType is the convenience wrapper returning the set of free
// EffectRow-kinded row-variable names in a complete type.
func freeEffectRowVarsInType(t Type) map[string]bool {
	free := make(map[string]bool)
	collectFreeEffectRowVarsInType(t, free, make(map[Type]bool))
	return free
}

// FreeEffectRowVarNames returns the sorted set of free EffectRow-kinded
// row-variable names in a complete type. Exported for the interface builder,
// which recomputes a scheme's quantified row vars after restoring source-level
// effect rows from the AST. M-EFFECT-ROW-SHOW-INTERP (#386).
func FreeEffectRowVarNames(t Type) []string {
	free := freeEffectRowVarsInType(t)
	names := make([]string, 0, len(free))
	for v := range free {
		names = append(names, v)
	}
	sort.Strings(names)
	return names
}

// checkLinearCapture analyzes lambda for linear value capture violations
func (ctx *InferenceContext) checkLinearCapture(lambda *ast.Lambda, _ []Type) error {
	// Find all free variables in the lambda body
	freeVars := findFreeVariables(lambda.Body, getParamNames(lambda.Params))

	// Check if any captured variables have linear capabilities
	for varName := range freeVars {
		varType, err := ctx.env.Lookup(varName)
		if err != nil {
			continue // Variable not in scope, will be caught by type inference
		}

		// Check if the variable type contains linear capabilities
		if hasLinearCapabilities(varType) {
			// Get the linear capability names for error reporting
			linearCaps := getLinearCapabilities(varType)
			for _, capName := range linearCaps {
				return fmt.Errorf("lambda captures linear value %s; pass it as a parameter instead", capName)
			}
		}
	}

	return nil
}

// findFreeVariables finds free variables in an expression, excluding bound parameters
func findFreeVariables(expr ast.Expr, boundParams []string) map[string]bool {
	freeVars := make(map[string]bool)
	boundSet := make(map[string]bool)
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
		if !bound[e.Name] {
			freeVars[e.Name] = true
		}
	case *ast.Lambda:
		newBound := make(map[string]bool)
		for k, v := range bound {
			newBound[k] = v
		}
		for _, param := range e.Params {
			newBound[param.Name] = true
		}
		findFreeVarsHelper(e.Body, freeVars, newBound)
	case *ast.Let:
		findFreeVarsHelper(e.Value, freeVars, bound)
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
	case *ast.Literal: // no-op
	default: // conservatively assume no free variables
	}
}
