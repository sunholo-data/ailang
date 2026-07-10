# M-EVAL-STREAM-HEALTH-RETRY: Mid-stream death detection + fast retry for the streaming executor

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (High — a single stream death silently burns a 20-minute hard-cap and corrupts a benchmark cell with a phantom `api_error`)
**Estimated**: 1–1.5 days
**Dependencies**: Extends `internal/eval_harness/agent_runner_streaming.go`; reuses the activity-monitor pattern from [M-EXECUTOR-IDLE-TIMEOUT](../../implemented/v0_8_1/m-executor-idle-timeout.md); complements [M-MOTOKO-OLLAMA-LOOP-CONVERGENCE](m-motoko-ollama-loop-convergence.md) Phase 4 (subprocess reaping).

> **📊 EMPIRICALLY OBSERVED (nightly run 2026-06-07 04:44, `opencode-qwen3-5-35b-a3b-mxfp8`):**
> At 04:40:32 an **ollama model reload silently killed an in-flight generation stream**. The
> opencode session did not error — it simply stopped receiving tokens. Nothing in the harness
> noticed the stall; the session sat idle until the **`generation_timeout: 1200` per-session
> hard cap** ([`models.yml:1191`](../../../internal/eval_harness/models.yml)) fired ~20 minutes
> later and the cell was recorded as `api_error`. This is the `json_parse` "1/2 · api_error"
> result in that run's `summary.json`. The benchmark recovered on its sibling trial, so it was
> correctly **not** escalated — but ~20 minutes of wall-clock and one of two trials were lost to
> a failure mode that is detectable in seconds.

---

## Problem statement

The streaming executor has exactly one liveness backstop: `generation_timeout` (currently 1200s,
bumped from 600s — see [`models.yml:1191`](../../../internal/eval_harness/models.yml)). That
timer measures **total session wall-clock**, not **progress**. It cannot distinguish:

1. A model legitimately working for 18 minutes on a hard benchmark (must not be killed), from
2. A stream that died at minute 1 and will never produce another token (should be killed and
   retried immediately).

Both look identical to a wall-clock cap, so the cap is forced to be generous (1200s) to avoid
false-killing slow-but-healthy generations. The cost: every genuine stream death pays the full
20-minute cap before the harness gives up, and the lost trial is mis-labelled `api_error`
(transport noise) when the real cause was a dead pipe.

The activity-monitor that solves exactly this already exists for the Claude/Gemini CLI executors
([M-EXECUTOR-IDLE-TIMEOUT](../../implemented/v0_8_1/m-executor-idle-timeout.md), v0.8.1) and the
TTFT-vs-idle split was already reasoned through for local models
([m-ollama-local-eval.md](../../implemented/v0_15_0/m-ollama-local-eval.md) §"prefill vs
per-token idle"). **Neither was wired into the opencode/streaming path.** This doc closes that gap
and adds the retry that the idle-timeout doc explicitly deferred to "existing retry logic" — which,
for this path, does not exist.

## Goals

1. **Detect a dead stream by progress, not wall-clock.** Add a per-stream *idle* timer that fires
   when no new bytes/tokens have arrived for `idle_timeout` seconds, independent of total session
   time.
2. **Distinguish three waits** so each can be bounded separately:
   - **TTFT** (time to first token) — prefill latency; hardware/prompt-size dependent.
   - **Inter-token idle** — gap *between* tokens after generation has started; the dead-stream signal.
   - **Total generation** — the existing `generation_timeout` wall-clock backstop (kept, unchanged).
3. **Fast retry on stream death.** When the idle timer fires (not the wall-clock cap), tear down
   the opencode subprocess cleanly and retry the cell once, before consuming the trial budget.
4. **Label correctly.** A stream death that retries-and-succeeds is invisible in the cell verdict.
   A stream death that exhausts retries is recorded as `stream_death` (a distinct, infra category),
   **not** `api_error` — so the regression detector keeps excluding it
   (see [M-EVAL-REGRESSION-DETECTOR-CONTRACT](m-eval-regression-detector-contract.md) `INFRA_CATEGORIES`).

## Non-goals

- Changing `generation_timeout` itself — it stays as the outer wall-clock backstop.
- Heartbeat/ping protocols into the model (opencode/ollama do not expose one; progress-based
  detection is sufficient and simpler).
- Unlimited retries — one fast retry per cell. A second death falls through to the trial budget
  and is reported honestly as a lost trial, not retried into oblivion.

## Design

### Per-stream activity monitor (Option A from M-EXECUTOR-IDLE-TIMEOUT, generalized)

In `agent_runner_streaming.go`, wrap the token/byte read loop with a monitor goroutine that stamps
`lastProgress` on every non-empty read. Three timers run against that stamp:

| Timer | Default | Resets on | Action on fire |
|-------|---------|-----------|----------------|
| `ttft_timeout` | 180s | — (first token only) | kill + retry: model never started |
| `idle_timeout` | 90s | each new token | kill + retry: stream died mid-generation |
| `generation_timeout` | 1200s | — (total) | kill, **no** retry: genuinely slow/runaway (existing behavior) |

Defaults are starting points to be A/B-validated on the local rig the way the 1M→4M token cap was
(see [m-eval-cost-and-speed-budgets.md](../../implemented/v0_15_1/m-eval-cost-and-speed-budgets.md)).
`idle_timeout` must sit comfortably above the slowest *healthy* inter-token gap observed for thinking
models — [m-eval-local-observability.md](../../implemented/v0_22_0/m-eval-local-observability.md)
notes models that reason 5–10 min before *first* visible output, which is why inter-token idle
(post-first-token) is the safe signal and TTFT gets its own, longer bound.

### Config (`models.yml`, per model)

```yaml
budgets:
  generation_timeout: 1200   # unchanged: total wall-clock hard cap
  ttft_timeout: 180          # NEW: max wait for the FIRST token
  idle_timeout: 90           # NEW: max gap BETWEEN tokens → dead-stream trip
  stream_death_retries: 1    # NEW: fast retries before consuming a trial
```

Absent keys → idle/TTFT detection off, preserving today's behavior for untouched models (safe
rollout, mirrors the M-EXECUTOR-IDLE-TIMEOUT feature-flag approach).

