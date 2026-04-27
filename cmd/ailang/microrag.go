package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/microrag"
	"github.com/sunholo-data/ailang/internal/prompt"
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
	case "user-prompt":
		runMicroragUserPrompt(args)
	case "init":
		runMicroragInit(args)
	case "bootstrap":
		runMicroragBootstrap(args)
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

// runMicroragUserPrompt is the UserPromptSubmit-driven entry point. The
// user's prompt itself is the embedding query — the right-fit hook for
// μRAG per ADR-002. Output envelope mirrors `context`: an Injection or a
// reason for skipping.
func runMicroragUserPrompt(args []string) {
	fs := flag.NewFlagSet("micro-rag user-prompt", flag.ExitOnError)
	prompt := fs.String("prompt", "", "User prompt text (or @path to read from file)")
	namespacesCSV := fs.String("namespaces", "", "Comma-list of brain namespaces to query (default: ailang-syntax,ailang-builtins)")
	configPath := fs.String("config", "", "Path to microrag.yaml (default: ~/.ailang/microrag.yaml)")
	binary := fs.String("ailang-binary", "", "Path to ailang binary for cache search shell-out (default: ailang)")
	_ = fs.Parse(args)

	promptText := readContentArg(*prompt)
	if promptText == "" {
		emitUserPromptResult(&microrag.UserPromptResult{State: "on", Reason: "missing_prompt"})
		return
	}

	if !microrag.EnabledFromEnv() {
		emitUserPromptResult(&microrag.UserPromptResult{State: "disabled", Reason: "env_disabled"})
		return
	}

	cfg, err := microrag.LoadConfig(*configPath)
	if err != nil {
		emitUserPromptResult(&microrag.UserPromptResult{State: "on", Reason: fmt.Sprintf("config_error: %v", err)})
		return
	}

	var namespaces []string
	if csv := strings.TrimSpace(*namespacesCSV); csv != "" {
		for _, n := range strings.Split(csv, ",") {
			n = strings.TrimSpace(n)
			if n != "" {
				namespaces = append(namespaces, n)
			}
		}
	}

	eng := &microrag.Engine{
		Cfg:        cfg,
		Searcher:   &microrag.CLISearcher{Binary: *binary},
		SessionDir: microrag.DefaultSessionDir(),
	}
	res, err := eng.UserPrompt(microrag.UserPromptRequest{Prompt: promptText, Namespaces: namespaces})
	if err != nil {
		emitUserPromptResult(&microrag.UserPromptResult{State: "on", Reason: fmt.Sprintf("engine_error: %v", err)})
		return
	}
	emitUserPromptResult(res)
}

