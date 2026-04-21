package runtime

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/types"
)

// ModuleRuntime manages module instances and orchestrates evaluation
//
// The ModuleRuntime is responsible for:
//   - Loading modules via the ModuleLoader
//   - Creating ModuleInstance objects
//   - Caching evaluated modules
//   - Evaluating modules in dependency order (topological sort)
//   - Linking imported modules
//   - Detecting circular imports
//   - Providing builtin function registry
//
// Thread-safety: The runtime uses sync.Once within each ModuleInstance
// to ensure each module is evaluated exactly once.
type ModuleRuntime struct {
	loader       *loader.ModuleLoader       // For loading and type-checking modules
	evaluator    *eval.CoreEvaluator        // For evaluating Core AST
	builtins     *BuiltinRegistry           // Registry of builtin functions
	instances    map[string]*ModuleInstance // Cache: path → instance
	basePath     string                     // Base path for resolving modules
	visiting     map[string]bool            // Track modules being visited (for cycle detection)
	pathStack    []string                   // Current DFS path (for cycle error messages)
	nullaryCache sync.Map                   // Cache for nullary constructors (None, True, False) - key: "modPath::Type::Ctor"
}

// NewModuleRuntime creates a new module runtime
//
// The runtime is initialized with a base path for module resolution and
// creates a fresh module loader and core evaluator.
//
// Parameters:
//   - basePath: The directory to use as the root for module resolution
//
// Returns:
//   - A new ModuleRuntime ready to load and evaluate modules
//
// Example:
//
//	rt := NewModuleRuntime("/path/to/project")
//	inst, err := rt.LoadAndEvaluate("examples/demo")
func NewModuleRuntime(basePath string) *ModuleRuntime {
	// Clean the base path
	cleanPath := filepath.Clean(basePath)

	// Create evaluator first
	evaluator := eval.NewCoreEvaluator()

	// Create builtins registry with evaluator reference
	builtins := NewBuiltinRegistry(evaluator)

	return &ModuleRuntime{
		loader:    loader.NewModuleLoader(cleanPath),
		evaluator: evaluator,
		builtins:  builtins,
		instances: make(map[string]*ModuleInstance),
		basePath:  cleanPath,
		visiting:  make(map[string]bool),
		pathStack: make([]string, 0),
	}
}

// PreloadModule adds a pre-loaded module to the loader's cache
//
// This is used to inject modules that were already loaded and elaborated
// by the pipeline, avoiding redundant loading and elaboration.
//
// Parameters:
//   - path: The module path
//   - loaded: The LoadedModule with Core AST already populated
func (rt *ModuleRuntime) PreloadModule(path string, loaded *loader.LoadedModule) {
	rt.loader.Preload(path, loaded)
}

