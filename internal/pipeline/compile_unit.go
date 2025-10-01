package pipeline

import (
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/iface"
)

// ConstructorInfo holds information about a constructor for interface building
type ConstructorInfo struct {
	TypeName   string // ADT type name (e.g., "Option")
	CtorName   string // Constructor name (e.g., "Some")
	FieldTypes []ast.Type // Field types from AST
	Arity      int    // Number of fields
}

// CompileUnit represents a module compilation unit
type CompileUnit struct {
	ID           string                      // Module ID/path
	Surface      *ast.File                   // Parsed AST
	Core         *core.Program               // Core representation
	Iface        *iface.Iface                // Module interface
	TypeEnv      interface{}                 // Type environment (placeholder)
	Constructors map[string]*ConstructorInfo // ADT constructors defined in this module
}

// GetCore returns the Core AST (implements link.CompileUnit interface)
func (cu *CompileUnit) GetCore() *core.Program {
	return cu.Core
}

// GetModuleID returns the module ID (implements link.CompileUnit interface)
func (cu *CompileUnit) GetModuleID() string {
	return cu.ID
}
