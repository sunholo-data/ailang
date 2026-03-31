package pipeline

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/types"
)

// moduleCompileResult holds the output of compiling a single module
// (type checking through lowering).
type moduleCompileResult struct {
	TypeChecker   *types.CoreTypeChecker
	DebugSink     *types.VerboseDebugSink
	Warnings      []*elaborate.ExhaustivenessWarning
	ModuleTypeEnv *types.TypeEnv // Accumulated type environment for interface building
}

// buildConstructorFactoryTypes creates $adt factory type schemes for a module's
// locally-defined constructors and adds them to the external types map.
// This allows the type checker to know about constructor factories.
func buildConstructorFactoryTypes(
	constructors map[string]*ConstructorInfo,
	externalTypes map[string]*types.Scheme,
) {
	for ctorName, ctorInfo := range constructors {
		factoryName := fmt.Sprintf("make_%s_%s", ctorInfo.TypeName, ctorName)
		factoryKey := fmt.Sprintf("$adt.%s", factoryName)

		// Build factory type: a0 -> a1 -> ... -> TypeName[t0, t1, ...]
		// Use TVar2 (new type system) for type variables with Star kind
		var typeVars []string
		var paramTypes []types.Type

		// M-TAPP-FIX: First create type vars for ADT type parameters
		// These will be used in the result type: TApp(TypeName, [t0, t1, ...])
		var adtTypeVars []types.Type
		for i := 0; i < ctorInfo.TypeParamCount; i++ {
			varName := fmt.Sprintf("t%d", i)
			typeVars = append(typeVars, varName)
			adtTypeVars = append(adtTypeVars, &types.TVar2{Name: varName, Kind: types.Star})
		}

		// M-POLY-ADT: Use actual field types instead of assuming positional match
		// This fixes the bug where Err(string) in Result[a] was incorrectly typed as forall a. a -> Result[a]
		// Build a map of type param names to their ADT type vars (t0, t1, ...)
		typeParamToVar := make(map[string]types.Type)
		for i, name := range ctorInfo.TypeParamNames {
			if i < len(adtTypeVars) {
				typeParamToVar[name] = adtTypeVars[i]
			}
		}

		for i := 0; i < ctorInfo.Arity; i++ {
			var fieldType types.Type
			if i < len(ctorInfo.InternalFieldTypes) && ctorInfo.InternalFieldTypes[i] != nil {
				// M-POLY-ADT: We have the actual field type from elaboration
				ft := ctorInfo.InternalFieldTypes[i]
				// Check if field type is a type variable that matches a type parameter
				if tvar, ok := ft.(*types.TVar2); ok {
					if mappedVar, found := typeParamToVar[tvar.Name]; found {
						// This field type IS a type parameter (e.g., 'a' in Ok(a))
						// Use the corresponding ADT type var (t0, t1, ...)
						fieldType = mappedVar
					} else {
						// Unknown type var - create fresh type var
						varName := fmt.Sprintf("a%d", i)
						typeVars = append(typeVars, varName)
						fieldType = &types.TVar2{Name: varName, Kind: types.Star}
					}
				} else {
					// Concrete type (e.g., 'string' in Err(string))
					// Use it directly - this is the key fix!
					fieldType = ft
				}
			} else if i < ctorInfo.TypeParamCount {
				// Fallback: no field types available, use old behavior
				fieldType = adtTypeVars[i]
			} else {
				// Create additional type var for extra fields
				varName := fmt.Sprintf("a%d", i)
				typeVars = append(typeVars, varName)
				fieldType = &types.TVar2{Name: varName, Kind: types.Star}
			}
			paramTypes = append(paramTypes, fieldType)
		}

		// M-TAPP-FIX: Build correct result type
		// For parameterized ADTs, use TApp(TCon(TypeName), [type vars])
		// For non-parameterized ADTs, use plain TCon
		var resultType types.Type
		if ctorInfo.TypeParamCount > 0 {
			resultType = &types.TApp{
				Constructor: &types.TCon{Name: ctorInfo.TypeName},
				Args:        adtTypeVars,
			}
		} else {
			resultType = &types.TCon{Name: ctorInfo.TypeName}
		}

		var factoryType types.Type
		if ctorInfo.Arity == 0 {
			// Nullary constructor: just the result type
			factoryType = resultType
		} else {
			// Constructor with fields
			// Use TFunc2 (new type system) for compatibility with unification
			factoryType = &types.TFunc2{
				Params:    paramTypes,
				EffectRow: nil, // Pure constructor
				Return:    resultType,
			}
		}

		// Add to external types with Scheme wrapper
		// TypeVars allows polymorphism over field types
		externalTypes[factoryKey] = &types.Scheme{
			TypeVars: typeVars,
			Type:     factoryType,
		}
	}
}

