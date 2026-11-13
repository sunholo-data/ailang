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
	envFreeVars := ctx.env.ownedVars().types // read-only

	generalizedTypeVars := []string{}
	for v := range typeFreeVars {
		if !envFreeVars[v] {
			generalizedTypeVars = append(generalizedTypeVars, v)
		}
	}

	// Find free row variables in effects but not in environment
	effectFreeVars := freeRowVars(effects)
	envFreeRowVars := ctx.env.ownedVars().rows // read-only

	generalizedRowVars := []string{}
	for v := range effectFreeVars {
		if !envFreeRowVars[v] {
			generalizedRowVars = append(generalizedRowVars, v)
		}
	}

	// Collect the class constraints on generalized variables (see schemeConstraintsFor)
	var classConstraints []ClassConstraint
	for _, c := range ctx.constraints {
		if cc, ok := c.(ClassConstraint); ok {
			classConstraints = append(classConstraints, cc)
		}
	}

	return &Scheme{
		TypeVars:    generalizedTypeVars,
		RowVars:     generalizedRowVars,
		Constraints: schemeConstraintsFor(classConstraints, generalizedTypeVars),
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
	if from < 0 {
		from = 0
	}
	return ctx.solveEqualitiesInto(make(Substitution), ctx.constraints[from:])
}

// solveEqualitiesInto folds the equality constraints of cs, in order, into sub.
// Folding a suffix into the result of its prefix equals folding the whole, so
// callers may extend a cached solve incrementally (#1958 unionEffects).
func (ctx *InferenceContext) solveEqualitiesInto(sub Substitution, cs []TypeConstraint) (Substitution, error) {
	sub, _, err := ctx.solveEqualitiesPending(sub, cs)
	return sub, err
}

// solveEqualitiesPending is solveEqualitiesInto that also folds every field
// access (field_access_deferred.go) whose receiver the equalities decide, and
// returns the accesses still undecided.
func (ctx *InferenceContext) solveEqualitiesPending(sub Substitution, cs []TypeConstraint) (Substitution, []*fieldAccess, error) {
	var pending []*fieldAccess
	for _, c := range cs {
		switch constraint := c.(type) {
		case TypeEq:
			var err error
			sub, err = ctx.unifier.Unify(
				ApplySubstitution(sub, constraint.Left),
				ApplySubstitution(sub, constraint.Right),
				sub,
			)
			if err != nil {
				return nil, nil, ctx.unifyFailure(constraint, sub, err)
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
		case *fieldAccess:
			next, decided, err := ctx.solveFieldAccess(constraint, sub)
			if err != nil {
				return nil, nil, err
			}
			if !decided {
				pending = append(pending, constraint)
				continue
			}
			sub = next
		}
	}
	// A later equality may have decided a receiver an earlier access met unknown.
	return ctx.retryFieldAccesses(sub, pending)
}

// SolveConstraints solves all collected constraints
func (ctx *InferenceContext) SolveConstraints() (Substitution, []ClassConstraint, error) {
	sub := make(Substitution)

	// Phase 1: Solve all equality constraints first to build up substitution,
	// with the field accesses they decide (field_access_deferred.go).
	sub, pending, err := ctx.solveEqualitiesPending(sub, ctx.constraints)
	if err != nil {
		return nil, nil, err
	}

	// Phase 1b (M-V1-NOMINAL-TYPE-IDENTITY, #2239): resolve the deferred
	// constructor choices and field accesses the equalities now decide
	// (ambiguous_ctor.go, field_access_deferred.go).
	sub, err = ctx.solveDeferred(sub, pending)
	if err != nil {
		return nil, nil, err
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
