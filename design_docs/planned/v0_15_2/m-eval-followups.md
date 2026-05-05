# M-EVAL-FOLLOWUPS-V0151

**Status**: Planned
**Target**: v0.15.2
**Priority**: P1 (Medium-high — one harness bug, four high-leverage measurement/UX fixes)
**Estimated**: ~1.5 working days (~12 hours)
**Dependencies**:
- M-EVAL-COST-AND-SPEED-BUDGETS (v0.15.1, shipped) — supplies the speed/cost data the adjusted-rate headline consumes
- v0.15.1 baseline (shipped, 1256 runs / $75.26) — empirical data driving every fix in this milestone

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|------:|---------------|
| A1: Determinism | 0 | No new runtime semantics |
| A2: Replayability | +1 | Adjusted rates make replay analysis cleaner (separates infra noise from capability) |
| A3: Effect Legibility | +1 | M1 strengthens the "every effect needs an explicit cap" message at the harness layer |
| A4: Explicit Authority | +1 | M1 fixes a hidden authority leak (agent runs without granted caps) |
| A5: Bounded Verification | 0 | n/a |
| A6: Safe Concurrency | 0 | n/a |
| A7: Machines First | **+2** | Better stdlib + clearer prompt = more machine-decidable code from same model |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | M5 makes infra-vs-capability cost visible per model |
| A10: Composability | 0 | n/a |
| A11: Structured Failure | +1 | M5 separates infra failures from capability failures in the headline metric |
| A12: System Boundary | +1 | M1 reinforces capability-grant as the authority boundary |

**Net Score: +8** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): Caps remain explicit (M1 fixes a leak — strengthens, not weakens)
- [x] A4 (Authority): No ambient access granted (M1 audits and tightens)
- [x] A7 (Machines First): Improves machine-decidability via stdlib + prompt fixes

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

**This doc: +8, no hard violations → proceed.**

## Problem Statement

The v0.15.1 baseline retro (1256 runs across 14 models, $75.26 spend) surfaced five concrete eval-driven gaps. Each is **grounded in a specific failure pattern** in the dataset, NOT speculation:

**Current State:**

| # | Gap | Evidence | Failures |
|---|-----|----------|----------|
| 1 | **CAP_001 harness bug** — agent runs `ailang run` without `--caps IO` | 5/5 `prompt_injection` AILANG runs fail with `effect 'IO' requires capability, but none provided` | 5 |
| 2 | **Stdlib gap** — `std/string.chars` missing | 4× `undefined variable: chars` in compile errors across `run_length_encode`, `csv_to_json_converter`, `graph_bfs` | 4 |
| 3 | **Prompt gap** — Ord defaulting + `show` for int | `polymorphic_ord_defaulting` 58% AIL vs 92% Py (-33pp); `intToStr` invented 1× | ~5 lost passes |
| 4 | **Benchmark spec drains api_error** — `config_file_parser` and `log_file_analyzer` | 7/9 + 5/10 failures = `api_error` (drains opencode quota / multi-turn loop blowup) | 12 infra failures |
| 5 | **Headline metric noise** — 84/124 (68%) of AILANG failures are `api_error`, not capability | Raw pass rate 74% vs adjusted 85% | conceals real signal |

**Impact:**

- **Who is affected?** Eval suite curators (us), the AILANG cost story (every consumer of the leaderboard), AI agents trying to learn AILANG syntax
- **How significant?** All 5 fixes are concrete with measurable expected outcomes (e.g., "5/5 → 0/5 CAP_001"). Total ~10-15pp expected lift on AILANG agent-mode pass rate plus a fundamentally cleaner headline metric.

The gaps are independent — none of the 5 milestones depends on another. Total scope ~12 hours = a one-and-a-half-day sprint.

## Goals

**Primary Goal:** Land 5 evidence-grounded fixes from the v0.15.1 retro — one harness CAP_001 bug, one stdlib addition, one prompt update, one benchmark spec audit, one headline-metric refinement — so the next baseline measures AILANG capability cleanly without infra/prompt/stdlib noise polluting the signal.

**Success Metrics:**

