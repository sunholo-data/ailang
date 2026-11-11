package pipeline

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/link"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/types"
)

// moduleCompileResult holds the output of compiling a single module
// (type checking through lowering).
type moduleCompileResult struct {
	TypeChecker   *types.CoreTypeChecker
	DebugSink     *types.VerboseDebugSink
	Warnings      []elaborate.Warning
	MatchWarnings []*types.MatchWarning // TC_MATCH_003, also in Warnings; cached with the module
	ModuleTypeEnv *types.TypeEnv        // Accumulated type environment for interface building
}

// buildConstructorFactoryTypes creates $adt factory type schemes for a module's
// locally-defined constructors and adds them to the external types map.
// This allows the type checker to know about constructor factories.
func buildConstructorFactoryTypes(
	modID string,
	constructors map[string]*ConstructorInfo,
	externalTypes map[string]*types.Scheme,
) {
	for ctorName, ctorInfo := range constructors {
		factoryKey := core.ADTFactoryKey(modID, ctorInfo.TypeName, ctorName)

		// M-POLY-ADT / M-TAPP-FIX / M-TYPE-LIST-SOUND round 3: build the factory
		// scheme via the shared helper, which remaps declared param vars to the
		// ADT's result-type vars (t0, t1, ...) across the ENTIRE field type. This
		// keeps compound fields ([a], (a,a), Option[a]) tied to the result type
		// instead of leaking an orphaned free var that unified with anything.
		externalTypes[factoryKey] = types.BuildConstructorFactoryScheme(
			modID,
			ctorInfo.TypeName,
			ctorInfo.Arity,
			ctorInfo.TypeParamCount,
			ctorInfo.TypeParamNames,
			ctorInfo.InternalFieldTypes,
		)
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
	if err := checkBuiltinShadow(unit.Surface); err != nil {
		return nil, err
	}

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
	ctorModules := make(map[string]string) // M-V1-NOMINAL-TYPE-IDENTITY: ctor -> ADT's module
	adtTypeParams := make(map[string]int)  // M-TAPP-FIX: keyed by types.NominalKey(module, ADT)

	// M-TAPP-FIX: First add imported constructors (from depIface.GetConstructor)
	for ctorName, typeName := range imports.ImportedCtorTypes {
		ctorTypes[ctorName] = typeName
		ctorModules[ctorName] = imports.ImportedCtorModules[ctorName]
	}
	for adtKey, paramCount := range imports.ImportedADTTypeParams {
		adtTypeParams[adtKey] = paramCount
	}

	// Then add local constructors (may override imports if same name)
	for ctorName, ctorInfo := range unit.Constructors {
		ctorTypes[ctorName] = ctorInfo.TypeName
		ctorModules[ctorName] = modID
		// M-TAPP-FIX: Only set if not already set (first ctor wins, all should have same count)
		if adtKey := types.NominalKey(modID, ctorInfo.TypeName); true {
			if _, exists := adtTypeParams[adtKey]; !exists {
				adtTypeParams[adtKey] = ctorInfo.TypeParamCount
			}
		}
	}
	typeChecker.SetConstructorTypes(ctorTypes)
	typeChecker.SetConstructorModules(ctorModules)
	typeChecker.SetAmbiguousConstructors(imports.AmbiguousCtors)
	typeChecker.SetADTTypeParams(adtTypeParams) // M-TAPP-FIX
	// M-V1-MATCH-EXHAUSTIVENESS: every visible ADT's full constructor list.
	typeChecker.SetMatchUniverse(moduleMatchUniverse(imports.MatchUniverse, modID, unit.Constructors))

	// M-MATCH-XCHECK-ERROR-QUALITY: diagnostic-only registry of ALL transitively-
	// loaded constructors, so foreign-constructor errors can enumerate an ADT's
	// constructors even when the user hasn't imported that ADT's module. Kept
	// separate from ctorTypes above — does NOT affect scope or inference.
	typeChecker.SetDiagnosticConstructorTypes(imports.AllCtorTypes)

	// M-FIX-RECORD-UPDATE: Pass type aliases to type checker for expansion during unification
	// This enables `type NPC = { pos: Pos, name: string }` to work with record update
	// M-FIX-RECORD-UPDATE / M-XMOD-ALIAS-POLY: local and imported aliases (with
	// their params) expand during unification. M-V1-NOMINAL-TYPE-IDENTITY:
	// every key is types.NominalKey(defining module, name), so an alias expands
	// only its own module's TCon, and newtype ADTs stay nominal except against
	// a record (field access).
	aliasEnv := qualifiedAliasEnv(modID, imports, elaborator)
	for key, target := range aliasEnv.Bodies {
		typeChecker.RegisterTypeAlias(key, target)
	}
	for key, params := range aliasEnv.Params {
		typeChecker.RegisterTypeAliasParams(key, params)
	}
	for key := range aliasEnv.Newtypes {
		typeChecker.RegisterNewtypeADT(key)
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

	// M-TYPE-LIST-ELEMENT-SOUNDNESS: Pass let-binding type annotations to type checker
	// so `let xs: [string] = [42]` is actually type-checked (was dropped before).
	letAnnots := elaborator.GetLetTypeAnnotations()
	if len(letAnnots) > 0 {
		typeChecker.SetLetTypeAnnotations(letAnnots)
	}

	// M-CAPABILITY-BUDGETS: Pass full effect annotations with budgets to type checker
	// This preserves @limit=N budget annotations from function declarations through elaboration
	effectAnnotsFull := elaborator.GetEffectAnnotationsFull()
	if len(effectAnnotsFull) > 0 {
		typeChecker.SetEffectAnnotationsFull(effectAnnotsFull)
	}

	// M-DX19 + M-EQ-DERIVE-CONTAINERS: register `deriving (Eq)` instances and
	// require Eq of every field
	if err := registerDerivedEq(cfg, modID, elaborator, aliasEnv.Bodies); err != nil {
		return nil, fmt.Errorf("type error in %s: %w", loader.DeclaredModuleDisplay(modID, unit.Surface), err)
	}

	// #327 interim diagnostic: record this module's function names so the type
	// checker can tell the truth when a local function resolves as "undefined
	// variable" in a record-update field (see types.localResolutionHint). This
	// only enriches an already-failing error message; it changes no types.
	if unit.Surface != nil && len(unit.Surface.Funcs) > 0 {
		modFuncNames := make(map[string]bool, len(unit.Surface.Funcs))
		for _, fn := range unit.Surface.Funcs {
			if fn != nil {
				modFuncNames[fn.Name] = true
			}
		}
		typeChecker.SetModuleFuncNames(modFuncNames)
	}

	// Type check ALL declarations in the module, accumulating types in moduleTypeEnv.
	// Each decl's own binding keeps full generalization under the
	// integer-literal restriction (M-SOUND-NUMERIC-LITERALS).
	typeChecker.SetModuleDeclarationMode(true)
	for i, decl := range unit.Core.Decls {
		// InferWithConstraints returns the updated env with new bindings
		var err error
		_, moduleTypeEnv, _, _, err = typeChecker.InferWithConstraints(decl, moduleTypeEnv)
		if err != nil {
			return nil, fmt.Errorf("type error in %s (decl %d): %w", loader.DeclaredModuleDisplay(modID, unit.Surface), i, err)
		}
	}

	// M-PERF-DOCPARSE: Apply all deferred CoreTI substitutions in one pass
	// Must happen before FillOperatorMethods which reads CoreTI for operator resolution
	typeChecker.FinalizeSubstitutions()

	// M-TYPECHECK-NO-AUTO-UNWRAP-RESULT (v0.20.0): post-inference verification
	// of record-access sites against tagged-union receivers. CoreTI now has
	// the resolved receiver types (post-substitution). Catches the
	// `let r = step(); r.field` shape that the early gate in
	// inferRecordAccess can't catch (receiver is still a TVar at that point).
	if err := typeChecker.VerifyTaggedUnionFieldAccesses(); err != nil {
		return nil, fmt.Errorf("type error in %s: %w", loader.DeclaredModuleDisplay(modID, unit.Surface), err)
	}

	// M-V1-MATCH-EXHAUSTIVENESS: a match whose unguarded arms miss a value of
	// the scrutinee's type is an error (TC_MATCH_002) naming the missing
	// pattern; an arm no value can reach is a warning (TC_MATCH_003). Runs on
	// the finalised types, before any pass copies or rewrites matches.
	matchWarnings, err := typeChecker.VerifyMatchExhaustiveness(unit.Core.Decls...)
	if err != nil {
		return nil, err
	}

	// M-SECRET-EFFECT (M5): static information-flow-control enforcement.
	// A <secret> (or any <label>) value reaching a {not label} sink without an
	// intervening ! {Declassify} step is a compile-time error. This is a
	// surface-AST analysis that reads unit.Surface plus the imported exports'
	// IFC summaries (M-IFC-CROSS-MODULE, #1134: a library's sink refinements,
	// parameter labels and return labels bind its importers), and never injects
	// TLabelled types into CoreTI/codegen.
	// The same walk reports the labelled call sites the trace must redact
	// (M-TRACE-LABEL-AWARE); they are stamped on the final Core below.
	ifcErrs, traceSites := types.CheckModuleIFCWithImportsAndTraceSites(unit.Surface, imports.IFC)
	if len(ifcErrs) > 0 {
		return nil, fmt.Errorf("type error in %s: %w", loader.DeclaredModuleDisplay(modID, unit.Surface), ifcErrs[0])
	}

	// Fill operator methods (resolve operators to type class methods)
	// This populates the Method field in resolved constraints before lowering
	for _, decl := range unit.Core.Decls {
		typeChecker.FillOperatorMethods(decl)
	}

	// M-CONTRACT-OPLOWERING-FIX: Type-check contract expressions to populate CoreTI.
	// Without this, OpLowering hits CoreTI misses for operators in ensures/requires clauses,
	// causing comparison ops to be deferred as raw Intrinsic nodes that fail at runtime.
	var checkedContracts []core.CoreExpr
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
				// A contract's matches are checked like a body's. A contract
				// whose inference failed has no final types to check against
				// (its type errors are not reported either: pre-existing).
				if inferErr == nil {
					checkedContracts = append(checkedContracts, contract.Expr)
				}
				// Also fill operator methods for contract expressions
				typeChecker.FillOperatorMethods(contract.Expr)
			}
		}
	}

	// The contracts' matches (VerifyMatchExhaustiveness sorts its errors, so
	// the map order above does not decide which is reported first).
	if len(checkedContracts) > 0 {
		ws, err := typeChecker.VerifyMatchExhaustiveness(checkedContracts...)
		if err != nil {
			return nil, err
		}
		matchWarnings = append(matchWarnings, ws...)
	}

	// M-DX23: Capture type info for codegen
	unit.CoreTI = typeChecker.CoreTI

	// Run post-typecheck phases (dict elaboration, mono, var resolution, lowering)
	if err := runPostTypeCheckPhases(unit, modID, typeChecker, cfg); err != nil {
		return nil, err
	}
	if unmatched := core.MarkTraceRedactions(unit.Core, traceSites); cfg.DebugCompile && len(traceSites) > 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG] trace redaction for %s: %d labelled call sites, %d with no Core call %v\n", modID, len(traceSites), len(unmatched), unmatched)
	}

	// TC_MATCH_003: unreachable match arms
	result.MatchWarnings = matchWarnings
	for _, w := range matchWarnings {
		result.Warnings = append(result.Warnings, w)
	}
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

	// Validate package effect ceiling (M-PKG, M-EFFECT-CEILING-REACHABLE).
	// Runs BEFORE the per-function effect check: the ceiling is computed from
	// the inferred types of every expression in the package's own modules, so
	// it does not depend on how the effect checker attributes a closure's
	// effects, and an authority breach is reported as one.
	if err := validateEffectCeiling(unit.Surface, unit.Core, typeChecker.CoreTI, modID); err != nil {
		return err
	}

	// Validate effects (M-SOUNDNESS)
	// Compare declared effects from Surface AST with required effects from Core AST
	if err := ValidateEffectsWithCalls(unit.Surface, unit.Core, typeChecker.CoreTI, typeChecker.ApplicationEffects, typeChecker.DeclaredLambdaEffectRow, typeChecker.EffectValueType); err != nil {
		return fmt.Errorf("effect checking failed in %s: %w", loader.DeclaredModuleDisplay(modID, unit.Surface), err)
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

	// M-SMT-INTERP-SHOW: drop the `show` the interpolation desugar inserts around
	// holes whose type makes it redundant (string) or trivially encodable (bool).
	// Runs AFTER monomorphization so a generic helper's string instantiation is
	// visible, and unconditionally so the DisableMonomorphization path is covered
	// too. See internal/pipeline/show_normalize.go.
	{
		normalizer := NewShowNormalizer(&typeChecker.CoreTI)
		normalized, err := normalizer.Normalize(unit.Core)
		if err != nil {
			return fmt.Errorf("show normalization failed in %s: %w", modID, err)
		}
		unit.Core = normalized
		if cfg.DebugCompile {
			fmt.Fprintf(os.Stderr, "[DEBUG] ShowNormalize (module %s): %d elided, %d rewritten, %d residue\n",
				modID, normalizer.Elided, normalizer.Rewritten, normalizer.Residue)
		}
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

		// M-DEBUG-SINK-STRUCTURED-LINES: inject call-site locations into
		// std/debug wrapper calls (always; erasure below sees the builtin shape)
		unit.Core = (&DebugLocationInjector{}).Inject(unit.Core)

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
// localAliases includes private dependencies; aliasParams preserves polymorphic heads.
func buildAndRegisterInterface(
	unit *CompileUnit,
	modID string,
	moduleTypeEnv *types.TypeEnv,
	modLinker *link.ModuleLinker,
	scope *types.TypeScope,
	aliases *aliasTables,
	derivedEq []string,
	ifcImports types.IFCImports,
) error {
	// Convert pipeline constructors to iface constructors
	ifaceCtors := convertToIfaceConstructors(unit.Constructors)
	// The export summaries see this module's imports, so a wrapper over an
	// imported secret producer exports that taint (transitive libraries, #1134).
	// The type scope qualifies every type name the interface re-reads from the
	// AST (constructor fields, alias bodies) exactly as the elaborator did.
	unitIface, err := iface.NewBuilder(modID, moduleTypeEnv).WithIFCImports(ifcImports.Sigs).WithTypeScope(scope).Build(unit.Core, ifaceCtors, unit.Surface)
	if err != nil {
		return fmt.Errorf("interface build error in %s: %w", modID, err)
	}

	closeInterfaceTypes(unitIface, aliases)

	// M-TYPE-ALIAS: Embed transitive type aliases referenced in exported function signatures.
	// When Package B exports getUsage() -> Result[Usage, string] and Usage is defined in
	// Package A, we need Usage in B's interface so Package C can resolve it transitively.
	embedTransitiveAliases(unitIface, aliases)

	// SetDerivedEq also finalizes the digest after closure and transitive embedding.
	if err := unitIface.SetDerivedEq(derivedEq); err != nil {
		return fmt.Errorf("interface build error in %s: %w", modID, err)
	}

	unit.Iface = unitIface
	modLinker.RegisterIface(unitIface)
	return nil
}

// embedTransitiveAliases adds imported type aliases to a module's interface when they are
// referenced in the module's exported function signatures or constructor types.
//
// M-V1-NOMINAL-TYPE-IDENTITY: an embedded alias is defined by ANOTHER module,
// so it is stored under its NominalKey ("module::name"); the interface's own
// aliases keep their bare names (ifaceAliasKey reads both).
func embedTransitiveAliases(ifc *iface.Iface, aliases *aliasTables) {
	// Collect all TCon keys referenced in exported signatures
	referenced := make(map[string]bool)
	for _, item := range ifc.Exports {
		if item.Type != nil {
			collectTConKeys(item.Type.Type, referenced)
		}
	}
	for _, ctor := range ifc.Constructors {
		for _, ft := range ctor.FieldTypes {
			collectTConKeys(ft, referenced)
		}
		collectTConKeys(ctor.ResultType, referenced)
	}

	// Add imported aliases that are referenced but not already in the interface
	for key := range referenced {
		module, _ := types.SplitNominalKey(key)
		if module == ifc.Module || module == "" {
			continue // the module's own aliases are already in the interface
		}
		if _, exists := ifc.TypeAliases[key]; exists {
			continue
		}
		if alias, ok := aliases.Bodies[key]; ok {
			ifc.AddTypeAlias(key, alias)
			if params := aliases.Params[key]; len(params) > 0 {
				ifc.AddTypeAliasParams(key, params)
			}
		}
	}
}

// collectTConKeys collects the NominalKey of every type constructor in t.
func collectTConKeys(t types.Type, keys map[string]bool) {
	collectTConRefs(t, func(c *types.TCon) { keys[c.KeyOf()] = true })
}

// collectTConRefs calls visit on every type constructor of a type.
func collectTConRefs(t types.Type, visit func(*types.TCon)) {
	if t == nil {
		return
	}
	switch ty := t.(type) {
	case *types.TCon:
		visit(ty)
	case *types.TApp:
		collectTConRefs(ty.Constructor, visit)
		for _, arg := range ty.Args {
			collectTConRefs(arg, visit)
		}
	case *types.TFunc2:
		for _, p := range ty.Params {
			collectTConRefs(p, visit)
		}
		collectTConRefs(ty.Return, visit)
	case *types.TRecord:
		for _, ft := range ty.Fields {
			collectTConRefs(ft, visit)
		}
		collectTConRefs(ty.Row, visit)
	case *types.TList:
		collectTConRefs(ty.Element, visit)
	case *types.TArray:
		collectTConRefs(ty.Element, visit)
	case *types.TMap:
		collectTConRefs(ty.Key, visit)
		collectTConRefs(ty.Value, visit)
	case *types.TTuple:
		for _, e := range ty.Elements {
			collectTConRefs(e, visit)
		}
	}
	// TVar2, TInt, TString, etc. — no TCon names to collect
}

// collectTConNames recursively collects all type constructor names from a type.
func collectTConNames(t types.Type, names map[string]bool) {
	collectTConRefs(t, func(c *types.TCon) { names[c.Name] = true })
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
