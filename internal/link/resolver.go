package link

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
)

// CompileUnit represents a compiled module (shared interface to avoid circular dependency)
type CompileUnit interface {
	GetCore() *core.Program
	GetModuleID() string
}

// Resolver implements GlobalResolver for the evaluator
type Resolver struct {
	linker        *ModuleLinker
	memo          map[string]map[string]eval.Value // module -> name -> value
	compiledCode  map[string]CompileUnit           // module -> compiled Core AST
	mu            sync.RWMutex                     // For thread-safe memoization
	builtinLookup func(string) (eval.Value, bool)  // Optional builtin lookup (v0.2.0 hotfix)
}

// NewResolver creates a new resolver backed by a module linker
func NewResolver(linker *ModuleLinker) *Resolver {
	return &Resolver{
		linker:       linker,
		memo:         make(map[string]map[string]eval.Value),
		compiledCode: make(map[string]CompileUnit),
	}
}

// RegisterCompiledModule adds a compiled module for on-demand evaluation
func (r *Resolver) RegisterCompiledModule(moduleID string, unit CompileUnit) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.compiledCode[moduleID] = unit
}

// SetBuiltinLookup sets the builtin lookup function (v0.2.0 hotfix)
// This allows the resolver to access builtin functions from the runtime
func (r *Resolver) SetBuiltinLookup(lookup func(string) (eval.Value, bool)) {
	r.builtinLookup = lookup
}

// ResolveValue resolves a global reference to its value
func (r *Resolver) ResolveValue(ref core.GlobalRef) (eval.Value, error) {
	// Check if this is an $adt factory function
	if ref.Module == "$adt" {
		return r.resolveAdtFactory(ref)
	}

	// Check if this is a builtin reference (v0.2.0 hotfix)
	if ref.Module == "$builtin" || strings.HasPrefix(ref.Name, "_") {
		if r.builtinLookup != nil {
			if val, ok := r.builtinLookup(ref.Name); ok {
				return val, nil
			}
		}
		// Builtin not found - fall through to normal resolution
		// (This allows user code to shadow builtins if needed)
	}

	r.mu.RLock()
	if moduleCache, ok := r.memo[ref.Module]; ok {
		if val, ok := moduleCache[ref.Name]; ok {
			r.mu.RUnlock()
			return val, nil
		}
	}
	r.mu.RUnlock()

	// Need to evaluate the export
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if moduleCache, ok := r.memo[ref.Module]; ok {
		if val, ok := moduleCache[ref.Name]; ok {
			return val, nil
		}
		// Module evaluated but export not found
		return nil, fmt.Errorf("EVA002: export %s not found in module %s", ref.Name, ref.Module)
	}

	// Get the compiled module
	unit, ok := r.compiledCode[ref.Module]
	if !ok {
		return nil, fmt.Errorf("EVA002: module not compiled: %s", ref.Module)
	}

	// Evaluate the module to get all its exports
	coreProgram := unit.GetCore()
	if coreProgram == nil || len(coreProgram.Decls) == 0 {
		return nil, fmt.Errorf("EVA002: module %s has no declarations", ref.Module)
	}

	// Create evaluator with recursive resolver (for transitive dependencies)
	evaluator := eval.NewCoreEvaluator()
	evaluator.SetGlobalResolver(r)

	// Initialize module cache
	if r.memo[ref.Module] == nil {
		r.memo[ref.Module] = make(map[string]eval.Value)
	}

	// Evaluate the module and extract bindings
	// Use the evaluator's EvalLetRecBindings method to properly extract recursive bindings
	for _, decl := range coreProgram.Decls {
		switch d := decl.(type) {
		case *core.LetRec:
			// Use special method to evaluate LetRec and extract bindings
			bindings, err := evaluator.EvalLetRecBindings(d)
			if err != nil {
				return nil, fmt.Errorf("EVA002: failed to evaluate LetRec in module %s: %w", ref.Module, err)
			}
			// Store all bindings in the memo
			for name, val := range bindings {
				r.memo[ref.Module][name] = val
			}

		case *core.Let:
			val, err := evaluator.Eval(d.Value)
			if err != nil {
				return nil, fmt.Errorf("EVA002: failed to evaluate Let in module %s: %w", ref.Module, err)
			}
			r.memo[ref.Module][d.Name] = val
		}
	}

	// Now look up the specific export
	if val, ok := r.memo[ref.Module][ref.Name]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("EVA002: export %s not found in module %s (available: %v)", ref.Name, ref.Module, func() []string {
		var names []string
		for n := range r.memo[ref.Module] {
			names = append(names, n)
		}
		return names
	}())
}

// InvalidateModule clears the cache for a specific module
func (r *Resolver) InvalidateModule(module string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.memo, module)
}

// InvalidateAll clears the entire cache
func (r *Resolver) InvalidateAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memo = make(map[string]map[string]eval.Value)
}

// resolveAdtFactory resolves $adt factory functions to builtin constructor creators
// Factory names follow pattern: make_<TypeName>_<CtorName>
func (r *Resolver) resolveAdtFactory(ref core.GlobalRef) (eval.Value, error) {
	// Parse factory name: "make_Option_Some" → TypeName="Option", CtorName="Some"
	if !strings.HasPrefix(ref.Name, "make_") {
		return nil, fmt.Errorf("IMP011_UNKNOWN_CTOR: invalid $adt factory name: %s", ref.Name)
	}

	parts := strings.SplitN(ref.Name[5:], "_", 2) // Remove "make_" prefix
	if len(parts) != 2 {
		return nil, fmt.Errorf("IMP011_UNKNOWN_CTOR: invalid $adt factory name format: %s", ref.Name)
	}

	typeName := parts[0]
	ctorName := parts[1]

	// Look up constructor arity from $adt interface
	adtIface := r.linker.GetIface("$adt")
	if adtIface == nil {
		return nil, fmt.Errorf("IMP011_UNKNOWN_CTOR: $adt module not registered")
	}

	var arity int
	found := false
	for _, ctor := range adtIface.Constructors {
		if ctor.TypeName == typeName && ctor.CtorName == ctorName {
			arity = ctor.Arity
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("IMP011_UNKNOWN_CTOR: constructor %s.%s not found in $adt module (looking for factory %s)", typeName, ctorName, ref.Name)
	}

	// For nullary constructors, return the TaggedValue directly
	if arity == 0 {
		return &eval.TaggedValue{
			TypeName: typeName,
			CtorName: ctorName,
			Fields:   []eval.Value{},
		}, nil
	}

	// For constructors with fields, return a function
	factoryFn := &eval.BuiltinFunction{
		Name: ref.Name,
		Fn: func(args []eval.Value) (eval.Value, error) {
			// Constructor factory: creates TaggedValue with given fields
			return &eval.TaggedValue{
				TypeName: typeName,
				CtorName: ctorName,
				Fields:   args,
			}, nil
		},
	}

	return factoryFn, nil
}
