package runtime

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
)

// moduleGlobalResolver resolves global references for module evaluation
//
// The resolver implements the eval.GlobalResolver interface and is used
// during module evaluation to resolve references to:
//   - Builtin functions (names starting with underscore)
//   - Local bindings within the current module
//   - Exported bindings from imported modules
//
// Encapsulation: The resolver only accesses exports from imported modules,
// ensuring that private bindings remain private.
//
// Thread-safety: The resolver is created per evaluation and does not
// maintain mutable state, so it is safe to use concurrently.
type moduleGlobalResolver struct {
	current *ModuleInstance // The module being evaluated
	runtime *ModuleRuntime  // For accessing builtins registry
}

// ResolveValue resolves a global reference to a runtime value
//
// Resolution logic:
//  1. Check if it's a builtin reference (module="$builtin" or name starts with "_")
//  2. If the reference has no module path (or matches current module),
//     resolve from current module's bindings (both exported and private)
//  3. If the reference has a module path, resolve from that module's
//     exports only (enforcing encapsulation)
//
// Parameters:
//   - ref: The global reference to resolve (module path + name)
//
// Returns:
//   - The resolved Value if found
//   - An error if the reference cannot be resolved
//
// Example:
//
//	// Resolve builtin
//	val, err := resolver.ResolveValue(core.GlobalRef{Module: "$builtin", Name: "_io_print"})
//
//	// Resolve local binding
//	val, err := resolver.ResolveValue(core.GlobalRef{Module: "", Name: "helper"})
//
//	// Resolve imported binding
//	val, err := resolver.ResolveValue(core.GlobalRef{Module: "std/io", Name: "println"})
func (r *moduleGlobalResolver) ResolveValue(ref core.GlobalRef) (eval.Value, error) {
	// Case 0: Builtin reference
	if ref.Module == "$builtin" || strings.HasPrefix(ref.Name, "_") {
		if val, ok := r.runtime.builtins.Get(ref.Name); ok {
			return val, nil
		}
		// Fall through to try local/imported lookup
	}

	// Case 1: Reference to current module (or unqualified reference)
	if ref.Module == "" || ref.Module == r.current.Path {
		val, ok := r.current.Bindings[ref.Name]
		if !ok {
			// Build list of available bindings for error message
			available := make([]string, 0, len(r.current.Bindings))
			for name := range r.current.Bindings {
				available = append(available, name)
			}

			if len(available) == 0 {
				return nil, fmt.Errorf("undefined binding '%s' in module %s (module has no bindings)",
					ref.Name, r.current.Path)
			}

			return nil, fmt.Errorf("undefined binding '%s' in module %s (available: %v)",
				ref.Name, r.current.Path, available)
		}
		return val, nil
	}

	// Case 2: Reference to imported module (exports only)
	dep, ok := r.current.Imports[ref.Module]
	if !ok {
		// Build list of available imports for error message
		available := make([]string, 0, len(r.current.Imports))
		for modPath := range r.current.Imports {
			available = append(available, modPath)
		}

		if len(available) == 0 {
			return nil, fmt.Errorf("module %s not imported by %s (module has no imports)",
				ref.Module, r.current.Path)
		}

		return nil, fmt.Errorf("module %s not imported by %s (available imports: %v)",
			ref.Module, r.current.Path, available)
	}

	// Get exported value from imported module (enforces encapsulation)
	val, err := dep.GetExport(ref.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s from module %s: %w",
			ref.Name, ref.Module, err)
	}

	return val, nil
}

// newModuleGlobalResolver creates a new global resolver for a module
//
// This is a helper function to create resolvers with a cleaner API.
//
// Parameters:
//   - inst: The ModuleInstance to create a resolver for
//   - rt: The ModuleRuntime (for accessing builtins)
//
// Returns:
//   - A new moduleGlobalResolver ready to use
func newModuleGlobalResolver(inst *ModuleInstance, rt *ModuleRuntime) eval.GlobalResolver {
	return &moduleGlobalResolver{
		current: inst,
		runtime: rt,
	}
}
