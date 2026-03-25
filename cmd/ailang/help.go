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
		// Show short hash on main line, full hash on next
		short := Commit
		if len(Commit) > 7 {
			short = Commit[:7]
		}
		fmt.Printf("Commit: %s\n", short)
		if len(Commit) > 7 {
			fmt.Printf("Full:   %s\n", Commit)
		}
	}
	if BuildTime != "unknown" {
		fmt.Printf("Built:  %s\n", BuildTime)
	}
	fmt.Println("\nThe AI-First Programming Language")
	fmt.Println("Copyright (c) 2025-2026")
}

func printHelp() {
	fmt.Println(bold("AILANG - The AI-First Programming Language"))
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ailang <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Printf("  %s                    Show version, commit hash, and build time\n", cyan("version"))
	fmt.Printf("  %s             Run an AILANG program\n", cyan("run [flags] <file>"))
	fmt.Printf("  %s                       Start the interactive REPL\n", cyan("repl"))
	fmt.Printf("  %s                   Run tests\n", cyan("test [path]"))
	fmt.Printf("  %s           Watch file for changes and auto-reload\n", cyan("watch <file>"))
	fmt.Printf("  %s           Type-check a file without running\n", cyan("check <file>"))
	fmt.Printf("  %s      Unified check+verify JSON output (for AI)\n", cyan("ai-check <file>"))
	fmt.Printf("  %s        Output normalized JSON interface for a module\n", cyan("iface <module>"))
	fmt.Printf("  %s  Export traces as AI training data\n", cyan("export-training <traces>"))
	fmt.Printf("  %s  Replay and verify against recorded trace\n", cyan("replay <trace.jsonl>"))
	fmt.Println()
	fmt.Println("Evaluation & Benchmarking:")
	fmt.Printf("  %s         Run AI benchmarks (AILANG vs Python)\n", cyan("eval [flags]"))
	fmt.Printf("  %s     Run full benchmark suite (parallel)\n", cyan("eval-suite [flags]"))
	fmt.Printf("  %s  Analyze eval results and generate design docs\n", cyan("eval-analyze [flags]"))
	fmt.Printf("  %s <results_dir> <version>   Generate comprehensive eval report\n", cyan("eval-report"))
	fmt.Printf("  %s <baseline> <new>    Compare two eval runs\n", cyan("eval-compare"))
	fmt.Printf("  %s <results_dir> <version>    Performance matrix with stats\n", cyan("eval-matrix"))
	fmt.Printf("  %s <results_dir>        Summarize eval results\n", cyan("eval-summary"))
	fmt.Println()
	fmt.Println("Development Tools:")
	fmt.Printf("  %s [--version V]   Display AILANG teaching prompt (for AI code generation)\n", cyan("prompt"))
	fmt.Printf("  %s [--version V]   Display AILANG dev tools reference (for AI agents)\n", cyan("devtools-prompt"))
	fmt.Printf("  %s                     List available stdlib modules\n", cyan("docs --list"))
	fmt.Printf("  %s               Show stdlib module documentation\n", cyan("docs <module>"))
	fmt.Printf("  %s                 Validate builtin registry\n", cyan("doctor builtins"))
	fmt.Printf("  %s [--by-effect|--by-module]  List all registered builtins\n", cyan("builtins list"))
	fmt.Printf("  %s      Debug AST and type information\n", cyan("debug ast [flags] <file>"))
	fmt.Printf("  %s    Install syntax highlighting (vscode, vim, neovim)\n", cyan("editor install <editor>"))
	fmt.Printf("  %s              Show design axiom compliance scorecard\n", cyan("axioms [--json]"))
	fmt.Printf("  %s            Search and explore working code examples\n", cyan("examples <cmd>"))
	fmt.Println()
	fmt.Println("Packages & Registry:")
	fmt.Printf("  %s          Search registry by keyword or tag\n", cyan("search [query] [--tag TAG]"))
	fmt.Printf("  %s  Install package (omit version for latest)\n", cyan("install <vendor/name[@version]>"))
	fmt.Printf("  %s                    Publish current package to registry\n", cyan("publish [--dry-run]"))
	fmt.Printf("  %s       Display package AGENT.md (AI usage guide)\n", cyan("pkg-docs <vendor/name>"))
	fmt.Printf("  %s  Create ailang.toml for a new package\n", cyan("init package --name <v/n>"))
	fmt.Printf("  %s              Add dependency (--path, --git, or --registry)\n", cyan("add [flags]"))
	fmt.Printf("  %s                       Resolve dependencies, generate lockfile\n", cyan("lock"))
	fmt.Printf("  %s                       Show dependency tree\n", cyan("tree"))
	fmt.Println()
	fmt.Println("Package Coordination:")
	fmt.Printf("  %s  Emit upgrade-available message\n", cyan("pkg notify-upgrade <pkg>@<ver>"))
	fmt.Printf("  %s       List workspaces depending on a package\n", cyan("pkg affected-by <pkg>"))
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
	fmt.Println("Web & API:")
	fmt.Printf("  %s [flags] <path...>  Serve AILANG exports as REST endpoints\n", cyan("serve-api"))
	fmt.Printf("  %s [name]       Scaffold a new AILANG web app (API + React)\n", cyan("init web-app"))
	fmt.Println()
	fmt.Println("WebAssembly (Browser):")
	fmt.Println("  Download ailang-wasm.tar.gz from GitHub releases:")
	fmt.Printf("    %s\n", cyan("gh release download --repo sunholo-data/ailang -p 'ailang-wasm.tar.gz'"))
	fmt.Println("  JavaScript API (window globals):")
	fmt.Println("    ailangEval(input)              Evaluate expression or :command")
	fmt.Println("    ailangReset()                  Reset REPL environment")
	fmt.Println("    ailangVersion()                Returns {version, buildTime, platform}")
	fmt.Println("    ailangLoadModule(name, code)   Load module, returns {success, exports?, error?}")
	fmt.Println("    ailangListModules()            Returns string[] of loaded module names")
	fmt.Println("    ailangCall(mod, func, ...args) Call function, returns {success, result?, error?}")
	fmt.Println("  JS Wrapper (web/ailang-repl.js):")
	fmt.Println("    repl.eval(input)               Evaluate expression")
	fmt.Println("    repl.loadModule(name, code)    Load AILANG module")
	fmt.Println("    repl.listModules()             List loaded modules")
	fmt.Println("    repl.importModule(name)        Import module into scope")
	fmt.Println("    repl.call(mod, func, ...args)  Call function with auto-import")
	fmt.Println()
	fmt.Println("Observatory & Collaboration:")
	fmt.Printf("  %s [--port PORT]        Start Observatory dashboard (default port: 1957)\n", cyan("server"))
	fmt.Printf("  %s                       Alias for server (backward compat)\n", cyan("serve"))
	fmt.Println()
	fmt.Println("Agent Coordination:")
	fmt.Printf("  %s <cmd>          Manage autonomous agent daemon (14 subcommands)\n", cyan("coordinator"))
	fmt.Printf("                          Use '%s' for full command list\n", yellow("ailang coordinator --help"))
	fmt.Printf("    %s       Start daemon\n", cyan("coordinator start"))
	fmt.Printf("    %s        Stop daemon\n", cyan("coordinator stop"))
	fmt.Printf("    %s      Show daemon status\n", cyan("coordinator status"))
	fmt.Printf("    %s     Interactive approval queue\n", cyan("coordinator pending"))
	fmt.Printf("    %s <task-id>     Show git diff of changes\n", cyan("coordinator diff"))
	fmt.Printf("    %s <task-id>     Show streaming execution logs\n", cyan("coordinator logs"))
	fmt.Printf("  %s <cmd>              View execution chains (task→session→chat linkage)\n", cyan("chains"))
	fmt.Printf("    %s                  List all chains\n", cyan("chains list"))
	fmt.Printf("    %s <chain-id>       View chain stages and details\n", cyan("chains view"))
	fmt.Printf("    %s <chain-id>       ASCII tree with chat history\n", cyan("chains tree"))
	fmt.Printf("    %s <chain-id>  Quick health report for a chain\n", cyan("chains diagnose"))
	fmt.Printf("    %s              System-wide data capture validation\n", cyan("chains health"))
	fmt.Printf("  %s <cmd>            Dashboard operations for task visualization\n", cyan("dashboard"))
	fmt.Printf("  %s <cmd>           Workspace management\n", cyan("workspaces"))
	fmt.Println()
	fmt.Println("Telemetry & Debugging:")
	fmt.Printf("  %s <cmd>               Distributed trace management (4 subcommands)\n", cyan("trace"))
	fmt.Printf("    %s            Show telemetry configuration\n", cyan("trace status"))
	fmt.Printf("    %s [--hours 1]     List recent traces (requires GCP)\n", cyan("trace list"))
	fmt.Printf("    %s <trace-id>      View trace hierarchy\n", cyan("trace view"))
	fmt.Printf("    %s [--limit 200]   Local span hierarchy\n", cyan("trace hierarchy"))
	fmt.Printf("  %s <cmd>         Observatory analytics (10+ subcommands)\n", cyan("observatory"))
	fmt.Printf("    %s <id>   View unified task/message/span hierarchy\n", cyan("observatory hierarchy"))
	fmt.Printf("    %s [--days 7]   Activity heatmap\n", cyan("observatory heatmap"))
	fmt.Printf("    %s            Token distribution histogram\n", cyan("observatory tokens"))
	fmt.Printf("  %s <cmd>              Budget monitoring (3 subcommands)\n", cyan("budget"))
	fmt.Printf("    %s [--json]        Show budget status\n", cyan("budget status"))
	fmt.Printf("    %s [--cost N]       Check if operation would exceed budget\n", cyan("budget check"))
	fmt.Println()
	fmt.Println("Advanced Tools:")
	fmt.Printf("  %s [flags]               Unified AI execution for programmatic use\n", cyan("exec"))
	fmt.Printf("  %s [flags] <file>       Compile AILANG to Go (emit-go)\n", cyan("compile"))
	fmt.Printf("  %s <cmd>        Access control management\n", cyan("access-control"))
	fmt.Println()
	fmt.Println("Run Command Flags (must come BEFORE filename):")
	fmt.Println("  --caps <list>        Enable capabilities (comma-separated: IO,FS,Net,AI)")
	fmt.Println("  --ai <model>         Enable AI effect with model (e.g., gemini-2-5-flash)")
	fmt.Println("  --ai-stub            Enable AI effect with stub handler (for testing)")
	fmt.Println("  --entry <name>       Entrypoint function name (default: main)")
	fmt.Println("  --args-json <json>   JSON arguments to pass to entrypoint")
	fmt.Println("  --trace              Enable execution tracing")
	fmt.Println("  --print              Print return value (default: true)")
	fmt.Println("  --no-print           Suppress output (exit code only)")
	fmt.Println("  --no-budgets         Bypass effect budget enforcement (allow unlimited operations)")
	fmt.Println("  --budget-report=MODE Print budget usage after execution (flat, json)")
	fmt.Println()
	fmt.Println("Global Flags:")
	fmt.Println("  --version            Print version information")
	fmt.Println("  --help               Show this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println()
	fmt.Println("AI Provider Authentication:")
	fmt.Println("  The --ai flag selects the model; authentication is via environment variables.")
	fmt.Println()
	fmt.Println("  Anthropic (claude-*):  ANTHROPIC_API_KEY=sk-ant-...")
	fmt.Println("  OpenAI (gpt-*):        OPENAI_API_KEY=sk-...")
	fmt.Println("  Ollama (ollama:*):     No key needed (local, http://localhost:11434)")
	fmt.Println()
	fmt.Println("  Google Gemini (gemini-*) supports TWO authentication methods:")
	fmt.Println("    Option 1 - API Key (AI Studio, simplest):")
	fmt.Println("      export GOOGLE_API_KEY=AIza...")
	fmt.Println("      Get one at: https://aistudio.google.com/apikey")
	fmt.Println()
	fmt.Println("    Option 2 - Application Default Credentials (Vertex AI):")
	fmt.Println("      gcloud auth application-default login")
	fmt.Println("      GOOGLE_API_KEY must be UNSET for Vertex AI to be used.")
	fmt.Println("      If GOOGLE_API_KEY is set, AI Studio is always used instead.")
	fmt.Println("      Use: unset GOOGLE_API_KEY  (or: GOOGLE_API_KEY= ailang run ...)")
	fmt.Println()
	fmt.Println("Debug Flags (for troubleshooting):")
	fmt.Println("  DEBUG_STRICT=1               Fail loudly on unhandled cases (recommended for CI)")
	fmt.Println("  DEBUG_MONO_VERBOSE=1         Trace monomorphization (type issues)")
	fmt.Println("  DEBUG_PARSER=1               Token position tracing (parser bugs)")
	fmt.Println("  DEBUG_CODEGEN=1              Warn on record fallback to map")
	fmt.Println("  DEBUG_APPROVAL_WATCHER=1     Verbose GitHub label detection")
	fmt.Println("  DEBUG_CONCURRENCY=1          serve-api per-request eval tracing with goroutine IDs")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  AILANG_RELAX_MODULES=1              Allow module path mismatch (prototyping)")
	fmt.Println("  AILANG_PARENT_TASK_ID=<id>          Set parent task for trace hierarchy")
	fmt.Println("  OTEL_EXPORTER_OTLP_ENDPOINT=<url>   Enable OTLP telemetry export")
	fmt.Println("  GOOGLE_CLOUD_PROJECT=<id>           Enable GCP Cloud Trace")
	fmt.Println()
	fmt.Println("Registry:")
	fmt.Println("  AILANG_REGISTRY=<url>               Registry URL (default: GCS bucket)")
	fmt.Println("  AILANG_REGISTRY_VALIDATOR=<url>     Cloud Run validator endpoint (for publish)")
	fmt.Println("  AILANG_REGISTRY_API_KEY=<key>       API key for publish/rebuild-index")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println()
	fmt.Println("Basic Usage:")
	fmt.Printf("  %s                        # Start REPL\n", cyan("ailang repl"))
	fmt.Printf("  %s              # Run program with IO capability\n", cyan("ailang run --caps IO hello.ail"))
	fmt.Printf("  %s  # Run with custom entrypoint\n", cyan("ailang run --caps IO --entry test main.ail"))
	fmt.Printf("  %s                  # Type-check without running\n", cyan("ailang check src/"))
	fmt.Println()
	fmt.Println("Evaluation & Benchmarking:")
	fmt.Printf("  %s            # Run AI benchmark\n", cyan("ailang eval --benchmark fizzbuzz --mock"))
	fmt.Printf("  %s  # Run full suite with multiple models\n", cyan("ailang eval-suite --models gpt5,claude-sonnet-4-6"))
	fmt.Printf("  %s                  # Resume interrupted baseline\n", cyan("ailang eval-suite --full --skip-existing"))
	fmt.Println()
	fmt.Println("Agent Coordination:")
	fmt.Printf("  %s                 # Start autonomous agent daemon\n", cyan("ailang coordinator start"))
	fmt.Printf("  %s                # Interactive approval UI\n", cyan("ailang coordinator pending"))
	fmt.Printf("  %s      # Send task to coordinator\n", cyan("ailang messages send coordinator \"Fix bug\" --type bug"))
	fmt.Printf("  %s       # View execution chain with chat history\n", cyan("ailang chains tree <chain-id> --json"))
	fmt.Println()
	fmt.Println("Packages & Registry:")
	fmt.Printf("  %s                  # Search for auth packages\n", cyan("ailang search \"auth\""))
	fmt.Printf("  %s       # Install from registry\n", cyan("ailang install sunholo/auth@0.1.0"))
	fmt.Printf("  %s          # View package AI usage guide\n", cyan("ailang pkg-docs sunholo/auth"))
	fmt.Printf("  %s                      # Publish (preview first)\n", cyan("ailang publish --dry-run"))
	fmt.Println()
	fmt.Println("Debugging & Telemetry:")
	fmt.Printf("  %s                     # Check telemetry config\n", cyan("ailang trace status"))
	fmt.Printf("  %s         # List recent compiler traces\n", cyan("ailang trace list --filter compile --hours 2"))
	fmt.Printf("  %s         # Detect compilation hangs\n", cyan("ailang check --timeout 30s file.ail"))
	fmt.Printf("  %s                 # Debug parser issues\n", cyan("DEBUG_PARSER=1 ailang run test.ail"))
	fmt.Printf("  %s                  # CI mode with strict checking\n", cyan("DEBUG_STRICT=1 make test"))
	fmt.Println()
	fmt.Println("Web & Services:")
	fmt.Printf("  %s                         # Start Observatory dashboard\n", cyan("ailang server"))
	fmt.Printf("  %s          # Serve modules as REST API\n", cyan("ailang serve-api ./api/"))
	fmt.Println()
	fmt.Println("WebAssembly (Browser):")
	fmt.Printf("  %s  # Download WASM\n", cyan("gh release download --repo sunholo-data/ailang -p 'ailang-wasm.tar.gz'"))
	fmt.Printf("  %s        # JS: Load module\n", cyan("repl.loadModule('math', 'let add = \\\\x. \\\\y. x + y')"))
	fmt.Printf("  %s             # JS: Call function\n", cyan("repl.call('math', 'add', 1, 2)"))
	fmt.Println()
	fmt.Println(yellow("Note: For 'run' command, flags must come BEFORE the filename"))
	fmt.Println(yellow("      Example: ailang run --caps IO file.ail  (NOT: ailang run file.ail --caps IO)"))
}
