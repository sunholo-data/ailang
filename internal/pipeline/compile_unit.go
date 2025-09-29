package pipeline

import (
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/iface"
)

// CompileUnit represents a module compilation unit
type CompileUnit struct {
	ID       string        // Module ID/path
	Surface  *ast.File     // Parsed AST
	Core     *core.Program // Core representation
	Iface    *iface.Iface  // Module interface
	TypeEnv  interface{}   // Type environment (placeholder)
}

// GetCore returns the Core AST (implements link.CompileUnit interface)
func (cu *CompileUnit) GetCore() *core.Program {
	return cu.Core
}

// GetModuleID returns the module ID (implements link.CompileUnit interface)
func (cu *CompileUnit) GetModuleID() string {
	return cu.ID
}