package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/pipeline"
	ailangTesting "github.com/sunholo-data/ailang/internal/testing"
)

// languageCommands are the commands a user or agent runs to work WITH AILANG.
// Membership is a contract, not a convenience: a language command opens no
// observatory DB and runs no git probe at startup (M-V1-SIMPLIFY-S1 M6). The
// list here is the pre-S5 isLanguageCommand() switch, unchanged —
// commands_probe_test.go proves it by counting probes.
func languageCommands() []Command {
	return []Command{
		{
			Name:    "version",
			Group:   groupLanguage,
			Summary: "Show version, commit hash and build time",
			Run: func([]string) error {
				printVersion()
				return nil
			},
		},
		{
			Name:    "run",
			Group:   groupLanguage,
			Summary: "Run an AILANG program",
			Run:     noArgs(runCommand),
		},
		{
			Name:    "repl",
			Group:   groupLanguage,
			Summary: "Start the interactive REPL",
			Run: func([]string) error {
				runREPL(globals.learn, globals.trace, globals.strictSyntax)
				return nil
			},
		},
		{
			Name:    "test",
			Group:   groupLanguage,
			Summary: "Run tests (--package for package mode)",
			Run:     runTestCommand,
		},
		{
			Name:    "watch",
			Group:   groupLanguage,
			Summary: "Watch a file for changes and re-run it",
			Run:     runWatchCommand,
		},
		{
			Name:    "check",
			Group:   groupLanguage,
			Summary: "Type-check a file or directory without running it",
			Run:     runCheckCommand,
		},
		{
			Name:    "fmt",
			Group:   groupLanguage,
			Summary: "Format AILANG source (stdout / --write / --check)",
			Run: func(args []string) error {
				runFmtCommand(args)
				return nil
			},
		},
		{
			Name:    "ai-check",
			Group:   groupLanguage,
			Summary: "Unified check+verify JSON output (for AI agents)",
			Run:     noArgs(aiCheckCommand),
		},
		{
			Name:    "iface",
			Group:   groupLanguage,
			Summary: "Print the normalized JSON interface of a module",
			Run:     runIfaceCommand,
		},
		{
			Name:    "internal-dump-iface",
			Group:   groupLanguage,
			Summary: "Dump a package's canonical interface JSON (internal)",
			Run:     runDumpIfaceCommand,
		},
		{
			Name:    "verify",
			Group:   groupLanguage,
			Summary: "Verify contracts and proof obligations",
			Run:     noArgs(verifyCommand),
		},
		{
			Name:    "compile",
			Group:   groupLanguage,
			Summary: "Compile AILANG to Go (emit-go)",
			Run:     noArgs(compileCommand),
		},
		{
			Name:    "disasm",
			Group:   groupLanguage,
			Summary: "Disassemble compiled AILANG",
			Run:     noArgs(disasmCommand),
		},
		{
			Name:    "debug",
			Group:   groupLanguage,
			Summary: "Debug AST and type information",
			Run:     noArgs(runDebug),
		},
		{
			Name:    "prompt",
			Group:   groupLanguage,
			Summary: "Display the AILANG teaching prompt (for AI code generation)",
			Run:     noArgs(runPrompt),
		},
		{
			Name:    "devtools-prompt",
			Group:   groupLanguage,
			Summary: "Display the AILANG dev tools reference (for AI agents)",
			Run:     noArgs(runDevtoolsPrompt),
		},
		{
			Name:    "docs",
			Group:   groupLanguage,
			Summary: "Show stdlib module documentation (--list to enumerate)",
			Run:     noArgs(docsCommand),
		},
		{
			Name:    "builtins",
			Group:   groupLanguage,
			Summary: "Inspect the builtin registry",
			Run:     noArgs(runBuiltins),
		},
		{
			Name:    "examples",
			Group:   groupLanguage,
			Summary: "Search and explore working code examples",
			Run: func(args []string) error {
				examplesCommand(args)
				return nil
			},
		},
		{
			Name:    "lsp",
			Group:   groupLanguage,
			Summary: "Language Server Protocol server (for AI agents and IDEs)",
			Run:     lspCommand,
		},
		{
			Name:    "init",
			Group:   groupLanguage,
			Summary: "Scaffold a new AILANG package or web app",
			Run:     initCommand,
		},
		{
			Name:    "select-best",
			Group:   groupLanguage,
			Summary: "Pick the best candidate from generated variants",
			Run:     noArgs(runSelectBest),
		},
		{
			Name:    "ast-edit",
			Group:   groupLanguage,
			Summary: "Structural edit of an AILANG declaration",
			Run:     noArgs(runAstEdit),
		},
		{
			Name:    "replay",
			Group:   groupLanguage,
			Summary: "Replay and verify against a recorded trace",
			Run:     noArgs(replayCommand),
		},
		{
			Name:    "export-training",
			Group:   groupLanguage,
			Summary: "Export traces as AI training data",
			Run:     noArgs(exportTraining),
		},
	}
}