// typeCheckAndLowerModule runs type checking, dictionary elaboration, monomorphization,
// var resolution, and operator lowering on a single compile unit.
// It returns the compile result including the accumulated moduleTypeEnv for interface building.
func typeCheckAndLowerModule(
	unit *CompileUnit,
	modID string,
	rootCanonical string,
	imports *moduleImports,
	elaborator *elaborate.Elaborator,
	cfg Config,
) (*moduleCompileResult, error) {
	result := &moduleCompileResult{}

	// Type check with external types from dependencies
	// Create a local TypeEnv for this module (inherits from global builtins)
	moduleTypeEnv := types.NewTypeEnvWithBuiltins()

	// Entry-module prelude injection (BEFORE type checking)
	// Detect entry modules by scanning AST for 'export func main' with 0 params
	if IsEntryModuleFromAST(unit.Surface) {
		moduleTypeEnv = InjectPrelude(moduleTypeEnv)
	}

	typeChecker := types.NewCoreTypeCheckerWithInstances(cfg.InstEnv)
	typeChecker.EnableTraceDefaulting(cfg.TraceDefaulting)
	if cfg.TrackInstantiations {
		typeChecker.EnableInstantiationTracking()
	}

	// M-DX11: Set up debug sink for root module only
	if cfg.DebugTypes && modID == rootCanonical {
		debugSink := types.NewVerboseDebugSink()
		typeChecker.SetDebugSink(debugSink)
		// Capture root module's type checker and debug sink
		result.TypeChecker = typeChecker
		result.DebugSink = debugSink
	}

	typeChecker.SetGlobalTypes(imports.ExternalTypes)

	// M-DX25.4: Pass constructor -> ADT type mappings to type checker
	// This enables correct type inference for pattern matching on ADTs
	ctorTypes := make(map[string]string)
	adtTypeParams := make(map[string]int) // M-TAPP-FIX: Track type param counts

	// M-TAPP-FIX: First add imported constructors (from depIface.GetConstructor)
	for ctorName, typeName := range imports.ImportedCtorTypes {
		ctorTypes[ctorName] = typeName
	}
	for typeName, paramCount := range imports.ImportedADTTypeParams {
		adtTypeParams[typeName] = paramCount
	}

	// Then add local constructors (may override imports if same name)
	for ctorName, ctorInfo := range unit.Constructors {
		ctorTypes[ctorName] = ctorInfo.TypeName
		// M-TAPP-FIX: Only set if not already set (first ctor wins, all should have same count)
		if _, exists := adtTypeParams[ctorInfo.TypeName]; !exists {
			adtTypeParams[ctorInfo.TypeName] = ctorInfo.TypeParamCount
		}
	}
	typeChecker.SetConstructorTypes(ctorTypes)
	typeChecker.SetADTTypeParams(adtTypeParams) // M-TAPP-FIX

	// M-FIX-RECORD-UPDATE: Pass type aliases to type checker for expansion during unification
	// This enables `type NPC = { pos: Pos, name: string }` to work with record update
	elabAliases := elaborator.GetTypeAliases()
	for name, target := range elabAliases {
		typeChecker.RegisterTypeAlias(name, target)
	}

	// M-FIX-RECORD-UPDATE: Also register type aliases from imported modules
	// This enables cross-module record update (e.g., { npc | pos: ... } where NPC is imported)
	for name, target := range imports.ImportedTypeAliases {
		typeChecker.RegisterTypeAlias(name, target)
	}

	// M-FIX-FLOAT-OP: Pass parameter type annotations to type checker
	// This preserves float annotations from function declarations through elaboration
	paramAnnots := elaborator.GetParamTypeAnnotations()
	if len(paramAnnots) > 0 {
		typeChecker.SetParamTypeAnnotations(paramAnnots)
	}

	// M-FIX-FLOAT-OP: Pass return type annotations to type checker
	// This ensures PI() -> float ACTUALLY constrains inference to return float
	returnAnnots := elaborator.GetReturnTypeAnnotations()
	if len(returnAnnots) > 0 {
		typeChecker.SetReturnTypeAnnotations(returnAnnots)
	}

	// M-CAPABILITY-BUDGETS: Pass full effect annotations with budgets to type checker
	// This preserves @limit=N budget annotations from function declarations through elaboration
	effectAnnotsFull := elaborator.GetEffectAnnotationsFull()
	if len(effectAnnotsFull) > 0 {
		typeChecker.SetEffectAnnotationsFull(effectAnnotsFull)
	}

	// M-DX19: Register derived Eq instances for types with `deriving (Eq)`
	// This allows == to work on user-defined ADT and record types
	derivedEqTypes := elaborator.GetDerivedEqTypes()
	for _, typeName := range derivedEqTypes {
		inst := &types.ClassInstance{
			ClassName: "Eq",
			TypeHead:  &types.TCon{Name: typeName},
			Dict: types.Dict{
				"eq":  fmt.Sprintf("derived_eq_%s", typeName),
				"neq": fmt.Sprintf("derived_neq_%s", typeName),
			},
		}
		if err := cfg.InstEnv.Add(inst); err != nil {
			// Ignore duplicate instance errors (may happen with multiple files)
			if cfg.DebugCompile {
				fmt.Fprintf(os.Stderr, "[DEBUG] Could not add derived Eq instance for %s: %v\n", typeName, err)
			}
		}

		// M-DX19: Also register in DictionaryRegistry for runtime lookup
		cfg.DictReg.RegisterDerivedEq(typeName)
	}

	// Type check ALL declarations in the module, accumulating types in moduleTypeEnv
	for i, decl := range unit.Core.Decls {
		// InferWithConstraints returns the updated env with new bindings
		var err error
		_, moduleTypeEnv, _, _, err = typeChecker.InferWithConstraints(decl, moduleTypeEnv)
		if err != nil {
			return nil, fmt.Errorf("type error in %s (decl %d): %w", modID, i, err)
		}
	}

	// Fill operator methods (resolve operators to type class methods)
	// This populates the Method field in resolved constraints before lowering
	for _, decl := range unit.Core.Decls {
		typeChecker.FillOperatorMethods(decl)
	}

	// M-CONTRACT-OPLOWERING-FIX: Type-check contract expressions to populate CoreTI.
	// Without this, OpLowering hits CoreTI misses for operators in ensures/requires clauses,
	// causing comparison ops to be deferred as raw Intrinsic nodes that fail at runtime.
	if unit.Core.Meta != nil {
		// Build a map of function name -> Lambda params for contract env construction
		funcParams := extractFuncParams(unit.Core)

		for funcName, meta := range unit.Core.Meta {
			if len(meta.Contracts) == 0 {
				continue
			}

			// Build contract environment: module env + function params + result
			contractEnv := moduleTypeEnv

			// Bind function parameters from the function's type signature
			if binding, err := moduleTypeEnv.Lookup(funcName); err == nil {
				paramTypes, retType := extractFuncSignature(binding)

				// Bind parameters by name (from Core Lambda) with types (from type env)
				if params, ok := funcParams[funcName]; ok {
					for i, paramName := range params {
						if i < len(paramTypes) {
							contractEnv = contractEnv.Extend(paramName, paramTypes[i])
						}
					}
				}

				// Bind "result" for ensures clauses
				if retType != nil {
					contractEnv = contractEnv.Extend("result", retType)
				}
			}

			for _, contract := range meta.Contracts {
				if contract.Expr == nil {
					continue
				}
				// Infer types for contract expression — populates CoreTI as side effect.
				// Errors are non-fatal: contract type-checking is best-effort for CoreTI population.
				_, _, _, _, inferErr := typeChecker.InferWithConstraints(contract.Expr, contractEnv)
				if inferErr != nil && cfg.DebugCompile {
					fmt.Fprintf(os.Stderr, "[DEBUG] Contract type inference for %s: %v\n", funcName, inferErr)
				}
				// Also fill operator methods for contract expressions
				typeChecker.FillOperatorMethods(contract.Expr)
			}
		}
	}

	// M-DX23: Capture type info for codegen
	unit.CoreTI = typeChecker.CoreTI

	// Run post-typecheck phases (dict elaboration, mono, var resolution, lowering)
	if err := runPostTypeCheckPhases(unit, modID, typeChecker, cfg); err != nil {
		return nil, err
	}

	// Collect exhaustiveness warnings
	warnings := elaborator.GetWarnings()
	result.Warnings = append(result.Warnings, warnings...)
	result.ModuleTypeEnv = moduleTypeEnv

	return result, nil
}

