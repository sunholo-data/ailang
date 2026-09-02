# M-ANTHROPIC-CACHE-HIT-RATE: Close the prompt-cache gap on the `Generate` path

**Status**: Planned
**Target**: v0.31.0
**Priority**: P1 (Medium — a standing ~2-6× input-cost multiplier on every 0-shot eval and classifier call; not a correctness bug, but Anthropic has flagged our org cache rate as very low)
**Estimated**: ~8 hours (~320 LOC across `internal/ai/`, `internal/eval_harness/`, `cmd/ailang/`)
**Dependencies**: None blocking. The [M-EVAL-TOKEN-HEADROOM](m-eval-token-headroom.md) sequencing constraint is **already satisfied** — its `FinishReason: mapStopReason(result.StopReason)` line has landed in `Generate` (verified at `internal/ai/anthropic/client.go:399`, 2026-07-29), so this work builds on top of it rather than racing it. Directly extends [M-AI-PROMPT-CACHING (v0.18.4)](../../implemented/v0_18_4/m-ai-prompt-caching.md) — this is that doc's deferred "Phase 2 — Wider Anthropic placement", plus the `Generate`-path gap it never covered.
**Author**: Claude Opus 5 + Mark
**Created**: 2026-07-29

---

## Honest framing — what this fixes vs what it doesn't

Anthropic reported that our organization's prompt-cache hit rate is very low. Investigation
found the cause is **structural, not configuration**: the code path that carries essentially all
of our Anthropic API traffic cannot emit `cache_control` at all.

Three distinct things need separating:

