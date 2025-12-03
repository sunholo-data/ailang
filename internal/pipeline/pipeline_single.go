package pipeline

import (
	"fmt"
	"os"
	"time"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/linked"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

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
	p.SetStrictSyntaxMode(cfg.StrictSyntaxMode)

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

	// M-DX25.4: Pass constructor → ADT type mappings to type checker
	// This enables correct type inference for pattern matching on ADTs
	elabCtors := elaborator.GetConstructors()
	if len(elabCtors) > 0 {
		ctorTypes := make(map[string]string)
		for ctorName, ctorInfo := range elabCtors {
			ctorTypes[ctorName] = ctorInfo.TypeName
		}
		typeChecker.SetConstructorTypes(ctorTypes)
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
	result.Artifacts.CoreTI = typeChecker.CoreTI // M-DX23: Capture type info for codegen
	result.PhaseTimings["typecheck"] = time.Since(start).Milliseconds()

	// Capture instantiation tracking if enabled
	if cfg.TrackInstantiations {
		result.Instantiations = typeChecker.DumpInstantiations()
	}

	// Phase 3.4: Dictionary Elaboration (M-POLY-B)
	// Transform operators (BinOp, UnOp) to dictionary applications (DictApp)
	// This matches REPL behavior and is required for correct operator resolution
	start = time.Now()

	resolved := typeChecker.GetResolvedConstraints()
	elaboratedProg, err := elaborate.ElaborateWithDictionaries(coreProg, resolved)
	if err != nil {
		return result, fmt.Errorf("dictionary elaboration error: %w", err)
	}
	coreProg = elaboratedProg

	result.PhaseTimings["dict_elaboration"] = time.Since(start).Milliseconds()
	if cfg.DebugCompile {
		fmt.Fprintf(os.Stderr, "[DEBUG] Dictionary elaboration complete (%dms)\n",
			result.PhaseTimings["dict_elaboration"])
	}

	// Phase 3.5: Monomorphization (v0.4.0)
	start = time.Now()

	// Validate CoreTypeInfo before specialization (M-DX4)
	// This ensures every Core node has type information before monomorphization/lowering begins
	if err := ValidateCoreTypeInfo(coreProg, typeChecker.CoreTI); err != nil {
		return result, fmt.Errorf("CoreTypeInfo validation failed: %w", err)
	}

	// Validate effects (M-SOUNDNESS)
	// This ensures functions declare all effects they use
	// Compare declared effects from Surface AST with required effects from Core AST
	if err := ValidateEffects(result.Artifacts.AST, coreProg, typeChecker.CoreTI); err != nil {
		return result, fmt.Errorf("effect checking failed: %w", err)
	}

	// Perform monomorphization unless explicitly disabled
	var specializationStats SpecializationStats
	if !cfg.DisableMonomorphization {
		specializer := NewSpecializer(&typeChecker.CoreTI)

		// Run specialization pass
		specializedProg, err := specializer.Specialize(coreProg)
		if err != nil {
			return result, fmt.Errorf("monomorphization error: %w", err)
		}
		coreProg = specializedProg

		specializationStats = specializer.GetStats()
		if cfg.DebugCompile {
			fmt.Fprintf(os.Stderr, "[DEBUG] Monomorphization: %d specializations, %d skipped (cache: %d hits, %d misses)\n",
				specializationStats.TotalSpecializations,
				len(specializationStats.SkippedFunctions),
				specializationStats.CacheHits,
				specializationStats.CacheMisses)

			// Show per-function breakdown if there were specializations
			if specializationStats.TotalSpecializations > 0 {
				fmt.Fprintln(os.Stderr, "[DEBUG] Per-function specializations:")
				for fn, count := range specializationStats.PerFunction {
					fmt.Fprintf(os.Stderr, "[DEBUG]   %s: %d\n", fn, count)
				}
			}

			// Show skip reasons if any
			if len(specializationStats.SkippedFunctions) > 0 {
				fmt.Fprintln(os.Stderr, "[DEBUG] Skipped functions:")
				for _, skip := range specializationStats.SkippedFunctions {
					fmt.Fprintf(os.Stderr, "[DEBUG]   %s: %s\n", skip.DefSym, skip.Reason)
				}
			}
		}
	} else {
		// User explicitly disabled monomorphization
		// This is an emergency escape hatch - emit diagnostic
		if cfg.DebugCompile {
			fmt.Fprintln(os.Stderr, "[DEBUG] Monomorphization disabled via --no-mono (emergency use only)")
		}
	}

	result.PhaseTimings["monomorphization"] = time.Since(start).Milliseconds()
	if cfg.DebugCompile && !cfg.DisableMonomorphization {
		fmt.Fprintf(os.Stderr, "[DEBUG] Monomorphization complete (%dms)\n",
			result.PhaseTimings["monomorphization"])
	}

	// Phase 3.5.5: Var Type Resolution (M-DX4 workaround)
	// Resolve Var types from monomorphic bindings to fix operand types for lowering.
	// This propagates concrete types from Let bindings to Var usages when the binding
	// has a monomorphic type (Int, Float, etc.) but the Var still has a TVar.
	//
	// This is a WORKAROUND for M-DX4. The principled fix (M-POLY-B) will re-elaborate
	// specialized bodies after monomorphization.
	if !cfg.DisableVarResolution {
		resolver := NewVarResolver(typeChecker.CoreTI)
		resolver.Resolve(coreProg)

		if cfg.DebugCompile {
			fmt.Fprintf(os.Stderr, "[DEBUG] Var type resolution complete\n")
		}
	}

	// Phase 3.6: Operator Lowering
	start = time.Now()

	// Check if shim is forbidden in CI mode (before any other logic)
	if cfg.FailOnShim && cfg.ExperimentalBinopShim {
		return result, fmt.Errorf("CI_SHIM001: Operator shim usage detected but forbidden with --fail-on-shim")
	}

	// If require lowering is set, we must lower regardless of shim flag
	// If shim is not enabled, we must lower
	if cfg.RequireLowering || !cfg.ExperimentalBinopShim {
		lowerer := NewOpLowerer(cfg.TypeEnv, typeChecker.CoreTI)

		// M-DX4: Enable telemetry if --debug-compile flag is set
		if cfg.DebugCompile {
			lowerer.SetEnableTelemetry(true)
		}

		loweredProg, err := lowerer.Lower(coreProg)
		if err != nil {
			return result, fmt.Errorf("lowering error: %w", err)
		}

		// M-DX4: Report telemetry if --debug-compile flag is set
		if cfg.DebugCompile {
			reportLoweringTelemetry(lowerer.GetTelemetry())
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