// runPostTypeCheckPhases runs dictionary elaboration, monomorphization,
// var resolution, and operator lowering on a compile unit.
func runPostTypeCheckPhases(
	unit *CompileUnit,
	modID string,
	typeChecker *types.CoreTypeChecker,
	cfg Config,
) error {
	// Phase 3.4: Dictionary Elaboration (M-POLY-B)
	// Transform operators (BinOp, UnOp) to dictionary applications (DictApp)
	resolved := typeChecker.GetResolvedConstraints()
	elaboratedProg, err := elaborate.ElaborateWithDictionaries(unit.Core, resolved)
	if err != nil {
		return fmt.Errorf("dictionary elaboration error in %s: %w", modID, err)
	}
	unit.Core = elaboratedProg

	if cfg.DebugCompile {
		fmt.Fprintf(os.Stderr, "[DEBUG] Dictionary elaboration complete for module %s\n", modID)
	}

	// Phase 3.5: Monomorphization (v0.4.0)
	// Validate CoreTypeInfo before specialization (M-DX4)
	if err := ValidateCoreTypeInfo(unit.Core, typeChecker.CoreTI); err != nil {
		return fmt.Errorf("CoreTypeInfo validation failed in %s: %w", modID, err)
	}

	// Validate effects (M-SOUNDNESS)
	// Compare declared effects from Surface AST with required effects from Core AST
	if err := ValidateEffects(unit.Surface, unit.Core, typeChecker.CoreTI); err != nil {
		return fmt.Errorf("effect checking failed in %s: %w", modID, err)
	}

	// Validate package effect ceiling (M-PKG)
	// If this module belongs to a package, check its declared effects against the ceiling
	if err := validateEffectCeiling(unit.Surface, modID); err != nil {
		return err
	}

	// Perform monomorphization unless explicitly disabled
	if !cfg.DisableMonomorphization {
		specializer := NewSpecializer(&typeChecker.CoreTI)

		// Run specialization pass on module
		specializedProg, err := specializer.Specialize(unit.Core)
		if err != nil {
			return fmt.Errorf("monomorphization error in %s: %w", modID, err)
		}
		unit.Core = specializedProg

		stats := specializer.GetStats()
		if cfg.DebugCompile {
			fmt.Fprintf(os.Stderr, "[DEBUG] Monomorphization (module %s): %d specializations, %d skipped (cache: %d hits, %d misses)\n",
				modID, stats.TotalSpecializations, len(stats.SkippedFunctions),
				stats.CacheHits, stats.CacheMisses)

			// Show per-function breakdown if there were specializations
			if stats.TotalSpecializations > 0 {
				fmt.Fprintf(os.Stderr, "[DEBUG] Module %s per-function specializations:\n", modID)
				for fn, count := range stats.PerFunction {
					fmt.Fprintf(os.Stderr, "[DEBUG]   %s: %d\n", fn, count)
				}
			}

			// Show skip reasons if any
			if len(stats.SkippedFunctions) > 0 {
				fmt.Fprintf(os.Stderr, "[DEBUG] Module %s skipped functions:\n", modID)
				for _, skip := range stats.SkippedFunctions {
					fmt.Fprintf(os.Stderr, "[DEBUG]   %s: %s\n", skip.DefSym, skip.Reason)
				}
			}
		}
	} else if cfg.DebugCompile {
		fmt.Fprintf(os.Stderr, "[DEBUG] Monomorphization disabled for module %s\n", modID)
	}

	// Phase 3.5.5: Var Type Resolution (M-DX4 workaround) for this module
	if !cfg.DisableVarResolution {
		resolver := NewVarResolver(typeChecker.CoreTI)
		resolver.Resolve(unit.Core)

		if cfg.DebugCompile {
			fmt.Fprintf(os.Stderr, "[DEBUG] Var type resolution complete for module %s\n", modID)
		}
	}

	// Phase 3.6: Operator Lowering
	// Check if shim is forbidden in CI mode (before any other logic)
	if cfg.FailOnShim && cfg.ExperimentalBinopShim {
		return fmt.Errorf("CI_SHIM001: Operator shim usage detected but forbidden with --fail-on-shim in module %s", modID)
	}

	// If require lowering is set, we must lower regardless of shim flag
	// If shim is not enabled, we must lower
	if cfg.RequireLowering || !cfg.ExperimentalBinopShim {
		lowerer := NewOpLowerer(cfg.TypeEnv, typeChecker.CoreTI)
		// Pass resolved constraints from type checker to lowerer
		lowerer.SetResolvedConstraints(typeChecker.GetResolvedConstraints())

		// M-DX4: Enable telemetry if --debug-compile flag is set
		if cfg.DebugCompile {
			lowerer.SetEnableTelemetry(true)
		}

		var err error
		unit.Core, err = lowerer.Lower(unit.Core)
		if err != nil {
			return fmt.Errorf("lowering error in %s: %w", modID, err)
		}

		// M-DX4: Report telemetry if --debug-compile flag is set
		if cfg.DebugCompile && len(lowerer.GetTelemetry()) > 0 {
			fmt.Fprintf(os.Stderr, "[DEBUG] Lowering telemetry for module %s:\n", modID)
			reportLoweringTelemetry(lowerer.GetTelemetry())
		}

		// Guard A: Assert no operators remain
		// TODO: Re-enable after assert_builtins.go is fixed
		// if err := AssertNoOperators(unit.Core); err != nil {
		// 	return fmt.Errorf("in module %s: %w", modID, err)
		// }

		// Guard B: Assert only builtins appear for ops
		// TODO: Re-enable after assert_builtins.go is fixed
		// if err := AssertOnlyBuiltinsForOps(unit.Core); err != nil {
		// 	return fmt.Errorf("in module %s: %w", modID, err)
		// }

		unit.Core.Flags.Lowered = true

		// M-DEBUG-ERASURE: Erase Debug ghost effect in release mode
		if cfg.ReleaseMode {
			eraser := &DebugEraser{}
			unit.Core = eraser.Erase(unit.Core)
		}
	}

	return nil
}

