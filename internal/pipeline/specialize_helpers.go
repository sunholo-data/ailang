package pipeline

import (
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// isRecursive checks if an expression contains a self-reference to the given name
func isRecursive(expr core.CoreExpr, name string) bool {
	switch e := expr.(type) {
	case *core.Var:
		return e.Name == name
	case *core.Lambda:
		// Check if name is shadowed by a parameter
		for _, param := range e.Params {
			if param == name {
				return false // Name is shadowed, not recursive
			}
		}
		return isRecursive(e.Body, name)
	case *core.Let:
		// Check value
		if isRecursive(e.Value, name) {
			return true
		}
		// Check body (name might be shadowed)
		if e.Name == name {
			return false // Shadowed
		}
		return isRecursive(e.Body, name)
	case *core.LetRec:
		// Check all bindings
		for _, binding := range e.Bindings {
			if isRecursive(binding.Value, name) {
				return true
			}
		}
		return isRecursive(e.Body, name)
	case *core.App:
		if isRecursive(e.Func, name) {
			return true
		}
		for _, arg := range e.Args {
			if isRecursive(arg, name) {
				return true
			}
		}
		return false
	case *core.If:
		return isRecursive(e.Cond, name) ||
			isRecursive(e.Then, name) ||
			isRecursive(e.Else, name)
	case *core.Match:
		if isRecursive(e.Scrutinee, name) {
			return true
		}
		for _, arm := range e.Arms {
			// Check if name is bound in pattern (shadowing)
			boundVars := patternBoundVars(arm.Pattern)
			shadowed := false
			for _, v := range boundVars {
				if v == name {
					shadowed = true
					break
				}
			}
			if !shadowed && isRecursive(arm.Body, name) {
				return true
			}
		}
		return false
	case *core.BinOp:
		return isRecursive(e.Left, name) || isRecursive(e.Right, name)
	case *core.UnOp:
		return isRecursive(e.Operand, name)
	case *core.RecordAccess:
		return isRecursive(e.Record, name)
	case *core.RecordUpdate:
		if isRecursive(e.Base, name) {
			return true
		}
		for _, val := range e.Updates {
			if isRecursive(val, name) {
				return true
			}
		}
		return false
	case *core.Lit, *core.VarGlobal, *core.Intrinsic:
		return false
	default:
		// For unknown expression types, conservatively assume non-recursive
		return false
	}
}

// patternBoundVars extracts all variables bound by a pattern
func patternBoundVars(pat core.CorePattern) []string {
	switch p := pat.(type) {
	case *core.VarPattern:
		return []string{p.Name}
	case *core.ConstructorPattern:
		var vars []string
		for _, arg := range p.Args {
			vars = append(vars, patternBoundVars(arg)...)
		}
		return vars
	case *core.ListPattern:
		var vars []string
		for _, elem := range p.Elements {
			vars = append(vars, patternBoundVars(elem)...)
		}
		if p.Tail != nil {
			vars = append(vars, patternBoundVars(*p.Tail)...)
		}
		return vars
	case *core.RecordPattern:
		var vars []string
		for _, field := range p.Fields {
			vars = append(vars, patternBoundVars(field)...)
		}
		return vars
	case *core.TuplePattern:
		var vars []string
		for _, elem := range p.Elements {
			vars = append(vars, patternBoundVars(elem)...)
		}
		return vars
	case *core.LitPattern, *core.WildcardPattern:
		return nil
	default:
		return nil
	}
}

// isMutuallyRecursive checks if any binding in a LetRec group is mutually recursive
func isMutuallyRecursive(bindings []core.RecBinding) bool {
	// Build set of all binding names
	names := make(map[string]bool)
	for _, binding := range bindings {
		names[binding.Name] = true
	}

	// Check if any binding references another binding in the group
	for _, binding := range bindings {
		for otherName := range names {
			if otherName != binding.Name && isRecursive(binding.Value, otherName) {
				return true
			}
		}
	}

	return false
}

// copyEnv creates a shallow copy of an environment map
func copyEnv(env map[string]types.Type) map[string]types.Type {
	newEnv := make(map[string]types.Type, len(env))
	for k, v := range env {
		newEnv[k] = v
	}
	return newEnv
}

// copyBindings creates a shallow copy of a bindings map
func copyBindings(bindings map[string]core.CoreExpr) map[string]core.CoreExpr {
	newBindings := make(map[string]core.CoreExpr, len(bindings))
	for k, v := range bindings {
		newBindings[k] = v
	}
	return newBindings
}
