package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"runtime/debug"
	rpprof "runtime/pprof"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/runtime"
	"github.com/sunholo/ailang/internal/telemetry"
	ailtrace "github.com/sunholo/ailang/internal/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func runCommand() {
	// Parse run subcommand flags
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	traceFlag := fs.Bool("trace", false, "Enable execution tracing")
	seedFlag := fs.Int("seed", 0, "Random seed for deterministic execution")
	virtualTime := fs.Bool("virtual-time", false, "Use virtual time for deterministic execution")
	jsonFlag := fs.Bool("json", false, "Output errors in structured JSON format")
	compactFlag := fs.Bool("compact", false, "Use compact JSON output")
	quietFlag := fs.Bool("quiet", false, "Suppress progress messages (only show program output)")
	binopShimFlag := fs.Bool("experimental-binop-shim", false, "Enable experimental operator shim")
	failOnShimFlag := fs.Bool("fail-on-shim", false, "Fail if operator shim would be used (CI mode)")
	requireLoweringFlag := fs.Bool("require-lowering", false, "Require operator lowering pass")
	trackInstantiationsFlag := fs.Bool("track-instantiations", false, "Track and dump polymorphic type instantiations")
	noMonoFlag := fs.Bool("no-mono", false, "Disable monomorphization (emergency escape hatch)")
	debugCompileFlag := fs.Bool("debug-compile", false, "Show compilation statistics (specialization counts, etc.)")
	strictSyntaxFlagRun := fs.Bool("strict-syntax", false, "Disable syntactic sugar (require canonical syntax)")
	entryFlag := fs.String("entry", "main", "Entrypoint function name to execute")
	argsJSONFlag := fs.String("args-json", "null", "JSON arguments to pass to entrypoint")
	printFlag := fs.Bool("print", true, "Print return value (even for unit type)")
	noPrintFlag := fs.Bool("no-print", false, "Suppress output (exit code only)")
	batchFlag := fs.Bool("batch", false, "Batch mode: compile once, run entrypoint per input (remaining args are inputs)")
	capsFlag := fs.String("caps", "", "Enable capabilities (comma-separated: IO,FS,Net,Env,Process)")
	maxRecursionDepthFlag := fs.Int("max-recursion-depth", 10000, "Maximum recursion depth (default: 10000)")

	// Stdlib resolution flags
	stdlibPathFlag := fs.String("stdlib-path", "", "Path to stdlib directory (overrides AILANG_STDLIB_PATH)")
	traceLoaderFlag := fs.Bool("trace-loader", false, "Enable module loader tracing")
	strictVersionFlag := fs.Bool("strict", false, "Fail on stdlib version mismatch")

	// Env capability flags
	allowEnvFlag := fs.String("allow-env", "", "Allowed environment variables (comma-separated: API_KEY,DEBUG)")
	allowEnvFileFlag := fs.String("allow-env-file", "", "File containing allowed environment variables (one per line)")
	envFlag := fs.String("env", "", "Override environment variables (comma-separated: KEY=value,FOO=bar)")
	envSnapshotFlag := fs.String("env-snapshot", "", "Load environment snapshot from JSON file")
	writeEnvSnapshotFlag := fs.String("write-env-snapshot", "", "Write environment snapshot to JSON file and exit")

	// Effect handler flags
	aiStubFlag := fs.Bool("ai-stub", false, "Enable AI effect with stub handler (returns default responses)")
	aiModelFlag := fs.String("ai", "", "Enable AI effect with model (e.g., claude-haiku-4-5, gpt5-mini, gemini-2-5-flash)")
	debugFlag := fs.Bool("debug", false, "Enable Debug effect with context (collects logs/assertions)")

	// Module relaxation flag
	relaxModulesFlag := fs.Bool("relax-modules", false, "Relax MOD010 validation (allow module path mismatches with warning)")

	// Type debugging flags (M-DX11)
	debugTypesFlag := fs.Bool("debug-types", false, "Show type inference debug output (substitutions, constraints, CoreTI)")
	debugTypesNodeFlag := fs.Uint64("node", 0, "Filter --debug-types output to specific node ID")

	// Net capability flags
	netAllowHTTPFlag := fs.Bool("net-allow-http", false, "Allow http:// URLs for Net effect (default: https only)")
	netAllowDomainsFlag := fs.String("net-allow-domains", "", "Domain allowlist for Net requests (comma-separated)")
	netAllowLocalhostFlag := fs.Bool("net-allow-localhost", false, "Allow localhost Net requests")
	netAllowMetadataFlag := fs.Bool("net-allow-metadata", false, "Allow cloud metadata server (169.254.169.254) for GCP/AWS/Azure")

	// Stream capability flags (M-STREAM-BIDI)
	streamAllowHTTPFlag := fs.Bool("stream-allow-http", false, "Allow insecure ws:// connections (default: wss:// only)")
	streamAllowDomainsFlag := fs.String("stream-allow-domains", "", "Domain allowlist for Stream connections (comma-separated)")
	streamAllowLocalhostFlag := fs.Bool("stream-allow-localhost", false, "Allow localhost WebSocket connections")

	// Process capability flags (M-PROCESS)
	processTimeoutFlag := fs.String("process-timeout", "30s", "Process execution timeout (e.g., 10s, 1m)")
	processAllowlistFlag := fs.String("process-allowlist", "", "Allowed commands (comma-separated, path-pinned at startup)")
	processMaxOutputFlag := fs.Int64("process-max-output", 10*1024*1024, "Maximum stdout+stderr bytes before kill (default: 10MB)")

	// Budget bypass flag (M-CAPABILITY-BUDGETS)
	noBudgetsFlag := fs.Bool("no-budgets", false, "Bypass effect budget enforcement (allow unlimited effect operations)")

	// Budget report flag (M-DX25)
	budgetReportFlag := fs.String("budget-report", "", "Print budget report after execution (flat, json)")

	// Contract verification flag (M-VERIFY-CONTRACTS)
	verifyContractsFlag := fs.Bool("verify-contracts", false, "Enable runtime contract validation (requires/ensures)")

	// Release mode flag (M-DEBUG-ERASURE)
	releaseFlag := fs.Bool("release", false, "Release mode: erase Debug ghost effect (zero-cost)")

	// Log level filtering for Debug ghost effect
	logLevelFlag := fs.String("log-level", "", "Minimum log level for Debug output (debug, info, warn, error, none)")

	// Semantic trace export flag (M-TRACE-EXPORT)
	emitTraceFlag := fs.String("emit-trace", "", "Export semantic execution trace (jsonl, otel, jsonl,otel, auto)")

	// Tracing tier flag (M-OBS-TRACE-TRIAGE). Wins over AILANG_TRACE env
	// and AILANG_NO_TRACE=1 legacy alias. Values: off|standard|deep.
	traceTierFlag := fs.String("trace-tier", "", "Tracing tier (off|standard|deep). Overrides AILANG_TRACE env var.")

	// Memory limit flag (M-EVAL-BOUNDED-PIPELINE)
	maxMemoryFlag := fs.String("max-memory", "", "Memory limit (e.g., 256MB, 1GB). Triggers aggressive GC near limit.")

	// CPU/memory profiling
	cpuprofileFlag := fs.String("cpuprofile", "", "Write CPU profile to file (Go pprof format)")
	memprofileFlag := fs.String("memprofile", "", "Write memory allocation profile to file (Go pprof format)")

	// M-BYTECODE-VM Phase 2D: bytecode VM execution path
	bytecodeFlag := fs.Bool("bytecode", false, "Run via the bytecode VM instead of the evaluator (Phase 2D)")
	strictBytecodeFlag := fs.Bool("strict-bytecode", false, "With --bytecode: fail instead of falling back to the evaluator")

	// Parse from os.Args[2:] (everything after "run")
	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Start CPU profiling if requested
	if *cpuprofileFlag != "" {
		f, err := os.Create(*cpuprofileFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating CPU profile: %v\n", err)
			os.Exit(1)
		}
		rpprof.StartCPUProfile(f)
		defer func() {
			rpprof.StopCPUProfile()
			f.Close()
			fmt.Fprintf(os.Stderr, "CPU profile written to %s\n", *cpuprofileFlag)
		}()
	}

	// M-PERF6B: Write memory allocation profile at exit
	if *memprofileFlag != "" {
		defer func() {
			f, err := os.Create(*memprofileFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating memory profile: %v\n", err)
				return
			}
			defer f.Close()
			debug.FreeOSMemory() // get up-to-date GC stats
			if err := rpprof.WriteHeapProfile(f); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing memory profile: %v\n", err)
				return
			}
			fmt.Fprintf(os.Stderr, "Memory profile written to %s\n", *memprofileFlag)
		}()
	}

	// Set log level filter for Debug ghost effect output
	if *logLevelFlag != "" {
		debugLogLevel = parseLogLevel(*logLevelFlag)
	}

	// Apply memory limit early (process-wide setting)
	if *maxMemoryFlag != "" {
		if err := applyMemoryLimit(*maxMemoryFlag); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			fmt.Println("Examples: 256MB, 512MB, 1GB, 2GB")
			os.Exit(1)
		}
	}

	// Check for filename argument
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		fmt.Println("Usage: ailang run [--caps IO] [--entry main] [--args-json '<json>'] <file.ail>")
		fmt.Println("Note: Flags must come BEFORE the filename")
		os.Exit(1)
	}

	filename := fs.Arg(0)

	// M-BYTECODE-VM Phase 2D M3: bytecode VM execution is now spliced into
	// the regular runFile path so the VM and evaluator share the same module
	// runtime, effect context, and bridge. The flags are threaded through
	// runFile → executeModuleEntrypoint, where the dispatch happens after
	// rt.LoadAndEvaluate populates the module instance the bridge needs.

	// Extract program arguments (everything after the filename)
	// e.g., "ailang run program.ail arg1 arg2" -> programArgs = ["arg1", "arg2"]
	// e.g., "ailang run program.ail -- arg1 arg2" -> programArgs = ["arg1", "arg2"] (-- stripped)
	programArgs := []string{}
	if fs.NArg() > 1 {
		programArgs = fs.Args()[1:] // Skip filename, take remaining args
		// Strip leading "--" separator if present (user convention for separating flags from program args)
		if len(programArgs) > 0 && programArgs[0] == "--" {
			programArgs = programArgs[1:]
		}
	}

	runFile(filename, programArgs, *traceFlag, *seedFlag, *virtualTime, *jsonFlag, *compactFlag, *quietFlag, *binopShimFlag, *failOnShimFlag, *requireLoweringFlag, *trackInstantiationsFlag, *noMonoFlag, *debugCompileFlag, *strictSyntaxFlagRun, *entryFlag, *argsJSONFlag, *printFlag, *noPrintFlag, *batchFlag, *capsFlag, *maxRecursionDepthFlag, *stdlibPathFlag, *traceLoaderFlag, *strictVersionFlag, *allowEnvFlag, *allowEnvFileFlag, *envFlag, *envSnapshotFlag, *writeEnvSnapshotFlag, *aiStubFlag, *aiModelFlag, *debugFlag, *relaxModulesFlag, *debugTypesFlag, *debugTypesNodeFlag, *noBudgetsFlag, *budgetReportFlag, *verifyContractsFlag, *emitTraceFlag, *traceTierFlag, *netAllowHTTPFlag, *netAllowDomainsFlag, *netAllowLocalhostFlag, *netAllowMetadataFlag, *streamAllowHTTPFlag, *streamAllowDomainsFlag, *streamAllowLocalhostFlag, *processTimeoutFlag, *processAllowlistFlag, *processMaxOutputFlag, *releaseFlag, *bytecodeFlag, *strictBytecodeFlag)
}

