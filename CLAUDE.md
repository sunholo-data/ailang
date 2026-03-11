# Claude Instructions for AILANG Development

## Session Start Routine

**At the start of EVERY session, check for agent messages.**

1. **SessionStart hook runs automatically** — injects unread messages into system reminders
2. **If no messages in reminders**, manually check: `ailang messages list --unread`
3. **When messages exist**: Summarize to user, ask what to do
4. **After handling**: `ailang messages ack --all`
5. **If task fails**: `ailang messages unack MSG_ID` (moves back for retry)

**Essential message commands:**
```bash
ailang messages list --unread              # Check for new messages
ailang messages read MSG_ID                # Read full message
ailang messages ack --all                  # Mark all as read
ailang messages send INBOX "msg" --title "Title" --from "agent-name"
```

**Full command reference:** See [docs/docs/guides/agent-messaging.md](docs/docs/guides/agent-messaging.md)

---

## Critical Principles

### 0. NEVER DESTROY LOCAL WORK WITH GIT OPERATIONS

**NEVER run these commands — they destroy uncommitted work:**
- `git checkout <branch>` when there are uncommitted changes
- `git pull` on a branch with local commits (use `git status` first!)
- `git reset --hard`, `git clean -fd`
- `git stash` followed by branch switching that causes fast-forward

**CORRECT approach:**
1. ALWAYS `git status` first
2. If uncommitted changes exist, **ASK THE USER** how to handle them
3. If fixing docs/CI, work on CURRENT branch — don't force a branch switch
4. NEVER assume it's safe to discard or stash work

**Safe alternatives:**
- Make fixes on current branch, push, let user handle merging
- Ask user before ANY branch switching
- Create new branches instead of switching to existing ones

### 0.1. VERIFY GITHUB ACCOUNT BEFORE RELEASES/TAGS

The developer uses multiple GitHub accounts. Before ANY release or git push:

```bash
gh auth status                              # Check active account
# This repo needs: sunholo-voight-kampff (Claude Code agent account)
# MarkEdmondson1234 = human developer account
# rw-markedmondson = WRONG (Rockwool project)
gh auth switch --user sunholo-voight-kampff  # Switch if needed
```

**Checklist:** Check account → Switch if needed → Push tag → Create release

### 1. ALWAYS USE EXISTING TOOLS FIRST

Before writing ANY new script or code:
1. Check `make help` for existing targets
2. Check `tools/` directory for existing scripts
3. Check this CLAUDE.md for documented workflows
4. Search codebase: `grep -r "function_name" internal/`

**The `ailang` CLI exists to make YOUR life easier.** Use `ailang chains`, `ailang messages`, `ailang eval-*`, and `ailang dashboard` instead of raw SQLite queries or ad-hoc scripts.

**If the CLI is insufficient:** Propose improvements (new flags, subcommands) rather than working around limitations with one-off scripts.

### 2. NO SILENT FALLBACKS - FAIL LOUDLY

Silent fallbacks hide bugs and produce wrong data that users trust.

**Rule:** If the fallback value affects data integrity, business logic, or user decisions → **NO FALLBACK**. Return zero, null, or error instead.

**Apply to:** Pricing/costs, model configs, required env vars, data validation, config loading.
**Fallbacks OK for:** UI defaults, optional features, caching, performance optimizations.

### 3. CORETYPEINFO INVARIANT - COMPLETE TYPE COVERAGE

Every Core node must have an entry in CoreTypeInfo before lowering. Validation enforces this in all paths (file pipeline, module pipeline, REPL).

```go
if err := ValidateCoreTypeInfo(coreProg, typeChecker.CoreTI); err != nil {
    return result, fmt.Errorf("CoreTypeInfo validation failed: %w", err)
}
```

Validated nodes may carry polymorphic types (accepted as long as a type exists). Validator: `internal/pipeline/validate_coretypeinfo.go`.

### 4. SAFE TYPE TRAVERSAL - CYCLE PROTECTION

Every function of shape `func(types.Type) T` MUST document cycle-safety. Either use `traverse.Walk` or add a `visited` parameter.

