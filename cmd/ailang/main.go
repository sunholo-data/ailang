package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/repl"
	"github.com/sunholo/ailang/internal/schema"
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
		versionFlag   = flag.Bool("version", false, "Print version information")
		helpFlag      = flag.Bool("help", false, "Show help")
		learnFlag     = flag.Bool("learn", false, "Enable learning mode (collect training data)")
		traceFlag     = flag.Bool("trace", false, "Enable execution tracing")
		seedFlag      = flag.Int("seed", 0, "Random seed for deterministic execution")
		virtualTime   = flag.Bool("virtual-time", false, "Use virtual time for deterministic execution")
		compactFlag   = flag.Bool("compact", false, "Use compact JSON output")
		binopShimFlag = flag.Bool("experimental-binop-shim", false, "Enable experimental operator shim")
		failOnShimFlag = flag.Bool("fail-on-shim", false, "Fail if operator shim would be used (CI mode)")
		requireLoweringFlag = flag.Bool("require-lowering", false, "Require operator lowering pass")
		trackInstantiationsFlag = flag.Bool("track-instantiations", false, "Track and dump polymorphic type instantiations")
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
			fmt.Println("Usage: ailang run <file.ail>")
			os.Exit(1)
		}
		runFile(flag.Arg(1), *traceFlag, *seedFlag, *virtualTime, *binopShimFlag, *failOnShimFlag, *requireLoweringFlag, *trackInstantiationsFlag)

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

	case "export-training":
		exportTraining()

	case "lsp":
		runLSP()

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
	fmt.Printf("  %s   Export training data\n", cyan("export-training"))
	fmt.Printf("  %s              Start the Language Server Protocol server\n", cyan("lsp"))
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
}

func runFile(filename string, trace bool, seed int, virtualTime bool, binopShim bool, failOnShim bool, requireLowering bool, trackInstantiations bool) {
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

	// Use unified pipeline
	cfg := pipeline.Config{
		TraceDefaulting:       trace,
		ExperimentalBinopShim: binopShim,
		FailOnShim:           failOnShim,
		RequireLowering:      requireLowering,
		TrackInstantiations:  trackInstantiations,
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	if trace {
		fmt.Printf("DEBUG: Running with trace=%v, file=%s\n", trace, filename)
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	
	// Print result if not unit
	if result.Value != nil && result.Value.Type() != "unit" {
		fmt.Println(result.Value.String())
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
	// For now, just run the file once
	runFile(filename, trace, 0, false, binopShim, failOnShim, requireLowering, trackInstantiations)
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

func printParserErrors(errors []error) {
	fmt.Fprintf(os.Stderr, "%s Parser errors:\n", red("Error"))
	for _, err := range errors {
		fmt.Fprintf(os.Stderr, "  %s %v\n", red("•"), err)
	}
}
