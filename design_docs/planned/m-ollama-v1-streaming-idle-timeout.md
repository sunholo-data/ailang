# M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT: Stream the ollama /v1 tool-calling path with an idle timeout

**Tracking**: ailang#618 (this) · ailang#619 (the publisher-validity fallout)
**Status**: REVIVED 2026-08-07 (was archived 2026-07-29 as charter-unreferenced — see Field Evidence below; the diagnosis was never wrong, it just wasn't on the charter)
**Target**: v0.34.0
**Priority**: **P0** (raised from P1 — measured 43-day production impact: 80 motoko runs lost, ~74.6 GPU-hours burned, and ACCELERATING)
**Estimated**: 2–3 days
**Dependencies**: None. Complements (does not block) motoko_agent#65 (raise total timeout) and P2 context-compression.

---

## Field Evidence (2026-08-07) — why this was revived

Measured by scanning all **2058** motoko session JSONLs in `mk-ast/.motoko/logfile` plus the live
ollama access log (`/private/tmp/ollama-serve-launchd.log` — NOT `~/.ollama/logs/server.log`,
which froze at the 2026-08-03 server restart).

| Metric | Value |
|---|---|
| Sessions carrying `context deadline exceeded` | **182 / 2058 (8.8%)** |
| — died outright (no `run_summary`) | **80** |
| — finished anyway after retrying (silently inflated latency/cost) | **102** |
| Total retry events | **895** |
| Lower-bound GPU time burned | **~74.6 h** (895 × 300s; excludes the 8–22s reload tax per retry and the final in-flight request killed at the wall clock) |
| First affected session → today | **2026-06-26 → 2026-08-07 = 43 days continuous** |

**It is accelerating, and not because of a model change** (`qwen3.6:35b-a3b-mxfp8` in every
affected session): retries/day **16.7 (Jun) → 13.5 (Jul) → 51.4 (Aug)**; retries per affected
session **2.7 → 4.6 → 7.5**. The step counter increments per failure and **step 0 itself times
out**, so "the context grew too long" is REFUTED — the cap is simply below the median turn.

**Why 300s is below a turn:** passing motoko-local runs measure **7–62 output tok/s** (slower at
long context), with whole sessions running **20–37 min**. At ~15 tok/s the 300s budget buys only
~4,500 tokens — right at the median thinking-mode turn, so easy benchmarks squeak under and
long-reasoning ones lose nearly every turn. Evidence it is real generation and not a hang:
durations spread 48s → 4m59 rather than always pinning the cap. On 2026-08-07 alone, **47 requests
ended at exactly `took=4m59.97x`**.

**Blast radius beyond lost runs:** the 30 resulting `api_error` rows all carried
`validity.valid=false, reason="harness_error"`, but `cmd/ailang/eval_publish.go` never reads
`validity` — so motoko-local's published v0.33.0 **frontier score was exactly 3/22 = 13.6%** where
**17 of those 22 were harness timeouts**. True figure ≈ 60% (n=5). That publisher defect is a
distinct bug tracked as W8 in
[m-eval-validity-discipline](m-eval-validity-discipline.md) / ailang#619; the corrupted rows were deleted
2026-08-07.

**Stopgap in place (REMOVE when this lands):** `AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800` set via
`launchctl setenv` and pinned in `tools/launchd/dev.ailang.os-rotation-filler.plist` +
`dev.ailang.nightly-eval.plist`. This is precisely the band-aid the Problem Statement below warns
about — it enlarges the guess and over-tolerates genuine hangs (the cap exists because of a
**1h54m** and a **~7h** rig-wedging hang; see 63fc63e09 / 772704cbb). It buys time, not a fix.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Rationale |
|---|---|---|
| A1: Determinism | 0 | Timeouts are inherently wall-clock; result determinism unchanged. Fewer *spurious* failures on big prompts. |
| A2: Replayability | +1 | Fewer false timeout aborts → fewer truncated/aborted traces to replay. |
| A3: Effect Legibility | 0 | No effect-signature changes (Net effect still {Net}). |
| A4: Explicit Authority | 0 | No new ambient authority. |
| A5: Bounded Verification | 0 | No type-system impact. |
| A6: Safe Concurrency | 0 | Single SSE connection read; no new concurrency. |
| A7: Machines First | +1 | Long-running local-LLM calls stop failing mid-task — directly serves AI-agent reliability. |
| A8: Minimal Syntax | 0 | No syntax. |
| A9: Cost Visibility | +1 | Idle-timeout + time-to-first-token are explicit, tunable env knobs; a genuine hang still surfaces fast. |
| A10: Composability | 0 | Reuses the native path's chunk-reassembly machinery. |
| A11: Structured Failure | +1 | Distinguishes "idle/hung" from "long but progressing"; typed timeout error instead of a swallowed cut stream. |
| A12: System Boundary | 0 | HTTP/ollama boundary unchanged. |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check
- [x] A1 (Determinism): no implicit nondeterminism introduced (timeout semantics only).
- [x] A3 (Effects): no hidden side effects.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): optimizes machine reliability, not human convenience.

