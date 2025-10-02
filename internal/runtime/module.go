package runtime

import (
	"fmt"
	"sync"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/loader"
)

// ModuleInstance represents a runtime module with evaluated bindings
//
// A ModuleInstance is created from a LoadedModule after type-checking and
// contains the runtime state of the module, including:
//   - All top-level bindings (both exported and private)
//   - Exported bindings only (for cross-module access)
//   - Links to imported module instances
//
// Thread-safety: Initialization is protected by sync.Once to ensure
// each module is evaluated exactly once, even with concurrent access.
type ModuleInstance struct {
	// Identity
	Path string // Module path (e.g., "stdlib/std/io")

	// Static Information (from type-checking)
	Iface *iface.Iface  // Module interface (exports, types)
	Core  *core.Program // Compiled Core AST

	// Runtime State
	Bindings map[string]eval.Value      // All top-level bindings
	Exports  map[string]eval.Value      // Exported bindings only
	Imports  map[string]*ModuleInstance // Imported modules

	// Evaluation State (thread-safe initialization)
	initOnce sync.Once // Ensures single evaluation
	initErr  error     // Evaluation error (if any)
}

// NewModuleInstance creates a new module instance from a loaded module
//
// The instance is created with empty Bindings and Exports maps, which will
// be populated during evaluation. The Iface and Core are copied from the
// LoadedModule.
//
// Parameters:
//   - loaded: A LoadedModule containing the parsed, type-checked module
//
// Returns:
//   - A new ModuleInstance ready for evaluation
func NewModuleInstance(loaded *loader.LoadedModule) *ModuleInstance {
	return &ModuleInstance{
		Path:     loaded.Path,
		Iface:    loaded.Iface,
		Core:     loaded.Core,
		Bindings: make(map[string]eval.Value),
		Exports:  make(map[string]eval.Value),
		Imports:  make(map[string]*ModuleInstance),
	}
}

// GetExport retrieves an exported value by name
//
// This method is used for cross-module access to ensure encapsulation:
// only exported bindings are accessible from other modules.
//
// Parameters:
//   - name: The name of the exported binding
//
// Returns:
//   - The exported value if found
//   - An error if the export does not exist
//
// Example:
//
//	val, err := moduleInst.GetExport("main")
//	if err != nil {
//	    // Export "main" not found
//	}
func (mi *ModuleInstance) GetExport(name string) (eval.Value, error) {
	val, ok := mi.Exports[name]
	if !ok {
		// Build list of available exports for error message
		available := make([]string, 0, len(mi.Exports))
		for exportName := range mi.Exports {
			available = append(available, exportName)
		}

		if len(available) == 0 {
			return nil, fmt.Errorf("module %s has no exports", mi.Path)
		}

		return nil, fmt.Errorf("export %s not found in module %s (available: %v)", name, mi.Path, available)
	}

	return val, nil
}

// HasExport checks if a module exports a given name
//
// Parameters:
//   - name: The name to check
//
// Returns:
//   - true if the module exports the name, false otherwise
func (mi *ModuleInstance) HasExport(name string) bool {
	_, ok := mi.Exports[name]
	return ok
}

// GetBinding retrieves a binding by name (exported or private)
//
// This method is used for internal module access during evaluation.
// Unlike GetExport, it can access private (non-exported) bindings.
//
// Parameters:
//   - name: The name of the binding
//
// Returns:
//   - The binding value if found
//   - An error if the binding does not exist
func (mi *ModuleInstance) GetBinding(name string) (eval.Value, error) {
	val, ok := mi.Bindings[name]
	if !ok {
		return nil, fmt.Errorf("undefined binding '%s' in module %s", name, mi.Path)
	}

	return val, nil
}

// ListExports returns a sorted list of export names
//
// This is useful for error messages and debugging.
//
// Returns:
//   - A slice of export names in the order they were added
func (mi *ModuleInstance) ListExports() []string {
	exports := make([]string, 0, len(mi.Exports))
	for name := range mi.Exports {
		exports = append(exports, name)
	}
	return exports
}

// IsEvaluated returns whether the module has been evaluated
//
// Returns:
//   - true if evaluation completed (successfully or with error)
//   - false if evaluation has not been attempted
func (mi *ModuleInstance) IsEvaluated() bool {
	// We can't directly check sync.Once state, but we can check if
	// initErr has been set or if Bindings is populated
	return len(mi.Bindings) > 0 || mi.initErr != nil
}

// GetEvaluationError returns the error from evaluation, if any
//
// Returns:
//   - The evaluation error if evaluation failed
//   - nil if evaluation succeeded or hasn't been attempted
func (mi *ModuleInstance) GetEvaluationError() error {
	return mi.initErr
}
