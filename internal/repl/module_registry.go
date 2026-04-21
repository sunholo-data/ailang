package repl

import (
	"fmt"
	"sync"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/types"
)

// ModuleRegistry stores compiled modules for REPL access.
// It enables loading AILANG modules in the browser via ailangLoadModule()
// and makes their exports available for subsequent ailangEval() calls.
type ModuleRegistry struct {
	mu           sync.RWMutex
	modules      map[string]*RegisteredModule
	constructors map[string]*CachedConstructor // Cached constructors from all loaded modules
	effContext   *effects.EffContext           // Shared effect context for InvokeExport (set by WASM)
}

// CachedConstructor stores constructor info for cross-module import resolution
type CachedConstructor struct {
	TypeName       string
	CtorName       string
	Arity          int
	TypeParamCount int
	TypeParamNames []string
	FieldTypes     []types.Type
}

// RegistryResolver resolves global references for module evaluation in WASM.
// It resolves:
// 1. Builtin functions ($builtin module or names starting with _)
// 2. ADT constructor factories ($adt module)
// 3. Exports from previously loaded modules
type RegistryResolver struct {
	registry *ModuleRegistry
	builtins *runtime.BuiltinRegistry
}

// NewRegistryResolver creates a resolver that can access the module registry
func NewRegistryResolver(registry *ModuleRegistry, builtins *runtime.BuiltinRegistry) *RegistryResolver {
	return &RegistryResolver{
		registry: registry,
		builtins: builtins,
	}
}

// ResolveValue resolves a global reference to a runtime value
func (r *RegistryResolver) ResolveValue(ref core.GlobalRef) (eval.Value, error) {
	// Debug: Enable to trace resolutions
	debug := false // Set to true for debugging

	// Case 1: Builtin reference
	if ref.Module == "$builtin" || (len(ref.Name) > 0 && ref.Name[0] == '_') {
		if val, ok := r.builtins.Get(ref.Name); ok {
			if debug {
				fmt.Printf("RegistryResolver: %s.%s -> builtin %T\n", ref.Module, ref.Name, val)
			}
			return val, nil
		}
		// Fall through - might be a module function starting with _
	}

	// Case 2: ADT constructor factory
	if ref.Module == "$adt" {
		return r.resolveAdtFactory(ref)
	}

	// Case 3: Import from another module
	if ref.Module != "" && ref.Module != "$builtin" && ref.Module != "$adt" {
		r.registry.mu.RLock()
		mod, ok := r.registry.modules[ref.Module]
		r.registry.mu.RUnlock()
		if ok {
			if export, exists := mod.Exports[ref.Name]; exists {
				if export.Value != nil {
					if debug {
						fmt.Printf("RegistryResolver: %s.%s -> export %T\n", ref.Module, ref.Name, export.Value)
					}
					return export.Value.(eval.Value), nil
				}
			}
		}
		if debug {
			fmt.Printf("RegistryResolver: %s.%s -> NOT FOUND (module loaded: %v)\n", ref.Module, ref.Name, ok)
		}
	}

	// Not found
	if debug {
		fmt.Printf("RegistryResolver: %s.%s -> nil (no match)\n", ref.Module, ref.Name)
	}
	return nil, nil
}

// resolveAdtFactory resolves $adt.make_Type_Ctor constructor factories
func (r *RegistryResolver) resolveAdtFactory(ref core.GlobalRef) (eval.Value, error) {
	// Parse "make_Option_Some" -> TypeName="Option", CtorName="Some"
	name := ref.Name
	if len(name) < 6 || name[:5] != "make_" {
		return nil, fmt.Errorf("invalid $adt factory name: %s", name)
	}

	rest := name[5:]
	underscoreIdx := -1
	for i := 0; i < len(rest); i++ {
		if rest[i] == '_' {
			underscoreIdx = i
			break
		}
	}
	if underscoreIdx < 0 {
		return nil, fmt.Errorf("invalid $adt factory format: %s", name)
	}

	typeName := rest[:underscoreIdx]
	ctorName := rest[underscoreIdx+1:]

	// Look up constructor in registry cache
	r.registry.mu.RLock()
	ctor, ok := r.registry.constructors[ctorName]
	r.registry.mu.RUnlock()

	if !ok || ctor.TypeName != typeName {
		return nil, fmt.Errorf("constructor %s.%s not found", typeName, ctorName)
	}

	// Nullary constructor: return singleton TaggedValue
	if ctor.Arity == 0 {
		return &eval.TaggedValue{
			TypeName: typeName,
			CtorName: ctorName,
			Fields:   nil,
		}, nil
	}

	// Non-nullary: return factory function
	expectedArity := ctor.Arity
	return &eval.BuiltinFunction{
		Name: ref.Name,
		Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != expectedArity {
				return nil, fmt.Errorf("constructor %s.%s expects %d arguments, got %d",
					typeName, ctorName, expectedArity, len(args))
			}
			return &eval.TaggedValue{
				TypeName: typeName,
				CtorName: ctorName,
				Fields:   args,
			}, nil
		},
	}, nil
}

// RegisteredModule represents a compiled and evaluated AILANG module.
type RegisteredModule struct {
	Name    string             // Module name (e.g., "invoice_processor")
	Source  string             // Original source code
	Exports map[string]*Export // Exported functions and values
}

// Export represents a single exported function or value from a module.
type Export struct {
	Name   string        // Export name
	Value  interface{}   // Evaluated value (closure, constant, etc.)
	Scheme *types.Scheme // Type signature for type checking
}

// NewModuleRegistry creates an empty module registry.
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules:      make(map[string]*RegisteredModule),
		constructors: make(map[string]*CachedConstructor),
	}
}

// SetEffContext sets the shared effect context used by InvokeExport.
// This allows WASM-configured effect handlers (AI, IO, etc.) to be available
// when calling module exports.
func (mr *ModuleRegistry) SetEffContext(ctx *effects.EffContext) {
	mr.effContext = ctx
}

// GetExport retrieves a specific export from a loaded module.
func (mr *ModuleRegistry) GetExport(moduleName, funcName string) (*Export, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	mod, ok := mr.modules[moduleName]
	if !ok {
		return nil, fmt.Errorf("module %s not loaded (use ailangLoadModule first)", moduleName)
	}

	exp, ok := mod.Exports[funcName]
	if !ok {
		return nil, fmt.Errorf("symbol %s not exported by %s", funcName, moduleName)
	}

	return exp, nil
}

// GetModule retrieves a loaded module by name.
func (mr *ModuleRegistry) GetModule(name string) (*RegisteredModule, bool) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	mod, ok := mr.modules[name]
	return mod, ok
}

// ListModules returns the names of all loaded modules.
func (mr *ModuleRegistry) ListModules() []string {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	names := make([]string, 0, len(mr.modules))
	for name := range mr.modules {
		names = append(names, name)
	}
	return names
}

// formatArgument converts an eval.Value to its AILANG source representation
func formatArgument(v eval.Value) string {
	switch val := v.(type) {
	case *eval.IntValue:
		return fmt.Sprintf("%d", val.Value)
	case *eval.FloatValue:
		return fmt.Sprintf("%g", val.Value)
	case *eval.StringValue:
		return fmt.Sprintf("%q", val.Value)
	case *eval.BoolValue:
		if val.Value {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