func emitUserPromptResult(r *microrag.UserPromptResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(r)
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

// --- bootstrap subcommand (M-BRAIN-BOOTSTRAP) -----------------------------
//
// Populates the brain DB with `ailang-syntax` (h2 chunks of the active
// embedded prompt) and `ailang-builtins` (one frame per registered builtin)
// using ONLY resources bundled in the binary. Required for fresh installs
// (Claude Code plugin / Gemini CLI extension) where the source repo is
// absent and the shell-based `tools/index_ailang_syntax.sh` cannot run.
//
// Default scope is `--scope user` (writes to ~/.ailang/state/brain.db) so
// the corpus survives across every project on the host. The hooks read
// from both scopes via BrainStore's union semantics.

const (
	bootstrapSourceTag       = "bootstrap-v0.15.1"
	bootstrapSyntaxNamespace = "ailang-syntax"
	bootstrapBuiltinsNS      = "ailang-builtins"
)

// promptChunk is the unit of work for syntax indexing: one ## section.
type promptChunk struct {
	Key     string // stable: "syntax-<version>-<slug>"
	Section string // raw heading text (without "## ")
	Body    string // section body including the heading line
}

// bootstrapResult is emitted in --json mode for install scripts.
type bootstrapResult struct {
	ActiveVersion    string `json:"active_version"`
	Scope            string `json:"scope"`
	SyntaxIndexed    int    `json:"syntax_indexed"`
	BuiltinsIndexed  int    `json:"builtins_indexed"`
	EmbedUsed        bool   `json:"embed_used"`
	Reset            bool   `json:"reset"`
	TookMs           int64  `json:"took_ms"`
	SyntaxNamespace  string `json:"syntax_namespace"`
	BuiltinNamespace string `json:"builtins_namespace"`
}

func runMicroragBootstrap(args []string) {
	fs := flag.NewFlagSet("micro-rag bootstrap", flag.ExitOnError)
	scopeFlag := fs.String("scope", "user", "Brain scope: user (default) | project")
	reset := fs.Bool("reset", false, "Drop ailang-syntax + ailang-builtins namespaces before indexing")
	noEmbed := fs.Bool("no-embed", false, "Skip Ollama embedding (SimHash + FTS only)")
	jsonOut := fs.Bool("json", false, "Emit machine-readable JSON result")
	_ = fs.Parse(args)

	start := time.Now()

	scope := parseScope(*scopeFlag)
	if scope == effects.ScopeBoth {
		// `parseScope` defaults to ScopeBoth — bootstrap requires explicit user/project.
		fmt.Fprintf(os.Stderr, "%s: --scope must be 'user' or 'project' (got %q)\n", red("Error"), *scopeFlag)
		os.Exit(1)
	}

	// Resolve active prompt version. No fallback: if the binary cannot tell
	// us, callers must know to run `ailang prompt --list`.
	activeVersion, err := prompt.GetActiveVersion()
	if err != nil || activeVersion == "" {
		fmt.Fprintf(os.Stderr, "%s: could not resolve active prompt version: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, embedUsed, err := openBootstrapStore(*noEmbed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not open brain store: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	// Optional reset. Only touches the two namespaces we own.
	if *reset {
		if err := bootstrapResetNamespaces(store, scope); err != nil {
			fmt.Fprintf(os.Stderr, "%s: reset failed: %v\n", red("Error"), err)
			os.Exit(1)
		}
	}

	if !*jsonOut {
		fmt.Println("═══ μRAG bootstrap ═══")
		fmt.Printf("  Active prompt:  %s\n", activeVersion)
		fmt.Printf("  Scope:          %s\n", scope)
		fmt.Printf("  Embed:          %v\n", embedUsed)
		fmt.Printf("  Reset:          %v\n\n", *reset)
	}

	syntaxN, err := bootstrapSyntax(store, scope, activeVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: syntax indexing failed: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if !*jsonOut {
		fmt.Printf("[1/2] ailang-syntax    indexed %d chunks\n", syntaxN)
	}

	builtinN, err := bootstrapBuiltins(store, scope, activeVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: builtins indexing failed: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if !*jsonOut {
		fmt.Printf("[2/2] ailang-builtins  indexed %d frames\n\n", builtinN)
	}

	result := bootstrapResult{
		ActiveVersion:    activeVersion,
		Scope:            string(scope),
		SyntaxIndexed:    syntaxN,
		BuiltinsIndexed:  builtinN,
		EmbedUsed:        embedUsed,
		Reset:            *reset,
		TookMs:           time.Since(start).Milliseconds(),
		SyntaxNamespace:  bootstrapSyntaxNamespace,
		BuiltinNamespace: bootstrapBuiltinsNS,
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
		return
	}

	fmt.Println("═══ Done ═══")
	fmt.Printf("  Syntax chunks:    %d\n", result.SyntaxIndexed)
	fmt.Printf("  Builtin frames:   %d\n", result.BuiltinsIndexed)
	fmt.Printf("  Elapsed:          %dms\n", result.TookMs)
}

// openBootstrapStore opens the brain store with optional embedder.
// Returns (store, embedUsed, err). If noEmbed is true, no embedder is even
// attempted. Otherwise we try createEmbedder() and fall back to no-embedder
// (with the warning createEmbedder already prints) if Ollama is down.
func openBootstrapStore(noEmbed bool) (*effects.BrainStore, bool, error) {
	if noEmbed {
		s, err := openBrainStore()
		return s, false, err
	}
	emb := createEmbedder()
	if emb == nil {
		s, err := openBrainStore()
		return s, false, err
	}
	s, err := openBrainStoreWithOpts(effects.WithEmbedder(emb))
	return s, true, err
}

// bootstrapResetNamespaces drops ailang-syntax and ailang-builtins from the
// selected scope. Other namespaces (resolutions, ailang-examples, etc.) are
// left untouched.
func bootstrapResetNamespaces(store *effects.BrainStore, scope effects.BrainScope) error {
	cache := scopeCache(store, scope)
	if cache == nil {
		return fmt.Errorf("scope %s has no cache available", scope)
	}
	for _, ns := range []string{bootstrapSyntaxNamespace, bootstrapBuiltinsNS} {
		if _, err := cache.DeleteNamespace(ns); err != nil {
			return fmt.Errorf("DeleteNamespace(%q): %w", ns, err)
		}
	}
	return nil
}

func scopeCache(store *effects.BrainStore, scope effects.BrainScope) *effects.SQLiteSharedCache {
	switch scope {
	case effects.ScopeUser:
		return store.User
	case effects.ScopeProject:
		return store.Project
	default:
		return nil
	}
}

// bootstrapSyntax loads the active embedded prompt, splits it into ##
// sections, and writes one BrainFrame per section under `ailang-syntax`.
// Returns the number of frames written.
func bootstrapSyntax(store *effects.BrainStore, scope effects.BrainScope, version string) (int, error) {
	content, err := prompt.LoadPrompt("")
	if err != nil {
		return 0, fmt.Errorf("LoadPrompt: %w", err)
	}
	chunks := chunkPromptByH2(content, version)

	count := 0
	for _, c := range chunks {
		body := fmt.Sprintf("[ns:%s] [version:%s] [section:%s]\n%s",
			bootstrapSyntaxNamespace, version, c.Section, c.Body)
		frame := effects.BrainFrame{
			Key:       c.Key,
			Namespace: bootstrapSyntaxNamespace,
			Content:   body,
			Source:    bootstrapSourceTag,
		}
		if err := store.Put(frame, scope); err != nil {
			return count, fmt.Errorf("put %s: %w", c.Key, err)
		}
		count++
	}
	return count, nil
}

// bootstrapBuiltins iterates the compiled-in builtin registry and writes one
// BrainFrame per spec under `ailang-builtins`. Returns the number of frames
// written. Iteration order is sorted by name for deterministic key
// emission across runs.
func bootstrapBuiltins(store *effects.BrainStore, scope effects.BrainScope, version string) (int, error) {
	specs := builtins.AllSpecs()
	names := make([]string, 0, len(specs))
	for n := range specs {
		names = append(names, n)
	}
	sort.Strings(names)

	count := 0
	for _, name := range names {
		spec := specs[name]
		effectLabel := "pure"
		if !spec.IsPure {
			if spec.Effect != "" {
				effectLabel = spec.Effect
			} else {
				effectLabel = "effectful"
			}
		}
		signature := formatBuiltinSignature(spec)
		desc := ""
		if spec.Metadata != nil && spec.Metadata.Description != "" {
			desc = spec.Metadata.Description
		}
		body := fmt.Sprintf("[ns:%s] [version:%s] [module:%s] [effect:%s]\n%s\n%s",
			bootstrapBuiltinsNS, version, spec.Module, effectLabel, signature, desc)
		frame := effects.BrainFrame{
			Key:       "builtin-" + sanitizeKey(spec.Name),
			Namespace: bootstrapBuiltinsNS,
			Content:   body,
			Source:    bootstrapSourceTag,
		}
		if err := store.Put(frame, scope); err != nil {
			return count, fmt.Errorf("put %s: %w", spec.Name, err)
		}
		count++
	}
	return count, nil
}

// chunkPromptByH2 splits a markdown document into chunks at every "## "
// heading. Each chunk's body includes the heading line itself. Bodies
// shorter than 200 bytes are filtered out (no signal). Returns chunks in
// document order with stable keys derived from version + slugified heading.
//
// Mirrors the awk in tools/index_ailang_syntax.sh so retrieval-side filters
// match across the two indexers.
func chunkPromptByH2(content, version string) []promptChunk {
	const minBodyBytes = 200
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<22)

	var (
		out     []promptChunk
		section string
		body    strings.Builder
	)

	flush := func() {
		if section == "" || body.Len() < minBodyBytes {
			return
		}
		out = append(out, promptChunk{
			Key:     fmt.Sprintf("syntax-%s-%s", version, slugify(section)),
			Section: section,
			Body:    body.String(),
		})
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			flush()
			section = strings.TrimPrefix(line, "## ")
			body.Reset()
			body.WriteString(line)
			body.WriteByte('\n')
			continue
		}
		body.WriteString(line)
		body.WriteByte('\n')
	}
	flush()
	return out
}

var slugInvalidRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// slugify lowercases, replaces non-alphanumerics with '-', trims dashes.
// Matches the awk gsub pattern from the shell indexer.
func slugify(s string) string {
	s = slugInvalidRe.ReplaceAllString(s, "-")
	s = strings.ToLower(s)
	s = strings.Trim(s, "-")
	return s
}

var keyInvalidRe = regexp.MustCompile(`[^A-Za-z0-9_]`)

// sanitizeKey replaces non [A-Za-z0-9_] with '_' for use as a brain key
// fragment. Matches `tr -c 'A-Za-z0-9_' '_'` in the shell indexer.
func sanitizeKey(s string) string {
	return keyInvalidRe.ReplaceAllString(s, "_")
}

func printMicroragHelp() {
	fmt.Print(`Usage: ailang micro-rag <subcommand> [options]

Just-in-time knowledge injection engine. Harness-agnostic.

Subcommands:
  context       Resolve injection for a tool-call event (Edit | Write | Read)
  user-prompt   Resolve injection for a UserPromptSubmit event (the right
                hook for embedding-driven retrieval — see ADR-002)
  lint-builtin  Resolve first-use builtin signature nudges (PostToolUse)
  init          Write a default ~/.ailang/microrag.yaml
  bootstrap     Populate ailang-syntax + ailang-builtins from embedded resources
                (run on fresh installs that lack the source repo)

Common flags (context):
  --tool NAME              Tool name (Edit | Write | Read)
  --file PATH              File path being acted on (required)
  --content TEXT           File content (or @path to read from file)
  --config PATH            Override config path (default: ~/.ailang/microrag.yaml)
  --ailang-binary PATH     ailang binary for cache search shell-out

User-prompt flags:
  --prompt TEXT            User prompt text (or @path to read from file)
  --namespaces CSV         Override default namespaces (default:
                           ailang-syntax,ailang-builtins)
  --config PATH            Override config path
  --ailang-binary PATH     ailang binary for cache search shell-out

Bootstrap flags:
  --scope user|project     Brain scope (default: user → ~/.ailang/state/brain.db)
  --reset                  Drop ailang-syntax + ailang-builtins before indexing
  --no-embed               Skip Ollama embedding (SimHash + FTS only)
  --json                   Emit machine-readable JSON result (for install scripts)

Eval toggle (env vars, honored by context/lint subcommands):
  AILANG_MICRORAG_ENABLED  0/1 master switch (default: 1)
  AILANG_MICRORAG_ROUTES   Comma-list KB allowlist (default: all)
  AILANG_MICRORAG_DRYRUN   1 = log to ledger, suppress injection (default: 0)
  AILANG_MICRORAG_SESSION  Session ID (default: pid-<pid>)

Output: JSON {injection: {...} | null, microrag_state: "...", reason: "..."}.

See design_docs/implemented/v0_15_0/m-brain-microrag.md and
design_docs/planned/v0_15_0/m-brain-bootstrap.md for details.
`)
}
