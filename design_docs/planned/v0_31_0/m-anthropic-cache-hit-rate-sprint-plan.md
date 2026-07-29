# Sprint Plan: M-ANTHROPIC-CACHE-HIT-RATE

**Design doc**: [m-anthropic-cache-hit-rate.md](m-anthropic-cache-hit-rate.md)
**Sprint ID**: `M-ANTHROPIC-CACHE-HIT-RATE`
**Target**: v0.31.0
**Duration**: 2 days (~8 hours)
**Total LOC estimate**: ~470 (≈290 implementation + ≈180 tests)
**Risk level**: Medium — the risk is concentrated entirely in M1's content-block branch ordering
**Created**: 2026-07-29

---

## Goal

Make our Anthropic API traffic cache-capable end to end. Today the `Generate` path — which
carries essentially all of our Anthropic spend — contains **zero** references to `CacheBreakpoint`
(design doc V1), so it cannot request a cache at all. Close that, put the breakpoint where the
~16K-token teaching prompt actually is, turn it on by default, and make the resulting hit rate a
banked metric instead of an invisible one.

## Design Freeze — resolved, do not re-litigate

| ID | Decision |
|----|----------|
| **D1** | Cache **in place**. Split the user message at the `\n\n## Task\n\n` boundary into two content blocks. Do **not** move the teaching prompt to the system role — that is behaviour-changing and would invalidate v0.30.0 baseline comparability |
| **D2** | `Request.CachedPrefix` is the provider-neutral carrier. Anthropic stamps it; every other provider concatenates `CachedPrefix + UserPrompt` and stays byte-identical |
| **D3** | Caching defaults **ON** for the eval harness and feedback gate on Anthropic, with a guard that skips the breakpoint when a run makes fewer than 2 calls per model |

## Velocity basis

Recent comparable milestones (`M-EVAL-MEASUREMENT-CONTRACT` M3/M4/M5, `M-Z3-HARD-TIMEOUT`) each
landed as a discrete commit at roughly half-day granularity with tests included. Three milestones
at ~2.5h each is consistent with that pace; the design doc's ~8h estimate stands.

---

## Milestones

### M1 — `Generate` becomes cache-capable (~3h, ~230 LOC)

The only milestone with real risk. Everything else is wiring.

**Tasks:**
- [ ] Add `CachedPrefix string` to `ai.Request` (`internal/ai/provider.go`), documented alongside
      `CacheBreakpoints`. Empty = today's behaviour exactly
- [ ] Add `"user_prefix"` to the `CacheBreakpoint.Position` vocabulary
- [ ] `internal/ai/anthropic/cache.go`: add `userContentFromPrompt(cachedPrefix, userPrompt, breakpoints)`
      returning `json.RawMessage` — bare string when not caching, two-block array when caching
- [ ] `internal/ai/anthropic/client.go`: call `systemFieldFromPrompt` + the new user-content
      builder from `Generate` (this is the V1 gap)
- [ ] Below-minimum-prefix one-shot warning (model-aware: 512 / 1024 / 4096 by tier)
- [ ] Non-Anthropic providers (`openai`, `gemini`, `ollama`, `openrouter`): concatenate
      `CachedPrefix + UserPrompt`
- [ ] New `internal/ai/anthropic/cache_generate_test.go`

**Acceptance criteria:**
- A `Generate` request declaring a `user_prefix` breakpoint emits the two-block user content array
  with `cache_control` on block 0
- A request declaring **no** breakpoints is byte-identical to v0.30.0 (golden-JSON test)
- Existing `step_vision_test.go` and `step_edge_test.go` pass **unmodified** — if either needs
  editing, the branch ordering is wrong and the milestone is not done
- Non-Anthropic providers produce byte-identical output with `CachedPrefix` set vs. pre-concatenated
- Below-minimum prefix warns once, naming model and shortfall

**Risk:** The user `content` field already has two array-producing branches — tool results
(`step.go:266-278`) and vision (`step.go:281-300`). The new branch must be **last**. Getting this
wrong silently breaks vision or tool calling. The unmodified-existing-tests criterion is the guard.

---

### M2 — Turn it on for our own callers (~2h, ~130 LOC)

**Tasks:**
- [ ] `internal/eval_harness/spec.go`: return `basePrompt` and `taskDescription` separately instead
      of pre-concatenating at line 274-276
- [ ] `internal/eval_harness/ai_provider.go`: set `CachedPrefix = basePrompt`,
      `UserPrompt = taskDescription`, declare the breakpoint when the provider is Anthropic (D3)
- [ ] Fewer-than-2-calls-per-model guard (D3)
- [ ] `internal/feedbackgate/classifier.go`: declare a `system` breakpoint; **measure first** — if
      the classifier prompt is under the model's minimum, skip it rather than ship a silent no-op
- [ ] `cmd/ailang/eval_parallel.go`: warm-up call before the fan-out (`max_tokens: 0` prefill)

**Acceptance criteria:**
- The prompt string reaching the model is unchanged from v0.30.0 (`CachedPrefix + UserPrompt`
  equals the old `fullPrompt` byte-for-byte) — asserted by test
- A 2-benchmark stub run shows call 1 writing cache and call 2 reading it
- Warm-up fires exactly once per (model, suite), before any fan-out
- Runs with <2 calls per model declare no breakpoint

**Risk:** Low. The byte-equality assertion is what protects baseline comparability.

---

### M3 — Make the hit rate visible (~2.5h, ~110 LOC)

Without this we cannot verify M1/M2 worked, and the next hit-rate collapse is again invisible.

**Tasks:**
- [ ] Thread `CacheReadInputTokens` / `CacheCreationInputTokens` from `ai.Response` through
      `GenerateResult` (`internal/eval_harness/ai_agent.go`) into the banked summary
- [ ] Add a cache hit-rate column to the `eval-report` renderer (`cmd/ailang/eval_tools.go:272`)
- [ ] Backward compatibility: pre-v0.31.0 baselines with absent fields read as 0, not error

**Acceptance criteria:**
- Banked summaries carry non-zero `cache_read_input_tokens` after a warm Anthropic run
- `ailang eval-report` against a **pre-v0.31.0** baseline directory still succeeds
- Hit rate appears in the report output

**Risk:** Low. Additive schema only.

---

## Success Metrics

- [ ] `cache_read_input_tokens > 0` in banked summaries (today: 0 across 28,139 files — V5)
- [ ] ≥70% cache-read share of shared-prefix input tokens on a repeat 30-benchmark suite
- [ ] Zero wire-shape change when no breakpoints are declared
- [ ] `make ci` green, `make check-boundaries` green
- [ ] CHANGELOG.md updated

## Out of scope (from the design doc's Non-Goals)

System-role prompt move · agent-mode/`claude` CLI caching · `tool_result` multi-turn breakpoints ·
Gemini Context Caching · auto-cache heuristics · retrofitting metrics onto old baselines

## Open questions for execution

- **Warm-up shape** — `max_tokens: 0` prefill vs. a real minimal call. Agent may choose;
  `max_tokens: 0` avoids billing output tokens
- **TTL tier** — `ephemeral` (5m) for dense suites. 1h deferred until telemetry justifies the 2×
  write premium
- **Classifier breakpoint** — ship only if the prompt clears the model minimum; measure in M2

---

**Sprint plan created**: 2026-07-29