// LoadAndEvaluate loads a module and all its dependencies, then evaluates them
//
// This is the main entry point for module execution. It performs the following steps:
//  1. Check cache for already-evaluated modules
//  2. Load the module (parse, type-check, build interface)
//  3. Create a ModuleInstance
//  4. Recursively load and evaluate dependencies (topological sort)
//  5. Evaluate this module (populate bindings and exports)
//
// The evaluation order is deterministic: dependencies are evaluated before
// dependents (depth-first search).
//
// Parameters:
//   - modulePath: The module path to load (e.g., "stdlib/std/io")
//
// Returns:
//   - The evaluated ModuleInstance
//   - An error if loading or evaluation fails
//
// Example:
//
//	inst, err := rt.LoadAndEvaluate("examples/hello")
//	if err != nil {
//	    log.Fatal("Failed to evaluate module:", err)
//	}
//	main, _ := inst.GetExport("main")
func (rt *ModuleRuntime) LoadAndEvaluate(modulePath string) (*ModuleInstance, error) {
	// 1. Check cache (fast path)
	if inst, ok := rt.instances[modulePath]; ok {
		// Return cached instance if already evaluated
		if inst.IsEvaluated() {
			return inst, inst.GetEvaluationError()
		}
	}

	// 2. Check for circular imports
	if rt.visiting[modulePath] {
		// Build cycle path for error message
		cyclePath := make([]string, 0, len(rt.pathStack)+1)
		foundStart := false
		for _, p := range rt.pathStack {
			if p == modulePath {
				foundStart = true
			}
			if foundStart {
				cyclePath = append(cyclePath, p)
			}
		}
		cyclePath = append(cyclePath, modulePath)

		// Format: A → B → C → A
		cycleStr := ""
		for i, p := range cyclePath {
			if i > 0 {
				cycleStr += " → "
			}
			cycleStr += p
		}

		return nil, fmt.Errorf("circular import detected: %s", cycleStr)
	}

	// Mark as visiting and add to path stack
	rt.visiting[modulePath] = true
	rt.pathStack = append(rt.pathStack, modulePath)

	// Ensure cleanup on exit
	defer func() {
		rt.visiting[modulePath] = false
		rt.pathStack = rt.pathStack[:len(rt.pathStack)-1]
	}()

	// 3. Load module (parse, type-check, build interface)
	loaded, err := rt.loader.Load(modulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load module %s: %w", modulePath, err)
	}

	// 3b. Elaborate AST to Core if not already done
	// (Pipeline preloads modules with Core already populated)
	if loaded.Core == nil && loaded.File != nil {
		elaborator := elaborate.NewElaboratorWithPath(loaded.Path)
		elaborator.SetModuleLoader(rt.loader) // Share the runtime's loader for consistent module resolution
		elaborator.AddBuiltinsToGlobalEnv()
		coreProgram, err := elaborator.ElaborateFile(loaded.File)
		if err != nil {
			return nil, fmt.Errorf("failed to elaborate module %s: %w", modulePath, err)
		}
		loaded.Core = coreProgram
	}

	// 3c. Build minimal interface if not already done
	// (Pipeline provides complete type-checked interface)
	if loaded.Iface == nil {
		loaded.Iface = rt.buildMinimalInterface(loaded)
	}

	// 4. Create module instance
	inst := NewModuleInstance(loaded)
	rt.instances[modulePath] = inst

	// 5. Recursively load and evaluate dependencies (topological sort)
	for _, importPath := range loaded.Imports {
		depInst, err := rt.LoadAndEvaluate(importPath)
		if err != nil {
			inst.initErr = fmt.Errorf("failed to load dependency %s: %w", importPath, err)
			return nil, inst.initErr
		}
		inst.Imports[importPath] = depInst
	}

	// 6. Evaluate this module (thread-safe via sync.Once)
	inst.initOnce.Do(func() {
		inst.initErr = rt.evaluateModule(inst)
	})

	return inst, inst.initErr
}

// evaluateModule evaluates a module's Core AST to populate bindings
//
// This method is called exactly once per module (protected by sync.Once).
// It performs the following steps:
//  1. Set up a GlobalResolver for cross-module references
//  2. Iterate over top-level declarations in the Core AST
//  3. Evaluate each declaration (currently only LetRec is supported)
//  4. Populate the Bindings map
//  5. Filter Exports based on the module interface
//
// Parameters:
//   - inst: The ModuleInstance to evaluate
//
// Returns:
//   - nil if evaluation succeeds
//   - An error if evaluation fails
//
// Note: This method is not exported because it should only be called
// internally by LoadAndEvaluate.
func (rt *ModuleRuntime) evaluateModule(inst *ModuleInstance) error {
	// 1. Set up global resolver for cross-module references
	resolver := newModuleGlobalResolver(inst, rt)
	rt.evaluator.SetGlobalResolver(resolver)

	// M-CAPABILITY-BUDGETS: Set CoreTypeInfo for effect budget enforcement
	if inst.CoreTI != nil {
		if cti, ok := inst.CoreTI.(types.CoreTypeInfo); ok {
			rt.evaluator.SetCoreTypeInfo(cti)
		}
	}

	// 2. Iterate over top-level declarations in the Core AST
	if inst.Core == nil {
		return fmt.Errorf("module %s has no Core AST (loader issue)", inst.Path)
	}
	if len(inst.Core.Decls) == 0 {
		// Empty module is valid only if there are no exports
		if len(inst.Iface.Exports) > 0 {
			return fmt.Errorf("module %s has %d exports but no Core declarations", inst.Path, len(inst.Iface.Exports))
		}
		return nil
	}

	// M-MODULE-SCOPE: Isolate each module's internal bindings in a child environment.
	// Without this, non-exported functions with the same name in different modules
	// collide in the shared flat namespace (e.g., docx_parser.joinParagraphTexts
	// gets overwritten by pptx_parser.joinParagraphTexts).
	//
	// How it works:
	// 1. Create a child env for this module's bindings (both internal + exported)
	// 2. Closures capture this child env via evalCoreLambda — they reference
	//    their own module's internal functions correctly
	// 3. After evaluation, promote EXPORTS to the parent env so subsequent
	//    modules can resolve imported names via Var lookup
	// 4. Internal (non-exported) bindings stay in the child env only
	//
	// Builtins remain accessible via the parent chain (child → parent).
	parentEnv := rt.evaluator.Env()
	moduleEnv := parentEnv.NewChildEnvironment()
	rt.evaluator.SetEnv(moduleEnv)
	defer rt.evaluator.SetEnv(parentEnv)

	// Process declarations - recursively extract nested Let bindings
	for _, decl := range inst.Core.Decls {
		err := rt.extractBindings(inst, decl)
		if err != nil {
			return err
		}
	}

	// 5. Filter Exports based on the module interface
	if inst.Iface != nil && inst.Iface.Exports != nil {
		for exportName := range inst.Iface.Exports {
			// Check if the binding exists
			val, ok := inst.Bindings[exportName]
			if !ok {
				return fmt.Errorf("exported binding '%s' not found in module %s bindings", exportName, inst.Path)
			}

			// Add to exports map
			inst.Exports[exportName] = val

			// M-MODULE-SCOPE: Promote exports to parent env so that importing
			// modules can resolve them via Var lookup (the elaborator may produce
			// Var references for imported names, not just VarGlobal).
			parentEnv.Set(exportName, val)
		}
	}

	return nil
}