Package: `internal/types/traverse/` — provides `Walk()`, `CollectFreeVars()`, `HasCycles()`.

### 5. MONOMORPHIZATION - CALL-SITE SPECIALIZATION

Enabled by default (v0.4.0). Polymorphic lambdas specialized at call sites.

```bash
ailang run --entry main --caps IO module.ail  # Normal (mono enabled)
ailang run --debug-compile module.ail          # Show specialization stats
ailang run --no-mono module.ail                # Disable (escape hatch)
```

Resource limits: 16 per-function, 512 per-module. See [design_docs/implemented/v0_4_0/monomorphization.md](design_docs/implemented/v0_4_0/monomorphization.md).

### 6. SYSTEMIC FIXES - AUDIT BEFORE PATCHING

Before fixing a bug, ALWAYS ask: "Is this part of a larger pattern?"

**Anti-pattern:** Fix case B → Fix case C → Fix case D → forever patching.
**Correct:** Before fixing B, search for similar code paths. Check if A, C, D have same gap. Design ONE unified fix.

**Warning signs:** Multiple maps tracking similar things, growing switch/case lists, bug fixes adding `|| specialCase`.

---

## Available Skills

- **use-ailang** — Write correct AILANG code with syntax reference and validation
- **skill-builder** — Create new skills with automated scaffolding (meta-skill)
- **release-manager** — Create releases with pre-flight checks and verification
- **post-release** — Run eval baselines and update website dashboard
- **sprint-planner** — Analyze design docs, create realistic sprint plans
- **sprint-executor** — Execute sprints with TDD, linting, and progress tracking
- **collaboration-hub** — Develop/modify the Collaboration Hub UI (React)
- **codebase-organizer** — Monitor and refactor large files into AI-friendly modules
- **design-spec-auditor** — Verify code matches design specifications
- **github-issue-triage** — Triage GitHub issues against design docs
- **test-coverage-guardian** — Analyze test coverage, identify gaps
- **perf-reviewer** — Performance review and cross-language benchmarks
- **trace-debugger** — Debug performance using OTEL traces
- **builtin-developer** — Add builtin functions to the language
- **parser-developer** — Parser development with conventions and patterns
- **eval-analyzer** — Identify language gaps from eval failures
- **benchmark-manager** — Create and manage eval benchmarks
- **coordinator-helper** — Manage coordinator tasks and approvals

**Complete docs**: [.claude/skills/README.md](.claude/skills/README.md) | **Create new**: Use `skill-builder` skill

Skills are invoked automatically when appropriate. **Skills** = focused workflows, **Agents** = multi-stage orchestration (e.g., `dev-cycle`), **Commands** = deprecated.

---

## Project Overview

**AILANG is a deterministic language designed for autonomous AI code synthesis and reasoning.**

Priorities: Machine decidability, semantic transparency, compositional determinism. File extension: `.ail`.

### Key Design Principles
1. **Explicit Effects** — All side effects declared in function signatures
2. **Everything is an Expression** — No statements, only expressions
3. **Type Safety** — Hindley-Milner inference + row polymorphism
4. **Deterministic** — All non-determinism must be explicit
5. **AI-Friendly** — Structured execution traces for training

### Project Structure
```
ailang/
├── cmd/ailang/         # CLI entry point
├── internal/
│   ├── ast/            # Surface AST
│   ├── lexer/          # Tokenizer
│   ├── parser/         # Parser
│   ├── core/           # Core AST (ANF)
│   ├── elaborate/      # Surface -> Core elaboration
│   ├── types/          # Type system
│   ├── typeclass/      # Type classes (stub)
│   ├── link/           # Dictionary linking
│   ├── pipeline/       # Full compilation pipeline
│   ├── eval/           # Evaluator (Core + module support)
│   ├── repl/           # Interactive REPL
│   ├── runtime/        # Module execution runtime
│   ├── effects/        # Effect system runtime
│   ├── loader/         # Module loader
│   ├── errors/         # Error reporting
│   ├── schema/         # JSON schemas
│   ├── ai/             # Unified AI providers (text generation via API)
│   ├── executor/       # CLI executors (agentic coding via Claude/Gemini CLI)
│   ├── eval_harness/   # AI evaluation framework
│   ├── eval_analysis/  # Go eval tools
│   ├── builtins/       # Builtin definitions
│   ├── coordinator/    # Task delegation daemon
│   ├── server/         # Collaboration Hub backend
│   └── ...             # dtree, iface, manifest, module, typedast, planning, etc.
├── stdlib/             # Standard library (27 modules)
├── tools/              # Development tools
├── benchmarks/         # AI code generation benchmarks
├── examples/           # Example .ail programs (66 files, 48 passing)
├── tests/              # Test suite
├── ui/                 # Collaboration Hub frontend (React + Vite)
└── docs/               # Documentation website
```

