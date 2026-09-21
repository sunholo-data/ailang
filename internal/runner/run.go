package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/telemetry"
	ailtrace "github.com/sunholo-data/ailang/internal/trace"
	"go.opentelemetry.io/otel"
)

// Options is everything `ailang run` decides from its flags. The CLI fills it
// and calls Run; an embedded caller builds one directly.
type Options struct {
	Filename    string
	ProgramArgs []string

	// Status output
	Trace       bool
	Seed        int
	VirtualTime bool
	JSONOutput  bool
	Compact     bool
	Quiet       bool

	// Compile
	BinopShim           bool
	FailOnShim          bool
	RequireLowering     bool
	TrackInstantiations bool
	NoMono              bool
	DebugCompile        bool
	StrictSyntax        bool
	RelaxModules        bool
	PackageDir          string // explicit package root (ailang.toml/ailang.lock); "" = search upward from the file
	DebugTypes          bool
	DebugTypesNode      uint64
	Release             bool
	StdlibPath          string
	// TraceLoader and StrictVersion are accepted but not yet wired into
	// ModuleRuntime (follow-up).
	TraceLoader   bool
	StrictVersion bool

	// Execute
	Entry             string
	ArgsJSON          string
	Print             bool
	NoPrint           bool
	Batch             bool
	Caps              string
	MaxRecursionDepth int
	BytecodeMode      bool
	StrictBytecode    bool

	// Effects
	Env     EnvFlags
	Net     NetOptions
	Stream  StreamOptions
	Process ProcessOptions
	// FSMaxBytes is the --fs-max-bytes text ("" = AILANG_FS_MAX_BYTES, else
	// unbounded); resolved by SetupFSLimit (M-V1-MEMORY-FOOTPRINT M3).
	FSMaxBytes      string
	DebugEffect     bool
	DebugLogLevel   int
	NoBudgets       bool
	BudgetReport    string
	VerifyContracts bool
	EmitTrace       string
	TraceTier       string

	// AI. The handler and routing policy are supplied by the caller because
	// their wiring reads the model registry and provider configuration, which
	// stay outside the run pipeline. Either may be nil.
	//
	// RoutingPolicy is resolved after type-check so it can consult the entry
	// function's declared AI mode (M-AI-EFFECT-MODES M2).
	RoutingPolicy func(programIface *iface.Iface, entry string) (*ai.AIRoutingPolicy, error)
	// AIHandler wires the AI effect handler onto the effect context.
	AIHandler func(effCtx *effects.EffContext, policy *ai.AIRoutingPolicy, attr *ai.Attribution) error
	// OpenRouter attribution overrides (--or-referer / --or-title / --or-categories).
	ORReferer    string
	ORTitle      string
	ORCategories string
}

