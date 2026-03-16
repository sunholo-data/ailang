package repl

import (
	"fmt"
	"sync"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/runtime"
	"github.com/sunholo/ailang/internal/types"
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
	// Parse "make_Option_Some" → TypeName="Option", CtorName="Some"
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

// LoadModule compiles and registers a module from source code.
// Returns the list of exported names on success.
func (mr *ModuleRegistry) LoadModule(name, sourceCode string) ([]string, error) {
	// Step 1: Parse
	l := lexer.New(sourceCode, name+".ail")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse error: %s", p.Errors()[0])
	}

	// Step 2: Elaborate to Core
	// Use ElaborateFile for modules (has module declaration) to get proper Meta map with IsExport flags
	// Use Elaborate for REPL-style code (bare expressions)
	elaborator := elaborate.NewElaborator()
	// CRITICAL: Add builtins to global environment so stdlib wrappers can reference them
	// Without this, calls like _string_intToStr are elaborated as Var instead of VarGlobal,
	// causing "undefined variable" errors when the closure is later executed.
	elaborator.AddBuiltinsToGlobalEnv()

	// CRITICAL: Register constructors from previously loaded modules for import resolution
	// Without this, imports like "import std/option (Option, Some, None)" fail because
	// the elaborator doesn't know that None/Some are constructors (they become unresolved Var nodes).
	mr.mu.RLock()
	for ctorName, ctor := range mr.constructors {
		elaborator.RegisterConstructorWithFields(
			ctor.TypeName,
			ctorName,
			ctor.Arity,
			true, // isImported
			ctor.TypeParamCount,
			ctor.TypeParamNames,
			ctor.FieldTypes,
		)
	}
	mr.mu.RUnlock()

	// CRITICAL: Handle module aliases for qualified access (e.g., cache.get)
	// Without this, "import std/sharedmem as cache" fails when code uses "cache.get(...)"
	// This mirrors the pattern in pipeline_module.go lines 360-372.
	globalRefs := make(map[string]core.GlobalRef)
	if program.File != nil {
		for _, imp := range program.File.Imports {
			// Handle module alias (import std/x as cache -> cache.get)
			if imp.ModuleAlias != "" {
				mr.mu.RLock()
				mod, ok := mr.modules[imp.Path]
				mr.mu.RUnlock()
				if ok {
					// Add all exports with qualified names (cache.get, cache.put, etc.)
					for exportName := range mod.Exports {
						qualifiedName := fmt.Sprintf("%s.%s", imp.ModuleAlias, exportName)
						globalRefs[qualifiedName] = core.GlobalRef{
							Module: imp.Path,
							Name:   exportName,
						}
					}
				}
			}

			// CRITICAL: Handle direct symbol imports (import std/option (wrap) -> wrap)
			// Without this, imported symbols like 'wrap' are elaborated as local Var
			// instead of VarGlobal, causing "undefined variable" at runtime.
			for _, symName := range imp.Symbols {
				// Determine the name to bind (use alias if present)
				// e.g., import std/list (length as listLength) -> bind "listLength"
				bindName := symName
				if imp.SymbolAliases != nil {
					if alias, ok := imp.SymbolAliases[symName]; ok {
						bindName = alias
					}
				}
				globalRefs[bindName] = core.GlobalRef{
					Module: imp.Path,
					Name:   symName,
				}
			}
		}
	}
	// CRITICAL: Merge globalRefs INTO the existing globalEnv (which contains builtins)
	// DO NOT call SetGlobalEnv() as it REPLACES the map, losing builtins added by AddBuiltinsToGlobalEnv()
	// Instead, use MergeGlobalEnv() which adds to the existing map
	if len(globalRefs) > 0 {
		elaborator.MergeGlobalEnv(globalRefs)
	}

	var coreProg *core.Program
	var err error
	if program.File != nil && program.File.Module != nil {
		// Module with explicit module declaration - use ElaborateFile for Meta population
		coreProg, err = elaborator.ElaborateFile(program.File)
	} else {
		// REPL-style code or bare expressions
		coreProg, err = elaborator.Elaborate(program)
	}
	if err != nil {
		return nil, fmt.Errorf("elaboration error: %w", err)
	}

	if len(coreProg.Decls) == 0 {
		return nil, fmt.Errorf("module %s has no declarations", name)
	}

	// Step 3: Type check
	typeEnv := types.NewTypeEnvWithBuiltins()
	typeEnv = pipeline.InjectPrelude(typeEnv)

	// Use NewCoreTypeChecker which sets up Num/Fractional defaults and loads builtin instances
	typeChecker := types.NewCoreTypeChecker()

	// CRITICAL: Populate globalTypes with builtin type schemes
	// Without this, VarGlobal references to builtins (e.g., $builtin._str_len)
	// fail with "undefined global variable" during type checking.
	// This mirrors the pattern in pipeline_module.go lines 202-214.
	modLinker := link.NewModuleLinker(nil)
	link.RegisterBuiltinModule(modLinker)
	if builtinIface := modLinker.GetIface("$builtin"); builtinIface != nil {
		for _, item := range builtinIface.Exports {
			// Add with qualified key (for VarGlobal references like $builtin._str_len)
			key := fmt.Sprintf("%s.%s", item.Ref.Module, item.Ref.Name)
			typeChecker.SetGlobalType(key, item.Type)
		}
	}

	// CRITICAL: Register global types for module alias exports
	// Without this, VarGlobal references to aliased imports (e.g., cache.get -> std/sharedmem.get)
	// fail with "undefined global variable" during type checking.
	// This pairs with the globalRefs registration above for the elaborator.
	for _, ref := range globalRefs {
		key := fmt.Sprintf("%s.%s", ref.Module, ref.Name)
		mr.mu.RLock()
		mod, ok := mr.modules[ref.Module]
		mr.mu.RUnlock()
		if ok {
			if export, exists := mod.Exports[ref.Name]; exists && export.Scheme != nil {
				typeChecker.SetGlobalType(key, export.Scheme)
			}
		}
	}

	// CRITICAL: Register ADT constructor factory types ($adt.make_X_Y)
	// Without this, VarGlobal references to constructors fail with "undefined global variable".
	// This mirrors the pattern in pipeline_module.go lines 407-504.
	elabCtors := elaborator.GetConstructors()
	ctorTypes := make(map[string]string)
	adtTypeParams := make(map[string]int)

	// Helper function to build and register $adt factory type for a constructor
	registerAdtFactory := func(ctorName string, typeName string, arity int, typeParamCount int, typeParamNames []string, fieldTypes []types.Type) {
		// Track constructor → ADT type mapping (for pattern matching)
		ctorTypes[ctorName] = typeName
		if _, exists := adtTypeParams[typeName]; !exists {
			adtTypeParams[typeName] = typeParamCount
		}

		// Build $adt.make_TypeName_CtorName factory type
		factoryName := fmt.Sprintf("make_%s_%s", typeName, ctorName)
		factoryKey := fmt.Sprintf("$adt.%s", factoryName)

		// Build type variables for polymorphic constructors
		var typeVars []string
		var adtTypeVars []types.Type
		for i := 0; i < typeParamCount; i++ {
			varName := fmt.Sprintf("t%d", i)
			typeVars = append(typeVars, varName)
			adtTypeVars = append(adtTypeVars, &types.TVar2{Name: varName, Kind: types.Star})
		}

		// Build map of type param names to type vars
		typeParamToVar := make(map[string]types.Type)
		for i, paramName := range typeParamNames {
			if i < len(adtTypeVars) {
				typeParamToVar[paramName] = adtTypeVars[i]
			}
		}

		// Build parameter types for the factory function
		var paramTypes []types.Type
		for i := 0; i < arity; i++ {
			var fieldType types.Type
			if i < len(fieldTypes) && fieldTypes[i] != nil {
				ft := fieldTypes[i]
				// Check if field type is a type variable that matches a type parameter
				if tvar, ok := ft.(*types.TVar2); ok {
					if mappedVar, found := typeParamToVar[tvar.Name]; found {
						fieldType = mappedVar
					} else {
						varName := fmt.Sprintf("a%d", i)
						typeVars = append(typeVars, varName)
						fieldType = &types.TVar2{Name: varName, Kind: types.Star}
					}
				} else {
					// Concrete type - use directly
					fieldType = ft
				}
			} else if i < typeParamCount {
				fieldType = adtTypeVars[i]
			} else {
				varName := fmt.Sprintf("a%d", i)
				typeVars = append(typeVars, varName)
				fieldType = &types.TVar2{Name: varName, Kind: types.Star}
			}
			paramTypes = append(paramTypes, fieldType)
		}

		// Build result type: TApp(TypeName, [t0, t1, ...]) for polymorphic, TCon for non-polymorphic
		var resultType types.Type
		if typeParamCount > 0 {
			resultType = &types.TApp{
				Constructor: &types.TCon{Name: typeName},
				Args:        adtTypeVars,
			}
		} else {
			resultType = &types.TCon{Name: typeName}
		}

		// Build factory function type
		var factoryType types.Type
		if arity == 0 {
			factoryType = resultType
		} else {
			factoryType = &types.TFunc2{
				Params:    paramTypes,
				EffectRow: nil, // Constructors are pure
				Return:    resultType,
			}
		}

		// Register with type checker
		typeChecker.SetGlobalType(factoryKey, &types.Scheme{
			TypeVars: typeVars,
			Type:     factoryType,
		})
	}

	// Register $adt factory types for IMPORTED constructors (from previously loaded modules)
	mr.mu.RLock()
	for ctorName, ctor := range mr.constructors {
		registerAdtFactory(ctorName, ctor.TypeName, ctor.Arity, ctor.TypeParamCount, ctor.TypeParamNames, ctor.FieldTypes)
	}
	mr.mu.RUnlock()

	// Register $adt factory types for LOCAL constructors (from this module)
	for ctorName, ctorInfo := range elabCtors {
		registerAdtFactory(ctorName, ctorInfo.TypeName, ctorInfo.Arity, ctorInfo.TypeParamCount, ctorInfo.TypeParamNames, ctorInfo.FieldTypes)
	}

	// Set constructor types and ADT type params for pattern matching
	typeChecker.SetConstructorTypes(ctorTypes)
	typeChecker.SetADTTypeParams(adtTypeParams)

	// CRITICAL: Register type aliases for unification
	// Without this, record type aliases like "type Invoice = {...}" fail with
	// "cannot unify type constructor Invoice with *types.TRecordOpen"
	// This mirrors the pattern in pipeline_module.go lines 558-563.
	elabAliases := elaborator.GetTypeAliases()
	for name, target := range elabAliases {
		typeChecker.RegisterTypeAlias(name, target)
	}

	// CRITICAL: Add exported function types from previously loaded modules
	// Without this, function imports like "import std/list (map)" fail because
	// the type checker doesn't know the type of `map` when type checking std/json.
	mr.mu.RLock()
	for _, mod := range mr.modules {
		for exportName, export := range mod.Exports {
			if export.Scheme != nil {
				// Add to type environment for import resolution
				typeEnv = typeEnv.ExtendScheme(exportName, export.Scheme)
			}
		}
	}
	mr.mu.RUnlock()

	// Type check each declaration and collect types
	// Handle both Let and LetRec declarations
	declTypes := make(map[string]*types.Scheme)
	for _, decl := range coreProg.Decls {
		var declName string
		var names []string

		switch d := decl.(type) {
		case *core.Let:
			declName = d.Name
			names = []string{d.Name}
		case *core.LetRec:
			if len(d.Bindings) > 0 {
				declName = d.Bindings[0].Name
			}
			for _, b := range d.Bindings {
				names = append(names, b.Name)
			}
		default:
			continue
		}

		_, updatedEnv, _, _, err := typeChecker.InferWithConstraints(decl, typeEnv)
		if err != nil {
			return nil, fmt.Errorf("type error in %s: %w", declName, err)
		}
		typeEnv = updatedEnv

		// Extract the scheme for each binding
		for _, name := range names {
			if result, err := typeEnv.Lookup(name); err == nil {
				if scheme, ok := result.(*types.Scheme); ok {
					declTypes[name] = scheme
				}
			}
		}
	}

	// Step 4: Dictionary elaboration
	resolved := typeChecker.GetResolvedConstraints()
	elaboratedProg, err := elaborate.ElaborateWithDictionaries(coreProg, resolved)
	if err != nil {
		return nil, fmt.Errorf("dictionary elaboration error: %w", err)
	}

	// Step 5: Op lowering
	lowerer := pipeline.NewOpLowerer(typeEnv, typeChecker.CoreTI)
	loweredProg, err := lowerer.Lower(elaboratedProg)
	if err != nil {
		return nil, fmt.Errorf("op lowering error: %w", err)
	}

	// Step 6: Link dictionaries
	linker := link.NewLinker()
	registerPreludeInstances(linker) // Register standard instances

	linkedDecls := make([]core.CoreExpr, 0, len(loweredProg.Decls))
	for _, decl := range loweredProg.Decls {
		linked, err := linker.Link(decl)
		if err != nil {
			return nil, fmt.Errorf("linking error: %w", err)
		}
		linkedDecls = append(linkedDecls, linked)
	}

	// Step 7: Evaluate to get values
	evaluator := eval.NewCoreEvaluator()
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	// Use RegistryResolver to resolve imports from previously loaded modules
	// This enables cross-module imports like "import std/json (js, encode)"
	registryResolver := NewRegistryResolver(mr, builtinRegistry)
	evaluator.SetGlobalResolver(registryResolver)

	// First, check if there are any explicit exports in the module
	hasExplicitExports := false
	for _, meta := range coreProg.Meta {
		if meta != nil && meta.IsExport {
			hasExplicitExports = true
			break
		}
	}

	exports := make(map[string]*Export)
	for i, decl := range linkedDecls {
		// Extract names and evaluate based on declaration type
		var names []string

		switch d := coreProg.Decls[i].(type) {
		case *core.Let:
			names = []string{d.Name}
		case *core.LetRec:
			for _, binding := range d.Bindings {
				names = append(names, binding.Name)
			}
		default:
			// Unknown declaration type, skip
			continue
		}

		// Evaluate the declaration
		val, err := evaluator.Eval(decl)
		if err != nil {
			return nil, fmt.Errorf("eval error: %w", err)
		}

		// For Let nodes, store the value directly in the environment
		// Note: LetRec bindings are now propagated to parent env in evalCoreLetRec (Phase 2.5)
		if _, ok := coreProg.Decls[i].(*core.Let); ok && len(names) == 1 {
			evaluator.Env().Set(names[0], val)
		}

		// Process each declared name
		for _, declName := range names {
			// Get the actual value for this binding
			var bindingVal eval.Value

			// For Let with single binding, use the direct evaluation result
			// For LetRec (any size), get from environment (set by Phase 2.5 in evalCoreLetRec)
			if _, ok := coreProg.Decls[i].(*core.Let); ok && len(names) == 1 {
				bindingVal = val
			} else {
				// LetRec: get the value from the evaluator's environment
				var found bool
				bindingVal, found = evaluator.Env().Get(declName)
				if !found {
					return nil, fmt.Errorf("binding %s not found after evaluation", declName)
				}
			}

			// Determine if this should be exported:
			// - If module has explicit exports, only export those marked with IsExport
			// - If no explicit exports, export all top-level bindings (REPL/backwards compat)
			meta := coreProg.Meta[declName]
			shouldExport := false
			if hasExplicitExports {
				shouldExport = meta != nil && meta.IsExport
			} else {
				// No explicit exports - export everything
				shouldExport = true
			}

			if shouldExport {
				exports[declName] = &Export{
					Name:   declName,
					Value:  bindingVal,
					Scheme: declTypes[declName],
				}
			}
		}
	}

	// Register module and cache its constructors for future imports
	mr.mu.Lock()
	mr.modules[name] = &RegisteredModule{
		Name:    name,
		Source:  sourceCode,
		Exports: exports,
	}
	// Cache constructors from this module for future import resolution
	for ctorName, ctorInfo := range elabCtors {
		mr.constructors[ctorName] = &CachedConstructor{
			TypeName:       ctorInfo.TypeName,
			CtorName:       ctorInfo.CtorName,
			Arity:          ctorInfo.Arity,
			TypeParamCount: ctorInfo.TypeParamCount,
			TypeParamNames: ctorInfo.TypeParamNames,
			FieldTypes:     ctorInfo.FieldTypes,
		}
	}
	mr.mu.Unlock()

	// Return export names
	exportNames := make([]string, 0, len(exports))
	for name := range exports {
		exportNames = append(exportNames, name)
	}
	return exportNames, nil
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

// CallExport formats a function call expression string for use with the REPL.
// This returns the expression that can be evaluated via ProcessExpression.
// Arguments are converted to their AILANG string representation.
func (mr *ModuleRegistry) CallExport(moduleName, funcName string, args []eval.Value) (string, error) {
	// Verify the export exists
	_, err := mr.GetExport(moduleName, funcName)
	if err != nil {
		return "", err
	}

	// Build curried call expression: moduleName.funcName(arg1)(arg2)...
	// First we need to import the module, then call the function
	expr := funcName
	for _, arg := range args {
		argStr := formatArgument(arg)
		expr = fmt.Sprintf("%s(%s)", expr, argStr)
	}

	return expr, nil
}

// InvokeExport calls an exported function directly with the provided arguments.
// This method bypasses REPL string evaluation and calls the function closure
// directly, ensuring that captured imports (like decode, encode from std/json)
// are properly resolved from the function's environment.
//
// Returns the result value and any error that occurred during execution.
func (mr *ModuleRegistry) InvokeExport(moduleName, funcName string, args []eval.Value) (eval.Value, error) {
	// Get the export
	export, err := mr.GetExport(moduleName, funcName)
	if err != nil {
		return nil, err
	}

	// Get the function value
	fn, ok := export.Value.(*eval.FunctionValue)
	if !ok {
		return nil, fmt.Errorf("export %s.%s is not a function (got %T)", moduleName, funcName, export.Value)
	}

	// Create an evaluator with the RegistryResolver so that any global references
	// (builtins, imports from other modules) can be resolved during execution
	evaluator := eval.NewCoreEvaluator()
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	registryResolver := NewRegistryResolver(mr, builtinRegistry)
	evaluator.SetGlobalResolver(registryResolver)

	// Set effect context if available (enables AI, IO, and other effects in WASM)
	// M-ITERATIVE-LIST: Always set an EffContext so FnCaller/FnCallerN are wired
	// for iterative builtins (_list_map, _list_foldl, etc.)
	if mr.effContext != nil {
		evaluator.SetEffContext(mr.effContext)
	} else {
		evaluator.SetEffContext(effects.NewEffContext(nil))
	}

	// Enable experimental binop shim (handles float equality until OpLowering is complete)
	evaluator.SetExperimentalBinopShim(true)

	// Register type class dictionaries (Num, Eq, Ord, Fractional) so that
	// module functions using arithmetic, comparisons, or string operations work.
	registerPreludeInstancesForEvaluator(evaluator)

	// Apply arguments, handling both multi-param and curried functions.
	// Multi-param: func f(a, b) compiled with Params=["a","b"] — needs all args at once.
	// Curried: func f(a)(b) compiled as nested lambdas — apply one at a time.
	var result eval.Value = fn
	remaining := args
	for len(remaining) > 0 {
		funcVal, isFn := result.(*eval.FunctionValue)
		if !isFn {
			return nil, fmt.Errorf("too many arguments: value is not a function (got %T) with %d args remaining", result, len(remaining))
		}

		arity := len(funcVal.Params)
		if arity <= 0 {
			arity = 1
		}

		if arity <= len(remaining) {
			// Apply arity-many arguments at once
			result, err = evaluator.CallFunction(funcVal, remaining[:arity])
			if err != nil {
				applied := len(args) - len(remaining)
				return nil, fmt.Errorf("error applying argument(s) %d-%d: %w", applied+1, applied+arity, err)
			}
			remaining = remaining[arity:]
		} else {
			// More params than remaining args — apply what we have one at a time
			// (shouldn't normally happen, but handle gracefully)
			result, err = evaluator.CallFunction(funcVal, remaining)
			if err != nil {
				return nil, fmt.Errorf("error applying final arguments: %w", err)
			}
			remaining = nil
		}
	}

	return result, nil
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

// registerPreludeInstances registers standard type class instances with the linker.
// This mirrors the instances registered in the REPL.
func registerPreludeInstances(linker *link.Linker) {
	// Helper to create builtin functions
	wrapInt2 := func(name string, f func(int64, int64) int64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{
			Name: name,
			Fn: func(args []eval.Value) (eval.Value, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("expected 2 arguments")
				}
				x, ok1 := args[0].(*eval.IntValue)
				y, ok2 := args[1].(*eval.IntValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("expected int arguments")
				}
				return &eval.IntValue{Value: int(f(int64(x.Value), int64(y.Value)))}, nil
			},
		}
	}

	wrapFloat2 := func(name string, f func(float64, float64) float64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{
			Name: name,
			Fn: func(args []eval.Value) (eval.Value, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("expected 2 arguments")
				}
				x, ok1 := args[0].(*eval.FloatValue)
				y, ok2 := args[1].(*eval.FloatValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("expected float arguments")
				}
				return &eval.FloatValue{Value: f(x.Value, y.Value)}, nil
			},
		}
	}

	// Num[Int]
	numInt := core.DictValue{
		TypeClass: "Num",
		Type:      "Int",
		Methods: map[string]interface{}{
			"add": wrapInt2("add", func(a, b int64) int64 { return a + b }),
			"sub": wrapInt2("sub", func(a, b int64) int64 { return a - b }),
			"mul": wrapInt2("mul", func(a, b int64) int64 { return a * b }),
			"div": wrapInt2("div", func(a, b int64) int64 { return a / b }),
		},
	}

	// Num[Float]
	numFloat := core.DictValue{
		TypeClass: "Num",
		Type:      "Float",
		Methods: map[string]interface{}{
			"add": wrapFloat2("add", func(a, b float64) float64 { return a + b }),
			"sub": wrapFloat2("sub", func(a, b float64) float64 { return a - b }),
			"mul": wrapFloat2("mul", func(a, b float64) float64 { return a * b }),
			"div": wrapFloat2("div", func(a, b float64) float64 { return a / b }),
		},
	}

	// Register with canonical keys
	for methodName := range numInt.Methods {
		key := types.MakeDictionaryKey("prelude", "Num", &types.TCon{Name: "Int"}, methodName)
		linker.AddDictionary(key, numInt)
	}

	for methodName := range numFloat.Methods {
		key := types.MakeDictionaryKey("prelude", "Num", &types.TCon{Name: "Float"}, methodName)
		linker.AddDictionary(key, numFloat)
	}
}

