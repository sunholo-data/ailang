package types

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
)

// TypeInfo maps AST expression nodes to their inferred types (principal types after generalization)
// This uses pointer identity as the key, which is stable across compilation passes
type TypeInfo map[ast.Expr]Type

// Must returns the type for the given expression, or panics with a helpful error if not found
func (ti TypeInfo) Must(expr ast.Expr) Type {
	if t, ok := ti[expr]; ok {
		return t
	}
	panic(fmt.Sprintf("TypeInfo.Must: no type found for expression %T at %v\nThis is likely a compiler bug. The typechecker should have populated TypeInfo for all expressions.", expr, expr.Position()))
}

// Get returns the type for the given expression and a boolean indicating if it was found
func (ti TypeInfo) Get(expr ast.Expr) (Type, bool) {
	t, ok := ti[expr]
	return t, ok
}

// Set stores a type for the given expression
func (ti TypeInfo) Set(expr ast.Expr, t Type) {
	ti[expr] = t
}

// Has checks if a type exists for the given expression
func (ti TypeInfo) Has(expr ast.Expr) bool {
	_, ok := ti[expr]
	return ok
}

// NewTypeInfo creates a new TypeInfo map
func NewTypeInfo() TypeInfo {
	return make(TypeInfo)
}

// CoreTypeInfo maps Core expression NodeIDs to their inferred types (principal types after generalization)
// This is populated during Core type checking and used during operator lowering
type CoreTypeInfo map[uint64]Type

// Must returns the type for the given Core NodeID, or panics with a helpful error if not found
func (cti CoreTypeInfo) Must(nodeID uint64) Type {
	if t, ok := cti[nodeID]; ok {
		return t
	}
	panic(fmt.Sprintf("CoreTypeInfo.Must: no type found for Core NodeID %d\nThis is likely a compiler bug. The typechecker should have populated CoreTypeInfo for all Core expressions.", nodeID))
}

// Get returns the type for the given Core NodeID and a boolean indicating if it was found
func (cti CoreTypeInfo) Get(nodeID uint64) (Type, bool) {
	t, ok := cti[nodeID]
	return t, ok
}

// Set stores a type for the given Core NodeID
func (cti CoreTypeInfo) Set(nodeID uint64, t Type) {
	cti[nodeID] = t
}

// Has checks if a type exists for the given Core NodeID
func (cti CoreTypeInfo) Has(nodeID uint64) bool {
	_, ok := cti[nodeID]
	return ok
}

// GetForExpr is a convenience method to get the type for a Core expression
func (cti CoreTypeInfo) GetForExpr(expr core.CoreExpr) (Type, bool) {
	if expr == nil {
		return nil, false
	}
	return cti.Get(expr.ID())
}

// MustForExpr is a convenience method that panics if the type is not found
func (cti CoreTypeInfo) MustForExpr(expr core.CoreExpr) Type {
	if expr == nil {
		panic("CoreTypeInfo.MustForExpr: nil expression")
	}
	return cti.Must(expr.ID())
}

// NewCoreTypeInfo creates a new CoreTypeInfo map
func NewCoreTypeInfo() CoreTypeInfo {
	return make(CoreTypeInfo)
}

// ApplySubstitution applies a type substitution to all entries in CoreTypeInfo
// This is critical for fixing the CoreTI population gaps bug (M-DX4):
// After defaulting resolves type variables (e.g., t1 → Float), we must update
// CoreTypeInfo with the concrete types, not leave them as type variables.
//
// Note: Substitutions may form chains (e.g., α37 → α38 → float).
// We repeatedly apply the substitution until we reach a fixed point to fully resolve chains.
func (cti CoreTypeInfo) ApplySubstitution(sub Substitution) {
	for nodeID, typ := range cti {
		// Apply substitution repeatedly until no more changes (resolve chains)
		prev := typ
		for {
			next := ApplySubstitution(sub, prev)
			// Check if we reached a fixed point
			if typesIdentical(next, prev) {
				break
			}
			prev = next
		}
		cti[nodeID] = prev
	}
}

// typesIdentical checks if two types are identical (same structure, same names)
// Used to detect fixed points in substitution application
func typesIdentical(t1, t2 Type) bool {
	// Quick check: same pointer
	if t1 == t2 {
		return true
	}

	// Check by type
	switch v1 := t1.(type) {
	case *TVar2:
		if v2, ok := t2.(*TVar2); ok {
			return v1.Name == v2.Name
		}
	case *TCon:
		if v2, ok := t2.(*TCon); ok {
			return v1.Name == v2.Name
		}
	// For other types, use string comparison as fallback
	default:
		return t1.String() == t2.String()
	}
	return false
}
