package repl

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/link"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/types"
)

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
	// M-WASM-TYPECHECK-FLOAT-DIVERGENCE: Register type aliases from previously
	// loaded modules so signatures like `st: CommonsState` resolve to the
	// underlying record type when `CommonsState` is declared in an imported
	// module. Without this, the elaborator/type-checker treats the alias as
	// unknown and infers an open record from usage, producing free type
	// variables that leak across modules. Mirrors pipeline_module_compile.go
	// lines 201-205 (CLI path).
	for _, mod := range mr.modules {
		for aliasName, target := range mod.TypeAliases {
			elaborator.RegisterTypeAlias(aliasName, target)
		}
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
		// Track constructor -> ADT type mapping (for pattern matching)
		ctorTypes[ctorName] = typeName
		if _, exists := adtTypeParams[typeName]; !exists {
			adtTypeParams[typeName] = typeParamCount
		}

		// Build $adt.make_TypeName_CtorName factory type
		factoryKey := fmt.Sprintf("$adt.make_%s_%s", typeName, ctorName)

		// M-POLY-ADT / M-TAPP-FIX / M-TYPE-LIST-SOUND round 3: build the factory
		// scheme via the shared helper, which remaps declared param vars to the
		// ADT's result-type vars (t0, t1, ...) across the ENTIRE field type so
		// compound fields ([a], (a,a), Option[a]) stay tied to the result type.
		typeChecker.SetGlobalType(factoryKey, types.BuildConstructorFactoryScheme(
			typeName,
			arity,
			typeParamCount,
			typeParamNames,
			fieldTypes,
		))
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
	// M-WASM-TYPECHECK-FLOAT-DIVERGENCE: Also register type aliases from
	// previously loaded modules on the type-checker. The elaborator received
	// them above (so signatures resolve in the AST); the type-checker needs
	// them too so unification can expand the alias when comparing types.
	// Mirrors pipeline_module_compile.go lines 201-205 (CLI path).
	mr.mu.RLock()
	for _, mod := range mr.modules {
		for aliasName, target := range mod.TypeAliases {
			if _, alreadyLocal := elabAliases[aliasName]; alreadyLocal {
				continue // local definition wins over imported
			}
			typeChecker.RegisterTypeAlias(aliasName, target)
		}
	}
	mr.mu.RUnlock()

	// M-WASM-TYPECHECK-FLOAT-DIVERGENCE: Propagate parameter and return type
	// annotations from the elaborator so declared signatures like
	// `func f(x: Point) -> float` actually constrain inference. Without this,
	// the type-checker infers parameter types from usage and produces overly
	// polymorphic schemes whose free type variables leak across modules.
	// Mirrors pipeline_module_compile.go lines 207-219 (CLI path).
	if paramAnnots := elaborator.GetParamTypeAnnotations(); len(paramAnnots) > 0 {
		typeChecker.SetParamTypeAnnotations(paramAnnots)
	}
	if returnAnnots := elaborator.GetReturnTypeAnnotations(); len(returnAnnots) > 0 {
		typeChecker.SetReturnTypeAnnotations(returnAnnots)
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

	// Step 4.5: Monomorphization (matches native pipeline Phase 3.5)
	// Without this, polymorphic functions remain unspecialized and operator
	// lowering can't determine concrete types for dictionary dispatch.
	// This fixes: import shadowing (Bug 1) and eq_Int dispatch in helper lambdas (Bug 2).
	specializer := pipeline.NewSpecializer(&typeChecker.CoreTI)
	specializedProg, err := specializer.Specialize(elaboratedProg)
	if err != nil {
		return nil, fmt.Errorf("monomorphization error: %w", err)
	}

	// Step 4.6: Var Type Resolution (matches native pipeline Phase 3.5.5)
	// Resolves remaining type variables in Var nodes after monomorphization.
	varResolver := pipeline.NewVarResolver(typeChecker.CoreTI)
	varResolver.Resolve(specializedProg)

	// Step 5: Op lowering
	lowerer := pipeline.NewOpLowerer(typeEnv, typeChecker.CoreTI)
	// Pass resolved constraints so the lowerer knows concrete types for == , <, etc.
	// Without this, the lowerer defaults to eq_Int for all equality operations.
	lowerer.SetResolvedConstraints(typeChecker.GetResolvedConstraints())
	loweredProg, err := lowerer.Lower(specializedProg)
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
	// Enable binop shim so comparison ops deferred by op lowering (unknown type)
	// can dispatch based on runtime operand types (M-WASM-DICTIONARY-DISPATCH)
	evaluator.SetExperimentalBinopShim(true)

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
	// M-WASM-TYPECHECK-FLOAT-DIVERGENCE: snapshot the elaborator's local type
	// aliases so future modules importing from this one can resolve declared
	// signatures (e.g. `st: CommonsState`). Without this snapshot the alias
	// vanishes after LoadModule returns.
	localAliases := make(map[string]types.Type, len(elabAliases))
	for k, v := range elabAliases {
		localAliases[k] = v
	}
	mr.modules[name] = &RegisteredModule{
		Name:        name,
		Source:      sourceCode,
		Exports:     exports,
		TypeAliases: localAliases,
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
