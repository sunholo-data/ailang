package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/link"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"github.com/sunholo-data/ailang/internal/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// modulePipelineState carries the per-invocation orchestration state for one
// compile run, spanning load -> validate/topo -> compile-loop -> cache ->
// resolve -> evaluate. It holds only existing per-invocation values (Result,
// Config, loader/linker, compiled units, cache counters, canonical root, root
// debug handles) and no global state.
type modulePipelineState struct {
	ctx          context.Context
	pipelineSpan trace.Span
	cfg          Config
	src          Source
	cacheDeps    cacheDependencies
	afterLoad    []func(map[string]*loader.LoadedModule)

	result *Result

	modLoader     *loader.ModuleLoader
	modLinker     *link.ModuleLinker
	modules       map[string]*loader.LoadedModule
	sortedModules []link.ModuleID
	rootCanonical string

	compiledUnits   map[string]*CompileUnit
	moduleCache     *cacheRuntime
	cacheHits       int
	cacheMisses     int
	rootTypeChecker *types.CoreTypeChecker
	rootDebugSink   *types.VerboseDebugSink
	resolver        *link.Resolver
}

// newModulePipelineState initializes the per-call state for a module run.
func newModulePipelineState(ctx context.Context, cfg Config, src Source, cacheDeps cacheDependencies, afterLoad []func(map[string]*loader.LoadedModule)) *modulePipelineState {
	return &modulePipelineState{
		ctx:       ctx,
		cfg:       cfg,
		src:       src,
		cacheDeps: cacheDeps,
		afterLoad: afterLoad,
		result:    &Result{PhaseTimings: make(map[string]int64)},
	}
}

// startModulePipelineSpan opens the root compilation span for a module run.
func startModulePipelineSpan(ctx context.Context, src Source) (context.Context, trace.Span) {
	childCtx, span := telemetry.StartSpan(ctx, compilerTracer, "compile: "+src.Filename,
		trace.WithAttributes(
			attribute.String("file.path", src.Filename),
			attribute.Int("file.size_bytes", len(src.Code)),
		),
	)
	return childCtx, span
}

// finishModulePipelineSpan records the memory delta and ends the pipeline span.
func finishModulePipelineSpan(span trace.Span, startMem *runtime.MemStats) {
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	memoryDeltaBytes := int64(endMem.TotalAlloc - startMem.TotalAlloc)
	allocsCount := int64(endMem.Mallocs - startMem.Mallocs)
	span.SetAttributes(
		attribute.Int64("compile.memory_delta_bytes", memoryDeltaBytes),
		attribute.Int64("compile.allocs_count", allocsCount),
		attribute.Int64("compile.heap_alloc_bytes", int64(endMem.HeapAlloc)),
	)
	span.End()
}

// initEnvironments fills only the nil configuration environments, exactly as
// the pre-extraction coordinator did.
func (st *modulePipelineState) initEnvironments() {
	if st.cfg.TypeEnv == nil {
		st.cfg.TypeEnv = types.NewTypeEnvWithBuiltins()
	}
	if st.cfg.InstEnv == nil {
		st.cfg.InstEnv = types.LoadBuiltinInstances()
	}
	if st.cfg.DictReg == nil {
		st.cfg.DictReg = types.NewDictionaryRegistry()
	}
	if st.cfg.EvalEnv == nil {
		st.cfg.EvalEnv = eval.NewEnvironment()
	}
	if st.cfg.Instances == nil {
		st.cfg.Instances = make(map[string]core.DictValue)
	}
}

// run executes the ordered coordinator phases. Every early return surfaces the
// current Result, so partially-populated results are preserved on error.
func (st *modulePipelineState) run() (Result, error) {
	ctx, span := startModulePipelineSpan(st.ctx, st.src)
	st.ctx, st.pipelineSpan = ctx, span
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	defer func() { finishModulePipelineSpan(span, &startMem) }()

	st.initEnvironments()

	if err := st.loadModules(); err != nil {
		return *st.result, err
	}
	if err := st.validateAndSort(); err != nil {
		return *st.result, err
	}
	if err := st.compileAllAndFinalize(); err != nil {
		return *st.result, err
	}
	if err := st.registerResolver(); err != nil {
		return *st.result, err
	}
	if err := st.evaluateAndAssemble(); err != nil {
		return *st.result, err
	}
	return *st.result, nil
}

