# M-EVAL-EXPAND: Expanding the AILANG Eval Bench (Harnesses, Languages, Open-Source Models)

**Status**: Planned
**Target**: v0.13.0
**Priority**: P1 (Medium — strategic eval coverage, not a release blocker)
**Estimated**: 4 weeks (four ~1-week sprints, sequenced)
**Dependencies**: Existing eval harness (`internal/eval_harness/`), executor factory (`internal/executor/`), Ollama integration (`internal/ai/ollama/`)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Eval infrastructure only; no language semantics changes |
| A2: Replayability | +1 | Harness/model/language added as explicit eval dimensions; runs reproducible |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No capability changes; each harness is a named executor |
| A5: Bounded Verification | 0 | No type system changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Broadens coverage of the machine audience (more agents, more models), directly serving AILANG's core thesis |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Adds `$0.00` local-model lane and cost-per-harness tracking |
| A10: Composability | +1 | New language registry replaces scattered `switch lang` dispatch — more composable eval pipeline |
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
- Every benchmark in [benchmarks/](../../benchmarks) declares `languages: ["python", "ailang"]` — no JavaScript, Go, or other mainstream language coverage.
- Only two agent harnesses are wired: Claude ([internal/executor/claude/](../../internal/executor/claude/)) and Gemini ([internal/executor/gemini/](../../internal/executor/gemini/)). Codex format analysis exists in [internal/executor/codex_compat_test.go](../../internal/executor/codex_compat_test.go) but no executor. Opencode and openbench are not integrated.
- Ollama is wired (including `ollama-gemma3` in [internal/eval_harness/models.yml:479-490](../../internal/eval_harness/models.yml)), but there is no `ollama-gemma4` entry, no documented path for exercising any Ollama model against the eval harness, and open-source model evaluations happen ad hoc.
- The chosen primary open-source target is **Gemma 4** (via `ollama pull gemma4:<size>`) because it is the newest generation with strongest reported coding capability; Gemma 3 is the on-disk fallback until Gemma 4 is pulled locally.
- Language dispatch lives in ~10 `switch lang` sites across [agent_prompt.go](../../internal/eval_harness/agent_prompt.go), [agent_runner.go:514](../../internal/eval_harness/agent_runner.go), [runner.go:489](../../internal/eval_harness/runner.go), and [spec.go:126](../../internal/eval_harness/spec.go). Each new language currently multiplies the dispatch surface.

**Impact:**
- Cross-language comparisons are limited to Python, which understates AILANG's positioning against the languages most AI-generated code actually targets.
- We cannot evaluate new coding agents (Codex, opencode) against AILANG without substantial engineering per agent.
- Open-source model coverage is fragmented — we can't run frequent, low-cost regression sweeps against Gemma/DeepSeek/Llama to track their capability trajectory.
- Adding even one more language without refactoring the dispatch surface means ~10 edits across files, raising the cost of every future language addition.

**This doc is distinct from [M-EVAL-XLANG](v0_11_0/m-eval-cross-language-benchmark.md):** that doc proposes joining *external* third-party benchmark suites (AutoCodeBench, leetgptsolver). This doc extends the *internal* eval bench so we own the dimensions we care about (harness × language × model × benchmark).

---

## Goals

**Primary Goal:** Extend the internal eval bench so that any combination of {harness, language, model, benchmark} can be evaluated without per-language or per-harness refactoring.

