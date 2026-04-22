# M-EXEC-EXPAND: Codex + opencode Executors (Uniform CLI-Subprocess Shape)

**Status**: Planned
**Target**: v0.15.0
**Priority**: P1 (Medium — harness diversity unblocks M-EVAL-EXPAND Sprints 3-4 and M-COORD-CODEX)
**Estimated**: 2 weeks (two ~1-week sprints; Sprint 2 preceded by a ~1-day research spike)
**Dependencies**: None. Uses the existing [`executor.Executor`](../../../internal/executor/executor.go) interface, the [`GlobalFactory()`](../../../internal/executor/factory.go) registration pattern, and the [`ExecutorProvider`](../../../internal/coordinator/provider_executor.go) auto-discovery layer that already ship in v0.14.1.

**Supersedes**:
- [design_docs/planned/v0_13_0/m-coord-codex-executor.md](../v0_13_0/m-coord-codex-executor.md) — its Phase 2 (coordinator wiring) was eliminated by the provider factory refactor; its Phase 1 (executor core) is absorbed here.
- [design_docs/planned/v0_13_0/m-coord-codex-executor-sprint-plan.md](../v0_13_0/m-coord-codex-executor-sprint-plan.md) — same.

**Relationship to M-EVAL-EXPAND**:
Extracts Sprints 3 (M-EVAL-CODEX) and 4 (M-EVAL-OPENCODE's executor half) from [m-eval-expand-harnesses-languages.md](../v0_13_0/m-eval-expand-harnesses-languages.md), leaving that doc to focus on `langreg` + JS/Go + openbench adapter + `ollama-gemma4`. The two tracks are independent and can land in parallel.

**Relationship to M-BRAIN-MICRORAG**:
[M-BRAIN-MICRORAG](m-brain-microrag.md) ships the engine behind `ailang micro-rag context` plus the Claude Code bash-hook frontend. This doc adds **three new microrag frontends** (Gemini CLI hooks, Codex CLI hooks, opencode plugin) in M4A and M7A, reusing the existing harness-agnostic engine. Justified here because this sprint is the only place where all three target executors are touched end-to-end; adding shims alongside wiring is cheaper than a separate sprint.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Executor infrastructure only; no language semantics changes |
| A2: Replayability | +1 | Adds harness as an explicit trace/eval dimension; runs reproducible per executor |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No capability changes; each executor is a named subprocess wrapper |
| A5: Bounded Verification | 0 | No type system changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Broadens the agent audience the harness and coordinator can drive — directly serves AILANG's core thesis |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Per-harness cost/token metrics flow through the same `executor.Result` schema; cross-harness cost comparisons become possible |
| A10: Composability | +1 | Formalizes the "uniform CLI-subprocess executor" shape so future harnesses (aider, cline, roo) plug in via one `init()` |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Expands machine audience coverage

---

## Problem Statement

**Current State:**
- Only two executors are registered in the global factory: `claude` ([claude.go:766](../../../internal/executor/claude/claude.go)) and `gemini` ([gemini.go:553](../../../internal/executor/gemini/gemini.go)).
- [`codex_compat_test.go`](../../../internal/executor/codex_compat_test.go) already documents the Codex NDJSON schema and proves it is **incompatible** with the Claude/Gemini `stream_event` wrapping — a separate parser is required. No executor has been implemented against that blueprint.
- [`internal/eval_harness/models.yml`](../../../internal/eval_harness/models.yml) has **11 `agent_cli: null` lines** annotated `# OpenAI Codex CLI not yet implemented` (lines 15, 31, 44, 63, 82, 101, 119) and `# Uses Responses API for text generation` (lines 141, 162, 182) — documentation promising Codex support that cannot actually run.
- No opencode integration exists at all (no executor, no `codex_compat_test.go` equivalent, no models.yml entries, no research).
- Two overlapping design docs ([M-COORD-CODEX](../v0_13_0/m-coord-codex-executor.md), [M-EVAL-EXPAND](../v0_13_0/m-eval-expand-harnesses-languages.md)) describe partially-decided Codex work across two different scopes, which fragments execution.
- The "CLI-subprocess executor" shape is an *implicit* convention encoded in two files (`claude.go`, `gemini.go`) with no single document specifying it, so a new executor author has to reverse-engineer both to know what's required.

**Impact:**
- `ailang eval-suite --agent --provider codex` fails with `provider "codex" not supported`; the CLI flag exists but has no implementation behind it.
- Cross-harness comparisons ("Claude vs Gemini vs Codex on benchmark X") are impossible.
- Coordinator cannot route tasks to Codex or opencode even when installed, despite `ExecutorProvider` being designed exactly for this.
- Every future CLI agent (aider, cline, roo, ...) re-pays the reverse-engineering tax.

---

## Goals

**Primary Goal:** Add Codex and opencode as first-class CLI-subprocess executors, and formalize the uniform executor shape so future harnesses plug in via a single `init()` registration with zero coordinator or eval-harness code changes.

**Success Metrics:**
- `executor.GlobalFactory().ListAvailable()` returns `["claude", "gemini", "codex", "opencode"]` after binaries are on PATH.
- `ailang eval-suite --models codex-<model> --benchmarks fizzbuzz` and `--models opencode-<model> --benchmarks fizzbuzz` both complete and emit `RunMetrics` byte-identical in schema to Claude's.
- Coordinator auto-discovers both new executors via the existing [`ExecutorProvider`](../../../internal/coordinator/provider_executor.go) with **zero** coordinator package changes (no new `provider_codex.go`, no new `provider_opencode.go`).
- A new executor (e.g., aider) can be added by authoring one file (`internal/executor/<name>/<name>.go` + `init()` → `GlobalFactory().Register(...)`), updating `provider_executor.go`'s blank import list, and adding models.yml entries — no other files change.
- All 11 `agent_cli: null  # OpenAI Codex CLI not yet implemented` annotations in models.yml are resolved: set to `agent_cli: "codex"` for supported models, or the entry documented as truly null-only.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Both Codex and opencode are **CLI-subprocess executors**, not direct-API integrations | Preserves the single executor shape; reuses `ExecutorProvider`; matches existing Claude/Gemini; zero coordinator diff | human | design | high |
| Ship Codex in Sprint 1, opencode in Sprint 2 (after research spike) | Codex has an NDJSON blueprint ([codex_compat_test.go](../../../internal/executor/codex_compat_test.go)); opencode does not — sequencing cuts risk | human | design | med |
| Formalize the uniform shape in a new `docs/internal/EXECUTOR_SHAPE.md` | Turns an implicit convention (learned by reading two files) into explicit contract; each future harness lands in hours, not days | human | design | med |
| `agent_suite` composite model grouping covers claude + gemini + codex + opencode | Enables one-command cross-harness sweeps (`--models agent_suite`); matches the M-EVAL-EXPAND plan | human | design | low |
| Integration tests gated on CLI presence (skip, not fail, when binary missing) | CI remains reliable without requiring every contributor to install 4 agent CLIs; live paths still tested when available | agent | compile | low |
| Codex CLI target is `codex` (OpenAI official), opencode CLI target is `opencode` (opencode-ai/opencode) | Pins the specific binary so the executor invocation is unambiguous | human | design | low |

### Design Freeze

Before Sprint 1 starts, these must be resolved:

- [ ] Approve the uniform CLI-subprocess executor contract (below, §"Uniform CLI-Subprocess Executor Shape"). Any new required hook for Codex/opencode (e.g., stdin-prompt vs. argv-prompt) is added to the contract before the first executor ships.
- [ ] Confirm exact Codex CLI flags for: model selection, non-interactive JSON output, permission bypass (analog of Claude's `--dangerously-skip-permissions`), and auth mode. Document in `internal/executor/codex/README.md` when resolved.
- [ ] Confirm `agent_suite` membership: {claude, gemini, codex, opencode}. If opencode integration slips, `agent_suite` ships with three and opencode is added post-hoc.

Before Sprint 2 starts:

- [ ] Research spike deliverable: `internal/executor/opencode_compat_test.go` (mirrors `codex_compat_test.go`) must exist and document the opencode stream format before the opencode executor is written. If the format is infeasible to parse cleanly (e.g., unversioned schema, no JSON mode), opencode is deferred and this doc ships codex-only.

---

## Solution Design

### Overview

Two sprints. Each sprint lands one executor by (a) authoring the package, (b) registering via `init()`, (c) updating `provider_executor.go` blank imports, (d) wiring models.yml. No coordinator or eval-harness dispatch code changes.

### Uniform CLI-Subprocess Executor Shape

The shape is already present implicitly in `claude.go` and `gemini.go`. This section makes it explicit so future executors (aider, cline, ...) are a mechanical exercise.

**Required package layout:**

```
internal/executor/<name>/
  <name>.go           # Executor implementation
  <name>_test.go      # Parser + registration tests
  README.md           # CLI invocation flags, env vars, known limits
```

**Required symbols:**

1. **`type <Name>Executor struct { ... }`** — implements [`executor.Executor`](../../../internal/executor/executor.go) (7 methods: `Name`, `Execute`, `ExecuteStreaming`, `Capabilities`, `CostModel`, `HealthCheck`, `Close`).
2. **`func New(cfg *executor.Config) (*<Name>Executor, error)`** — constructor; validates binary on PATH, parses flags.
3. **`func Register()`** — `executor.GlobalFactory().Register("<name>", func(cfg *executor.Config) (executor.Executor, error) { return New(cfg) })`.
4. **`func init() { Register() }`** — automatic registration when the package is imported.

**Required wiring (one line per executor):**

In [`internal/coordinator/provider_executor.go`](../../../internal/coordinator/provider_executor.go) top-of-file blank-import block:

```go
_ "github.com/sunholo-data/ailang/internal/executor/<name>"  // triggers init() registration
```

That is the *only* coordinator change needed. `ExecutorProvider.CanHandle()` already returns `true` for any registered executor; `executor.GlobalFactory().GetExecutor(name)` finds it; `Execute()` routes through the standard `executor.Task` → `executor.Result` flow.

**Required models.yml updates:**

Per-model entry for each model this executor supports:

```yaml
<model-id>:
  api_name: "<api-model-name>"
  provider: "<openai|opencode|etc>"
  agent_cli: "<executor-name>"       # REQUIRED — must match GlobalFactory() key
  agent_model_name: "<flag-value>"   # What to pass to the CLI's --model flag
  pricing: {...}
```

And optionally, add to the `agent_suite` composite (see §Architecture).

**Required test coverage:**

- Format fixture test: unit test parses a recorded CLI output fixture into `executor.Result` with expected token/cost/turn counts.
- Registration test: `TestInit_Registers<Name>` verifies the package's `init()` makes the name resolvable from `GlobalFactory().ListAvailable()`.
- Gated integration test: `TestLiveRun_<Name>` with `t.Skip` guard when binary is missing from PATH.

### Architecture

**Components:**

1. **Codex Executor** (`internal/executor/codex/codex.go`, ~500 LOC mirroring [gemini.go](../../../internal/executor/gemini/gemini.go)) — subprocess wrapper around the `codex` CLI. Spawns the binary, pipes stdin/stdout, parses the NDJSON schema documented in [codex_compat_test.go](../../../internal/executor/codex_compat_test.go): `{"type":"message","turn_number":N,"text":"...","tokens_used":{"input":N,"output":N}}`. Field nesting is flat (unlike Claude's nested `event.message.usage`), so a dedicated parser is required and cannot be shared with the Claude/Gemini parser (this is the central finding of `codex_compat_test.go`).

2. **opencode Executor** (`internal/executor/opencode/opencode.go`, ~500 LOC, exact shape determined by spike) — subprocess wrapper around the `opencode` CLI. Stream format documented by the Sprint 2 research spike into `internal/executor/opencode_compat_test.go`.

3. **Executor Shape Contract** (`docs/internal/EXECUTOR_SHAPE.md`, new, ~200 LOC) — one-page document spelling out the conventions above, referenced from `CLAUDE.md` and [.claude/rules/coordinator.md](../../../.claude/rules/coordinator.md).

4. **Updated `provider_executor.go` blank imports** — adds two lines (codex, opencode) to the existing `_ "..."` block. No functional changes to `ExecutorProvider`.

5. **models.yml wiring** — resolves all 11 `agent_cli: null  # OpenAI Codex CLI not yet implemented` lines to `agent_cli: "codex"` where OpenAI models support Codex CLI; adds new `opencode-*` entries if opencode has its own branded models; adds `agent_suite` composite (per [m-eval-expand-harnesses-languages.md §Sprint 3](../v0_13_0/m-eval-expand-harnesses-languages.md)).

6. **Multi-harness microrag frontends** — three new thin adapters that invoke the existing `ailang micro-rag context` CLI (engine already shipped by [M-BRAIN-MICRORAG](m-brain-microrag.md)):
   - **Gemini CLI hook shim** — `.gemini/settings.json` + bash shim mirroring Claude Code's `microrag_context.sh`. Gemini CLI supports `PreToolUse`, `AfterTool`, `SessionStart` events via stdin/stdout JSON — near-identical shape to Claude Code hooks, so the existing shim is reusable with a minimal config wrapper.
   - **Codex CLI hook shim** — `hooks.json` config (documented at developers.openai.com/codex/hooks) plus the same bash shim. Codex exposes `PreToolUse`/`PostToolUse`/`SessionStart` events with a JSON protocol compatible with the Gemini/Claude shim.
   - **opencode plugin** — TypeScript/JavaScript module exporting hook functions. opencode's plugin system is a different shape (JS module, not bash) but the logic is equivalent: on tool event, shell out to `ailang micro-rag context`, inject returned snippet.

   All three reuse the existing engine (`ailang micro-rag context` CLI) and the existing eval toggle (`AILANG_MICRORAG_ENABLED` env var checked inside the engine, not in the shim), so A/B measurement works uniformly across all four harnesses.

### Implementation Plan

**Sprint 1: M-EXEC-CODEX** (~1 week)

- [ ] Create `internal/executor/codex/codex.go` scaffolded from [gemini.go](../../../internal/executor/gemini/gemini.go) — copy-modify approach is intentional; the shape *is* the pattern.
- [ ] Port NDJSON parser directly from [codex_compat_test.go](../../../internal/executor/codex_compat_test.go) fixtures (`{"type":"message",...}`). Store raw event in `Result.ProviderData` per the tolerance pattern in [M-COORD-CODEX Risks & Mitigations](../v0_13_0/m-coord-codex-executor.md).
- [ ] Implement `HealthCheck(ctx)` via `codex --version` (same pattern as `gemini --version`).
- [ ] Register via `init()` → `GlobalFactory().Register("codex", ...)`.
- [ ] Determine and document: `--model` flag, non-interactive JSON mode flag, permission bypass equivalent, `--non-interactive` or equivalent. Document in `internal/executor/codex/README.md`.
- [ ] Add `_ "github.com/sunholo-data/ailang/internal/executor/codex"` to [provider_executor.go:10](../../../internal/coordinator/provider_executor.go) blank-import block.
- [ ] Update [models.yml](../../../internal/eval_harness/models.yml): replace all 11 `agent_cli: null  # OpenAI Codex CLI not yet implemented` lines with `agent_cli: "codex"` for models that support it. Set `agent_model_name` to the correct flag value per model.
- [ ] Add `agent_suite` composite (claude + gemini + codex; opencode added in Sprint 2 or deferred).
- [ ] Unit tests: NDJSON parser fixtures, registration test, CostModel calculation from `tokens_used`.
- [ ] Gated integration test: `ailang eval-suite --models codex-gpt-5 --benchmarks fizzbuzz` — skips cleanly if `codex` not on PATH, runs end-to-end if present.
- [ ] Author `docs/internal/EXECUTOR_SHAPE.md` documenting the uniform contract (can defer authorship until after codex lands, but must land in Sprint 1).
- [ ] **M4A — Microrag frontends for Gemini + Codex**: Author harness-neutral `microrag_context.sh` variant (or confirm the existing Claude Code shim is reusable verbatim); ship `.claude/skills/microrag/frontends/gemini/settings.json` and `.../codex/hooks.json` templates; document setup in [`m-brain-microrag.md`](m-brain-microrag.md) §Frontends as D (Gemini) and E (Codex); verify via toy-prompt live tests that μRAG injection marker appears in both harness logs when `AILANG_MICRORAG_ENABLED=true`.

**Sprint 2: M-EXEC-OPENCODE** (~1 week + 1-day spike)

- [ ] **Research spike (Day 1)**: clone/install opencode locally, run it non-interactively, capture stream output to `testdata/opencode_response.jsonl`, author `internal/executor/opencode_compat_test.go` documenting the schema (mirrors `codex_compat_test.go`). **Design-freeze checkbox**: if schema is infeasible, defer opencode and close this sprint.
- [ ] Create `internal/executor/opencode/opencode.go` using the shape doc as checklist.
- [ ] Register via `init()` → `GlobalFactory().Register("opencode", ...)`.
- [ ] Add `_ "github.com/sunholo-data/ailang/internal/executor/opencode"` to [provider_executor.go:10](../../../internal/coordinator/provider_executor.go) blank-import block.
- [ ] Update models.yml: add opencode-capable entries, update `agent_suite` membership.
- [ ] Unit tests + gated integration test (same pattern as Codex).
- [ ] Update `docs/internal/EXECUTOR_SHAPE.md` with any contract refinements uncovered by the spike (e.g., stdin-prompt vs. argv-prompt).
- [ ] **M7A — Microrag frontend for opencode**: Author `internal/executor/opencode/plugins/microrag-plugin.ts` (TypeScript plugin loaded by opencode), calling out to `ailang micro-rag context`; document plugin install path and config in `internal/executor/opencode/README.md`; add §Frontend F to [`m-brain-microrag.md`](m-brain-microrag.md); verify μRAG injection marker appears in opencode session log when toggle is ON.
- [ ] Cross-harness E2E: `ailang eval-suite --models agent_suite --benchmarks fizzbuzz` runs all four harnesses and produces identical `RunMetrics` schema; microrag toggle (`AILANG_MICRORAG_ENABLED`) measurable across all four harnesses.

### Files to Modify/Create

**New files:**
- `internal/executor/codex/codex.go` — Codex executor (~500 LOC, mirrors `gemini/gemini.go`)
- `internal/executor/codex/codex_test.go` — NDJSON parser + registration tests (~200 LOC)
- `internal/executor/codex/README.md` — Flag reference, known limits (~80 LOC)
- `internal/executor/codex/testdata/codex_response.jsonl` — NDJSON fixture from a live run
- `internal/executor/opencode/opencode.go` — opencode executor (~500 LOC, shape determined by spike)
- `internal/executor/opencode/opencode_test.go` — parser + registration tests (~200 LOC)
- `internal/executor/opencode/README.md` — Flag reference (~80 LOC)
- `internal/executor/opencode/testdata/opencode_response.jsonl` — fixture
- `internal/executor/opencode_compat_test.go` — opencode format analysis (~200 LOC, mirrors `codex_compat_test.go`)
- `docs/internal/EXECUTOR_SHAPE.md` — Uniform CLI-subprocess executor contract (~200 LOC)
- `.claude/skills/microrag/frontends/gemini/settings.json` — Gemini CLI hooks template (~40 LOC)
- `.claude/skills/microrag/frontends/gemini/microrag_context.sh` — Gemini-compatible shim (symlink or copy of existing Claude shim) (~20 LOC)
- `.claude/skills/microrag/frontends/codex/hooks.json` — Codex CLI hooks template (~40 LOC)
- `.claude/skills/microrag/frontends/codex/microrag_context.sh` — Codex-compatible shim (~20 LOC)
- `internal/executor/opencode/plugins/microrag-plugin.ts` — opencode plugin (~100 LOC)
- `internal/executor/opencode/plugins/package.json` — plugin manifest (~20 LOC)
- `internal/executor/opencode/plugins/microrag-plugin.test.ts` — plugin unit tests (~80 LOC)

**Modified files:**
- [internal/coordinator/provider_executor.go](../../../internal/coordinator/provider_executor.go) — Add two blank imports (~2 LOC)
- [internal/eval_harness/models.yml](../../../internal/eval_harness/models.yml) — Resolve 11 `agent_cli: null` lines to `agent_cli: "codex"`; add opencode entries; add `agent_suite` composite (~80 LOC net)
- [.claude/rules/coordinator.md](../../../.claude/rules/coordinator.md) — Reference `docs/internal/EXECUTOR_SHAPE.md` (~5 LOC)
- `CHANGELOG.md` — v0.15.0 entry

**Not modified (intentional — proves the uniform shape works):**
- `internal/coordinator/provider.go`, `internal/coordinator/task_executor.go`, any other coordinator dispatch code
- `internal/eval_harness/agent_runner.go`, `agent_runner_multi.go`, `runner.go` — agent_cli dispatch is already data-driven via models.yml

---

## Examples

### Example 1: Cross-harness eval sweep

**Before:**
```bash
ailang eval-suite --models claude-opus-4-7,gemini-3-pro --benchmarks fizzbuzz
# codex and opencode not available
```

**After:**
```bash
ailang eval-suite --models agent_suite --benchmarks fizzbuzz
# runs claude + gemini + codex + opencode, identical RunMetrics schema
```

### Example 2: Coordinator routes to Codex

**Before:**
```bash
ailang messages send coordinator "Fix bug in parser" --provider codex
# → error: provider "codex" not registered
```

**After:**
```bash
ailang messages send coordinator "Fix bug in parser" --provider codex
# → ExecutorProvider auto-discovers codex from GlobalFactory().ListAvailable()
# → runs via codex CLI, metrics captured in executor.Result
```

Zero new coordinator code made this work — only the blank import in `provider_executor.go`.

### Example 3: Adding aider (hypothetical) after this sprint

```go
// internal/executor/aider/aider.go
package aider

func init() { Register() }
func Register() {
    executor.GlobalFactory().Register("aider", func(cfg *executor.Config) (executor.Executor, error) {
        return New(cfg)
    })
}
// ... New(), Execute(), ExecuteStreaming(), etc. per EXECUTOR_SHAPE.md
```

Then one line in `provider_executor.go`:
```go
_ "github.com/sunholo-data/ailang/internal/executor/aider"
```

Plus models.yml entries. That's it. Coordinator, eval harness, dashboard, chains view — all adapt automatically.

---

## Success Criteria

- [ ] **Sprint 1**: `executor.GlobalFactory().ListAvailable()` includes `codex`; `ailang eval-suite --models codex-gpt-5 --benchmarks fizzbuzz` runs green and emits `RunMetrics` with tokens/cost/turns populated; all 11 `# OpenAI Codex CLI not yet implemented` annotations in models.yml resolved; Gemini + Codex microrag shims injecting `🧠 μRAG` marker when `AILANG_MICRORAG_ENABLED=true`
- [ ] **Sprint 2**: `opencode_compat_test.go` documents the stream format; `executor.GlobalFactory().ListAvailable()` includes `opencode`; `ailang eval-suite --models opencode-<model> --benchmarks fizzbuzz` runs green; `agent_suite` runs all four harnesses; opencode microrag plugin injecting `🧠 μRAG` marker when toggle is ON
- [ ] **Uniform shape**: `docs/internal/EXECUTOR_SHAPE.md` exists and is referenced from coordinator rules
- [ ] **Zero coordinator changes beyond blank imports**: a diff audit confirms `provider_executor.go` gains 2 import lines and nothing else; no new `provider_<name>.go` files
- [ ] Coordinator `ailang messages send coordinator "..." --provider codex` completes end-to-end
- [ ] All tests passing (`make ci`)
- [ ] Gated integration tests skip cleanly on CI where `codex`/`opencode` binaries are absent
- [ ] Documentation updated: `docs/internal/EXECUTOR_SHAPE.md`, CHANGELOG.md, [docs/docs/guides/evaluation/](../../../docs/docs/guides/evaluation/)

---

## Testing Strategy

**Unit tests:**
- NDJSON parser: Codex fixture → `executor.Result` with known token/cost/turn values. Include edge cases: missing `tokens_used`, schema drift (extra fields), truncated stream.
- opencode parser: symmetric with format determined by spike.
- Registration: assert both executors appear in `GlobalFactory().ListAvailable()` after package import.
- CostModel: verify `CalculateCost` matches documented CLI billing for each supported model.

**Integration tests:**
- Gated `TestLiveRun_Codex` / `TestLiveRun_Opencode` — skip if binary absent, run a toy prompt if present, verify non-empty `Result.Output` and populated metrics.
- Cross-harness E2E: `ailang eval-suite --models agent_suite --benchmarks fizzbuzz` produces 4 result rows with identical JSON schema.
- Coordinator dry-run: `ailang messages send coordinator "test" --provider codex --dry-run` confirms routing without executing.

**Manual testing:**
- `ailang coordinator pending --provider codex` after queueing a real task verifies full task flow (approval → execution → artifacts).
- Confirm chains viewer (`ailang chains view <id> --spans`) shows codex spans with correct executor name tag.

---

## Deferred Decisions

The following are intentionally left to the implementer:

- **Codex JSON mode flag name** — may have changed since `codex_compat_test.go` was authored; agent probes current CLI and documents in README.
- **opencode parser shape** — emerges from the research spike; agent designs based on `opencode_compat_test.go` findings.
- **Whether opencode gets its own branded `opencode-*` models in models.yml or just an `agent_cli` on existing OpenAI entries** — depends on whether opencode exposes a model-selection abstraction separate from OpenAI's.
- **How to surface `agent_cli: null` models that Codex could theoretically support** — agent decides whether to flip to `"codex"` (opt-in) or keep `null` (explicit non-support). Err on the side of explicit.

## Non-Goals

**Not attempted in this feature:**
- **Language registry refactor (`langreg`)** — M-EVAL-EXPAND Sprint 1. Independent and parallelizable.
- **JavaScript + Go language support** — M-EVAL-EXPAND Sprint 2. Independent and parallelizable.
- **openbench adapter** — M-EVAL-EXPAND Sprint 4.
- **`ollama-gemma4` or other open-source model entries** — M-EVAL-EXPAND Sprint 4.
- **Direct OpenAI Responses API integration as an executor** — Codex CLI is the chosen path; Responses API remains separate (used by the text-generation `ai_provider` path, not the agentic executor path).
- **Aider, cline, roo integrations** — the shape doc makes these trivial follow-ups, but they ship separately.
- **Switching the default executor** — `gemini` remains default per [factory.go:44](../../../internal/executor/factory.go).
- **Changes to AILANG language semantics** — pure infrastructure sprint.

## Timeline

**Week 1** (M-EXEC-CODEX):
- Days 1-2: Codex executor core + NDJSON parser + unit tests
- Day 3: models.yml wiring + `agent_suite` + registration test
- Day 4: Integration test + `README.md` + `EXECUTOR_SHAPE.md`
- Day 5: Polish, CHANGELOG, review

**Week 2** (M-EXEC-OPENCODE):
- Day 1: Research spike — author `opencode_compat_test.go`; design-freeze checkpoint
- Days 2-3: opencode executor core + parser + unit tests
- Day 4: models.yml wiring + integration test + README
- Day 5: Cross-harness E2E + docs + CHANGELOG

**Total: ~2 weeks across 2 sprints**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Codex CLI JSON schema has drifted since `codex_compat_test.go` was authored | Med | Tolerant parsing; store raw event in `Result.ProviderData`; fixture captured from live run pins current schema |
| Codex has no permission-bypass flag (unlike Claude's `--dangerously-skip-permissions`) | Med | Document as limitation in README; surface as a missing `Capability` (e.g., absence of `CapToolControl`); fall back to default-interactive mode if CLI supports it |
| opencode stream format is not JSON or lacks a non-interactive mode | High | Research spike occurs before implementation starts; if infeasible, opencode is deferred and this doc ships codex-only |
| opencode CLI is under active rewrite and schema changes mid-sprint | Med | Pin opencode version in README; gate integration test on version match |
| `agent_suite` composite runs 4x the cost per benchmark sweep | Low | Document in evaluation guide; default `make eval-smoke` remains single-model |
| Future aider/cline executors reveal the shape contract was incomplete | Low | `EXECUTOR_SHAPE.md` is a living doc; v0.15.0 captures what we know; extensions are additive |

## Related Documents

<!-- Auto-populated by Ollama neural search on "exec expand codex opencode" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_14_0/m-eval-suite-prep-sprint-plan.md](../../implemented/v0_14_0/m-eval-suite-prep-sprint-plan.md) — just-landed tier/tag infrastructure that this work feeds into
- [design_docs/implemented/v0_10_0/m-exit-code.md](../../implemented/v0_10_0/m-exit-code.md) (0.41)
- [design_docs/implemented/v0_6_1/m-exec-gemini-sprint-plan.md](../../implemented/v0_6_1/m-exec-gemini-sprint-plan.md) — original Gemini executor sprint; reference for the shape

**Planned (check for overlap):**
- [design_docs/planned/v0_13_0/m-eval-expand-harnesses-languages.md](../v0_13_0/m-eval-expand-harnesses-languages.md) — **complementary**, not overlapping: this doc owns executors (Sprints 3-4 of that doc); that doc owns languages (Sprints 1-2) and openbench/Gemma (Sprint 4 tail)
- [design_docs/planned/v0_13_0/m-coord-codex-executor.md](../v0_13_0/m-coord-codex-executor.md) — **superseded** by this doc
- [design_docs/planned/v0_13_0/m-coord-codex-executor-sprint-plan.md](../v0_13_0/m-coord-codex-executor-sprint-plan.md) — **superseded** by this doc

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [internal/executor/executor.go](../../../internal/executor/executor.go) — `Executor` interface (7 methods)
- [internal/executor/factory.go](../../../internal/executor/factory.go) — `GlobalFactory()` and `Register` pattern
- [internal/executor/claude/claude.go:766](../../../internal/executor/claude/claude.go) — reference `init()` registration
- [internal/executor/gemini/gemini.go:553](../../../internal/executor/gemini/gemini.go) — reference `init()` registration
- [internal/executor/codex_compat_test.go](../../../internal/executor/codex_compat_test.go) — NDJSON schema blueprint
- [internal/coordinator/provider_executor.go](../../../internal/coordinator/provider_executor.go) — auto-discovery layer; the reason coordinator needs no new code
- [internal/eval_harness/models.yml](../../../internal/eval_harness/models.yml) — `agent_cli` field pattern

## Future Work

- **Aider, cline, roo executors** — one file each per `EXECUTOR_SHAPE.md`; potential v0.16.0+ follow-up sprint.
- **OpenAI Responses API as a *text-generation* provider** (orthogonal to Codex CLI) — remains on the `internal/ai/` side of the provider-vs-executor split documented in [CLAUDE.md](../../../CLAUDE.md).
- **Per-harness span tagging in chains viewer** — surface `executor_name` as a filter in `ailang chains list --executor codex`.
- **Cross-harness comparison reports** — "benchmark X on Claude vs Gemini vs Codex vs opencode side-by-side" follow-up after baselines accumulate.
- **Feed findings into [M-EVAL-XLANG](../v0_11_0/m-eval-cross-language-benchmark.md)** — once we have codex+opencode numbers, direct comparison with external benchmarks that include those harnesses.

---

**Document created**: 2026-04-22
**Last updated**: 2026-04-22

---
DESIGN_DOC_PATH: design_docs/planned/v0_15_0/m-exec-expand-codex-opencode.md