// loadModules sets up the loader/package resolver, loads all modules, invokes
// only the first post-load callback, and records the load phase timing.
func (st *modulePipelineState) loadModules() error {
	start := time.Now()
	_, loadSpan := telemetry.StartSpan(st.ctx, compilerTracer, "compile.load",
		trace.WithAttributes(
			attribute.String("root.module", st.src.Filename),
		),
	)

	// ModuleLoader basePath determines where relative imports ("./foo") are
	// resolved. When cfg.PackageDir is set (e.g. by the named-test harness),
	// use that dir instead of "." so sibling imports resolve correctly.
	loaderBaseDir := "."
	if st.cfg.PackageDir != "" {
		loaderBaseDir = st.cfg.PackageDir
	}
	modLoader := loader.NewModuleLoader(loaderBaseDir)
	modLoader.SetStrictSyntaxMode(st.cfg.StrictSyntaxMode)

	pkgSearchDir := packageSearchDir(st.cfg.PackageDir, loaderBaseDir, st.src.Filename)
	pkgResolver, err := tryLoadPackageResolver(pkgSearchDir)
	if err != nil {
		loadErr := fmt.Errorf("package resolution setup failed: %w", err)
		loadSpan.RecordError(loadErr)
		loadSpan.SetStatus(codes.Error, "package resolution setup failed")
		loadSpan.End()
		st.pipelineSpan.RecordError(loadErr)
		return loadErr
	}
	if pkgResolver == nil {
		pkgResolver = tryLoadSelfOnlyPackageResolver(pkgSearchDir)
	}
	if pkgResolver == nil {
		modLoader.SetPackageResolverAbsentReason(packageResolverAbsentReason(pkgSearchDir))
	}
	if pkgResolver != nil {
		modLoader.SetPackageResolver(pkgResolver)
		if len(currentModulePrefixMap) > 0 {
			modLoader.SetModulePrefixMap(currentModulePrefixMap)
		}
		if currentRootPkgName != "" {
			modLoader.SetCurrentPackageName(currentRootPkgName)
		}
	}

	st.modules, err = modLoader.LoadAll([]string{st.src.Filename})
	if err != nil {
		loadErr := fmt.Errorf("module loading error: %w", err)
		loadSpan.RecordError(loadErr)
		loadSpan.SetStatus(codes.Error, "module loading failed")
		loadSpan.End()
		st.pipelineSpan.RecordError(loadErr)
		return loadErr
	}
	if len(st.afterLoad) > 0 && st.afterLoad[0] != nil {
		st.afterLoad[0](st.modules)
	}

	loadSpan.SetAttributes(attribute.Int("modules.count", len(st.modules)))
	loadSpan.End()
	st.result.PhaseTimings["load"] = time.Since(start).Milliseconds()
	st.modLoader = modLoader
	return nil
}

// validateAndSort runs the collision/overlap diagnostics and the topological
// sort, recording the topo phase timing.
func (st *modulePipelineState) validateAndSort() error {
	if err := detectModulePathCollisions(st.modules); err != nil {
		st.pipelineSpan.RecordError(err)
		return err
	}
	if err := detectModulePrefixOverlap(currentModulePrefixMap, currentRootPkgName); err != nil {
		st.pipelineSpan.RecordError(err)
		return err
	}

	start := time.Now()
	_, topoSpan := telemetry.StartSpan(st.ctx, compilerTracer, "compile.topo_sort")

	modLinker := link.NewModuleLinker(st.modLoader)
	link.RegisterBuiltinModule(modLinker)
	st.rootCanonical = loader.CanonicalModuleID(st.src.Filename)
	sortedModules, err := modLinker.TopoSortFromRoot(st.rootCanonical, st.modules)
	if err != nil {
		topoErr := fmt.Errorf("dependency cycle: %w", err)
		topoSpan.RecordError(topoErr)
		topoSpan.SetStatus(codes.Error, "dependency cycle detected")
		topoSpan.End()
		st.pipelineSpan.RecordError(topoErr)
		return topoErr
	}

	topoSpan.SetAttributes(attribute.Int("modules.sorted", len(sortedModules)))
	topoSpan.End()
	st.result.PhaseTimings["topo"] = time.Since(start).Milliseconds()
	st.modLinker = modLinker
	st.sortedModules = sortedModules
	return nil
}

