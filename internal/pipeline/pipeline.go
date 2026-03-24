// Package pipeline provides a unified compilation pipeline for AILANG
//
// # Architecture
//
// The pipeline is split into several files by responsibility:
//   - pipeline.go: Main types, Config, Result, Run entry point (THIS FILE)
//   - pipeline_single.go: Single-file/REPL pipeline (runSingle)
//   - pipeline_module.go: Multi-module pipeline with dependencies (runModule)
//   - pipeline_telemetry.go: Lowering telemetry reporting
//   - pipeline_helpers.go: Utility functions
//   - pipeline_types.go: CompileUnit and related types
//   - lower.go: Operator lowering pass
//   - specialize.go: Monomorphization pass
//   - validate_coretypeinfo.go: CoreTypeInfo validation
//   - var_resolver.go: Var type resolution
//
// # Usage
//
//	result, err := pipeline.Run(cfg, src)
//
// # See Also
//
//   - internal/ast: Surface AST
//   - internal/core: Core AST (ANF)
//   - internal/types: Type system
//   - internal/elaborate: Surface → Core elaboration
//   - internal/eval: Core evaluator
package pipeline

import (
	"context"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/telemetry"
	"github.com/sunholo/ailang/internal/types"
)

// compilerTracer is the OpenTelemetry tracer for compiler instrumentation.
// When no TracerProvider is configured, this returns a no-op tracer with ~2ns overhead.
var compilerTracer = telemetry.Tracer("ailang.compiler")

// Mode determines pipeline execution behavior
type Mode int

const (
	ModeCheck Mode = iota // Parse + type + elaborate + build interface (NO evaluation)
	ModeEval              // Include evaluation (REPL only)
)

// Config contains pipeline configuration options
type Config struct {
	Mode                    Mode                  // Execution mode (Check or Eval)
	JSON                    bool                  // Output JSON format
	Compact                 bool                  // Use compact JSON
	DumpCore                bool                  // Show Core AST
	DumpCoreLowered         bool                  // Show Core after lowering
	DumpTyped               bool                  // Show Typed AST
	TraceDefaulting         bool                  // Trace type defaulting
	DryLink                 bool                  // Show linking without eval
	RequireLowering         bool                  // Fail if operators not lowered
	ExperimentalBinopShim   bool                  // Feature flag for operator shim
	FailOnShim              bool                  // Fail if shim would be used (CI mode)
	TrackInstantiations     bool                  // Track polymorphic type instantiations
	LedgerHook              func(decision string) // Optional decision hook
	DisableMonomorphization bool                  // Disable monomorphization pass (emergency escape hatch)
	DisableVarResolution    bool                  // Disable Var type resolution (M-DX4 workaround, default enabled)
	DebugCompile            bool                  // Show compilation statistics (specialization counts, etc.)
	StrictSyntaxMode        bool                  // Disable syntactic sugar (require canonical syntax)
	RelaxModules            bool                  // Relax MOD010 validation (allow module path mismatches with warning)
	NoCache                 bool                  // M-PERF6: Disable compilation cache (--no-cache)
	ReleaseMode             bool                  // M-DEBUG-ERASURE: Erase Debug ghost effect (--release)

	// M-DX11: Type debugging
	DebugTypes     bool   // Enable type inference debugging output
	DebugTypesNode uint64 // Filter debug output to specific node (0 = all nodes)

	// Warning tracking (to avoid duplicate warnings)
	mod010WarnedPaths map[string]bool // Tracks paths that have already been warned for MOD010

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
	CoreTI types.CoreTypeInfo // M-DX23: Type info for Core expressions (for typed codegen)
	Typed  interface{}        // TODO: Add typed AST when available
	Linked interface{}        // TODO: Add linked program when available
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

	// M-DX11: Type debugging
	TypeChecker *types.CoreTypeChecker  // Type checker (for debug output)
	DebugSink   *types.VerboseDebugSink // Debug events (when DebugTypes enabled)

	// M-DX19: Dictionary registry with derived type class instances
	DictReg *types.DictionaryRegistry
}

// Run executes the full compilation pipeline
//
// For simple expressions/REPL, routes to runSingle (pipeline_single.go).
// For files with potential imports, routes to runModule (pipeline_module.go).
//
// Pipeline metrics are collected when AILANG_METRICS=1 is set.
// Use AILANG_METRICS_VERBOSE=1 for detailed timing breakdown to stderr.
// Use AILANG_HUB_URL to send metrics to the collaboration hub.
func Run(cfg Config, src Source) (Result, error) {
	return RunWithContext(context.Background(), cfg, src)
}

// RunWithContext runs the compilation pipeline with an explicit context for tracing.
// Use this when you have a parent span that should be the root of compilation traces.
func RunWithContext(ctx context.Context, cfg Config, src Source) (Result, error) {
	// Initialize metrics collector (only active if AILANG_METRICS=1)
	isModule := !(src.IsREPL || src.Filename == "" || src.Filename == "<repl>")
	metrics := NewMetricsCollector(src.Filename, isModule)

	var result Result
	var err error

	// For simple expressions/REPL, use the original single-file pipeline
	if src.IsREPL || src.Filename == "" || src.Filename == "<repl>" {
		result, err = runSingleWithContext(ctx, cfg, src)
	} else {
		// For files with potential imports, use the module pipeline
		result, err = runModuleWithContext(ctx, cfg, src)
	}

	// Record metrics from result and finalize
	if metrics.IsEnabled() {
		metrics.RecordFromResult(&result)
		metrics.Finalize()
	}

	return result, err
}