// extractBindings recursively extracts Let and LetRec bindings from nested expressions
//
// Module elaboration produces nested Let expressions like:
//
//	let f1 = ... in (let f2 = ... in Var(...))
//
// This function recursively traverses the structure to extract all bindings.
//
// Parameters:
//   - inst: The module instance to populate
//   - expr: The expression to extract bindings from
//
// Returns:
//   - An error if binding evaluation fails
func (rt *ModuleRuntime) extractBindings(inst *ModuleInstance, expr core.CoreExpr) error {
	switch e := expr.(type) {
	case *core.LetRec:
		// Evaluate let rec bindings
		bindings, err := rt.evaluator.EvalLetRecBindings(e)
		if err != nil {
			return fmt.Errorf("failed to evaluate let rec in module %s: %w", inst.Path, err)
		}

		// Store bindings and add to evaluator environment
		for name, val := range bindings {
			// M-VERIFY-CONTRACTS: Attach contracts to FunctionValues
			if fn, ok := val.(*eval.FunctionValue); ok && inst.Core != nil && inst.Core.Meta != nil {
				if meta, ok := inst.Core.Meta[name]; ok && len(meta.Contracts) > 0 {
					attachContracts(fn, meta.Contracts)
				}
			}
			inst.Bindings[name] = val
			// Add to environment so subsequent bindings can reference these
			rt.evaluator.Env().Set(name, val)
		}

		// Recursively process body if it exists
		if e.Body != nil {
			return rt.extractBindings(inst, e.Body)
		}

	case *core.Let:
		// Evaluate let binding
		val, err := rt.evaluator.Eval(e.Value)
		if err != nil {
			return fmt.Errorf("failed to evaluate let %s in module %s: %w", e.Name, inst.Path, err)
		}

		// M-VERIFY-CONTRACTS: Attach contracts to FunctionValues
		if fn, ok := val.(*eval.FunctionValue); ok && inst.Core != nil && inst.Core.Meta != nil {
			if meta, ok := inst.Core.Meta[e.Name]; ok && len(meta.Contracts) > 0 {
				attachContracts(fn, meta.Contracts)
			}
		}

		inst.Bindings[e.Name] = val

		// CRITICAL: Add binding to evaluator's environment so subsequent bindings can reference it
		// This allows functions within the same module to call each other
		rt.evaluator.Env().Set(e.Name, val)

		// Recursively process body if it exists
		if e.Body != nil {
			return rt.extractBindings(inst, e.Body)
		}

	case *core.Var:
		// Var at module level is typically the final "result" expression
		// For modules, we ignore this - we only care about bindings
		return nil

	case *core.Lit:
		// Literal at module level (used in stdlib equation-form exports)
		// These don't create bindings, just represent constant values
		// We ignore them during module evaluation
		return nil

	default:
		// Other expression types are not expected at module level
		return fmt.Errorf("unexpected module-level expression type %T in module %s", e, inst.Path)
	}

	return nil
}

// GetInstance retrieves a module instance from the cache
//
// This is useful for debugging and testing.
//
// Parameters:
//   - modulePath: The module path to look up
//
// Returns:
//   - The cached ModuleInstance if found
//   - nil if not found
//
// # GetEvaluator returns the runtime's evaluator
//
// This allows external code to access the evaluator for setting
// the effect context or other configuration.
//
// Returns:
//   - The CoreEvaluator used by this runtime
func (rt *ModuleRuntime) GetEvaluator() *eval.CoreEvaluator {
	return rt.evaluator
}

func (rt *ModuleRuntime) GetInstance(modulePath string) *ModuleInstance {
	return rt.instances[modulePath]
}