// compileAllAndFinalize runs the caching compile loop for every sorted module,
// then the sorted post-compile warning detectors, ADT registration, cache save
// and summary, ending the compile span and recording its timing.
func (st *modulePipelineState) compileAllAndFinalize() error {
	var phaseLog []string
	for _, m := range st.sortedModules {
		phaseLog = append(phaseLog, string(m))
	}
	if st.cfg.TraceDefaulting {
		fmt.Printf("PHASE ORDER: ELAB+TC+IFACE: %v; EVAL: %s\n", phaseLog, st.src.Filename)
	}

	start := time.Now()
	_, compileSpan := telemetry.StartSpan(st.ctx, compilerTracer, "compile.modules",
		trace.WithAttributes(
			attribute.Int("modules.count", len(st.sortedModules)),
		),
	)

	st.compiledUnits = make(map[string]*CompileUnit)
	st.moduleCache = newPipelineModuleCache(st.cfg, st.cacheDeps, st.src)

	for _, modID := range st.sortedModules {
		unit := &CompileUnit{ID: string(modID), Surface: st.modules[string(modID)].File}
		if err := st.compileOneModule(modID, unit); err != nil {
			return err
		}
	}

	argOrderModIDs := make([]string, 0, len(st.compiledUnits))
	for modID := range st.compiledUnits {
		argOrderModIDs = append(argOrderModIDs, modID)
	}
	sort.Strings(argOrderModIDs)
	for _, modID := range argOrderModIDs {
		st.result.Warnings = append(st.result.Warnings, DetectArgOrderWarnings(st.compiledUnits[modID].Core)...)
		st.result.Warnings = append(st.result.Warnings, DetectTakeAfterFlatMap(st.compiledUnits[modID].Core)...)
		st.result.Warnings = append(st.result.Warnings,
			DetectStrictFallbacks(st.compiledUnits[modID].Surface, st.compiledUnits[modID].Core)...)
	}

	link.RegisterAdtModule(st.modLinker)
	st.saveCacheAndReportStats()

	compileSpan.SetAttributes(attribute.Int("modules.compiled", len(st.compiledUnits)))
	compileSpan.End()
	st.result.PhaseTimings["compile"] = time.Since(start).Milliseconds()
	return nil
}

// newPipelineModuleCache builds the per-run cache runtime unless the cache is
// disabled. The returned runtime may have a nil store after a failed init.
func newPipelineModuleCache(cfg Config, deps cacheDependencies, src Source) *cacheRuntime {
	if cfg.NoCache {
		return nil
	}
	return newCacheRuntime(filepath.Dir(src.Filename), deps)
}

// compileOneModule compiles a single sorted module: it attempts a verified
// cache hit and, only if that does not serve the unit, falls through to a
// fresh single-module compile.
func (st *modulePipelineState) compileOneModule(modID link.ModuleID, unit *CompileUnit) error {
	mod := st.modules[string(modID)]
	key, cacheable := st.prepareCacheLookup(mod, string(modID))
	if cacheable && st.serveFromCache(mod, unit, key) {
		return nil
	}
	return st.compileFreshModule(mod, string(modID), unit, key)
}

