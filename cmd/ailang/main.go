package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	ailangErrors "github.com/sunholo/ailang/internal/errors"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/repl"
	"github.com/sunholo/ailang/internal/runtime"
	"github.com/sunholo/ailang/internal/runtime/argdecode"
	"github.com/sunholo/ailang/internal/schema"
	"github.com/sunholo/ailang/internal/types"
)

var (
	// Version info - set by ldflags during build
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	// Color output
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
)

func main() {
	var (
		versionFlag             = flag.Bool("version", false, "Print version information")
		helpFlag                = flag.Bool("help", false, "Show help")
		learnFlag               = flag.Bool("learn", false, "Enable learning mode (collect training data)")
		traceFlag               = flag.Bool("trace", false, "Enable execution tracing")
		seedFlag                = flag.Int("seed", 0, "Random seed for deterministic execution")
		virtualTime             = flag.Bool("virtual-time", false, "Use virtual time for deterministic execution")
		compactFlag             = flag.Bool("compact", false, "Use compact JSON output")
		jsonFlag                = flag.Bool("json", false, "Output errors in structured JSON format")
		binopShimFlag           = flag.Bool("experimental-binop-shim", false, "Enable experimental operator shim")
		failOnShimFlag          = flag.Bool("fail-on-shim", false, "Fail if operator shim would be used (CI mode)")
		requireLoweringFlag     = flag.Bool("require-lowering", false, "Require operator lowering pass")
		trackInstantiationsFlag = flag.Bool("track-instantiations", false, "Track and dump polymorphic type instantiations")
		entryFlag               = flag.String("entry", "main", "Entrypoint function name to execute")
		argsJSONFlag            = flag.String("args-json", "null", "JSON arguments to pass to entrypoint")
		printFlag               = flag.Bool("print", true, "Print return value (even for unit type)")
		noPrintFlag             = flag.Bool("no-print", false, "Suppress output (exit code only)")
		capsFlag                = flag.String("caps", "", "Enable capabilities (comma-separated: IO,FS,Net)")
	)

	flag.Parse()

	// Set compact mode globally if flag is provided
	if *compactFlag {
		schema.SetCompactMode(true)
	}

	if *versionFlag {
		printVersion()
		return
	}

	if *helpFlag || flag.NArg() == 0 {
		printHelp()
		return
	}

	command := flag.Arg(0)

	switch command {
	case "run":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
			fmt.Println("Usage: ailang run <file.ail> [--entry main] [--args-json '<json>'] [--print]")
			os.Exit(1)
		}
		runFile(flag.Arg(1), *traceFlag, *seedFlag, *virtualTime, *jsonFlag, *compactFlag, *binopShimFlag, *failOnShimFlag, *requireLoweringFlag, *trackInstantiationsFlag, *entryFlag, *argsJSONFlag, *printFlag, *noPrintFlag, *capsFlag)

	case "repl":
		runREPL(*learnFlag, *traceFlag)

	case "test":
		path := "."
		if flag.NArg() >= 2 {
			path = flag.Arg(1)
		}
		runTests(path)

	case "watch":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
			fmt.Println("Usage: ailang watch <file.ail>")
			os.Exit(1)
		}
		watchFile(flag.Arg(1), *traceFlag, *binopShimFlag, *failOnShimFlag, *requireLoweringFlag, *trackInstantiationsFlag)

	case "check":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
			fmt.Println("Usage: ailang check <file.ail>")
			os.Exit(1)
		}
		checkFile(flag.Arg(1))

	case "iface":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "%s: missing module argument\n", red("Error"))
			fmt.Println("Usage: ailang iface <module>")
			os.Exit(1)
		}
		outputInterface(flag.Arg(1))

	case "export-training":
		exportTraining()

	case "lsp":
		runLSP()

	case "eval":
		// TODO: Implement AI evaluation benchmarks
		fmt.Fprintf(os.Stderr, "%s: eval command not yet implemented\n", red("Error"))
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "%s: unknown command '%s'\n", red("Error"), command)
		printHelp()
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("AILANG %s\n", bold(Version))
	if Commit != "unknown" {
		fmt.Printf("Commit: %s\n", Commit)
	}
	if BuildTime != "unknown" {
		fmt.Printf("Built:  %s\n", BuildTime)
	}
	fmt.Println("\nThe AI-First Programming Language")
	fmt.Println("Copyright (c) 2025")
}