func runFile(filename string, programArgs []string, trace bool, seed int, virtualTime bool, jsonOutput bool, compact bool, quiet bool, binopShim bool, failOnShim bool, requireLowering bool, trackInstantiations bool, noMono bool, debugCompile bool, strictSyntax bool, entry string, argsJSON string, print bool, noprint bool, batch bool, caps string, maxRecursionDepth int, stdlibPath string, traceLoader bool, strictVersion bool, allowEnv string, allowEnvFile string, env string, envSnapshot string, writeEnvSnapshot string, aiStub bool, aiModel string, debugEffect bool, relaxModules bool, debugTypes bool, debugTypesNode uint64, noBudgets bool, budgetReport string, verifyContracts bool, emitTrace string, traceTier string, netAllowHTTP bool, netAllowDomains string, netAllowLocalhost bool, netAllowMetadata bool, streamAllowHTTP bool, streamAllowDomains string, streamAllowLocalhost bool, processTimeout string, processAllowlist string, processMaxOutput int64, release bool, bytecodeMode bool, strictBytecode bool) {
	// M-PERF-DOCPARSE: Reduce GC pressure for batch/CLI workloads.
	// Default GOGC=100 triggers GC when heap doubles — too aggressive for short-lived CLI runs.
	// GOGC=500 allows heap to grow 6x before GC, trading ~50MB extra memory for 25%+ speedup.
	// Only applies when GOGC is not already set by the user.
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(500)
	}

	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-run")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: telemetry init failed: %v\n", err)
	} else {
		defer shutdownTelemetry(ctx)
	}

	// Extract parent trace context from environment (if running as subprocess)
	// This enables distributed tracing when ailang is spawned by coordinator/executors
	ctx = telemetry.ExtractTraceContext(ctx)

	// Extract correlation IDs for span attributes (fallback linking)
	corrTaskID, sessionID := telemetry.ExtractCorrelationIDs()

	// Generate task ID for this run command
	runTaskID := fmt.Sprintf("run_%d", time.Now().UnixNano())

	// Inherit parent task from environment if set
	// This enables automatic hierarchy linking when ailang exec spawns ailang run
	parentTaskID := os.Getenv("AILANG_PARENT_TASK_ID")

	// If no parent task, use generic root marker for analytics
	// This ensures all runs appear in Observatory hierarchy views
	if parentTaskID == "" {
		parentTaskID = "root"
	}

	// Export current task ID so child processes inherit the hierarchy
	os.Setenv("AILANG_PARENT_TASK_ID", runTaskID)

	// Create root span for the entire run command
	// All child spans (compile, execute) will be linked under this
	tracer := otel.Tracer("ailang.cli")
	ctx, rootSpan := tracer.Start(ctx, "ailang run: "+filename,
		oteltrace.WithAttributes(
			attribute.String("file.path", filename),
			attribute.String("entry.function", entry),
			attribute.String("exec.task_id", runTaskID),
			attribute.String("exec.parent_task_id", parentTaskID),
		),
	)
	defer rootSpan.End()

	// Add correlation IDs as span attributes (if present)
	if corrTaskID != "" {
		rootSpan.SetAttributes(attribute.String("ailang.task_id", corrTaskID))
	}
	if sessionID != "" {
		rootSpan.SetAttributes(attribute.String("ailang.session_id", sessionID))
	}

	// Add capabilities to span (if any granted)
	if caps != "" {
		capList := strings.Split(caps, ",")
		rootSpan.SetAttributes(attribute.StringSlice("caps.granted", capList))
	}

	// Configure stdlib resolver via environment variables
	// CLI flags override environment variables
	if stdlibPath != "" {
		os.Setenv("AILANG_STDLIB_PATH", stdlibPath)
	}
	// Note: traceLoader and strictVersion will need ModuleRuntime integration (TODO: follow-up)
	// For now, they're accepted but not fully wired up
	_ = traceLoader
	_ = strictVersion

	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// When JSONL tracing is active, status messages go to stderr so stdout is clean JSONL
	statusOut := os.Stdout
	if strings.Contains(emitTrace, "jsonl") {
		statusOut = os.Stderr
	}

	// Check file extension
	if !strings.HasSuffix(filename, ".ail") {
		fmt.Fprintf(os.Stderr, "%s: file must have .ail extension\n", yellow("Warning"))
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
	if trace {
		fmt.Fprintf(statusOut, "  %s Tracing enabled\n", yellow("⚡"))
	}
	if seed != 0 {
		fmt.Fprintf(statusOut, "  %s Seed: %d\n", yellow("🎲"), seed)
	}
	if virtualTime {
		fmt.Fprintf(statusOut, "  %s Virtual time enabled\n", yellow("⏰"))
	}

	// Create builtin resolver for non-module evaluation (v0.2.0 hotfix)
	// This ensures arithmetic operators and string functions work in all files
	evaluator := eval.NewCoreEvaluator()
	if maxRecursionDepth > 0 {
		evaluator.SetMaxRecursionDepth(maxRecursionDepth)
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
	relaxModulesEffective := relaxModules
	if envVal := os.Getenv("AILANG_RELAX_MODULES"); envVal != "" {
		switch strings.ToLower(envVal) {
		case "1", "true", "yes":
			relaxModulesEffective = true
		}
	}

	cfg := pipeline.Config{
		Mode:                    mode,
		TraceDefaulting:         trace,
		ExperimentalBinopShim:   binopShim,
		FailOnShim:              failOnShim,
		RequireLowering:         requireLowering,
		TrackInstantiations:     trackInstantiations,
		DisableMonomorphization: noMono,
		DebugCompile:            debugCompile,
		NoCache:                 os.Getenv("AILANG_NO_CACHE") == "1",
		StrictSyntaxMode:        strictSyntax,
		RelaxModules:            relaxModulesEffective,
		GlobalResolver:          builtinResolver, // Provide builtin access for type checking
		DebugTypes:              debugTypes,      // M-DX11: Enable type inference debug output
		DebugTypesNode:          debugTypesNode,  // M-DX11: Filter to specific node ID
		ReleaseMode:             release,         // M-DEBUG-ERASURE: Erase Debug ghost effect
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, err := pipeline.RunWithContext(ctx, cfg, src)
	if err != nil {
		if jsonOutput {
			// Structured JSON output
			handleStructuredError(err, compact)
		} else {
			// Human-readable error output
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		}
		os.Exit(1)
	}

	// M-DX11: Display type inference debug output if enabled
	if debugTypes && result.DebugSink != nil && result.TypeChecker != nil {
		FormatTypeDebugOutput(result.DebugSink, result.TypeChecker, debugTypesNode)
	}

	// Display exhaustiveness warnings
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "%s\n", yellow(warning.String()))
	}

	// Entrypoint resolution and execution
	// Only attempt entrypoint resolution if the module has exports
	if result.Interface != nil && len(result.Interface.Exports) > 0 {

		// M-PERF7: Batch mode -- compile once, execute entrypoint per input
		// In batch mode, programArgs are treated as inputs (one execution per arg).
		// Each execution gets a fresh runtime and effect context to avoid state leaks.
		if batch {
			if len(programArgs) == 0 {
				fmt.Fprintf(os.Stderr, "%s: --batch requires at least one input argument\n", red("Error"))
				fmt.Fprintln(os.Stderr, "Usage: ailang run --batch <file.ail> input1 input2 ...")
				os.Exit(1)
			}
			if !quiet {
				fmt.Fprintf(os.Stderr, "%s Batch mode: %d inputs, compiled once\n", cyan("→"), len(programArgs))
			}

			batchErrors := 0
			for i, input := range programArgs {
				if !quiet {
					fmt.Fprintf(os.Stderr, "\n%s [%d/%d] %s\n", cyan("─── "), i+1, len(programArgs), input)
				}
				batchErr := executeBatchItem(ctx, result, input, entry, argsJSON, print, noprint, caps,
					maxRecursionDepth, noBudgets, budgetReport, debugEffect, verifyContracts,
					emitTrace, binopShim, quiet,
					netAllowHTTP, netAllowDomains, netAllowLocalhost, netAllowMetadata,
					streamAllowHTTP, streamAllowDomains, streamAllowLocalhost,
					processTimeout, processAllowlist, processMaxOutput,
					aiStub, aiModel, allowEnv, allowEnvFile, env, envSnapshot, writeEnvSnapshot,
					filename, bytecodeMode, strictBytecode)
				if batchErr != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), batchErr)
					batchErrors++
				}
			}
			if !quiet {
				fmt.Fprintf(os.Stderr, "\n%s Batch complete: %d/%d succeeded\n",
					green("✓"), len(programArgs)-batchErrors, len(programArgs))
			}
			if batchErrors > 0 {
				os.Exit(1)
			}
			// Skip normal execution path -- batch handled above
		} else {

			// Module execution with runtime (v0.2.0+)
			// Use CWD as base path for module resolution, not file directory.
			// This ensures imports like "sim/protocol" from "sim/world.ail" resolve
			// from project root, not relative to the importing file's directory.
			// Fix for: LDR001 module not found when importing from subdirectories
			rt := runtime.NewModuleRuntime(".")

			// Set up effect context with capability grants
			effCtx := effects.NewEffContext(programArgs)
			grantCapabilities(effCtx, caps)

			// M-PERF6B: Buffer stdout writes to reduce syscall overhead from println
			stdoutBuf := bufio.NewWriterSize(os.Stdout, 64*1024) // 64KB buffer
			effCtx.IOWriter = stdoutBuf

			// OTEL: Wire Go context and span wrapper for effect tracing
			effCtx.GoCtx = ctx
			effCtx.SpanWrapper = telemetry.NewEffectSpanWrapper()

			// M-CAPABILITY-BUDGETS: Allow bypassing budget enforcement via --no-budgets flag
			if noBudgets {
				effCtx.DisableBudgets = true
			}

			// M-DX25: Initialize budget report if requested
			if budgetReport != "" {
				effCtx.BudgetReport = effects.NewBudgetReport()
			}

			// Set up effect handlers if requested
			setupSharedMemHandler(effCtx)                                                               // SharedMem for semantic caching (M-DX15)
			setupSharedIndexHandler(effCtx)                                                             // SharedIndex for semantic retrieval (M-DX16)
			setupNetHandler(effCtx, netAllowHTTP, netAllowDomains, netAllowLocalhost, netAllowMetadata) // Net HTTP request security settings
			setupStreamHandler(effCtx, streamAllowHTTP, streamAllowDomains, streamAllowLocalhost)       // Stream for WebSocket connections (M-STREAM-BIDI)
			if err := setupProcessHandler(effCtx, processTimeout, processAllowlist, processMaxOutput); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
				os.Exit(1)
			}
			if err := setupAIHandler(effCtx, aiStub, aiModel); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
				os.Exit(1)
			}
			if debugEffect {
				effCtx.Debug = effects.NewDebugContext()
			}

			// M-VERIFY-CONTRACTS: Enable contract verification if requested
			if verifyContracts {
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
			if tier, err := ailtrace.ResolveTier(traceTier); err == nil {
				traceOpts.Tier = tier
			} else if !quiet {
				fmt.Fprintf(os.Stderr, "%s: %v (using standard)\n", red("Warning"), err)
			}
			noTrace := traceOpts.Tier == ailtrace.TierOff
			if !noTrace && emitTrace == "" && telemetry.IsEnabled() {
				emitTrace = "auto"
			}
			if !noTrace && emitTrace != "" {
				effCtx.Trace = ailtrace.NewCollector()
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
			envConfig := envFlags{
				allowEnv:         allowEnv,
				allowEnvFile:     allowEnvFile,
				env:              env,
				envSnapshot:      envSnapshot,
				writeEnvSnapshot: writeEnvSnapshot,
			}
			if shouldExit := setupEnvContext(effCtx, envConfig); shouldExit {
				os.Exit(0)
			}

			rt.GetEvaluator().SetEffContext(effCtx)

			// M-STREAM-BIDI: Wire function caller for stream event handlers
			{
				evaluator := rt.GetEvaluator()
				effCtx.FnCaller = evaluator.CallValue
				effCtx.FnCallerN = evaluator.CallValueN
			}

			// M-VERIFY-CONTRACTS: Enable binop shim for contract evaluation if needed
			if binopShim {
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
			execParams := moduleExecParams{
				filename:          filename,
				iface:             result.Interface,
				modules:           result.Modules,
				entry:             entry,
				argsJSON:          argsJSON,
				print:             print,
				noprint:           noprint,
				maxRecursionDepth: maxRecursionDepth,
				bytecodeMode:      bytecodeMode,
				strictBytecode:    strictBytecode,
				quiet:             quiet,
				pipelineResult:    &result,
			}
			// Catch EvalExitCode sentinel panic from exit() builtin.
			// Wraps execution so we can flush telemetry before os.Exit(code).
			var exitCode *int
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
				execErr := executeModuleEntrypoint(rt, execParams)
				if execErr != nil {
					stdoutBuf.Flush() // M-PERF6B: flush before exit
					fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), execErr)
					os.Exit(1)
				}
			}()

			// M-PERF6B: Flush buffered stdout before any post-execution output
			stdoutBuf.Flush()

			// Flush Debug ghost effect output to stderr
			flushDebugOutput(effCtx)

			// M-TRACE-EXPORT: Record module end with duration
			if effCtx.Trace != nil && effCtx.Trace.Enabled() {
				durationNS := time.Since(moduleStartTime).Nanoseconds()
				effCtx.Trace.RecordModuleEnd(moduleName, durationNS)
			}

			// If exit() was called, flush is done above, now exit with requested code
			if exitCode != nil {
				os.Exit(*exitCode)
			}

			// M-DX25: Print budget report after successful execution
			if budgetReport != "" && effCtx.BudgetReport != nil && effCtx.BudgetReport.HasUsage() {
				var output string
				switch budgetReport {
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

		} // end non-batch else
	} else {
		// Non-module mode - print result if evaluated by pipeline (ModeEval)
		printNonModuleResult(result.Value, print, noprint)
	}

	// Dump instantiations if tracking
	if trackInstantiations && result.Instantiations != nil {
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
}