// compileFreshModule performs the fresh single-module compile path: MOD010
// validation, import resolution, elaboration, typechecking/lowering, interface
// construction, fresh artifact publication and unit registration.
func (st *modulePipelineState) compileFreshModule(mod *loader.LoadedModule, modID string, unit *CompileUnit, cacheKey string) error {
	if err := validateModulePath(mod, modID, &st.cfg); err != nil {
		return err
	}

	imports := resolveModuleImports(mod.File.Imports, modID, st.modLinker, st.cfg)

	elaborator := elaborate.NewElaboratorWithPath(modID)
	elaborator.SetGlobalEnv(imports.GlobalRefs)
	elaborator.SetModuleLoader(st.modLoader)
	elaborator.AddBuiltinsToGlobalEnv()
	for ctorName, info := range imports.ImportedCtorInfos {
		elaborator.RegisterConstructor(info.TypeName, ctorName, info.Arity, true, info.TypeParamCount)
	}

	var err error
	unit.Core, err = elaborator.ElaborateFile(mod.File)
	if err != nil {
		return err
	}
	unit.Constructors = convertConstructors(elaborator.GetConstructors())
	buildConstructorFactoryTypes(unit.Constructors, imports.ExternalTypes)

	compileResult, err := typeCheckAndLowerModule(unit, modID, st.rootCanonical, imports, elaborator, st.cfg)
	if err != nil {
		return err
	}

	st.result.Warnings = append(st.result.Warnings, compileResult.Warnings...)

	if compileResult.TypeChecker != nil {
		st.rootTypeChecker = compileResult.TypeChecker
		st.rootDebugSink = compileResult.DebugSink
	}

	if err := buildAndRegisterInterface(unit, modID, compileResult.ModuleTypeEnv, st.modLinker, imports.ImportedTypeAliases); err != nil {
		return err
	}

	st.publishFreshModule(modID, cacheKey, unit)
	st.compiledUnits[modID] = unit
	return nil
}

// registerResolver registers all compiled modules with the shared resolver and
// wires the provided builtin lookup onto it.
func (st *modulePipelineState) registerResolver() error {
	resolver := st.modLinker.Resolver()
	st.resolver = resolver
	for modID, unit := range st.compiledUnits {
		resolver.RegisterCompiledModule(modID, unit)
	}
	if st.cfg.GlobalResolver != nil {
		resolver.SetBuiltinLookup(func(name string) (eval.Value, bool) {
			ref := core.GlobalRef{Module: "$builtin", Name: name}
			val, err := st.cfg.GlobalResolver.ResolveValue(ref)
			if err != nil || val == nil {
				return nil, false
			}
			return val, true
		})
	}
	return nil
}

// evaluateAndAssemble locates the root (canonical ID first, original filename
// fallback), configures the evaluator, evaluates the first declaration ONLY in
// ModeEval, and assembles the result.
func (st *modulePipelineState) evaluateAndAssemble() error {
	start := time.Now()
	rootUnit := st.compiledUnits[st.rootCanonical]
	if rootUnit == nil {
		rootUnit = st.compiledUnits[st.src.Filename]
		if rootUnit == nil {
			return fmt.Errorf("root module not found: %s (canonical: %s)", st.src.Filename, st.rootCanonical)
		}
	}

	coreEval := eval.NewCoreEvaluator()
	coreEval.SetGlobalResolver(st.resolver)
	if rootUnit != nil && rootUnit.CoreTI != nil {
		coreEval.SetCoreTypeInfo(rootUnit.CoreTI)
	}
	if st.cfg.ExperimentalBinopShim && !st.cfg.RequireLowering && !st.cfg.FailOnShim {
		coreEval.SetExperimentalBinopShim(true)
	}

	if st.cfg.Mode == ModeEval {
		if len(rootUnit.Core.Decls) > 0 {
			value, err := coreEval.Eval(rootUnit.Core.Decls[0])
			if err != nil {
				return fmt.Errorf("evaluation error: %w", err)
			}
			st.result.Value = value
		}
	}
	st.result.PhaseTimings["evaluate"] = time.Since(start).Milliseconds()

	st.result.Artifacts.AST = rootUnit.Surface
	st.result.Artifacts.Core = rootUnit.Core
	st.result.Artifacts.CoreTI = rootUnit.CoreTI
	st.result.Interface = rootUnit.Iface

	st.result.TypeChecker = st.rootTypeChecker
	st.result.DebugSink = st.rootDebugSink
	st.result.Modules = assembleModuleResult(st.compiledUnits)
	st.result.DictReg = st.cfg.DictReg

	return nil
}
