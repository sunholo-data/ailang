package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"runtime/debug"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/telemetry"
	ailtrace "github.com/sunholo-data/ailang/internal/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"golang.org/x/term"
)

// isStdoutTTY reports whether stdout is connected to a terminal. Used to
// decide whether to enable the M-PERF6B 64KB stdout buffer (terminal: yes,
// pipe: no — see comment at the buffer construction site for context).
func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func runFile(filename string, programArgs []string, trace bool, seed int, virtualTime bool, jsonOutput bool, compact bool, quiet bool, binopShim bool, failOnShim bool, requireLowering bool, trackInstantiations bool, noMono bool, debugCompile bool, strictSyntax bool, entry string, argsJSON string, print bool, noprint bool, batch bool, caps string, maxRecursionDepth int, stdlibPath string, traceLoader bool, strictVersion bool, allowEnv string, allowEnvFile string, env string, envSnapshot string, writeEnvSnapshot string, aiStub bool, aiModel string, aiRoutingValues routingFlagValues, debugEffect bool, relaxModules bool, debugTypes bool, debugTypesNode uint64, noBudgets bool, budgetReport string, verifyContracts bool, emitTrace string, traceTier string, netAllowHTTP bool, netAllowDomains string, netAllowLocalhost bool, netAllowMetadata bool, streamAllowHTTP bool, streamAllowDomains string, streamAllowLocalhost bool, processTimeout string, processAllowlist string, processMaxOutput int64, release bool, bytecodeMode bool, strictBytecode bool, orReferer string, orTitle string, orCategories string) {
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

			// M-AI-EFFECT-MODES M2: resolve the routing policy once per
			// batch run; the declared mode is the same for every input
			// because the same entry function is invoked.
			batchRoutingPolicy, batchRoutingErr := resolveRoutingPolicy(aiRoutingValues, result.Interface, entry)
			if batchRoutingErr != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), batchRoutingErr)
				os.Exit(1)
			}

			// Build OpenRouter attribution overrides if any flag is set
			var attr *ai.Attribution
			if orReferer != "" || orTitle != "" || orCategories != "" {
				attr = &ai.Attribution{
					HTTPReferer: orReferer,
					Title:       orTitle,
					Categories:  orCategories,
				}
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
					aiStub, aiModel, batchRoutingPolicy, attr, allowEnv, allowEnvFile, env, envSnapshot, writeEnvSnapshot,
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
			stdoutBuf := bufio.NewWriterSize(os.Stdout, 64*1024) // 64KB buffer
			if isStdoutTTY() {
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

			// M-AI-EFFECT-MODES M2: build the routing policy now that typecheck
			// has produced the entry function's effect row. The declared AI
			// mode (routeable / replay-only) bypasses the runtime --allow-routing
			// gate; bare !{AI} (mode=fixed) still requires it.
			aiRoutingPolicy, routingErr := resolveRoutingPolicy(aiRoutingValues, result.Interface, entry)
			if routingErr != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), routingErr)
				os.Exit(1)
			}
			// Build OpenRouter attribution overrides if any flag is set
			var attr *ai.Attribution
			if orReferer != "" || orTitle != "" || orCategories != "" {
				attr = &ai.Attribution{
					HTTPReferer: orReferer,
					Title:       orTitle,
					Categories:  orCategories,
				}
				if !quiet {
					fmt.Fprintf(os.Stderr, "OpenRouter attribution: referer=%s title=%s categories=%s\n",
						orReferer, orTitle, orCategories)
				}
			}
			if err := setupAIHandler(effCtx, aiStub, aiModel, aiRoutingPolicy, attr); err != nil {
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