// extractFuncSignature extracts parameter types and return type from a type env binding.
// Handles both TFunc (v1) and TFunc2 (v2) function types, and Scheme wrappers.
func extractFuncSignature(binding interface{}) (paramTypes []types.Type, retType types.Type) {
	// Unwrap Scheme if present
	typ := binding
	if scheme, ok := binding.(*types.Scheme); ok {
		typ = scheme.Type
	}
	switch fn := typ.(type) {
	case *types.TFunc2:
		return fn.Params, fn.Return
	}
	return nil, nil
}

// extractFuncParams walks Core declarations and extracts function name -> parameter names.
// This maps DeclMeta function names to their Lambda parameter names for contract env construction.
func extractFuncParams(prog *core.Program) map[string][]string {
	result := make(map[string][]string)
	for _, decl := range prog.Decls {
		extractFuncParamsFromExpr(decl, result)
	}
	return result
}

func extractFuncParamsFromExpr(expr core.CoreExpr, result map[string][]string) {
	switch e := expr.(type) {
	case *core.Let:
		// Top-level let: let funcName = Lambda{...} in ...
		if lam, ok := e.Value.(*core.Lambda); ok {
			result[e.Name] = lam.Params
		}
		if e.Body != nil {
			extractFuncParamsFromExpr(e.Body, result)
		}
	case *core.LetRec:
		for _, binding := range e.Bindings {
			if lam, ok := binding.Value.(*core.Lambda); ok {
				result[binding.Name] = lam.Params
			}
		}
		if e.Body != nil {
			extractFuncParamsFromExpr(e.Body, result)
		}
	}
}

