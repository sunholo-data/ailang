package pipeline

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/types"
)

// runModule runs the pipeline for a module with dependencies
func runModule(cfg Config, src Source) (Result, error) {
	// DEBUG: if cfg.TraceDefaulting { fmt.Printf("DEBUG: runModule called for %s\n", src.Filename) }
	result := Result{
		PhaseTimings: make(map[string]int64),
	}

	// Initialize environments if not provided
	if cfg.TypeEnv == nil {
		cfg.TypeEnv = types.NewTypeEnvWithBuiltins()
	}
	if cfg.InstEnv == nil {
		cfg.InstEnv = types.LoadBuiltinInstances()
	}
	if cfg.DictReg == nil {
		cfg.DictReg = types.NewDictionaryRegistry()
	}
	if cfg.EvalEnv == nil {
		cfg.EvalEnv = eval.NewEnvironment()
	}
	if cfg.Instances == nil {
		cfg.Instances = make(map[string]core.DictValue)
	}

	// Phase 1: Load module and dependencies
	start := time.Now()
	modLoader := loader.NewModuleLoader(".")
	modLoader.SetStrictSyntaxMode(cfg.StrictSyntaxMode)
	modules, err := modLoader.LoadAll([]string{src.Filename})
	if err != nil {
		return result, fmt.Errorf("module loading error: %w", err)
	}
	result.PhaseTimings["load"] = time.Since(start).Milliseconds()

	// Phase 2: Topological sort
	start = time.Now()
	modLinker := link.NewModuleLinker(modLoader)
	// Register $builtin as a first-class module
	link.RegisterBuiltinModule(modLinker)
	// Pass only the root module to TopoSort (dependencies will be discovered via DFS)
	rootCanonical := loader.CanonicalModuleID(src.Filename)
	sortedModules, err := modLinker.TopoSortFromRoot(rootCanonical, modules)
	if err != nil {
		return result, fmt.Errorf("dependency cycle: %w", err)
	}
	result.PhaseTimings["topo"] = time.Since(start).Milliseconds()

	// Phase 3: Two-phase compilation
	// Phase 3a: Build interfaces for all modules in dependency order
	// Log phase order for debugging
	var phaseLog []string
	for _, m := range sortedModules {
		phaseLog = append(phaseLog, string(m))
	}
	if cfg.TraceDefaulting {
		fmt.Printf("PHASE ORDER: ELAB+TC+IFACE: %v; EVAL: %s\n", phaseLog, src.Filename)
	}

	start = time.Now()
	compiledUnits := make(map[string]*CompileUnit)

	for _, modID := range sortedModules {
		mod := modules[string(modID)]
		unit := &CompileUnit{
			ID:      string(modID),
			Surface: mod.File,
		}

		// Validate module declaration matches canonical path (MOD010)
		if mod.File.Module != nil {
			canonicalID := loader.CanonicalModuleID(string(modID))
			// Exception: std/* modules bypass this check
			if !strings.HasPrefix(canonicalID, "std/") && mod.File.Module.Path != canonicalID {
				// Check if relaxation applies
				isTempPath := loader.IsTempPath(string(modID))
				shouldRelax := cfg.RelaxModules || isTempPath

				if shouldRelax {
					// Emit warning (once per path)
					if cfg.mod010WarnedPaths == nil {
						cfg.mod010WarnedPaths = make(map[string]bool)
					}
					if !cfg.mod010WarnedPaths[string(modID)] {
						cfg.mod010WarnedPaths[string(modID)] = true

						var reason string
						if isTempPath {
							reason = "temp-path"
						} else {
							reason = "relaxed"
						}
						warnMOD010Relaxed(mod.File.Module.Path, canonicalID, reason)
					}
				} else {
					// Strict mode: emit error with suggestions
					return result, fmt.Errorf("MOD010: module declaration '%s' doesn't match canonical path '%s'\nSuggestions:\n  1. Rename module to: module %s\n  2. Move file to: %s.ail\n  3. For temp/scratch files: use --relax-modules or AILANG_RELAX_MODULES=1",
						mod.File.Module.Path, canonicalID, canonicalID, mod.File.Module.Path)
				}
			}
		}

		// Build external environment from already-compiled dependencies
		externalTypes := make(map[string]*types.Scheme)
		globalRefs := make(map[string]core.GlobalRef)
		importedTypeAliases := make(map[string]types.Type)  // M-FIX-RECORD-UPDATE: Collect type aliases from imports
		importedCtorTypes := make(map[string]string)        // M-TAPP-FIX: Track imported constructor → ADT type
		importedADTTypeParams := make(map[string]int)       // M-TAPP-FIX: Track imported ADT → type param count

		// Always include $builtin module exports (available to all modules)
		if builtinIface := modLinker.GetIface("$builtin"); builtinIface != nil {
			for name, item := range builtinIface.Exports {
				// Add with qualified key (for explicit $builtin.name references)
				key := fmt.Sprintf("%s.%s", item.Ref.Module, item.Ref.Name)
				externalTypes[key] = item.Type

				// CRITICAL FIX: Also add with simple name so stdlib can reference _io_print directly
				// This preserves the effect row from the spec registry
				externalTypes[name] = item.Type

				globalRefs[name] = item.Ref
			}
		}

		// Get imports for this module
		if len(mod.File.Imports) > 0 {
			for _, imp := range mod.File.Imports {
				// Get the interface of the imported module
				depIface := modLinker.GetIface(imp.Path)
				if depIface == nil {
					if cfg.TraceDefaulting {
						fmt.Printf("WARNING: No interface for module %s (importing from %s)\n", imp.Path, modID)
					}
					continue
				}
				if len(imp.Symbols) > 0 {
					// Selective import
					for _, sym := range imp.Symbols {
						found := false

						// Determine the name to bind (use alias if present)
						bindName := sym
						if imp.SymbolAliases != nil {
							if alias, ok := imp.SymbolAliases[sym]; ok {
								bindName = alias
							}
						}

						// Try to import as a regular export (function/value)
						if item, ok := depIface.GetExport(sym); ok {
							key := fmt.Sprintf("%s.%s", item.Ref.Module, item.Ref.Name)
							externalTypes[key] = item.Type
							globalRefs[bindName] = item.Ref // Use alias name for binding
							if cfg.TraceDefaulting {
								fmt.Printf("  Import value %s as %s -> %s (%s)\n", sym, bindName, key, item.Type)
							}
							found = true
						}

						// Try to import as a type name
						if typ, ok := depIface.GetType(sym); ok {
							// Type names don't need to be added to externalTypes/globalRefs for now
							// They're handled by the type checker
							if cfg.TraceDefaulting {
								fmt.Printf("  Import type %s (arity %d)\n", typ.Name, typ.Arity)
							}
							// M-FIX-RECORD-UPDATE: Also import type alias if present
							// This enables cross-module record update syntax
							if alias, hasAlias := depIface.GetTypeAlias(sym); hasAlias {
								importedTypeAliases[sym] = alias
								if cfg.TraceDefaulting {
									fmt.Printf("  Import type alias %s -> %s\n", sym, alias)
								}
							}
							found = true
						}

						// Try to import as a constructor
						// DEBUG: fmt.Printf("DEBUG: Checking if %s is a constructor in %s (has %d constructors)...\n", sym, imp.Path, len(depIface.Constructors))
						for range depIface.Constructors {
							// DEBUG: fmt.Printf("DEBUG:   Constructor %s in interface\n", k)
						}
						if ctor, ok := depIface.GetConstructor(sym); ok {
							// Constructors are added to global environment
							// They're factory functions from $adt module
							factoryName := fmt.Sprintf("make_%s_%s", ctor.TypeName, ctor.CtorName)
							key := fmt.Sprintf("$adt.%s", factoryName)

							globalRefs[sym] = core.GlobalRef{
								Module: "$adt",
								Name:   factoryName,
							}

							// CRITICAL FIX: Also add to externalTypes so type checker knows the signature
							// Build the factory type scheme from the constructor info
							var factoryType types.Type
							if ctor.Arity == 0 {
								// Nullary constructor: just the result type
								factoryType = ctor.ResultType
							} else {
								// Constructor with fields: FieldTypes -> ResultType
								factoryType = &types.TFunc2{
									Params:    ctor.FieldTypes,
									EffectRow: nil, // Pure constructor
									Return:    ctor.ResultType,
								}
							}

							// Extract type variables from result type for polymorphism
							var typeVars []string
							if ctor.ResultType != nil {
								// Extract type vars from result type (e.g., Option[a] -> ["a"])
								typeVars = extractTypeVarsFromType(ctor.ResultType)
							}

							externalTypes[key] = &types.Scheme{
								TypeVars: typeVars,
								Type:     factoryType,
							}

							// DEBUG: fmt.Printf("DEBUG: Import constructor %s -> %s with type scheme (vars: %v)\n", sym, key, typeVars)
							if cfg.TraceDefaulting {
								fmt.Printf("  Import constructor %s -> %s\n", sym, key)
							}

							// M-TAPP-FIX: Track imported constructor for pattern matching type inference
							importedCtorTypes[sym] = ctor.TypeName

							// M-TAPP-FIX: Derive type param count from ResultType
							// If ResultType is TApp, count the args; otherwise 0
							if _, exists := importedADTTypeParams[ctor.TypeName]; !exists {
								paramCount := 0
								if tapp, ok := ctor.ResultType.(*types.TApp); ok {
									paramCount = len(tapp.Args)
								}
								importedADTTypeParams[ctor.TypeName] = paramCount
							}

							found = true
						}
						// No else needed - if constructor not found, we continue searching

						if !found && cfg.TraceDefaulting {
							fmt.Printf("  Symbol %s not found in %s\n", sym, imp.Path)
						}
					}
				}

				// Handle module alias: import std/list as List
				// Add all exports with qualified names (List.map, List.filter, etc.)
				if imp.ModuleAlias != "" {
					for name, item := range depIface.Exports {
						qualifiedName := fmt.Sprintf("%s.%s", imp.ModuleAlias, name)
						key := fmt.Sprintf("%s.%s", item.Ref.Module, item.Ref.Name)
						externalTypes[key] = item.Type
						globalRefs[qualifiedName] = item.Ref
						if cfg.TraceDefaulting {
							fmt.Printf("  Module alias %s.%s -> %s\n", imp.ModuleAlias, name, key)
						}
					}
				}
			}
		}

		// Elaborate to Core
		elaborator := elaborate.NewElaboratorWithPath(string(modID))
		elaborator.SetGlobalEnv(globalRefs)
		// Share the pipeline's module loader with the elaborator
		// CRITICAL: The elaborator creates its own loader with filepath.Dir(modID) as basePath
		// which would be wrong for subdirectory modules (e.g., "sim/world" -> basePath "sim")
		// By sharing the pipeline's loader (basePath "."), we ensure correct resolution
		elaborator.SetModuleLoader(modLoader)
		// Add builtins to global environment so they can be referenced
		elaborator.AddBuiltinsToGlobalEnv()

		unit.Core, err = elaborator.ElaborateFile(mod.File)
		if err != nil {
			// Preserve structured error reports without wrapping
			return result, err
		}

		// Collect exhaustiveness warnings
		warnings := elaborator.GetWarnings()
		result.Warnings = append(result.Warnings, warnings...)

		// Extract constructors from elaborator and store in CompileUnit
		unit.Constructors = convertConstructors(elaborator.GetConstructors())

		// Add $adt factory types for this module's constructors to externalTypes
		// This allows the type checker to know about constructor factories
		for ctorName, ctorInfo := range unit.Constructors {
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

			// For constructor fields, use the same type vars for the first TypeParamCount fields
			// (This assumes common case where field types are the ADT's type params)
			// Additional fields get fresh type vars
			for i := 0; i < ctorInfo.Arity; i++ {
				if i < ctorInfo.TypeParamCount {
					// Use the ADT type var
					paramTypes = append(paramTypes, adtTypeVars[i])
				} else {
					// Create additional type var for extra fields
					varName := fmt.Sprintf("a%d", i)
					typeVars = append(typeVars, varName)
					paramTypes = append(paramTypes, &types.TVar2{Name: varName, Kind: types.Star})
				}
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
		typeChecker.SetGlobalTypes(externalTypes)

		// M-DX25.4: Pass constructor → ADT type mappings to type checker
		// This enables correct type inference for pattern matching on ADTs
		ctorTypes := make(map[string]string)
		adtTypeParams := make(map[string]int) // M-TAPP-FIX: Track type param counts

		// M-TAPP-FIX: First add imported constructors (from depIface.GetConstructor)
		for ctorName, typeName := range importedCtorTypes {
			ctorTypes[ctorName] = typeName
		}
		for typeName, paramCount := range importedADTTypeParams {
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
		for name, target := range importedTypeAliases {
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

		// Type check ALL declarations in the module, accumulating types in moduleTypeEnv
		for i, decl := range unit.Core.Decls {
			// InferWithConstraints returns the updated env with new bindings
			_, moduleTypeEnv, _, _, err = typeChecker.InferWithConstraints(decl, moduleTypeEnv)
			if err != nil {
				return result, fmt.Errorf("type error in %s (decl %d): %w", modID, i, err)
			}
		}

		// Fill operator methods (resolve operators to type class methods)
		// This populates the Method field in resolved constraints before lowering
		for _, decl := range unit.Core.Decls {
			typeChecker.FillOperatorMethods(decl)
		}

		// M-DX23: Capture type info for codegen
		unit.CoreTI = typeChecker.CoreTI

		// Phase 3.4: Dictionary Elaboration (M-POLY-B)
		// Transform operators (BinOp, UnOp) to dictionary applications (DictApp)
		// This matches REPL behavior and is required for correct operator resolution
		resolved := typeChecker.GetResolvedConstraints()
		elaboratedProg, err := elaborate.ElaborateWithDictionaries(unit.Core, resolved)
		if err != nil {
			return result, fmt.Errorf("dictionary elaboration error in %s: %w", modID, err)
		}
		unit.Core = elaboratedProg

		if cfg.DebugCompile {
			fmt.Fprintf(os.Stderr, "[DEBUG] Dictionary elaboration complete for module %s\n", modID)
		}

		// Phase 3.5: Monomorphization (v0.4.0)
		// Validate CoreTypeInfo before specialization (M-DX4)
		if err := ValidateCoreTypeInfo(unit.Core, typeChecker.CoreTI); err != nil {
			return result, fmt.Errorf("CoreTypeInfo validation failed in %s: %w", modID, err)
		}

		// Validate effects (M-SOUNDNESS)
		// Compare declared effects from Surface AST with required effects from Core AST
		if err := ValidateEffects(unit.Surface, unit.Core, typeChecker.CoreTI); err != nil {
			return result, fmt.Errorf("effect checking failed in %s: %w", modID, err)
		}

		// Perform monomorphization unless explicitly disabled
		if !cfg.DisableMonomorphization {
			specializer := NewSpecializer(&typeChecker.CoreTI)

			// Run specialization pass on module
			specializedProg, err := specializer.Specialize(unit.Core)
			if err != nil {
				return result, fmt.Errorf("monomorphization error in %s: %w", modID, err)
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

		// Phase 3.6: Operator Lowering
		// Phase 3.5.5: Var Type Resolution (M-DX4 workaround) for this module
		if !cfg.DisableVarResolution {
			resolver := NewVarResolver(typeChecker.CoreTI)
			resolver.Resolve(unit.Core)

			if cfg.DebugCompile {
				fmt.Fprintf(os.Stderr, "[DEBUG] Var type resolution complete for module %s\n", modID)
			}
		}

		// Check if shim is forbidden in CI mode (before any other logic)
		if cfg.FailOnShim && cfg.ExperimentalBinopShim {
			return result, fmt.Errorf("CI_SHIM001: Operator shim usage detected but forbidden with --fail-on-shim in module %s", modID)
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

			unit.Core, err = lowerer.Lower(unit.Core)
			if err != nil {
				return result, fmt.Errorf("lowering error in %s: %w", modID, err)
			}

			// M-DX4: Report telemetry if --debug-compile flag is set
			if cfg.DebugCompile && len(lowerer.GetTelemetry()) > 0 {
				fmt.Fprintf(os.Stderr, "[DEBUG] Lowering telemetry for module %s:\n", modID)
				reportLoweringTelemetry(lowerer.GetTelemetry())
			}

			// Guard A: Assert no operators remain
			// TODO: Re-enable after assert_builtins.go is fixed
			// if err := AssertNoOperators(unit.Core); err != nil {
			// 	return result, fmt.Errorf("in module %s: %w", modID, err)
			// }

			// Guard B: Assert only builtins appear for ops
			// TODO: Re-enable after assert_builtins.go is fixed
			// if err := AssertOnlyBuiltinsForOps(unit.Core); err != nil {
			// 	return result, fmt.Errorf("in module %s: %w", modID, err)
			// }

			unit.Core.Flags.Lowered = true
		}

		// Build and register interface (using module-local type environment)
		// Convert pipeline constructors to iface constructors
		ifaceCtors := convertToIfaceConstructors(unit.Constructors)
		unitIface, err := iface.BuildInterfaceWithTypesAndConstructors(string(modID), unit.Core, moduleTypeEnv, unit.Surface, ifaceCtors)
		if err != nil {
			return result, fmt.Errorf("interface build error in %s: %w", modID, err)
		}

		unit.Iface = unitIface
		modLinker.RegisterIface(unitIface)

		compiledUnits[string(modID)] = unit
	}

	// Register $adt module after all modules are loaded and their interfaces are built
	// This allows $adt to collect all constructors from all loaded modules
	link.RegisterAdtModule(modLinker)

	result.PhaseTimings["compile"] = time.Since(start).Milliseconds()

	// Phase 3b: Register compiled modules with resolver for on-demand evaluation
	resolver := modLinker.Resolver()
	for modID, unit := range compiledUnits {
		resolver.RegisterCompiledModule(modID, unit)
	}

	// Wire builtin lookup if provided (v0.2.0 hotfix)
	if cfg.GlobalResolver != nil {
		// Extract builtin lookup capability from the provided resolver
		// We assume GlobalResolver supports builtin lookups via the same interface
		resolver.SetBuiltinLookup(func(name string) (eval.Value, bool) {
			ref := core.GlobalRef{Module: "$builtin", Name: name}
			val, err := cfg.GlobalResolver.ResolveValue(ref)
			if err != nil || val == nil {
				return nil, false
			}
			return val, true
		})
	}

	// Phase 4: Evaluate the root module ONLY
	// Assert: Only evaluate root, after all interfaces built
	if cfg.TraceDefaulting {
		fmt.Printf("PHASE: Evaluating root module: %s\n", src.Filename)
	}

	start = time.Now()
	// Use canonical ID to look up root (already computed above)
	rootUnit := compiledUnits[rootCanonical]
	if rootUnit == nil {
		// Try with original filename if canonical lookup fails
		rootUnit = compiledUnits[src.Filename]
		if rootUnit == nil {
			return result, fmt.Errorf("root module not found: %s (canonical: %s)", src.Filename, rootCanonical)
		}
	}

	// Create Core evaluator with global resolver
	coreEval := eval.NewCoreEvaluator()
	coreEval.SetGlobalResolver(resolver)
	// Pass experimental flag only if allowed
	if cfg.ExperimentalBinopShim && !cfg.RequireLowering && !cfg.FailOnShim {
		coreEval.SetExperimentalBinopShim(true)
	}

	// Guard B: Ensure program was lowered (unless using allowed shim)
	// TODO: Re-enable after assert_builtins.go is fixed
	// if cfg.RequireLowering || !cfg.ExperimentalBinopShim {
	// 	if err := AssertProgramLowered(rootUnit.Core); err != nil {
	// 		return result, err
	// 	}
	// }

	// Evaluate the root module ONLY in ModeEval (REPL)
	// In ModeCheck (CLI run), defer all execution to ModuleRuntime
	if cfg.Mode == ModeEval {
		if len(rootUnit.Core.Decls) > 0 {
			value, err := coreEval.Eval(rootUnit.Core.Decls[0])
			if err != nil {
				return result, fmt.Errorf("evaluation error: %w", err)
			}
			result.Value = value
		}
	}
	result.PhaseTimings["evaluate"] = time.Since(start).Milliseconds()

	// Store artifacts
	result.Artifacts.AST = rootUnit.Surface
	result.Artifacts.Core = rootUnit.Core
	result.Artifacts.CoreTI = rootUnit.CoreTI // M-DX23: Type info for codegen
	result.Interface = rootUnit.Iface         // Store module interface

	// Convert CompileUnits to LoadedModules for runtime execution (v0.2.0+)
	result.Modules = make(map[string]*loader.LoadedModule)
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

		result.Modules[modID] = loaded
	}

	return result, nil
}

// warnMOD010Relaxed emits a warning for module path mismatch in relaxed mode.
// The warning is printed to stderr with context about why it was relaxed.
func warnMOD010Relaxed(declaredPath, canonicalPath, reason string) {
	switch reason {
	case "temp-path":
		fmt.Fprintf(os.Stderr, "WARNING MOD010 (%s): module '%s' does not match canonical path '%s'\n  Auto-relaxed for temporary directory. For strict checking, move file outside temp directory.\n",
			reason, declaredPath, canonicalPath)
	case "relaxed":
		fmt.Fprintf(os.Stderr, "WARNING MOD010 (%s): module '%s' does not match canonical path '%s'\n  Running under --relax-modules; mismatch ignored. For strict checking, omit --relax-modules flag.\n",
			reason, declaredPath, canonicalPath)
	default:
		fmt.Fprintf(os.Stderr, "WARNING MOD010: module '%s' does not match canonical path '%s'\n",
			declaredPath, canonicalPath)
	}
}