- M1: `prompt_injection` AILANG CAP_001 failures drop from 5/5 to 0/5 in next baseline
- M2: `std/string.chars` exists with tests; `run_length_encode` and `csv_to_json_converter` no longer show `undefined chars` compile errors
- M3: `polymorphic_ord_defaulting` AILANG pass-rate improves ≥10pp; no `intToStr` failures in next baseline
- M4: `config_file_parser` and `log_file_analyzer` `api_error` rates drop ≥50% (e.g., 7/9 → 3/9)
- M5: Dashboard headline shows adjusted-rate; tooltip explains the distinction; Value Score formula uses adjusted-rate numerator

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| M1 fix location: agent-prompt vs auto-default-caps env var | Per-executor prompt tweak vs single-point env-var fix; affects all 5 executors | human (this doc) | design | med |
| M2: implement `chars` via existing `_str_splitAny("")` vs new `_str_chars` builtin | Pure-AILANG vs new builtin; affects future stdlib pattern | agent (impl-time) | compile | low |
| M5: which adjusted rate becomes "primary" — agent-only or combined? | Affects every dashboard read | human (this doc) | design | med |
| M3: which prompt version gets the update — v0.15.x active or v0.16.0 future? | v0.16.0 is already prepped but not yet active | human (this doc) | design | low |
| M4: spec audit fixes vs deferring benchmark cuts | Cutting benchmarks loses signal but saves money; spec fixes preserve signal | human (this doc) | design | low |

### Design Freeze

Resolved here:

