package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/link"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/version"
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

	// ModuleLoader basePath determines where relative imports ("./foo") are resolved.
	// When cfg.PackageDir is set (e.g. by the named-test harness), use that dir
	// instead of "." so that sibling imports resolve correctly regardless of CWD.
	loaderBaseDir := "."
	if cfg.PackageDir != "" {
		loaderBaseDir = cfg.PackageDir
	}
	modLoader := loader.NewModuleLoader(loaderBaseDir)
	modLoader.SetStrictSyntaxMode(cfg.StrictSyntaxMode)

	// Wire up package loader. Prefer the lock-backed resolver; if no lock
	// file exists yet but an ailang.toml does, fall back to a self-only
	// resolver so intra-package imports work during authoring. External
	// pkg/<other>/... imports under the fallback still error clearly.
	//
	// cfg.PackageDir overrides "." so callers (e.g. the named-test harness)
	// can resolve package manifests relative to the source file rather than CWD.
	pkgSearchDir := loaderBaseDir
	pkgResolver := tryLoadPackageResolver(pkgSearchDir)
	if pkgResolver == nil {
		pkgResolver = tryLoadSelfOnlyPackageResolver(pkgSearchDir)
	}
	if pkgResolver != nil {
		modLoader.SetPackageResolver(pkgResolver)
		// Pass module_prefix map so bare imports within packages resolve correctly
		// e.g., "docparse/types/document" → "pkg/sunholo/ailang_parse/types/document"
		if len(currentModulePrefixMap) > 0 {
			modLoader.SetModulePrefixMap(currentModulePrefixMap)
		}
		// Tell the loader the current package name so bare canonical imports
		// (e.g. `import sunholo/linkedin/types` from within sunholo/linkedin)
		// route through the self-reference path instead of failing as LDR001.
		if currentRootPkgName != "" {
			modLoader.SetCurrentPackageName(currentRootPkgName)
		}
	}

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

	// MOD011: Detect module path collisions.
	//
	// Two different source files must not declare the same `module X` header.
	// If they do, runtime dispatch (`callFunction("X", "f", ...)`) is ambiguous
	// — whichever one wins the loader cache silently shadows the other.
	//
	// This happens in practice with `module_prefix`-enabled packages: a local
	// file at `docparse/services/mcp_tools.ail` and a package file at
	// `sunholo/ailang_parse/services/mcp_tools.ail` (which uses
	// `module_prefix = "docparse"`) can both declare
	// `module docparse/services/mcp_tools`. v0.10.8 fixed route *registration*
	// in this case, but function *dispatch* still went to whichever module
	// was preloaded last — a silent footgun.
	//
	// Fail loudly per the "no silent fallbacks" principle.
	if err := detectModulePathCollisions(modules); err != nil {
		pipelineSpan.RecordError(err)
		return result, err
	}

	// MOD013: Detect module_prefix overlap between root project and a dependency.
	//
	// When the root project and a dep share the same module_prefix (e.g. both
	// use module_prefix = "src"), imports of root-only modules like rpc.ail can
	// silently cross the package boundary — the compiler strips the shared prefix
	// and constructs a pkg/-qualified path that points at the dep instead of the
	// root, producing a confusing "not exported by package" error with no pointer
	// to the underlying ambiguity.
	if err := detectModulePrefixOverlap(currentModulePrefixMap, currentRootPkgName); err != nil {
		pipelineSpan.RecordError(err)
		return result, err
	}

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
			// Read source from disk for content hash.
			// mod.Path is the canonical module identity (e.g. "benchmarks/workloads/warm_eval");
			// the actual disk path lives on mod.File.Path. Using mod.Path here caused
			// os.ReadFile to fail silently, leaving sourceContent="" so every module that
			// shared the same imports collided on the same cache key — edits to .ail files
			// were ignored after the first compile.
			sourceContent := ""
			srcPath := mod.Path
			if mod.File != nil && mod.File.Path != "" {
				srcPath = mod.File.Path
			}
			if srcBytes, err := os.ReadFile(srcPath); err == nil {
				sourceContent = string(srcBytes)
			}
			// Use build commit (from internal/version) as the compiler-identity component
			// of the cache key. Rebuilding `ailang` at a new commit invalidates cache,
			// so bugfixes to elaboration / type-checking / op-lowering take effect without
			// a manual cache nuke. The source hash and dep digests still catch edits.
			moduleCacheKey = ModuleCacheKey(version.Commit, sourceContent, depDigests)
			if entry, ok := cacheStore.Lookup(string(modID), moduleCacheKey); ok {
				cacheHits++
				// M-INCREMENTAL-TYPECHECK: Try to load cached artifacts and skip compilation
				if cached, loadErr := cacheStore.LoadArtifacts(string(modID)); loadErr == nil {
					if cfg.DebugCompile {
						fmt.Fprintf(os.Stderr, "[CACHE] %s: SKIP (cached %s ago)\n", modID, time.Since(entry.Timestamp).Truncate(time.Second))
					}
					unit.Core = cached.Core
					unit.CoreTI = cached.CoreTI
					unit.Iface = cached.Iface
					unit.Constructors = cached.Constructors
					// Register interface with linker so downstream modules can resolve imports
					if unit.Iface != nil {
						modLinker.RegisterIface(unit.Iface)
					}
					compiledUnits[string(modID)] = unit
					continue
				}
				// Fall through to normal compilation if load fails
				if cfg.DebugCompile {
					fmt.Fprintf(os.Stderr, "[CACHE] %s: HIT but load failed, recompiling (compiled %s ago)\n", modID, time.Since(entry.Timestamp).Truncate(time.Second))
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
		// M-TYPE-ALIAS: Pass imported aliases so transitive aliases can be embedded in the interface
		if err := buildAndRegisterInterface(unit, string(modID), compileResult.ModuleTypeEnv, modLinker, imports.ImportedTypeAliases); err != nil {
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
			// M-INCREMENTAL-TYPECHECK: Store full compiled artifacts for skip on next run
			_ = cacheStore.StoreArtifacts(string(modID), &CachedModule{
				Core:         unit.Core,
				CoreTI:       unit.CoreTI,
				Iface:        unit.Iface,
				Constructors: unit.Constructors,
			})
		}

		compiledUnits[string(modID)] = unit
	}

	// M-DX-SPLIT-ARG: same-typed-arg swap warnings (e.g. reversed split args).
	// Runs over every compiled unit's final Core. In user code, a call to an
	// imported std/string.split stays as App{Func: VarGlobal{std/string.split}}
	// even after lowering (the _str_split builtin substitution happens only
	// inside std/string itself), so this correctly detects on both freshly
	// compiled AND cache-hit-loaded modules. Non-blocking: warnings only.
	// The VarGlobal module-guard means std library internals and user-defined
	// local `split` functions never trigger. Iterate in sorted module order so
	// warning output is deterministic.
	argOrderModIDs := make([]string, 0, len(compiledUnits))
	for modID := range compiledUnits {
		argOrderModIDs = append(argOrderModIDs, modID)
	}
	sort.Strings(argOrderModIDs)
	for _, modID := range argOrderModIDs {
		result.Warnings = append(result.Warnings, DetectArgOrderWarnings(compiledUnits[modID].Core)...)
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

// detectModulePathCollisions returns an error (MOD011) if two *different*
// source files declare the same `module X` header.
//
// Runtime dispatch in serve-api and elsewhere looks up functions by the
// declared module path. If two different files claim the same path, whichever
// one wins the loader cache silently shadows the other — a silent footgun.
//
// The SAME file loaded under two canonical IDs is NOT a collision. This
// happens routinely with `module_prefix`-aliased packages: e.g. the package
// file at `~/.ailang/pkg/.../services/csv_parser.ail` can be loaded under
// both `pkg/sunholo/ailang_parse/services/csv_parser` (direct pkg/ import)
// and `docparse/services/csv_parser` (alias import resolved via
// module_prefix). Both entries point to the same physical file.
//
// We distinguish the two cases by comparing the module's resolved disk path
// (absolute filesystem path after symlinks are followed), which is populated
// by the loader at parse time.
func detectModulePathCollisions(modules map[string]*loader.LoadedModule) error {
	// Step 1: dedupe entries that point to the same physical file. Two
	// canonical IDs backed by the same `filepath.EvalSymlinks(absPath)`
	// represent the same source — keep only the lexically smallest canonical
	// ID for each physical file, so error messages remain deterministic.
	type entry struct {
		canonicalID string
		filePath    string // resolved absolute path (may be "" if unavailable)
		declared    string
	}
	byFile := make(map[string]entry) // resolvedFilePath -> best entry
	var unresolved []entry           // entries where we couldn't determine a disk path

	canonicalIDs := make([]string, 0, len(modules))
	for id := range modules {
		canonicalIDs = append(canonicalIDs, id)
	}
	sort.Strings(canonicalIDs)

	for _, canonicalID := range canonicalIDs {
		mod := modules[canonicalID]
		if mod == nil || mod.File == nil || mod.File.Module == nil {
			continue
		}
		declared := mod.File.Module.Path
		if declared == "" {
			continue
		}

		// Resolve to an absolute, symlink-free path so that two different
		// spellings of the same file are treated as identical.
		resolved := ""
		if mod.File.Path != "" {
			if abs, err := filepath.Abs(mod.File.Path); err == nil {
				if eval, err := filepath.EvalSymlinks(abs); err == nil {
					resolved = eval
				} else {
					resolved = abs
				}
			}
		}

		e := entry{canonicalID: canonicalID, filePath: resolved, declared: declared}
		if resolved == "" {
			// Without a resolvable disk path we can't safely dedupe,
			// so keep the entry around for downstream comparison by
			// canonical ID.
			unresolved = append(unresolved, e)
			continue
		}
		if _, seen := byFile[resolved]; !seen {
			byFile[resolved] = e // first (lexically smallest) canonical ID wins
		}
	}

	// Step 2: check for two *different* files claiming the same declared
	// module path. Iterate in sorted order for stable error messages.
	declaredBy := make(map[string]entry) // declared path -> first claimant

	// Resolved entries first, in deterministic order.
	resolvedPaths := make([]string, 0, len(byFile))
	for p := range byFile {
		resolvedPaths = append(resolvedPaths, p)
	}
	sort.Strings(resolvedPaths)

	checkCollision := func(e entry) error {
		if existing, seen := declaredBy[e.declared]; seen {
			return fmt.Errorf(
				"Error MOD011: module %q is declared in two different files:\n"+
					"  1. %s (canonical: %s)\n"+
					"  2. %s (canonical: %s)\n"+
					"  Fix: rename one of the module declarations so each module path is unique.\n"+
					"  Note: this commonly happens when a local file and a `module_prefix`-aliased\n"+
					"  package file claim the same namespace — runtime dispatch would be ambiguous.",
				e.declared, existing.filePath, existing.canonicalID, e.filePath, e.canonicalID)
		}
		declaredBy[e.declared] = e
		return nil
	}

	for _, p := range resolvedPaths {
		if err := checkCollision(byFile[p]); err != nil {
			return err
		}
	}
	for _, e := range unresolved {
		if err := checkCollision(e); err != nil {
			return err
		}
	}
	return nil
}

// detectModulePrefixOverlap returns an error (MOD013) when the root project and
// one or more dependency packages share the same module_prefix value.
//
// When root and dep share module_prefix = "src", imports of root-only modules
// (e.g. rpc.ail) silently cross the package boundary: the compiler strips the
// shared prefix and builds a pkg/-qualified path that resolves to the dep instead
// of the root, yielding an opaque "not exported by package" error with no pointer
// to the ambiguity. The motoko_agent scenario (root + sunholo/motoko_core both
// using module_prefix = "src") is the canonical reproduction.
//
// Only fires when the root package is involved in the overlap — two deps sharing
// a prefix without the root is allowed (they import each other via pkg/ explicitly).
func detectModulePrefixOverlap(prefixMap map[string]string, rootPkgName string) error {
	if rootPkgName == "" || len(prefixMap) == 0 {
		return nil
	}

	// Group package names by their module_prefix value.
	byPrefix := make(map[string][]string)
	for pkg, prefix := range prefixMap {
		if prefix != "" {
			byPrefix[prefix] = append(byPrefix[prefix], pkg)
		}
	}

	for prefix, pkgs := range byPrefix {
		if len(pkgs) < 2 {
			continue
		}
		// Only fire when the root package is one of the claimants.
		rootInGroup := false
		for _, p := range pkgs {
			if p == rootPkgName {
				rootInGroup = true
				break
			}
		}
		if !rootInGroup {
			continue
		}

		// Collect the dep names (everything except the root) for the message.
		deps := make([]string, 0, len(pkgs)-1)
		for _, p := range pkgs {
			if p != rootPkgName {
				deps = append(deps, p)
			}
		}
		sort.Strings(deps)

		return fmt.Errorf(
			"Error MOD013: ambiguous module ownership under shared module_prefix\n\n"+
				"  Root project:  %s\n"+
				"  Dependency:    %s\n"+
				"  Shared prefix: %q\n\n"+
				"  Both the root project and the dependency use module_prefix = %q.\n"+
				"  Imports of root-only modules (e.g. `import src/core/rpc`) can silently\n"+
				"  cross the package boundary and resolve against the dep instead of the root.\n\n"+
				"  Fix one of:\n"+
				"    1. Remove %s from your [dependencies] if your project IS\n"+
				"       the canonical source of those modules.\n"+
				"    2. Change one side's module_prefix to a distinct value (e.g. rename\n"+
				"       the dep's prefix from %q to a longer, unique segment).\n"+
				"    3. Use explicit pkg/ imports in extension packages that need the dep:\n"+
				"       `import pkg/%s/core/tool_contract (...)`",
			rootPkgName, strings.Join(deps, ", "), prefix, prefix,
			strings.Join(deps, " or "), prefix, strings.Join(deps, "/"),
		)
	}
	return nil
}

// validateModulePath validates that the module declaration matches the canonical path (MOD010).
func validateModulePath(mod *loader.LoadedModule, modID string, cfg *Config) error {
	if mod.File.Module == nil {
		// MOD014: A file with top-level *function declarations* but no `module`
		// header exports nothing, so the entry (e.g. `main`) never runs and the
		// runner silently falls through to a non-module print of unit — exit 0,
		// no output. Fail loudly with an actionable fix instead.
		//
		// Guard on Funcs ONLY, never Statements/Decls. A file that is a lone
		// bare expression to be evaluated (e.g. `1 + 1` -> 2) is parsed into
		// Statements (and, for back-compat, ALSO into Decls — see
		// parser_file.go), with Funcs empty. Gating on Decls would break that
		// eval path. `func main`-style footguns always populate Funcs.
		if len(mod.File.Funcs) > 0 {
			canonicalID := loader.CanonicalModuleID(modID)
			return fmt.Errorf("Error MOD014: no 'module' declaration — this file has top-level "+
				"declarations but no module, so nothing is exported and the entry never runs.\n"+
				"  Fix: add 'module %s' as the first line of the file", canonicalID)
		}
		return nil
	}

	canonicalID := loader.CanonicalModuleID(modID)
	// Exception: std/* modules bypass this check
	if strings.HasPrefix(canonicalID, "std/") || mod.File.Module.Path == canonicalID {
		return nil
	}
	// Exception: pkg/* imports — strip pkg/ prefix before comparing
	// The module declares "vendor/name/module" but the import path is "pkg/vendor/name/module"
	if strings.HasPrefix(canonicalID, "pkg/") {
		stripped := strings.TrimPrefix(canonicalID, "pkg/")
		if mod.File.Module.Path == stripped {
			return nil
		}
		// Also check module_prefix mapping: if the package has module_prefix="docparse",
		// then pkg/sunholo/docparse/services/api can declare "module docparse/services/api"
		if currentModulePrefixMap != nil {
			parts := strings.SplitN(stripped, "/", 3)
			if len(parts) >= 2 {
				pkgName := parts[0] + "/" + parts[1]
				if prefix, ok := currentModulePrefixMap[pkgName]; ok && len(parts) == 3 {
					prefixedPath := prefix + "/" + parts[2]
					if mod.File.Module.Path == prefixedPath {
						return nil
					}
				}
			}
		}
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

	// Strict mode: lead with actionable fix, no search trace noise
	return fmt.Errorf("Error MOD010: module '%s' doesn't match file path '%s'.\n  Fix: use --relax-modules flag or set AILANG_RELAX_MODULES=1\n  Alt: rename module declaration to: module %s\n  Alt: move file to: %s.ail",
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