// HasInstance checks if a module instance is cached
//
// Parameters:
//   - modulePath: The module path to check
//
// Returns:
//   - true if the module is cached, false otherwise
func (rt *ModuleRuntime) HasInstance(modulePath string) bool {
	_, ok := rt.instances[modulePath]
	return ok
}

// DeleteInstance removes a cached module instance, forcing re-evaluation on next load.
// This is used by hot reload to invalidate stale modules.
func (rt *ModuleRuntime) DeleteInstance(modulePath string) {
	delete(rt.instances, modulePath)
}

// GetLoader returns the module loader for cache management.
func (rt *ModuleRuntime) GetLoader() *loader.ModuleLoader {
	return rt.loader
}

// ListInstances returns a list of all cached module paths
//
// This is useful for debugging and testing.
//
// Returns:
//   - A slice of module paths in the cache
func (rt *ModuleRuntime) ListInstances() []string {
	paths := make([]string, 0, len(rt.instances))
	for path := range rt.instances {
		paths = append(paths, path)
	}
	return paths
}

// buildMinimalInterface creates a minimal interface from loader exports
// This is used when modules are loaded without full type checking (e.g., in tests)
func (rt *ModuleRuntime) buildMinimalInterface(loaded *loader.LoadedModule) *iface.Iface {
	exports := make(map[string]*iface.IfaceItem)

	for name := range loaded.Exports {
		exports[name] = &iface.IfaceItem{
			Name:   name,
			Type:   nil, // No type info available without type checking
			Purity: false,
			Ref: core.GlobalRef{
				Module: loaded.Path,
				Name:   name,
			},
		}
	}

	// Build constructor map from loaded module's constructor info
	constructors := make(map[string]*iface.ConstructorScheme)
	if loaded.Constructors != nil {
		for ctorName, typeName := range loaded.Constructors {
			// Find arity and type params from the type declaration
			arity := 0
			typeParamCount := 0
			if typeDecl, ok := loaded.Types[typeName]; ok {
				typeParamCount = len(typeDecl.TypeParams)
				// Check if this is an algebraic type (sum type)
				if algType, ok := typeDecl.Definition.(*ast.AlgebraicType); ok {
					for _, ctor := range algType.Constructors {
						if ctor.Name == ctorName {
							arity = len(ctor.Fields)
							break
						}
					}
				}
			}

			// M-TAPP-FIX: Create type vars for ADT type parameters
			var adtTypeVars []types.Type
			for i := 0; i < typeParamCount; i++ {
				adtTypeVars = append(adtTypeVars, &types.TVar2{Name: fmt.Sprintf("t%d", i), Kind: types.Star})
			}

			// Create placeholder field types
			fieldTypes := make([]types.Type, arity)
			for i := 0; i < arity; i++ {
				if i < typeParamCount {
					// Use ADT type var for simple cases
					fieldTypes[i] = adtTypeVars[i]
				} else {
					fieldTypes[i] = &types.TVar2{Name: fmt.Sprintf("a%d", i), Kind: types.Star}
				}
			}

			// M-TAPP-FIX: Build correct result type
			var resultType types.Type
			if typeParamCount > 0 {
				resultType = &types.TApp{
					Constructor: &types.TCon{Name: typeName},
					Args:        adtTypeVars,
				}
			} else {
				resultType = &types.TCon{Name: typeName}
			}

			constructors[ctorName] = &iface.ConstructorScheme{
				TypeName:   typeName,
				CtorName:   ctorName,
				FieldTypes: fieldTypes,
				ResultType: resultType,
				Arity:      arity,
			}
		}
	}

	return &iface.Iface{
		Module:       loaded.Path,
		Exports:      exports,
		Constructors: constructors,
		Types:        make(map[string]*iface.TypeExport),
	}
}

// attachContracts attaches contract specifications to a FunctionValue
// M-VERIFY-CONTRACTS: Called during module evaluation to connect
// contracts from core.DeclMeta to the FunctionValue at runtime.
func attachContracts(fn *eval.FunctionValue, contracts []*core.Contract) {
	for _, c := range contracts {
		spec := &eval.ContractSpec{
			Kind:     c.Kind.String(), // "requires", "ensures", or "invariant"
			Expr:     c.Expr,
			Message:  c.Message,
			Location: c.Location,
		}
		switch c.Kind {
		case core.RequiresKind:
			fn.Preconditions = append(fn.Preconditions, spec)
		case core.EnsuresKind:
			fn.Postconditions = append(fn.Postconditions, spec)
		case core.InvariantKind:
			// Invariants are treated as both pre and post conditions
			fn.Preconditions = append(fn.Preconditions, spec)
			fn.Postconditions = append(fn.Postconditions, spec)
		}
	}
}