## Problem Statement

The ollama `/v1` tool-calling path in `internal/ai/ollama/step.go` is **non-streaming**: it `io.ReadAll`s
the full response body, so it can only be bounded by a **TOTAL** `http.Client.Timeout` /
`context.WithTimeout` (default 300s via `AILANG_OLLAMA_HTTP_TIMEOUT_SEC`). The whole exchange —
connection + prefill (processing the entire prompt) + generating the *complete* response — must finish
within that single budget.

**Current State:**
- On large-context agentic tasks, the accumulated prompt makes a single qwen3.6 request exceed the total
  budget, and the AILANG runtime aborts the **entire run** with
  `Post "http://localhost:11434/v1/chat/completions": context deadline exceeded`.
- Observed concretely on the `docx_reimplement` P0 large-context instrument: motoko died at step 14 (300s)
  and again at step 27 after writing a full 526-line reimplementation — the run was killed mid-task, not
  because the model failed but because one request outran the total clock.
- The code already knows this is a limitation — `step.go:252`: *"the /v1 path does not [stream], so we MUST
  cap it ourselves."* The native `/api/chat` path **does** stream chunk-by-chunk and "surfaces a dropped
  stream quickly," but the tool-calling path doesn't get that benefit.
- Band-aid in flight: motoko_agent#65 raises the total to 1800s. That just enlarges the guess; it doesn't
  remove the fundamental "must size a total budget" problem and over-tolerates genuine hangs.

**Impact:**
- Any harness (motoko, pi, the eval harness) running multi-step agentic AILANG synthesis on local ollama
  hits this on long/large-context tasks — the regime where AILANG-native harnesses are supposed to win.
