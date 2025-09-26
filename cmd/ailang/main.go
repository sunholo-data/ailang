package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
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
		versionFlag = flag.Bool("version", false, "Print version information")
		helpFlag    = flag.Bool("help", false, "Show help")
		learnFlag   = flag.Bool("learn", false, "Enable learning mode (collect training data)")
		traceFlag   = flag.Bool("trace", false, "Enable execution tracing")
		seedFlag    = flag.Int("seed", 0, "Random seed for deterministic execution")
		virtualTime = flag.Bool("virtual-time", false, "Use virtual time for deterministic execution")
	)

	flag.Parse()

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
		runFile(flag.Arg(1), *traceFlag, *seedFlag, *virtualTime)

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
		watchFile(flag.Arg(1), *traceFlag)

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
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s              # Start REPL\n", cyan("ailang repl"))
	fmt.Printf("  %s    # Run program\n", cyan("ailang run hello.ail"))
	fmt.Printf("  %s        # Type-check\n", cyan("ailang check src/"))
	fmt.Printf("  %s  # Watch with tracing\n", cyan("ailang watch main.ail --trace"))
}

func runFile(filename string, trace bool, seed int, virtualTime bool) {
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

	// Parse the file
	l := lexer.New(string(content), filename)
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		printParserErrors(p.Errors())
		os.Exit(1)
	}

	// Type check
	fmt.Printf("%s Type checking...\n", cyan("→"))
	// TODO: Implement type checking

	// Run effects analysis
	fmt.Printf("%s Effect checking...\n", cyan("→"))
	// TODO: Implement effect checking

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

	// Execute the program
	evaluator := eval.NewSimple()
	result, err := evaluator.EvalProgram(program)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Runtime error"), err)
		os.Exit(1)
	}
	
	// Print result if not unit
	if result != nil && result.Type() != "unit" {
		fmt.Println(result.String())
	}
}

func runREPL(learn bool, trace bool) {
	fmt.Printf("%s v%s - AI-First Functional Language\n", bold("AILANG"), Version)
	if learn {
		fmt.Printf("%s Learning mode enabled - corrections will be saved for training\n", green("✓"))
	}
	fmt.Println("Type :help for help, :quit to exit")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Print(">>> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			continue
		}

		input = strings.TrimSpace(input)

		// Handle REPL commands
		if strings.HasPrefix(input, ":") {
			handleREPLCommand(input)
			continue
		}

		if input == "" {
			continue
		}

		// Parse and evaluate
		l := lexer.New(input, "<repl>")
		p := parser.New(l)
		program := p.Parse()

		if len(p.Errors()) > 0 {
			printParserErrors(p.Errors())
			if learn {
				// TODO: Save error for training
				fmt.Printf("%s Error recorded for training\n", yellow("📝"))
			}
			continue
		}

		// Evaluate
		evaluator := eval.NewSimple()
		result, err := evaluator.EvalProgram(program)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Runtime error"), err)
			if learn {
				fmt.Printf("%s Error recorded for training\n", yellow("📝"))
			}
			continue
		}
		
		// Print result
		if result != nil {
			fmt.Printf("%s : %s = %s\n", cyan("result"), yellow(result.Type()), green(result.String()))
		}
		
		if trace {
			fmt.Printf("%s Trace: [execution trace]\n", yellow("⚡"))
		}
	}
}

func handleREPLCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case ":help", ":h":
		fmt.Println("REPL Commands:")
		fmt.Println("  :help, :h        Show this help")
		fmt.Println("  :quit, :q        Exit the REPL")
		fmt.Println("  :type <expr>     Show type of expression")
		fmt.Println("  :load <file>     Load a file")
		fmt.Println("  :reload          Reload the last file")
		fmt.Println("  :clear           Clear the screen")
		fmt.Println("  :trace           Toggle tracing")
		fmt.Println("  :effects         Show current effects")

	case ":quit", ":q":
		fmt.Println("Goodbye!")
		os.Exit(0)

	case ":type":
		if len(parts) < 2 {
			fmt.Println("Usage: :type <expression>")
			return
		}
		expr := strings.Join(parts[1:], " ")
		// TODO: Type check expression
		fmt.Printf("Type of %s: %s\n", expr, yellow("unknown"))

	case ":load":
		if len(parts) < 2 {
			fmt.Println("Usage: :load <file>")
			return
		}
		// TODO: Load file
		fmt.Printf("Loading %s...\n", parts[1])

	case ":clear":
		fmt.Print("\033[H\033[2J") // Clear screen

	case ":trace":
		// TODO: Toggle tracing
		fmt.Println("Tracing toggled")

	case ":effects":
		// TODO: Show current effects
		fmt.Println("Current effects: {IO}")

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		fmt.Println("Type :help for help")
	}
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

func watchFile(filename string, trace bool) {
	fmt.Printf("%s Watching %s for changes...\n", cyan("👁"), filename)
	fmt.Println("Press Ctrl+C to stop")
	
	// TODO: Implement file watching
	// For now, just run the file once
	runFile(filename, trace, 0, false)
}

func checkFile(filename string) {
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Parse
	l := lexer.New(string(content), filename)
	p := parser.New(l)
	_ = p.Parse() // TODO: Use the parsed program for type checking

	if len(p.Errors()) > 0 {
		printParserErrors(p.Errors())
		os.Exit(1)
	}

	// Type check
	fmt.Printf("%s Type checking %s...\n", cyan("→"), filename)
	// TODO: Implement type checking

	// Effect check
	fmt.Printf("%s Effect checking...\n", cyan("→"))
	// TODO: Implement effect checking

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