package core

// ResolveValue follows ANF variable bindings to find the ultimate bound value.
// This is useful when you need to inspect what a variable is bound to in ANF form.
//
// ANF (A-Normal Form) represents complex expressions as sequences of let-bindings:
//
//	let x = [1, 2, 3] in
//	let y = x in
//	let z = y in
//	z ++ [4, 5]
//
// In this example, ResolveValue(z, bindings) will follow the chain:
//
//	z → y → x → [1, 2, 3]
//
// Parameters:
//   - expr: The Core expression to resolve
//   - bindings: Map of variable names to their bound expressions (ANF-local scope only)
//
// Returns:
//   - The resolved expression (stops when it hits a non-variable, missing binding, or cycle)
//
// Cycle detection: Uses a visited set to prevent infinite loops. If a cycle is detected,
// returns the last resolvable expression (fail-closed).
//
// Note: Prefer using CoreTypeInfo from type inference for type-guided operations.
// ResolveValue is a fallback for non-typed passes and debug utilities.
//
// Example usage:
//
//	bindings := map[string]CoreExpr{
//	    "x": &List{Elements: [...]},
//	    "y": &Var{Name: "x"},
//	    "z": &Var{Name: "y"},
//	}
//	resolved := ResolveValue(&Var{Name: "z"}, bindings)
//	// resolved is &List{Elements: [...]}
func ResolveValue(expr CoreExpr, bindings map[string]CoreExpr) CoreExpr {
	visited := make(map[string]struct{})
	return resolveValueWithVisited(expr, bindings, visited)
}

// resolveValueWithVisited is the internal implementation with cycle detection
func resolveValueWithVisited(expr CoreExpr, bindings map[string]CoreExpr, visited map[string]struct{}) CoreExpr {
	// If it's a variable, try to look up what it's bound to
	if v, ok := expr.(*Var); ok {
		// Check for cycles
		if _, seen := visited[v.Name]; seen {
			// Cycle detected - return current expression (fail-closed)
			return expr
		}

		if binding, exists := bindings[v.Name]; exists {
			// Mark as visited
			visited[v.Name] = struct{}{}

			// Recursively resolve the binding
			return resolveValueWithVisited(binding, bindings, visited)
		}
	}

	// Not a variable, or no binding found - return as-is
	return expr
}

// IsListValue checks if an expression (after resolving variables) is a List literal.
// This is useful for determining the concrete type of a value in ANF.
//
// Note: Prefer using CoreTypeInfo from type inference for type-guided operations.
// IsListValue is a fallback for non-typed passes and debug utilities.
func IsListValue(expr CoreExpr, bindings map[string]CoreExpr) bool {
	resolved := ResolveValue(expr, bindings)
	_, ok := resolved.(*List)
	return ok
}

// IsStringValue checks if an expression (after resolving variables) is a String literal.
func IsStringValue(expr CoreExpr, bindings map[string]CoreExpr) bool {
	resolved := ResolveValue(expr, bindings)
	if lit, ok := resolved.(*Lit); ok {
		return lit.Kind == StringLit
	}
	return false
}

// IsIntValue checks if an expression (after resolving variables) is an Int literal.
func IsIntValue(expr CoreExpr, bindings map[string]CoreExpr) bool {
	resolved := ResolveValue(expr, bindings)
	if lit, ok := resolved.(*Lit); ok {
		return lit.Kind == IntLit
	}
	return false
}

// IsFloatValue checks if an expression (after resolving variables) is a Float literal.
func IsFloatValue(expr CoreExpr, bindings map[string]CoreExpr) bool {
	resolved := ResolveValue(expr, bindings)
	if lit, ok := resolved.(*Lit); ok {
		return lit.Kind == FloatLit
	}
	return false
}

// IsBoolValue checks if an expression (after resolving variables) is a Bool literal.
func IsBoolValue(expr CoreExpr, bindings map[string]CoreExpr) bool {
	resolved := ResolveValue(expr, bindings)
	if lit, ok := resolved.(*Lit); ok {
		return lit.Kind == BoolLit
	}
	return false
}
