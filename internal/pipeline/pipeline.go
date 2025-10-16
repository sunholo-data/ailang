// Package pipeline provides a unified compilation pipeline for AILANG
package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"

	// "github.com/sunholo/ailang/internal/errors" // TODO: Use structured errors
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/linked"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/parser"
	_ "github.com/sunholo/ailang/internal/typedast" // For type checker return value
	"github.com/sunholo/ailang/internal/types"
)

// Mode determines pipeline execution behavior
type Mode int

const (
	ModeCheck Mode = iota // Parse + type + elaborate + build interface (NO evaluation)
	ModeEval              // Include evaluation (REPL only)
)

// Config contains pipeline configuration options
type Config struct {
	Mode                  Mode                  // Execution mode (Check or Eval)
	JSON                  bool                  // Output JSON format
	Compact               bool                  // Use compact JSON
	DumpCore              bool                  // Show Core AST
	DumpCoreLowered       bool                  // Show Core after lowering
	DumpTyped             bool                  // Show Typed AST
	TraceDefaulting       bool                  // Trace type defaulting
	DryLink               bool                  // Show linking without eval
	RequireLowering       bool                  // Fail if operators not lowered
	ExperimentalBinopShim bool                  // Feature flag for operator shim
	FailOnShim            bool                  // Fail if shim would be used (CI mode)
	TrackInstantiations   bool                  // Track polymorphic type instantiations
	LedgerHook            func(decision string) // Optional decision hook

	// Environment from REPL (optional)
	TypeEnv   *types.TypeEnv
	InstEnv   *types.InstanceEnv
	DictReg   *types.DictionaryRegistry
	Instances map[string]core.DictValue
	EvalEnv   *eval.Environment

	// Global resolver for non-module evaluation (v0.2.0 hotfix)
	GlobalResolver eval.GlobalResolver
}

// Source represents input source
type Source struct {
	Code     string
	Filename string
	IsREPL   bool
	REPLNum  int // REPL snippet number
}

// Artifacts contains intermediate representations
type Artifacts struct {
	AST    *ast.File
	Core   *core.Program
	Typed  interface{} // TODO: Add typed AST when available
	Linked interface{} // TODO: Add linked program when available
}

// Result contains pipeline output
type Result struct {
	Value          eval.Value
	Type           types.Type
	Constraints    []types.Constraint
	Errors         []error                            // TODO: Use structured errors
	Warnings       []*elaborate.ExhaustivenessWarning // Exhaustiveness warnings
	Artifacts      Artifacts
	Interface      *iface.Iface                    // Module interface (for modules only)
	Modules        map[string]*loader.LoadedModule // Loaded modules with Core (for module execution)
	EnvLockDigest  string
	PhaseTimings   map[string]int64       // milliseconds
	Instantiations map[string]interface{} // Polymorphic instantiation tracking
}

// Run executes the full compilation pipeline
func Run(cfg Config, src Source) (Result, error) {
	// For simple expressions/REPL, use the original single-file pipeline
	if src.IsREPL || src.Filename == "" || src.Filename == "<repl>" {
		// DEBUG: if cfg.TraceDefaulting { fmt.Printf("DEBUG: Using runSingle for %s\n", src.Filename) }
		return runSingle(cfg, src)
	}

	// For files with potential imports, use the module pipeline
	// DEBUG: if cfg.TraceDefaulting { fmt.Printf("DEBUG: Using runModule for %s\n", src.Filename) }
	return runModule(cfg, src)
}