// Run compiles and executes one AILANG file per opts and returns the process
// exit code: 0 on success, the code passed to exit() by the program, 1 on any
// error. All status and error text goes to stdout/stderr exactly as the CLI
// printed it; the caller owns telemetry init, the root span on ctx, and the
// os.Exit.
func Run(ctx context.Context, opts Options) int {
	filename := opts.Filename
	programArgs := opts.ProgramArgs
	entry := opts.Entry
	caps := opts.Caps
	quiet := opts.Quiet
	emitTrace := opts.EmitTrace

	// Configure stdlib resolver via environment variables
	// CLI flags override environment variables
	if opts.StdlibPath != "" {
		os.Setenv("AILANG_STDLIB_PATH", opts.StdlibPath)
	}
	// Note: TraceLoader and StrictVersion will need ModuleRuntime integration (TODO: follow-up)
	// For now, they're accepted but not fully wired up
	_ = opts.TraceLoader
	_ = opts.StrictVersion

	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		return 1
	}

	// When JSONL tracing is active, status messages go to stderr so stdout is clean JSONL
	statusOut := os.Stdout
	if strings.Contains(emitTrace, "jsonl") {
		statusOut = os.Stderr
	}

	// Check file extension. `.ail` is the one canonical extension — the module
	// resolver only ever appends `.ail`, so a non-.ail file (e.g. `.ailang`) also
	// fails to import/resolve. Name the file and the fix so the failure is obvious.
	if !strings.HasSuffix(filename, ".ail") {
		fmt.Fprintf(os.Stderr, "%s: AILANG source files must use the .ail extension (got %q). Rename it to %s.ail — module resolution requires .ail.\n",
			yellow("Warning"), filename, strings.TrimSuffix(filename, filepath.Ext(filename)))
	}

	// Type check
	if !quiet {
		fmt.Fprintf(statusOut, "%s Type checking...\n", cyan("→"))
	}

	// Run effects analysis
	if !quiet {
		fmt.Fprintf(statusOut, "%s Effect checking...\n", cyan("→"))
	}

	// Execute
	if !quiet {
		fmt.Fprintf(statusOut, "%s Running %s\n", green("✓"), filename)
	}
	if opts.Trace {
		fmt.Fprintf(statusOut, "  %s Tracing enabled\n", yellow("⚡"))
	}
	if opts.Seed != 0 {
		fmt.Fprintf(statusOut, "  %s Seed: %d\n", yellow("🎲"), opts.Seed)
	}
	if opts.VirtualTime {
		fmt.Fprintf(statusOut, "  %s Virtual time enabled\n", yellow("⏰"))
	}

	// Create builtin resolver for non-module evaluation (v0.2.0 hotfix)
	// This ensures arithmetic operators and string functions work in all files
	evaluator := eval.NewCoreEvaluator()
	if opts.MaxRecursionDepth > 0 {
		evaluator.SetMaxRecursionDepth(opts.MaxRecursionDepth)
	}
	builtins := runtime.NewBuiltinRegistry(evaluator)
	builtinResolver := runtime.NewBuiltinOnlyResolver(builtins)

	// Determine if this is a module file by checking for "module" keyword
	// Non-module files (v0.1.0 style) need ModeEval for proper execution
	contentStr := string(content)
	hasModuleKeyword := false
	for _, line := range strings.Split(contentStr, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "module ") {
			hasModuleKeyword = true
			break
		}
	}
	isModuleFile := hasModuleKeyword

	// Use unified pipeline
	//  - ModeCheck for module files (execution via ModuleRuntime)
	//  - ModeEval for non-module files (evaluation in pipeline with proper resolvers)
	mode := pipeline.ModeCheck
	if !isModuleFile {
		mode = pipeline.ModeEval
	}

	// Check AILANG_RELAX_MODULES environment variable
	// CLI flag takes precedence, but env var can also enable relaxation
	relaxModulesEffective := opts.RelaxModules || config.RelaxModules()

	cfg := pipeline.Config{
		Mode:                    mode,
		TraceDefaulting:         opts.Trace,
		ExperimentalBinopShim:   opts.BinopShim,
		FailOnShim:              opts.FailOnShim,
		RequireLowering:         opts.RequireLowering,
		TrackInstantiations:     opts.TrackInstantiations,
		DisableMonomorphization: opts.NoMono,
		DebugCompile:            opts.DebugCompile,
		NoCache:                 config.NoCache(),
		StrictSyntaxMode:        opts.StrictSyntax,
		RelaxModules:            relaxModulesEffective,
		PackageDir:              opts.PackageDir,
		GlobalResolver:          builtinResolver,     // Provide builtin access for type checking
		DebugTypes:              opts.DebugTypes,     // M-DX11: Enable type inference debug output
		DebugTypesNode:          opts.DebugTypesNode, // M-DX11: Filter to specific node ID
		ReleaseMode:             opts.Release,        // M-DEBUG-ERASURE: Erase Debug ghost effect
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, err := pipeline.RunWithContext(ctx, cfg, src)
	if err != nil {
		if opts.JSONOutput {
			// Structured JSON output
			HandleStructuredError(err, opts.Compact)
		} else {
			// Human-readable error output
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		}
		return 1
	}

	// M-DX11: Display type inference debug output if enabled
	if opts.DebugTypes && result.DebugSink != nil && result.TypeChecker != nil {
		FormatTypeDebugOutput(result.DebugSink, result.TypeChecker, opts.DebugTypesNode)
	}

	// Display exhaustiveness warnings
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "%s\n", yellow(warning.String()))
	}

	// Entrypoint resolution and execution
	// Only attempt entrypoint resolution if the module has exports
	if result.Interface != nil && len(result.Interface.Exports) > 0 {
		if caps == "auto" {
			autoCaps := ResolveAutoCaps(result.Interface, entry)
			caps = strings.Join(autoCaps, ",")
			opts.Caps = caps
			// Progress, not program output: an installed [bin] shim runs
			// --quiet --caps auto on every invocation and its stderr must
			// stay the program's (M-PKG-BIN-ENTRYPOINTS).
			if !opts.Quiet {
				if len(autoCaps) == 0 {
					fmt.Fprintln(os.Stderr, "auto-granted capabilities: none")
				} else {
					fmt.Fprintf(os.Stderr, "auto-granted capabilities: %s\n", caps)
				}
			}
		}

		// M-PERF7: Batch mode -- compile once, execute entrypoint per input
		// In batch mode, programArgs are treated as inputs (one execution per arg).
		// Each execution gets a fresh runtime and effect context to avoid state leaks.
		if opts.Batch {
			if code := runBatch(ctx, result, opts); code != 0 {
				return code
			}
			// Skip normal execution path -- batch handled above
		} else if code := runSingle(ctx, result, opts, programArgs, caps); code >= 0 {
			return code
		}
	} else {
		// Non-module mode - print result if evaluated by pipeline (ModeEval)
		PrintNonModuleResult(result.Value, opts.Print, opts.NoPrint)
	}

	// Dump instantiations if tracking
	if opts.TrackInstantiations && result.Instantiations != nil {
		fmt.Printf("\n%s Polymorphic Instantiations:\n", cyan("📊"))
		if insts, ok := result.Instantiations["instantiations"].([]map[string]interface{}); ok {
			for i, inst := range insts {
				fmt.Printf("  [%d] %s @ %s\n", i, inst["var"], inst["location"])
				if fresh, ok := inst["fresh"].([]string); ok && len(fresh) > 0 {
					fmt.Printf("      Fresh vars: %v\n", fresh)
				}
				fmt.Printf("      Type: %s\n", inst["type"])
			}
		}
	}
	return 0
}

