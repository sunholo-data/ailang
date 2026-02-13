# M-DEV-TOOLS-AI-DOC: AI-Consumable Dev Tools Documentation

**Status**: Partially Implemented (Milestones 1-2 complete, Milestone 3 in progress)
**Priority**: Medium
**Estimated LOC**: ~400 (documentation) + ~100 (tooling)
**Target**: v0.8.0

### Implementation Progress (2026-02-11)
- [x] Milestone 1: Dev tools prompt content (`prompts/devtools/v0.8.0.md` - 283 lines, 10 workflow categories)
- [x] Milestone 2: CLI command (`ailang devtools-prompt` with `--version`, `--list`, `--info`, `--help`)
- [x] Milestone 2: Backend loader (`internal/devtoolsprompt/loader.go`)
- [x] Milestone 2: Registered in `main.go` and `help.go`
- [x] Milestone 3 (partial): Skills updated with devtools-prompt references
- [ ] Milestone 3: Website page explaining both prompts
- [ ] Milestone 3: CHANGELOG.md entry

## Problem Statement

AILANG has a rich set of development tools beyond the language itself:
- **Trace export & replay** (`--emit-trace`, `ailang replay`)
- **Execution chains** (`ailang chains list/view/tree/stats/diagnose`)
- **Coordinator daemon** (`ailang coordinator start/status/pending`)
- **Agent messaging** (`ailang messages send/read/ack`)
- **Observatory & telemetry** (`ailang trace status/list/view`)
- **Eval harness** (`ailang eval-suite/eval-compare/eval-report`)
- **Debug tools** (`ailang check --timeout`, `ailang debug ast`)

The current AI teaching prompt (`ailang prompt`) covers **language syntax only** - how to write `.ail` files. There is no equivalent document that teaches AI agents (or human developers) how to use the AILANG toolchain for:
- Debugging programs using traces
- Verifying determinism via replay
- Monitoring agent execution chains
- Delegating tasks to the coordinator
- Analyzing eval results

This means AI agents working with AILANG (via Claude Code, Gemini CLI, etc.) don't know these tools exist and can't leverage them for self-diagnosis, testing, or workflow automation.

## Scope

### What This Document Covers
A new **AI-consumable dev tools reference** that complements the existing teaching prompt. Two artifacts:

1. **`ailang devtools-prompt`** — CLI command that outputs the dev tools reference (same pattern as `ailang prompt` for syntax)
2. **Dev tools reference document** — versioned markdown covering all non-syntax tooling

### What This Does NOT Cover
- Language syntax (already in `ailang prompt`)
- Internal architecture (for CLAUDE.md, not for external AIs)
- Human-oriented tutorials (for docs website)

## Current State

### Teaching Prompt (`ailang prompt`)
- 41 versions tracked in `prompts/versions.json`
- Covers: module structure, types, effects, pattern matching, stdlib, CLI exploration basics
- CLI section mentions: `ailang docs`, `ailang builtins`, `ailang check`, `ailang run`
- **Missing**: All dev tools listed in Problem Statement

### CLAUDE.md
- 900+ lines covering ALL tools in detail
- But this is project-internal — only visible to Claude Code sessions working on AILANG itself
- Not consumable by external AI agents or other projects using AILANG

### Website Docs
- Guides for coordinator, telemetry, evaluation
- But AI agents don't browse websites — they need inline prompt context

## Proposed Solution

### New Command: `ailang devtools-prompt`

```bash
# Display current dev tools reference
ailang devtools-prompt

# Display specific version
ailang devtools-prompt --version v0.8.0

# List available versions
ailang devtools-prompt --list

# Save to file for injection into AI context
ailang devtools-prompt > devtools_context.md
```

Implementation mirrors `cmd/ailang/prompt.go` — embedded versioned markdown, no external dependencies.

### Document Structure

The dev tools reference is organized by **workflow** (what the AI is trying to accomplish), not by command (alphabetical listing). This makes it actionable.

```markdown
# AILANG Dev Tools Reference v0.8.0

## 1. Running & Testing Programs
ailang run --caps IO,FS --entry main module.ail
ailang check --timeout 30s module.ail
ailang test tests/
ailang watch module.ail

## 2. Debugging with Traces
ailang run --emit-trace jsonl --caps IO --entry main module.ail
ailang run --emit-trace otel --caps IO --entry main module.ail
ailang replay baseline.jsonl
ailang replay --json baseline.jsonl

## 3. Verifying Determinism
ailang replay recorded.jsonl  # Re-run and compare
ailang replay --file alt.ail recorded.jsonl  # Override source
ailang replay --json recorded.jsonl | jq .match

## 4. Monitoring Agent Execution
ailang chains list --since 24h
ailang chains view <chain-id> --spans
ailang chains tree <chain-id> --detailed
ailang chains stats --by-agent
ailang chains diagnose <chain-id>

## 5. Delegating Tasks
ailang messages send coordinator "Fix bug in parser.go" --title "Bug" --from "user"
ailang coordinator status
ailang coordinator pending  # Review completed work

## 6. Evaluating AI Code Generation
ailang eval-suite --models gpt5-mini,claude-haiku-4-5
ailang eval-compare baseline_v1/ baseline_v2/
ailang eval-report results/ v0.8.0 --format=json

## 7. Exploring the Stdlib
ailang docs --list
ailang docs std/string
ailang builtins list --by-module
ailang doctor builtins

## 8. Telemetry & Observability
ailang trace status
ailang trace list --hours 1
ailang chains health
```

