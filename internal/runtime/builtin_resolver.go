package runtime

import (
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
)

// BuiltinOnlyResolver resolves builtin functions for non-module execution
//
// This resolver is used in the legacy pipeline execution path (non-module files)
// to provide access to the builtin registry. It only resolves:
//   - References to the synthetic $builtin module
//   - Names starting with underscore (e.g., _io_print)
//
// It does NOT resolve user bindings or module imports, as those are handled
// by the module runtime's moduleGlobalResolver.
//
// Thread-safety: Safe for concurrent use as it only reads from the registry.
type BuiltinOnlyResolver struct {
	Builtins *BuiltinRegistry
}

// ResolveValue attempts to resolve a global reference to a builtin function
//
// Resolution logic:
//  1. Check if it's a builtin reference (module="$builtin" or name starts with "_")
//  2. Look up the builtin in the registry
//  3. Return nil if not found (allows fallback to other resolution mechanisms)
//
// Parameters:
//   - ref: The global reference to resolve
//
// Returns:
//   - The builtin Value if found
//   - nil, nil if not a builtin (NOT an error - allows chaining resolvers)
//
// Example:
//
//	resolver := &BuiltinOnlyResolver{Builtins: registry}
//	val, err := resolver.ResolveValue(core.GlobalRef{Module: "$builtin", Name: "add_Int"})
func (r *BuiltinOnlyResolver) ResolveValue(ref core.GlobalRef) (eval.Value, error) {
	// Only resolve builtin references
	if ref.Module == "$builtin" || strings.HasPrefix(ref.Name, "_") {
		if val, ok := r.Builtins.Get(ref.Name); ok {
			return val, nil
		}
	}

	// Not a builtin - return nil (not an error)
	// This allows the evaluator to try other resolution mechanisms
	return nil, nil
}

// NewBuiltinOnlyResolver creates a new builtin-only resolver
//
// This is a convenience constructor for creating resolvers with a cleaner API.
//
// Parameters:
//   - builtins: The BuiltinRegistry to use for lookups
//
// Returns:
//   - A new BuiltinOnlyResolver ready to use
func NewBuiltinOnlyResolver(builtins *BuiltinRegistry) *BuiltinOnlyResolver {
	return &BuiltinOnlyResolver{
		Builtins: builtins,
	}
}