- [x] **M1 fix location**: Update agent system prompt to ALWAYS run with `--caps IO,FS,Env`. Per-benchmark `spec.Caps` propagated as additional context. Plus an `AILANG_BENCHMARK_DEFAULT_CAPS` env var fallback for safety. (Belt-and-braces: prompt + env var.)
- [x] **M2 implementation**: Use `_str_splitAny(s, [""])` if it splits per char, else add `_str_chars` builtin. Agent decides at impl time based on existing builtin behavior.
- [x] **M5 primary metric**: Agent-mode adjusted rate (most users care about agent-mode capability). Standard-mode shown as secondary on per-model breakdown.
- [x] **M3 target prompt**: Update **both** `prompts/v0.15.1.md` (currently active) AND `prompts/v0.16.0.md` (forward-looking) to keep them in sync.
- [x] **M4 approach**: Spec fix only (don't cut benchmarks). Reduce expected_output strictness if test is overly brittle; tighten task_prompt if confusing.

All "med" change-cost decisions resolved. Sprint-executor may proceed without further human gates.

## Solution Design

### Overview

Five separable fixes, each independently verifiable. M1 + M2 + M3 directly improve next-baseline pass rates. M4 cleans up benchmark noise. M5 changes how we *display* the data (no language behavior change).

### Architecture

```
v0.15.1 baseline retro (input data)
         │
         ▼
   ┌─────────────────────────────────────────────────┐
   │  M1: Harness CAP fix (executor + prompt + env)  │  → eliminates 5 CAP_001 failures
   │  M2: std/string.chars (stdlib + recipe)         │  → eliminates 4 'chars' failures
   │  M3: prompt update (Ord + show + caps reminder) │  → ~5pp pass-rate lift on stretch
   │  M4: benchmark spec audit (2 yml files)          │  → -50% api_error in 2 benchmarks
   │  M5: adjusted-rate headline (dashboard React)    │  → cleaner display, no language change
   └─────────────────────────────────────────────────┘
         │
         ▼
   v0.15.2 baseline (cleaner signal)
```

### Implementation Plan

**M1: AI harness CAP_001 fix** (~2h, ~80 LOC) — **HIGHEST LEVERAGE**
- [ ] Audit `internal/executor/{claude,codex,gemini,opencode,pi}/*.go` for cap-passing in tool-call templates (each agent invokes `ailang run` differently)
- [ ] Update agent system prompts (claude_settings.json, codex.go template, gemini.go template, opencode wrapper, pi.go) to default to `--caps IO,FS,Env`
- [ ] Add `AILANG_BENCHMARK_DEFAULT_CAPS` env var fallback in `cmd/ailang/exec.go` so even if a prompt is missed, the harness sets a safe default
- [ ] Test: re-run `prompt_injection` on claude-sonnet-4-6 and gemini-3-flash; expect 5/5 → 0/5 CAP_001

**M2: stdlib gap — `std/string.chars`** (~3h, ~80 LOC + tests + recipe)
- [ ] Try `_str_splitAny(s, [""])` first — if it returns per-char, no new builtin needed
- [ ] If not, add `_str_chars` Go builtin in `internal/builtins/string.go`
- [ ] Add `chars(s: string) -> list[string]` to `std/string.ail`
- [ ] Update `prompts/{v0.15.1,v0.16.0}.md` "Common operations" cheat sheet
- [ ] Sync `prompts/versions.json` SHA256 hashes
- [ ] Add `examples/string_chars_demo.ail`
- [ ] Add recipe entry under `docs/docs/recipes/string-operations.md`
- [ ] Test: `run_length_encode` and `csv_to_json_converter` no longer fail on `undefined chars`

**M3: prompt update — Ord defaulting + `show` + caps reminder** (~3h, ~120 LOC docs)
- [ ] Add worked example to active prompt: `compare(x, y)` where `x, y: int` resolves to `Ord int` (explicit instance disambiguation)
- [ ] Promote `show(x: int) -> string` in cheat sheet; explicitly note `intToStr`/`toString` don't exist (preempts invention)
- [ ] Add 1-2 lines on capability declarations in main signatures (preempts CAP_001 from M1's territory at the agent-system-prompt level)
- [ ] Sync hashes in `prompts/versions.json`
- [ ] Test: re-run `polymorphic_ord_defaulting` (expect ≥10pp lift); zero `intToStr` failures

**M4: benchmark spec audit** (~2h, ~50 LOC YAML)
- [ ] Read `benchmarks/config_file_parser.yml` — compare structure to `api_call_json.yml` (100% pass)
- [ ] Read `benchmarks/log_file_analyzer.yml` — same comparison
- [ ] Identify difference: long task_prompt? Strict expected_output? Unusual capability requirement?
- [ ] Propose ONE concrete spec fix per benchmark (clarify task_prompt, relax expected_output to regex if appropriate, or split into smaller checks)
- [ ] Update `notes:` field documenting why the change was made
- [ ] Test: re-run both on opencode-sonnet-4-6 and claude-sonnet-4-6; expect ≥50% reduction in `api_error` rate

**M5: adjusted pass rate as headline** (~2h, ~150 LOC React)
- [ ] Update `BenchmarkDashboard/index.jsx` headline aggregates — primary number = adjusted; secondary = raw with "excludes provider failures" subtitle
- [ ] Update `ModelComparisonTable.jsx` % column to use `agentSuccessRateAdjusted` when available, fall back to raw
- [ ] Add tooltip explaining the distinction (already partially done in `Cell` component — verify)
- [ ] Update `ValueDashboard/index.jsx` Value Score formula numerator to use adjusted rate
- [ ] Test: visual check on `/docs/benchmarks/performance` — claude-sonnet-4-6 should show ~94% (adjusted) vs ~83% (raw)

### Files to Modify/Create

**New files:**
- `examples/string_chars_demo.ail` — runnable demo for M2 (~20 LOC)
- `docs/docs/recipes/string-operations.md` — recipe page for M2 (~40 LOC)

**Modified files:**

| File | Milestone | LOC |
|------|-----------|----:|
| `internal/executor/claude/claude_settings.json` (or runtime template) | M1 | ~10 |
| `internal/executor/{codex,gemini,opencode,pi}/*.go` agent prompt | M1 | ~30 |
| `cmd/ailang/exec.go` env-var fallback | M1 | ~30 |
| `std/string.ail` add `chars` | M2 | ~30 |
| `internal/builtins/string.go` (if needed) | M2 | ~50 |
| `prompts/v0.15.1.md` updates | M2+M3 | ~80 |
| `prompts/v0.16.0.md` updates | M2+M3 | ~60 |
| `prompts/versions.json` SHA hashes | M2+M3 | ~5 |
| `benchmarks/config_file_parser.yml` | M4 | ~10 |
| `benchmarks/log_file_analyzer.yml` | M4 | ~10 |
| `docs/src/components/BenchmarkDashboard/index.jsx` | M5 | ~80 |
| `docs/src/components/BenchmarkDashboard/ModelComparisonTable.jsx` | M5 | ~40 |
| `docs/src/components/ValueDashboard/index.jsx` | M5 | ~30 |

Total: ~515 LOC

## Examples

### Example 1: M1 — CAP_001 fix (before/after)

**Before** (v0.15.1, agent-mode `prompt_injection` solution.ail):
```bash
$ ailang run benchmark/solution.ail
Error: execution failed: effect 'IO' requires capability, but none provided
```

**After** (v0.15.2, agent default caps):
```bash
$ ailang run --caps IO,FS,Env benchmark/solution.ail   # auto-injected by agent
Hello, world!   # passes
```

### Example 2: M2 — `std/string.chars` (before/after)

**Before** — model writes:
```ailang
let cs = chars(s)   -- Error: undefined variable: chars
```

**After** — `chars` is in stdlib:
```ailang
import std/string (chars)
let cs = chars("hello")   -- ["h", "e", "l", "l", "o"]
```

### Example 3: M5 — Adjusted-rate headline (before/after)

**Before** (raw rate):
```
claude-sonnet-4-6   83%   34 runs, 5 api_errors
```

**After** (adjusted as primary):
```
claude-sonnet-4-6   94% (adjusted)   29 valid runs ⓘ
                    raw 83% — excludes 5 api_errors
```

## Success Criteria

- [ ] M1: prompt_injection AILANG CAP_001 failures drop from 5/5 to 0/5 in next baseline
- [ ] M2: `std/string.chars` exists with tests; `run_length_encode` and `csv_to_json_converter` no longer show `undefined chars` errors
- [ ] M3: polymorphic_ord_defaulting AILANG pass-rate improves ≥10pp; no `intToStr` failures in next baseline
- [ ] M4: `config_file_parser` and `log_file_analyzer` api_error rates drop ≥50%
- [ ] M5: Dashboard headline shows adjusted-rate; tooltip explains the distinction
- [ ] All `make test`, `make lint`, `make verify-examples` green
- [ ] CHANGELOG entry under v0.15.2 with milestone-by-milestone summary
- [ ] Empirical impact on next baseline: ~10-15pp lift on AILANG agent-mode pass rate

## Testing Strategy

**Unit tests:**
- M2: `internal/builtins/string_test.go` adds `TestChars*` cases
- M5: dashboard component tests verify adjusted vs raw rate display

**Integration tests:**
- M1: spawn opencode subprocess with a benchmark that uses `println`, verify caps are passed
- M4: run benchmark suite locally on a single model, confirm api_error count drops

**Manual testing:**
- Re-smoke `prompt_injection`, `polymorphic_ord_defaulting`, `run_length_encode`, `csv_to_json_converter` on claude-sonnet-4-6 (cheap, fast); verify expected fixes
- Visual check `/docs/benchmarks/performance` and `/docs/benchmarks/value` after M5

**Cost: M1+M2+M3+M4 verification ~$0.50 in eval spend (single model, ~10 benchmarks).**

## Deferred Decisions

The following are intentionally left open for the implementer:

- **M2 builtin path**: agent picks at impl time whether `_str_splitAny` already supports per-char split or a new `_str_chars` builtin is needed
- **M4 benchmark fix shape**: agent picks the most-targeted change (clarify prompt vs. relax output vs. tighten timeout) per benchmark after reading the YAMLs
- **M5 backward-compat**: agent decides whether to ship a "view raw rates" toggle for users who want the old display behaviour
- **M3 prompt order**: where exactly to insert the Ord/show/caps additions in the existing prompts — agent picks based on least-disruption to teaching flow

## Non-Goals

**Not attempted in this feature:**

- **Cost-budget tier-aware defaults** (smoke=$0.10, core=$0.50, stretch=$1.00) — bigger architectural change; deferred to v0.16.0 once we see distributions from v0.15.2 results
- **Standard-eval rows on speed chart** — could filter agent-only models, but that's a separate UX project; deferred until v0.16.0 dashboard polish
- **Investigating remaining `WRONG_LANG` failures** — needs a prompt-engineering experiment (separate sprint scope)
- **AILANG-side `!{AI[budget=$X]}` parameterised effect** — v1.0.0 axis (M-EFFECT-REFINEMENT Phase 6+)
- **Cutting any benchmarks** — M4 fixes specs only; benchmark cuts deserve their own curation discussion

## Timeline

**Day 1 (~8 hours):**
- M1: harness CAP fix (2h)
- M2: stdlib chars + tests + example (3h)
- M3: prompt update (3h)

**Day 2 (~4 hours):**
- M4: benchmark spec audit (2h)
- M5: adjusted-rate headline (2h)

**Total: ~12 hours across 1.5 working days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| M1 cap-default leaks authority (security regression) | Med | Default cap set is `IO,FS,Env` — common-but-bounded. Cap-required ones like `Net,AI` still need explicit grant. Document in M1 commit. |
| M2 `_str_splitAny("")` doesn't behave per-char | Low | Fall back to dedicated `_str_chars` builtin (~30 extra LOC). Pre-impl smoke check. |
| M3 prompt updates regress other benchmarks | Low | Prompts grow ~120 LOC — keep additions in clearly-marked sections; re-run cheap verification subset before commit. |
| M4 spec fix breaks an existing model that DID pass | Med | Re-run any model that previously scored 100% on the affected benchmark before commit; flag any regression. |
| M5 changes break user expectations of historic comparison | Low | Display BOTH adjusted and raw on detail rows; only the headline changes. Add prominent tooltip explanation. |

## Conflict Surface

**N/A — no parser/typechecker/codegen changes.**

This sprint touches:
- agent harness templates (Go strings)
- stdlib (additive `chars` function — new symbol, no shadowing)
- prompts (markdown additions)
- benchmark YAMLs (spec text)
- dashboard JSX (display logic)

No syntactic position is extended, no parser disambiguation rule changes, no AST node added. Conflict-surface analysis is only required for parser/typechecker/codegen work — this sprint is additive-stdlib + harness/UI plumbing only.

## Related Documents

<!-- Auto-populated by Ollama neural search on "eval followups" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_15_1/m-eval-cost-and-speed-budgets.md](../../implemented/v0_15_1/m-eval-cost-and-speed-budgets.md) — direct predecessor; introduced cost-budget infrastructure this milestone consumes
- [design_docs/implemented/v0_8_0/m-eval-loop.md](../../implemented/v0_8_0/m-eval-loop.md) — original eval-harness design; M1 audits its agent-tool-call layer

**Planned (check for overlap):**
- [design_docs/planned/v0_15_0/m-eval-trust-signals.md](../../planned/v0_15_0/m-eval-trust-signals.md) — adjacent eval-harness improvements
- v0.16.0 cost-budget tier-aware defaults — future work, gated on v0.15.2 data

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Cost-and-Speed Budgets guide](/docs/guides/evaluation/cost-and-speed-budgets) — predecessor work
- v0.15.1 baseline retro analysis (this session, captured in CHANGELOG entry)
- [.claude/skills/model-manager/SKILL.md](../../../.claude/skills/model-manager/SKILL.md) — Smoke-Test Gate methodology

