package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/sunholo/ailang/internal/agentprompt"
	"github.com/sunholo/ailang/internal/devtoolsprompt"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/observatory"
	"github.com/sunholo/ailang/internal/prompt"
	"github.com/sunholo/ailang/internal/schema"
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
	agentprompt.SetEmbeddedFS(embeddedPrompts)

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

	// Observatory health check — detect bloated DB early (M-OBS-RETENTION).
	// Fast path: just os.Stat, no DB open unless cleanup needed.
	observatory.CheckHealth(observatory.DefaultDatabasePath())

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
		packageFlag := testFlags.Bool("package", false, "Run tests in package mode (discovers *_test.ail via ailang.toml)")
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

		if *packageFlag {
			runPackageTests(path, *formatFlag, !*noColorFlag)
		} else {
			runTestsV2(path, *formatFlag, !*noColorFlag)
		}

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
		packageCheck := checkFS.Bool("package", false, "Check entire package (reads ailang.toml for module discovery)")

		_ = checkFS.Parse(flag.Args()[1:])

		// --package mode: check a package directory using ailang.toml
		if *packageCheck {
			dir := "."
			if checkFS.NArg() >= 1 {
				dir = checkFS.Arg(0)
			}
			checkPackageWithContext(dir, *strictSyntaxCheck, *relaxModulesCheck, *timeoutCheck, *debugCompileCheck, *jsonCheck, *quietCheck)
			return
		}

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
			fmt.Println("  --package          Check entire package (reads ailang.toml)")
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

	case "cache", "brain":
		cacheCommand()

	case "prompt":
		runPrompt()

	case "devtools-prompt":
		runDevtoolsPrompt()

	case "agent-prompt":
		runAgentPrompt()

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

	case "storage":
		if err := storageCommand(flag.Args()[1:]); err != nil {
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

	case "add":
		if err := pkgAddCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "lock":
		if err := pkgLockCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "tree":
		if err := pkgTreeCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "install":
		if err := pkgInstallCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "search":
		if err := pkgSearchCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "publish":
		if err := pkgPublishCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "pkg-docs":
		if err := pkgDocsCommand(flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

	case "pkg":
		// Sub-commands under "ailang pkg"
		subArgs := flag.Args()[1:]
		if len(subArgs) == 0 {
			fmt.Println("Usage: ailang pkg <command>")
			fmt.Println()
			fmt.Println("Commands:")
			fmt.Println("  notify-upgrade <pkg>@<ver>  Emit upgrade-available message")
			fmt.Println("  affected-by <pkg>           List workspaces depending on a package")
			fmt.Println("  provenance <pkg>@<ver>      Show provenance chain for a version")
			fmt.Println("  history <pkg>@<ver>         Show version history timeline")
			os.Exit(1)
		}
		switch subArgs[0] {
		case "notify-upgrade":
			if err := pkgNotifyUpgradeCommand(subArgs[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
				os.Exit(1)
			}
		case "affected-by":
			if err := pkgAffectedByCommand(subArgs[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
				os.Exit(1)
			}
		case "provenance":
			if err := pkgProvenanceCommand(subArgs[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
				os.Exit(1)
			}
		case "history":
			if err := pkgHistoryCommand(subArgs[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "%s: unknown pkg command '%s'\n", red("Error"), subArgs[0])
			os.Exit(1)
		}

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
