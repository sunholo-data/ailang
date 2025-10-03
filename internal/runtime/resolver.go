package runtime

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
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
	// Case 0a: ADT constructor factories
	if ref.Module == "$adt" {
		return r.resolveAdtFactory(ref)
	}

	// Case 0b: Builtin reference
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

// resolveAdtFactory resolves $adt factory functions for ADT constructors
//
// This method handles references like "$adt.make_Option_Some" which are generated
// by the elaborator when it encounters constructor usage like `Some(42)`.
//
// The method:
//  1. Parses the factory name to extract TypeName and CtorName
//  2. Searches all imported modules for a matching constructor
//  3. Returns an error if the constructor is ambiguous (found in multiple modules)
//  4. For nullary constructors, returns a cached singleton TaggedValue
//  5. For constructors with fields, returns a factory function that creates TaggedValues
//
// Parameters:
//   - ref: GlobalRef with Module="$adt" and Name="make_TypeName_CtorName"
//
// Returns:
//   - For nullary (arity 0): A cached TaggedValue singleton
//   - For non-nullary: A BuiltinFunction that creates TaggedValues
//   - Error if constructor not found or ambiguous
func (r *moduleGlobalResolver) resolveAdtFactory(ref core.GlobalRef) (eval.Value, error) {
	// Parse "make_Option_Some" → TypeName="Option", CtorName="Some"
	if !strings.HasPrefix(ref.Name, "make_") {
		return nil, fmt.Errorf("invalid $adt factory name: %s", ref.Name)
	}

	parts := strings.SplitN(ref.Name[5:], "_", 2) // Remove "make_" prefix
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid $adt factory format: %s (expected make_Type_Ctor)", ref.Name)
	}

	typeName, ctorName := parts[0], parts[1]

	// Find constructor across current module + all imports
	matches := r.findConstructorMatches(typeName, ctorName)

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("constructor %s.%s not found in scope", typeName, ctorName)
	case 1:
		// OK - unambiguous
	default:
		// AMBIGUOUS - list all candidate modules
		var modulePaths []string
		for _, m := range matches {
			modulePaths = append(modulePaths, m.ModulePath)
		}
		return nil, fmt.Errorf("ambiguous constructor %s.%s found in multiple modules: %v (use qualified import or alias)",
			typeName, ctorName, modulePaths)
	}

	match := matches[0]

	// Nullary constructors: return singleton
	if match.Arity == 0 {
		key := fmt.Sprintf("%s::%s::%s", match.ModulePath, typeName, ctorName)

		if cached, ok := r.runtime.nullaryCache.Load(key); ok {
			return cached.(eval.Value), nil
		}

		singleton := &eval.TaggedValue{
			ModulePath: match.ModulePath,
			TypeName:   typeName,
			CtorName:   ctorName,
			Fields:     nil,
		}
		r.runtime.nullaryCache.Store(key, singleton)
		return singleton, nil
	}

	// Non-nullary: return factory function
	modPath := match.ModulePath // Capture for closure
	expectedArity := match.Arity // Capture arity for closure
	return &eval.BuiltinFunction{
		Name: ref.Name,
		Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != expectedArity {
				return nil, fmt.Errorf("constructor %s.%s expects %d arguments, got %d",
					typeName, ctorName, expectedArity, len(args))
			}
			return &eval.TaggedValue{
				ModulePath: modPath,
				TypeName:   typeName,
				CtorName:   ctorName,
				Fields:     args,
			}, nil
		},
	}, nil
}

// constructorMatch represents a matched constructor with its metadata
type constructorMatch struct {
	ModulePath string
	Arity      int
}

// findConstructorMatches searches for constructors matching the given type and constructor name
//
// This method searches:
//  1. The current module's interface for locally-defined constructors
//  2. All imported modules' interfaces for imported constructors
//
// Parameters:
//   - typeName: The ADT type name (e.g., "Option")
//   - ctorName: The constructor name (e.g., "Some")
//
// Returns:
//   - A list of all matching constructors (may be empty if none found, or >1 if ambiguous)
func (r *moduleGlobalResolver) findConstructorMatches(typeName, ctorName string) []constructorMatch {
	var matches []constructorMatch

	// Helper to scan a module's interface
	scanModule := func(iface *iface.Iface, modulePath string) {
		if iface == nil || iface.Constructors == nil {
			return
		}
		for _, ctor := range iface.Constructors {
			if ctor.TypeName == typeName && ctor.CtorName == ctorName {
				matches = append(matches, constructorMatch{
					ModulePath: modulePath,
					Arity:      ctor.Arity,
				})
			}
		}
	}

	// Check current module's constructors
	scanModule(r.current.Iface, r.current.Path)

	// Check all imported modules
	for modPath, dep := range r.current.Imports {
		scanModule(dep.Iface, modPath)
	}

	return matches
}