### Retry path

On an `idle_timeout` or `ttft_timeout` trip (NOT a `generation_timeout` trip):
1. SIGTERM → reap the opencode subprocess (reuse the reaping work tracked in
   [M-MOTOKO-OLLAMA-LOOP-CONVERGENCE](m-motoko-ollama-loop-convergence.md) Phase 4 so we don't
   leave orphaned ollama generations).
2. Emit a `stream_retry` telemetry span (so the dashboard can count how often streams die — a
   leading indicator of ollama-reload thrash worth its own budget).
3. Re-issue the identical request. On success the cell passes normally. On a second death, record
   `error_category: stream_death` and let the trial budget absorb it.

### Reporting

- `stream_death` joins `INFRA_CATEGORIES` in the regression detector — never escalates as a
  regression (it is transport, not a language/prompt/stdlib defect).
- The nightly suite log gains a one-line `stream deaths: N (retried: M)` counter so a night with
  heavy ollama-reload thrash is visible even when every cell ultimately recovered.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 Determinism | +1 | Removes a 20-min-variance failure mode; retries converge cells that today flip on transport luck. |
| A2 Replayability | +1 | A re-run no longer depends on whether ollama happened to reload mid-stream. |
| A7 Machines First | +1 | A phantom `api_error` teaches the leaderboard the wrong thing; correct `stream_death` labelling gives a clean signal. |
| A9 Cost Visibility | +1 | Surfaces stream-death frequency as a first-class counter; reclaims ~20 min/death of wasted compute. |
| A11 Structured Failure | +2 | Replaces "looked like api_error after 1200s" with a typed, detected-in-seconds, retried failure mode. |
| A12 System Boundary | +1 | The harness stops trusting the executor to fail loudly; it detects silence itself. |

**Hard violation check:** none.

## Acceptance

- [ ] A killed stream (simulate: `kill -STOP` the ollama serve mid-generation, or a fault-injection
      hook) is detected within `idle_timeout`, not `generation_timeout`.
- [ ] The cell retries once and passes when the stream recovers; records `stream_death` only after
      the retry also dies.
- [ ] A genuinely slow-but-healthy generation (steady tokens, 15 min) is **not** killed by
      `idle_timeout`.
- [ ] `stream_death` is excluded from regression escalation; appears in the GAP/non-regression bucket.
- [ ] Nightly log prints the `stream deaths: N (retried: M)` counter.

## Open questions

1. Does opencode expose per-token stream framing the harness can observe, or only coarse stdout
   chunks? If only chunks, `idle_timeout` keys off chunk arrival (coarser but still seconds, not
   minutes) — verify against `agent_runner_streaming.go`.
2. Should `idle_timeout` be benchmark-aware (hard benchmarks tolerate longer thinking pauses)? Defer
   to per-tier config if a single global value false-trips on the `dense_operator_program` class.
