package pipeline

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// ConstructorInfo holds information about a constructor for interface building
type ConstructorInfo struct {
	TypeName           string       // ADT type name (e.g., "Option")
	CtorName           string       // Constructor name (e.g., "Some")
	FieldTypes         []ast.Type   // Field types from AST
	Arity              int          // Number of fields
	TypeParamCount     int          // M-TAPP-FIX: Number of type parameters (e.g., Option[a] = 1)
	TypeParamNames     []string     // M-POLY-ADT: Type parameter names (e.g., ["a"] for Result[a])
	InternalFieldTypes []types.Type // M-POLY-ADT: Actual field types for type scheme building
}

// CompileUnit represents a module compilation unit
type CompileUnit struct {
	ID           string                      // Module ID/path
	Surface      *ast.File                   // Parsed AST
	Core         *core.Program               // Core representation
	CoreTI       types.CoreTypeInfo          // M-DX23: Type info for Core expressions
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
