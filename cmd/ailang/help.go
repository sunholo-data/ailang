package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// checkStaleBinary warns if the binary is older than recent source changes
// This prevents confusion when testing changes with an old binary
func checkStaleBinary() {
	// Get the binary's executable path
	execPath, err := os.Executable()
	if err != nil {
		return // Can't determine, skip check
	}

	// Get binary modification time
	binaryInfo, err := os.Stat(execPath)
	if err != nil {
		return // Can't stat binary, skip check
	}
	binaryTime := binaryInfo.ModTime()

	// Check key source directories for recent changes
	// We check parser and elaborator since those are most commonly modified
	checkDirs := []string{
		"internal/parser",
		"internal/elaborate",
		"internal/eval",
		"cmd/ailang",
	}

	for _, dir := range checkDirs {
		// Walk directory to find most recent .go file
		newerFound := false
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if !info.IsDir() && strings.HasSuffix(path, ".go") {
				if info.ModTime().After(binaryTime) {
					newerFound = true
					return filepath.SkipDir // Found one, stop walking
				}
			}
			return nil
		})

		if newerFound {
			// Found source files newer than binary
			fmt.Fprintf(os.Stderr, "%s Binary may be stale (source files modified after build)\n", yellow("⚠"))
			fmt.Fprintf(os.Stderr, "  Run '%s' to rebuild\n", bold("make quick-install"))
			return // Only warn once
		}
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
	fmt.Printf("  %s             Run an AILANG program\n", cyan("run [flags] <file>"))
	fmt.Printf("  %s                       Start the interactive REPL\n", cyan("repl"))
	fmt.Printf("  %s                   Run tests\n", cyan("test [path]"))
	fmt.Printf("  %s           Watch file for changes and auto-reload\n", cyan("watch <file>"))
	fmt.Printf("  %s           Type-check a file without running\n", cyan("check <file>"))
	fmt.Printf("  %s        Output normalized JSON interface for a module\n", cyan("iface <module>"))
	fmt.Printf("  %s           Export training data\n", cyan("export-training"))
	fmt.Println()
	fmt.Println("Evaluation & Benchmarking:")
	fmt.Printf("  %s         Run AI benchmarks (AILANG vs Python)\n", cyan("eval [flags]"))
	fmt.Printf("  %s     Run full benchmark suite (parallel)\n", cyan("eval-suite [flags]"))
	fmt.Printf("  %s  Analyze eval results and generate design docs\n", cyan("eval-analyze [flags]"))
	fmt.Printf("  %s <results_dir> <version>   Generate comprehensive eval report\n", cyan("eval-report"))
	fmt.Printf("  %s <baseline> <new>    Compare two eval runs\n", cyan("eval-compare"))
	fmt.Printf("  %s <results_dir> <version>    Performance matrix with stats\n", cyan("eval-matrix"))
	fmt.Printf("  %s <results_dir>        Summarize eval results\n", cyan("eval-summary"))
	fmt.Printf("  %s <benchmark> [baseline]  Validate specific fix\n", cyan("eval-validate"))
	fmt.Println()
	fmt.Println("Development Tools:")
	fmt.Printf("  %s [--version V]   Display AILANG teaching prompt (for AI code generation)\n", cyan("prompt"))
	fmt.Printf("  %s                 Validate builtin registry\n", cyan("doctor builtins"))
	fmt.Printf("  %s [--by-effect|--by-module]  List all registered builtins\n", cyan("builtins list"))
	fmt.Printf("  %s      Debug AST and type information\n", cyan("debug ast [flags] <file>"))
	fmt.Printf("  %s    Install syntax highlighting (vscode, vim, neovim)\n", cyan("editor install <editor>"))
	fmt.Println()
	fmt.Println("Messages:")
	fmt.Printf("  %s                  List messages (alias: msg ls)\n", cyan("messages list"))
	fmt.Printf("  %s          List unread messages only\n", cyan("messages list --unread"))
	fmt.Printf("  %s         Send message to inbox\n", cyan("messages send <inbox> <msg>"))
	fmt.Printf("  %s             Mark message as read\n", cyan("messages ack <msg-id>"))
	fmt.Printf("  %s           Mark message as unread\n", cyan("messages unack <msg-id>"))
	fmt.Printf("  %s            Show message content\n", cyan("messages read <msg-id>"))
	fmt.Printf("  %s                 Watch for new messages\n", cyan("messages watch"))
	fmt.Printf("  %s               Clean up old messages\n", cyan("messages cleanup"))
	fmt.Println()
	fmt.Println("Collaboration:")
	fmt.Printf("  %s [--port PORT]         Start Collaboration Hub server (default port: 1957)\n", cyan("serve"))
	fmt.Println()
	fmt.Println("Run Command Flags (must come BEFORE filename):")
	fmt.Println("  --caps <list>        Enable capabilities (comma-separated: IO,FS,Net)")
	fmt.Println("  --entry <name>       Entrypoint function name (default: main)")
	fmt.Println("  --args-json <json>   JSON arguments to pass to entrypoint")
	fmt.Println("  --trace              Enable execution tracing")
	fmt.Println("  --print              Print return value (default: true)")
	fmt.Println("  --no-print           Suppress output (exit code only)")
	fmt.Println()
	fmt.Println("Global Flags:")
	fmt.Println("  --version            Print version information")
	fmt.Println("  --help               Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s                        # Start REPL\n", cyan("ailang repl"))
	fmt.Printf("  %s              # Run program with IO capability\n", cyan("ailang run --caps IO hello.ail"))
	fmt.Printf("  %s  # Run with custom entrypoint\n", cyan("ailang run --caps IO --entry test main.ail"))
	fmt.Printf("  %s                  # Type-check without running\n", cyan("ailang check src/"))
	fmt.Printf("  %s            # Run AI benchmark\n", cyan("ailang eval --benchmark fizzbuzz --mock"))
	fmt.Printf("  %s                          # Start Collaboration Hub\n", cyan("ailang serve"))
	fmt.Println()
	fmt.Println(yellow("Note: For 'run' command, flags must come BEFORE the filename"))
	fmt.Println(yellow("      Example: ailang run --caps IO file.ail  (NOT: ailang run file.ail --caps IO)"))
}