// runTestCommand is the `ailang test` entry point, moved verbatim out of
// main.go's switch. args is the tail after "test".
func runTestCommand(args []string) error {
	testFlags := flag.NewFlagSet("test", flag.ExitOnError)
	formatFlag := testFlags.String("format", "human", "Output format: human or json")
	noColorFlag := testFlags.Bool("no-color", false, "Disable colored output")
	packageFlag := testFlags.Bool("package", false, "Run tests in package mode (discovers *_test.ail via ailang.toml)")
	allowSkipsFlag := testFlags.Bool("allow-skips", false, "Exit 0 even when all tests are skipped (default: skipped-only suites exit 1)")
	seedFlag := testFlags.Int64("seed", 0, "Master seed for property generation (signed int64)")
	randomSeedFlag := testFlags.Bool("random-seed", false, "Read one master seed from crypto/rand and report it")
	helpTestFlag := testFlags.Bool("help", false, "Show help for test command")

	_ = testFlags.Parse(args) // Parse errors handled by flags package

	if *helpTestFlag {
		printTestHelp()
		return nil
	}

	// Both flags are registered above so the CLI accepts them. D1 mandates
	// detecting PRESENCE via Visit — presence, not value — so the returned
	// randomSeedFlag value is intentionally unused (--seed 0 must not be
	// confused with unset); blank it rather than dereference it.
	_ = randomSeedFlag

	// Detect flag PRESENCE via Visit (D1) — presence, not value, because
	// --seed 0 is a legitimate explicit master seed and must not be confused
	// with "unset". Visit iterates only flags that were set on the command line.
	seedSet, randomSet := false, false
	testFlags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "seed":
			seedSet = true
		case "random-seed":
			randomSet = true
		}
	})
	if seedSet && randomSet { // D2 — mutual exclusion on stderr, exit 2
		fmt.Fprintln(os.Stderr, "Error: --seed and --random-seed cannot be used together")
		os.Exit(2)
	}
	// Capture os.Getwd() ONCE, before any path walking.
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	cfg := ailangTesting.TestConfig{WorkspaceRoot: cwd, SeedMode: ailangTesting.SeedModeDerived, MasterSeed: 0}
	switch {
	case seedSet:
		cfg.SeedMode, cfg.MasterSeed = ailangTesting.SeedModeMaster, *seedFlag
	case randomSet:
		m, err := ailangTesting.NewRandomMasterSeed()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1) // BEFORE any property runs
		}
		cfg.SeedMode, cfg.MasterSeed = ailangTesting.SeedModeMaster, m
	}

	path := "."
	if testFlags.NArg() >= 1 {
		path = testFlags.Arg(0)
	}

	// Record the CLI argument tail that reproduces this run, shell-safe, so
	// the emitted replay command is runnable (defect §3(A)); see
	// replayTargetArg for the quoting rules.
	if *packageFlag {
		cfg.ReplayTarget = "--package " + replayTargetArg(path)
	} else {
		cfg.ReplayTarget = replayTargetArg(path)
	}

	if *packageFlag {
		runPackageTests(path, *formatFlag, !*noColorFlag, *allowSkipsFlag, cfg)
	} else {
		runTestsV2(path, *formatFlag, !*noColorFlag, *allowSkipsFlag, cfg)
	}
	return nil
}

// replayTargetArg renders a CLI argument tail that reproduces a test run,
// shell-safe. A space, single quote, or double quote triggers single-quote
// wrapping with embedded single quotes escaped as '\”; otherwise the argument
// is emitted bare. It lives in ONE helper (used only by the test subcommand)
// so the package mode and ordinary mode targets cannot diverge.
func replayTargetArg(path string) string {
	if strings.ContainsAny(path, " '\"") {
		return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
	}
	return path
}