// registerPreludeInstancesForEvaluator registers all type class dictionaries
// (Num, Eq, Ord, Fractional) with a CoreEvaluator so that functions using
// arithmetic, comparisons, or string operations work via InvokeExport.
func registerPreludeInstancesForEvaluator(evaluator *eval.CoreEvaluator) {
	// --- Wrapper helpers ---
	wrapInt2 := func(name string, f func(int64, int64) int64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{Name: name, Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*eval.IntValue)
			y, ok2 := args[1].(*eval.IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			return &eval.IntValue{Value: int(f(int64(x.Value), int64(y.Value)))}, nil
		}}
	}

	wrapFloat2 := func(name string, f func(float64, float64) float64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{Name: name, Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*eval.FloatValue)
			y, ok2 := args[1].(*eval.FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			return &eval.FloatValue{Value: f(x.Value, y.Value)}, nil
		}}
	}

	wrapIntCmp2 := func(name string, f func(int64, int64) bool) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{Name: name, Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*eval.IntValue)
			y, ok2 := args[1].(*eval.IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			return &eval.BoolValue{Value: f(int64(x.Value), int64(y.Value))}, nil
		}}
	}

	wrapFloatCmp2 := func(name string, f func(float64, float64) bool) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{Name: name, Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*eval.FloatValue)
			y, ok2 := args[1].(*eval.FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			return &eval.BoolValue{Value: f(x.Value, y.Value)}, nil
		}}
	}

	// --- Build all type class instances ---
	instances := map[string]core.DictValue{
		"Num[Int]": {
			TypeClass: "Num", Type: "Int",
			Methods: map[string]interface{}{
				"add": wrapInt2("add", func(a, b int64) int64 { return a + b }),
				"sub": wrapInt2("sub", func(a, b int64) int64 { return a - b }),
				"mul": wrapInt2("mul", func(a, b int64) int64 { return a * b }),
				"div": wrapInt2("div", func(a, b int64) int64 { return a / b }),
			},
		},
		"Num[Float]": {
			TypeClass: "Num", Type: "Float",
			Methods: map[string]interface{}{
				"add": wrapFloat2("add", func(a, b float64) float64 { return a + b }),
				"sub": wrapFloat2("sub", func(a, b float64) float64 { return a - b }),
				"mul": wrapFloat2("mul", func(a, b float64) float64 { return a * b }),
				"div": wrapFloat2("div", func(a, b float64) float64 { return a / b }),
			},
		},
		"Eq[Int]": {
			TypeClass: "Eq", Type: "Int",
			Methods: map[string]interface{}{
				"eq":  wrapIntCmp2("eq", func(a, b int64) bool { return a == b }),
				"neq": wrapIntCmp2("neq", func(a, b int64) bool { return a != b }),
			},
		},
		"Eq[Float]": {
			TypeClass: "Eq", Type: "Float",
			Methods: map[string]interface{}{
				"eq":  wrapFloatCmp2("eq", func(a, b float64) bool { return a == b }),
				"neq": wrapFloatCmp2("neq", func(a, b float64) bool { return a != b }),
			},
		},
		"Ord[Int]": {
			TypeClass: "Ord", Type: "Int",
			Methods: map[string]interface{}{
				"lt":  wrapIntCmp2("lt", func(a, b int64) bool { return a < b }),
				"lte": wrapIntCmp2("lte", func(a, b int64) bool { return a <= b }),
				"gt":  wrapIntCmp2("gt", func(a, b int64) bool { return a > b }),
				"gte": wrapIntCmp2("gte", func(a, b int64) bool { return a >= b }),
			},
			Provides: []string{"Eq[Int]"},
		},
		"Ord[Float]": {
			TypeClass: "Ord", Type: "Float",
			Methods: map[string]interface{}{
				"lt":  wrapFloatCmp2("lt", func(a, b float64) bool { return a < b }),
				"lte": wrapFloatCmp2("lte", func(a, b float64) bool { return a <= b }),
				"gt":  wrapFloatCmp2("gt", func(a, b float64) bool { return a > b }),
				"gte": wrapFloatCmp2("gte", func(a, b float64) bool { return a >= b }),
			},
			Provides: []string{"Eq[Float]"},
		},
	}

	// Register each instance with canonical keys for evaluator lookups
	for _, dict := range instances {
		typeForKey := &types.TCon{Name: dict.Type}
		for methodName := range dict.Methods {
			canonicalKey := types.MakeDictionaryKey("prelude", dict.TypeClass, typeForKey, methodName)
			evaluator.AddDictionary(canonicalKey, dict)
		}
		// Also register the base dictionary (no method name) for lookups
		baseKey := types.MakeDictionaryKey("prelude", dict.TypeClass, typeForKey, "")
		evaluator.AddDictionary(baseKey, dict)
	}
}
