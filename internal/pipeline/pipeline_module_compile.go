package pipeline

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/ast"
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
	}

	return nil
}

// buildAndRegisterInterface builds the module interface and registers it with the linker.
func buildAndRegisterInterface(
	unit *CompileUnit,
	modID string,
	moduleTypeEnv *types.TypeEnv,
	modLinker *link.ModuleLinker,
) error {
	// Convert pipeline constructors to iface constructors
	ifaceCtors := convertToIfaceConstructors(unit.Constructors)
	unitIface, err := iface.BuildInterfaceWithTypesAndConstructors(modID, unit.Core, moduleTypeEnv, unit.Surface, ifaceCtors)
	if err != nil {
		return fmt.Errorf("interface build error in %s: %w", modID, err)
	}

	unit.Iface = unitIface
	modLinker.RegisterIface(unitIface)
	return nil
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