// runWatchCommand is `ailang watch`, moved verbatim out of main.go's switch.
func runWatchCommand(args []string) error {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		fmt.Println("Usage: ailang watch <file.ail>")
		os.Exit(1)
	}
	watchFile(args[0], globals.trace, globals.binopShim, globals.failOnShim, globals.requireLowering,
		globals.trackInstantiations, globals.noMono, globals.debugCompile, globals.maxRecursionDepth)
	return nil
}

// runCheckCommand is `ailang check`, moved verbatim out of main.go's switch.
func runCheckCommand(args []string) error {
	checkFS := flag.NewFlagSet("check", flag.ExitOnError)
	strictSyntaxCheck := checkFS.Bool("strict-syntax", false, "Disable syntactic sugar (require canonical syntax)")
	relaxModulesCheck := checkFS.Bool("relax-modules", false, "Relax MOD010 validation (allow module path mismatches with warning)")
	timeoutCheck := checkFS.String("timeout", "", "Compilation timeout (e.g., 30s, 2m). Dumps stack on timeout.")
	debugCompileCheck := checkFS.Bool("debug-compile", false, "Show compilation phase timing breakdown")
	jsonCheck := checkFS.Bool("json", false, "Output errors in JSON format (for AI/machine consumption)")
	formatCheck := checkFS.String("format", "human", "Output format: human, json, or agent (compact one-line diagnostics for AI agent context)")
	quietCheck := checkFS.Bool("quiet", false, "Suppress progress lines, only output errors")
	packageCheck := checkFS.Bool("package", false, "Check entire package (reads ailang.toml for module discovery)")

	_ = checkFS.Parse(args)

	// Resolve output format (M-AILANG-SEMANTIC-CONTEXT R1). --json is the
	// back-compat alias for --format=json. Both json and agent are "machine"
	// formats that reuse the structured-error path; agent is the compact,
	// token-lean one-line rendering for AI agent loops.
	checkAgentFormat = (*formatCheck == "agent")
	machineFormat := *jsonCheck || *formatCheck == "json" || *formatCheck == "agent"

	// --package mode: check a package directory using ailang.toml
	if *packageCheck {
		dir := "."
		if checkFS.NArg() >= 1 {
			dir = checkFS.Arg(0)
		}
		checkPackageWithContext(dir, *strictSyntaxCheck, *relaxModulesCheck, *timeoutCheck, *debugCompileCheck, machineFormat, *quietCheck)
		return nil
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
		fmt.Println("  --format <fmt>     Output format: human (default), json, or agent (compact one-line diagnostics)")
		fmt.Println("  --quiet            Suppress progress lines, only output errors")
		fmt.Println("  --package          Check entire package (reads ailang.toml)")
		fmt.Println()
		fmt.Println("If a directory is given, all .ail files are checked recursively.")
		os.Exit(1)
	}
	checkFile(checkFS.Arg(0), *strictSyntaxCheck, *relaxModulesCheck, *timeoutCheck, *debugCompileCheck, machineFormat, *quietCheck)
	return nil
}

// runIfaceCommand is `ailang iface`, moved verbatim out of main.go's switch.
func runIfaceCommand(args []string) error {
	ifaceFS := flag.NewFlagSet("iface", flag.ExitOnError)
	ifaceCompact := ifaceFS.Bool("compact", false, "Compact one-line-per-export signatures (dense typed-interface view for agent context)")
	_ = ifaceFS.Parse(args)
	if ifaceFS.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing module argument\n", red("Error"))
		fmt.Println("Usage: ailang iface [--compact] <module>")
		os.Exit(1)
	}
	outputInterface(ifaceFS.Arg(0), *ifaceCompact)
	return nil
}

// runDumpIfaceCommand is `ailang internal-dump-iface`, moved verbatim out of
// main.go's switch.
func runDumpIfaceCommand(args []string) error {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: ailang internal-dump-iface <package-dir> <module-path>")
		os.Exit(1)
	}
	jsonBytes, err := pipeline.BuildCanonicalJSON(context.Background(), args[0], args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Println(string(jsonBytes))
	return nil
}
