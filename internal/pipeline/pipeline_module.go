package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/telemetry"
	"github.com/sunholo/ailang/internal/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// runModuleWithContext runs the pipeline for a module with dependencies
func runModuleWithContext(ctx context.Context, cfg Config, src Source) (Result, error) {
	// DEBUG: if cfg.TraceDefaulting { fmt.Printf("DEBUG: runModule called for %s\n", src.Filename) }
	result := Result{
		PhaseTimings: make(map[string]int64),
	}

	// Start OTEL span for module compilation pipeline (child of passed context)
	// Span name includes filename for easy identification in trace UI
	ctx, pipelineSpan := telemetry.StartSpan(ctx, compilerTracer, "compile: "+src.Filename,
		trace.WithAttributes(
			attribute.String("file.path", src.Filename),
			attribute.Int("file.size_bytes", len(src.Code)),
		),
	)

	// Capture starting memory for resource tracking
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)

	// Deferred function to record memory metrics before span ends
	defer func() {
		var endMem runtime.MemStats
		runtime.ReadMemStats(&endMem)
		memoryDeltaBytes := int64(endMem.TotalAlloc - startMem.TotalAlloc)
		allocsCount := int64(endMem.Mallocs - startMem.Mallocs)
		pipelineSpan.SetAttributes(
			attribute.Int64("compile.memory_delta_bytes", memoryDeltaBytes),
			attribute.Int64("compile.allocs_count", allocsCount),
			attribute.Int64("compile.heap_alloc_bytes", int64(endMem.HeapAlloc)),
		)
		pipelineSpan.End()
	}()

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
	_, loadSpan := telemetry.StartSpan(ctx, compilerTracer, "compile.load",
		trace.WithAttributes(
			attribute.String("root.module", src.Filename),
		),
	)

	modLoader := loader.NewModuleLoader(".")
	modLoader.SetStrictSyntaxMode(cfg.StrictSyntaxMode)
	modules, err := modLoader.LoadAll([]string{src.Filename})
	if err != nil {
		loadErr := fmt.Errorf("module loading error: %w", err)
		loadSpan.RecordError(loadErr)
		loadSpan.SetStatus(codes.Error, "module loading failed")
		loadSpan.End()
		pipelineSpan.RecordError(loadErr)
		return result, loadErr
	}

	loadSpan.SetAttributes(attribute.Int("modules.count", len(modules)))
	loadSpan.End()
	result.PhaseTimings["load"] = time.Since(start).Milliseconds()

	// Phase 2: Topological sort
	start = time.Now()
	_, topoSpan := telemetry.StartSpan(ctx, compilerTracer, "compile.topo_sort")

	modLinker := link.NewModuleLinker(modLoader)
	// Register $builtin as a first-class module
	link.RegisterBuiltinModule(modLinker)
	// Pass only the root module to TopoSort (dependencies will be discovered via DFS)
	rootCanonical := loader.CanonicalModuleID(src.Filename)
	sortedModules, err := modLinker.TopoSortFromRoot(rootCanonical, modules)
	if err != nil {
		topoErr := fmt.Errorf("dependency cycle: %w", err)
		topoSpan.RecordError(topoErr)
		topoSpan.SetStatus(codes.Error, "dependency cycle detected")
		topoSpan.End()
		pipelineSpan.RecordError(topoErr)
		return result, topoErr
	}

	topoSpan.SetAttributes(attribute.Int("modules.sorted", len(sortedModules)))
	topoSpan.End()
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
	_, compileSpan := telemetry.StartSpan(ctx, compilerTracer, "compile.modules",
		trace.WithAttributes(
			attribute.Int("modules.count", len(sortedModules)),
		),
	)

	compiledUnits := make(map[string]*CompileUnit)

	// M-PERF6: Load compilation cache for hit/miss tracking
	var cacheStore *CacheStore
	var cacheHits, cacheMisses int
	if !cfg.NoCache {
		projectDir := filepath.Dir(src.Filename)
		if cs, err := NewCacheStore(projectDir); err == nil {
			cacheStore = cs
		}
	}

	// M-DX11: Variables to capture root module's type checker and debug sink
	var rootTypeChecker *types.CoreTypeChecker
	var rootDebugSink *types.VerboseDebugSink

	for _, modID := range sortedModules {
		mod := modules[string(modID)]
		unit := &CompileUnit{
			ID:      string(modID),
			Surface: mod.File,
		}

		// M-PERF6: Compute cache key and check for hits
		var moduleCacheKey string
		if cacheStore != nil {
			// Build dep digests from already-compiled dependencies
			depDigests := make(map[string]string)
			for _, imp := range mod.Imports {
				if cu, ok := compiledUnits[imp]; ok && cu.Iface != nil {
					depDigests[imp] = cu.Iface.Digest
				}
			}
			// Read source from disk for content hash
			sourceContent := ""
			if srcBytes, err := os.ReadFile(mod.Path); err == nil {
				sourceContent = string(srcBytes)
			}
			moduleCacheKey = ModuleCacheKey(cacheKeyVersion, sourceContent, depDigests)
			if entry, ok := cacheStore.Lookup(string(modID), moduleCacheKey); ok {
				cacheHits++
				if cfg.DebugCompile {
					fmt.Fprintf(os.Stderr, "[CACHE] %s: HIT (compiled %s ago)\n", modID, time.Since(entry.Timestamp).Truncate(time.Second))
				}
			} else {
				cacheMisses++
				if cfg.DebugCompile {
					fmt.Fprintf(os.Stderr, "[CACHE] %s: MISS\n", modID)
				}
			}
		}

		// Validate module declaration matches canonical path (MOD010)
		if err := validateModulePath(mod, string(modID), &cfg); err != nil {
			return result, err
		}

		// Build external environment from already-compiled dependencies
		imports := resolveModuleImports(mod.File.Imports, string(modID), modLinker, cfg)

		// Elaborate to Core
		elaborator := elaborate.NewElaboratorWithPath(string(modID))
		elaborator.SetGlobalEnv(imports.GlobalRefs)
		// Share the pipeline's module loader with the elaborator
		// CRITICAL: The elaborator creates its own loader with filepath.Dir(modID) as basePath
		// which would be wrong for subdirectory modules (e.g., "sim/world" -> basePath "sim")
		// By sharing the pipeline's loader (basePath "."), we ensure correct resolution
		elaborator.SetModuleLoader(modLoader)
		// Add builtins to global environment so they can be referenced
		elaborator.AddBuiltinsToGlobalEnv()

		// M-RT1-FIX: Register imported constructors with elaborator
		// This is critical for pattern matching: without this, nullary constructors
		// like None are treated as variable patterns instead of constructor patterns
		for ctorName, info := range imports.ImportedCtorInfos {
			elaborator.RegisterConstructor(info.TypeName, ctorName, info.Arity, true, info.TypeParamCount)
		}

		unit.Core, err = elaborator.ElaborateFile(mod.File)
		if err != nil {
			// Preserve structured error reports without wrapping
			return result, err
		}

		// Extract constructors from elaborator and store in CompileUnit
		unit.Constructors = convertConstructors(elaborator.GetConstructors())

		// Add $adt factory types for this module's constructors to externalTypes
		buildConstructorFactoryTypes(unit.Constructors, imports.ExternalTypes)

		// Type check, elaborate dictionaries, monomorphize, resolve vars, lower operators
		compileResult, err := typeCheckAndLowerModule(unit, string(modID), rootCanonical, imports, elaborator, cfg)
		if err != nil {
			return result, err
		}

		// Collect warnings
		result.Warnings = append(result.Warnings, compileResult.Warnings...)

		// Capture root module's type checker and debug sink
		if compileResult.TypeChecker != nil {
			rootTypeChecker = compileResult.TypeChecker
			rootDebugSink = compileResult.DebugSink
		}

		// Build and register interface (using accumulated module type environment)
		if err := buildAndRegisterInterface(unit, string(modID), compileResult.ModuleTypeEnv, modLinker); err != nil {
			return result, err
		}

		// M-PERF6: Store cache entry after successful compilation
		if cacheStore != nil && moduleCacheKey != "" {
			ifaceJSON, _ := unit.Iface.ToNormalizedJSON()
			cacheStore.Store(string(modID), &CacheEntry{
				CacheKey:      moduleCacheKey,
				IfaceDigest:   unit.Iface.Digest,
				IfaceJSON:     ifaceJSON,
				CompileTimeMs: 0, // TODO: per-module timing
				Timestamp:     time.Now(),
			})
		}

		compiledUnits[string(modID)] = unit
	}

	// Register $adt module after all modules are loaded and their interfaces are built
	// This allows $adt to collect all constructors from all loaded modules
	link.RegisterAdtModule(modLinker)

	// M-PERF6: Save cache and report stats
	if cacheStore != nil {
		_ = cacheStore.Save()
		if cfg.DebugCompile {
			totalEntries, _ := cacheStore.Stats()
			fmt.Fprintf(os.Stderr, "[CACHE] Summary: %d hits, %d misses (%d modules cached)\n",
				cacheHits, cacheMisses, totalEntries)
		}
	}

	compileSpan.SetAttributes(attribute.Int("modules.compiled", len(compiledUnits)))
	compileSpan.End()
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
	// Set CoreTypeInfo for effect budget enforcement (M-CAPABILITY-BUDGETS)
	if rootUnit != nil && rootUnit.CoreTI != nil {
		coreEval.SetCoreTypeInfo(rootUnit.CoreTI)
	}
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

	// M-DX11: Store type checker and debug sink for --debug-types output
	result.TypeChecker = rootTypeChecker
	result.DebugSink = rootDebugSink

	// Convert CompileUnits to LoadedModules for runtime execution (v0.2.0+)
	result.Modules = assembleModuleResult(compiledUnits)

	// M-DX19: Include dictionary registry for runtime to use derived instances
	result.DictReg = cfg.DictReg

	return result, nil
}