**Success Metrics:**
- At least 10 existing benchmarks run under JavaScript and Go (in addition to Python and AILANG).
- At least three agent harnesses are wired (Claude, Gemini, Codex) via a single factory pattern; opencode integration is landed or scoped.
- Open-source models (starting with `ollama-gemma4`, with `ollama-gemma3` as the on-disk fallback) are runnable against the eval suite end-to-end with `$0.00` recorded cost.
- Adding a new language requires a single file change (`internal/eval_harness/langreg/<name>.go`) and a registry entry — no edits to `agent_prompt.go`, `agent_runner.go`, `runner.go`, or `spec.go`.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Introduce a `langreg` language registry before adding JS/Go | Without it, every new language multiplies dispatch edits across ~10 sites; with it, each language is one file | human | design | high |
| Codex shipped before opencode | Codex format already analyzed ([codex_compat_test.go](../../internal/executor/codex_compat_test.go)); opencode needs a research spike | human | design | med |
| Gemma 4 is the primary open-source model target | Newest generation; user explicitly chose it over Gemma 3 for initial coverage | human | design | low |
| Gemma / openbench local-dev only (no CI cadence) | Keeps CI cost predictable; avoids flakiness from local Ollama availability in CI | human | design | low |
| Reference solutions limited to 10 benchmarks for starter JS/Go | Full port of 50+ benchmarks would be weeks of solution-writing; 10 gives representative coverage | human | design | med |
| Language system prompts live in Go code (not YAML) | Type safety over hot-reload; prompts are code-adjacent and change with the registry | agent | compile | low |
| openbench integration shape (adapter vs drop-in replacement) | Determines whether we invest in our own pipeline or delegate | human | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Approve the `langreg` interface signature (`Name`, `FileExt`, `SystemPrompt`, `BuildCmd`, `RunCmd`, `VerifyOutput`) before M-EVAL-LANGREG starts.
- [x] **Decided**: Codex executor targets the `codex` CLI (not the OpenAI Responses API) to match the Claude and Gemini integration pattern — one uniform subprocess/NDJSON executor shape across all three harnesses.
- [ ] Confirm openbench integration is an adapter in `internal/eval_harness/openbench/`, not a replacement for our harness.
- [ ] Confirm the exact Gemma 4 Ollama tag to target (e.g., `gemma4:12b` vs `gemma4:27b`) — resolve once the tag is published and pulled locally.

---

## Solution Design

### Overview

Four sequenced sprints. Track A (harnesses) and Track B (languages) are independent; Track B is best done after its prerequisite (the language registry). Track C (open-source via openbench) is last because it depends on a research spike to confirm integration shape.

### Architecture

**Components:**

1. **Language Registry** (`internal/eval_harness/langreg/`) — new package. A `Language` interface with concrete implementations for each supported language. Replaces every `switch lang` site in the eval harness.

2. **New Harness Executors** (`internal/executor/codex/`, `internal/executor/opencode/`) — new packages mirroring [`internal/executor/gemini/gemini.go`](../../internal/executor/gemini/gemini.go). **Both are CLI-subprocess executors** (not direct-API integrations): Codex uses the `codex` CLI, opencode uses the `opencode` CLI, matching the Claude/Gemini pattern so all four harnesses share one uniform executor shape. Each implements the 7-method [`executor.Executor`](../../internal/executor/executor.go) interface and self-registers via `init()` → `GlobalFactory().Register(name, ...)` per the pattern in [claude.go:771-773](../../internal/executor/claude/claude.go).

3. **Reference Solutions** (`examples/reference/<bench>/main.{js,go}`) — one pair per starter benchmark, used by the test harness to verify JS/Go runs produce correct output.

4. **openbench Adapter** (`internal/eval_harness/openbench/`) — thin wrapper that maps one of our benchmarks → openbench task format, invokes openbench, parses results back into `RunMetrics`.

### Implementation Plan

**Sprint 1: M-EVAL-LANGREG** (~1 week)
- [ ] Define `langreg.Language` interface in `internal/eval_harness/langreg/langreg.go`.
- [ ] Implement `python.go` and `ailang.go` concretes with parity to current behavior.
- [ ] Replace `switch lang` in [agent_prompt.go](../../internal/eval_harness/agent_prompt.go) (5 sites), [agent_runner.go:514](../../internal/eval_harness/agent_runner.go), [runner.go:489](../../internal/eval_harness/runner.go), [spec.go:126](../../internal/eval_harness/spec.go).
- [ ] Unit tests for `langreg.Get` lookups and interface contract.
- [ ] Verify no behavioral regression: `make ci` passes and all existing eval benchmarks produce identical output bytes.