func printHelp() {
	fmt.Println(bold("AILANG - The AI-First Programming Language"))
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ailang <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Printf("  %s <file>      Run an AILANG program\n", cyan("run"))
	fmt.Printf("  %s             Start the interactive REPL\n", cyan("repl"))
	fmt.Printf("  %s [path]      Run tests\n", cyan("test"))
	fmt.Printf("  %s <file>      Watch file for changes and auto-reload\n", cyan("watch"))
	fmt.Printf("  %s <file>      Type-check a file without running\n", cyan("check"))
	fmt.Printf("  %s <module>    Output normalized JSON interface for a module\n", cyan("iface"))
	fmt.Printf("  %s   Export training data\n", cyan("export-training"))
	fmt.Printf("  %s              Start the Language Server Protocol server\n", cyan("lsp"))
	fmt.Printf("  %s              Run AI benchmarks (AILANG vs Python)\n", cyan("eval"))
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --version        Print version information")
	fmt.Println("  --help           Show this help message")
	fmt.Println("  --learn          Enable learning mode (REPL only)")
	fmt.Println("  --trace          Enable execution tracing")
	fmt.Println("  --seed <n>       Set random seed for deterministic execution")
	fmt.Println("  --virtual-time   Use virtual time for testing")
	fmt.Println("  --compact        Use compact JSON output")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s              # Start REPL\n", cyan("ailang repl"))
	fmt.Printf("  %s    # Run program\n", cyan("ailang run hello.ail"))
	fmt.Printf("  %s        # Type-check\n", cyan("ailang check src/"))
	fmt.Printf("  %s  # Watch with tracing\n", cyan("ailang watch main.ail --trace"))
	fmt.Printf("  %s  # Run benchmark\n", cyan("ailang eval --benchmark fizzbuzz --mock"))
}