// attribution builds the OpenRouter attribution overrides if any flag is set.
func attribution(opts Options) *ai.Attribution {
	if opts.ORReferer != "" || opts.ORTitle != "" || opts.ORCategories != "" {
		return &ai.Attribution{
			HTTPReferer: opts.ORReferer,
			Title:       opts.ORTitle,
			Categories:  opts.ORCategories,
		}
	}
	return nil
}

// resolveRouting applies the caller's routing-policy resolver, if any.
func resolveRouting(opts Options, programIface *iface.Iface) (*ai.AIRoutingPolicy, error) {
	if opts.RoutingPolicy == nil {
		return nil, nil
	}
	return opts.RoutingPolicy(programIface, opts.Entry)
}

// runBatch executes the entrypoint once per program argument and returns the
// exit code (1 if any item failed or a flag was invalid, else 0).
func runBatch(ctx context.Context, result pipeline.Result, opts Options) int {
	programArgs := opts.ProgramArgs
	quiet := opts.Quiet
	if len(programArgs) == 0 {
		fmt.Fprintf(os.Stderr, "%s: --batch requires at least one input argument\n", red("Error"))
		fmt.Fprintln(os.Stderr, "Usage: ailang run --batch <file.ail> input1 input2 ...")
		return 1
	}
	if !quiet {
		fmt.Fprintf(os.Stderr, "%s Batch mode: %d inputs, compiled once\n", cyan("→"), len(programArgs))
	}

	// M-AI-EFFECT-MODES M2: resolve the routing policy once per
	// batch run; the declared mode is the same for every input
	// because the same entry function is invoked.
	batchRoutingPolicy, batchRoutingErr := resolveRouting(opts, result.Interface)
	if batchRoutingErr != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), batchRoutingErr)
		return 1
	}

	// Build OpenRouter attribution overrides if any flag is set
	attr := attribution(opts)

	batchErrors := 0
	for i, input := range programArgs {
		if !quiet {
			fmt.Fprintf(os.Stderr, "\n%s [%d/%d] %s\n", cyan("─── "), i+1, len(programArgs), input)
		}
		batchErr := ExecuteBatchItem(ctx, result, input, opts, batchRoutingPolicy, attr)
		if batchErr != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), batchErr)
			// A malformed --env* flag is the same for every item: fatal for
			// the run, as it always was.
			var envErr *EnvFlagError
			if errors.As(batchErr, &envErr) {
				return 1
			}
			batchErrors++
		}
	}
	if !quiet {
		fmt.Fprintf(os.Stderr, "\n%s Batch complete: %d/%d succeeded\n",
			green("✓"), len(programArgs)-batchErrors, len(programArgs))
	}
	if batchErrors > 0 {
		return 1
	}
	return 0
}