**Sprint 2: M-EVAL-LANG-JSGO** (~1 week)
- [ ] Add `javascript.go` and `go.go` to `internal/eval_harness/langreg/`.
- [ ] Add toolchain presence checks (`node --version`, `go version`) to executor `HealthCheck` and CI.
- [ ] Extend 10 starter benchmarks' `languages:` arrays (fizzbuzz, recursion_fibonacci, graph_bfs, binary_tree_sum, balanced_parens, csv_to_json_converter, expression_evaluator, gcd_lcm, fold_reduce, higher_order_functions).
- [ ] Author reference solutions under `examples/reference/<bench>/main.{js,go}`.
- [ ] CI job to verify reference solutions execute and produce expected output.

**Sprint 3: M-EVAL-CODEX** (~1 week)
- [ ] Create `internal/executor/codex/codex.go` as a **`codex` CLI subprocess executor** — same shape as `internal/executor/gemini/gemini.go` and `internal/executor/claude/claude.go`. Spawns `codex` with the right flags, pipes stdin/stdout, parses streaming NDJSON. **Not** a direct-API integration.
- [ ] Port NDJSON parser from blueprint in [codex_compat_test.go](../../internal/executor/codex_compat_test.go) (events are `{"type":"message","text":...}`).
- [ ] Register via `init()` → `GlobalFactory().Register("codex", ...)`.
- [ ] Determine correct `codex` CLI invocation flags (auth, model selection, JSON output, permission bypass equivalent to Claude's `--dangerously-skip-permissions`). Document in the executor README.
- [ ] Add `codex-*` entries to [internal/eval_harness/models.yml](../../internal/eval_harness/models.yml) with `agent_cli: "codex"` (mirrors the `agent_cli: "claude"` / `agent_cli: "gemini"` pattern).
- [ ] Add `agent_suite` to models.yml (line 521-574 region) containing claude + gemini + codex.
- [ ] Add `codex --version` to executor `HealthCheck` and CI toolchain presence checks (same pattern as the `node`/`go` checks added in Sprint 2).
- [ ] E2E test: `ailang eval-suite --models codex-<model> --benchmarks fizzbuzz` captures metrics in the same schema as Claude.

**Sprint 4: M-EVAL-OPENCODE-OPENBENCH** (~1 week)
- [ ] Research spike (~1 day): document opencode's stream format (produce `opencode_compat_test.go` matching the codex-compat precedent); confirm which `openbench` we're targeting (likely `openbench-ai/openbench`).
- [ ] Create `internal/executor/opencode/opencode.go`.
- [ ] Create `internal/eval_harness/openbench/adapter.go` — maps one benchmark task to openbench format, invokes openbench, parses results into `RunMetrics`.
- [ ] Add `ollama-gemma4` entry to [internal/eval_harness/models.yml](../../internal/eval_harness/models.yml) mirroring the `ollama-gemma3` block (lines 479-490). Pin `api_name` to the chosen Gemma 4 tag (e.g., `gemma4:12b`); `pricing.input_per_1k: 0.0`, `pricing.output_per_1k: 0.0`.
- [ ] Document local-dev workflow for Gemma 4 (`ollama serve && ollama pull gemma4:<tag> && ailang eval-suite --models ollama-gemma4 --benchmarks fizzbuzz`); include a note that `ollama-gemma3` remains available as the on-disk fallback.

### Files to Modify/Create

**New files:**
- `internal/eval_harness/langreg/langreg.go` — Language interface + registry (~150 LOC)
- `internal/eval_harness/langreg/python.go` — Python impl (~80 LOC)
- `internal/eval_harness/langreg/ailang.go` — AILANG impl (~80 LOC)
- `internal/eval_harness/langreg/javascript.go` — JS impl (~80 LOC)
- `internal/eval_harness/langreg/go.go` — Go impl (~80 LOC)
- `internal/eval_harness/langreg/langreg_test.go` — registry tests (~120 LOC)
- `internal/executor/codex/codex.go` — Codex executor (~500 LOC, mirrors gemini.go)
- `internal/executor/opencode/opencode.go` — opencode executor (~500 LOC, shape from spike)
- `internal/executor/opencode_compat_test.go` — format analysis doc (~200 LOC, mirrors codex_compat_test.go)
- `internal/eval_harness/openbench/adapter.go` — openbench integration (~300 LOC, shape from spike)
- `examples/reference/<bench>/main.js` × 10 — JS reference solutions
- `examples/reference/<bench>/main.go` × 10 — Go reference solutions

**Modified files:**
- [internal/eval_harness/agent_prompt.go](../../internal/eval_harness/agent_prompt.go) — remove 5 `switch lang` sites, delegate to `langreg` (net −80 LOC)
- [internal/eval_harness/agent_runner.go](../../internal/eval_harness/agent_runner.go) line 514 — delegate to `langreg` (net −20 LOC)
- [internal/eval_harness/runner.go](../../internal/eval_harness/runner.go) line 489 — delegate to `langreg` (net −20 LOC)
- [internal/eval_harness/spec.go](../../internal/eval_harness/spec.go) line 126 — delegate to `langreg` (net −20 LOC)
- [internal/eval_harness/models.yml](../../internal/eval_harness/models.yml) — add `codex-*` entries, add `agent_suite`, add `ollama-gemma4` entry mirroring lines 479-490 (~65 LOC)
- 10 benchmark YAMLs in [benchmarks/](../../benchmarks) — extend `languages:` arrays (~10 × 1 LOC)

---

## Examples

### Example 1: Running a benchmark in JavaScript

**Before** (not possible — benchmark only declares python+ailang):
```bash
ailang eval-suite --benchmarks fizzbuzz --languages javascript
# → error: language 'javascript' not supported for benchmark 'fizzbuzz'
```

**After:**
```bash
ailang eval-suite --benchmarks fizzbuzz --languages javascript
# → runs benchmark against fizzbuzz with JS reference solution verification
# → emits same RunMetrics schema as python/ailang runs
```

### Example 2: Running the full agent suite (including Codex)

**Before:**
```bash
ailang eval-suite --models claude-opus-4-7,gemini-3-1-pro --benchmarks fizzbuzz
# no codex support
```

**After:**
```bash
ailang eval-suite --models agent_suite --benchmarks fizzbuzz
# runs claude + gemini + codex against fizzbuzz, each streaming metrics back in identical format
```

### Example 3: Free local-dev run with Gemma 4 (primary open-source target)

```bash
ollama serve &
ollama pull gemma4:<tag>        # e.g., gemma4:12b — tag confirmed in Design Freeze
ailang eval-suite --models ollama-gemma4 --benchmarks fizzbuzz
# cost_usd: 0.00 in the recorded metrics
```

If Gemma 4 is not yet pulled locally, `ollama-gemma3` remains available as a fallback:

```bash
ailang eval-suite --models ollama-gemma3 --benchmarks fizzbuzz
```

### Example 4: Adding a new language (Rust, hypothetical)

**Before the registry:** edit 10 sites across `agent_prompt.go`, `agent_runner.go`, `runner.go`, `spec.go`.

**After the registry:**
```go
// internal/eval_harness/langreg/rust.go
package langreg

func init() {
    Register(&Rust{})
}

type Rust struct{}
func (*Rust) Name() string { return "rust" }
func (*Rust) FileExt() string { return ".rs" }
// ... rest of interface
```

That's it. No other files change.

---

## Success Criteria

- [ ] **M-EVAL-LANGREG**: All existing benchmarks produce byte-identical output before and after the refactor; `make ci` passes; unit tests cover `langreg.Get` for all registered languages.
- [ ] **M-EVAL-LANG-JSGO**: `ailang eval-suite --benchmarks fizzbuzz --languages javascript,go` runs green for all 10 starter benchmarks; reference solutions under `examples/reference/<bench>/main.{js,go}` execute correctly; CI validates Node + Go toolchain presence.
- [ ] **M-EVAL-CODEX**: `ailang eval-suite --models codex-<model> --benchmarks fizzbuzz` completes and emits metrics matching Claude's record schema (tokens, cost_usd, num_turns, duration_ms); `agent_suite` runs three harnesses in one invocation.
- [ ] **M-EVAL-OPENCODE-OPENBENCH**: at least one benchmark runs end-to-end via opencode; at least one benchmark runs under openbench evaluation with results parsed into `RunMetrics`; `ollama-gemma4` is added to `models.yml`; `ailang eval-suite --models ollama-gemma4 --benchmarks fizzbuzz` runs locally with `cost_usd == 0.00`; local-dev Gemma 4 workflow is documented in `docs/docs/guides/evaluation/`.
- [ ] All tests passing
- [ ] Documentation updated (`docs/docs/guides/evaluation/`)
- [ ] Examples added (starter 10 benchmarks have JS + Go reference solutions)

## Testing Strategy

**Unit tests:**
- `langreg_test.go`: contract tests — every registered language correctly implements the interface; `Get` returns the right impl; unknown language returns an error.
- Codex executor: NDJSON parser test cases based on [codex_compat_test.go](../../internal/executor/codex_compat_test.go) fixtures.

**Integration tests:**
- Before/after snapshot: run the current eval suite against a small benchmark set; snapshot output; confirm no drift after the langreg refactor.
- End-to-end: `ailang eval-suite --models codex-<model> --benchmarks fizzbuzz` produces a valid `eval_results/` record.

**Manual testing:**
- Local Gemma 4 run: `ollama serve && ollama pull gemma4:<tag> && ailang eval-suite --models ollama-gemma4 --benchmarks fizzbuzz` — verify `cost_usd == 0.00`.
- Gemma 3 fallback: `ailang eval-suite --models ollama-gemma3 --benchmarks fizzbuzz` — verify existing Ollama entry still works.
- CI toolchain gating: intentionally remove `node` from PATH in a scratch container; verify JS benchmark entries fail with a clear "node not found" error, not a silent skip.

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **`langreg` registry data structure** — map vs slice vs init-based — agent may choose; must support concurrent reads.
- **Opencode stream format parser shape** — depends on research spike output; agent implements based on `opencode_compat_test.go` findings.
- **openbench adapter invocation style** — subprocess vs library — agent may choose based on what openbench exposes.
- **Reference solution style** — idiomatic vs minimal — agent may choose per language (e.g., Go can use stdlib; JS can use Node stdlib only, no npm deps).
- **Which `codex` CLI flags mirror Claude's `--dangerously-skip-permissions`** and which JSON output mode to use — agent decides when porting (CLI target is fixed; only the flag specifics are deferred); document in executor README.

## Non-Goals

**Not attempted in this feature:**
- **Full port of all 50+ benchmarks to JS + Go** — starter 10 only; broader coverage is a follow-up sprint.
- **New `free_suite` for CI cadence of Gemma runs** — per Phase 1 decision, Gemma is local-dev only. No new CI jobs.
- **TypeScript, Rust, Java, or other additional languages** — the registry makes these easy to add later, but they are out of scope here.
- **A new benchmark runner UI** — existing `ailang eval-suite` CLI is sufficient.
- **Replacing the eval harness with openbench** — openbench is an adapter, not a replacement.
- **External third-party benchmark integration** — covered by [M-EVAL-XLANG](v0_11_0/m-eval-cross-language-benchmark.md), not here.

## Timeline

**Week 1** (M-EVAL-LANGREG):
- Sprint 1: Language registry + refactor all dispatch sites
- Snapshot test to confirm no behavioral change

**Week 2** (M-EVAL-LANG-JSGO):
- Sprint 2: JS + Go language entries
- 10 benchmark YAML updates + reference solutions
- CI toolchain checks

**Week 3** (M-EVAL-CODEX):
- Sprint 3: Codex executor + NDJSON parser
- models.yml updates + `agent_suite` definition
- E2E test

**Week 4** (M-EVAL-OPENCODE-OPENBENCH):
- Research spike (day 1)
- Sprint 4: opencode executor + openbench adapter
- Local-dev Gemma docs

**Total: ~4 weeks across 4 sprints**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Langreg refactor introduces subtle behavioral drift | High | Snapshot the current eval output before the refactor; diff after; require byte-identical output on 5+ benchmarks before merging |
| opencode stream format is incompatible with our executor contract | Med | Research spike up-front produces `opencode_compat_test.go`; if infeasible, defer opencode and ship codex + openbench only |
| openbench is not the right abstraction | Med | Spike confirms integration shape before committing adapter design; fallback is to skip openbench and document Gemma as local-dev only |
| CI toolchain flakiness (Node/Go versions drift) | Low | Pin versions in `.github/workflows/` and document required versions |
| JS/Go reference solutions become stale when benchmarks change | Med | CI verifies reference solutions execute correctly on every run |
| Codex CLI behavior differs from Claude/Gemini in unexpected ways (auth, rate limits) | Med | Mirror the health check pattern from `claude.go`; fail loudly on auth errors |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_7_0/M-EVAL-AGENT-QUEUE.md](../implemented/v0_7_0/M-EVAL-AGENT-QUEUE.md) — agent queue / orchestration patterns (0.43)
- [design_docs/implemented/v0_7_0/m-script-invoke.md](../implemented/v0_7_0/m-script-invoke.md) — script invocation patterns (0.42)

