package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sunholo/ailang/internal/devtoolsprompt"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/prompt"
	"github.com/sunholo/ailang/internal/runtime"
	"github.com/sunholo/ailang/internal/schema"
	"github.com/sunholo/ailang/internal/telemetry"
	ailtrace "github.com/sunholo/ailang/internal/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

//go:embed all:prompts
var embeddedPrompts embed.FS

var (
	// Version info - set by ldflags during build
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	// Color output
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	bold    = color.New(color.Bold).SprintFunc()
	// dim is defined in debug_types.go

	// Global flags
	_ = false // quietMode placeholder for future use
)

func main() {
	// Set embedded filesystem for prompts (bundled in binary)
	// This allows `ailang prompt` and `ailang devtools-prompt` to work from anywhere
	prompt.SetEmbeddedFS(embeddedPrompts)
	devtoolsprompt.SetEmbeddedFS(embeddedPrompts)

	var (
		versionFlag             = flag.Bool("version", false, "Print version information")
		helpFlag                = flag.Bool("help", false, "Show help")
		learnFlag               = flag.Bool("learn", false, "Enable learning mode (collect training data)")
		traceFlag               = flag.Bool("trace", false, "Enable execution tracing")
		compactFlag             = flag.Bool("compact", false, "Use compact JSON output")
		quietFlag               = flag.Bool("quiet", false, "Suppress progress messages (only show program output)")
		binopShimFlag           = flag.Bool("experimental-binop-shim", false, "Enable experimental operator shim")
		failOnShimFlag          = flag.Bool("fail-on-shim", false, "Fail if operator shim would be used (CI mode)")
		requireLoweringFlag     = flag.Bool("require-lowering", false, "Require operator lowering pass")
		trackInstantiationsFlag = flag.Bool("track-instantiations", false, "Track and dump polymorphic type instantiations")
		maxRecursionDepthFlag   = flag.Int("max-recursion-depth", 10000, "Maximum recursion depth (default: 10000)")
		noMonoFlag              = flag.Bool("no-mono", false, "Disable monomorphization (emergency escape hatch)")
		debugCompileFlag        = flag.Bool("debug-compile", false, "Show compilation statistics (specialization counts, etc.)")
		strictSyntaxFlag        = flag.Bool("strict-syntax", false, "Disable syntactic sugar (require canonical syntax)")
	)

	flag.Parse()

	// Set binary version for stdlib compatibility check
	// Version is set by ldflags at build time (e.g., "v0.4.8")
	loader.BinaryVersion = Version

	// Set compact mode globally if flag is provided
	if *compactFlag {
		schema.SetCompactMode(true)
	}

	// Set quiet mode globally (placeholder for future use)
	_ = *quietFlag

	if *versionFlag {
		printVersion()
		return
	}

	if *helpFlag || flag.NArg() == 0 {
		printHelp()
		return
	}

	// Check for stale binary (DX: prevents confusion when testing changes)
	checkStaleBinary()

	command := flag.Arg(0)

	switch command {
	case "run":
		runCommand()

	case "repl":
		runREPL(*learnFlag, *traceFlag, *strictSyntaxFlag)

	case "test":
		// Test command with flags
		testFlags := flag.NewFlagSet("test", flag.ExitOnError)
		formatFlag := testFlags.String("format", "human", "Output format: human or json")
		noColorFlag := testFlags.Bool("no-color", false, "Disable colored output")
		helpTestFlag := testFlags.Bool("help", false, "Show help for test command")

		_ = testFlags.Parse(flag.Args()[1:]) // Parse errors handled by flags package

		if *helpTestFlag {
			printTestHelp()
			return
		}

		path := "."
		if testFlags.NArg() >= 1 {
			path = testFlags.Arg(0)
		}

		runTestsV2(path, *formatFlag, !*noColorFlag)

	case "watch":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
			fmt.Println("Usage: ailang watch <file.ail>")
			os.Exit(1)
		}
		watchFile(flag.Arg(1), *traceFlag, *binopShimFlag, *failOnShimFlag, *requireLoweringFlag, *trackInstantiationsFlag, *noMonoFlag, *debugCompileFlag, *maxRecursionDepthFlag)

	case "check":
		// Parse check subcommand flags
		checkFS := flag.NewFlagSet("check", flag.ExitOnError)
		strictSyntaxCheck := checkFS.Bool("strict-syntax", false, "Disable syntactic sugar (require canonical syntax)")
		relaxModulesCheck := checkFS.Bool("relax-modules", false, "Relax MOD010 validation (allow module path mismatches with warning)")
		timeoutCheck := checkFS.String("timeout", "", "Compilation timeout (e.g., 30s, 2m). Dumps stack on timeout.")
		debugCompileCheck := checkFS.Bool("debug-compile", false, "Show compilation phase timing breakdown")
		jsonCheck := checkFS.Bool("json", false, "Output errors in JSON format (for AI/machine consumption)")
		quietCheck := checkFS.Bool("quiet", false, "Suppress progress lines, only output errors")

		_ = checkFS.Parse(flag.Args()[1:])

		if checkFS.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "%s: missing file or directory argument\n", red("Error"))
			fmt.Println("Usage: ailang check [options] <file.ail|directory>")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --strict-syntax    Disable syntactic sugar (require canonical syntax)")
			fmt.Println("  --relax-modules    Relax MOD010 validation (allow module path mismatches)")
			fmt.Println("  --timeout <dur>    Compilation timeout (e.g., 30s, 2m). Dumps stack on timeout.")
			fmt.Println("  --debug-compile    Show compilation phase timing breakdown")
			fmt.Println("  --json             Output errors in JSON format")
			fmt.Println("  --quiet            Suppress progress lines, only output errors")
			fmt.Println()
			fmt.Println("If a directory is given, all .ail files are checked recursively.")
			os.Exit(1)
		}
		checkFile(checkFS.Arg(0), *strictSyntaxCheck, *relaxModulesCheck, *timeoutCheck, *debugCompileCheck, *jsonCheck, *quietCheck)

	case "iface":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "%s: missing module argument\n", red("Error"))
			fmt.Println("Usage: ailang iface <module>")
			os.Exit(1)
		}
		outputInterface(flag.Arg(1))

	case "export-training":
		exportTraining()

	case "eval":
		runEval()

	case "eval-analyze":
		runEvalAnalyze()

	case "eval-compare":
		runEvalCompare()

	case "eval-matrix":
		runEvalMatrix()

	case "eval-summary":
		runEvalSummary()

	case "eval-report":
		runEvalReport()

	case "eval-suite":
		runEvalSuite()

	case "eval-chains":
		evalChainsCommand()

	case "doctor":
		runDoctor()

	case "builtins":
		runBuiltins()

	case "docs":
		docsCommand()

	case "debug":
		runDebug()

	case "messages", "msg":
		messagesCommand()

	case "prompt":
		runPrompt()

	case "devtools-prompt":
		runDevtoolsPrompt()

	case "server", "serve": // "serve" kept as alias for backward compatibility
		if err := serverCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "serve-api":
		if err := serveAPICommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "init":
		if err := initCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "access-control":
		if err := accessControlCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "compile":
		compileCommand()

	case "editor":
		editorCommand()

	case "axioms":
		axiomsCommand()

	case "replay":
		replayCommand()

	case "trace":
		traceCommand()

	case "observatory":
		observatoryCommand()

	case "chains":
		chainsCommand()

	case "dashboard":
		dashboardCommand()

	case "budget":
		budgetCommand()

	case "coordinator":
		if err := coordinatorCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "workspaces":
		if err := workspacesCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "exec":
		runExec()

	case "verify":
		verifyCommand()

	case "ai-check":
		aiCheckCommand()

	case "examples":
		examplesCommand(flag.Args()[1:])

	default:
		fmt.Fprintf(os.Stderr, "%s: unknown command '%s'\n", red("Error"), command)
		printHelp()
		os.Exit(1)
	}
}

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
	capsFlag := fs.String("caps", "", "Enable capabilities (comma-separated: IO,FS,Net,Env)")
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

	// Budget bypass flag (M-CAPABILITY-BUDGETS)
	noBudgetsFlag := fs.Bool("no-budgets", false, "Bypass effect budget enforcement (allow unlimited effect operations)")

	// Budget report flag (M-DX25)
	budgetReportFlag := fs.String("budget-report", "", "Print budget report after execution (flat, json)")

	// Contract verification flag (M-VERIFY-CONTRACTS)
	verifyContractsFlag := fs.Bool("verify-contracts", false, "Enable runtime contract validation (requires/ensures)")

	// Semantic trace export flag (M-TRACE-EXPORT)
	emitTraceFlag := fs.String("emit-trace", "", "Export semantic execution trace (jsonl, otel, jsonl,otel, auto)")

	// Parse from os.Args[2:] (everything after "run")
	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Check for filename argument
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		fmt.Println("Usage: ailang run [--caps IO] [--entry main] [--args-json '<json>'] <file.ail>")
		fmt.Println("Note: Flags must come BEFORE the filename")
		os.Exit(1)
	}

	filename := fs.Arg(0)

	// Extract program arguments (everything after the filename)
	// e.g., "ailang run program.ail arg1 arg2" → programArgs = ["arg1", "arg2"]
	programArgs := []string{}
	if fs.NArg() > 1 {
		programArgs = fs.Args()[1:] // Skip filename, take remaining args
	}

	runFile(filename, programArgs, *traceFlag, *seedFlag, *virtualTime, *jsonFlag, *compactFlag, *quietFlag, *binopShimFlag, *failOnShimFlag, *requireLoweringFlag, *trackInstantiationsFlag, *noMonoFlag, *debugCompileFlag, *strictSyntaxFlagRun, *entryFlag, *argsJSONFlag, *printFlag, *noPrintFlag, *capsFlag, *maxRecursionDepthFlag, *stdlibPathFlag, *traceLoaderFlag, *strictVersionFlag, *allowEnvFlag, *allowEnvFileFlag, *envFlag, *envSnapshotFlag, *writeEnvSnapshotFlag, *aiStubFlag, *aiModelFlag, *debugFlag, *relaxModulesFlag, *debugTypesFlag, *debugTypesNodeFlag, *noBudgetsFlag, *budgetReportFlag, *verifyContractsFlag, *emitTraceFlag)
}