// buildAndRegisterInterface builds the module interface and registers it with the linker.
// importedAliases are type aliases from this module's imports, used to embed transitive
// aliases referenced in exported function signatures (M-TYPE-ALIAS).
func buildAndRegisterInterface(
	unit *CompileUnit,
	modID string,
	moduleTypeEnv *types.TypeEnv,
	modLinker *link.ModuleLinker,
	importedAliases map[string]types.Type,
) error {
	// Convert pipeline constructors to iface constructors
	ifaceCtors := convertToIfaceConstructors(unit.Constructors)
	unitIface, err := iface.BuildInterfaceWithTypesAndConstructors(modID, unit.Core, moduleTypeEnv, unit.Surface, ifaceCtors)
	if err != nil {
		return fmt.Errorf("interface build error in %s: %w", modID, err)
	}

	// M-TYPE-ALIAS: Embed transitive type aliases referenced in exported function signatures.
	// When Package B exports getUsage() -> Result[Usage, string] and Usage is defined in
	// Package A, we need Usage in B's interface so Package C can resolve it transitively.
	if len(importedAliases) > 0 {
		embedTransitiveAliases(unitIface, importedAliases)
	}

	unit.Iface = unitIface
	modLinker.RegisterIface(unitIface)
	return nil
}

// embedTransitiveAliases adds imported type aliases to a module's interface when they are
// referenced in the module's exported function signatures or constructor types.
func embedTransitiveAliases(ifc *iface.Iface, importedAliases map[string]types.Type) {
	// Collect all TCon names referenced in exported signatures
	referenced := make(map[string]bool)
	for _, item := range ifc.Exports {
		if item.Type != nil {
			collectTConNames(item.Type.Type, referenced)
		}
	}
	for _, ctor := range ifc.Constructors {
		for _, ft := range ctor.FieldTypes {
			collectTConNames(ft, referenced)
		}
		collectTConNames(ctor.ResultType, referenced)
	}

	// Add imported aliases that are referenced but not already in the interface
	for name := range referenced {
		if _, exists := ifc.TypeAliases[name]; !exists {
			if alias, ok := importedAliases[name]; ok {
				ifc.AddTypeAlias(name, alias)
			}
		}
	}
}

