package runtime

import (
	"fmt"
	"path/filepath"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/loader"
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
//
// Thread-safety: The runtime uses sync.Once within each ModuleInstance
// to ensure each module is evaluated exactly once.
type ModuleRuntime struct {
	loader    *loader.ModuleLoader       // For loading and type-checking modules
	evaluator *eval.CoreEvaluator        // For evaluating Core AST
	instances map[string]*ModuleInstance // Cache: path → instance
	basePath  string                     // Base path for resolving modules
	visiting  map[string]bool            // Track modules being visited (for cycle detection)
	pathStack []string                   // Current DFS path (for cycle error messages)
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

	return &ModuleRuntime{
		loader:    loader.NewModuleLoader(cleanPath),
		evaluator: eval.NewCoreEvaluator(),
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
	resolver := newModuleGlobalResolver(inst)
	rt.evaluator.SetGlobalResolver(resolver)

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

		// Store bindings
		for name, val := range bindings {
			inst.Bindings[name] = val
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
		inst.Bindings[e.Name] = val

		// Recursively process body if it exists
		if e.Body != nil {
			return rt.extractBindings(inst, e.Body)
		}

	case *core.Var:
		// Var at module level is typically the final "result" expression
		// For modules, we ignore this - we only care about bindings
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