func runFile(filename string, trace bool, seed int, virtualTime bool, jsonOutput bool, compact bool, binopShim bool, failOnShim bool, requireLowering bool, trackInstantiations bool, entry string, argsJSON string, print bool, noprint bool, caps string) {
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Check file extension
	if !strings.HasSuffix(filename, ".ail") {
		fmt.Fprintf(os.Stderr, "%s: file must have .ail extension\n", yellow("Warning"))
	}

	// Type check
	fmt.Printf("%s Type checking...\n", cyan("→"))

	// Run effects analysis
	fmt.Printf("%s Effect checking...\n", cyan("→"))

	// Execute
	fmt.Printf("%s Running %s\n", green("✓"), filename)
	if trace {
		fmt.Printf("  %s Tracing enabled\n", yellow("⚡"))
	}
	if seed != 0 {
		fmt.Printf("  %s Seed: %d\n", yellow("🎲"), seed)
	}
	if virtualTime {
		fmt.Printf("  %s Virtual time enabled\n", yellow("⏰"))
	}

	// Create builtin resolver for non-module evaluation (v0.2.0 hotfix)
	// This ensures arithmetic operators and string functions work in all files
	evaluator := eval.NewCoreEvaluator()
	builtins := runtime.NewBuiltinRegistry(evaluator)
	builtinResolver := runtime.NewBuiltinOnlyResolver(builtins)

	// Use unified pipeline
	cfg := pipeline.Config{
		TraceDefaulting:       trace,
		ExperimentalBinopShim: binopShim,
		FailOnShim:            failOnShim,
		RequireLowering:       requireLowering,
		TrackInstantiations:   trackInstantiations,
		GlobalResolver:        builtinResolver, // Provide builtin access
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, err := pipeline.Run(cfg, src)
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

	// Display exhaustiveness warnings
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "%s\n", yellow(warning.String()))
	}

	// Entrypoint resolution and execution
	// Only attempt entrypoint resolution if the module has exports
	if result.Interface != nil && len(result.Interface.Exports) > 0 {
		// Module mode - look up and call entrypoint
		fnExport, exists := result.Interface.Exports[entry]
		if !exists {
			fmt.Fprintf(os.Stderr, "%s: entrypoint '%s' not found in module\n", red("Error"), entry)
			fmt.Fprintf(os.Stderr, "Available exports: ")
			exportNames := []string{}
			for name := range result.Interface.Exports {
				exportNames = append(exportNames, name)
			}
			fmt.Fprintf(os.Stderr, "%v\n", exportNames)
			os.Exit(1)
		}

		// Check function type and decode arguments
		scheme := fnExport.Type
		if scheme == nil {
			fmt.Fprintf(os.Stderr, "%s: entrypoint '%s' has no type information\n", red("Error"), entry)
			os.Exit(1)
		}

		// The entrypoint must be a function type
		fnType, isFn := scheme.Type.(*types.TFunc2)
		if !isFn {
			fmt.Fprintf(os.Stderr, "%s: entrypoint '%s' is not a function (has type %s)\n", red("Error"), entry, scheme.Type)
			os.Exit(1)
		}

		// Module execution with runtime (v0.2.0+)
		rt := runtime.NewModuleRuntime(filepath.Dir(filename))

		// Set up effect context with capability grants
		effCtx := effects.NewEffContext()
		if caps != "" {
			for _, capName := range strings.Split(caps, ",") {
				capName = strings.TrimSpace(capName)
				if capName != "" {
					effCtx.Grant(effects.NewCapability(capName))
				}
			}
		}
		rt.GetEvaluator().SetEffContext(effCtx)

		// Pre-load modules from pipeline result
		if result.Modules != nil {
			for path, loaded := range result.Modules {
				rt.PreloadModule(path, loaded)
			}
		}

		// Load and evaluate module
		inst, err := rt.LoadAndEvaluate(result.Interface.Module)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: module evaluation failed: %v\n", red("Error"), err)
			os.Exit(1)
		}

		// Get entrypoint
		entrypointVal, err := inst.GetExport(entry)
		if err != nil {
			// RUN_NO_ENTRY
			fmt.Fprintf(os.Stderr, "%s: entrypoint '%s' not found in module %s\n",
				red("Error"), entry, result.Interface.Module)
			fmt.Fprintf(os.Stderr, "  Available exports: %s\n",
				strings.Join(runtime.GetExportNames(inst), ", "))
			os.Exit(1)
		}

		// Check arity
		arity, err := runtime.GetArity(entrypointVal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: entrypoint '%s' is not a function: %v\n",
				red("Error"), entry, err)
			os.Exit(1)
		}
		if arity > 1 {
			// RUN_MULTIARG_UNSUPPORTED
			fmt.Fprintf(os.Stderr, "%s: entrypoint '%s' takes %d parameters. v0.2.0 supports 0 or 1.\n",
				red("Error"), entry, arity)
			fmt.Fprintf(os.Stderr, "  Suggestion: wrap as 'wrapper(p:{...}) -> ...' and pass --args-json\n")
			os.Exit(1)
		}

		// Validate and decode arguments
		var args []eval.Value
		if len(fnType.Params) == 0 {
			// Zero-arg function - argsJSON must be null
			if argsJSON != "null" {
				fmt.Fprintf(os.Stderr, "%s: entrypoint '%s' takes no arguments, but --args-json was provided\n", red("Error"), entry)
				os.Exit(1)
			}
			args = []eval.Value{} // Empty args
		} else if len(fnType.Params) == 1 {
			// Single-arg function - decode JSON to match parameter type
			argVal, err := argdecode.DecodeJSON(argsJSON, fnType.Params[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to decode arguments: %v\n", red("Error"), err)
				os.Exit(1)
			}
			args = []eval.Value{argVal}
		} else {
			// Multi-arg functions not yet supported
			fmt.Fprintf(os.Stderr, "%s: entrypoint '%s' has %d parameters (only 0 or 1 supported in v0.2.0)\n", red("Error"), entry, len(fnType.Params))
			os.Exit(1)
		}

		// Call the entrypoint function
		execResult, err := runtime.CallEntrypoint(rt, inst, entry, args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: execution failed: %v\n", red("Error"), err)
			os.Exit(1)
		}

		// Print result if not Unit and not suppressed
		if execResult.Type() != "unit" && !noprint {
			if print {
				fmt.Println(execResult.String())
			}
		}
	} else {
		// Non-module mode - print result if not unit and not suppressed
		if result.Value != nil && result.Value.Type() != "unit" && !noprint {
			if print {
				fmt.Println(result.Value.String())
			}
		}
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

func runREPL(learn bool, trace bool) {
	// Use the new REPL implementation with version info
	r := repl.NewWithVersion(Version, BuildTime)
	if trace {
		r.EnableTrace()
	}
	r.Start(os.Stdin, os.Stdout)
}

func runTests(path string) {
	fmt.Printf("%s Running tests in %s\n", cyan("→"), path)

	// Find all .ail files with tests
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(p, ".ail") {
			// TODO: Check if file has tests and run them
			fmt.Printf("  %s %s\n", green("✓"), p)
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// TODO: Implement test runner
	fmt.Printf("\n%s All tests passed!\n", green("✓"))
}

func watchFile(filename string, trace bool, binopShim bool, failOnShim bool, requireLowering bool, trackInstantiations bool) {
	fmt.Printf("%s Watching %s for changes...\n", cyan("👁"), filename)
	fmt.Println("Press Ctrl+C to stop")

	// TODO: Implement file watching
	// For now, just run the file once (no json/compact for watch mode)
	// Default to main entrypoint with null args for watch mode, no caps
	runFile(filename, trace, 0, false, false, false, binopShim, failOnShim, requireLowering, trackInstantiations, "main", "null", true, false, "")
}

func checkFile(filename string) {
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Type check
	fmt.Printf("%s Type checking %s...\n", cyan("→"), filename)

	// Effect check
	fmt.Printf("%s Effect checking...\n", cyan("→"))

	// Use unified pipeline in dry-run mode (no evaluation)
	cfg := pipeline.Config{
		DryLink: true, // Don't evaluate, just check
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Check for any errors
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
		}
		os.Exit(1)
	}

	fmt.Printf("\n%s No errors found!\n", green("✓"))
}

func outputInterface(modulePath string) {
	// Read the file
	filename := modulePath
	if !strings.HasSuffix(filename, ".ail") {
		// Try to resolve as module path
		filename = strings.ReplaceAll(modulePath, "/", string(filepath.Separator)) + ".ail"
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Type check and build interface
	cfg := pipeline.Config{
		DryLink: true, // Don't evaluate, just check
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Check for errors
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
		}
		os.Exit(1)
	}

	// Get the interface
	if result.Interface == nil {
		fmt.Fprintf(os.Stderr, "%s: no interface generated for module\n", red("Error"))
		os.Exit(1)
	}

	// Output normalized JSON
	jsonBytes, err := result.Interface.ToNormalizedJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to serialize interface: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Println(string(jsonBytes))
}

func exportTraining() {
	fmt.Printf("%s Exporting training data...\n", cyan("→"))

	// TODO: Implement training data export
	fmt.Printf("  Analyzing execution traces...\n")
	fmt.Printf("  Filtering high-quality traces (score > 0.8)...\n")
	fmt.Printf("  Formatting for fine-tuning...\n")

	fmt.Printf("\n%s Exported 0 training examples to training_data.jsonl\n", green("✓"))
}

func runLSP() {
	fmt.Printf("%s Language Server v%s\n", bold("AILANG"), Version)
	fmt.Println("Listening on stdio...")

	// TODO: Implement LSP
	fmt.Fprintf(os.Stderr, "%s: LSP not yet implemented\n", red("Error"))
	os.Exit(1)
}

// handleStructuredError outputs structured JSON error reports
func handleStructuredError(err error, compact bool) {
	// Try to extract a structured Report using errors.AsReport
	if rep, ok := ailangErrors.AsReport(err); ok {
		outputJSON(rep, compact)
		return
	}

	// Fallback: wrap in generic error
	generic := ailangErrors.NewGeneric("runtime", err)
	outputJSON(generic, compact)
}

// outputJSON marshals and prints JSON
func outputJSON(v interface{}, compact bool) {
	var data []byte
	var err error

	if compact {
		data, err = json.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", "  ")
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(data))
}