**(A) In scope — the `Generate` path has no cache support whatsoever.**
[`internal/ai/anthropic/client.go:221`](../../../internal/ai/anthropic/client.go#L221) assigns
`apiReq.System = req.SystemPrompt` as a bare JSON string, and the file contains **zero**
references to `CacheBreakpoint` (V1 below). M-AI-PROMPT-CACHING wired caching into `Step` only.
Everything that actually spends money goes through `Generate`: standard-mode evals, the
feedback-gate classifier, and AILANG's `ai.generate`. All of it is uncached by construction.

**(B) In scope — the largest reusable block sits where no breakpoint can reach it.**
Standard-mode evals concatenate a ~16K-token teaching prompt with the per-benchmark task into a
single user string ([`spec.go:274-276`](../../../internal/eval_harness/spec.go#L274)), behind a
**70-token** system prompt (V8). Anthropic's minimum cacheable prefix is 1024 tokens on Sonnet
4.6/5 and Opus 4.8 (512 on Opus 5, 4096 on Haiku 4.5), so even a working system breakpoint would
cache nothing and report `cache_creation_input_tokens: 0` — silently, with no error.

**(C) NOT in scope — agent-mode evals.** Those shell out to the `claude` CLI, which manages its
own caching internally. Our only lever there is session reuse, which is a separate concern and
is deliberately excluded (see Non-Goals). This doc is about API-key traffic.

**What this does not claim:** we could not measure our current hit rate locally — every file
under `eval_results/` carries zero cache-token fields (V5) and the local observatory has 0 spans.
The Anthropic Console usage breakdown is the authority on the size of the prize; this doc fixes
a gap that is real regardless of that number, and Phase 3 makes it measurable going forward.

---

## Verification Log

Every claim below was checked against the code at `588ffcb67`. Negative-existence claims
(the load-bearing ones) each carry their own row.

| # | Claim | Verification | Result |
|---|-------|--------------|--------|
| V1 | `Generate` never emits `cache_control`; `CacheBreakpoints` is silently dropped on that path | `grep -c "CacheBreakpoint" internal/ai/anthropic/client.go` | **0** — confirmed. `System` set as bare string at `client.go:221` |
| V2 | No Go-native caller ever populates `Request.CacheBreakpoints` | `grep -rn "CacheBreakpoints:" --include="*.go" internal/ cmd/ \| grep -v _test` | Only `handler.go:385,406` — pure pass-through. Sole origin is `internal/effects/ai_step.go:296` (the AILANG `stepWithCache` builtin) and the WASM JS handler. Confirmed: no Go caller |
| V3 | Only the 5-minute `ephemeral` tier is emitted; no 1h tier | `grep -n '"ephemeral"\|"1h"\|ttl' internal/ai/anthropic/cache.go` | Only `"ephemeral"` literal at `cache.go:80`. Confirmed |
| V4 | `last_user` and `tool_result` breakpoint positions are unimplemented | `grep -rn "last_user" --include="*.go" internal/ai/anthropic/` | No hits. `tool_result` appears only as a *message content block type* (`step.go:272,322`), never as a cache position. Confirmed |
| V5 | No banked eval result carries cache-token fields | `grep -rl "cache_read_input_tokens\|cacheReadInputTokens" eval_results/ \| wc -l` | **0** of 28,139 JSON files. Structs exist (`agent_runner.go:176`, `provider.go:302`) but nothing serializes them into summaries. Confirmed |
| V6 | Eval teaching prompt is concatenated into the user message, not the system prompt | Read `internal/eval_harness/spec.go:274-276` | `fullPrompt = basePrompt + "\n\n## Task\n\n" + taskDescription`, passed as the single `prompt` arg to `GenerateCode` → `Request.UserPrompt` (`ai_provider.go:129`). Confirmed |
| V7 | Eval suite fans out concurrently by default | `grep -n 'parallel"' cmd/ailang/eval_suite.go` | `fs.Int("parallel", 10, ...)` — 10 concurrent calls default. Confirmed |
| V8 | The eval system prompt is far below the minimum cacheable prefix | Measured the literal at `ai_provider.go:116-119` | 283 chars ≈ **70 tokens** vs a 1024-token minimum. Confirmed |
| V9 | The teaching prompt is large enough to be worth caching on every model tier | `ls -l prompts/v0.9.0.md` | 63,939 bytes ≈ **~16K tokens** — clears the 4096-token Haiku 4.5 minimum by 4×. Confirmed |

**Not verified (stated as unknown, not as fact):** our actual current org-wide cache hit rate,
and the split of Anthropic spend between API-key traffic and subscription/CLI traffic. Both live
in the Anthropic Console / prod observatory, not in this repo.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Cache markers are advisory and do not change produced bytes. The one behavior-adjacent change (splitting the user message into two content blocks) is deterministic and versioned; the *behavior-changing* system-role move is explicitly deferred, not bundled |
| A2: Replayability | 0 | Cache state is not part of the trace; replay unaffected |
| A3: Effect Legibility | 0 | No new effect. Rides the existing `! {AI}` effect exactly as M-AI-PROMPT-CACHING established |
| A4: Explicit Authority | 0 | No new capabilities; the AI cap remains the gate |
| A5: Bounded Verification | 0 | Additive optional fields with zero-value defaults; type-checking unaffected |
| A6: Safe Concurrency | +1 | Makes an existing concurrency hazard explicit and fixes it: the current parallel fan-out guarantees every concurrent request misses (an entry is readable only once the first response begins streaming). The warm-up turns an accidental worst case into a declared one |
| A7: Machines First | +2 | Directly reduces input-token cost — the dominant cost an agent pays. This is the axiom the whole milestone serves |
| A8: Minimal Syntax | 0 | No new AILANG syntax. One additive `Request` field; the existing `CacheBreakpoint` vocabulary gains one position value |
| A9: Cost Visibility | +2 | Phase 3 is the point: today cache tokens are *modelled* (`provider.go:302-311`) but never *persisted* (V5), so we cannot see our own cache behavior. This makes hit rate a first-class banked metric |
| A10: Composability | +1 | The `CachedPrefix` field is provider-neutral: Anthropic stamps it, every other provider concatenates it back and stays byte-identical |
| A11: Structured Failure | +1 | Extends the existing one-shot `cache_hint_ignored_*` warning (`cache_warnings.go:46`) to the `Generate` path, and adds a loud below-minimum-prefix warning for the silent-no-op case that bit us in (B) |
| A12: System Boundary | +1 | Keeps the "AILANG-side hint → provider wire shape" mapping in one place per provider, as v0.18.4 established, rather than leaking cache concerns into the eval harness |

**Net Score: +8** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No new nondeterminism. The behavior-changing prompt-placement question is split out as a Deferred Decision precisely so it can be A/B'd on its own merits
- [x] A3 (Effects): No hidden side effects; cache hints flow through the existing `! {AI}` effect
- [x] A4 (Authority): No ambient access; no new capabilities
- [x] A7 (Machines First): The milestone exists to cut machine-side token cost

---

## Problem Statement

Anthropic flagged our organization's prompt-cache hit rate as very low. It is very low because
our highest-volume Anthropic code path is physically incapable of requesting a cache.

**Current State:**

- **The `Generate` path cannot cache at all.** `internal/ai/anthropic/client.go` contains zero
  references to `CacheBreakpoint` (V1). M-AI-PROMPT-CACHING wired `Step` only. `Generate` carries
  standard-mode evals ([`ai_provider.go:126`](../../../internal/eval_harness/ai_provider.go#L126)),
  the feedback-gate classifier ([`classifier.go:149`](../../../internal/feedbackgate/classifier.go#L149)),
  and AILANG `ai.generate`.
- **No Go caller opts in even where it could.** The only origin of `CacheBreakpoints` is the
  AILANG `stepWithCache` builtin (V2) — i.e. user-written `.ail` code. Our own Go services never
  ask for caching.
- **The cacheable content is in the uncacheable slot.** ~16K tokens of teaching prompt sit inside
  the user message (V6, V9) behind a 70-token system prompt (V8). The only implemented breakpoint
  position is `system` (V4), which here is below Anthropic's minimum prefix and would cache
  nothing — silently.
- **Every parallel request misses.** `--parallel 10` by default (V7); a cache entry only becomes
  readable once the first response begins streaming, so all 10 concurrent calls pay a full write.
- **We cannot see any of this.** Zero banked eval results carry cache-token fields (V5).

**Impact:**

- Every 0-shot eval re-pays ~16K uncached input tokens. On a 30-benchmark suite against one
  model that is ~480K input tokens where ~66K would do.
- The feedback-gate classifier re-pays its full system prompt on every classification.
- Cost regressions in this area are invisible to us — we would not notice a hit-rate collapse,
  which is exactly why the signal arrived from Anthropic rather than from our own dashboard.

---

## Goals

**Primary Goal:** Make our Anthropic API traffic cache-capable end to end — the `Generate` path
can emit `cache_control`, the breakpoint can be placed where the reusable content actually is,
and the resulting hit rate is a banked, visible metric.

**Success Metrics:**

- `cache_read_input_tokens > 0` in banked eval summaries for every Anthropic standard-mode run
  (today: literally zero across 28,139 files — V5)
- ≥ 70% cache-read share of shared-prefix input tokens on a repeat 30-benchmark suite
- ≥ 80% reduction in shared-prefix input tokens on a warm suite vs. the v0.30.0 baseline
- Zero wire-shape change for any request that declares no breakpoints (byte-identical, per the
  guarantee M-AI-PROMPT-CACHING made and this doc preserves)
- A loud warning whenever a declared breakpoint yields a below-minimum prefix — no more silent
  no-ops

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1: Cache the teaching prompt in-place via a user-message split, rather than moving it to the system role** | Moving it changes model inputs and therefore eval outputs, invalidating baseline comparability. The split keeps content semantically identical and lands the cost win without a re-baseline | human | design | **high** |
| **D2: Add `Request.CachedPrefix` as the provider-neutral carrier** (vs. overloading `UserPrompt` with markers, or an Anthropic-only field) | Determines whether every other provider stays byte-identical. A provider-specific field would leak Anthropic semantics into shared code and break A10/A12 | human | design | **high** |
| **D3: Default caching ON for Anthropic in the eval harness and feedback gate** (vs. opt-in flag) | Opt-in is why we are here — V2 shows nobody opts in. But ON-by-default changes cost accounting for every run | human | design | med |
| **D4: Warm-up request before the parallel fan-out** | Difference between a 52% and an 86% saving (see projection). Adds one serial round-trip to every suite | agent | compile | low |
| **D5: 5-minute `ephemeral` vs. 1-hour TTL default** | 1h costs 2× on write vs 1.25×, and needs ≥3 reads to break even. Right answer differs for a dense suite vs. a day-long rotation | agent | runtime | low |
| **D6: Persist cache tokens into the eval summary schema** | Schema change to banked results; downstream dashboard/report readers must tolerate the new fields | agent | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **D1** — **RESOLVED 2026-07-29 (Mark): cache in place.** The teaching prompt stays in the
      user message and is split into two content blocks with `cache_control` at the
      `\n\n## Task\n\n` boundary. Full cache win, model sees identical content, v0.30.0 eval
      baselines stay comparable. The system-role move remains deferred for its own A/B — it is a
      quality change, not a cost change, and bundling them makes both unmeasurable.
- [x] **D2** — **RESOLVED 2026-07-29 (agent latitude, per recommendation): `Request.CachedPrefix`
      is the carrier.** Provider-neutral: Anthropic stamps it as a cached leading block; every
      other provider concatenates `CachedPrefix + UserPrompt` and stays byte-identical to
      v0.30.0.
- [x] **D3** — **RESOLVED 2026-07-29 (Mark): ON by default** for the eval harness and feedback
      gate on Anthropic. Opt-in is precisely why the hit rate is near zero today (V2 — no Go
      caller ever opts in). Guard: skip the breakpoint when a run makes fewer than 2 calls per
      model, so short runs never pay the 1.25× write premium for a cache that is never read.

---

## Solution Design

### Overview

Four changes, ordered so the pure-cost fix lands and can be measured before anything
behavior-adjacent is considered:

1. Teach the `Generate` path to emit `cache_control` (it currently cannot).
2. Give callers a provider-neutral way to say "this leading chunk is stable" —
   `Request.CachedPrefix` — so the breakpoint can sit where the 16K teaching prompt actually is.
3. Turn it on by default for our own Anthropic callers, with a warm-up before the fan-out.
4. Persist cache tokens so the hit rate becomes a metric we own.

### Architecture

**Components:**

1. **`Request.CachedPrefix string`** (`internal/ai/provider.go`) — content that logically precedes
   `UserPrompt`. Empty = today's behavior exactly. The contract is that
   `CachedPrefix + UserPrompt` is what the model sees, regardless of provider; only the *encoding*
   differs.

2. **Anthropic encoding** (`internal/ai/anthropic/cache.go`, `client.go`) — when `CachedPrefix` is
   non-empty and a `user_prefix` breakpoint is declared, the user message becomes a two-block
   content array:
   ```json
   [{"type":"text","text":"<16K teaching prompt>","cache_control":{"type":"ephemeral"}},
    {"type":"text","text":"\n\n## Task\n\n<benchmark task>"}]
   ```
   The system field keeps its existing `systemFieldFromPrompt` treatment. `Generate` gains the
   same `systemFieldFromPrompt` call `Step` already has, closing V1.

3. **Every other provider** — concatenates `CachedPrefix + UserPrompt` into the single user string
   they send today. Wire shape byte-identical to v0.30.0. OpenAI keeps auto-caching ≥1024 tokens;
   Gemini/Ollama keep the existing one-shot `cache_hint_ignored_*` warning.

4. **Below-minimum guard** (`internal/ai/anthropic/cache.go`) — if a declared breakpoint's prefix
   is under the model's minimum cacheable size, emit a one-shot structured warning naming the
   model and the shortfall. This is the failure mode that made (B) invisible; it must be loud
   (Principle 2 — no silent fallbacks).

5. **Cache-token persistence** (`internal/eval_harness/`) — thread the existing
   `CacheReadInputTokens` / `CacheCreationInputTokens` from `ai.Response` through `GenerateResult`
   into the banked summary, and surface hit rate in `ailang eval-report`.

### Why in-place beats moving to the system role (D1)

Both placements cache equally well — Anthropic's render order is `tools → system → messages`, and
a breakpoint works at any of them. The difference is blast radius:

| | Cache win | Changes model input? | Invalidates baselines? |
|---|---|---|---|
| Split user message in place | Full | Content identical; encoding differs (one text block → two) | No |
| Move teaching prompt to system role | Full | Yes — different role, different model treatment | **Yes** |

The system-role move has independent motivation (teaching content in a user message is lost on
compaction) and may well be right — but it is a *quality* change that needs its own A/B, not a
rider on a *cost* change. Bundling them makes both unmeasurable. Deferred, not rejected.

### Conflict Surface

Not a parser/typechecker change, but it modifies the outgoing wire bytes for every Anthropic
call, and v0.18.4 made an explicit bit-for-bit guarantee. Enumerating what else lives in these
positions:

1. **What this extends:** the `system` field encoding (string ↔ content array) and the user
   message `content` field encoding (string ↔ content array).
2. **What else already lives there:**
   - `system` — `systemFieldFromPrompt` already switches string↔array on the `Step` path
     (`cache.go:65-84`). `Generate` has only ever emitted a string.
   - user `content` — already switches to an array for **tool results** (`step.go:266-278`) and
     for **vision input** (`step.go:281-300`). A third array-producing case must not collide with
     either.
3. **Disambiguation:** the array form is used iff (`CachedPrefix != ""` AND a `user_prefix`
   breakpoint is declared). Tool-result and vision messages take their existing branches first
   and are never eligible — a tool-result message has no `CachedPrefix`, and the vision branch
   already owns its block layout. The new branch is last in the chain.
4. **Programs that MUST still work post-change** (all verified to exist):
   - `examples/runnable/ai_caching.ail` — the `stepWithCache` example; exercises the existing
     `system` breakpoint path
   - `internal/ai/anthropic/cache_test.go` — asserts the byte-identical no-breakpoint shape
   - `internal/ai/anthropic/step_vision_test.go` — the vision content-array branch
   - `internal/ai/anthropic/step_edge_test.go` — tool-result correlation
   - `internal/ai/anthropic/client_test.go` — the `Generate` request shape
5. **What deliberately changes:** requests that declare a `user_prefix` breakpoint emit a
   two-block user content array instead of a string. That is the whole point, and it is opt-in —
   no declaration, no change.

**The honest answer is not "no conflicts":** the user `content` field already has two
array-producing branches, and getting the ordering wrong silently breaks vision or tool calls.
That ordering is the single most important detail in this milestone.

### Implementation Plan

**Phase 1: Make `Generate` cache-capable** (~3 hours)
- [ ] Add `CachedPrefix string` to `ai.Request` with doc comment mirroring `CacheBreakpoints`
- [ ] Add `"user_prefix"` to the `CacheBreakpoint.Position` vocabulary (`provider.go:195-206`)
- [ ] Extend `cache.go` with `userContentFromPrompt(cachedPrefix, userPrompt, breakpoints)`
- [ ] Wire `systemFieldFromPrompt` + the new user-content builder into `client.go` `Generate`
- [ ] Non-Anthropic providers: concatenate `CachedPrefix + UserPrompt` (byte-identical guard test)
- [ ] Below-minimum-prefix one-shot warning

**Phase 2: Turn it on for our own callers** (~2 hours)
- [ ] `spec.go` — return `basePrompt` and `taskDescription` separately instead of pre-concatenated
- [ ] `ai_provider.go` — set `CachedPrefix = basePrompt`, `UserPrompt = task`, declare the
      breakpoint when the provider is Anthropic
- [ ] `classifier.go` — declare a `system` breakpoint on the classifier prompt
- [ ] Warm-up call before `runBenchmarksParallel` fan-out (D4)

**Phase 3: Make the hit rate visible** (~3 hours)
- [ ] Thread cache tokens through `GenerateResult` → banked summary schema
- [ ] Surface cache hit rate in `ailang eval-report`
- [ ] Backward-compatible schema (absent fields read as 0 for pre-v0.31.0 baselines)

### Files to Modify/Create

**Modified files:**
- `internal/ai/provider.go` — `CachedPrefix` field + `user_prefix` position, ~25 LOC
- `internal/ai/anthropic/cache.go` — user-content builder + minimum guard, ~90 LOC
- `internal/ai/anthropic/client.go` — wire cache into `Generate`, ~30 LOC
- `internal/ai/{openai,gemini,ollama,openrouter}/step.go` — concatenate `CachedPrefix`, ~10 LOC each
- `internal/eval_harness/spec.go` — return prompt parts unconcatenated, ~25 LOC
- `internal/eval_harness/ai_provider.go` — set `CachedPrefix` + breakpoint, ~30 LOC
- `internal/eval_harness/ai_agent.go` — carry cache tokens on `GenerateResult`, ~20 LOC
- `internal/feedbackgate/classifier.go` — declare system breakpoint, ~5 LOC
- `cmd/ailang/eval_parallel.go` — warm-up before fan-out, ~30 LOC
- `cmd/ailang/eval_tools.go` — hit-rate column in the `eval-report` renderer (`eval_tools.go:272`), ~25 LOC

**New test files:**
- `internal/ai/anthropic/cache_generate_test.go` — `Generate`-path shapes + branch ordering, ~150 LOC

---

## Examples

### Example 1: A 30-benchmark standard-mode suite (the projection)

Shared prefix = ~16K tokens (V9). Cost in units of one uncached prefix (cache read ≈ 0.1×,
5-minute write ≈ 1.25×):

**Before:**
```
30 calls × 1.00 (full price)                              = 30.00
```

**After (no warm-up — 10 concurrent calls all miss, V7):**
```
10 writes × 1.25  +  20 reads × 0.10                      = 14.50   (−52%)
```

**After (with warm-up, D4):**
```
 1 write  × 1.25  +  29 reads × 0.10                      =  4.15   (−86%)
```

The warm-up is one extra serial round-trip and is worth 34 percentage points.

### Example 2: The wire shape

**Before** (every `Generate` call today — V1):
```json
{"model":"claude-sonnet-5",
 "system":"You are a code generation engine...",
 "messages":[{"role":"user","content":"<16K teaching prompt>\n\n## Task\n\n<task>"}]}
```

**After** (identical content, one cacheable boundary):
```json
{"model":"claude-sonnet-5",
 "system":"You are a code generation engine...",
 "messages":[{"role":"user","content":[
   {"type":"text","text":"<16K teaching prompt>","cache_control":{"type":"ephemeral"}},
   {"type":"text","text":"\n\n## Task\n\n<task>"}]}]}
```

A request that declares no breakpoint emits the **Before** shape byte-for-byte.

---

## Success Criteria

- [ ] `Generate` emits `cache_control` when a breakpoint is declared (acceptance: new
      `cache_generate_test.go` asserts the two-block shape)
- [ ] A request with no breakpoints is byte-identical to v0.30.0 (acceptance: golden-JSON test)
- [ ] Vision and tool-result branches still take precedence (acceptance: existing
      `step_vision_test.go` + `step_edge_test.go` pass unmodified)
- [ ] A below-minimum prefix warns loudly (acceptance: test asserts the warning fires for a
      70-token prefix on a 1024-minimum model)
- [ ] Banked eval summaries carry non-zero `cache_read_input_tokens` (acceptance: live smoke run
      against one Anthropic model, ≥ 70% read share on the second pass)
- [ ] `ailang eval-report` shows a cache hit-rate column
- [ ] `make ci` green, `make check-boundaries` green
- [ ] CHANGELOG.md updated

## Testing Strategy

**Unit tests:**
- Wire-shape golden tests for all four cases: no breakpoint, system-only, user-prefix-only, both
- Branch-ordering: a vision message and a tool-result message with `CachedPrefix` set must take
  their existing branches
- Non-Anthropic providers: `CachedPrefix + UserPrompt` concatenation is byte-identical to today
- Below-minimum guard fires once, not per-call

**Integration tests:**
- `Generate` against a recorded fixture asserting `cache_control` presence
- Eval harness end-to-end with a stub provider, asserting cache tokens reach the summary

**Manual testing:**
- Live 2-benchmark run against one Anthropic model; confirm call 1 reports
  `cache_creation_input_tokens > 0` and call 2 reports `cache_read_input_tokens > 0`
- Confirm the Anthropic Console hit rate moves after a full suite

## Deferred Decisions

- **TTL tier per workload** (D5) — agent may choose; `ephemeral` for dense suites, 1h for
  day-long rotations. Needs ≥3 reads to beat the 2× write premium
- **Whether the warm-up shares the real system prompt or uses `max_tokens: 0`** — agent may
  choose; `max_tokens: 0` prefills the cache without billing output tokens
- **Exact eval-summary field names for cache tokens** — agent may choose, provided pre-v0.31.0
  baselines still parse
- **Whether the feedback-gate classifier prompt clears the minimum** — agent to measure; if it
  does not, skip the breakpoint there rather than shipping a silent no-op

## Non-Goals

**Not attempted in this feature:**
- **Moving the teaching prompt to the system role** — behavior-changing, invalidates baseline
  comparability, and deserves its own A/B. Independently motivated; tracked separately
- **Agent-mode / `claude` CLI caching** — the CLI manages its own; session reuse is a different
  milestone
- **`tool_result` breakpoints for multi-turn agentic loops** — real and worth doing, but the
  `Generate` path carries our volume today. Phase 2 of *this* line of work
- **Gemini Context Caching API** — still deferred from v0.18.4; structurally different async API
- **Auto-detection of "should we cache?"** — explicit opt-in, consistent with v0.18.4
- **Retrofitting cache metrics onto pre-v0.31.0 baselines** — the data was never recorded (V5);
  it cannot be reconstructed

## Timeline

**Day 1** (~5 hours):
- Phase 1: `Generate` cache support + wire-shape tests
- Phase 2: eval harness + classifier wiring, warm-up

**Day 2** (~3 hours):
- Phase 3: cache-token persistence + report column
- Live smoke run, CHANGELOG, docs

**Total: ~8 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Branch-ordering bug silently breaks vision or tool-result requests | **High** | Conflict Surface §3 fixes the order explicitly; existing vision/tool tests must pass **unmodified** — if either needs editing, the ordering is wrong |
| Splitting one text block into two subtly changes model behavior, shifting eval results | Med | Content is identical and the split is at a natural `\n\n## Task\n\n` boundary. Mitigate by running one benchmark both ways before banking; if outputs shift materially, escalate to D1 as a full re-baseline decision |
| Cache-write premium makes short suites *more* expensive | Med | Break-even is 2 requests at 5-min TTL. Guard: skip the breakpoint when a run has fewer than 2 calls per model |
| New summary fields break the dashboard / report readers | Med | Additive fields only; absent = 0. Test against a pre-v0.31.0 baseline directory |
| We ship this and the org hit rate barely moves because the volume is actually CLI/subscription traffic | Med | Check the Anthropic Console breakdown **before** starting; Phase 3 makes our own side measurable either way |
| 5-minute TTL expires between rotation batches, so writes never amortize | Low | D5 — switch those workloads to the 1h tier |

## Related Documents

**Direct predecessor — read before implementing:**
- [design_docs/implemented/v0_18_4/m-ai-prompt-caching.md](../../implemented/v0_18_4/m-ai-prompt-caching.md) (0.44) —
  established the `CacheBreakpoint` vocabulary, the byte-identical-when-unset guarantee, and the
  one-shot warning pattern. Its "Future Work → Phase 2 — Wider Anthropic placement" is
  precisely this doc. Note its scope gap: it wired `Step` only and never covered `Generate`.

**Implemented (may inform design):**
- [design_docs/implemented/v0_9_8/m-cost2-dashboard-firestore-optimization-sprint-plan.md](../../implemented/v0_9_8/m-cost2-dashboard-firestore-optimization-sprint-plan.md) (0.34) — cost-metric plumbing precedent

**Planned (checked for overlap — ⚠️ one real collision):**
- [design_docs/planned/v0_31_0/m-eval-token-headroom.md](m-eval-token-headroom.md) (0.32) —
  **⚠️ Touches the same function in the same release.** Its subject is distinct (*output*
  headroom and truncation visibility vs. this doc's *input* cache reuse), but its Files-to-Modify
  list includes `internal/ai/anthropic/client.go` — specifically a `FinishReason:
  mapStopReason(result.StopReason)` line inside `Generate`, the exact function this doc rewires
  for `cache_control`. Both target v0.31.0. Low conflict risk (different lines, no shared
  semantics) but **they must not be implemented in parallel on separate branches.** Sequence:
  land the headroom one-liner first — it is ~1 LOC — then build cache support on top. It also
  touches `internal/eval_harness/models.go`; this doc does not
- [design_docs/planned/v0_29_0/m-anthropic-sandbox.md](../v0_29_0/m-anthropic-sandbox.md) (0.33) —
  no overlap in files, but worth noting directionally: it introduces a **new** `ANTHROPIC_API_KEY`
  consumer (managed-agents sandbox executor). That is new API-side traffic which should be
  cache-aware from day one rather than retrofitted — the same mistake this doc exists to correct

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Anthropic prompt caching: render order `tools → system → messages`; max 4 breakpoints;
  minimum cacheable prefix 512 (Opus 5) / 1024 (Sonnet 4.6, Sonnet 5, Opus 4.8) / 4096
  (Haiku 4.5) tokens; cache read ≈ 0.1×, 5-min write 1.25×, 1-hour write 2×
- `.claude/rules/coding-standards.md` — Principle 2 (no silent fallbacks) motivates the
  below-minimum warning

## Future Work

- **`tool_result` / multi-turn breakpoints** — the remaining half of v0.18.4's deferred Phase 2;
  matters once agentic loops move onto the API path
- **System-role teaching prompt A/B** — the deferred D1 alternative, on quality grounds
- **Auto-cache heuristic** — with Phase 3 telemetry in hand, thresholds become tunable rather
  than guessed
- **Cache hit rate as a dashboard panel + regression gate** — so the next hit-rate collapse is
  caught by us, not reported to us

---

**Document created**: 2026-07-29
**Last updated**: 2026-07-29