// runSingle executes the module entrypoint once (the non-batch path). It
// returns an exit code >= 0 when the run must stop there — an error, a
// --write-env-snapshot exit, or the program calling exit(N) — and -1 when
// execution completed and Run's trailing output should follow.
func runSingle(ctx context.Context, result pipeline.Result, opts Options, programArgs []string, caps string) int {
	quiet := opts.Quiet
	emitTrace := opts.EmitTrace

	// Module execution with runtime (v0.2.0+)
	// Use CWD as base path for module resolution, not file directory.
	// This ensures imports like "sim/protocol" from "sim/world.ail" resolve
	// from project root, not relative to the importing file's directory.
	// Fix for: LDR001 module not found when importing from subdirectories
	rt := runtime.NewModuleRuntime(".")

	// Set up effect context with capability grants
	effCtx := effects.NewEffContext(programArgs)
	// This runner OWNS the sandbox root handle (M-EXECUTOR-POLICY-HARDENING
	// M1): derived contexts share it, only the owner closes it, once.
	defer effCtx.CloseFSRoot()
	if err := GrantCapabilities(effCtx, caps); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		return 1
	}

	// M-PERF6B: Buffer stdout writes to reduce syscall overhead from println
	// — but only when stdout is a terminal. Long-running programs that
	// emit JSON events to a piped stdout (e.g. motoko_agent's TS frontend
	// reading the AILANG runtime's stdout) need real-time line delivery,
	// not block-buffered batches. With a 64KB buffer, small events sit in
	// memory until the buffer fills or the process exits — the agent loop
	// never exits during a turn, so events never reach downstream
	// consumers. Detect non-TTY and skip buffering in that case.
	//
	// Caught while running motoko_agent end-to-end on AILANG v0.15.x:
	// every agent turn produced 0 visible events at the TUI because all
	// thinking_stream_*, thinking_delta, session_start, etc. JSON lines
	// stayed buffered. See https://github.com/arniwesth/motoko_agent/pull/3
	// M-TERMINAL-IO: On a TTY, line-buffer stdout (flush on each newline)
	// so per-line / per-frame renders appear in real time — games,
	// progress bars, dashboards. Partial-line output (\r without \n) is
	// flushed explicitly via the IO.flush() effect.
	stdoutBuf := NewLineBufferedWriter(os.Stdout)
	if IsStdoutTTY() {
		effCtx.IOWriter = stdoutBuf
	} else {
		// Piped stdout: emit each println directly so downstream
		// line-readers (TS env-servers, logging pipes, journalctl, etc.)
		// see events in real time.
		effCtx.IOWriter = os.Stdout
	}

	// OTEL: Wire Go context and span wrapper for effect tracing
	effCtx.GoCtx = ctx
	effCtx.SpanWrapper = telemetry.NewEffectSpanWrapper()

	// M-CAPABILITY-BUDGETS: Allow bypassing budget enforcement via --no-budgets flag
	if opts.NoBudgets {
		effCtx.DisableBudgets = true
	}

	// M-DX25: Initialize budget report if requested
	if opts.BudgetReport != "" {
		effCtx.BudgetReport = effects.NewBudgetReport()
	}

	// Set up effect handlers if requested
	SetupSharedMemHandler(effCtx)   // SharedMem for semantic caching (M-DX15)
	SetupSharedIndexHandler(effCtx) // SharedIndex for semantic retrieval (M-DX16)
	// Net HTTP request security settings
	if err := SetupNetHandler(effCtx, opts.Net.AllowHTTP, opts.Net.AllowDomains, opts.Net.AllowLocalhost, opts.Net.AllowMetadata, opts.Net.Timeout); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		return 1
	}
	SetupStreamHandler(effCtx, opts.Stream.AllowHTTP, opts.Stream.AllowDomains, opts.Stream.AllowLocalhost) // Stream for WebSocket connections (M-STREAM-BIDI)
	if err := SetupFSLimit(effCtx, opts.FSMaxBytes); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		return 1
	}
	if err := SetupProcessHandler(effCtx, opts.Process.Timeout, opts.Process.Allowlist, opts.Process.MaxOutput); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		return 1
	}

	// M-AI-EFFECT-MODES M2: build the routing policy now that typecheck
	// has produced the entry function's effect row. The declared AI
	// mode (routeable / replay-only) bypasses the runtime --allow-routing
	// gate; bare !{AI} (mode=fixed) still requires it.
	aiRoutingPolicy, routingErr := resolveRouting(opts, result.Interface)
	if routingErr != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), routingErr)
		return 1
	}
	// Build OpenRouter attribution overrides if any flag is set
	attr := attribution(opts)
	if attr != nil && !quiet {
		fmt.Fprintf(os.Stderr, "OpenRouter attribution: referer=%s title=%s categories=%s\n",
			opts.ORReferer, opts.ORTitle, opts.ORCategories)
	}
	if opts.AIHandler != nil {
		if err := opts.AIHandler(effCtx, aiRoutingPolicy, attr); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			return 1
		}
	}
	// Debug.log streams to stderr on arrival under --log-level
	// (M-V1-MEMORY-FOOTPRINT M2, D-E): a logging loop no longer retains every
	// line until the run ends, and a below-threshold line is dropped before it
	// is stored. The final FlushDebugOutput stays as the no-sink fallback.
	effCtx.Debug = effects.NewDebugContext()
	effects.DebugSink{MinLevel: opts.DebugLogLevel}.Attach(effCtx.Debug) // W nil: current os.Stderr
	_ = opts.DebugEffect                                                 // the context now always exists; the flag is kept for CLI compatibility

	// M-VERIFY-CONTRACTS: Enable contract verification if requested
	if opts.VerifyContracts {
		effCtx.Contracts = effects.NewContractContextWithMode(effects.ContractModePanic)
		if !quiet {
			fmt.Fprintln(os.Stderr, "Contract verification enabled (panic mode)")
		}
	}

	// M-TRACE-EXPORT: Enable semantic execution trace collection
	// Auto-enable only when an OTEL exporter endpoint is configured.
	// M-PERF6B: Don't auto-enable on GOOGLE_CLOUD_PROJECT alone — that
	// env var is used for GCP auth, not tracing. Auto-trace without an
	// exporter creates ~2.7M objects (21% of allocations) that go nowhere.
	// AILANG_NO_TRACE=1 disables all tracing regardless of OTEL config.
	//
	// M-OBS-TRACE-TRIAGE: --trace-tier CLI flag / AILANG_TRACE env var
	// control the granularity of emitted spans:
	//   off      — nothing is emitted (same as AILANG_NO_TRACE=1)
	//   standard — module/effect(top-level)/compile/task-linked spans
	//   deep     — everything, including per-call eval.function.* spans
	// Precedence: --trace-tier > AILANG_TRACE > AILANG_NO_TRACE=1 > standard.
	traceOpts := ailtrace.DefaultTracingOptions()
	if tier, err := ailtrace.ResolveTier(opts.TraceTier); err == nil {
		traceOpts.Tier = tier
	} else if !quiet {
		fmt.Fprintf(os.Stderr, "%s: %v (using standard)\n", red("Warning"), err)
	}
	noTrace := traceOpts.Tier == ailtrace.TierOff
	if !noTrace && emitTrace == "" && telemetry.IsEnabled() {
		emitTrace = "auto"
	}
	if !noTrace && emitTrace != "" {
		collector := ailtrace.NewCollectorWithTier(traceOpts.Tier)
		// AILANG_TRACE_VALUES=off records the full call tree with no
		// payloads — for workloads under confidentiality terms, where
		// the structure is the audit value and the content must not be
		// written to disk at all.
		vm, vmErr := ailtrace.ResolveValueMode("")
		if vmErr != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), vmErr)
			return 1
		}
		collector.SetValueMode(vm)
		if !quiet && vm == ailtrace.ValuesRedacted {
			fmt.Fprintln(os.Stderr, "Trace: values redacted (AILANG_TRACE_VALUES=off) — structure only, no payloads")
		}
		effCtx.Trace = collector
		if strings.Contains(emitTrace, "jsonl") {
			effCtx.IOWriter = os.Stderr // Program output to stderr so stdout is pure JSONL
		}
		if !quiet && emitTrace != "auto" {
			fmt.Fprintf(os.Stderr, "Trace collection enabled (%s)\n", emitTrace)
		}
		if !quiet && emitTrace == "auto" {
			if traceOpts.Tier == ailtrace.TierDeep {
				fmt.Fprintln(os.Stderr, "Trace: deep (per-call function + effect spans)")
			} else {
				fmt.Fprintln(os.Stderr, "Trace: standard (set AILANG_TRACE=deep or --trace-tier deep for per-call spans)")
			}
		}
	}

	// Process environment variable flags
	shouldExit, envErr := SetupEnvContext(effCtx, opts.Env)
	if envErr != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), envErr)
		return 1
	}
	if shouldExit {
		return 0
	}

	rt.GetEvaluator().SetEffContext(effCtx)

	// M-STREAM-BIDI: Wire function caller for stream event handlers
	{
		evaluator := rt.GetEvaluator()
		effCtx.FnCaller = evaluator.CallValue
		effCtx.FnCallerN = evaluator.CallValueN
	}

	// M-VERIFY-CONTRACTS: Enable binop shim for contract evaluation if needed
	if opts.BinopShim {
		rt.GetEvaluator().SetExperimentalBinopShim(true)
	}

	// M-DX19: Inject dictionary registry with derived type class instances
	if result.DictReg != nil {
		rt.GetEvaluator().SetDictionaryRegistry(result.DictReg)
	}

	// M-TRACE-EXPORT: Record module start for replay metadata
	moduleName := ""
	if result.Interface != nil {
		moduleName = result.Interface.Module
	}
	var capsList []string
	if caps != "" {
		capsList = strings.Split(caps, ",")
	}
	if effCtx.Trace != nil && effCtx.Trace.Enabled() {
		effCtx.Trace.RecordModuleStart(moduleName, capsList)
	}

	// Execute module entrypoint
	moduleStartTime := time.Now()
	execParams := ModuleExecParams{
		Filename:          opts.Filename,
		Iface:             result.Interface,
		Modules:           result.Modules,
		Entry:             opts.Entry,
		ArgsJSON:          opts.ArgsJSON,
		Print:             opts.Print,
		NoPrint:           opts.NoPrint,
		MaxRecursionDepth: opts.MaxRecursionDepth,
		BytecodeMode:      opts.BytecodeMode,
		StrictBytecode:    opts.StrictBytecode,
		Quiet:             quiet,
		PipelineResult:    &result,
	}
	// Catch EvalExitCode sentinel panic from exit() builtin.
	// Wraps execution so we can flush before returning the code.
	var exitCode *int
	failed := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				if ec, ok := r.(*eval.EvalExitCode); ok {
					exitCode = &ec.Code
					return
				}
				panic(r) // re-panic for non-exit panics
			}
		}()
		execErr := ExecuteModuleEntrypoint(rt, execParams)
		if execErr != nil {
			stdoutBuf.Flush() // M-PERF6B: flush before exit
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), execErr)
			failed = true
		}
	}()
	if failed {
		return 1
	}

	// M-PERF6B: Flush buffered stdout before any post-execution output
	stdoutBuf.Flush()

	// Flush Debug ghost effect output to stderr
	FlushDebugOutput(effCtx, opts.DebugLogLevel, "")

	// M-TRACE-EXPORT: Record module end with duration
	if effCtx.Trace != nil && effCtx.Trace.Enabled() {
		durationNS := time.Since(moduleStartTime).Nanoseconds()
		effCtx.Trace.RecordModuleEnd(moduleName, durationNS)
	}

	// If exit() was called, flush is done above, now exit with requested code
	if exitCode != nil {
		return *exitCode
	}

	// M-DX25: Print budget report after successful execution
	if opts.BudgetReport != "" && effCtx.BudgetReport != nil && effCtx.BudgetReport.HasUsage() {
		var output string
		switch opts.BudgetReport {
		case "json":
			if data, err := effects.FormatReportJSON(effCtx.BudgetReport); err == nil {
				output = string(data)
			}
		default: // "flat" or any other value
			output = effects.FormatReport(effCtx.BudgetReport)
		}
		if output != "" {
			fmt.Fprintln(os.Stderr, output)
		}
	}

	// M-TRACE-EXPORT: Output semantic execution trace
	if emitTrace != "" && effCtx.Trace != nil {
		events := effCtx.Trace.Events()

		// Phase 1: JSONL output to stdout
		if strings.Contains(emitTrace, "jsonl") && len(events) > 0 {
			if err := ailtrace.WriteJSONL(os.Stdout, events); err != nil {
				fmt.Fprintf(os.Stderr, "%s: trace output: %v\n", red("Error"), err)
			}
		}

		// Phase 2: OTEL span emission
		if (strings.Contains(emitTrace, "otel") || emitTrace == "auto") && len(events) > 0 {
			evalTracer := otel.Tracer("ailang.eval")
			if err := ailtrace.EmitOTELSpansWithOptions(ctx, evalTracer, events, effCtx.Trace.BaseTime(), traceOpts); err != nil {
				fmt.Fprintf(os.Stderr, "%s: OTEL trace emission: %v\n", red("Error"), err)
			}
		}
	}

	return -1
}
