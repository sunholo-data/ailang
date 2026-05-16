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
3. Work on CURRENT branch — don't force a branch switch
4. NEVER assume it's safe to discard or stash work

### 0.1. VERIFY GITHUB ACCOUNT BEFORE RELEASES/TAGS

The developer uses multiple GitHub accounts. Before ANY release or git push:

```bash
gh auth status                              # Check active account
# This repo needs: sunholo-voight-kampff (Claude Code agent account)
# MarkEdmondson1234 = human developer account
# rw-markedmondson = WRONG (Rockwool project)
gh auth switch --user sunholo-voight-kampff  # Switch if needed
```

### 1. ALWAYS USE EXISTING TOOLS FIRST

Before writing ANY new script or code:
1. Check `make help` for existing targets
2. Check `tools/` directory for existing scripts
3. Search codebase: `grep -r "function_name" internal/`

**The `ailang` CLI exists to make YOUR life easier.** Use `ailang chains`, `ailang messages`, `ailang eval-*`, and `ailang dashboard` instead of raw SQLite queries or ad-hoc scripts.

### 2. NO SILENT FALLBACKS - FAIL LOUDLY

If the fallback value affects data integrity, business logic, or user decisions → **NO FALLBACK**. Return zero, null, or error instead.

**Apply to:** Pricing/costs, model configs, required env vars, data validation.
**Fallbacks OK for:** UI defaults, optional features, caching.

### 3. SYSTEMIC FIXES - AUDIT BEFORE PATCHING

Before fixing a bug, ALWAYS ask: "Is this part of a larger pattern?" Search for similar code paths. Design ONE unified fix instead of patching case-by-case.

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

### AI Provider vs Executor Architecture

| Package | Purpose | Use For |
|---------|---------|---------|
| `internal/ai/` | Text generation via HTTP APIs | Research, docs, Q&A |
| `internal/executor/` | Agentic coding with file editing | Bug fixes, features, refactoring |

---

## Available Skills

- **use-ailang** — Write correct AILANG code
- **skill-builder** — Create new skills (meta-skill)
- **release-manager** / **post-release** — Release workflow
- **sprint-planner** / **sprint-executor** / **sprint-evaluator** — Sprint workflow (plan, execute, evaluate)
- **collaboration-hub** — Collaboration Hub UI (React)
- **codebase-organizer** — Refactor large files
- **design-spec-auditor** — Verify code matches specs
- **github-issue-triage** — Triage GitHub issues
- **test-coverage-guardian** — Analyze test coverage
- **perf-reviewer** — Performance review and benchmarks
- **trace-debugger** — Debug via OTEL traces
- **builtin-developer** — Add builtin functions
- **parser-developer** — Parser development
- **eval-analyzer** / **benchmark-manager** — Eval tools
- **coordinator-helper** — Manage coordinator tasks

**Docs**: [.claude/skills/README.md](.claude/skills/README.md)

---

## Claude Code Plugins (LSP)

This repo ships a local Claude Code marketplace at [.claude-plugin/marketplace.json](.claude-plugin/marketplace.json) with:

- **ailang-go-lsp** — runs `gopls` so Claude gets real-time Go diagnostics, go-to-definition, find-references, and hover types instead of grepping the codebase. Requires `gopls` on PATH (`go install golang.org/x/tools/gopls@latest`).
- **ailang-lsp** — runs `ailang lsp --stdio` so Claude gets the same capabilities for `.ail` files (diagnostics, hover types with effect rows, go-to-def, references, document symbols). Requires the `ailang` binary on PATH (`make install` from this repo).

One-time install (per developer):

```
/plugin marketplace add /Users/mark/dev/sunholo/ailang
/plugin install ailang-go-lsp@ailang-tools
/plugin install ailang-lsp@ailang-tools
```

Verify with `/plugin` (both should show under Installed, no entries under Errors). User guide: [docs/docs/guides/lsp.md](docs/docs/guides/lsp.md).

---

## Conditional Rules (`.claude/rules/`)

Domain-specific rules load automatically when working with matching files:

| Rule File | Loads When Touching |
|-----------|-------------------|
| `parser.md` | `internal/parser/`, `internal/lexer/`, `internal/ast/` |
| `type-system.md` | `internal/types/`, `internal/elaborate/`, `internal/iface/`, `internal/pipeline/` |
| `coordinator.md` | `internal/coordinator/`, `internal/executor/`, `internal/server/`, `ui/` |
| `eval.md` | `internal/eval_harness/`, `benchmarks/`, `eval_results/` |
| `ailang-syntax.md` | `examples/*.ail`, `stdlib/`, `prompts/` |
| `coding-standards.md` | Always loaded |
| `dev-workflow.md` | Always loaded |

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
