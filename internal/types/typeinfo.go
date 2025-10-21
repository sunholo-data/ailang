package types

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
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