### Key Design Decisions

**1. Workflow-organized, not command-organized**
AI agents think in goals ("I need to verify this program is deterministic"), not in command names. Each section answers a "how do I..." question.

**2. Show command + expected output pattern**
For each command, show what the agent should expect to see. This reduces trial-and-error.

**3. Version-locked like the teaching prompt**
Each AILANG release gets a matching dev tools prompt version. This prevents AI agents from trying commands that don't exist yet.

**4. Composable with teaching prompt**
Projects using AILANG can inject BOTH prompts:
```bash
cat <(ailang prompt) <(ailang devtools-prompt) > full_context.md
```

**5. Concise — target ~200 lines**
AI context windows are precious. The dev tools prompt should be actionable, not encyclopedic. Link to website docs for deep dives.

## Implementation Plan

### Milestone 1: Create Initial Dev Tools Prompt (~200 LOC)

**New file: `prompts/devtools/v0.8.0.md`**
- Workflow-organized reference covering all 8 categories above
- Includes command examples with expected output patterns
- Cross-references to `ailang prompt` for syntax questions

**New file: `prompts/devtools/versions.json`**
- Version tracking (mirrors `prompts/versions.json`)

### Milestone 2: Add CLI Command (~100 LOC)

**Modified: `cmd/ailang/main.go`**
- Register `devtools-prompt` subcommand

**New file: `cmd/ailang/devtools_prompt.go`**
- Mirrors `cmd/ailang/prompt.go` implementation
- Flags: `--version`, `--list`, `--info`
- Embeds devtools prompt markdown via `//go:embed`

### Milestone 3: Integration & Docs (~100 LOC)

**Modified: `cmd/ailang/help.go`**
- Add `devtools-prompt` to help output

**Modified: `docs/docs/guides/` (new page)**
- Website page explaining both prompts and when to use each

**Modified: `CHANGELOG.md`**
- Document the new command

## Content Categories (Detailed)

### Category 1: Running & Testing
Essential for any AI agent writing AILANG code.
- `ailang run` with all important flags (`--caps`, `--entry`, `--ai`, `--emit-trace`, `--verify-contracts`, `--budget-report`)
- `ailang check` with `--timeout` and `--debug-compile`
- `ailang test` for test suites
- `ailang watch` for iterative development

### Category 2: Debugging with Traces
New in Phase 1-3. AI agents can use traces to understand program behavior.
- `--emit-trace jsonl` — captures function calls, effect invocations, contract checks, budget deltas
- `--emit-trace otel` — sends spans to Cloud Trace / observatory
- `--emit-trace auto` — automatically emits OTEL when telemetry configured
- Trace event types: `module_start`, `function_enter/exit`, `effect`, `contract_check`, `budget_delta`, `error`

### Category 3: Verifying Determinism
Key differentiator for AILANG. AI agents should verify their programs produce consistent results.
- `ailang replay` — re-run and compare against baseline
- Exit code semantics: 0=match, 1=mismatch, 2=error
- JSON output for programmatic comparison
- Source file resolution (module name → file path)

### Category 4: Monitoring Agent Execution
For AI agents that delegate sub-tasks or need to understand execution history.
- Chain hierarchy: execution_chains → chain_stages → sessions → spans
- Short chain IDs (like git — first 8 chars)
- Stats and cost tracking

### Category 5: Task Delegation
For agentic workflows where one AI delegates to another.
- Message system for asynchronous communication
- Coordinator daemon for autonomous execution
- Approval workflow for human-in-the-loop

### Category 6: Eval & Benchmarking
For AI agents evaluating their own code generation quality.
- Running benchmarks against specific models
- Comparing baselines across versions
- Dashboard updates with history preservation

### Category 7: Stdlib Exploration
Already partially covered in teaching prompt; dev tools prompt adds deeper exploration.
- `ailang docs` with module-specific documentation
- `ailang builtins` with filtering and validation
- `ailang doctor` for health checks

### Category 8: Telemetry
For AI agents diagnosing performance or understanding distributed execution.
- OTEL configuration check
- Trace listing and viewing
- System health validation

## Success Criteria

- [x] `ailang devtools-prompt` outputs actionable, AI-consumable reference
- [x] Document covers all 10 workflow categories (expanded from 8)
- [ ] Under 250 lines (currently 283 - slightly over, acceptable for 10 categories)
- [x] Version-locked and tracked in versions.json
- [ ] At least 1 external AI agent (coordinator task) successfully uses a dev tool it learned from the prompt
- [ ] Website page documents both prompts

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Document gets stale | AI tries nonexistent commands | Version-lock; update on release |
| Too verbose for context | Wastes context window | Hard cap at 250 lines; link to docs |
| Overlaps with teaching prompt | Confusion about which to use | Clear scope: syntax vs tools |
| New tools added without updating | Missing from reference | Add to release checklist |

## Future Work

- **Auto-generation**: Parse `cmd/ailang/help.go` to auto-generate portions of the dev tools prompt
- **Context-aware**: `ailang devtools-prompt --for debugging` returns only debugging-relevant sections
- **Combined mode**: `ailang prompt --full` returns both syntax + dev tools in one output
- **MCP integration**: Expose as MCP resource for AI agents to query on-demand