## Future Work

- **Cost-budget tier-aware defaults** — adapt `max_cost_usd` per tier once v0.15.2 data shows distribution
- **Headline metric A/B test** — once adjusted rates are primary, validate that user understanding improves
- **`WRONG_LANG` prompt-engineering experiment** — concentrate examples for syntax-divergent features (typeclass, ADT, effects)
- **CI-fail on capability-leak regression** — once M1 lands, add a check that benchmarks declare caps and harness honors them

## Empirical Impact Estimates

If all 5 fixes land:

| Fix | Expected lift | Affected metric |
|-----|--------------|-----------------|
| M1 (CAP_001) | +5 of 5 prompt_injection runs | direct fail-recover |
| M2 (chars) | +4 of 4 'chars' failures | direct fail-recover |
| M3 (Ord/show/caps) | ≥+10pp on polymorphic_ord_defaulting; +1 intToStr | broad |
| M4 (api_error) | ≥+50% reduction in 2 benchmarks' api_error | infra hygiene |
| M5 (adjusted headline) | n/a (display only) | clarity |

**Aggregate**: ~10-15pp lift on AILANG agent-mode pass rate. Headline pass rate moves from ~74% (raw) to ~85% (adjusted). 3 currently-dominated models (gpt5-4-mini, gemini-3-flash, gemini-3-1-pro) might escape Pareto-domination once api_errors are excluded from cost-per-success.

This is a **polish + measurement-fix sprint**, not a feature sprint — but the data quality lift is foundational for v0.16.0 cost-and-speed-budgets follow-up planning.

---

**Document created**: 2026-05-05
**Last updated**: 2026-05-05
