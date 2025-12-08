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
