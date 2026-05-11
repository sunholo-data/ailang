package types

// helpers.go provides utility functions for working with type variables.
// These functions abstract the TVar vs TVar2 duality to prevent silent failures.

// ExtractTVarName extracts the name from a type variable.
// Returns (name, true) if the type is TVar or TVar2, ("", false) otherwise.
//
// This helper prevents silent failures when pattern matching on type variables.
// Use this instead of checking *TVar and *TVar2 separately.
//
// Example:
//
//	if name, ok := types.ExtractTVarName(paramType); ok {
//	    // paramType is a type variable with name
//	} else {
//	    // paramType is not a type variable
//	}
func ExtractTVarName(t Type) (string, bool) {
	switch tv := t.(type) {
	case *TVar:
		return tv.Name, true
	case *TVar2:
		return tv.Name, true
	default:
		return "", false
	}
}

// IsTVar checks if a type is a type variable (TVar or TVar2).
// Returns true if the type is TVar or TVar2, false otherwise.
//
// Example:
//
//	if types.IsTVar(paramType) {
//	    // paramType is polymorphic
//	}
func IsTVar(t Type) bool {
	_, ok := ExtractTVarName(t)
	return ok
}

// ExtractTVarKind extracts the kind from a type variable.
// Returns (kind, true) if the type is TVar2, (nil, false) otherwise.
//
// Note: TVar (v1) does not have a Kind field, only TVar2 does.
//
// Example:
//
//	if kind, ok := types.ExtractTVarKind(paramType); ok {
//	    // paramType is TVar2 with kind
//	}
func ExtractTVarKind(t Type) (Kind, bool) {
	if tv2, ok := t.(*TVar2); ok {
		return tv2.Kind, true
	}
	return nil, false
}

// extractADTName returns the constructor name of an ADT type (e.g.
// "Option", "Result"), or "" if the type is not concretely resolved to
// a named ADT.
//
// Handles both unparameterized constructors (`*TCon{Name}`) and applied
// generic ADTs (`*TApp{Constructor: *TCon{Name}, Args: [...]}`).
// Returns "" for type variables (polymorphic / unresolved), function
// types, records, lists, tuples, and primitive types — none of which
// are named ADTs in the constructorTypes map sense.
//
// M-MATCH-ADT-XCHECK (v0.18.10): used by the match-arm cross-check in
// typechecker_patterns.go to decide whether a constructor pattern is
// matching against a concretely-known scrutinee ADT. When this returns
// "", the cross-check defers to the existing constraint-based path so
// polymorphic scrutinees still work.
func extractADTName(t Type) string {
	switch tt := t.(type) {
	case *TCon:
		// Skip primitive types — they're TCon{} but not ADTs in the
		// match-constructor sense. The cross-check only fires for
		// constructor patterns, and primitives can't be matched by
		// constructor patterns anyway, but be defensive.
		switch tt.Name {
		case "int", "float", "string", "bool", "unit":
			return ""
		}
		return tt.Name
	case *TApp:
		h, _ := decomposeApp(tt)
		if con, ok := h.(*TCon); ok {
			// Skip the built-in list constructor — list patterns use
			// ListPattern (a separate case in checkPattern), not
			// ConstructorPattern. If a user wrote `match xs { Some(_) }`
			// against a list[T], we DO want to reject it via the
			// cross-check, so returning "list" here is correct.
			return con.Name
		}
		return ""
	default:
		// TVar (unresolved type var), TFunc, TRecord, TTuple, TList,
		// effect rows, etc. None of these are scrutinizable by
		// ConstructorPattern in a way the cross-check needs to handle.
		return ""
	}
}

// AsList checks if a type is a list type (either TList or TApp("list", ...)).
// Returns the element type and true if it's a list, nil and false otherwise.
//
// This helper addresses DX-17: TList/TApp unification bug. The parser creates
// TList for [T] syntax, while the type builder creates TApp("list", T) for builtins.
// This helper recognizes both representations.
//
// Example:
//
//	if elem, ok := types.AsList(paramType); ok {
//	    // paramType is a list with element type elem
//	}
//
// Note: Case-sensitive - only matches lowercase "list" (not "List").
func AsList(t Type) (elem Type, ok bool) {
	switch tt := t.(type) {
	case *TList:
		return tt.Element, true
	case *TApp:
		h, args := decomposeApp(tt)
		if con, ok := h.(*TCon); ok && con.Name == "list" && len(args) == 1 {
			return args[0], true
		}
		return nil, false
	default:
		return nil, false
	}
}