### AI Provider vs Executor Architecture

**Two different ways to use AI in AILANG:**

| Package | Purpose | Interface | Use For |
|---------|---------|-----------|---------|
| `internal/ai/` | Text generation via HTTP APIs | `ai.Provider.Generate()` | Research, docs, Q&A |
| `internal/executor/` | Agentic coding with file editing | `executor.Executor.Execute()` | Bug fixes, features, refactoring |

**Key difference:** `executor/` runs CLI tools that edit files. `ai/` just generates text.

---

## Development Workflow

### Building and Testing
```bash
make build          # Build the interpreter
make install        # Install ailang to system
make quick-install  # Fast reinstall after changes (recommended)
make test           # Run all tests
make run FILE=...   # Run an AILANG file
make repl           # Start interactive REPL
```

**Important**: `ailang` in PATH points to `/Users/mark/go/bin/ailang` (system). Always reinstall after building.

### Collaboration Hub Server

Use the `collaboration-hub` skill for development. Quick start:
```bash
make services-start     # Start server + coordinator
make services-status    # Check both services
make services-stop      # Stop both services
```
**Full guide**: [docs/docs/guides/collaboration-hub.md](docs/docs/guides/collaboration-hub.md)

### Coordinator Daemon

The coordinator executes tasks autonomously using AI agents in isolated git worktrees.

**Quick start:**
```bash
make services-start                          # Start server + coordinator
ailang coordinator status                    # Check if running
ailang messages send coordinator "Fix bug"   # Delegate a task
ailang coordinator pending                   # Review pending approvals
```

**Agent workflow:** GitHub Issue → design-doc-creator → [Approval] → sprint-planner → [Approval] → sprint-executor → [Approval] → Merged

**Config**: `~/.ailang/config.yaml` | **Cloud mode**: Pub/Sub + Cloud Run (v0.9.0+)
**Full guide**: [docs/docs/guides/coordinator.md](docs/docs/guides/coordinator.md)

### Chain Execution Monitoring

`ailang chains` is the canonical CLI for examining executions. Works offline (direct SQLite).

```bash
ailang chains list                       # List all chains
ailang chains list --agent X --since 24h # Filter by agent/time
ailang chains view <chain-id> --spans    # Full execution with sessions + tools
ailang chains tree <chain-id> --detailed # ASCII tree with tool timeline
ailang chains stats --by-agent           # Cost/token breakdown
ailang chains diagnose <chain-id>        # Quick health report
```

### Auditing Agent Work

After a coordinator task completes:
1. `ailang chains view <chain-id> --spans` — execution flow
2. `ailang coordinator logs <task-id> --limit 500` — conversation text
3. `ailang coordinator diff <task-id>` — git changes

**Key checks:** Model used (Haiku too weak for compiler), turn count/cost, code changes in `internal/`, runtime vs compile testing.

### Adding Builtin Functions

Use the `builtin-developer` skill. Validation: `ailang doctor builtins`.

### M-EVAL: AI Evaluation

Use the `eval-analyzer` skill or `ailang eval-*` commands. Two modes: Standard (0-shot API) and Agent (agentic CLI).

**CRITICAL:** `ailang eval-suite` OVERWRITES the output directory. Run all models in ONE command:
```bash
ailang eval-suite --models gpt5,claude-sonnet-4-5,gemini-2-5-pro  # Correct
```

**Dashboard updates preserve history automatically:**
```bash
ailang eval-report eval_results/baselines/v0.3.10 v0.3.10 --format=json  # Correct
# DON'T redirect stdout — bypasses history preservation
```

