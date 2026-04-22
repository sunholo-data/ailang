package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/microrag"
)

// microragCommand handles the 'micro-rag' subcommand. The engine is harness-
// agnostic — Claude Code hooks, MCP server, and direct CLI all funnel through
// here. See design_docs/planned/v0_15_0/m-brain-microrag.md.
func microragCommand() {
	if len(os.Args) < 3 {
		printMicroragHelp()
		return
	}

	subCmd := os.Args[2]
	args := os.Args[3:]

	switch subCmd {
	case "context":
		runMicroragContext(args)
	case "lint-builtin":
		runMicroragLintBuiltin(args)
	case "init":
		runMicroragInit(args)
	case "--help", "-h", "help":
		printMicroragHelp()
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown micro-rag subcommand %q\n", red("Error"), subCmd)
		printMicroragHelp()
		os.Exit(1)
	}
}

func runMicroragContext(args []string) {
	fs := flag.NewFlagSet("micro-rag context", flag.ExitOnError)
	tool := fs.String("tool", "", "Tool name (Edit | Write | Read)")
	file := fs.String("file", "", "File path being acted on")
	content := fs.String("content", "", "File content (Edit/Write); pass via @path to read from file")
	configPath := fs.String("config", "", "Path to microrag.yaml (default: ~/.ailang/microrag.yaml)")
	binary := fs.String("ailang-binary", "", "Path to ailang binary for cache search shell-out (default: ailang)")
	_ = fs.Parse(args)

	if *file == "" {
		fmt.Fprintln(os.Stderr, `{"microrag_state":"on","reason":"missing_file"}`)
		os.Exit(0)
	}

	// Fast-path: env-disabled. Don't even load config.
	if !microrag.EnabledFromEnv() {
		emitContextResult(&microrag.ContextResult{State: "disabled", Reason: "env_disabled"})
		return
	}

	cfg, err := microrag.LoadConfig(*configPath)
	if err != nil {
		// Treat config errors as graceful no-op so hooks never fail loud.
		emitContextResult(&microrag.ContextResult{State: "on", Reason: fmt.Sprintf("config_error: %v", err)})
		return
	}

	contentText := readContentArg(*content)
	eng := &microrag.Engine{
		Cfg:        cfg,
		Searcher:   &microrag.CLISearcher{Binary: *binary},
		SessionDir: microrag.DefaultSessionDir(),
	}
	res, err := eng.Context(microrag.Request{ToolName: *tool, FilePath: *file, Content: contentText})
	if err != nil {
		emitContextResult(&microrag.ContextResult{State: "on", Reason: fmt.Sprintf("engine_error: %v", err)})
		return
	}
	emitContextResult(res)
}

func emitContextResult(r *microrag.ContextResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(r)
}

// readContentArg resolves --content. If the value starts with '@', the rest
// is treated as a file path to read. Otherwise the value is used verbatim.
// Errors degrade to empty content so the engine still routes on file path.
func readContentArg(v string) string {
	if v == "" {
		return ""
	}
	if v[0] != '@' {
		return v
	}
	data, err := os.ReadFile(v[1:])
	if err != nil {
		return ""
	}
	return string(data)
}

func runMicroragLintBuiltin(args []string) {
	fs := flag.NewFlagSet("micro-rag lint-builtin", flag.ExitOnError)
	code := fs.String("code", "", "Code snippet to scan for first-use builtins (or @path)")
	file := fs.String("file", "", "Optional file path tag for telemetry")
	binary := fs.String("ailang-binary", "", "Path to ailang binary for builtins lookup (default: ailang)")
	maxNudges := fs.Int("max-nudges", 0, "Max nudges per call (default 2)")
	maxTokens := fs.Int("max-tokens", 0, "Per-nudge token cap (default 80)")
	_ = fs.Parse(args)

	codeText := readContentArg(*code)
	if codeText == "" {
		emitLintResult(&microrag.LintResult{State: "on", Reason: "empty_code"})
		return
	}
	linter := &microrag.Linter{
		Resolver:   &microrag.CLIBuiltinResolver{Binary: *binary},
		SessionDir: microrag.DefaultSessionDir(),
		MaxNudges:  *maxNudges,
		MaxTokens:  *maxTokens,
	}
	res, err := linter.Lint(microrag.LintRequest{FilePath: *file, Code: codeText})
	if err != nil {
		emitLintResult(&microrag.LintResult{State: "on", Reason: fmt.Sprintf("lint_error: %v", err)})
		return
	}
	emitLintResult(res)
}

func emitLintResult(r *microrag.LintResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(r)
}

func runMicroragInit(_ []string) {
	cfg := defaultMicroragYAML()
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	target := home + "/.ailang/microrag.yaml"
	if _, err := os.Stat(target); err == nil {
		fmt.Printf("Already exists: %s (not overwriting)\n", target)
		return
	}
	if err := os.MkdirAll(home+"/.ailang", 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if err := os.WriteFile(target, []byte(cfg), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("Wrote default config: %s\n", target)
}

func defaultMicroragYAML() string {
	return `# micro-rag (μRAG) glob router config
# See: design_docs/planned/v0_15_0/m-brain-microrag.md
enabled: true

routes:
  - glob: "**/*.ail"
    kb: ailang-syntax
    max_tokens_per_injection: 150
    relevance_floor: 0.30
  - glob: "**/CHANGELOG.md"
    kb: ailang-breaking-changes
    max_tokens_per_injection: 150
    relevance_floor: 0.40
  - glob: "**/CLAUDE.md"
    kb: skip

dedup:
  windows:
    ailang-breaking-changes: 15000
    ailang-syntax: 30000
    ailang-builtins: 80000
    project-resolutions: 40000
    default: 30000
  relevance_bypass:
    ailang-breaking-changes: 0.60
    ailang-syntax: 0.70
    ailang-builtins: 0.80
    default: 0.70
  wall_clock_max: 240

session_budget: 5000
marker_style: unicode  # unicode | ascii
`
}

func printMicroragHelp() {
	fmt.Print(`Usage: ailang micro-rag <subcommand> [options]

Just-in-time knowledge injection engine. Harness-agnostic.

Subcommands:
  context       Resolve injection for a tool-call event (Edit | Write | Read)
  lint-builtin  Resolve first-use builtin signature nudges (PostToolUse)
  init          Write a default ~/.ailang/microrag.yaml

Common flags (context):
  --tool NAME              Tool name (Edit | Write | Read)
  --file PATH              File path being acted on (required)
  --content TEXT           File content (or @path to read from file)
  --config PATH            Override config path (default: ~/.ailang/microrag.yaml)
  --ailang-binary PATH     ailang binary for cache search shell-out

Eval toggle (env vars, honored by all subcommands):
  AILANG_MICRORAG_ENABLED  0/1 master switch (default: 1)
  AILANG_MICRORAG_ROUTES   Comma-list KB allowlist (default: all)
  AILANG_MICRORAG_DRYRUN   1 = log to ledger, suppress injection (default: 0)
  AILANG_MICRORAG_SESSION  Session ID (default: pid-<pid>)

Output: JSON {injection: {...} | null, microrag_state: "...", reason: "..."}.

See design_docs/planned/v0_15_0/m-brain-microrag.md for details.
`)
}