// collectTConNames recursively collects all type constructor names from a type.
func collectTConNames(t types.Type, names map[string]bool) {
	if t == nil {
		return
	}
	switch ty := t.(type) {
	case *types.TCon:
		names[ty.Name] = true
	case *types.TApp:
		collectTConNames(ty.Constructor, names)
		for _, arg := range ty.Args {
			collectTConNames(arg, names)
		}
	case *types.TFunc2:
		for _, p := range ty.Params {
			collectTConNames(p, names)
		}
		collectTConNames(ty.Return, names)
	case *types.TRecord:
		for _, ft := range ty.Fields {
			collectTConNames(ft, names)
		}
		collectTConNames(ty.Row, names)
	case *types.TList:
		collectTConNames(ty.Element, names)
	case *types.TArray:
		collectTConNames(ty.Element, names)
	case *types.TMap:
		collectTConNames(ty.Key, names)
		collectTConNames(ty.Value, names)
	case *types.TTuple:
		for _, e := range ty.Elements {
			collectTConNames(e, names)
		}
	}
	// TVar2, TInt, TString, etc. — no TCon names to collect
}

// assembleModuleResult converts compiled units into LoadedModule entries
// for runtime execution (v0.2.0+).
func assembleModuleResult(
	compiledUnits map[string]*CompileUnit,
) map[string]*loader.LoadedModule {
	modules := make(map[string]*loader.LoadedModule)
	for modID, unit := range compiledUnits {
		// Skip $builtin - it's a virtual module
		if modID == "$builtin" {
			continue
		}

		loaded := &loader.LoadedModule{
			Path:    unit.ID,
			File:    unit.Surface,
			Core:    unit.Core,
			Iface:   unit.Iface,
			CoreTI:  unit.CoreTI, // M-CAPABILITY-BUDGETS: Pass type info for runtime budget enforcement
			Imports: []string{},
		}

		// Extract import paths from AST
		if unit.Surface != nil && len(unit.Surface.Imports) > 0 {
			for _, imp := range unit.Surface.Imports {
				loaded.Imports = append(loaded.Imports, imp.Path)
			}
		}

		// Initialize empty maps for compatibility with loader interface
		// (The actual export/type/constructor information is in the Iface)
		loaded.Exports = make(map[string]*ast.FuncDecl)
		loaded.Types = make(map[string]*ast.TypeDecl)
		loaded.Constructors = make(map[string]string)

		modules[modID] = loaded
	}
	return modules
}
