# Eval Harness: Batch API Support for Standard-Mode Evals

**Status**: Planned
**Target**: Unversioned (opportunistic — not tied to a release)
**Priority**: P2 (Medium-low — cost optimization, not blocking)
**Estimated**: 4 days (~32h, 2x-buffered)
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is eval-harness infrastructure (Go tooling), not an AILANG language feature — most
axioms genuinely don't apply and are scored 0 rather than inflated.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|----------------|
| A1: Determinism | 0 | No impact on AILANG program execution or traces |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No AILANG effect changes |
| A4: Explicit Authority | 0 | Uses the same API keys/credentials already used for synchronous eval calls; no new authority granted |
| A5: Bounded Verification | 0 | No type-checking or verification impact |
| A6: Safe Concurrency | 0 | Batch submission is provider-side async, not new AILANG concurrency |
| A7: Machines First | 0 | Internal tooling change, not an AI-facing language surface |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | The whole point: banked cost fields now reflect batch vs. standard pricing explicitly (tagged `billing_mode`), instead of a single implicit rate |
| A10: Composability | +1 | New batch path composes alongside the existing synchronous `generate()` path in `ai_provider.go` without replacing it — opt-in, backward compatible |
| A11: Structured Failure | +1 | Batch-specific outcomes (`errored`/`canceled`/`expired`) get their own `ErrorCategory` constants in `error_categorizer.go`, instead of falling through to the generic `ErrorCategoryAPI` catch-all |
| A12: System Boundary | 0 | Same external-provider boundary as the existing synchronous calls; only the request/response shape changes |

**Net Score: +3** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

AILANG's eval harness makes standard-mode (non-agentic, single-shot) LLM calls at full
synchronous API price, even though nearly all of that traffic tolerates a delayed result.
Both Anthropic and OpenAI offer a standing 50% discount for exactly this workload shape
via their Batch APIs, and the harness isn't using it.