// validateModulePath validates that the module declaration matches the canonical path (MOD010).
func validateModulePath(mod *loader.LoadedModule, modID string, cfg *Config) error {
	if mod.File.Module == nil {
		return nil
	}

	canonicalID := loader.CanonicalModuleID(modID)
	// Exception: std/* modules bypass this check
	if strings.HasPrefix(canonicalID, "std/") || mod.File.Module.Path == canonicalID {
		return nil
	}

	// Check if relaxation applies
	isTempPath := loader.IsTempPath(modID)
	shouldRelax := cfg.RelaxModules || isTempPath

	if shouldRelax {
		// Emit warning (once per path)
		if cfg.mod010WarnedPaths == nil {
			cfg.mod010WarnedPaths = make(map[string]bool)
		}
		if !cfg.mod010WarnedPaths[modID] {
			cfg.mod010WarnedPaths[modID] = true

			var reason string
			if isTempPath {
				reason = "temp-path"
			} else {
				reason = "relaxed"
			}
			warnMOD010Relaxed(mod.File.Module.Path, canonicalID, reason)
		}
		return nil
	}

	// Strict mode: emit error with suggestions
	return fmt.Errorf("MOD010: module declaration '%s' doesn't match canonical path '%s'\nSuggestions:\n  1. Rename module to: module %s\n  2. Move file to: %s.ail\n  3. For temp/scratch files: use --relax-modules or AILANG_RELAX_MODULES=1",
		mod.File.Module.Path, canonicalID, canonicalID, mod.File.Module.Path)
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