// runSingle runs the pipeline for a single file/expression (REPL mode)
func runSingle(cfg Config, src Source) (Result, error) {
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

	// Phase 1: Parse
	start := time.Now()
	l := lexer.New(src.Code, src.Filename)
	p := parser.New(l)

	var astFile *ast.File
	if src.IsREPL {
		// For REPL, wrap expression in synthetic module
		program := p.Parse()
		if len(p.Errors()) > 0 {
			return result, convertParserErrors(p.Errors())
		}

		// Create synthetic module wrapper with session ID
		moduleName := fmt.Sprintf("_repl/%d", src.REPLNum)
		astFile = &ast.File{
			Module: &ast.ModuleDecl{
				Path: moduleName,
				Pos:  ast.Pos{Line: 1, Column: 1},
			},
			Statements: []ast.Node{},
		}
		// Convert program to statements
		if program.Module != nil {
			astFile.Statements = append(astFile.Statements, program.Module.Decls...)
		}
	} else {
		// For files, parse as complete file
		astFile = p.ParseFile()
		if len(p.Errors()) > 0 {
			return result, convertParserErrors(p.Errors())
		}
	}

	result.Artifacts.AST = astFile
	result.PhaseTimings["parse"] = time.Since(start).Milliseconds()

	// Phase 2: Elaborate to Core
	start = time.Now()
	var elaborator *elaborate.Elaborator
	if src.Filename != "" && src.Filename != "<repl>" {
		elaborator = elaborate.NewElaboratorWithPath(src.Filename)
	} else {
		elaborator = elaborate.NewElaborator()
	}
	// Add builtins to global environment so they can be referenced
	elaborator.AddBuiltinsToGlobalEnv()
	coreProg, err := elaborator.ElaborateFile(astFile)
	if err != nil {
		return result, fmt.Errorf("elaboration error: %w", err)
	}

	// Collect exhaustiveness warnings
	result.Warnings = elaborator.GetWarnings()

	result.Artifacts.Core = coreProg
	result.PhaseTimings["elaborate"] = time.Since(start).Milliseconds()

	if cfg.DumpCore { //nolint:staticcheck // Flag for caller to display Core AST
		// Core will be displayed by caller
	}

	// Phase 3: Type Check
	start = time.Now()
	typeChecker := types.NewCoreTypeCheckerWithInstances(cfg.InstEnv)
	typeChecker.EnableTraceDefaulting(cfg.TraceDefaulting)
	if cfg.TrackInstantiations {
		typeChecker.EnableInstantiationTracking()
	}

	// For REPL, extract first declaration as expression
	var coreExpr core.CoreExpr
	if src.IsREPL && len(coreProg.Decls) > 0 {
		coreExpr = coreProg.Decls[0]
	} else if len(coreProg.Decls) > 0 {
		// For files, type check the whole program
		// TODO: Implement full program type checking
		coreExpr = coreProg.Decls[0]
	} else {
		return result, fmt.Errorf("empty program")
	}

	typedNode, _, qualType, constraints, err := typeChecker.InferWithConstraints(coreExpr, cfg.TypeEnv)
	if err != nil {
		return result, fmt.Errorf("type error: %w", err)
	}

	result.Type = qualType
	result.Constraints = constraints
	result.PhaseTimings["typecheck"] = time.Since(start).Milliseconds()

	// Capture instantiation tracking if enabled
	if cfg.TrackInstantiations {
		result.Instantiations = typeChecker.DumpInstantiations()
	}

	// Phase 3.5: Operator Lowering
	start = time.Now()

	// Check if shim is forbidden in CI mode (before any other logic)
	if cfg.FailOnShim && cfg.ExperimentalBinopShim {
		return result, fmt.Errorf("CI_SHIM001: Operator shim usage detected but forbidden with --fail-on-shim")
	}

	// If require lowering is set, we must lower regardless of shim flag
	// If shim is not enabled, we must lower
	if cfg.RequireLowering || !cfg.ExperimentalBinopShim {
		lowerer := NewOpLowerer(cfg.TypeEnv)
		loweredProg, err := lowerer.Lower(coreProg)
		if err != nil {
			return result, fmt.Errorf("lowering error: %w", err)
		}

		// Guard A: Assert no operators remain
		// TODO: Re-enable after assert_builtins.go is fixed
		// if err := AssertNoOperators(loweredProg); err != nil {
		// 	return result, err
		// }

		// Guard B: Assert only builtins appear for ops
		// TODO: Re-enable after assert_builtins.go is fixed
		// if err := AssertOnlyBuiltinsForOps(loweredProg); err != nil {
		// 	return result, err
		// }

		loweredProg.Flags.Lowered = true
		coreProg = loweredProg

		if cfg.DumpCoreLowered { //nolint:staticcheck // Flag for caller to display lowered Core
			// Core will be displayed by caller
		}
	}
	result.PhaseTimings["lower"] = time.Since(start).Milliseconds()

	// Phase 4: Dictionary Elaboration
	start = time.Now()
	// TODO: Implement proper dictionary elaboration
	// For now, just use the typed node as-is
	elaborated := coreExpr
	_ = typedNode
	_ = constraints
	result.PhaseTimings["dict_elab"] = time.Since(start).Milliseconds()

	// Phase 5: ANF Verification
	start = time.Now()
	// TODO: Implement ANF verification
	_ = elaborated
	result.PhaseTimings["anf_verify"] = time.Since(start).Milliseconds()

	// Phase 6: Link
	start = time.Now()
	linker := linked.NewLinker()

	// Register runtime dictionaries
	for key, dict := range cfg.Instances {
		cfg.DictReg.RegisterInstance(key, dict)
	}

	// Linking expects CoreExpr, but we have core.Expr
	// TODO: Unify these types
	linkedExpr := elaborated
	result.PhaseTimings["link"] = time.Since(start).Milliseconds()

	if cfg.DryLink {
		// Skip evaluation for dry link
		return result, nil
	}

	// Phase 7: Evaluate
	start = time.Now()
	// Use Core evaluator for proper evaluation
	coreEval := eval.NewCoreEvaluator()
	// Set global resolver if provided (v0.2.0 hotfix for builtins)
	if cfg.GlobalResolver != nil {
		coreEval.SetGlobalResolver(cfg.GlobalResolver)
	}
	// Set experimental flag only if allowed
	if cfg.ExperimentalBinopShim && !cfg.RequireLowering && !cfg.FailOnShim {
		coreEval.SetExperimentalBinopShim(true)
	}

	// Guard B: Ensure program was lowered (unless using allowed shim)
	// TODO: Re-enable after assert_builtins.go is fixed
	// if cfg.RequireLowering || !cfg.ExperimentalBinopShim {
	// 	if err := AssertProgramLowered(coreProg); err != nil {
	// 		return result, err
	// 	}
	// }

	// Evaluate the program ONLY in ModeEval (REPL)
	if cfg.Mode == ModeEval {
		if len(coreProg.Decls) > 0 {
			value, err := coreEval.Eval(coreProg.Decls[0])
			if err != nil {
				return result, fmt.Errorf("runtime error: %w", err)
			}
			result.Value = value
		}
	}

	_ = linkedExpr
	_ = linker
	result.PhaseTimings["evaluate"] = time.Since(start).Milliseconds()

	// Calculate environment digest for determinism
	// TODO: Implement proper digest calculation
	result.EnvLockDigest = "TODO"

	return result, nil
}

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
				return result, fmt.Errorf("MOD010: module declaration '%s' doesn't match canonical path '%s'\nSuggestions:\n  1. Rename module to: module %s\n  2. Move file to: %s.ail",
					mod.File.Module.Path, canonicalID, canonicalID, mod.File.Module.Path)
			}
		}

		// Build external environment from already-compiled dependencies
		externalTypes := make(map[string]*types.Scheme)
		globalRefs := make(map[string]core.GlobalRef)

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

						// Try to import as a regular export (function/value)
						if item, ok := depIface.GetExport(sym); ok {
							key := fmt.Sprintf("%s.%s", item.Ref.Module, item.Ref.Name)
							externalTypes[key] = item.Type
							globalRefs[sym] = item.Ref
							if cfg.TraceDefaulting {
								fmt.Printf("  Import value %s -> %s (%s)\n", sym, key, item.Type)
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
							found = true
						}
						// No else needed - if constructor not found, we continue searching

						if !found && cfg.TraceDefaulting {
							fmt.Printf("  Symbol %s not found in %s\n", sym, imp.Path)
						}
					}
				}
			}
		}

		// Elaborate to Core
		elaborator := elaborate.NewElaboratorWithPath(string(modID))
		elaborator.SetGlobalEnv(globalRefs)
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

			// Build factory type: a0 -> a1 -> ... -> TypeName
			// Use TVar2 (new type system) for type variables with Star kind
			var typeVars []string
			var paramTypes []types.Type
			for i := 0; i < ctorInfo.Arity; i++ {
				varName := fmt.Sprintf("a%d", i)
				typeVars = append(typeVars, varName)
				paramTypes = append(paramTypes, &types.TVar2{Name: varName, Kind: types.Star})
			}

			// Result type - monomorphic for now (M-P3 limitation)
			// Full polymorphic ADTs (Option[Int]) will require type application support in unifier
			resultType := &types.TCon{Name: ctorInfo.TypeName}

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

		typeChecker := types.NewCoreTypeCheckerWithInstances(cfg.InstEnv)
		typeChecker.EnableTraceDefaulting(cfg.TraceDefaulting)
		if cfg.TrackInstantiations {
			typeChecker.EnableInstantiationTracking()
		}
		typeChecker.SetGlobalTypes(externalTypes)

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

		// Phase 3.5: Operator Lowering
		// Check if shim is forbidden in CI mode (before any other logic)
		if cfg.FailOnShim && cfg.ExperimentalBinopShim {
			return result, fmt.Errorf("CI_SHIM001: Operator shim usage detected but forbidden with --fail-on-shim in module %s", modID)
		}

		// If require lowering is set, we must lower regardless of shim flag
		// If shim is not enabled, we must lower
		if cfg.RequireLowering || !cfg.ExperimentalBinopShim {
			lowerer := NewOpLowerer(cfg.TypeEnv)
			// Pass resolved constraints from type checker to lowerer
			lowerer.SetResolvedConstraints(typeChecker.GetResolvedConstraints())
			unit.Core, err = lowerer.Lower(unit.Core)
			if err != nil {
				return result, fmt.Errorf("lowering error in %s: %w", modID, err)
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
	result.Interface = rootUnit.Iface // Store module interface

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

// convertParserErrors converts parser errors to structured AILANG errors
func convertParserErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	// For now, return the first error
	// TODO: Return all errors with proper structure
	return errs[0]
}

// convertConstructors converts elaborator constructors to pipeline ConstructorInfo
func convertConstructors(elabCtors map[string]*elaborate.ConstructorInfo) map[string]*ConstructorInfo {
	ctors := make(map[string]*ConstructorInfo)
	for name, elabCtor := range elabCtors {
		ctors[name] = &ConstructorInfo{
			TypeName:   elabCtor.TypeName,
			CtorName:   elabCtor.CtorName,
			FieldTypes: nil, // We don't have AST types here, will infer from Core
			Arity:      elabCtor.Arity,
		}
	}
	return ctors
}

// convertToIfaceConstructors converts pipeline constructors to iface constructors
func convertToIfaceConstructors(pipeCtors map[string]*ConstructorInfo) map[string]*iface.ConstructorInfo {
	if pipeCtors == nil {
		return nil
	}
	ifaceCtors := make(map[string]*iface.ConstructorInfo)
	for name, pipeCtor := range pipeCtors {
		ifaceCtors[name] = &iface.ConstructorInfo{
			TypeName: pipeCtor.TypeName,
			CtorName: pipeCtor.CtorName,
			Arity:    pipeCtor.Arity,
		}
	}
	return ifaceCtors
}

// extractTypeVarsFromType extracts type variable names from a type
// For example: Option[a] -> ["a"], Result[t, e] -> ["t", "e"]
func extractTypeVarsFromType(typ types.Type) []string {
	var vars []string
	seen := make(map[string]bool)

	var extract func(types.Type)
	extract = func(t types.Type) {
		if t == nil {
			return
		}
		switch typ := t.(type) {
		case *types.TVar2:
			if !seen[typ.Name] {
				vars = append(vars, typ.Name)
				seen[typ.Name] = true
			}
		case *types.TApp:
			extract(typ.Constructor)
			for _, arg := range typ.Args {
				extract(arg)
			}
		case *types.TFunc2:
			for _, param := range typ.Params {
				extract(param)
			}
			extract(typ.Return)
		case *types.TList:
			extract(typ.Element)
		case *types.TTuple:
			for _, elem := range typ.Elements {
				extract(elem)
			}
			// TCon and other base types don't have type variables
		}
	}

	extract(typ)
	return vars
}