**Current State:**
- Standard-mode calls are genuinely one-shot and stateless: `internal/eval_harness/ai_provider.go:171` (`generate()`) builds one `ai.Request` and makes one blocking `p.provider.Generate(ctx, req)` call — no tool loop, no dependency on prior turns. `cmd/ailang/eval_benchmark.go:19` (`runSingleBenchmark`) routes here whenever `agentConfig == nil` (verified: the function's agent branch at line ~83 always returns early via `runSingleBenchmarkAgent`, falling through to the standard path below it otherwise).
- `internal/eval_harness/repair.go:47` (`RepairRunner.Run`) adds at most one more independent call — a first attempt, then a conditional repair retry only if `CategorizeErrorWithCode` (repair.go, using the `ErrorCategory*` constants in `internal/eval_harness/error_categorizer.go`) flags it as repairable.
- A full release baseline (`core`+`stretch`+`frontier` tiers × 18 extended-suite models × 2 languages) runs ~2,000 standard-mode calls costing **$98–135 per release** (`.claude/skills/post-release/scripts/run_eval_baseline.sh:487`). Releases (v0.30.0 → v0.31.0 → v0.32.0) recur every few weeks, so this spend compounds.
- Post-release evals already run decoupled from the release tag and don't block anything — a batch API's slower turnaround (usually <1h, worst case 24h) costs nothing in practice here.
- 16 Anthropic and 21 OpenAI models are in the standard-mode provider mix (`internal/eval_harness/ai_provider.go:49`, `models.yml`) — all batch-eligible today, at zero code-path cost to the 38 OpenRouter models (confirmed no batch API — pure sync proxy) or the 22 local Ollama models (free, irrelevant).

**Impact:**
- Affects whoever runs/pays for release-baseline and ad-hoc standard-mode eval sweeps. Not a blocker — evals run fine today — but a recurring, avoidable cost on the Anthropic+OpenAI slice of every release.

## Goals

**Primary Goal:** Cut the Anthropic+OpenAI portion of standard-mode eval spend by ~50% by routing those calls through each provider's Batch API instead of the synchronous API, with zero change to agent-mode evals or to OpenRouter/Ollama-routed models.

**Success Metrics:**
- Anthropic+OpenAI standard-mode spend on a batch-mode release run measured at ~50% of the equivalent synchronous run's cost
- Agent-mode eval code path (`runSingleBenchmarkAgent`, `executor.Executor` implementations) untouched — zero diff in behavior or cost
- OpenRouter/Ollama models in the same eval run continue through the existing synchronous path unchanged
- A batch round (submit → poll → bank) completes without operator intervention within the documented worst-case window (24h)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Batch mode is opt-in per eval run (flag/config), never the default | A silently-defaulted batch mode could turn an expected-fast interactive spot-check into an hours-long wait | human | design | low |
| `custom_id` encoding: `{benchmark_id}__{model_id}__{lang}__attempt{N}` | Every result-matching, retry, and repair-dispatch step depends on this being stable and collision-free across ~2,000 requests; must respect the `^[a-zA-Z0-9_-]{1,64}$` constraint both providers enforce | agent | design | med |
| Repair runs as an independent second batch round, never folded into round 1 | Repair depends on seeing round 1's result, which isn't known until the batch ends — the architecture must support two sequential batch submissions, not one | human | design | high |
| New `ErrorCategory` constants for batch outcomes (`errored`/`canceled`/`expired`) added to `error_categorizer.go`, never defaulted to `ErrorCategoryAPI` | `ErrorCategoryAPI` is documented as "cause unknown," not "model failed" (CLAUDE.md, and the prior `ailang fmt` incident where this exact catch-all masked a real cause for weeks) — lumping batch-specific outcomes into it would recreate that failure mode for every expired/canceled request | human | design | med |
| Google/Vertex batch support is out of scope until independently verified | This doc found no confirmed evidence either way for Vertex — must not assume it exists just because Anthropic/OpenAI do | human | design | low |
| Provider capability split: Anthropic + OpenAI get a batch path; OpenRouter (confirmed no batch API) and Ollama (local, no billing) stay synchronous | Determines the shape of the provider abstraction — must not force a batch-shaped interface onto providers that structurally can't implement it | agent | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Repair is a second batch round, not folded into round 1 (blocks Phase 3)
- [ ] New `ErrorCategory` values for batch outcomes named and reviewed — not defaulting to `ErrorCategoryAPI` (blocks Phase 3)

## Solution Design

### Overview

Add a batch-mode code path alongside the existing synchronous `generate()` path in
`ai_provider.go`, gated to the two providers that support it (Anthropic, OpenAI). When
enabled, standard-mode benchmark×model×lang jobs for those providers are collected into
one or more provider-native batch submissions instead of being dispatched as individual
blocking calls. The harness submits, polls to completion — reusing the submit-then-poll
shape already used for coordinator-dispatched jobs in `eval_parallel.go:188`
(`runBenchmarksViaQueue`, 5s poll interval / 2h max wait there; batch needs a longer
ceiling given the provider's own 24h window) — then demultiplexes results by `custom_id`
back onto the original benchmark/model/lang identity. Repair retries run as an
independent second batch round, submitted only for benchmarks whose round-1 result
needed repair, after round 1 fully resolves. OpenRouter and Ollama models are entirely
unaffected: they continue through the existing synchronous `generate()` path, selected
per-model via a capability check in the provider abstraction.

### Architecture

**Components:**
1. **BatchProvider interface** — a capability check (`SupportsBatch() bool`) plus
   `SubmitBatch`/`PollBatch`/`FetchResults` methods, implemented only for the Anthropic
   and OpenAI providers in `internal/eval_harness/ai_provider.go`. OpenRouter, Ollama,
   and Google providers report `SupportsBatch() == false` and are routed to the
   existing synchronous path unchanged.
2. **Batch dispatcher** (new) — groups pending standard-mode jobs by provider, builds
   the `custom_id`-tagged request list respecting each provider's per-batch limits
   (Anthropic: 100,000 requests or 256MB; OpenAI: 50,000 requests or 200MB, whichever
   is hit first), submits, and drives the poll loop.
3. **Result demultiplexer** — parses `custom_id` back into `(benchmark_id, model_id,
   lang, attempt)`, maps each provider outcome type (`succeeded`/`errored`/`canceled`/
   `expired`) onto a new `ErrorCategory` constant rather than the generic
   `ErrorCategoryAPI` fallback, and feeds results into the existing banking/scoring
   pipeline unchanged.
4. **Repair round controller** — after round 1 fully resolves, uses the same
   repair-needed logic `RepairRunner.Run` already applies (`CategorizeErrorWithCode`)
   to select which benchmarks need a round 2, builds and submits a second batch
   containing only those, and merges results the same way.

### Implementation Plan

**Phase 1: Anthropic batch client** (~8h)
- [ ] Wrap `client.messages.batches.create/retrieve/results` behind the `BatchProvider` interface for the Anthropic provider
- [ ] `custom_id` encode/decode helpers, enforcing the `^[a-zA-Z0-9_-]{1,64}$` constraint
- [ ] Unit tests: submit/poll/results round trip against a mocked batch endpoint, including all four outcome types

**Phase 2: OpenAI batch client** (~4h)
- [ ] Files-API upload + batch create/retrieve/download wrapped behind the same `BatchProvider` interface
- [ ] Reuse the `custom_id` helpers from Phase 1 (format is provider-agnostic)

**Phase 3: Harness integration** (~12h)
- [ ] Batch dispatcher: group standard-mode jobs by provider, split into multiple batches when a provider's request-count/size limit is exceeded
- [ ] Poll loop (mirrors `eval_parallel.go:188`'s shape; interval/ceiling tuned for batch's up-to-24h window — see Deferred Decisions)
- [ ] Result demultiplexer: map `succeeded`/`errored`/`canceled`/`expired` onto new `ErrorCategory` constants in `error_categorizer.go`
- [ ] Repair round controller: second batch submission for round-1 failures only, using the existing `CategorizeErrorWithCode` repair-needed check

**Phase 4: Wiring, docs, testing** (~8h)
- [ ] CLI/config flag to opt into batch mode for a given eval run (naming left to implementer — see Deferred Decisions)
- [ ] End-to-end test: a handful of cheap benchmarks × 2 models (one Anthropic, one OpenAI), verifying banked cost matches the 50% discount and results match an equivalent synchronous run
- [ ] Update `docs/docs/guides/evaluation.md` and `CHANGELOG.md`

### Files to Modify/Create

**New files:**
- `internal/eval_harness/batch_provider.go` (~200 LOC) — `BatchProvider` interface + Anthropic/OpenAI implementations
- `internal/eval_harness/batch_dispatcher.go` (~250 LOC) — grouping, submission, poll loop, result demux
- `internal/eval_harness/batch_provider_test.go` (~150 LOC)
- `internal/eval_harness/batch_dispatcher_test.go` (~150 LOC)

**Modified files:**
- `internal/eval_harness/ai_provider.go` (+30/-5 LOC) — `SupportsBatch()` capability check, route to batch dispatcher when enabled
- `internal/eval_harness/error_categorizer.go` (+15 LOC) — new `ErrorCategoryBatchExpired`/`ErrorCategoryBatchCanceled`/etc. constants
- `internal/eval_harness/repair.go` (+20 LOC) — expose the existing repair-needed predicate for reuse by the round-2 batch controller without re-running round 1
- `cmd/ailang/eval_benchmark.go` (+15 LOC) — CLI flag wiring
- `docs/docs/guides/evaluation.md` (+~40 lines) — document batch mode, its cost/latency tradeoff, and its Anthropic/OpenAI-only scope

## Examples

### Example 1: Release baseline run

**Before:**
```bash
ailang eval-baseline --tier core,stretch,frontier
# ~2,000 synchronous calls, full price, results as each completes
```

**After:**
```bash
ailang eval-baseline --tier core,stretch,frontier --batch
# Anthropic + OpenAI jobs grouped into provider batches, submitted, harness polls
# (usually <1h, worst case 24h); banked at 50% of standard cost.
# OpenRouter/Ollama models in the same run proceed synchronously as before —
# unaffected by the --batch flag.
```

### Example 2: `custom_id` round trip

```
custom_id: "list_sort__claude-opus-5__ailang__attempt1"
  → decodes to: benchmark_id=list_sort, model_id=claude-opus-5, lang=ailang, attempt=1

Batch result (succeeded) → banked normally via existing scoring pipeline
Batch result (expired)   → banked with ErrorCategoryBatchExpired, NOT ErrorCategoryAPI
                          → surfaced distinctly so a re-run targets only expired jobs
```

## Success Criteria

- [ ] Anthropic + OpenAI standard-mode benchmarks run via `--batch` and bank results identical in shape to a synchronous run
- [ ] Banked cost for a batch run reflects the provider's 50% discount (verified against a real small-scale run)
- [ ] OpenRouter/Ollama models in the same eval run are unaffected — no code-path change for those providers
- [ ] Repair round only re-submits benchmarks that needed repair after round 1
- [ ] Batch outcome types (`errored`/`canceled`/`expired`) are banked with dedicated `ErrorCategory` values, never `ErrorCategoryAPI`
- [ ] All tests passing
- [ ] `docs/docs/guides/evaluation.md` updated
- [ ] `CHANGELOG.md` updated

## Testing Strategy

**Unit tests:**
- `custom_id` encode/decode round trip, including rejection of invalid characters
- `BatchProvider` mocked submit/poll/results for both Anthropic and OpenAI
- Result demultiplexer mapping for all four outcome types
- Batch-splitting logic when request count/size exceeds a provider's per-batch limit

**Integration tests:**
- Small real batch run (a handful of cheap benchmarks × Haiku-tier + gpt-mini-tier model) verifying end-to-end submit → poll → bank
- A repair-triggering scenario (a benchmark seeded to fail round 1) verifying round 2 fires only for that failure

**Manual testing:**
- Run a real full-tier batch alongside a synchronous run of the same tier; diff results and compare banked cost

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact batching granularity (one batch per model vs. one batch spanning all models for a given language, subject to the 100k-request/256MB Anthropic and 50k-request/200MB OpenAI limits) — agent may choose
- Poll interval and max-wait tuning for the batch-status loop (mirroring `eval_parallel.go`'s 5s/2h shape, adjusted upward for batch's up-to-24h ceiling) — agent may choose
- Whether to use `cache_control` within Anthropic batch requests for large shared system/teaching prompts (supported, best-effort 30–98% hit rate) — agent may choose based on cost/complexity tradeoff
- CLI flag naming and config surface (e.g., `--batch` flag vs. a `models.yml` per-provider setting) — agent may choose, consistent with existing eval CLI conventions

## Non-Goals

**Not attempted in this feature:**
- Agent-mode evals — structurally incompatible (live, multi-turn tool-use loop that a 24h-latency queue can't serve mid-turn); not attempted here
- Google/Vertex batch support — unverified whether it exists; would need its own research spike before a future doc
- DeepSeek cost optimization — DeepSeek has no batch API; it instead has time-of-day pricing (2x during Beijing peak hours). That's a scheduling lever, not an API integration, and is a candidate for a separate future design doc, not this one
- OpenRouter batch support — confirmed not to exist (pure synchronous proxy); no future work item
- Making batch mode the default for standard evals — stays opt-in given the latency tradeoff

## Timeline

**Week 1** (~16h):
- Phase 1: Anthropic batch client (8h)
- Phase 2: OpenAI batch client (4h)
- Phase 3 start: batch dispatcher scaffolding (4h)

**Week 2** (~16h):
- Phase 3 finish: poll loop, result demux, repair round controller (8h)
- Phase 4: CLI wiring, docs, end-to-end testing (8h)

**Total: ~32 hours across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Batch outcomes get lumped into `ErrorCategoryAPI`, recreating the exact "catch-all masks real cause" failure mode this project already hit once (`ailang fmt`, prior incident) | High | Design Freeze item requires named `ErrorCategory` values for `errored`/`canceled`/`expired` before Phase 3 starts |
| Under high provider demand, batches run closer to the 24h ceiling and unstarted requests expire (uncharged, but dropped) | Medium | Track `request_counts.expired` explicitly; re-submit expired requests as a follow-up batch rather than treating expiry as "benchmark failed" |
| Repair's two-round design roughly doubles worst-case turnaround for any benchmark needing repair | Low | Acceptable — post-release evals are already async/non-blocking; document the expected worst-case latency in `docs/docs/guides/evaluation.md` |
| `custom_id` collisions if benchmark/model/lang naming isn't unique-safe | Medium | Encode with the enforced regex; assert uniqueness across the full request list before submission |
| Google/Vertex assumed batch-capable without verification | Medium | Explicitly excluded (Non-Goals) until independently confirmed |

## Related Documents

<!-- Auto-populated by Ollama neural search on "eval batch api"; the two closest planned
     matches were checked by hand and are unrelated (below the skill's 0.45 warn threshold):
     m-arch-boundaries-eval-exclusion-tighten.md is about import-boundary lint scoping for
     internal/eval, and m-eval-results-folder-structure.md is about on-disk results layout —
     neither touches LLM API call shape or cost. -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_5_10/m-codegen-list-sprint-plan.md](../implemented/v0_5_10/m-codegen-list-sprint-plan.md) (weak match, 1.00 SimHash but not topically relevant)

**Planned (checked, no overlap):**
- [design_docs/planned/v0_30_0/m-arch-boundaries-eval-exclusion-tighten.md](v0_30_0/m-arch-boundaries-eval-exclusion-tighten.md) (0.43) — import-boundary scoping, unrelated
- [design_docs/planned/v0_29_0/m-eval-results-folder-structure.md](v0_29_0/m-eval-results-folder-structure.md) (0.42) — results file layout, unrelated

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- Anthropic Message Batches API — `POST /v1/messages/batches`, 50% discount, 24h ceiling, `custom_id`-keyed `.jsonl` results
- OpenAI Batch API — Files-API-backed `.jsonl` batches, 50% discount, 24h completion window
- `internal/eval_harness/ai_provider.go:171` (`generate()`), `internal/eval_harness/repair.go:47` (`RepairRunner.Run`), `internal/eval_harness/error_categorizer.go`, `cmd/ailang/eval_parallel.go:188` (`runBenchmarksViaQueue`) — all verified against the current codebase while authoring this doc

## Future Work

- Google/Vertex batch support, if independently confirmed to exist
- DeepSeek off-peak-hour scheduling as a separate cost lever (different mechanism — time-of-day pricing, not a batch endpoint)
- Extending batch mode to cover the repair-retry round with a single combined submission, if provider APIs ever support conditional/dependent requests within one batch

---

**Document created**: 2026-08-03
**Last updated**: 2026-08-03