**Planned (check for overlap):**
- [design_docs/planned/v0_11_0/m-eval-cross-language-benchmark.md](v0_11_0/m-eval-cross-language-benchmark.md) — **complementary**, not overlapping: that doc is about *external* benchmarks (AutoCodeBench, leetgptsolver); this doc is about extending the *internal* eval bench.
- [design_docs/planned/v0_11_0/m-eval-category-analysis.md](v0_11_0/m-eval-category-analysis.md) — eval result categorization (0.49)
- [design_docs/planned/v0_11_0/m-cloud-eval-workers.md](v0_11_0/m-cloud-eval-workers.md) — distributed eval execution (0.44)

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) — Block-universe determinism
- [internal/executor/codex_compat_test.go](../../internal/executor/codex_compat_test.go) — Codex event format analysis (blueprint for M-EVAL-CODEX parser)
- [internal/executor/executor.go](../../internal/executor/executor.go) — Executor interface contract
- [internal/eval_harness/models.yml](../../internal/eval_harness/models.yml) — Model configuration (lines 479-490 for `ollama-gemma3`, 521-574 for suites)
- [ai-coding-lang-bench](https://github.com/mame/ai-coding-lang-bench) — External Ruby benchmark for comparison; already covers 15 languages with Claude only

## Future Work

- Port remaining 40+ benchmarks to JS + Go (follow-up sprint after M-EVAL-LANG-JSGO).
- Add TypeScript, Rust, Java via the registry (one file each).
- Add OpenAI Responses API as an executor option separate from Codex CLI.
- Build a `free_suite` and nightly CI job if Gemma 4 (or newer local models) prove reliable enough under repeated runs.
- Add additional open-source models to the registry once Gemma 4 is validated: DeepSeek Coder, Qwen Coder, Llama 3 variants — the path is one-line `models.yml` additions.
- Cross-harness comparison reports: "benchmark X on Claude vs Gemini vs Codex side-by-side."
- Integrate findings back into [M-EVAL-XLANG](v0_11_0/m-eval-cross-language-benchmark.md) — once we have internal JS/Go numbers, we can compare directly with AutoCodeBench results for the same languages.

---

**Document created**: 2026-04-19
**Last updated**: 2026-04-19

---
DESIGN_DOC_PATH: design_docs/planned/v0_13_0/m-eval-expand-harnesses-languages.md