**Full guide**: [docs/docs/guides/evaluation/](docs/docs/guides/evaluation/)

### Code Quality
```bash
make test-coverage-badge  # Quick coverage check
make lint                 # Run golangci-lint
make fmt                  # Format all Go code
make ci                   # Full CI verification locally
make verify-examples      # Check example files
make check-file-sizes     # Fails if >800 lines
```

### Debug Flags

| Flag | Purpose |
|------|---------|
| `DEBUG_STRICT=1` | Fail loudly on unhandled cases |
| `DEBUG_MONO_VERBOSE=1` | Monomorphization tracing |
| `DEBUG_OPERATOR_LOWERING=1` | Operator resolution |
| `DEBUG_PARSER=1` | Token position tracing |
| `DEBUG_CODEGEN=1` | Record type fallback warnings |
| `DEBUG_APPROVAL_WATCHER=1` | ApprovalWatcher polling |
| `--timeout 30s` | Compilation timeout with stack dump (CLI flag) |
| `--debug-compile` | Phase timing breakdown (CLI flag) |

**Full guide**: [docs/docs/guides/debugging.md](docs/docs/guides/debugging.md)

### Telemetry & Traces

Use the `trace-debugger` skill. Quick: `ailang trace status`, `ailang trace list --hours 1`.
**Full guide**: [docs/docs/guides/telemetry.md](docs/docs/guides/telemetry.md)

### Writing AILANG Code

| Command | Purpose | Size |
|---------|---------|------|
| `ailang prompt` | Full syntax reference (0-shot generation) | ~1600 lines |
| `ailang devtools-prompt` | Toolchain/CLI reference | ~600 lines |
| `ailang agent-prompt` | Minimal agent coding guide | ~180 lines |

**Critical syntax notes:**
- **Import Aliasing (v0.4.8):** `import std/list as List (map, filter)`
- **Relaxed Module Matching:** `ailang run --relax-modules` or `AILANG_RELAX_MODULES=1`
- **Nullary functions (v0.4.6+):** Call with empty parens: `getArgs()`, `readLine()`
- **Type parameters:** Use `[T]` NOT `(T)`
- **Match in blocks:** Known parser bug — extract to helper function
- **Flags before filename:** `ailang run --caps IO,FS --entry main module.ail`

**See also**: [docs/LIMITATIONS.md](docs/LIMITATIONS.md) | [examples/](examples/)

### Documentation Updates

Every change requires:
1. **CHANGELOG.md** — Semantic versioning, grouped by category
2. **README.md** — Update status/capabilities if public-facing
3. **Design docs** — Before: `design_docs/planned/`, After: `design_docs/implemented/vX_Y/`
4. **Example files** — Every language feature needs `examples/feature_name.ail`
5. **Website examples** — Import from `examples/` using raw-loader, never embed code inline

### Adding a New Language Feature
1. Token definitions: `internal/lexer/token.go`
2. Lexer: `internal/lexer/lexer.go`
3. AST nodes: `internal/ast/ast.go`
4. Parser: `internal/parser/parser.go`
5. Type rules: `internal/types/`
6. Evaluation: `internal/eval/`
7. Tests, examples, CHANGELOG, README (all REQUIRED)

---

## Release Workflow

**For releases**: Use the `release-manager` skill
**After release**: Use the `post-release` skill

## Code Organization

File size targets: 200-500 (sweet spot), 500-800 (acceptable), 1200+ (MUST split).
Use `codebase-organizer` skill for refactoring. `make check-file-sizes` enforces in CI.

---

## Critical Warnings

### Testing Policy
ALWAYS remove out-of-date tests. No backward compatibility. When architecture changes, delete old tests and write new ones.

### Linting & "Unused" Code Warnings

**NEVER delete functions just because linter says "unused".**

The Import System Disaster (Sept 2025): Linter said functions were unused because calls were renamed/commented out. Functions were blindly deleted. Result: working import system completely broken.

**Rules:**
1. Understand WHY they're unused — check git history, search for commented-out references
2. If renaming calls, rename definitions too
3. Test between each change (`make test` after rename, after commenting, after deleting)
4. Special care for parser/module/import code — run `make test-imports` and `make verify-examples`

