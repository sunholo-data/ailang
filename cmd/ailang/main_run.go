package main

import (
	"flag"
	"fmt"
	"os"

	"runtime/debug"
	rpprof "runtime/pprof"
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
	argsJSONFlag := fs.String("args-json", "null", "JSON arguments to pass to entrypoint (use '-' to read from stdin)")
	argsFileFlag := fs.String("args-file", "", "Path to a file containing JSON arguments (alternative to -args-json; bypasses shell quoting on Windows/PowerShell)")
	printFlag := fs.Bool("print", true, "Print return value (even for unit type)")
	noPrintFlag := fs.Bool("no-print", false, "Suppress output (exit code only)")
	batchFlag := fs.Bool("batch", false, "Batch mode: compile once, run entrypoint per input (remaining args are inputs)")
	capsFlag := fs.String("caps", "", "Enable capabilities (comma-separated: IO,FS,Net,Env,Process; or 'auto' to infer from the entrypoint)")
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
	aiModelFlag := fs.String("ai", "", "Enable AI effect with model (e.g., claude-haiku-4-5, gpt5-mini, gemini-2-5-flash, anthropic/claude-sonnet-4.5 (OpenRouter))")
	debugFlag := fs.Bool("debug", false, "Enable Debug effect with context (collects logs/assertions)")

	// AI routing flags (M-AI-OPENROUTER follow-up). Shared with `ailang exec`
	// via routing_flags.go. The routing policy is consumed by the OpenRouter
	// provider only; other providers reject a non-zero policy with
	// ai.ErrRoutingNotSupported. Safety gate: any --routing-* flag without
	// --allow-routing fails loudly because routing introduces dynamic
	// provider selection.
	routingFlags := registerRoutingFlags(fs)

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

	// OpenRouter app-attribution flags
	orRefererFlag := fs.String("openrouter-referer", "", "Override HTTP-Referer for OpenRouter app attribution")
	orTitleFlag := fs.String("openrouter-title", "", "Override X-OpenRouter-Title for OpenRouter app attribution")
	orCategoriesFlag := fs.String("openrouter-categories", "", "Override X-OpenRouter-Categories for OpenRouter app attribution (e.g., cli-agent,programming-app)")

	// Parse from os.Args[2:] (everything after "run")
	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// M-AI-EFFECT-MODES M2: snapshot the routing flag values now and defer
	// routing-policy construction until AFTER typecheck so the safety-gate
	// decision can consult the entry function's declared AI mode. A program
	// declared !{AI[mode=routeable]} attests routing intent at the type
	// level, which is stronger evidence than the runtime --allow-routing
	// flag, so the gate is bypassed in that case. Bare !{AI} (desugars to
	// mode=fixed) and programs without AI in the entry effect row still
	// require --allow-routing.
	//
	// At policy build time we pass providerHint="" because `run` determines
	// the provider from --ai <model> at handler-setup time (via
	// ai.GuessProvider). The handler dispatch surfaces "non-openrouter
	// provider with routing" mismatches via ai.ErrRoutingNotSupported on the
	// first AI call, which is the right level for that error.
	routingValues := routingFlags.snapshot()

	// Resolve --args-json / --args-file / stdin into a single effective JSON
	// string. PowerShell quote-mangling makes inline JSON unusable on Windows;
	// -args-file and -args-json - give operators a quote-free input path.
	resolvedArgsJSON, err := resolveArgsJSON(*argsJSONFlag, *argsFileFlag, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(2)
	}
	*argsJSONFlag = resolvedArgsJSON

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

	runFile(filename, programArgs, *traceFlag, *seedFlag, *virtualTime, *jsonFlag, *compactFlag, *quietFlag, *binopShimFlag, *failOnShimFlag, *requireLoweringFlag, *trackInstantiationsFlag, *noMonoFlag, *debugCompileFlag, *strictSyntaxFlagRun, *entryFlag, *argsJSONFlag, *printFlag, *noPrintFlag, *batchFlag, *capsFlag, *maxRecursionDepthFlag, *stdlibPathFlag, *traceLoaderFlag, *strictVersionFlag, *allowEnvFlag, *allowEnvFileFlag, *envFlag, *envSnapshotFlag, *writeEnvSnapshotFlag, *aiStubFlag, *aiModelFlag, routingValues, *debugFlag, *relaxModulesFlag, *debugTypesFlag, *debugTypesNodeFlag, *noBudgetsFlag, *budgetReportFlag, *verifyContractsFlag, *emitTraceFlag, *traceTierFlag, *netAllowHTTPFlag, *netAllowDomainsFlag, *netAllowLocalhostFlag, *netAllowMetadataFlag, *streamAllowHTTPFlag, *streamAllowDomainsFlag, *streamAllowLocalhostFlag, *processTimeoutFlag, *processAllowlistFlag, *processMaxOutputFlag, *releaseFlag, *bytecodeFlag, *strictBytecodeFlag, *orRefererFlag, *orTitleFlag, *orCategoriesFlag)
}