func runFile(filename string, programArgs []string, trace bool, seed int, virtualTime bool, jsonOutput bool, compact bool, quiet bool, binopShim bool, failOnShim bool, requireLowering bool, trackInstantiations bool, noMono bool, debugCompile bool, strictSyntax bool, entry string, argsJSON string, print bool, noprint bool, caps string, maxRecursionDepth int, stdlibPath string, traceLoader bool, strictVersion bool, allowEnv string, allowEnvFile string, env string, envSnapshot string, writeEnvSnapshot string, aiStub bool, aiModel string, debugEffect bool, relaxModules bool, debugTypes bool, debugTypesNode uint64, noBudgets bool, budgetReport string, verifyContracts bool, emitTrace string) {
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
		StrictSyntaxMode:        strictSyntax,
		RelaxModules:            relaxModulesEffective,
		GlobalResolver:          builtinResolver, // Provide builtin access for type checking
		DebugTypes:              debugTypes,      // M-DX11: Enable type inference debug output
		DebugTypesNode:          debugTypesNode,  // M-DX11: Filter to specific node ID
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
		// Module execution with runtime (v0.2.0+)
		// Use CWD as base path for module resolution, not file directory.
		// This ensures imports like "sim/protocol" from "sim/world.ail" resolve
		// from project root, not relative to the importing file's directory.
		// Fix for: LDR001 module not found when importing from subdirectories
		rt := runtime.NewModuleRuntime(".")

		// Set up effect context with capability grants
		effCtx := effects.NewEffContext(programArgs)
		grantCapabilities(effCtx, caps)

		// M-CAPABILITY-BUDGETS: Allow bypassing budget enforcement via --no-budgets flag
		if noBudgets {
			effCtx.DisableBudgets = true
		}

		// M-DX25: Initialize budget report if requested
		if budgetReport != "" {
			effCtx.BudgetReport = effects.NewBudgetReport()
		}

		// Set up effect handlers if requested
		setupSharedMemHandler(effCtx)   // SharedMem for semantic caching (M-DX15)
		setupSharedIndexHandler(effCtx) // SharedIndex for semantic retrieval (M-DX16)
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
		// Auto-enable when OTEL is configured (zero extra flags needed)
		if emitTrace == "" && (telemetry.IsGoogleCloudEnabled() || telemetry.IsEnabled()) {
			emitTrace = "auto"
		}
		if emitTrace != "" {
			effCtx.Trace = ailtrace.NewCollector()
			if strings.Contains(emitTrace, "jsonl") {
				effCtx.IOWriter = os.Stderr // Program output to stderr so stdout is pure JSONL
			}
			if !quiet && emitTrace != "auto" {
				fmt.Fprintf(os.Stderr, "Trace collection enabled (%s)\n", emitTrace)
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
		}
		execErr := executeModuleEntrypoint(rt, execParams)

		// M-TRACE-EXPORT: Record module end with duration
		if effCtx.Trace != nil && effCtx.Trace.Enabled() {
			durationNS := time.Since(moduleStartTime).Nanoseconds()
			effCtx.Trace.RecordModuleEnd(moduleName, durationNS)
		}

		if execErr != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), execErr)
			os.Exit(1)
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
				if err := ailtrace.EmitOTELSpans(ctx, evalTracer, events, effCtx.Trace.BaseTime()); err != nil {
					fmt.Fprintf(os.Stderr, "%s: OTEL trace emission: %v\n", red("Error"), err)
				}
			}
		}
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