### ast.Type Switch Exhaustiveness

**ALL 8 variants must be handled.** Silent `default` cases corrupt imported polymorphic types.

| Variant | Example |
|---------|---------|
| `*ast.SimpleType` | `int`, `string`, `Result` |
| `*ast.TypeVar` | `a`, `e` (type parameters) |
| `*ast.FuncType` | `(int) -> string` |
| `*ast.ListType` | `[int]` |
| `*ast.ArrayType` | `Array[int]` |
| `*ast.TupleType` | `(int, string)` |
| `*ast.TypeApp` | `Result[a, e]` |
| `*ast.RecordType` | `{name: string}` |

**Rules:** `default:` in ast.Type switches MUST `panic()`, never return fake data. When adding a new variant, grep for ALL type switches: `grep -rn "case \*ast\." internal/`

### Lexer/Parser: NEWLINE Tokens Don't Exist

The lexer NEVER generates NEWLINE tokens — `skipWhitespace()` consumes `\n`. Never check for `lexer.NEWLINE` in parser code. Multi-line syntax "just works" because whitespace is already skipped.

### Claude Code CLI: TRACEPARENT Not Propagated

Claude Code does NOT propagate TRACEPARENT to subprocess environments. Child spans are in DIFFERENT traces. This is a known, accepted limitation.

**DO NOT:** Try to inject TRACEPARENT, attempt runtime fixes, or re-investigate.
**Workaround:** Use `task_id`/`parent_task_id` attributes for cross-trace linking. Timestamp correlation works for script-based agents.

---

## Parser Development

Use the `parser-developer` skill.

**Critical conventions:**
1. Parser leaves cursor AT last token (not after)
2. No NEWLINE tokens — lexer skips `\n` as whitespace
3. Use `DEBUG_PARSER=1` for token position tracing
4. Use `make doc PKG=<package>` for API discovery

**Common gotchas:**
- IntLit is `int64`, not `int` (will panic!)
- Never check for `lexer.NEWLINE`
- Print errors BEFORE `t.Fatalf` in tests
- `string(rune(i))` produces unprintable chars (use `fmt.Sprintf`)

**Constructors:**
- `parser.New(lexer)` — Takes lexer instance
- `elaborate.NewElaborator()` — No arguments
- `types.NewTypeChecker(core, imports)` — Takes Core prog + imports
- `link.NewLinker()` — No arguments

---

## Database Architecture

Three SQLite databases: `observatory.db` (spans/traces), `coordinator.db` (tasks/approvals), `collaboration.db` (messages).

**Full reference with ID formats, flow diagrams, and querying tips:** See [docs/docs/guides/database-architecture.md](docs/docs/guides/database-architecture.md)

---

## Reference Documentation

- **AILANG Syntax**: `ailang prompt` or [prompts/](prompts/)
- **Limitations**: [docs/LIMITATIONS.md](docs/LIMITATIONS.md)
- **Development Workflow**: [docs/docs/guides/development-workflow.md](docs/docs/guides/development-workflow.md)
- **Coordinator**: [docs/docs/guides/coordinator.md](docs/docs/guides/coordinator.md)
- **Messaging**: [docs/docs/guides/agent-messaging.md](docs/docs/guides/agent-messaging.md)
- **Evaluation**: [docs/docs/guides/evaluation/](docs/docs/guides/evaluation/)
- **Telemetry**: [docs/docs/guides/telemetry.md](docs/docs/guides/telemetry.md)
- **Debugging**: [docs/docs/guides/debugging.md](docs/docs/guides/debugging.md)
- **Collaboration Hub**: [docs/docs/guides/collaboration-hub.md](docs/docs/guides/collaboration-hub.md)
- **Database Architecture**: [docs/docs/guides/database-architecture.md](docs/docs/guides/database-architecture.md)
- **Design Docs**: [design_docs/](design_docs/)
- **Examples**: [examples/](examples/)

---

**Remember**: This is a living document. Keep it focused on **actionable instructions** for Claude, not reference material that belongs in docs/.