- It silently caps the achievable difficulty of the eval suite (large-context tasks can't complete), so it
  blocks the P0 differentiator measurement entirely.

## Goals

**Primary Goal:** Make long-running local-LLM tool-calling calls robust to legitimately-long generation
without having to guess a total time budget.

**Success Metrics:**
- A `/v1` request whose *total* time exceeds 300s but which is *continuously producing tokens* completes
  successfully (no spurious abort).
- A genuinely hung/stalled stream (no bytes for the idle window) still fails fast (≤ idle window + slack).
- `docx_reimplement` completes a full motoko run end-to-end (produces an X/17 grade) rather than dying on a
  timeout mid-task.
- No regression on normal small calls (latency + correctness unchanged).

## High-Impact Decisions

| Decision | Why it matters | Who decides | When | Change cost |
|---|---|---|---|---|
| Idle/inter-chunk deadline vs total deadline | The core fix — removes the "guess a total budget" failure mode | human/maintainer | design | high |
| Stream the `/v1` path (SSE `stream:true`) + reassemble tool-call deltas | Enables idle-timeout; must not break tool-calling | agent (impl), human (review) | runtime | med |
| Time-to-first-token (TTFT) allowance separate from inter-chunk idle | Prefill on a huge prompt is token-less; a naive idle-timeout would false-trip during prefill | human/maintainer | design | med |
| Keep a hard ceiling as a backstop? | Defense-in-depth against a slow-drip stream that never ends | human/maintainer | design | low |

### Design Freeze (decide before coding)
- [ ] Idle-window default (proposal: `AILANG_OLLAMA_IDLE_TIMEOUT_SEC`, default 120s).
- [ ] TTFT-window default (proposal: `AILANG_OLLAMA_TTFT_TIMEOUT_SEC`, default 600s — prefill on big prompts).
- [ ] Whether `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` becomes an *optional* hard ceiling (0 = disabled) once idle-timeout lands.

## Solution Design

**Overview:** Switch the `/v1` tool-calling call from a single buffered read under a total deadline to a
**streamed** read (OpenAI-compatible SSE, `stream:true`) governed by an **idle/read deadline** that resets on
every received chunk, plus a separate, more generous **time-to-first-token** window to absorb prefill silence.

**Architecture:**
1. Request `stream:true` on the `/v1` chat-completions call (the OpenAI Go client supports `CreateChatCompletionStream`).
2. Reassemble streamed deltas into a final message — **content deltas** and **tool-call deltas** (index/id/
   name/arguments arrive incrementally). The native `/api/chat` path already does cross-chunk tool-call
   reassembly (`step.go:366`, *"Tool calls arrive in the message (possibly across streamed chunks)"*) — reuse
   that logic so behavior is identical to today once assembled.
3. Replace the total `context.WithTimeout` with a **rolling read deadline**: arm a timer for the TTFT window;
   on the first chunk, switch to the inter-chunk idle window; reset on each subsequent chunk. If the timer
   fires, cancel the context → surface a typed `timeout` error (preserving the existing "don't swallow a cut
   stream as EOF" guard at `step.go:378`).
4. Optionally keep `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` as a *hard ceiling* backstop (default 0/disabled once
   idle-timeout is the primary mechanism).

**Files to modify:**
- `internal/ai/ollama/step.go` (~80–150 LOC): the `/v1` client construction + read loop; new idle/TTFT env
  knobs in `ollamaV1Timeout()`'s neighborhood; reuse the native path's delta-reassembly.
- `internal/ai/ollama/step_test.go`: unit tests with a fake SSE server (slow-but-progressing → passes;
  idle/hung → fails fast; prefill-silence-then-tokens → passes).

**Out of scope / non-goals:** changing the native `/api/chat` path (already streams); context compression
(that is P2 — this fix handles "long generation", P2 handles "huge prefill"); any motoko-side change beyond
already-merged env forwarding (motoko_agent#65).

## Risks / Conflict Surface

This change is confined to `internal/ai/ollama` (NOT parser/typechecker/codegen), so there is no
grammar/type conflict surface. Runtime risks:
- **Tool-call delta reassembly correctness** — streamed tool-calls split name/arguments across chunks. Mitigation: reuse the native path's proven reassembly; golden-test assembled output equals the buffered result on a captured fixture.
- **ollama SSE compatibility** — confirm ollama's `/v1` honors `stream:true` with tool-calls for the target models. Mitigation: capture a real streamed response in a test fixture; gate behind an env flag (`AILANG_OLLAMA_V1_STREAM=1`) for one release before defaulting on.
- **Prefill silence false-trip** — handled by the separate TTFT window; must be ≥ worst-case prefill for the largest context the eval suite uses.

## Success Criteria
- [ ] Streamed `/v1` assembled output is byte-identical to the buffered output on a captured fixture.
- [ ] Slow-but-progressing stream (>300s total) completes; idle/hung stream fails within idle window + slack.
- [ ] Tool-calling behavior unchanged (eval suite tool-call benchmarks unaffected).
- [ ] `docx_reimplement` produces an end-to-end grade (no mid-task timeout abort).
- [ ] All tests passing; `make verify-examples` and `make test-imports` green.
- [ ] Documentation updated (debug flags table + this doc moved to implemented/).

## Timeline (realistic, ~2x first guess)
- Day 1: streamed `/v1` read + delta reassembly; fixture golden vs buffered.
- Day 2: idle + TTFT deadline logic; unit tests (fake SSE: progressing / idle / prefill-silence).
- Day 3: env-flag gating, end-to-end docx validation on the rig, docs.

## Related Documents
- [m-eval-stream-health-retry](v0_24_0/m-eval-stream-health-retry.md) (neural 0.48) — **distinct**: that covers eval-harness-level stream health + retry policy; this is the ollama client's per-call timeout semantics underneath it. Complementary.
- [m-ai-streaming-helper](v0_17_0/m-ai-streaming-helper.md) (neural 0.37) — **distinct**: std/ai user-facing streaming wrapper; this is the internal tool-calling transport.
- motoko_agent#65 — the env-forwarding band-aid (raise total timeout) this design supersedes as the robust fix.
- P2 (context_mode / on_tool_handle compression) — attacks prefill *size*; orthogonal and complementary.
